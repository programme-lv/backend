package exec

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
)

// ExecRepo interface for execution storage
type ExecRepo interface {
	Save(ctx context.Context, exec *Execution) error
	Get(ctx context.Context, id uuid.UUID) (*Execution, error)
}

type ExecSrvcClient interface {
	Enqueue(ctx context.Context, uuid uuid.UUID, srcCode string, prLangId string, tests []TestFile, params TestingParams) error
	Listen(ctx context.Context, uuid uuid.UUID) (<-chan Event, error)
	Get(ctx context.Context, execUuid uuid.UUID) (Execution, error)
}

type ExecSrvcFacade interface {
	ExecSrvcClient
	StartPollingResultQueue() error
	Close()
}

// ExecSrvcImpl handles communication with testers
// for code execution and result streaming
type ExecSrvcImpl struct {
	logger *slog.Logger

	// either in-mem or s3
	execRepo ExecRepo

	sqsClient *sqs.Client

	// submission sqs queue url
	submQ string
	// response sqs queue url
	respQ string

	mu sync.Mutex
	// maps exec IDs to client result channels
	notifiers map[uuid.UUID]chan Event
	// tracks completion status of executions
	execWg sync.Map // notifies get listener when execution is finished

	listenCancel     context.CancelFunc
	listenWait       sync.WaitGroup // on close, waits for sqs jobs to finish
	listeningStarted bool           // tracks if StartListening has been called

	organizers map[uuid.UUID]*ExecResStreamOrganizer
	executions map[uuid.UUID]*Execution
}

// StartPollingResultQueue begins listening for SQS messages in a separate goroutine.
// Returns an error if called more than once.
func (e *ExecSrvcImpl) StartPollingResultQueue() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.listeningStarted {
		return fmt.Errorf("listening already started")
	}

	e.listeningStarted = true
	e.listenWait.Add(1)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		e.listenCancel = cancel
		defer cancel()
		err := StartReceivingResultsFromSqs(
			ctx,
			e.respQ,
			e.sqsClient,
			e.handleSqsMsg,
			e.logger,
		)
		if err != nil {
			slog.Error("failed to listen for sqs messages", "error", err)
		}
		e.listenWait.Done()
	}()

	return nil
}

func NewExecSrvc(ctx context.Context, repo ExecRepo) ExecSrvcFacade {
	esrvc := &ExecSrvcImpl{
		logger:     ctxlog.FromContext(ctx),
		sqsClient:  getSqsClientFromEnv(),
		submQ:      getSubmSqsUrlFromEnv(),
		respQ:      getResponseSqsUrlFromEnv(),
		execRepo:   repo,
		notifiers:  make(map[uuid.UUID]chan Event),
		organizers: make(map[uuid.UUID]*ExecResStreamOrganizer),
		executions: make(map[uuid.UUID]*Execution),
	}

	return esrvc
}

// Enqueue processes a code execution request by:
//  1. Validating the programming language and
//     constraints
//  2. Setting up result handlers and notification
//     channels
//  3. Sending the execution request to the
//     processing queue
//
// Returns the execution UUID for tracking
func (e *ExecSrvcImpl) Enqueue(
	ctx context.Context,
	execUuid uuid.UUID,
	srcCode string,
	langId string,
	tests []TestFile,
	params TestingParams,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. validate programming language
	lang, err := getPrLangById(langId)
	if err != nil {
		return err
	}

	// 2. validate tester execution constraints,
	// checker
	err = params.IsValid()
	if err != nil {
		return err
	}

	// 3. validate test files
	if len(tests) > 200 {
		return fmt.Errorf(
			"too many tests",
		)
	}
	for _, test := range tests {
		if err := test.IsValid(); err != nil {
			return err
		}
	}

	// Add WaitGroup before preparing results
	wg := &sync.WaitGroup{}
	wg.Add(1)
	e.execWg.Store(execUuid, wg)

	// 5. initialize organizer, processor and
	// notifier
	err = e.prepareForResults(
		execUuid,
		lang,
		params,
		len(tests),
	)
	if err != nil {
		return err
	}

	// 6. enqueue execution request to sqs
	err = enqueue(
		execUuid,
		srcCode,
		lang,
		tests,
		params,
		e.sqsClient,
		e.submQ,
	)
	if err != nil {
		return err
	}

	return nil
}

// Listen returns a channel that streams execution
// events to clients. The channel is automatically
// closed once the execution is complete
func (e *ExecSrvcImpl) Listen(
	ctx context.Context,
	execId uuid.UUID,
) (<-chan Event, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch, ok := e.notifiers[execId]
	if !ok {
		format := "no listener for exec %s"
		errMsg := fmt.Errorf(format, execId)
		return nil, errMsg
	}
	return ch, nil
}

// Get retrieves the execution results for a given
// execution ID. It waits for completion if the
// execution is still in progress
func (e *ExecSrvcImpl) Get(
	ctx context.Context,
	execId uuid.UUID,
) (Execution, error) {
	// Get the WaitGroup for this execution
	wgVal, exists := e.execWg.Load(execId)
	if !exists {
		exec, err := e.execRepo.Get(
			ctx,
			execId,
		)
		if err != nil {
			return Execution{}, fmt.Errorf(
				"no execution found for id %s",
				execId,
			)
		}
		return *exec, nil
	}

	wg := wgVal.(*sync.WaitGroup)

	// Wait for completion with context
	// cancellation support
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Clean up the WaitGroup
		e.execWg.Delete(execId)
		exec, err := e.execRepo.Get(
			ctx,
			execId,
		)
		if err != nil {
			return Execution{}, err
		}
		return *exec, nil
	case <-ctx.Done():
		return Execution{}, ctx.Err()
	}
}

