package subm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	submgen "github.com/programme-lv/backend/gen/submissions"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	usergen "github.com/programme-lv/backend/gen/users"
	"github.com/programme-lv/backend/task"
	"github.com/programme-lv/backend/user"
	"goa.design/clue/log"
)

// submissions service example implementation.
// The example methods log the requests and return zero values.
type submissionssrvc struct {
	ddbClient     *dynamodb.Client
	submTableName string
	userSrvc      usergen.Service
	taskSrvc      taskgen.Service
	jwtKey        []byte
	sqsClient     *sqs.Client
	submQueueUrl  string
}

// NewSubmissions returns the submissions service implementation.
func NewSubmissions(ctx context.Context) submgen.Service {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatalf(ctx,
			errors.New("JWT_KEY is not set"),
			"cant read JWT_KEY from env in new user service constructor")
	}

	submTableName := os.Getenv("DDB_SUBM_TABLE_NAME")
	if submTableName == "" {
		log.Fatalf(ctx,
			errors.New("DDB_SUBM_TABLE_NAME is not set"),
			"cant read DDB_SUBM_TABLE_NAME from env in new user service constructor")
	}

	sqsClient := sqs.NewFromConfig(cfg)

	submQueueUrl := os.Getenv("SUBM_SQS_QUEUE_URL")
	if submQueueUrl == "" {
		panic("SUBM_SQS_QUEUE_URL not set in .env file")
	}

	srvc := &submissionssrvc{
		submTableName: submTableName,
		ddbClient:     dynamodbClient,
		userSrvc:      user.NewUsers(ctx),
		taskSrvc:      task.NewTasks(ctx),
		jwtKey:        []byte(jwtKey),
		sqsClient:     sqsClient,
		submQueueUrl:  submQueueUrl,
	}

	go srvc.StartProcessingSubmEvalResults(ctx)

	return srvc
}

func (s *submissionssrvc) StartProcessingSubmEvalResults(ctx context.Context) (err error) {
	submEvalResQueueUrl := "https://sqs.eu-central-1.amazonaws.com/975049886115/standard_subm_eval_results"
	throtleChan := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		throtleChan <- struct{}{}
	}
	for {
		output, err := s.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(submEvalResQueueUrl),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     5,
		})
		if err != nil {
			fmt.Printf("failed to receive messages, %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, message := range output.Messages {
			_, err = s.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(submEvalResQueueUrl),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				fmt.Printf("failed to delete message, %v\n", err)
			}

			log.Printf(ctx, "received eval message: %s\n", (*message.Body)[:min(200, len(*message.Body))])

			var qMsg struct {
				EvalUuid string           `json:"eval_uuid"`
				Data     *json.RawMessage `json:"data"`
			}
			err := json.Unmarshal([]byte(*message.Body), &qMsg)
			if err != nil {
				fmt.Printf("failed to unmarshal message: %v\n", err)
				continue
			}

			msgType := struct {
				MsgType string `json:"msg_type"`
			}{}
			err = json.Unmarshal(*qMsg.Data, &msgType)
			if err != nil {
				fmt.Printf("failed to unmarshal message: %v\n", err)
				continue
			}

			// TODO throttle for each eval uuid individually
			<-throtleChan
			go s.processEvalResult(qMsg.EvalUuid, msgType.MsgType, qMsg.Data)
			throtleChan <- struct{}{}
		}
	}
}

type Validatable interface {
	IsValid() error
}

type SubmissionContent struct {
	Value string
}

func (subm *SubmissionContent) IsValid() error {
	const maxSubmissionLength = 64000 // 64 KB
	if len(subm.Value) > maxSubmissionLength {
		return submgen.InvalidSubmissionDetails(
			"maksimālais iesūtījuma garums ir 64 KB",
		)
	}
	return nil
}

func (subm *SubmissionContent) String() string {
	return subm.Value
}
