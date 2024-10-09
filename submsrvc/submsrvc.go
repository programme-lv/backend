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
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"

	_ "github.com/lib/pq"
)

type SubmissionSrvc struct {
	userSrvc *user.UserService
	taskSrvc *tasksrvc.TaskService

	postgres *sqlx.DB

	sqsClient  *sqs.Client
	submSqsUrl string
	resSqsUrl  string

	// real-time updates
	createNewSubmChan        chan *Submission
	updateSubmEvalStageChan  chan *SubmEvalStageUpdate
	updateTestGroupScoreChan chan *TestGroupScoringUpdate
	updateTestScoreChan      chan *TestSetScoringUpdate
	listenerLock             sync.Mutex
	listeners                []chan *SubmissionListUpdate

	evalUuidToSubmUuid sync.Map
}

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

	sqsClient := sqs.NewFromConfig(cfg)

	submQueueUrl := os.Getenv("SUBM_SQS_QUEUE_URL")
	if submQueueUrl == "" {
		panic("SUBM_SQS_QUEUE_URL not set in .env file")
	}

	responseSQSURL := os.Getenv("RESPONSE_SQS_URL")
	if responseSQSURL == "" {
		panic("RESPONSE_SQS_URL not set in .env file")
	}

	db, err := sqlx.Connect("postgres", os.Getenv("POSTGRES_CONN_STR"))
	if err != nil {
		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
	}

	srvc := &SubmissionSrvc{
		userSrvc:                 user.NewUsers(),
		taskSrvc:                 taskSrvc,
		postgres:                 db,
		sqsClient:                sqsClient,
		submSqsUrl:               submQueueUrl,
		createNewSubmChan:        make(chan *Submission, 1000),
		updateSubmEvalStageChan:  make(chan *SubmEvalStageUpdate, 1000),
		updateTestGroupScoreChan: make(chan *TestGroupScoringUpdate, 1000),
		updateTestScoreChan:      make(chan *TestSetScoringUpdate, 1000),
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
