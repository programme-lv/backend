package task

import (
	"context"
	"fmt"
	"strings"

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
		ddbTaskTable: NewDynamoDbTaskTable(dynamodbClient, "ProglvTasks"),
	}
}

// List all tasks
func (s *taskssrvc) ListTasks(ctx context.Context) (res []*taskgen.Task, err error) {
	log.Printf(ctx, "tasks.listTasks")
	return
}

// Get a task by its ID
func (s *taskssrvc) GetTask(ctx context.Context, p *taskgen.GetTaskPayload) (res *taskgen.Task, err error) {
	row, err := s.ddbTaskTable.Get(ctx, p.TaskID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, fmt.Errorf("task not found")
	}

	taskManifest, err := ParseTaskTomlManifest(row.TomlManifest)
	if err != nil {
		return nil, fmt.Errorf("could not parse task toml manifest: %w", err)
	}

	mds := taskManifest.Statement.MDs
	var responseDefaulMdStatement *taskgen.MarkdownStatement = nil
	if len(mds) > 0 {
		defaultMd := mds[0]
		resolveImgsToUrls := func(mdSection string) string {
			for uuid, key := range defaultMd.ImgUuidToS3Key {
				url := fmt.Sprintf("https://dvhk4hiwp1rmf.cloudfront.net/%s", key)
				mdSection = strings.Replace(mdSection, uuid, url, 1)
			}
			return mdSection
		}
		var notes *string = nil
		if defaultMd.Notes.Content != "" {
			notesStr := resolveImgsToUrls(defaultMd.Notes.Content)
			notes = &notesStr
		}
		var scoring *string = nil
		if defaultMd.Scoring.Content != "" {
			scoringStr := resolveImgsToUrls(defaultMd.Scoring.Content)
			scoring = &scoringStr
		}

		responseDefaulMdStatement = &taskgen.MarkdownStatement{
			Story:   resolveImgsToUrls(defaultMd.Story.Content),
			Input:   resolveImgsToUrls(defaultMd.Input.Content),
			Output:  resolveImgsToUrls(defaultMd.Output.Content),
			Notes:   notes,
			Scoring: scoring,
		}
	}

	res = &taskgen.Task{
		PublishedTaskID:        "hello",
		TaskFullName:           "",
		MemoryLimitMegabytes:   0,
		CPUTimeLimitSeconds:    0,
		OriginOlympiad:         "",
		IllustrationImgURL:     new(string),
		DifficultyRating:       0,
		DefaultMdStatement:     responseDefaulMdStatement,
		Examples:               []*taskgen.Example{},
		DefaultPdfStatementURL: new(string),
		OriginNotes:            map[string]string{},
		VisibleInputSubtasks:   []*taskgen.StInputs{},
	}
	log.Printf(ctx, "tasks.getTask")
	return
}