// handleSqsMsg processes incoming SQS messages
// and routes them to appropriate handlers
func (e *ExecSrvcImpl) handleSqsMsg(
	msg SqsResponseMsg,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	org, exists := e.organizers[msg.ExecId]
	if !exists {
		e.logger.Debug(
			"no organizer found for execution",
			"exec_id",
			msg.ExecId,
		)
		return fmt.Errorf("no organizer found for execution %s", msg.ExecId)
	}

	if org == nil {
		e.logger.Error(
			"organizer is nil for execution",
			"exec_id",
			msg.ExecId,
		)
		return fmt.Errorf("organizer is nil for execution %s", msg.ExecId)
	}

	if org.HasFinished() {
		return nil
	}

	events, err := org.Add(msg.Data)
	if err != nil {
		return fmt.Errorf(
			"failed to process msg: %w",
			err,
		)
	}
	exec := e.executions[msg.ExecId]
	if exec == nil {
		e.logger.Error(
			"execution not found",
			"exec_id",
			msg.ExecId,
		)
		return fmt.Errorf("execution not found for %s", msg.ExecId)
	}

	for _, event := range events {
		err := applyEventToExec(exec, event)
		if err != nil {
			return fmt.Errorf(
				"failed to apply event: %w",
				err,
			)
		}
		e.notifiers[msg.ExecId] <- event
	}
	if !org.HasFinished() {
		return nil
	}
	close(e.notifiers[msg.ExecId])
	delete(e.notifiers, msg.ExecId)  // deleting closes the channel
	delete(e.organizers, msg.ExecId) // cleanup the organizer
	delete(e.executions, msg.ExecId) // cleanup the execution
	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()
	err = e.execRepo.Save(ctxWithTimeout, exec)
	if err != nil {
		e.logger.Error(
			"failed to save execution",
			"error",
			err,
		)
		return fmt.Errorf(
			"failed to save execution: %w",
			err,
		)
	}
	wgVal, exists := e.execWg.Load(msg.ExecId)
	if exists {
		wg := wgVal.(*sync.WaitGroup)
		wg.Done()
	}
	return nil
}

// prepareForResults sets up the event processing
// pipeline for an execution including result
// organization and client notification channels
// we are in a locked mutex state in this function
func (e *ExecSrvcImpl) prepareForResults(
	execId uuid.UUID,
	lang PrLang,
	params TestingParams,
	numTests int,
) error {
	// initialize some kind of mysthical organizer
	// that reorders events the organizer has to
	// know the number of tests and whether the
	// submission has a compilation step
	e.notifiers[execId] = make(
		chan Event,
		1000,
	)

	org, err := NewExecResStreamOrganizer(
		lang.CompCmd != nil,
		numTests,
	)
	if err != nil {
		return err
	}
	// we need to get a sync map of organizers
	e.organizers[execId] = org

	exec := Execution{
		UUID:      execId,
		Stage:     StageWaiting,
		TestRes:   []TestRes{},
		PrLang:    lang,
		Params:    params,
		ErrorMsg:  nil,
		SysInfo:   nil,
		CreatedAt: time.Now(),
		SubmComp:  nil,
	}
	// insert empty tests
	for i := 0; i < numTests; i++ {
		exec.TestRes = append(
			exec.TestRes,
			TestRes{ID: i + 1},
		)
	}
	e.executions[execId] = &exec

	return nil
}

// based on incoming events. Handles various event
// types including compilation, testing, and error
// states
func applyEventToExec(
	exec *Execution,
	event Event,
) error {
	switch e := event.(type) {
	case ReceivedSubmission:
		exec.SysInfo = &e.SysInfo
	case StartedCompiling:
		exec.Stage = StageCompiling
	case FinishedCompiling:
		exec.Stage = StageFinished
		exec.SubmComp = e.RuntimeData
	case ReachedTest:
		exec.TestRes[e.TestId-1].Input = e.In
		exec.TestRes[e.TestId-1].Answer = e.Ans
		exec.TestRes[e.TestId-1].Reached = true
	case FinishedTest:
		exec.TestRes[e.TestID-1].Subm = e.Subm
		exec.TestRes[e.TestID-1].Checker = e.Checker
		exec.TestRes[e.TestID-1].Finished = true
	case IgnoredTest:
		exec.TestRes[e.TestId-1].Ignored = true
	case FinishedTesting:
		exec.Stage = StageFinished
	case InternalServerError:
		exec.Stage = StageInternalError
		exec.ErrorMsg = e.ErrorMsg
	case CompilationError:
		exec.Stage = StageCompileError
		exec.ErrorMsg = e.ErrorMsg
	}
	return nil
}

func (e *ExecSrvcImpl) Close() {
	e.logger.Info("closing execsrvc")
	e.listenCancel()
	e.listenWait.Wait()
	e.logger.Info("execsrvc closed")
}
