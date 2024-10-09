package submsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"
)

// submissions service example implementation.
// The example methods log the requests and return zero values.
type SubmissionSrvc struct {
	userSrvc *user.UserService
	taskSrvc *tasksrvc.TaskService

	sqsClient  *sqs.Client
	submSqsUrl string
	resSqsUrl  string

	// real-time updates
	createNewSubmChan        chan *BriefSubmission
	updateSubmEvalStageChan  chan *SubmEvalStageUpdate
	updateTestGroupScoreChan chan *TestGroupScoreUpdate
	updateTestScoreChan      chan *TestScoreUpdate
	listenerLock             sync.Mutex
	listeners                []chan *SubmissionListUpdate

	evalUuidToSubmUuid sync.Map
}

// NewSubmissions returns the submissions service implementation.
func NewSubmissions(taskSrvc *tasksrvc.TaskService) *SubmissionSrvc {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 10)
		}),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}

	submTableName := os.Getenv("DDB_SUBM_TABLE_NAME")
	if submTableName == "" {
		slog.Error("DDB_SUBM_TABLE_NAME is not set")
		os.Exit(1)
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

	srvc := &SubmissionSrvc{
		userSrvc:                 user.NewUsers(),
		taskSrvc:                 taskSrvc,
		sqsClient:                sqsClient,
		submSqsUrl:               submQueueUrl,
		createNewSubmChan:        make(chan *BriefSubmission, 1000),
		updateSubmEvalStageChan:  make(chan *SubmEvalStageUpdate, 1000),
		updateTestGroupScoreChan: make(chan *TestGroupScoreUpdate, 1000),
		updateTestScoreChan:      make(chan *TestScoreUpdate, 1000),
		listenerLock:             sync.Mutex{},
		listeners:                make([]chan *SubmissionListUpdate, 0, 100),
		evalUuidToSubmUuid:       sync.Map{},
		resSqsUrl:                responseSQSURL,
	}

	go srvc.StartProcessingSubmEvalResults(context.TODO())
	go srvc.StartStreamingSubmListUpdates(context.TODO())

	return srvc
}

func (s *SubmissionSrvc) StartProcessingSubmEvalResults(ctx context.Context) (err error) {
	submEvalResQueueUrl := s.resSqsUrl
	throtleChan := make(chan struct{}, 20)
	for i := 0; i < 20; i++ {
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

			slog.Info("received eval message", "body", (*message.Body)[:min(200, len(*message.Body))])

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
			s.processEvalResult(qMsg.EvalUuid, msgType.MsgType, qMsg.Data)
			throtleChan <- struct{}{}
		}
	}
}
