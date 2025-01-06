package execsrvc

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

// ExecRepo defines the interface for execution storage operations
type ExecRepo interface {
	Save(ctx context.Context, exec Execution) error
	Get(ctx context.Context, id uuid.UUID) (*Execution, error)
}

// ExecSrvc handles code execution workflow and manages communication
// between different components of the execution system
type ExecSrvc struct {
	logger *slog.Logger

	sqsClient *sqs.Client
	execRepo  ExecRepo // either in-mem or s3

	submQ string // submission sqs queue url
	respQ string // response sqs queue url

	extPartnerPw string // api key for external requests

	mu        sync.Mutex
	handlers  map[uuid.UUID]chan Event // maps exec IDs to their event handlers
	notifiers map[uuid.UUID]chan Event // maps exec IDs to client notification channels
	execWg    sync.Map                 // tracks completion status of executions
}

// NewDefaultExecSrvc creates an execution service with default configuration
// using environment variables for AWS services setup
func NewDefaultExecSrvc() *ExecSrvc {
	logger := slog.Default().With("module", "exec")
	s3Repo := NewS3ExecRepo(logger, getS3ClientFromEnv(), getExecS3BucketFromEnv())

	esrvc := &ExecSrvc{
		logger:       logger,
		sqsClient:    getSqsClientFromEnv(),
		submQ:        getSubmSqsUrlFromEnv(),
		execRepo:     s3Repo,
		respQ:        getResponseSqsUrlFromEnv(),
		extPartnerPw: getExtPartnerPwFromEnv(),
		handlers:     make(map[uuid.UUID]chan Event),
		notifiers:    make(map[uuid.UUID]chan Event),
	}

	go StartReceivingResultsFromSqs(context.Background(),
		esrvc.respQ,
		esrvc.sqsClient,
		esrvc.handleSqsMsg,
		slog.Default(),
	)

	return esrvc
}

// NewExecSrvc creates a customized execution service with provided dependencies
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
		handlers:     make(map[uuid.UUID]chan Event),
		notifiers:    make(map[uuid.UUID]chan Event),
	}
}

// Enqueue processes a code execution request by:
// 1. Validating the programming language and constraints
// 2. Setting up result handlers and notification channels
// 3. Sending the execution request to the processing queue
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

	// 2. validate tester execution constraints, checker
	err = params.IsValid() // validate tester parameters
	if err != nil {
		return uuid.Nil, err
	}

	// 3. validate test files
	if len(tests) > 200 {
		return uuid.Nil, fmt.Errorf("too many tests")
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

	// 5. initialize organizer, processor and notifier
	err = e.prepareForResults(execUuid, lang, params, len(tests))
	if err != nil {
		return uuid.Nil, err
	}

	// 6. enqueue execution request to sqs
	err = enqueue(execUuid, code.SrcCode, lang, tests, params,
		e.sqsClient, e.submQ, e.respQ)
	if err != nil {
		return uuid.Nil, err
	}

	return execUuid, nil
}

// Listen returns a channel that streams execution events to clients
// The channel is automatically closed once the execution is complete
func (e *ExecSrvc) Listen(execId uuid.UUID) (<-chan Event, error) {
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

// Get retrieves the execution results for a given execution ID
// It waits for completion if the execution is still in progress
func (e *ExecSrvc) Get(ctx context.Context, execId uuid.UUID) (Execution, error) {
	// Get the WaitGroup for this execution
	wgVal, exists := e.execWg.Load(execId)
	if !exists {
		exec, err := e.execRepo.Get(ctx, execId)
		if err != nil {
			return Execution{}, fmt.Errorf("no execution found for id %s", execId)
		}
		return *exec, nil
	}

	wg := wgVal.(*sync.WaitGroup)

	// Wait for completion with context cancellation support
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.execWg.Delete(execId) // Clean up the WaitGroup
		exec, err := e.execRepo.Get(ctx, execId)
		if err != nil {
			return Execution{}, err
		}
		return *exec, nil
	case <-ctx.Done():
		return Execution{}, ctx.Err()
	}
}

// handleSqsMsg processes incoming SQS messages and routes them to appropriate handlers
func (e *ExecSrvc) handleSqsMsg(msg SqsResponseMsg) error {
	e.mu.Lock()
	ch, ok := e.handlers[msg.ExecId]
	e.mu.Unlock()
	if !ok {
		errMsg := fmt.Errorf("no handler for exec %s", msg.ExecId)
		return errMsg // returning error to indicate that the message was not processed
	}
	ch <- msg.Data
	return nil
}

