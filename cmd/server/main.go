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
	"github.com/programme-lv/backend/subm/submhttp"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	http1 "github.com/programme-lv/backend/task/http"
	"github.com/programme-lv/backend/task/pgrepo"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/usersrvc"
)

func main() {
	w := os.Stderr

	// set global logger with custom options
	slog.SetDefault(slog.New(
		tint.NewHandler(w, &tint.Options{
			Level:      slog.LevelDebug,
			TimeFormat: time.Kitchen,
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
	userSrvc := usersrvc.NewUserService()

	pg, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}

	repo := pgrepo.NewTaskPgRepo(pg)
	taskSrvc, err := srvc.NewDefaultTaskSrvc(repo)
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}

	submHttpHandler := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)
	taskHttpHandler := http1.NewTaskHttpHandler(taskSrvc)

	httpServer := http.NewHttpServer(submHttpHandler, taskHttpHandler, userSrvc, execSrvc, []byte(jwtKey))

	address := ":8080"
	slog.Info("starting server", "address", address)
	err = httpServer.Start(":8080")
	slog.Info("server stopped", "error", err)
}

func newSubmHttpHandler(userSrvc *usersrvc.UserSrvc, taskSrvc srvc.TaskSrvcClient, execSrvc *execsrvc.ExecSrvc) *submhttp.SubmHttpHandler {
	pool, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}

	submPgRepo := submpgrepo.NewPgSubmRepo(pool)
	evalPgRepo := submpgrepo.NewPgEvalRepo(pool)
	submSrvc := submsrvc.NewSubmSrvc(userSrvc, taskSrvc, execSrvc, submPgRepo, evalPgRepo)
	if err != nil {
		log.Fatalf("error creating submission service: %v", err)
	}

	submHttpServer := submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)

	return submHttpServer
}
