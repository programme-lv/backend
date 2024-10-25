package submsrvc

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

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
	submCreated       chan *Submission
	evalStageUpd      chan *SubmEvalStageUpdate
	testGroupScoreUpd chan *TestGroupScoringUpdate
	testSetScoreUpd   chan *TestSetScoringUpdate

	listenerLock sync.Mutex
	listeners    []chan *SubmissionListUpdate

	evalUuidToSubmUuid sync.Map
}

func getPostgresConnStr() string {
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")
	postgresDbName := os.Getenv("POSTGRES_DB")
	postgresSslmode := os.Getenv("POSTGRES_SSLMODE")

	// encodedPassword := url.PathEscape(postgresPassword)

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		postgresHost, postgresPort, postgresUser, postgresPassword, postgresDbName, postgresSslmode)
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

	postgresConnStr := getPostgresConnStr()
	log.Printf("postgresConnStr: %s\n", postgresConnStr)
	db, err := sqlx.Connect("postgres", postgresConnStr)
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
