package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/modules/exec"
	exechttp "github.com/programme-lv/backend/modules/exec/http"
	planghttp "github.com/programme-lv/backend/modules/plang/http"
	"github.com/programme-lv/backend/modules/subm/domain"
	submhttp "github.com/programme-lv/backend/modules/subm/http"
	submpgrepo "github.com/programme-lv/backend/modules/subm/pgrepo"
	"github.com/programme-lv/backend/modules/subm/srvc"
	taskhttp "github.com/programme-lv/backend/modules/task/http"
	"github.com/programme-lv/backend/modules/task/repo"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	usersrvc "github.com/programme-lv/backend/modules/user"
	userhttp "github.com/programme-lv/backend/modules/user/http"
	"github.com/schollz/progressbar/v3"
)

const (
	address = ":8080"
)

func main() {
	setupLogger()

	// Add flag for SQS listening
	listenSQS := flag.Bool("listen-sqs", true, "Whether to listen to result SQS queue")
	flag.Parse()

	jwtKey := conf.MustGetJwtKeyFromEnv()
	cookieDomain := conf.MustGetCookieDomainFromEnv()

	pgPool := conf.MustGetPgxPoolFromEnv()
	conf.MustRunPostgresMigrationsFromEnv()

	storageRoot := conf.MustGetFileStorageRootFromEnv()
	apiPublicBaseURL := conf.MustGetPublicAPIBaseURLFromEnv()
	testfileSigningKey := conf.MustGetTestfileDownloadSigningKeyFromEnv()
	publicStore := conf.MustGetFileStore(storageRoot, "public")
	testfileStore := conf.MustGetFileStore(storageRoot, "testfiles")
	execStore := conf.MustGetFileStore(storageRoot, "exec")

	execCtx := context.Background()
	execCtx = ctxlog.WithLogger(execCtx, slog.Default().With("module", "exec"))
	execRepo := exec.NewFileExecRepo(execCtx, execStore)
	natsConn := conf.MustGetNatsConnFromEnv(execCtx)
	execSrvc := exec.NewExecSrvc(execCtx, execRepo, natsConn)
	if *listenSQS {
		err := execSrvc.StartPollingResultQueue(execCtx)
		if err != nil {
			slog.Error("failed to listen to result SQS", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("listening to execution result SQS disabled")
	}

	// Initialize user service
	userSrvc := usersrvc.NewUserService(pgPool)

	// Initialize task service
	taskRepo := repo.NewTaskPgRepo(pgPool)
	taskSrvc := tasksrvc.NewTaskSrvc(
		taskRepo,
		publicStore,
		testfileStore,
		tasksrvc.WithPublicAPIBaseURL(apiPublicBaseURL),
		tasksrvc.WithTestfileDownloadSigningKey(testfileSigningKey),
	)

	// Initialize HTTP handlers
	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := taskhttp.NewTaskHttpHandler(
		taskSrvc,
		taskhttp.WithFileStores(publicStore, testfileStore, testfileSigningKey),
	)
	userHttpHandler := userhttp.NewUserHttpHandler(userSrvc, jwtKey, userhttp.WithCookieDomain(cookieDomain))
	execHttpHandler := exechttp.NewExecHttpHandler(execSrvc)
	plangHttpHandler := planghttp.NewPlangHttpHandler()

	// Start HTTP server
	httpServer := newHTTPServer(
		submHttpHandler,
		taskHttpHandler,
		userHttpHandler,
		execHttpHandler,
		plangHttpHandler,
		jwtKey,
	)

	slog.Info("starting server", "address", address)
	err := httpServer.start(address)
	slog.Info("server stopped", "error", err)
}

func setupLogger() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stdout, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	))
}

func newSubmHttpHandler(userSrvc usersrvc.UserService, taskSrvc tasksrvc.TaskService, execSrvc exec.CodeExecutionService) *submhttp.SubmHttpHandler {
	pgPool, err := conf.GetPgxPoolFromEnv()
	if err != nil {
		slog.Error("failed to create pg pool", "error", err)
		os.Exit(1)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pgPool)
	submSrvc := srvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)

	// Check if migration is needed and run it
	runScoreMigrationIfNeeded(pgPool, submSrvc, evalPgRepo)

	return submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)
}

// runScoreMigrationIfNeeded checks if there are evaluations without score info
// and recalculates/stores them. This is a one-time migration.
func runScoreMigrationIfNeeded(pgPool *pgxpool.Pool, submSrvc srvc.SubmissionService, evalPgRepo interface {
	StoreEval(ctx context.Context, eval domain.Eval) error
}) {
	ctx := context.Background()

	// Check if there are evaluations missing score info
	var count int
	err := pgPool.QueryRow(ctx, `
		SELECT COUNT(*) FROM submissions s 
		LEFT OUTER JOIN evaluations e ON s.curr_eval_uuid = e.uuid 
		WHERE e.received_score IS NULL
	`).Scan(&count)
	if err != nil {
		slog.Error("failed to check for missing scores", "error", err)
		return
	}

	if count == 0 {
		slog.Info("no evaluations missing score info, skipping migration")
		return
	}

	slog.Warn("found evaluations missing score info, running migration", "count", count)

	subms, err := submSrvc.ListSubms(ctx, srvc.ListSubmsParams{
		Limit:  10000000,
		Offset: 0,
		Search: "",
		Author: nil,
	})
	if err != nil {
		slog.Error("failed to list submissions", "error", err)
		return
	}

	slog.Info("processing submissions", "total", len(subms))
	bar := progressbar.Default(int64(len(subms)), "recalculating scores")

	for _, subm := range subms {
		bar.Add(1)

		if subm.CurrEvalUUID == uuid.Nil {
			reEvalErr := submSrvc.ReEvalSubm(ctx, subm.UUID)
			slog.Info("re-evaluating submission", "subm_uuid", subm.UUID)
			if reEvalErr != nil {
				slog.Error("failed to re-evaluate submission", "error", reEvalErr)
				panic(reEvalErr)
			}

			subm, err = submSrvc.ViewSubm(ctx, subm.UUID)
			if err != nil {
				slog.Error("failed to get subm", "error", err)
				panic(err)
			}

			for {
				eval, err := submSrvc.GetEval(ctx, subm.CurrEvalUUID)
				if err != nil {
					slog.Error("failed to get eval", "error", err)
					panic(err)
				}
				if eval.Stage == domain.EvalStageFinished {
					break
				}
				time.Sleep(1 * time.Second)
			}
			slog.Info("eval finished", "eval_uuid", subm.CurrEvalUUID)
		}

		eval, err := submSrvc.GetEval(ctx, subm.CurrEvalUUID)
		if err != nil {
			slog.Error("failed to get eval", "error", err)
			fmt.Printf("%+v\n", subm)
			panic(err)
		}

		storeEvalErr := evalPgRepo.StoreEval(ctx, eval)
		if storeEvalErr != nil {
			slog.Error("failed to store eval", "error", storeEvalErr)
			panic(storeEvalErr)
		}
	}

	slog.Info("score migration completed")
}
