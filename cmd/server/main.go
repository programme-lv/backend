package main

import (
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/exec"
	"github.com/programme-lv/backend/http"
	submhttp "github.com/programme-lv/backend/subm/http"
	submpgrepo "github.com/programme-lv/backend/subm/pgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	taskhttp "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user"
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

	execSrvc := exec.NewExecSrvc()
	if *listenSQS {
		err := execSrvc.ListenToResultSQS()
		if err != nil {
			slog.Error("failed to listen to result SQS", "error", err)
			os.Exit(1)
		}
	} else {
		slog.Info("listening to execution result SQS disabled")
	}

	// Initialize user service
	userSrvc := user.NewUserService(pgPool)

	// Initialize task service
	taskRepo := repo.NewTaskPgRepo(pgPool)
	taskSrvc, err := srvc.NewTaskSrvc(taskRepo, cdnS3, testS3)
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

func newSubmHttpHandler(userSrvc *user.UserSrvc, taskSrvc srvc.TaskSrvcClient, execSrvc *exec.ExecSrvc) *submhttp.SubmHttpHandler {
	pgPool, err := conf.GetPgxPoolFromEnv()
	if err != nil {
		slog.Error("failed to create pg pool", "error", err)
		os.Exit(1)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pgPool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pgPool)
	submSrvc := submsrvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)

	return submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)
}
