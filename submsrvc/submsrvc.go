package submsrvc

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"

	_ "github.com/lib/pq"
)

type SubmissionSrvc struct {
	userSrvc *user.UserService
	taskSrvc *tasksrvc.TaskService

	postgres *sqlx.DB

	evalSrvc *evalsrvc.EvalSrvc

	// real-time updates
	submCreated    chan *Submission
	evalStageUpd   chan *EvalStageUpd
	tGroupScoreUpd chan *TGroupScoreUpd
	tSetScoreUpd   chan *TSetScoreUpd

	listenerLock sync.Mutex
	listeners    []chan *SubmissionListUpdate

	evalUuidToSubmUuid sync.Map
}

func NewSubmissions(taskSrvc *tasksrvc.TaskService, evalSrvc *evalsrvc.EvalSrvc) *SubmissionSrvc {
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
		evalSrvc:           evalSrvc,
		submCreated:        make(chan *Submission, 1000),
		evalStageUpd:       make(chan *EvalStageUpd, 1000),
		tGroupScoreUpd:     make(chan *TGroupScoreUpd, 1000),
		tSetScoreUpd:       make(chan *TSetScoreUpd, 1000),
		listenerLock:       sync.Mutex{},
		listeners:          make([]chan *SubmissionListUpdate, 0, 100),
		evalUuidToSubmUuid: sync.Map{},
	}

	go srvc.StartProcessingSubmEvalResults(context.TODO())
	go srvc.StartStreamingSubmListUpdates(context.TODO())

	return srvc
}

func getPostgresConnStr() string {
	user := os.Getenv("POSTGRES_USER")
	pw := os.Getenv("POSTGRES_PASSWORD")
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	ssl := os.Getenv("POSTGRES_SSLMODE")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pw, db, ssl)
}
