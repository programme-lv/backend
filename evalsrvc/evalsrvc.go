package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/tester"
	"github.com/programme-lv/tester/sqsgath"
)

func (e *EvalSrvc) Enqueue(req Request) (uuid.UUID, error) {
	evalUuid, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to generate UUID: %w", err)
	}
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

	queue, err := e.sqsClient.CreateQueue(context.TODO(), &sqs.CreateQueueInput{
		QueueName: aws.String(evalUuid.String()),
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create queue: %w", err)
	}

	return e.enqueueCommon(&req, evalUuid, *queue.QueueUrl, lang)
}

func (e *EvalSrvc) enqueueCommon(req *Request,
	evalUuid uuid.UUID,
	resSqsUrl string,
	lang *planglist.ProgrammingLang,
) (uuid.UUID, error) {
	tests := make([]tester.ReqTest, len(req.Tests))
	for i, test := range req.Tests {
		tests[i] = tester.ReqTest{
			ID:         i,
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
	queueUrl, err := e.sqsClient.GetQueueUrl(context.TODO(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(evalUuid.String()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get queue URL: %w", err)
	}
	if queueUrl.QueueUrl == nil {
		return nil, fmt.Errorf("queue URL is nil")
	}
	return e.receive(*queueUrl.QueueUrl)
}

func (e *EvalSrvc) Receive() ([]Msg, error) {
	return e.receive(e.resSqsUrl)
}

func (e *EvalSrvc) receive(queueUrl string) ([]Msg, error) {
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
		msgs[i].EvalId, err = uuid.Parse(header.EvalUuid)
		if err != nil {
			return nil, fmt.Errorf("failed to parse eval_uuid: %w", err)
		}

		switch header.MsgType {
		case sqsgath.MsgTypeStartedEvaluation:
			startedEvaluation := sqsgath.StartedEvaluation{}
			err = json.Unmarshal([]byte(*msg.Body), &startedEvaluation)
			msgs[i].Data = StartedEvaluation{}
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

func (e *EvalSrvc) Ack(handle string) error {
	_, err := e.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(e.resSqsUrl),
		ReceiptHandle: aws.String(handle),
	})
	return err
}

type Event interface {
	Type() string
}

type Msg struct {
	EvalId uuid.UUID
	Handle string // receipt handle for acknowledgment / delete
	Data   Event  // data specific to the message / event type
}

type EvalSrvc struct {
	sqsClient *sqs.Client

	submSqsUrl string
	resSqsUrl  string

	extEvalKey string // api key for external evaluation requests
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

	return &EvalSrvc{
		sqsClient:  sqsClient,
		submSqsUrl: submQueueUrl,
		resSqsUrl:  responseSQSURL,
		extEvalKey: extEvalKey,
	}
}
