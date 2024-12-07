package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/emirpasic/gods/v2/queues"
	"github.com/emirpasic/gods/v2/queues/linkedlistqueue"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/tester"
	"github.com/programme-lv/tester/sqsgath"
)

// parameters needed to create a new evaluation request
type NewEvalParams struct {
	Code   string // user submitted solution source code
	LangId string // short compiler, interpreter id

	Tests []TestFile // test cases to run against the code

	CpuMs  int // maximum user-mode CPU time in milliseconds
	MemKiB int // maximum resident set size in kibibytes

	// optional testlib.h checker program. If not provided,
	// only output of the user's solution is returned from tester
	// and is not viable for grading. "run program" use case.
	Checker *string

	// optional testlib.h interactor program.
	Interactor *string
}

// Enqueue adds an evaluation request to the processing queue using a pre-generated UUID
func (e *EvalSrvc) Enqueue(req NewEvalParams, evalUuid uuid.UUID) (uuid.UUID, error) {
	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}
	err = e.enqueue(&req, evalUuid, e.resSqsUrl, lang)
	if err != nil {
		return uuid.Nil, err
	}
	return evalUuid, nil
}

// EnqueueExternal adds an external evaluation request to a separate queue after API key validation
func (e *EvalSrvc) EnqueueExternal(apiKey string, req NewEvalParams) (uuid.UUID, error) {
	// check validity of programming language before creating a new queue
	lang, err := planglist.GetProgrammingLanguageById(req.LangId)
	if err != nil {
		return uuid.Nil, err
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}

	return evalUuid, e.enqueue(&req, evalUuid, e.extEvalSqsUrl, lang)
}

// enqueue handles the common logic of sending an evaluation request to an SQS queue.
// It converts the request into the tester's format and sends it as a JSON message.
func (e *EvalSrvc) enqueue(req *NewEvalParams,
	evalUuid uuid.UUID,
	resSqsUrl string,
	lang *planglist.ProgrammingLang,
) error {
	// Convert tests to tester format
	tests := make([]tester.ReqTest, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = tester.ReqTest{
			ID:         i + 1,
			InSha256:   test.InSha256,
			InUrl:      test.InDownlUrl,
			InContent:  test.InContent,
			AnsSha256:  test.AnsSha256,
			AnsUrl:     test.AnsDownlUrl,
			AnsContent: test.AnsContent,
		}
	}

	// Prepare evaluation request
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
		format := "failed to marshal evaluation request: %w"
		errMsg := fmt.Errorf(format, err)
		return errMsg
	}

	// Send to SQS queue
	_, err = e.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(e.submSqsUrl),
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		format := "failed to send message to evaluation queue: %w"
		errMsg := fmt.Errorf(format, err)
		return errMsg
	}

	return nil
}

func (e *EvalSrvc) ReceiveFor(evalUuid uuid.UUID) ([]Msg, error) {
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
	return e.receiveFromSqs(e.resSqsUrl)
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
	EvalId   uuid.UUID
	QueueUrl string // url of queue it was received from
	Handle   string // receipt handle for acknowledgment / delete
	Data     Event  // data specific to the message / event type
}

