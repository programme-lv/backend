package task

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	"goa.design/clue/log"
)

// tasks service example implementation.
// The example methods log the requests and return zero values.
type taskssrvc struct {
	ddbTaskTable *DynamoDbTaskTable
}

// NewTasks returns the tasks service implementation.
func NewTasks() taskgen.Service {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	return &taskssrvc{
		ddbTaskTable: NewDynamoDbTaskTable(dynamodbClient, "ProglvUsers"),
	}
}

// List all tasks
func (s *taskssrvc) ListTasks(ctx context.Context) (res []*taskgen.Task, err error) {
	log.Printf(ctx, "tasks.listTasks")
	return
}

// Get a task by its ID
func (s *taskssrvc) GetTask(ctx context.Context, p *taskgen.GetTaskPayload) (res *taskgen.Task, err error) {
	res = &taskgen.Task{
		PublishedTaskID:        "hello",
		TaskFullName:           "",
		MemoryLimitMegabytes:   0,
		CPUTimeLimitSeconds:    0,
		OriginOlympiad:         "",
		IllustrationImgURL:     new(string),
		DifficultyRating:       0,
		DefaultMdStatement:     &taskgen.MarkdownStatement{},
		Examples:               []*taskgen.Example{},
		DefaultPdfStatementURL: new(string),
		OriginNotes:            map[string]string{},
		VisibleInputSubtasks:   []*taskgen.StInputs{},
	}
	log.Printf(ctx, "tasks.getTask")
	return
}
