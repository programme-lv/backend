package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/conf"
	"github.com/programme-lv/backend/task/repo"
	"github.com/programme-lv/backend/task/srvc"
)

func main() {

	err := godotenv.Load(".env.prod")
	if err != nil {
		panic("Error loading .env file")
	}

	pool, err := pgxpool.New(context.Background(), conf.GetPgConnStrFromEnv())
	if err != nil {
		log.Fatalf("failed to create pg pool: %v", err)
	}

	repo := repo.NewTaskPgRepo(pool)

	taskJsonL, err := os.ReadFile("./all_tasks.jsonl")
	if err != nil {
		log.Fatalf("failed to read tasks JSONL file: %v", err)
	}

	// Split the JSONL file into lines
	lines := bytes.Split(taskJsonL, []byte("\n"))
	log.Printf("Found %d task entries in JSONL file", len(lines))

	// Process each line
	for i, line := range lines {
		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Unmarshal the JSON line into a task
		var task srvc.Task
		if err := json.Unmarshal(line, &task); err != nil {
			log.Printf("Error unmarshalling task at line %d: %v", i+1, err)
			continue
		}

		// Create the task in the repository
		if err := repo.CreateTask(context.Background(), task); err != nil {
			log.Printf("Error creating task %s: %v", task.ShortId, err)
			continue
		}

		log.Printf("Successfully created task: %s", task.ShortId)
	}

	log.Println("Task import completed")
}
