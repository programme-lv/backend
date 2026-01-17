package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/http"
	submhttp "github.com/programme-lv/backend/subm/http"
	submpgrepo "github.com/programme-lv/backend/subm/pgrepo"
	"github.com/programme-lv/backend/subm/srvc"
	taskhttp "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/repo"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
	usersrvc "github.com/programme-lv/backend/user"
	userhttp "github.com/programme-lv/backend/user/http"
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

	cdnS3 := conf.MustGetPublicS3Bucket()
	testS3 := conf.MustGetTestfileS3Bucket()

	execCtx := context.Background()
	execCtx = ctxlog.WithLogger(execCtx, slog.Default().With("module", "exec"))
	s3Client := conf.MustGetS3ClientFromEnv(execCtx)
	s3Bucket := os.Getenv("S3_EXEC_BUCKET")
	if s3Bucket == "" {
		panic("S3_EXEC_BUCKET not set in .env file")
	}
	s3Repo := exec.NewS3ExecRepo(execCtx, s3Client, s3Bucket)
	execSrvc := exec.NewExecSrvc(execCtx, s3Repo)
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
	taskSrvc, err := tasksrvc.NewTaskSrvc(taskRepo, cdnS3, testS3)
	if err != nil {
		slog.Error("error creating task service", "error", err)
		os.Exit(1)
	}

	// Initialize HTTP handlers
	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := taskhttp.NewTaskHttpHandler(taskSrvc)
	userHttpHandler := userhttp.NewUserHttpHandler(userSrvc, jwtKey, userhttp.WithCookieDomain(cookieDomain))

	// Start HTTP server
	httpServer := http.NewHttpServer(submHttpHandler, taskHttpHandler, userHttpHandler, execSrvc, jwtKey)

	slog.Info("starting server", "address", address)
	err = httpServer.Start(address)
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

func newSubmHttpHandler(userSrvc usersrvc.UserSrvcClient, taskSrvc tasksrvc.TaskSrvcClient, execSrvc exec.ExecSrvcClient) *submhttp.SubmHttpHandler {
	pgPool, err := conf.GetPgxPoolFromEnv()
	if err != nil {
		slog.Error("failed to create pg pool", "error", err)
		os.Exit(1)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pgPool)
	submSrvc := srvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)

	return submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)
}
