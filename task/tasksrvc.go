package task

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"goa.design/clue/log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// tasks service example implementation.
// The example methods log the requests and return zero values.
type TaskSrvc struct {
	ddbTaskTable *DynamoDbTaskTable
}

// NewTasks returns the tasks service implementation.
func NewTasks(ctx context.Context) *TaskSrvc {
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
		log.Fatalf(ctx,
			errors.New("DDB_TASK_TABLE_NAME is not set"),
			"cant read DDB_TASK_TABLE_NAME from env in new tasks service constructor")
	}

	return &TaskSrvc{
		ddbTaskTable: NewDynamoDbTaskTable(dynamodbClient, taskTableName),
	}
}

// List all tasks
func (s *TaskSrvc) ListTasks(ctx context.Context) (res []*Task, err error) {
	all, err := s.ddbTaskTable.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not list tasks: %w", err)
	}

	res = make([]*Task, 0)

	for _, row := range all {
		task, err := ddbTaskRowToResponse(row)
		if err != nil {
			return nil, fmt.Errorf("could not convert task row to response: %w", err)
		}
		res = append(res, task)
	}

	return
}

// Get a task by its ID
func (s *TaskSrvc) GetTask(ctx context.Context, p *GetTaskPayload) (res *Task, err error) {
	row, err := s.ddbTaskTable.Get(ctx, p.TaskID)
	if err != nil {
		return nil, fmt.Errorf("could not get task: %w", err)
	}
	if row == nil {
		return nil, newErrTaskNotFound()
	}

	return ddbTaskRowToResponse(row)
}

func ddbTaskRowToResponse(row *TaskRow) (res *Task, err error) {
	taskManifest, err := ParseTaskTomlManifest(row.TomlManifest)
	if err != nil {
		return nil, fmt.Errorf("could not parse task toml manifest: %w", err)
	}

	mds := taskManifest.Statement.MDs
	var responseDefaulMdStatement *MarkdownStatement = nil
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

		responseDefaulMdStatement = &MarkdownStatement{
			Story:   resolveImgsToUrls(defaultMd.Story.Content),
			Input:   resolveImgsToUrls(defaultMd.Input.Content),
			Output:  resolveImgsToUrls(defaultMd.Output.Content),
			Notes:   notes,
			Scoring: scoring,
		}
	}

	var illustrationImgUrl *string = nil
	if taskManifest.Statement.IllustrationImg.S3ObjKey != "" {
		illustrationImgUrlStrl := fmt.Sprintf("https://dvhk4hiwp1rmf.cloudfront.net/%s", taskManifest.Statement.IllustrationImg.S3ObjKey)
		illustrationImgUrl = &illustrationImgUrlStrl
	}

	var examples []*Example = make([]*Example, 0)
	for _, example := range taskManifest.Statement.Examples {
		examples = append(examples, &Example{
			Input:  example.Input,
			Output: example.Output,
			MdNote: example.MdNote,
		})
	}

	var stInputs []*StInputs = make([]*StInputs, 0)
	for _, st := range taskManifest.Statement.VisInpSTs {
		stInputs = append(stInputs, &StInputs{
			Subtask: st.Subtask,
			Inputs:  st.Inputs,
		})
	}

	var defaultPdfStatementURL *string = nil
	if len(taskManifest.Statement.PDFs) > 0 {
		pdf := taskManifest.Statement.PDFs[0]
		defaultPdfStatementURLStr := fmt.Sprintf("https://dvhk4hiwp1rmf.cloudfront.net/task-pdf-statements/%s.pdf", pdf.SHA256)
		defaultPdfStatementURL = &defaultPdfStatementURLStr
	}

	res = &Task{
		PublishedTaskID:        row.Id,
		TaskFullName:           taskManifest.FullName,
		MemoryLimitMegabytes:   taskManifest.Contraints.MemoryLimMB,
		CPUTimeLimitSeconds:    taskManifest.Contraints.CpuTimeInSecs,
		OriginOlympiad:         taskManifest.Metadata.OriginOlympiad,
		IllustrationImgURL:     illustrationImgUrl,
		DifficultyRating:       taskManifest.Metadata.Difficulty,
		DefaultMdStatement:     responseDefaulMdStatement,
		Examples:               examples,
		DefaultPdfStatementURL: defaultPdfStatementURL,
		OriginNotes:            taskManifest.Metadata.OriginNotes,
		VisibleInputSubtasks:   stInputs,
	}

	return
}

// GetTaskSubmEvalData implements tasks.Service.
func (s *TaskSrvc) GetTaskSubmEvalData(ctx context.Context, p *GetTaskSubmEvalDataPayload) (res *TaskSubmEvalData, err error) {
	row, err := s.ddbTaskTable.Get(ctx, p.TaskID)
	if err != nil {
		return nil, fmt.Errorf("could not get task: %w", err)
	}
	if row == nil {
		return nil, newErrTaskNotFound()
	}

	taskManifest, err := ParseTaskTomlManifest(row.TomlManifest)
	if err != nil {
		return nil, fmt.Errorf("could not parse task toml manifest: %w", err)
	}

	tests := make([]*TaskEvalTestInformation, 0)
	testToTestGroupMap := make(map[int]*TomlTestGroup)
	for j, testGroup := range taskManifest.TestGroups {
		for i := range testGroup.TestIDs {
			testToTestGroupMap[testGroup.TestIDs[i]] = &taskManifest.TestGroups[j]
		}
	}

	for i, test := range taskManifest.Tests {
		// TODO: as of now subtasks without testgroups are not implemented
		// currently test subtasks are determined by the testgroup that they belong to
		subtasks := make([]int, 0)
		if testGroup, ok := testToTestGroupMap[i+1]; ok {
			subtasks = append(subtasks, testGroup.Subtask)
		}
		var testGroupId *int = nil
		if testGroup, ok := testToTestGroupMap[i+1]; ok {
			testGroupId = &testGroup.GroupID
		}
		tests = append(tests, &TaskEvalTestInformation{
			TestID:          i + 1,
			FullInputS3URI:  fmt.Sprintf("s3://proglv-tests/%s.zst", test.InputSHA256),
			InputSha256:     test.InputSHA256,
			FullAnswerS3URI: fmt.Sprintf("s3://proglv-tests/%s.zst", test.AnswerSHA256),
			AnswerSha256:    test.AnswerSHA256,
			Subtasks:        subtasks,
			TestGroup:       testGroupId,
		})
	}

	testgroups := make([]*TaskEvalTestGroupInformation, 0)
	for _, testGroup := range taskManifest.TestGroups {
		testgroups = append(testgroups, &TaskEvalTestGroupInformation{
			TestGroupID: testGroup.GroupID,
			Score:       testGroup.Points,
			Subtask:     testGroup.Subtask,
		})
	}

	res = &TaskSubmEvalData{
		PublishedTaskID:      row.Id,
		TaskFullName:         taskManifest.FullName,
		MemoryLimitMegabytes: taskManifest.Contraints.MemoryLimMB,
		CPUTimeLimitSeconds:  taskManifest.Contraints.CpuTimeInSecs,
		Tests:                tests,
		TestlibCheckerCode:   testlib_default_checker,
		SubtaskScores:        []*TaskEvalSubtaskScore{},
		TestGroupInformation: testgroups,
	}

	return res, nil
}
