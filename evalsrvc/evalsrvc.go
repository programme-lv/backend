package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/emirpasic/gods/v2/queues"
	"github.com/emirpasic/gods/v2/queues/linkedlistqueue"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/tester"
	"github.com/puzpuzpuz/xsync/v3"
)

func (e *EvalSrvc) Enqueue(req Request, evalUuid uuid.UUID) (uuid.UUID, error) {
	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}
	return e.enqueueCommon(&req, evalUuid, e.resSqsUrl, lang)
}

// EnqueueExternal is used for evaluation requests from external services
// such as CleverCode.lv. The difference is that it creates a new response queue
// for each evaluation request.
func (e *EvalSrvc) EnqueueExternal(apiKey string, req Request) (uuid.UUID, error) {
	if apiKey != e.extEvalKey {
		return uuid.Nil, ErrInvalidApiKey()
	}

	// check validity of programming language before creating a new queue
	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return e.enqueueCommon(&req, evalUuid, e.extEvalSqsUrl, lang)
}

func (e *EvalSrvc) enqueueCommon(req *Request,
	evalUuid uuid.UUID,
	resSqsUrl string,
	lang *planglist.ProgrammingLang,
) (uuid.UUID, error) {
	tests := make([]tester.ReqTest, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = tester.ReqTest{
			ID:         int(test.ID),
			InSha256:   test.InSha256,
			InUrl:      test.InUrl,
			InContent:  test.InContent,
			AnsSha256:  test.AnsSha256,
			AnsUrl:     test.AnsUrl,
			AnsContent: test.AnsContent,
		}
	}
	jsonReq, err := json.Marshal(tester.EvalReq{
		EvalUuid:  evalUuid.String(),
		ResSqsUrl: resSqsUrl,
		Code:      req.Code,
		Language: tester.Language{
			LangID:        lang.ID,
			LangName:      lang.FullName,
			CodeFname:     lang.CodeFilename,
			CompileCmd:    lang.CompileCmd,
			CompiledFname: lang.CompiledFilename,
			ExecCmd:       lang.ExecuteCmd,
		},
		Tests:      tests,
		Checker:    req.Checker,
		Interactor: req.Interactor,
		CpuMillis:  req.CpuMs,
		MemoryKiB:  req.MemKiB,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal evaluation request: %w", err)
	}
	_, err = e.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(e.submSqsUrl),
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to send message to evaluation queue: %w", err)
	}

	return evalUuid, nil
}

func (e *EvalSrvc) ReceiveFrom(evalUuid uuid.UUID) ([]Msg, error) {
	// this is long polling. we retrieve channel for evalUuid
	nq := linkedlistqueue.New[Pair[Msg, time.Time]]()
	nc := sync.NewCond(&sync.Mutex{})
	p, _ := e.accumulated.LoadOrStore(evalUuid, Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]{First: nc, Second: nq})

	cond := p.First
	q := p.Second

	cond.L.Lock()

	vals := q.Values()
	q.Clear()

	if len(vals) > 0 {
		msgs := make([]Msg, len(vals))
		for i, val := range vals {
			msgs[i] = val.First
		}
		cond.L.Unlock()
		return msgs, nil
	}

	cond.L.Unlock()

	// if the channel is empty, wait at most 5 seconds for a message
	// check every 50ms
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// check if there are any messages in the queue and if so, return them
			cond.L.Lock()
			vals := q.Values()
			q.Clear()
			if len(vals) > 0 {
				msgs := make([]Msg, len(vals))
				for i, val := range vals {
					msgs[i] = val.First
				}
				cond.L.Unlock()
				return msgs, nil
			}
			cond.L.Unlock()
		case <-timer.C:
			return []Msg{}, nil
		}
	}
}

func (e *EvalSrvc) Receive() ([]Msg, error) {
	return e.receive10MsgsFrom(e.resSqsUrl)
}

func (e *EvalSrvc) Ack(queueUrl string, handle string) error {
	_, err := e.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueUrl),
		ReceiptHandle: aws.String(handle),
	})
	return err
}

type Event interface {
	Type() string
}

type Msg struct {
	EvalId uuid.UUID
	Queue  string // url of queue it was received from
	Handle string // receipt handle for acknowledgment / delete
	Data   Event  // data specific to the message / event type
}

type Pair[T1 any, T2 any] struct {
	First  T1
	Second T2
}

type EvalSrvc struct {
	sqsClient *sqs.Client

	submSqsUrl    string
	resSqsUrl     string
	extEvalSqsUrl string

	extEvalKey string // api key for external evaluation requests

	accumulated *xsync.MapOf[uuid.UUID, Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]]
}

func NewEvalSrvc() *EvalSrvc {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	sqsClient := sqs.NewFromConfig(cfg)

	submQueueUrl := os.Getenv("SUBM_SQS_QUEUE_URL")
	if submQueueUrl == "" {
		panic("SUBM_SQS_QUEUE_URL not set in .env file")
	}

	responseSQSURL := os.Getenv("RESPONSE_SQS_URL")
	if responseSQSURL == "" {
		panic("RESPONSE_SQS_URL not set in .env file")
	}

	extEvalKey := os.Getenv("EXTERNAL_EVAL_KEY")
	if extEvalKey == "" {
		panic("EXTERNAL_EVAL_KEY not set in .env file")
	}

	extEvalSqsUrl := os.Getenv("EXT_EVAL_SQS_URL")
	if extEvalSqsUrl == "" {
		panic("EXT_EVAL_SQS_URL not set in .env file")
	}

	esrvc := &EvalSrvc{
		sqsClient:     sqsClient,
		submSqsUrl:    submQueueUrl,
		resSqsUrl:     responseSQSURL,
		extEvalKey:    extEvalKey,
		extEvalSqsUrl: extEvalSqsUrl,
		accumulated:   xsync.NewMapOf[uuid.UUID, Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]](),
	}

	return esrvc
}

// this may not be pretty but'll work
func (e *EvalSrvc) StartReceivingFromExternalEvalQueue() {
	for {
		msgs, err := e.receive10MsgsFrom(e.extEvalSqsUrl)
		if err != nil {
			log.Printf("error receiving from external eval queue: %v", err)
		}
		for _, msg := range msgs {
			nq := linkedlistqueue.New[Pair[Msg, time.Time]]() // new queue
			nc := sync.NewCond(&sync.Mutex{})                 // new condition variable
			p, _ := e.accumulated.LoadOrStore(msg.EvalId, Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]{First: nc, Second: nq})
			cond := p.First
			cond.L.Lock()
			q := p.Second
			q.Enqueue(Pair[Msg, time.Time]{First: msg, Second: time.Now()})
			cond.L.Unlock()
			// cond.Broadcast()
		}
	}
}

func (e *EvalSrvc) StartDeletingOldMessages() {
	for {
		time.Sleep(1 * time.Minute)
		e.accumulated.Range(func(key uuid.UUID, value Pair[*sync.Cond, queues.Queue[Pair[Msg, time.Time]]]) bool {
			cond := value.First
			cond.L.Lock()
			defer cond.L.Unlock()

			q := value.Second

			for q.Size() > 0 {
				msg, ok := q.Peek()
				if !ok {
					return true
				}
				t := msg.Second
				if t.Before(time.Now().Add(-2 * time.Minute)) {
					q.Dequeue()
				} else {
					break
				}
			}
			return true
		})
	}
}
