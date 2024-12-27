package submsrvc

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"

	_ "github.com/lib/pq"
)

type submRepo interface {
	Store(ctx context.Context, subm Submission) error
	Get(ctx context.Context, uuid uuid.UUID) (*Submission, error)
}

type SubmissionSrvc struct {
	repo submRepo

	userSrvc *usersrvc.UserService
	taskSrvc *tasksrvc.TaskService
	evalSrvc *evalsrvc.EvalSrvc

	// real-time updates
	submEvalUpdSubs []struct {
		submUuid uuid.UUID
		ch       chan *EvalUpdate
	}
	submCreatedSubs []chan *Submission
	listenerLock    sync.Mutex
	// submCreated        chan *Submission
	// evalUpdate         chan *Evaluation
	// listenerLock       sync.Mutex
	// evalUpdSubscribers []chan *EvalUpdate
}

func NewSubmSrvc(taskSrvc *tasksrvc.TaskService, evalSrvc *evalsrvc.EvalSrvc) *SubmissionSrvc {
	srvc := &SubmissionSrvc{
		userSrvc: usersrvc.NewUsers(),
		taskSrvc: taskSrvc,
		repo:     newInMemRepo(),
		evalSrvc: evalSrvc,
		// submCreated:        make(chan *Submission, 1000),
		// evalUpdate:         make(chan *Evaluation, 1000),
		// listenerLock:       sync.Mutex{},
		// evalUpdSubscribers: make([]chan *SubmListUpdate, 0, 100),
	}

	return srvc
}

// func getPgConn() *sqlx.DB {
// 	postgresConnStr := getPostgresConnStr()
// 	db, err := sqlx.Connect("postgres", postgresConnStr)
// 	if err != nil {
// 		panic(fmt.Sprintf("failed to connect to postgres: %v", err))
// 	}
// 	return db
// }

// func getPostgresConnStr() string {
// 	user := os.Getenv("POSTGRES_USER")
// 	pw := os.Getenv("POSTGRES_PASSWORD")
// 	host := os.Getenv("POSTGRES_HOST")
// 	port := os.Getenv("POSTGRES_PORT")
// 	db := os.Getenv("POSTGRES_DB")
// 	ssl := os.Getenv("POSTGRES_SSLMODE")

// 	return fmt.Sprintf(
// 		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
// 		host, port, user, pw, db, ssl)
// }
