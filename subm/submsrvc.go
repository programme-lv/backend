package subm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	"github.com/programme-lv/backend/task"
	"github.com/programme-lv/backend/user"
	"goa.design/clue/log"
)

// submissions service example implementation.
// The example methods log the requests and return zero values.
type SubmissionsService struct {
	ddbClient     *dynamodb.Client
	submTableName string
	userSrvc      *user.UsersSrvc
	taskSrvc      taskgen.Service
	jwtKey        []byte
	sqsClient     *sqs.Client
	submQueueUrl  string

	createdSubmChan        chan *Submission
	updateSubmStateChan    chan *SubmissionStateUpdate
	updateTestgroupResChan chan *TestgroupResultUpdate

	updateListenerLock     sync.Mutex
	updateListeners        []chan *SubmissionListUpdate
	updateRemovedListeners []chan *SubmissionListUpdate

	evalUuidToSubmUuid map[string]string
}

type SubmissionStateUpdate struct {
	SubmUuid string
	EvalUuid string
	NewState string
}

type TestgroupResultUpdate struct {
	SubmUuid      string
	EvalUuid      string
	TestgroupId   int
	AcceptedTests int
	WrongTests    int
	UntestedTests int
}

// NewSubmissions returns the submissions service implementation.
func NewSubmissions(ctx context.Context) *SubmissionsService {
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

	srvc := &SubmissionsService{
		ddbClient:              dynamodbClient,
		submTableName:          submTableName,
		userSrvc:               user.NewUsers(ctx),
		taskSrvc:               task.NewTasks(ctx),
		jwtKey:                 []byte(jwtKey),
		sqsClient:              sqsClient,
		submQueueUrl:           submQueueUrl,
		createdSubmChan:        make(chan *Submission, 1000),
		updateSubmStateChan:    make(chan *SubmissionStateUpdate, 1000),
		updateTestgroupResChan: make(chan *TestgroupResultUpdate, 1000),
		updateListenerLock:     sync.Mutex{},
		updateListeners:        make([]chan *SubmissionListUpdate, 0, 100),
		updateRemovedListeners: make([]chan *SubmissionListUpdate, 0, 100),
		evalUuidToSubmUuid:     map[string]string{},
	}

	go srvc.StartProcessingSubmEvalResults(ctx)
	go srvc.StartStreamingSubmListUpdates(ctx)

	return srvc
}

func (s *SubmissionsService) StartProcessingSubmEvalResults(ctx context.Context) (err error) {
	submEvalResQueueUrl := "https://sqs.eu-central-1.amazonaws.com/975049886115/standard_subm_eval_results"
	throtleChan := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
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
	const maxSubmissionLengthKilobytes = 64 // 64 KB
	if len(subm.Value) > maxSubmissionLengthKilobytes*1000 {
		return newErrInvalidSubmissionDetailsContentTooLong(maxSubmissionLengthKilobytes)
	}
	return nil
}

func (subm *SubmissionContent) String() string {
	return subm.Value
}

func (s *SubmissionsService) StartStreamingSubmListUpdates(ctx context.Context) {
	sendUpdate := func(update *SubmissionListUpdate) {
		s.updateListenerLock.Lock()
		for _, listener := range s.updateListeners {
			if len(listener) == cap(listener) {
				<-listener
			}
			listener <- update
		}
		s.updateListenerLock.Unlock()
	}

	for {
		select {
		case created := <-s.createdSubmChan:
			// notify all listeners about the new submission
			update := &SubmissionListUpdate{
				SubmCreated: created,
			}
			sendUpdate(update)
		case stateUpdate := <-s.updateSubmStateChan:
			// notify all listeners about the state update
			update := &SubmissionListUpdate{
				StateUpdate: &SubmissionStateUpdate{
					SubmUuid: stateUpdate.SubmUuid,
					EvalUuid: stateUpdate.EvalUuid,
					NewState: stateUpdate.NewState,
				},
			}
			sendUpdate(update)
		case testgroupResUpdate := <-s.updateTestgroupResChan:
			// notify all listeners about the testgroup result update
			update := &SubmissionListUpdate{
				TestgroupResUpdate: &TestgroupScoreUpdate{
					SubmUUID:      testgroupResUpdate.SubmUuid,
					EvalUUID:      testgroupResUpdate.EvalUuid,
					TestGroupID:   testgroupResUpdate.TestgroupId,
					AcceptedTests: testgroupResUpdate.AcceptedTests,
					WrongTests:    testgroupResUpdate.WrongTests,
					UntestedTests: testgroupResUpdate.UntestedTests,
				},
			}

			sendUpdate(update)
		}
	}
}

// // StreamSubmissionUpdates implements submissions.Service.
// func (s *SubmissionsService) StreamSubmissionUpdates(ctx context.Context, p StreamSubmissionUpdatesServerStream) (err error) {
// 	// register myself as a listener to the submission updates
// 	myChan := make(chan *SubmissionListUpdate, 1000)
// 	s.updateListenerLock.Lock()
// 	s.updateListeners = append(s.updateListeners, myChan)
// 	s.updateListenerLock.Unlock()

// 	defer func() {
// 		// lock listener slice
// 		s.updateListenerLock.Lock()
// 		// remove myself from the listeners slice
// 		for i, listener := range s.updateListeners {
// 			if listener == myChan {
// 				s.updateListeners = append(s.updateListeners[:i], s.updateListeners[i+1:]...)
// 				break
// 			}
// 		}
// 		s.updateListenerLock.Unlock()
// 		close(myChan)
// 	}()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return p.Close()
// 		case update := <-myChan:
// 			err = p.Send(update)
// 			if err != nil {
// 				return p.Close()
// 			}
// 		}
// 	}
// }