func (e *EvalSrvc) StartReceivingFromExternalEvalQueue() {
	for {
		msgs, err := e.receiveFromSqs(e.extEvalSqsUrl)
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

func (e *EvalSrvc) receiveFromSqs(queueUrl string) ([]Msg, error) {
	output, err := e.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueUrl),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     5,
	})
	if err != nil {
		log.Printf("failed to receive messages, %v\n", err)
		time.Sleep(1 * time.Second)
		return nil, err
	}
	msgs := make([]Msg, len(output.Messages))
	for i, msg := range output.Messages {
		if msg.Body == nil {
			return nil, fmt.Errorf("message body is nil")
		}

		var header sqsgath.Header
		err = json.Unmarshal([]byte(*msg.Body), &header)
		if err != nil {
			log.Printf("failed to unmarshal message: %v\n", err)
			continue
		}

		if msg.ReceiptHandle == nil {
			return nil, fmt.Errorf("receipt handle is nil")
		}
		msgs[i].Handle = *msg.ReceiptHandle
		msgs[i].QueueUrl = queueUrl
		msgs[i].EvalId, err = uuid.Parse(header.EvalUuid)
		if err != nil {
			return nil, fmt.Errorf("failed to parse eval_uuid: %w", err)
		}

		switch header.MsgType {
		case sqsgath.MsgTypeStartedEvaluation:
			startedEvaluation := sqsgath.StartedEvaluation{}
			err = json.Unmarshal([]byte(*msg.Body), &startedEvaluation)
			startedAt, err := time.Parse(time.RFC3339, startedEvaluation.StartedTime)
			if err != nil {
				return nil, fmt.Errorf("failed to parse started_at: %w", err)
			}
			msgs[i].Data = StartedEvaluation{
				SysInfo:   startedEvaluation.SystemInfo,
				StartedAt: startedAt,
			}
		case sqsgath.MsgTypeStartedCompilation:
			startedCompilation := sqsgath.StartedCompilation{}
			err = json.Unmarshal([]byte(*msg.Body), &startedCompilation)
			msgs[i].Data = StartedCompiling{}
		case sqsgath.MsgTypeFinishedCompilation:
			finishedCompilation := sqsgath.FinishedCompilation{}
			err = json.Unmarshal([]byte(*msg.Body), &finishedCompilation)
			msgs[i].Data = FinishedCompiling{
				RuntimeData: mapRunData(finishedCompilation.RuntimeData),
			}
		case sqsgath.MsgTypeStartedTesting:
			startedTesting := sqsgath.StartedTesting{}
			err = json.Unmarshal([]byte(*msg.Body), &startedTesting)
			msgs[i].Data = StartedTesting{}
		case sqsgath.MsgTypeReachedTest:
			reachedTest := sqsgath.ReachedTest{}
			err = json.Unmarshal([]byte(*msg.Body), &reachedTest)
			msgs[i].Data = ReachedTest{
				TestId: reachedTest.TestId,
				In:     reachedTest.Input,
				Ans:    reachedTest.Answer,
			}
		case sqsgath.MsgTypeIgnoredTest:
			ignoredTest := sqsgath.IgnoredTest{}
			err = json.Unmarshal([]byte(*msg.Body), &ignoredTest)
			msgs[i].Data = IgnoredTest{
				TestId: ignoredTest.TestId,
			}
		case sqsgath.MsgTypeFinishedTest:
			finishTest := sqsgath.FinishedTest{}
			err = json.Unmarshal([]byte(*msg.Body), &finishTest)
			msgs[i].Data = FinishedTest{
				TestID:  finishTest.TestId,
				Subm:    mapRunData(finishTest.Submission),
				Checker: mapRunData(finishTest.Checker),
			}
		case sqsgath.MsgTypeFinishedTesting:
			finishTesting := sqsgath.FinishedTesting{}
			err = json.Unmarshal([]byte(*msg.Body), &finishTesting)
			msgs[i].Data = FinishedTesting{}
		case sqsgath.MsgTypeFinishedEvaluation:
			finishEval := sqsgath.FinishedEvaluation{}
			err = json.Unmarshal([]byte(*msg.Body), &finishEval)
			msgs[i].Data = FinishedEvaluation{
				CompileError:  finishEval.CompileError,
				InternalError: finishEval.InternalError,
				ErrorMsg:      finishEval.ErrorMessage,
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s message: %w", header.MsgType, err)
		}
	}
	return msgs, nil
}

func mapRunData(rd *sqsgath.RuntimeData) *RunData {
	if rd != nil {
		return &RunData{
			StdIn:    rd.Stdin,
			StdOut:   rd.Stdout,
			StdErr:   rd.Stderr,
			CpuMs:    rd.CpuMillis,
			WallMs:   rd.WallMillis,
			MemKiB:   rd.MemoryKiBytes,
			ExitCode: rd.ExitCode,
			CtxSwV:   rd.CtxSwF,
			CtxSwF:   rd.CtxSwF,
			Signal:   rd.ExitSignal,
		}
	}
	return nil
}
