package subm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	submgen "github.com/programme-lv/backend/gen/submissions"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	usergen "github.com/programme-lv/backend/gen/users"
	"github.com/programme-lv/backend/task"
	"github.com/programme-lv/backend/user"
	"goa.design/clue/log"
)

// submissions service example implementation.
// The example methods log the requests and return zero values.
type submissionssrvc struct {
	ddbSubmTable *DynamoDbSubmTable
	userSrvc     usergen.Service
	taskSrvc     taskgen.Service
	jwtKey       []byte
	sqsClient    *sqs.Client
	submQueueUrl string
}

// NewSubmissions returns the submissions service implementation.
func NewSubmissions(ctx context.Context) submgen.Service {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("eu-central-1"),
		config.WithSharedConfigProfile("kp"),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load SDK config, %v", err))
	}
	dynamodbClient := dynamodb.NewFromConfig(cfg)

	jwtKey := os.Getenv("JWT_KEY")
	if jwtKey == "" {
		log.Fatalf(ctx,
			errors.New("JWT_KEY is not set"),
			"cant read JWT_KEY from env in new user service constructor")
	}

	submTableName := os.Getenv("DDB_SUBM_TABLE_NAME")
	if submTableName == "" {
		log.Fatalf(ctx,
			errors.New("DDB_SUBM_TABLE_NAME is not set"),
			"cant read DDB_SUBM_TABLE_NAME from env in new user service constructor")
	}

	sqsClient := sqs.NewFromConfig(cfg)

	submQueueUrl := os.Getenv("SUBM_SQS_QUEUE_URL")
	if submQueueUrl == "" {
		panic("SUBM_SQS_QUEUE_URL not set in .env file")
	}

	return &submissionssrvc{
		ddbSubmTable: NewDynamoDbSubmTable(dynamodbClient, submTableName),
		userSrvc:     user.NewUsers(ctx),
		taskSrvc:     task.NewTasks(ctx),
		jwtKey:       []byte(jwtKey),
		sqsClient:    sqsClient,
		submQueueUrl: submQueueUrl,
	}
}

type Validatable interface {
	IsValid() error
}

type SubmissionContent struct {
	Value string
}

func (subm *SubmissionContent) IsValid() error {
	const maxSubmissionLength = 128000 // 128 KB
	if len(subm.Value) > maxSubmissionLength {
		return submgen.InvalidSubmissionDetails(
			"maksimālais iesūtījuma garums ir 128 KB",
		)
	}
	return nil
}

func (subm *SubmissionContent) String() string {
	return subm.Value
}

// List all submissions
func (s *submissionssrvc) ListSubmissions(ctx context.Context) (res []*submgen.Submission, err error) {
	subms, err := s.ddbSubmTable.List(ctx)
	if err != nil {
		return nil, submgen.InternalError("error retrieving submission list")
	}

	users, err := s.userSrvc.ListUsers(ctx)
	if err != nil {
		return nil, submgen.InternalError("error retrieving users")
	}

	userUuidToUsername := make(map[string]string)
	for _, user := range users {
		userUuidToUsername[user.UUID] = user.Username
	}

	pLangIdToDetails := make(map[string]struct {
		fullName string
		monacoId string
	})
	pLangs := getHardcodedLanguageList()
	for _, lang := range pLangs {
		pLangIdToDetails[lang.ID] = struct {
			fullName string
			monacoId string
		}{
			fullName: lang.FullName,
			monacoId: lang.MonacoId,
		}
	}

	tasks, err := s.taskSrvc.ListTasks(ctx)
	if err != nil {
		return nil, submgen.InternalError("error retrieving tasks")
	}

	taskIdToDetailsMap := make(map[string]*taskgen.Task)
	for _, task := range tasks {
		taskIdToDetailsMap[task.PublishedTaskID] = task
	}

	res = make([]*submgen.Submission, 0)
	for _, subm := range subms {
		author := subm.AuthorUuid
		username, ok := userUuidToUsername[author]
		if !ok {
			log.Printf(ctx, "user %v not found for submission %v", subm.AuthorUuid, subm.Uuid)
			continue
		}
		createdAt := time.Unix(subm.UnixTime, 0)
		createdAtRfc3339 := createdAt.Format(time.RFC3339)
		pLangDetails, ok := pLangIdToDetails[subm.ProgLangId]
		if !ok {
			log.Printf(ctx, "programming language %v not found for submission %v", subm.ProgLangId, subm.Uuid)
			continue
		}

		submTask, ok := taskIdToDetailsMap[subm.TaskId]
		if !ok {
			log.Printf(ctx, "task %v not found for submission %v", subm.TaskId, subm.Uuid)
			continue
		}

		res = append(res, &submgen.Submission{
			UUID:       subm.Uuid,
			Submission: subm.Content,
			Username:   username,
			CreatedAt:  createdAtRfc3339,
			Evaluation: nil,
			Language: &submgen.SubmProgrammingLang{
				ID:       subm.ProgLangId,
				FullName: pLangDetails.fullName,
				MonacoID: pLangDetails.fullName,
			},
			Task: &submgen.SubmTask{
				Name: submTask.TaskFullName,
				Code: submTask.PublishedTaskID,
			},
		})
	}

	return res, nil
}

// Get a submission by UUID
func (s *submissionssrvc) GetSubmission(ctx context.Context, p *submgen.GetSubmissionPayload) (res *submgen.Submission, err error) {
	res = &submgen.Submission{}
	log.Printf(ctx, "submissions.getSubmission")
	return
}

// ListProgrammingLanguages implements submissions.Service.
func (s *submissionssrvc) ListProgrammingLanguages(context.Context) (res []*submgen.ProgrammingLang, err error) {
	res = make([]*submgen.ProgrammingLang, 0)
	langs := getHardcodedLanguageList()
	for _, lang := range langs {
		res = append(res, &submgen.ProgrammingLang{
			ID:               lang.ID,
			FullName:         lang.FullName,
			CodeFilename:     &lang.CodeFilename,
			CompileCmd:       lang.CompileCmd,
			ExecuteCmd:       lang.ExecuteCmd,
			EnvVersionCmd:    lang.EnvVersionCmd,
			HelloWorldCode:   lang.HelloWorldCode,
			MonacoID:         lang.MonacoId,
			CompiledFilename: lang.CompiledFilename,
			Enabled:          lang.Enabled,
		})
	}
	return res, nil
}
