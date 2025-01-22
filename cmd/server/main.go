package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/http/submhttp"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
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

	evalSrvc := execsrvc.NewExecSrvc()
	userSrvc := usersrvc.NewUserService()

	taskSrvc, err := tasksrvc.NewTaskSrvc()
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}
	submSrvc, err := submsrvc.NewSubmSrvc(userSrvc, taskSrvc, evalSrvc)
	if err != nil {
		log.Fatalf("error creating submission service: %v", err)
	}
	submHttpServer := submhttp.NewSubmHttpHandler(submSrvc, taskSrvc, userSrvc)
	httpServer := http.NewHttpServer(submHttpServer, submSrvc, userSrvc, taskSrvc, evalSrvc,
		[]byte(jwtKey))

	address := ":8080"
	slog.Info("starting server", "address", address)
	err = httpServer.Start(":8080")
	slog.Info("server stopped", "error", err)
}
