package evalsrvc

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

type EvalRepo interface {
	Save(eval Evaluation) error
	Get(evalUuid uuid.UUID) (Evaluation, error)
	Delete(evalUuid uuid.UUID) error
}

type EvalSrvc struct {
	sqsClient *sqs.Client
	evalRepo  EvalRepo // either in-mem or s3

	submQ string // submission sqs queue url
	respQ string // response sqs queue url

	extEvalKey string // api key for external requests

	mu        sync.Mutex
	handlers  map[uuid.UUID]chan Event
	notifiers map[uuid.UUID]chan Event
	evalWg    sync.Map // maps uuid.UUID to *sync.WaitGroup
}

func NewEvalSrvc() *EvalSrvc {
	esrvc := &EvalSrvc{
		sqsClient:  getSqsClientFromEnv(),
		submQ:      getSubmSqsUrlFromEnv(),
		evalRepo:   NewInMemEvalRepo(),
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

// Enqueues code for evaluation by tester, returns eval uuid:
// 1. validates programming language;
// 2. validates cpu, mem constraints & checker, interactor size;
// 3. validates test files (max 200 tests);
// 4. constructs initial evaluation object;
// 5. initializes response stream processor with evaluation;
// 6. enqueues evaluation request to sqs
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
	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

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

// Returns a channel to stream events to a singular client.
// The same channel is returned for the same eval uuid.
// Once evaluation is finished, the channel is closed.
func (e *EvalSrvc) Listen(evalId uuid.UUID) (chan Event, error) {
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

func (e *EvalSrvc) Get(ctx context.Context, evalId uuid.UUID) (Evaluation, error) {
	// Get the WaitGroup for this evaluation
	wgVal, exists := e.evalWg.Load(evalId)
	if !exists {
		return Evaluation{}, fmt.Errorf("no evaluation found for id %s", evalId)
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
		return e.evalRepo.Get(evalId)
	case <-ctx.Done():
		return Evaluation{}, ctx.Err()
	}
}

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

// Initialize response stream organizer, processor and notifier.
func (e *EvalSrvc) prepareForResults(evalId uuid.UUID, lang PrLang, params TesterParams, numTests int) error {
	// initialize some kind of mysthical organizer that reorders events
	// the organizer has to know the number of tests and whether the submission has a compilation step
	e.handlers[evalId] = make(chan Event)
	e.notifiers[evalId] = make(chan Event, 1000)

	organizer, err := NewEvalResOrganizer(lang.CompCmd != nil, numTests)
	if err != nil {
		return fmt.Errorf("failed to create organizer: %v", err)
	}

	go e.handleResults(evalId, lang, params, organizer, numTests)
	return nil
}

func (e *EvalSrvc) handleResults(evalId uuid.UUID, lang PrLang, params TesterParams, org *EvalResOrganizer, numTests int) {
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
		if org.Finished() {
			break
		}
	}
	close(e.handlers[evalId])
	close(e.notifiers[evalId])
	delete(e.handlers, evalId)
	delete(e.notifiers, evalId)
	err := e.evalRepo.Save(eval)
	if err != nil {
		log.Printf("failed to save evaluation: %v", err)
		return
	}
	if wgVal, exists := e.evalWg.Load(evalId); exists {
		wg := wgVal.(*sync.WaitGroup)
		wg.Done()
	}
}

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
