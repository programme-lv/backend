package execsrvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/tester/sqsgath"
)

// Starts receiving msgs until ctx is cancelled and passes them to handler function
func StartReceivingResultsFromSqs(ctx context.Context,
	sqsUrl string, client *sqs.Client,
	handleFunc func(msg SqsResponseMsg) error,
	logger *slog.Logger,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			output, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(sqsUrl),
				MaxNumberOfMessages: 10,
				WaitTimeSeconds:     1,
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				logger.Error("failed to receive messages", "error", err)
				continue
			}

			msgs := make([]SqsResponseMsg, len(output.Messages))
			for i, msg := range output.Messages {
				if msg.Body == nil {
					return fmt.Errorf("message body is nil")
				}

				var header sqsgath.Header
				err = json.Unmarshal([]byte(*msg.Body), &header)
				if err != nil {
					logger.Error("failed to unmarshal message", "error", err)
					continue
				}

				if msg.ReceiptHandle == nil {
					return fmt.Errorf("receipt handle is nil")
				}
				msgs[i].Handle = *msg.ReceiptHandle
				msgs[i].QueueUrl = sqsUrl
				msgs[i].ExecId, err = uuid.Parse(header.EvalUuid)
				if err != nil {
					return fmt.Errorf("failed to parse eval_uuid: %w", err)
				}

				switch header.MsgType {
				case sqsgath.MsgTypeStartedEvaluation:
					startedEvaluation := sqsgath.StartedEvaluation{}
					err = json.Unmarshal([]byte(*msg.Body), &startedEvaluation)
					startedAt, err := time.Parse(time.RFC3339, startedEvaluation.StartedTime)
					if err != nil {
						return fmt.Errorf("failed to parse started_at: %w", err)
					}
					msgs[i].Data = ReceivedSubmission{
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
						TestId: int(reachedTest.TestId),
						In:     reachedTest.Input,
						Ans:    reachedTest.Answer,
					}
				case sqsgath.MsgTypeIgnoredTest:
					ignoredTest := sqsgath.IgnoredTest{}
					err = json.Unmarshal([]byte(*msg.Body), &ignoredTest)
					msgs[i].Data = IgnoredTest{
						TestId: int(ignoredTest.TestId),
					}
				case sqsgath.MsgTypeFinishedTest:
					finishTest := sqsgath.FinishedTest{}
					err = json.Unmarshal([]byte(*msg.Body), &finishTest)
					msgs[i].Data = FinishedTest{
						TestID:  int(finishTest.TestId),
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
					if finishEval.CompileError {
						msgs[i].Data = CompilationError{
							ErrorMsg: finishEval.ErrorMessage,
						}
					} else if finishEval.InternalError {
						msgs[i].Data = InternalServerError{
							ErrorMsg: finishEval.ErrorMessage,
						}
					} else {
						continue
					}
				}

				if err != nil {
					format := "failed to unmarshal %s message: %v"
					errMsg := fmt.Errorf(format, header.MsgType, err)
					logger.Error("message unmarshal failed",
						"msgType", header.MsgType,
						"error", err)
					return errMsg
				}

				go func(msg SqsResponseMsg) {
					err = handleFunc(msg)
					if err != nil {
						logger.Error("failed to process tester result", "error", err)
					}
					_, err := client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
						QueueUrl:      aws.String(sqsUrl),
						ReceiptHandle: aws.String(msg.Handle),
					})
					if err != nil {
						logger.Error("failed to ack message", "error", err)
					}
				}(msgs[i])
			}
		}
	}
}

type SqsResponseMsg struct {
	ExecId   uuid.UUID
	QueueUrl string // url of queue it was received from
	Handle   string // receipt handle for acknowledgment / delete
	Data     Event  // data specific to the message / event type
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
			CtxSwV:   rd.CtxSwV,
			CtxSwF:   rd.CtxSwF,
			Signal:   rd.ExitSignal,
		}
	}
	return nil
}
