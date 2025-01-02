package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/s3bucket"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"

	_ "github.com/lib/pq"
)

type submRepo interface {
	Store(ctx context.Context, subm Submission) error
	Get(ctx context.Context, uuid uuid.UUID) (*Submission, error)
	List(ctx context.Context) ([]Submission, error)
}

type SubmissionSrvc struct {
	logger *slog.Logger

	tests *s3bucket.S3Bucket
	repo  submRepo
	inMem map[uuid.UUID]Submission // in-progress submissions

	userSrvc *usersrvc.UserService
	taskSrvc *tasksrvc.TaskService
	evalSrvc *evalsrvc.EvalSrvc

	// real-time updates
	submUuidEvalUpdSubs []struct {
		submUuid uuid.UUID
		ch       chan *EvalUpdate
	}
	submListEvalUpdSubs []chan *EvalUpdate
	submCreatedSubs     []chan Submission
	listenerLock        sync.Mutex
	// submCreated        chan *Submission
	// evalUpdate         chan *Evaluation
	// listenerLock       sync.Mutex
	// evalUpdSubscribers []chan *EvalUpdate
}

func NewSubmSrvc(taskSrvc *tasksrvc.TaskService, evalSrvc *evalsrvc.EvalSrvc) (*SubmissionSrvc, error) {
	testBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		return nil, fmt.Errorf("failed to create test bucket: %w", err)
	}

	srvc := &SubmissionSrvc{
		logger:   slog.Default().With("module", "subm"),
		tests:    testBucket,
		userSrvc: usersrvc.NewUserService(),
		taskSrvc: taskSrvc,
		repo:     newInMemRepo(),
		evalSrvc: evalSrvc,
		inMem:    make(map[uuid.UUID]Submission),

		// submCreated:        make(chan *Submission, 1000),
		// evalUpdate:         make(chan *Evaluation, 1000),
		// listenerLock:       sync.Mutex{},
		// evalUpdSubscribers: make([]chan *SubmListUpdate, 0, 100),
	}

	return srvc, nil
}

func (s *SubmissionSrvc) GetSubm(ctx context.Context, uuid uuid.UUID) (*Submission, error) {
	if subm, ok := s.inMem[uuid]; ok {
		return &subm, nil
	}
	subm, err := s.repo.Get(ctx, uuid)
	if err != nil {
		return nil, err
	}
	s.inMem[uuid] = *subm
	return subm, nil
}

func (s *SubmissionSrvc) ListSubms(ctx context.Context) ([]Submission, error) {
	repoSubms, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Create map of submissions, preferring in-memory ones
	submMap := make(map[uuid.UUID]Submission)
	for _, subm := range repoSubms {
		submMap[subm.UUID] = subm
	}
	for _, subm := range s.inMem {
		submMap[subm.UUID] = subm
	}

	subms := make([]Submission, 0, len(submMap))
	for _, subm := range submMap {
		subms = append(subms, subm)
	}
	sort.Slice(subms, func(i, j int) bool {
		return subms[i].CreatedAt.After(subms[j].CreatedAt)
	})
	return subms, nil
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
