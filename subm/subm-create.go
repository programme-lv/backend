package subm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/auth"
	submgen "github.com/programme-lv/backend/gen/submissions"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
)

func (s *submissionssrvc) createSubmissionWithValidatedInput(
	ctx context.Context,
	subm *string,
	user *usergen.User,
	task *taskgen.TaskSubmEvalData,
	lang *ProgrammingLang,
) (*submgen.Submission, error) {

	createdAt := time.Now().UTC()
	submUuid := uuid.New()
	evalUuid := uuid.New()

	var err error

	// DETERMINE WHETHER SCORING IS "tests", "subtask", OR "testgroup"
	scoringMethod := determineScoringMethod(task)
	switch scoringMethod {
	case "tests":
		// PUT SUBMISSION SCORING TESTS ROW
		submScoringTestsRow := &SubmissionScoringTestsRow{
			SubmUuid: submUuid.String(),
			SortKey:  fmt.Sprintf("subm#scoring#tests"),
			Accepted: 0,
			Wrong:    0,
			Untested: len(task.Tests),
			Gsi1Pk:   1,
		}
		item, err := attributevalue.MarshalMap(submScoringTestsRow)
		if err != nil {
			log.Printf(ctx, "error marshalling scoring tests row: %+v", err)
			return nil, submgen.InternalError("error marshalling scoring tests row")
		}
		_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: &s.submTableName, Item: item})
		if err != nil {
			log.Printf(ctx, "error saving submission scoring tests: %+v", err)
			return nil, submgen.InternalError("error saving submission scoring tests")
		}
	case "subtask":
		// PUT SUBMISSION SCORING SUBTASK ROWS
		submScoringSubtaskRows := make([]*SubmissionScoringSubtaskRow, 0)
		for _, subtask := range task.SubtaskScores {
			stTestCount := 0
			for _, test := range task.Tests {
				for _, testSt := range test.Subtasks {
					if testSt == subtask.SubtaskID {
						stTestCount++
					}
				}
			}
			row := &SubmissionScoringSubtaskRow{
				SubmUuid:      submUuid.String(),
				SortKey:       fmt.Sprintf("subm#scoring#subtask#%d", subtask.SubtaskID),
				ReceivedScore: 0,
				PossibleScore: subtask.Score,
				AcceptedTests: 0,
				WrongTests:    0,
				UntestedTests: stTestCount,
				Gsi1Pk:        1,
			}
			submScoringSubtaskRows = append(submScoringSubtaskRows, row)
		}
		batchSize := 25
		start := 0
		for start < len(submScoringSubtaskRows) {
			end := min(start+batchSize, len(submScoringSubtaskRows))
			batch := submScoringSubtaskRows[start:end]
			items := make([]types.WriteRequest, len(batch))
			for i := range len(batch) {
				item, err := attributevalue.MarshalMap(batch[i])
				if err != nil {
					log.Printf(ctx, "error marshalling scoring subtask row: %+v", err)
					return nil, submgen.InternalError("error marshalling scoring subtask row")
				}
				items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
			}
			_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
			})
			if err != nil {
				log.Printf(ctx, "error saving submission scoring subtasks: %+v", err)
				return nil, submgen.InternalError("error saving submission scoring subtasks")
			}
			start = end
		}
	case "testgroup":
		// PUT SUBMISSION SCORING TESTGROUP ROWS
		submScoringTestgroupRows := make([]*SubmissionScoringTestgroupRow, 0)
		for _, testGroup := range task.TestGroupInformation {
			tgTests := 0
			for _, test := range task.Tests {
				if test.TestGroup != nil && *test.TestGroup == testGroup.TestGroupID {
					tgTests++
				}
			}
			row := &SubmissionScoringTestgroupRow{
				SubmUuid:         submUuid.String(),
				SortKey:          fmt.Sprintf("subm#scoring#testgroup#%d", testGroup.TestGroupID),
				StatementSubtask: testGroup.Subtask,
				TestgroupScore:   testGroup.Score,
				AcceptedTests:    0,
				WrongTests:       0,
				UntestedTests:    tgTests,
				Gsi1Pk:           1,
			}

			submScoringTestgroupRows = append(submScoringTestgroupRows, row)
		}
		batchSize := 25
		start := 0
		for start < len(submScoringTestgroupRows) {
			end := min(start+batchSize, len(submScoringTestgroupRows))
			batch := submScoringTestgroupRows[start:end]
			items := make([]types.WriteRequest, len(batch))
			for i := range len(batch) {
				item, err := attributevalue.MarshalMap(batch[i])
				if err != nil {
					log.Printf(ctx, "error marshalling scoring testgroup row: %+v", err)
					return nil, submgen.InternalError("error marshalling scoring testgroup row")
				}
				items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
			}
			_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
			})
			if err != nil {
				log.Printf(ctx, "error saving submission scoring testgroups: %+v", err)
				return nil, submgen.InternalError("error saving submission scoring testgroups")
			}
			start = end
		}
	}

	// PUT EVALUATION DETAILS ROW
	evalDetailsRow := &SubmissionEvaluationDetailsRow{
		SubmUuid:                   submUuid.String(),
		SortKey:                    fmt.Sprintf("eval#%s#details", evalUuid.String()),
		EvaluationStage:            "waiting",
		TestlibCheckerCode:         task.TestlibCheckerCode,
		SystemInformation:          nil,
		SubmCompileStdout:          nil,
		SubmCompileStderr:          nil,
		SubmCompileExitCode:        nil,
		SubmCompileCpuTimeMillis:   nil,
		SubmCompileWallTimeMillis:  nil,
		SubmCompileMemoryKibiBytes: nil,
		ProgrammingLang: SubmEvalDetailsProgrammingLang{
			PLangId:        lang.ID,
			DisplayName:    lang.FullName,
			SubmCodeFname:  lang.CodeFilename,
			CompileCommand: lang.CompileCmd,
			CompiledFname:  lang.CompiledFilename,
			ExecCommand:    lang.ExecuteCmd,
		},
		CreatedAtRfc3339: createdAt.Format(time.RFC3339),
		Version:          1,
	}
	item, err := attributevalue.MarshalMap(evalDetailsRow)
	if err != nil {
		log.Printf(ctx, "error marshalling eval details row: %+v", err)
		return nil, submgen.InternalError("error marshalling eval details row")
	}
	_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &s.submTableName, Item: item})
	if err != nil {
		log.Printf(ctx, "error saving submission evaluation details: %+v", err)
		return nil, submgen.InternalError("error saving submission evaluation details")
	}

	// TODO: subtask, testgroup information
	// TODO: eval_uuid as field

	// PUT EVALUATION TEST ROWS
	evaluationTestRows := make([]*SubmissionEvaluationTestRow, 0)
	for _, test := range task.Tests {
		evaluationTestRows = append(evaluationTestRows, &SubmissionEvaluationTestRow{
			SubmUuid:               submUuid.String(),
			SortKey:                fmt.Sprintf("eval#%s#test#%04d", evalUuid.String(), test.TestID),
			FullInputS3Uri:         test.FullInputS3URI,
			FullAnswerS3Uri:        test.FullAnswerS3URI,
			Reached:                false,
			Ignored:                false,
			Finished:               false,
			InputTrimmed:           nil,
			AnswerTrimmed:          nil,
			CheckerStdout:          nil,
			CheckerStderr:          nil,
			CheckerExitCode:        nil,
			CheckerCpuTimeMillis:   nil,
			CheckerWallTimeMillis:  nil,
			CheckerMemoryKibiBytes: nil,
			SubmStdout:             nil,
			SubmStderr:             nil,
			SubmExitCode:           nil,
			SubmCpuTimeMillis:      nil,
			SubmWallTimeMillis:     nil,
			SubmMemoryKibiBytes:    nil,
			Subtasks:               test.Subtasks,
			TestGroup:              test.TestGroup,
		})
	}
	batchSize := 25
	start := 0
	for start < len(evaluationTestRows) {
		end := min(start+batchSize, len(evaluationTestRows))
		batch := evaluationTestRows[start:end]
		items := make([]types.WriteRequest, len(batch))
		for i := range len(batch) {
			item, err := attributevalue.MarshalMap(batch[i])
			if err != nil {
				log.Printf(ctx, "error marshalling evaluation test row: %+v", err)
				return nil, submgen.InternalError("error marshalling evaluation test row")
			}
			items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
		}
		_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
		})
		if err != nil {
			log.Printf(ctx, "error saving submission evaluation tests: %+v", err)
			return nil, submgen.InternalError("error saving submission evaluation tests")
		}
		start = end
	}
	// TODO: subtask, testgroup information in evaluation
	// TODO: eval_uuid as field

	// PUT SUBMISSION DETAILS ROW
	submDetailsRow := &SubmissionDetailsRow{
		SubmUuid:          submUuid.String(),
		SortKey:           "subm#details",
		Content:           *subm,
		AuthorUuid:        user.UUID,
		TaskUuid:          task.PublishedTaskID,
		ProgLangId:        lang.ID,
		CurrentEvalUuid:   evalUuid.String(),
		CurrentEvalStatus: "waiting",
		Gsi1Pk:            1,
		CreatedAtRfc3339:  createdAt.UTC().Format(time.RFC3339),
		Version:           1,
	}
	item, err = attributevalue.MarshalMap(submDetailsRow)
	if err != nil {
		log.Printf(ctx, "error marshalling submission details row: %+v", err)
		return nil, submgen.InternalError("error marshalling submission details row")
	}
	_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &s.submTableName, Item: item})
	if err != nil {
		log.Printf(ctx, "error saving submission details: %+v", err)
		return nil, submgen.InternalError("error saving submission details")
	}

	// ENQUEUE EVALUATION REQUEST TO SQS
	var tests []Test = make([]Test, 0)
	for _, test := range task.Tests {
		tests = append(tests, Test{
			ID:            test.TestID,
			InputSha256:   test.InputSha256,
			InputS3URI:    test.FullInputS3URI,
			InputContent:  nil,
			InputHttpURL:  nil,
			AnswerSha256:  test.AnswerSha256,
			AnswerS3URI:   test.FullAnswerS3URI,
			AnswerContent: nil,
			AnswerHttpURL: nil,
		})
	}
	reqWithUuid := EvalReqWithUuid{
		EvaluationUuid: evalUuid.String(),
		Request: EvalRequest{
			Submission: *subm,
			Language: LanguageDetails{
				ID:               lang.ID,
				Name:             lang.FullName,
				CodeFilename:     lang.CodeFilename,
				CompileCmd:       lang.CompileCmd,
				CompiledFilename: lang.CompiledFilename,
				ExecCmd:          lang.ExecuteCmd,
			},
			Limits: LimitsDetails{
				CPUTimeMillis:   int(task.CPUTimeLimitSeconds * 1000),
				MemoryKibibytes: int(float64(task.MemoryLimitMegabytes) * 976.5625),
			},
			Tests:          tests,
			TestlibChecker: task.TestlibCheckerCode,
		},
	}
	jsonReq, err := json.Marshal(reqWithUuid)
	if err != nil {
		log.Printf(ctx, "error marshalling eval request: %+v", err)
		return nil, submgen.InternalError("error marshalling eval request")
	}
	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.submQueueUrl),
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		fmt.Printf("failed to send message %s, %v\n", submDetailsRow.SubmUuid, err)
		return nil, submgen.InternalError("error sending message to evaluation queue")
	}

	// RETURN SUBMISSION TO USER
	res := &submgen.Submission{
		UUID:       submDetailsRow.SubmUuid,
		Submission: submDetailsRow.Content,
		Username:   user.Username,
		CreatedAt:  createdAt.Format(time.RFC3339),
		Evaluation: nil,
		Language: &submgen.SubmProgrammingLang{
			ID:       lang.ID,
			FullName: lang.FullName,
			MonacoID: lang.MonacoId,
		},
		Task: &submgen.SubmTask{
			Name: task.TaskFullName,
			Code: task.PublishedTaskID,
		},
	}

	return res, nil
}

func determineScoringMethod(task *taskgen.TaskSubmEvalData) string {
	if len(task.SubtaskScores) > 0 {
		return "subtask"
	}
	if len(task.TestGroupInformation) > 0 {
		return "testgroup"
	}
	return "tests"
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

	userByUsername, err := s.userSrvc.GetUserByUsername(ctx, &usergen.GetUserByUsernamePayload{Username: p.Username})
	if err != nil {
		log.Errorf(ctx, err, "error getting user: %+v", err.Error())
		if e, ok := err.(usergen.NotFound); ok {
			return nil, submgen.InvalidSubmissionDetails(string(e))
		}
		return nil, submgen.InternalError("error getting user")
	}

	claims := ctx.Value(ClaimsKey("claims")).(*auth.Claims)
	log.Printf(ctx, "%+v", claims)

	if claims.UUID != userByUsername.UUID {
		return nil, submgen.Unauthorized("jwt claims uuid does not match username's user's uuid")
	}

	taskEvalData, err := s.taskSrvc.GetTaskSubmEvalData(ctx, &taskgen.GetTaskSubmEvalDataPayload{
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

	return s.createSubmissionWithValidatedInput(ctx, &submContent.Value, userByUsername, taskEvalData, foundPLang)
}
