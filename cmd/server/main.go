package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/submsrvc"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/usersrvc"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		slog.Error("JWT_KEY is not set")
		os.Exit(1)
	}

	evalSrvc := evalsrvc.NewEvalSrvc()

	taskSrvc, err := tasksrvc.NewTaskSrvc()
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}
	submSrvc, err := submsrvc.NewSubmSrvc(taskSrvc, evalSrvc)
	if err != nil {
		log.Fatalf("error creating submission service: %v", err)
	}
	userSrvc := usersrvc.NewUsers()
	httpServer := http.NewHttpServer(submSrvc, userSrvc, taskSrvc, evalSrvc,
		[]byte(jwtKey))

	address := ":8080"
	log.Printf("Starting server on %s", address)
	err = httpServer.Start(":8080")
	log.Printf("Server stopped with error: %v", err)
}
