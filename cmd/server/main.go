package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/subm/submhttp"
	"github.com/programme-lv/backend/subm/submpgrepo"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/task"
	"github.com/programme-lv/backend/usersrvc"
)

func main() {
	slog.SetDefault(slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
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

	taskSrvc, err := task.NewDefaultTaskSrvc()
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}

	submHttpServer := newSubmHttpHandler(userSrvc, taskSrvc, execSrvc)

	httpServer := http.NewHttpServer(submHttpServer, userSrvc, taskSrvc, execSrvc, []byte(jwtKey))

	address := ":8080"
	slog.Info("starting server", "address", address)
	err = httpServer.Start(":8080")
	slog.Info("server stopped", "error", err)
}

func newSubmHttpHandler(userSrvc *usersrvc.UserSrvc, taskSrvc task.TaskSrvcClient, execSrvc *execsrvc.ExecSrvc) *submhttp.SubmHttpHandler {
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
