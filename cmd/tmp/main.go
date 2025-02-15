package main

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/task"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	taskSrvc, err := task.NewDefaultTaskSrvc()
	if err != nil {
		log.Fatalf("failed to create task srvc: %v", err)
	}

	tasks, err := taskSrvc.ListTasks(context.Background())
	if err != nil {
		log.Fatalf("failed to list tasks: %v", err)
	}

	log.Printf("tasks: %v", tasks)
}
