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

// EvalRepo defines the interface for evaluation storage operations
type EvalRepo interface {
	Save(ctx context.Context, eval Evaluation) error
	Get(ctx context.Context, evalUuid uuid.UUID) (*Evaluation, error)
}

// EvalSrvc handles code evaluation workflow and manages communication
// between different components of the evaluation system
type EvalSrvc struct {
	logger *slog.Logger

	sqsClient *sqs.Client
	evalRepo  EvalRepo // either in-mem or s3

	submQ string // submission sqs queue url
	respQ string // response sqs queue url

	extEvalKey string // api key for external requests

	mu        sync.Mutex
	handlers  map[uuid.UUID]chan Event // maps eval IDs to their event handlers
	notifiers map[uuid.UUID]chan Event // maps eval IDs to client notification channels
	evalWg    sync.Map                 // tracks completion status of evaluations
}

// NewDefaultEvalSrvc creates an evaluation service with default configuration
// using environment variables for AWS services setup
func NewDefaultEvalSrvc() *EvalSrvc {
	logger := slog.Default().With("module", "eval")
	s3Repo := NewS3EvalRepo(logger, getS3ClientFromEnv(), getEvalS3BucketFromEnv())

	esrvc := &EvalSrvc{
		logger:     logger,
		sqsClient:  getSqsClientFromEnv(),
		submQ:      getSubmSqsUrlFromEnv(),
		evalRepo:   s3Repo,
		respQ:      getResponseSqsUrlFromEnv(),
		extEvalKey: getExtEvalKeyFromEnv(),
		handlers:   make(map[uuid.UUID]chan Event),
		notifiers:  make(map[uuid.UUID]chan Event),
	}

	go receiveResultsFromSqs(context.Background(),
		esrvc.respQ,
		esrvc.sqsClient,
		esrvc.handleSqsMsg,
		slog.Default(),
	)

	return esrvc
}

// NewEvalSrvc creates a customized evaluation service with provided dependencies
func NewEvalSrvc(
	logger *slog.Logger,
	sqsClient *sqs.Client,
	submQ string,
	evalRepo EvalRepo,
	respQ string,
	extEvalKey string,
) *EvalSrvc {
	return &EvalSrvc{
		logger:     logger,
		sqsClient:  sqsClient,
		submQ:      submQ,
		evalRepo:   evalRepo,
		respQ:      respQ,
		extEvalKey: extEvalKey,
		handlers:   make(map[uuid.UUID]chan Event),
		notifiers:  make(map[uuid.UUID]chan Event),
	}
}

// Enqueue processes a code evaluation request by:
// 1. Validating the programming language and constraints
// 2. Setting up result handlers and notification channels
// 3. Sending the evaluation request to the processing queue
// Returns the evaluation UUID for tracking
func (e *EvalSrvc) Enqueue(
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

	// 4. construct evaluation object
	evalUuid := uuid.New()

	// Add WaitGroup before preparing results
	wg := &sync.WaitGroup{}
	wg.Add(1)
	e.evalWg.Store(evalUuid, wg)

	// 5. initialize organizer, processor and notifier
	err = e.prepareForResults(evalUuid, lang, params, len(tests))
	if err != nil {
		return uuid.Nil, err
	}

	// 6. enqueue evaluation request to sqs
	err = enqueue(evalUuid, code.SrcCode, lang, tests, params,
		e.sqsClient, e.submQ, e.respQ)
	if err != nil {
		return uuid.Nil, err
	}

	return evalUuid, nil
}

// Listen returns a channel that streams evaluation events to clients
// The channel is automatically closed once the evaluation is complete
func (e *EvalSrvc) Listen(evalId uuid.UUID) (<-chan Event, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	ch, ok := e.notifiers[evalId]
	if !ok {
		format := "no listener for eval %s"
		errMsg := fmt.Errorf(format, evalId)
		return nil, errMsg
	}
	return ch, nil
}

// Get retrieves the evaluation results for a given evaluation ID
// It waits for completion if the evaluation is still in progress
func (e *EvalSrvc) Get(ctx context.Context, evalId uuid.UUID) (Evaluation, error) {
	// Get the WaitGroup for this evaluation
	wgVal, exists := e.evalWg.Load(evalId)
	if !exists {
		eval, err := e.evalRepo.Get(ctx, evalId)
		if err != nil {
			return Evaluation{}, fmt.Errorf("no evaluation found for id %s", evalId)
		}
		return *eval, nil
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
		e.evalWg.Delete(evalId) // Clean up the WaitGroup
		eval, err := e.evalRepo.Get(ctx, evalId)
		if err != nil {
			return Evaluation{}, err
		}
		return *eval, nil
	case <-ctx.Done():
		return Evaluation{}, ctx.Err()
	}
}

