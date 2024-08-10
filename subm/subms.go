package subm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	"github.com/guregu/dynamo/v2"
	"github.com/programme-lv/backend/auth"
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

	return &submissionssrvc{
		ddbSubmTable: NewDynamoDbSubmTable(dynamodbClient, "proglv_submissions"),
		userSrvc:     user.NewUsers(ctx),
		taskSrvc:     task.NewTasks(),
		jwtKey:       []byte(jwtKey),
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

// CreateSubmission implements submissions.Service.
func (s *submissionssrvc) CreateSubmission(ctx context.Context, p *submgen.CreateSubmissionPayload) (res *submgen.Submission, err error) {
	submContent := SubmissionContent{Value: p.Submission}

	for _, v := range []Validatable{&submContent} {
		err := v.IsValid()
		if err != nil {
			return nil, err
		}
	}

	userSrvcUser, err := s.userSrvc.GetUserByUsername(ctx, &usergen.GetUserByUsernamePayload{Username: p.Username})
	if err != nil {
		log.Errorf(ctx, err, "error getting user: %+v", err.Error())
		if e, ok := err.(usergen.NotFound); ok {
			return nil, submgen.InvalidSubmissionDetails(string(e))
		}
		return nil, submgen.InternalError("error getting user")
	}

	claims := ctx.Value(ClaimsKey("claims")).(*auth.Claims)
	log.Printf(ctx, "%+v", claims)

	if claims.UUID != userSrvcUser.UUID {
		return nil, submgen.Unauthorized("jwt claims uuid does not match username's user's uuid")
	}

	taskSrvTask, err := s.taskSrvc.GetTask(ctx, &taskgen.GetTaskPayload{
		TaskID: p.TaskCodeID,
	})
	if err != nil {
		log.Errorf(ctx, err, "error getting task: %+v", err.Error())
		if e, ok := err.(taskgen.TaskNotFound); ok {
			return nil, submgen.InvalidSubmissionDetails(string(e))
		}
		return nil, submgen.InternalError("error getting task")
	}

	langs := getHardcodedLanguageList()
	var foundPLang *ProgrammingLang = nil
	for _, lang := range langs {
		if lang.ID == p.ProgrammingLangID {
			foundPLang = &lang
		}
	}

	if foundPLang == nil {
		return nil, submgen.InvalidSubmissionDetails("invalid programming language")
	}

	uuid := uuid.New()
	createdAt := time.Now()
	row := &SubmissionRow{
		Uuid:       uuid.String(),
		UnixTime:   createdAt.Unix(),
		Content:    submContent.String(),
		Version:    0,
		AuthorUuid: userSrvcUser.UUID,
		ProgLangId: foundPLang.ID,
		TaskId:     taskSrvTask.PublishedTaskID,
	}

	err = s.ddbSubmTable.Save(ctx, row)
	if err != nil {
		// TODO: automatically retry with exponential backoff on version conflict
		if dynamo.IsCondCheckFailed(err) {
			log.Errorf(ctx, err, "version conflict saving user")
			return nil, submgen.InternalError("version conflict saving user")
		} else {
			log.Errorf(ctx, err, "error saving user")
			return nil, submgen.InternalError("error saving user")
		}
	}

	createdAtRfc3339 := createdAt.Format(time.RFC3339)

	res = &submgen.Submission{
		UUID:       row.Uuid,
		Submission: row.Content,
		Username:   userSrvcUser.Username,
		CreatedAt:  createdAtRfc3339,
		Evaluation: nil,
		Language: &submgen.SubmProgrammingLang{
			ID:       foundPLang.ID,
			FullName: foundPLang.FullName,
			MonacoID: foundPLang.MonacoId,
		},
		Task: &submgen.SubmTask{
			Name: taskSrvTask.TaskFullName,
			Code: taskSrvTask.PublishedTaskID,
		},
	}

	return res, nil
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
