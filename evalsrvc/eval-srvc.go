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
	testRes := []TestRes{}
	for i := range tests {
		testRes = append(testRes, TestRes{ID: i + 1})
	}
	eval := Evaluation{
		UUID:      evalUuid,
		Stage:     EvalStageWaiting,
		TestRes:   testRes,
		PrLang:    lang,
		Params:    params,
		ErrorMsg:  nil,
		SysInfo:   nil,
		CreatedAt: time.Now(),
		SubmComp:  nil,
	}

	// 5. initialize organizer, processor and notifier
	e.prepareForResults(eval)

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
func (e *EvalSrvc) prepareForResults(eval Evaluation) {
	// initialize some kind of mysthical organizer that reorders events
	// the organizer has to know the number of tests and whether the submission has a compilation step
	e.handlers[eval.UUID] = make(chan Event)
	e.notifiers[eval.UUID] = make(chan Event, 1000)

	hasCompilation := eval.PrLang.CompCmd != nil
	numTests := len(eval.TestRes)
	organizer, err := NewEvalResOrganizer(hasCompilation, numTests)
	if err != nil {
		log.Printf("failed to create organizer: %v", err)
		return
	}

	go func() {
		for ev := range e.handlers[eval.UUID] {
			events, err := organizer.Add(ev)
			if err != nil {
				log.Printf("failed to process event: %v", err)
				return
			}
			for _, event := range events {
				e.notifiers[eval.UUID] <- event
			}
			if organizer.Finished() {
				close(e.handlers[eval.UUID])
				close(e.notifiers[eval.UUID])
				delete(e.handlers, eval.UUID)
				delete(e.notifiers, eval.UUID)
			}
		}
	}()
}