// handleSqsMsg processes incoming SQS messages and routes them to appropriate handlers
func (e *EvalSrvc) handleSqsMsg(msg SqsResponseMsg) error {
	e.mu.Lock()
	ch, ok := e.handlers[msg.EvalId]
	e.mu.Unlock()
	if !ok {
		errMsg := fmt.Errorf("no handler for eval %s", msg.EvalId)
		return errMsg // returning error to indicate that the message was not processed
	}
	ch <- msg.Data
	return nil
}

// prepareForResults sets up the event processing pipeline for an evaluation
// including result organization and client notification channels
func (e *EvalSrvc) prepareForResults(evalId uuid.UUID, lang PrLang, params TesterParams, numTests int) error {
	// initialize some kind of mysthical organizer that reorders events
	// the organizer has to know the number of tests and whether the submission has a compilation step
	e.handlers[evalId] = make(chan Event)
	e.notifiers[evalId] = make(chan Event, 1000)

	organizer, err := NewExecResStreamOrganizer(lang.CompCmd != nil, numTests)
	if err != nil {
		return fmt.Errorf("failed to create organizer: %v", err)
	}

	go e.handleResultStreamForEval(evalId, lang, params, organizer, numTests)
	return nil
}

// handleResultStreamForEval manages the evaluation lifecycle by:
// - Processing incoming events
// - Updating evaluation state
// - Managing client notifications
// - Persisting final results
func (e *EvalSrvc) handleResultStreamForEval(evalId uuid.UUID, lang PrLang, params TesterParams, org *ExecResStreamOrganizer, numTests int) {
	eval := Evaluation{
		UUID:      evalId,
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
		eval.TestRes = append(eval.TestRes, TestRes{ID: i + 1})
	}
	for ev := range e.handlers[evalId] {
		events, err := org.Add(ev)
		if err != nil {
			log.Printf("failed to process event: %v", err)
			return
		}
		for _, event := range events {
			err := applyEventToEval(&eval, event)
			if err != nil {
				log.Printf("failed to apply event: %v", err)
				return
			}
			e.notifiers[evalId] <- event
		}
		if org.HasFinished() {
			break
		}
	}
	close(e.handlers[evalId])
	close(e.notifiers[evalId])
	delete(e.handlers, evalId)
	delete(e.notifiers, evalId)
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := e.evalRepo.Save(ctxWithTimeout, eval)
	if err != nil {
		slog.Error("failed to save evaluation", "error", err)
		return
	}
	if wgVal, exists := e.evalWg.Load(evalId); exists {
		wg := wgVal.(*sync.WaitGroup)
		wg.Done()
	}
}

// applyEventToEval updates the evaluation state based on incoming events
// Handles various event types including compilation, testing, and error states
func applyEventToEval(eval *Evaluation, event Event) error {
	switch event.Type() {
	case ReceivedSubmissionType:
		rcvSubm, ok := event.(ReceivedSubmission)
		if !ok {
			return fmt.Errorf("event is not a ReceivedSubmission")
		}
		eval.SysInfo = &rcvSubm.SysInfo
	case StartedCompilationType:
		eval.Stage = StageCompiling
	case FinishedCompilationType:
		finComp, ok := event.(FinishedCompiling)
		if !ok {
			return fmt.Errorf("event is not a FinishedCompiling")
		}
		eval.SubmComp = finComp.RuntimeData
	case StartedTestingType:
		eval.Stage = StageTesting
	case ReachedTestType:
		rt, ok := event.(ReachedTest)
		if !ok {
			return fmt.Errorf("event is not a ReachedTest")
		}
		eval.TestRes[rt.TestId-1].Input = rt.In
		eval.TestRes[rt.TestId-1].Answer = rt.Ans
		eval.TestRes[rt.TestId-1].Reached = true
	case FinishedTestType:
		ft, ok := event.(FinishedTest)
		if !ok {
			return fmt.Errorf("event is not a FinishedTest")
		}
		eval.TestRes[ft.TestID-1].ProgramReport = ft.Subm
		eval.TestRes[ft.TestID-1].CheckerReport = ft.Checker
		eval.TestRes[ft.TestID-1].Finished = true
	case IgnoredTestType:
		ig, ok := event.(IgnoredTest)
		if !ok {
			return fmt.Errorf("event is not an IgnoredTest")
		}
		eval.TestRes[ig.TestId-1].Ignored = true
	case FinishedTestingType:
		eval.Stage = StageFinished
	case InternalServerErrorType:
		eval.Stage = StageInternalError
		ise, ok := event.(InternalServerError)
		if !ok {
			return fmt.Errorf("event is not an InternalServerError")
		}
		eval.ErrorMsg = ise.ErrorMsg
	case CompilationErrorType:
		eval.Stage = StageCompileError
		ce, ok := event.(CompilationError)
		if !ok {
			return fmt.Errorf("event is not a CompilationError")
		}
		eval.ErrorMsg = ce.ErrorMsg
	}
	return nil
}
