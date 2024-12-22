package evalsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/tester/sqsgath"
)

// starts receiving msgs indefinitely and passes them to processor
func receiveResultsFromSqs(ctx context.Context,
	sqsUrl string, client *sqs.Client,
	handleFunc func(msg Msg) error,
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
				log.Printf("failed to receive messages, %v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}

			msgs := make([]Msg, len(output.Messages))
			for i, msg := range output.Messages {
				if msg.Body == nil {
					return fmt.Errorf("message body is nil")
				}

				var header sqsgath.Header
				err = json.Unmarshal([]byte(*msg.Body), &header)
				if err != nil {
					log.Printf("failed to unmarshal message: %v\n", err)
					continue
				}

				if msg.ReceiptHandle == nil {
					return fmt.Errorf("receipt handle is nil")
				}
				msgs[i].Handle = *msg.ReceiptHandle
				msgs[i].QueueUrl = sqsUrl
				msgs[i].EvalId, err = uuid.Parse(header.EvalUuid)
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
					format := "failed to unmarshal %s message: %v"
					errMsg := fmt.Errorf(format, header.MsgType, err)
					log.Print(errMsg)
					return errMsg
				}

				go func(msg Msg) {
					err = handleFunc(msg)
					if err != nil {
						log.Printf("failed to process tester result: %v", err)
					} else { // there were no errors
						_, err := client.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
							QueueUrl:      aws.String(sqsUrl),
							ReceiptHandle: aws.String(msg.Handle),
						})
						if err != nil {
							log.Printf("failed to ack message: %v", err)
						}
					}
				}(msgs[i])
			}
		}
	}
}
