package execsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

// ExecRepo interface for execution storage
type ExecRepo interface {
	Save(
		ctx context.Context,
		exec Execution,
	) error
	Get(
		ctx context.Context,
		id uuid.UUID,
	) (*Execution, error)
}

// ExecSrvc handles communication with testers
// for code execution and result streaming
type ExecSrvc struct {
	logger *slog.Logger

	sqsClient *sqs.Client
	// either in-mem or s3
	execRepo ExecRepo

	// submission sqs queue url
	submQ string
	// response sqs queue url
	respQ string

	// external partner api key
	extPartnerPw string

	mu sync.Mutex
	// maps exec IDs to their event handlers
	handlers map[uuid.UUID]chan Event
	// maps exec IDs to client result channels
	notifiers map[uuid.UUID]chan Event
	// tracks completion status of executions
	execWg sync.Map // notifies get listener when execution is finished

	listenCancel context.CancelFunc
	listenWait   sync.WaitGroup // on close, waits for sqs jobs to finish
}

// NewDefaultExecSrvc creates an execution service
// with default configuration using environment
// variables for AWS services setup
func NewDefaultExecSrvc() *ExecSrvc {
	logger := slog.Default().With(
		"module",
		"exec",
	)
	s3Repo := NewS3ExecRepo(
		logger,
		getS3ClientFromEnv(),
		getExecS3BucketFromEnv(),
	)

	esrvc := &ExecSrvc{
		logger:       logger,
		sqsClient:    getSqsClientFromEnv(),
		submQ:        getSubmSqsUrlFromEnv(),
		execRepo:     s3Repo,
		respQ:        getResponseSqsUrlFromEnv(),
		extPartnerPw: getExtPartnerPwFromEnv(),
		handlers: make(
			map[uuid.UUID]chan Event,
		),
		notifiers: make(
			map[uuid.UUID]chan Event,
		),
	}

	esrvc.listenWait.Add(1)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		esrvc.listenCancel = cancel
		defer cancel()
		err := StartReceivingResultsFromSqs(
			ctx,
			esrvc.respQ,
			esrvc.sqsClient,
			esrvc.handleSqsMsg,
			esrvc.logger,
		)
		if err != nil {
			slog.Error("failed to listen for sqs messages", "error", err)
		}
		esrvc.listenWait.Done()
	}()

	return esrvc
}

// NewExecSrvc creates a customized execution
// service with provided dependencies
func NewExecSrvc(
	logger *slog.Logger,
	sqsClient *sqs.Client,
	submQ string,
	execRepo ExecRepo,
	respQ string,
	extPartnerPw string,
) *ExecSrvc {
	return &ExecSrvc{
		logger:       logger,
		sqsClient:    sqsClient,
		submQ:        submQ,
		execRepo:     execRepo,
		respQ:        respQ,
		extPartnerPw: extPartnerPw,
		handlers: make(
			map[uuid.UUID]chan Event,
		),
		notifiers: make(
			map[uuid.UUID]chan Event,
		),
	}
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
func (e *ExecSrvc) Enqueue(
	code CodeWithLang,
	tests []TestFile,
	params TesterParams,
) (uuid.UUID, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 1. validate programming language
	lang, err := getPrLangById(code.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	// 2. validate tester execution constraints,
	// checker
	err = params.IsValid()
	if err != nil {
		return uuid.Nil, err
	}

	// 3. validate test files
	if len(tests) > 200 {
		return uuid.Nil, fmt.Errorf(
			"too many tests",
		)
	}
	for _, test := range tests {
		if err := test.IsValid(); err != nil {
			return uuid.Nil, err
		}
	}

	// 4. construct execution object
	execUuid := uuid.New()

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
		return uuid.Nil, err
	}

	// 6. enqueue execution request to sqs
	err = enqueue(
		execUuid,
		code.SrcCode,
		lang,
		tests,
		params,
		e.sqsClient,
		e.submQ,
		e.respQ,
	)
	if err != nil {
		return uuid.Nil, err
	}

	return execUuid, nil
}

// Listen returns a channel that streams execution
// events to clients. The channel is automatically
// closed once the execution is complete
func (e *ExecSrvc) Listen(
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
func (e *ExecSrvc) Get(
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
func (e *ExecSrvc) handleSqsMsg(
	msg SqsResponseMsg,
) error {
	e.logger.Info("locking mu")
	e.mu.Lock()
	e.logger.Info("locked mu")
	ch, ok := e.handlers[msg.ExecId]
	defer e.mu.Unlock()
	if !ok {
		errMsg := fmt.Errorf(
			"no handler for exec %s",
			msg.ExecId,
		)
		// returning error to indicate that the
		// message was not processed
		return errMsg
	}

	// Try sending on channel, return error if closed
	e.logger.Info("sending event to handler", "event", msg.Data)
	ch <- msg.Data
	e.logger.Info("event sent to handler", "event", msg.Data)
	return nil
}

// prepareForResults sets up the event processing
// pipeline for an execution including result
// organization and client notification channels
func (e *ExecSrvc) prepareForResults(
	execId uuid.UUID,
	lang PrLang,
	params TesterParams,
	numTests int,
) error {
	// initialize some kind of mysthical organizer
	// that reorders events the organizer has to
	// know the number of tests and whether the
	// submission has a compilation step
	e.handlers[execId] = make(chan Event)
	e.notifiers[execId] = make(
		chan Event,
		1000,
	)

	go e.handleResultStreamForExec(
		execId,
		lang,
		params,
		numTests,
	)
	return nil
}

// handleResultStreamForExec manages the execution
// lifecycle by:
// - Processing incoming events
// - Updating execution state
// - Managing client notifications
// - Persisting final results
func (e *ExecSrvc) handleResultStreamForExec(
	execId uuid.UUID,
	lang PrLang,
	params TesterParams,
	numTests int,
) {
	org, err := NewExecResStreamOrganizer(
		lang.CompCmd != nil,
		numTests,
	)
	if err != nil {
		slog.Error(
			"failed to create organizer",
			"error",
			err,
		)
		return
	}

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
	e.mu.Lock()
	ch := e.handlers[execId]
	e.mu.Unlock()
	for ev := range ch {
		events, err := org.Add(ev)
		if err != nil {
			slog.Error(
				"failed to process event",
				"error",
				err,
			)
			return
		}
		for _, event := range events {
			err := applyEventToExec(&exec, event)
			if err != nil {
				slog.Error(
					"failed to apply event",
					"error",
					err,
				)
				return
			}
			e.notifiers[execId] <- event
			// make sure to delete the notifier channel 10 minutes after receiving the first event

		}
		if org.HasFinished() {
			break
		}
	}
	e.mu.Lock()
	close(e.handlers[execId])
	close(e.notifiers[execId])
	delete(e.handlers, execId)  // deleting closes the channel
	delete(e.notifiers, execId) // deleting closes the channel
	e.mu.Unlock()

	ctxWithTimeout, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()
	err = e.execRepo.Save(ctxWithTimeout, exec)
	if err != nil {
		slog.Error(
			"failed to save execution",
			"error",
			err,
		)
		return
	}
	wgVal, exists := e.execWg.Load(execId)
	if exists {
		wg := wgVal.(*sync.WaitGroup)
		wg.Done()
	}
}

// applyEventToExec updates the execution state
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

func (e *ExecSrvc) Close() {
	e.logger.Info("closing execsrvc")
	e.listenCancel()
	e.listenWait.Wait()
	e.logger.Info("execsrvc closed")
}