// prepareForResults sets up the event processing pipeline for an execution
// including result organization and client notification channels
func (e *ExecSrvc) prepareForResults(execId uuid.UUID, lang PrLang, params TesterParams, numTests int) error {
	// initialize some kind of mysthical organizer that reorders events
	// the organizer has to know the number of tests and whether the submission has a compilation step
	e.handlers[execId] = make(chan Event)
	e.notifiers[execId] = make(chan Event, 1000)

	organizer, err := NewExecResStreamOrganizer(lang.CompCmd != nil, numTests)
	if err != nil {
		return fmt.Errorf("failed to create organizer: %v", err)
	}

	go e.handleResultStreamForExec(execId, lang, params, organizer, numTests)
	return nil
}

// handleResultStreamForExec manages the execution lifecycle by:
// - Processing incoming events
// - Updating execution state
// - Managing client notifications
// - Persisting final results
func (e *ExecSrvc) handleResultStreamForExec(execId uuid.UUID, lang PrLang, params TesterParams, org *ExecResStreamOrganizer, numTests int) {
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
		exec.TestRes = append(exec.TestRes, TestRes{ID: i + 1})
	}
	for ev := range e.handlers[execId] {
		events, err := org.Add(ev)
		if err != nil {
			log.Printf("failed to process event: %v", err)
			return
		}
		for _, event := range events {
			err := applyEventToExec(&exec, event)
			if err != nil {
				log.Printf("failed to apply event: %v", err)
				return
			}
			e.notifiers[execId] <- event
		}
		if org.HasFinished() {
			break
		}
	}
	close(e.handlers[execId])
	close(e.notifiers[execId])
	delete(e.handlers, execId)
	delete(e.notifiers, execId)
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := e.execRepo.Save(ctxWithTimeout, exec)
	if err != nil {
		slog.Error("failed to save execution", "error", err)
		return
	}
	if wgVal, exists := e.execWg.Load(execId); exists {
		wg := wgVal.(*sync.WaitGroup)
		wg.Done()
	}
}

// applyEventToExec updates the execution state based on incoming events
// Handles various event types including compilation, testing, and error states
func applyEventToExec(exec *Execution, event Event) error {
	switch event.Type() {
	case ReceivedSubmissionType:
		rcvSubm, ok := event.(ReceivedSubmission)
		if !ok {
			return fmt.Errorf("event is not a ReceivedSubmission")
		}
		exec.SysInfo = &rcvSubm.SysInfo
	case StartedCompilationType:
		exec.Stage = StageCompiling
	case FinishedCompilationType:
		finComp, ok := event.(FinishedCompiling)
		if !ok {
			return fmt.Errorf("event is not a FinishedCompiling")
		}
		exec.SubmComp = finComp.RuntimeData
	case StartedTestingType:
		exec.Stage = StageTesting
	case ReachedTestType:
		rt, ok := event.(ReachedTest)
		if !ok {
			return fmt.Errorf("event is not a ReachedTest")
		}
		exec.TestRes[rt.TestId-1].Input = rt.In
		exec.TestRes[rt.TestId-1].Answer = rt.Ans
		exec.TestRes[rt.TestId-1].Reached = true
	case FinishedTestType:
		ft, ok := event.(FinishedTest)
		if !ok {
			return fmt.Errorf("event is not a FinishedTest")
		}
		exec.TestRes[ft.TestID-1].ProgramReport = ft.Subm
		exec.TestRes[ft.TestID-1].CheckerReport = ft.Checker
		exec.TestRes[ft.TestID-1].Finished = true
	case IgnoredTestType:
		ig, ok := event.(IgnoredTest)
		if !ok {
			return fmt.Errorf("event is not an IgnoredTest")
		}
		exec.TestRes[ig.TestId-1].Ignored = true
	case FinishedTestingType:
		exec.Stage = StageFinished
	case InternalServerErrorType:
		exec.Stage = StageInternalError
		ise, ok := event.(InternalServerError)
		if !ok {
			return fmt.Errorf("event is not an InternalServerError")
		}
		exec.ErrorMsg = ise.ErrorMsg
	case CompilationErrorType:
		exec.Stage = StageCompileError
		ce, ok := event.(CompilationError)
		if !ok {
			return fmt.Errorf("event is not a CompilationError")
		}
		exec.ErrorMsg = ce.ErrorMsg
	}
	return nil
}
