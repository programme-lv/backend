package submsrvc

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
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"
	"github.com/programme-lv/tester/sqsgath"

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
	submCreated       chan *Submission
	evalStageUpd      chan *SubmEvalStageUpdate
	testGroupScoreUpd chan *TestGroupScoringUpdate
	testSetScoreUpd   chan *TestSetScoringUpdate

	listenerLock sync.Mutex
	listeners    []chan *SubmissionListUpdate

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
		userSrvc:           user.NewUsers(),
		taskSrvc:           taskSrvc,
		postgres:           db,
		sqsClient:          sqsClient,
		submSqsUrl:         submQueueUrl,
		submCreated:        make(chan *Submission, 1000),
		evalStageUpd:       make(chan *SubmEvalStageUpdate, 1000),
		testGroupScoreUpd:  make(chan *TestGroupScoringUpdate, 1000),
		testSetScoreUpd:    make(chan *TestSetScoringUpdate, 1000),
		listenerLock:       sync.Mutex{},
		listeners:          make([]chan *SubmissionListUpdate, 0, 100),
		evalUuidToSubmUuid: sync.Map{},
		resSqsUrl:          responseSQSURL,
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
			log.Printf("failed to receive messages, %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, message := range output.Messages {
			_, err = s.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(submEvalResQueueUrl),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				log.Printf("failed to delete message, %v\n", err)
			}

			var header sqsgath.Header
			err = json.Unmarshal([]byte(*message.Body), &header)
			if err != nil {
				log.Printf("failed to unmarshal message: %v\n", err)
				continue
			}

			switch header.MsgType {
			case sqsgath.MsgTypeStartedEvaluation:
				startedEvaluation := sqsgath.StartedEvaluation{}
				err = json.Unmarshal([]byte(*message.Body), &startedEvaluation)
				if err != nil {
					log.Printf("failed to unmarshal StartedEvaluation message: %v\n", err)
				} else {
					s.handleStartedEvaluation(&startedEvaluation)
				}
			case sqsgath.MsgTypeStartedCompilation:
				startedCompilation := sqsgath.StartedCompilation{}
				err = json.Unmarshal([]byte(*message.Body), &startedCompilation)
				if err != nil {
					log.Printf("failed to unmarshal StartedCompilation message: %v\n", err)
				} else {
					s.handleStartedCompilation(&startedCompilation)
				}
			case sqsgath.MsgTypeFinishedCompilation:
				finishedCompilation := sqsgath.FinishedCompilation{}
				err = json.Unmarshal([]byte(*message.Body), &finishedCompilation)
				if err != nil {
					log.Printf("failed to unmarshal FinishedCompilation message: %v\n", err)
				} else {
					s.handleFinishedCompilation(&finishedCompilation)
				}
			case sqsgath.MsgTypeStartedTesting:
				startedTesting := sqsgath.StartedTesting{}
				err = json.Unmarshal([]byte(*message.Body), &startedTesting)
				if err != nil {
					log.Printf("failed to unmarshal StartedTesting message: %v\n", err)
				} else {
					s.handleStartedTesting(&startedTesting)
				}
			case sqsgath.MsgTypeReachedTest:
				reachedTest := sqsgath.ReachedTest{}
				err = json.Unmarshal([]byte(*message.Body), &reachedTest)
				if err != nil {
					log.Printf("failed to unmarshal ReachedTest message: %v\n", err)
				} else {
					s.handleReachedTest(&reachedTest)
				}
			case sqsgath.MsgTypeIgnoredTest:
				ignoredTest := sqsgath.IgnoredTest{}
				err = json.Unmarshal([]byte(*message.Body), &ignoredTest)
				if err != nil {
					log.Printf("failed to unmarshal IgnoredTest message: %v\n", err)
				} else {
					s.handleIgnoredTest(&ignoredTest)
				}
			case sqsgath.MsgTypeFinishedTest:
				finishedTest := sqsgath.FinishedTest{}
				err = json.Unmarshal([]byte(*message.Body), &finishedTest)
				if err != nil {
					log.Printf("failed to unmarshal FinishedTest message: %v\n", err)
				} else {
					s.handleFinishedTest(&finishedTest)
				}
			case sqsgath.MsgTypeFinishedTesting:
				finishedTesting := sqsgath.FinishedTesting{}
				err = json.Unmarshal([]byte(*message.Body), &finishedTesting)
				if err != nil {
					log.Printf("failed to unmarshal FinishedTesting message: %v\n", err)
				} else {
					s.handleFinishedTesting(&finishedTesting)
				}
			case sqsgath.MsgTypeFinishedEvaluation:
				finishedEvaluation := sqsgath.FinishedEvaluation{}
				err = json.Unmarshal([]byte(*message.Body), &finishedEvaluation)
				if err != nil {
					log.Printf("failed to unmarshal FinishedEvaluation message: %v\n", err)
				} else {
					s.handleFinishedEvaluation(&finishedEvaluation)
				}
			}

			<-throtleChan
			throtleChan <- struct{}{}
		}
	}
}
