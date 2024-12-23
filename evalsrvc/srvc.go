package evalsrvc

import (
	"context"
	"fmt"
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

	mu         sync.Mutex
	organizers map[uuid.UUID]chan Event
	processors map[uuid.UUID]chan Event
	notifiers  map[uuid.UUID]chan Event
}

func NewEvalSrvc() *EvalSrvc {
	esrvc := &EvalSrvc{
		sqsClient:  getSqsClientFromEnv(),
		submQ:      getSubmSqsUrlFromEnv(),
		evalRepo:   NewInMemEvalRepo(),
		respQ:      getResponseSqsUrlFromEnv(),
		extEvalKey: getExtEvalKeyFromEnv(),
	}

	go receiveResultsFromSqs(context.Background(),
		esrvc.respQ,
		esrvc.sqsClient,
		esrvc.handleSqsMsg,
	)

	return esrvc
}

// Enqueues code for evaluation by tester, returns eval uuid:
// 1. validates programming language;
// 2. validates cpu, mem constraints & checker, interactor size;
// 3. validates test files;
// 4. initializes response stream processor with evaluation;
// 5. enqueues evaluation request to sqs
func (e *EvalSrvc) Enqueue(
	code CodeWithLang,
	tests []TestFile,
	params TesterParams,
) (uuid.UUID, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	lang, err := getPrLangById(code.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	err = params.IsValid() // validate tester parameters
	if err != nil {
		return uuid.Nil, err
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	for _, test := range tests {
		if err := test.IsValid(); err != nil {
			return uuid.Nil, err
		}
	}

	testRes := []TestRes{}
	for i := range tests {
		testRes = append(testRes, TestRes{ID: i + 1})
	}

	e.foo(Evaluation{
		UUID:      evalUuid,
		Stage:     EvalStageWaiting,
		TestRes:   testRes,
		PrLang:    lang,
		Params:    params,
		ErrorMsg:  nil,
		SysInfo:   nil,
		CreatedAt: time.Now(),
		SubmComp:  nil,
	})

	err = enqueue(evalUuid, code.SrcCode, lang, tests, params,
		e.sqsClient, e.submQ, e.respQ)
	if err != nil {
		return uuid.Nil, err
	}

	return evalUuid, nil
}

// Returns a channel to stream events to the client.
// The same channel is returned for the same eval uuid.
// Once evaluation is finished, the channel is closed.
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

func (e *EvalSrvc) handleSqsMsg(msg SqsResponseMsg) error {
	e.mu.Lock()
	ch, ok := e.processors[msg.EvalId]
	e.mu.Unlock()
	if !ok {
		return nil // no listeners for this evaluation
	}
	ch <- msg.Data
	return nil
}

// initialize response stream reorderor
func (e *EvalSrvc) foo(eval Evaluation) {
	// initialize some kind of mysthical organizer that reorders events
	// the organizer has to know the number of tests and whether the submission has a compilation step
	// and so does the processor

	e.mu.Lock()
	defer e.mu.Unlock()

	e.organizers[eval.UUID] = make(chan Event)
	e.processors[eval.UUID] = make(chan Event)
	e.notifiers[eval.UUID] = make(chan Event, 100)
}
