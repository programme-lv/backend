package submsrvc

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/s3bucket"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"

	_ "github.com/lib/pq"
)

type submRepo interface {
	Store(ctx context.Context, subm SubmissionEntity) error
	Get(ctx context.Context, uuid uuid.UUID) (SubmissionEntity, error)
	List(ctx context.Context, limit int, offset int) ([]SubmissionEntity, error)
	AssignEval(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error
}

type evalRepo interface {
	Store(ctx context.Context, eval Evaluation) error
	Get(ctx context.Context, uuid uuid.UUID) (Evaluation, error)
}

type SubmissionSrvc struct {
	logger *slog.Logger

	tests    *s3bucket.S3Bucket
	submRepo submRepo // persistent subm storage
	evalRepo evalRepo
	inMem    map[uuid.UUID]Evaluation // subm id to corresponding in-progress evaluation

	userSrvc *usersrvc.UserService
	taskSrvc *tasksrvc.TaskService
	evalSrvc *execsrvc.EvalSrvc

	// real-time updates
	submUuidEvalUpdSubs []struct {
		submUuid uuid.UUID
		ch       chan *EvalUpdate
	}
	submListEvalUpdSubs []chan *EvalUpdate
	submCreatedSubs     []chan Submission
	listenerLock        sync.Mutex
}

func NewSubmSrvc(taskSrvc *tasksrvc.TaskService, evalSrvc *execsrvc.EvalSrvc) (*SubmissionSrvc, error) {
	testBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		return nil, fmt.Errorf("failed to create test bucket: %w", err)
	}

	pool, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		return nil, fmt.Errorf("failed to create pg pool: %w", err)
	}

	srvc := &SubmissionSrvc{
		logger:   slog.Default().With("module", "subm"),
		tests:    testBucket,
		userSrvc: usersrvc.NewUserService(),
		taskSrvc: taskSrvc,
		submRepo: NewPgSubmRepo(pool),
		evalRepo: NewPgEvalRepo(pool),
		evalSrvc: evalSrvc,
		inMem:    make(map[uuid.UUID]Evaluation),
	}

	return srvc, nil
}

func (s *SubmissionSrvc) GetSubm(ctx context.Context, submUuid uuid.UUID) (*Submission, error) {
	entity, err := s.submRepo.Get(ctx, submUuid)
	if err != nil {
		return nil, err
	}
	return s.constructSubm(ctx, entity)
}

func (s *SubmissionSrvc) constructSubm(ctx context.Context, subm SubmissionEntity) (*Submission, error) {
	var eval *Evaluation
	if evalVal, ok := s.inMem[subm.UUID]; ok {
		eval = &evalVal
	} else if subm.CurrEvalID != uuid.Nil {
		evalVal, err := s.evalRepo.Get(ctx, subm.CurrEvalID)
		if err != nil {
			return nil, err
		}
		eval = &evalVal
	}

	user, err := s.userSrvc.GetUserByUUID(ctx, subm.AuthorUUID)
	if err != nil {
		return nil, err
	}

	task, err := s.taskSrvc.GetTask(ctx, subm.TaskShortID)
	if err != nil {
		return nil, err
	}

	lang, err := planglist.GetProgrammingLanguageById(subm.LangShortID)
	if err != nil {
		return nil, err
	}

	return &Submission{
		UUID:      subm.UUID,
		Content:   subm.Content,
		Author:    Author{UUID: user.UUID, Username: user.Username},
		Task:      TaskRef{ShortID: task.ShortId, FullName: task.FullName},
		Lang:      PrLang{ShortID: lang.ID, Display: lang.FullName, MonacoID: lang.MonacoId},
		CurrEval:  eval,
		CreatedAt: subm.CreatedAt,
	}, nil
}

func (s *SubmissionSrvc) ListSubms(ctx context.Context, limit int, offset int) ([]Submission, error) {
	repoSubms, err := s.submRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Create map of submissions, preferring in-memory ones
	submMap := make(map[uuid.UUID]Submission)
	for _, subm := range repoSubms {
		subm2, err := s.constructSubm(ctx, subm)
		if err != nil {
			return nil, err
		}
		submMap[subm.UUID] = *subm2
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
