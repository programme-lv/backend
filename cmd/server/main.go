package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/s3bucket"
	http2 "github.com/programme-lv/backend/subm/http"
	pgrepo1 "github.com/programme-lv/backend/subm/pgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	taskhttp "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user"
	userhttp "github.com/programme-lv/backend/user/http"
)

func main() {
	w := os.Stderr

	// set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelInfo,
			TimeFormat: time.Kitchen,
			AddSource:  true,
		}),
	))

	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		slog.Error("JWT_KEY is not set")
		os.Exit(1)
	}

	execSrvc := execsrvc.NewExecSrvc()

	pg, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}
	userSrvc := user.NewUserService(pg)

	repo := repo.NewTaskPgRepo(pg)

	// Create task service
	var taskSrvc srvc.TaskSrvcClient
	publicS3, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-public")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		log.Fatalf(format, err)
	}
	testS3, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		log.Fatalf(format, err)
	}

	taskSrvc, err = srvc.NewTaskSrvc(repo, publicS3, testS3)
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}
	slog.Info("Task service initialized")

	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := taskhttp.NewTaskHttpHandler(taskSrvc)
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	userHttpHandler := userhttp.NewUserHttpHandler(userSrvc, []byte(jwtKey), userhttp.WithCookieDomain(cookieDomain))

	httpServer := http.NewHttpServer(submHttpHandler, taskHttpHandler, userHttpHandler, execSrvc, []byte(jwtKey))

	address := ":8080"
	slog.Info("starting server", "address", address)
	err = httpServer.Start(":8080")
	slog.Info("server stopped", "error", err)
}

func newSubmHttpHandler(userSrvc *user.UserSrvc, taskSrvc srvc.TaskSrvcClient, execSrvc *execsrvc.ExecSrvc) *http2.SubmHttpHandler {
	pool, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}

	submPgRepo := pgrepo1.NewPgSubmRepo(pool)
	evalPgRepo := pgrepo1.NewPgEvalRepo(pool)
	submSrvc := submsrvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)
	if err != nil {
		log.Fatalf("error creating submission service: %v", err)
	}

	submHttpServer := http2.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)

	return submHttpServer
}
