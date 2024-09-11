package task

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/exp/slog"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// tasks service example implementation.
// The example methods log the requests and return zero values.
type TaskService struct {
	ddbClient      *dynamodb.Client
	taskTableName  string
	s3PublicBucket *s3Bucket
}

func NewTaskSrvc() *TaskService {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	taskTableName := os.Getenv("DDB_TASK_TABLE_NAME")
	if taskTableName == "" {
		slog.Error("DDB_TASK_TABLE_NAME is not set")
		os.Exit(1)
	}

	return &TaskService{
		ddbClient:      dynamodbClient,
		taskTableName:  "proglv_tasks_v2",
		s3PublicBucket: NewS3BucketUploader("eu-central-1", "proglv-public"),
	}
}
