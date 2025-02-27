package execsrvc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
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
			// before returning, wait for everything to finish
			return ctx.Err()
		default:
			wg2 := sync.WaitGroup{}
			parallel := 3
			wg2.Add(parallel)
			for i := 0; i < parallel; i++ {
				go func() {
					defer wg2.Done()

					output, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
						QueueUrl:            aws.String(sqsUrl),
						MaxNumberOfMessages: 10,
						WaitTimeSeconds:     10,
					})
					if err != nil {
						if errors.Is(err, context.Canceled) {
							return
						}
						logger.Error("failed to receive messages", "error", err)
						return
					}

					msgs := make([]SqsResponseMsg, len(output.Messages))
					wg := sync.WaitGroup{}
					for i, msg := range output.Messages {
						if msg.Body == nil {
							logger.Error("message body is nil")
							continue
						}

						var header sqsgath.Header
						err = json.Unmarshal([]byte(*msg.Body), &header)
						if err != nil {
							logger.Error("failed to unmarshal message", "error", err)
							continue
						}

						if msg.ReceiptHandle == nil {
							logger.Error("receipt handle is nil")
							continue
						}
						msgs[i].Handle = *msg.ReceiptHandle
						msgs[i].QueueUrl = sqsUrl
						msgs[i].ExecId, err = uuid.Parse(header.EvalUuid)
						if err != nil {
							logger.Error("failed to parse eval_uuid", "error", err)
							continue
						}

						switch header.MsgType {
						case sqsgath.MsgTypeStartedEvaluation:
							startedEvaluation := sqsgath.StartedEvaluation{}
							err = json.Unmarshal([]byte(*msg.Body), &startedEvaluation)
							startedAt, err := time.Parse(time.RFC3339, startedEvaluation.StartedTime)
							if err != nil {
								logger.Error("failed to parse started_at", "error", err)
								continue
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
							logger.Error("message unmarshal failed",
								"msgType", header.MsgType,
								"error", err)
							continue
						}

						wg.Add(1)
						go func(msg SqsResponseMsg) {
							err := handleFunc(msg)
							if err != nil {
								logger.Debug("failed to process tester result", "error", err)
							}
							ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
							defer cancel()
							_, err = client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
								QueueUrl:      aws.String(sqsUrl),
								ReceiptHandle: aws.String(msg.Handle),
							})
							if err != nil {
								logger.Error("failed to ack message", "error", err)
							}
							wg.Done()
						}(msgs[i])
					}
					wg.Wait()
				}()
			}
			wg2.Wait()
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
