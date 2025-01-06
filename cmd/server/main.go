package main

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/lmittmann/tint"
	"github.com/programme-lv/backend/execsrvc"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
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

	evalSrvc := execsrvc.NewDefaultExecSrvc()

	taskSrvc, err := tasksrvc.NewTaskSrvc()
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}
	submSrvc, err := submsrvc.NewSubmSrvc(taskSrvc, evalSrvc)
	if err != nil {
		log.Fatalf("error creating submission service: %v", err)
	}
	userSrvc := usersrvc.NewUserService()
	httpServer := http.NewHttpServer(submSrvc, userSrvc, taskSrvc, evalSrvc,
		[]byte(jwtKey))

	address := ":8080"
	slog.Info("starting server", "address", address)
	err = httpServer.Start(":8080")
	slog.Info("server stopped", "error", err)
}
