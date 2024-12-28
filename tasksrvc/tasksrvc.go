package tasksrvc

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/programme-lv/backend/s3bucket"
)

type TaskService struct {
	tasks []Task

	testFileCache sync.Map

	s3PublicBucket   *s3bucket.S3Bucket
	s3TestfileBucket *s3bucket.S3Bucket
	s3TaskBucket     *s3bucket.S3Bucket
}

func NewTaskSrvc() (*TaskService, error) {
	publicBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-public")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	testFileBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tests")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}
	taskBucket, err := s3bucket.NewS3Bucket("eu-central-1", "proglv-tasks")
	if err != nil {
		format := "failed to create S3 bucket: %w"
		return nil, fmt.Errorf(format, err)
	}

	start := time.Now()
	slog.Info("downloading tasks from S3", "bucket", taskBucket.Bucket())
	taskFiles, err := taskBucket.ListAndGetAllFiles("")
	if err != nil {
		return nil, fmt.Errorf("failed to list task files: %w", err)
	}
	elapsed := time.Since(start)
	slog.Info("downloaded tasks from S3",
		"bucket", taskBucket.Bucket(),
		"count", len(taskFiles),
		"time_ms", elapsed.Milliseconds())

	// unmarshall jsons in taskFiles to tasks
	tasks := []Task{}
	for _, taskFile := range taskFiles {
		task := Task{}
		err = json.Unmarshal(taskFile.Content, &task)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal task: %w", err)
		}
		tasks = append(tasks, task)
	}

	return &TaskService{
		tasks: tasks,

		testFileCache: sync.Map{},

		s3PublicBucket:   publicBucket,
		s3TestfileBucket: testFileBucket,
		s3TaskBucket:     taskBucket,
	}, nil
}
