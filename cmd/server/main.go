package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/http"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/tasksrvc"
	"github.com/programme-lv/backend/user"
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

	submSrvc := subm.NewSubmissions()
	userSrvc := user.NewUsers()
	taskSrvc, err := tasksrvc.NewTaskSrvc()
	if err != nil {
		log.Fatalf("error creating task service: %v", err)
	}
	httpServer := http.NewHttpServer(submSrvc, userSrvc, taskSrvc,
		[]byte(jwtKey))

	address := ":8080"
	log.Printf("Starting server on %s", address)
	err = httpServer.Start(":8080")
	log.Printf("Server stopped with error: %v", err)
}
