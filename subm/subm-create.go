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
	"github.com/programme-lv/backend/srvcerr"
	"github.com/programme-lv/backend/task"
	"github.com/programme-lv/backend/user"
	"goa.design/clue/log"
)

func (s *SubmissionSrvc) createSubmissionWithValidatedInput(
	subm *string,
	user *user.User,
	task *task.TaskSubmEvalData,
	lang *ProgrammingLang,
) (*Submission, error) {

	createdAt := time.Now().UTC()
	submUuid := uuid.New()
	evalUuid := uuid.New()

	var err error

	// DETERMINE WHETHER SCORING IS "tests", "subtask", OR "testgroup"
	scoringMethod := determineScoringMethod(task)
	switch scoringMethod {
	case "tests":
		// PUT EVALUATION SCORING TESTS ROW
		evalScoringTestsRow := &EvalScoringTestsRow{
			SubmUuid: submUuid.String(),
			SortKey:  fmt.Sprintf("eval#%s#scoring#tests", evalUuid.String()),
			Accepted: 0,
			Wrong:    0,
			Untested: len(task.Tests),
			Version:  1,
		}
		item, err := attributevalue.MarshalMap(evalScoringTestsRow)
		if err != nil {
			return nil, fmt.Errorf("error marshalling eval scoring tests row: %w", err)
		}
		_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: &s.submTableName, Item: item})
		if err != nil {
			return nil, fmt.Errorf("error saving evaluation scoring tests: %w", err)
		}

		// PUT SUBMISSION SCORING TESTS ROW
		submScoringTestsRow := &SubmScoringTestsRow{
			SubmUuid:        submUuid.String(),
			SortKey:         "subm#scoring#tests",
			Accepted:        0,
			Wrong:           0,
			Untested:        len(task.Tests),
			Gsi1Pk:          1,
			CurrentEvalUuid: evalUuid.String(),
			Version:         1,
			Gsi1SortKey:     fmt.Sprintf("%s#%s#scoring#tests", createdAt.Format(time.RFC3339), submUuid.String()),
		}
		item, err = attributevalue.MarshalMap(submScoringTestsRow)
		if err != nil {
			return nil, fmt.Errorf("error marshalling scoring tests row: %w", err)
		}
		_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: &s.submTableName, Item: item})
		if err != nil {
			return nil, fmt.Errorf("error saving submission scoring tests: %w", err)
		}
	case "subtask":
		// PUT EVALUATION, SUBMISSION SCORING SUBTASK ROWS
		evalScoringSubtaskRows := make([]*EvalScoringSubtaskRow, 0)
		submScoringSubtaskRows := make([]*SubmScoringSubtaskRow, 0)
		for _, subtask := range task.SubtaskScores {
			stTestCount := 0
			for _, test := range task.Tests {
				for _, testSt := range test.Subtasks {
					if testSt == subtask.SubtaskID {
						stTestCount++
					}
				}
			}
			evalRow := &EvalScoringSubtaskRow{
				SubmUuid:      submUuid.String(),
				SortKey:       fmt.Sprintf("eval#%s#scoring#subtask#%02d", evalUuid.String(), subtask.SubtaskID),
				SubtaskScore:  subtask.Score,
				AcceptedTests: 0,
				WrongTests:    0,
				UntestedTests: stTestCount,
				Version:       1,
			}
			evalScoringSubtaskRows = append(evalScoringSubtaskRows, evalRow)

			submRow := &SubmScoringSubtaskRow{
				SubmUuid:        submUuid.String(),
				SortKey:         fmt.Sprintf("subm#scoring#subtask#%02d", subtask.SubtaskID),
				SubtaskScore:    subtask.Score,
				AcceptedTests:   0,
				WrongTests:      0,
				CurrentEvalUuid: evalUuid.String(),
				Version:         1,
				UntestedTests:   stTestCount,
				Gsi1Pk:          1,
				Gsi1SortKey:     fmt.Sprintf("%s#%s#scoring#subtask#%02d", createdAt.Format(time.RFC3339), submUuid.String(), subtask.SubtaskID),
			}
			submScoringSubtaskRows = append(submScoringSubtaskRows, submRow)
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
					return nil, fmt.Errorf("error marshalling scoring subtask row: %w", err)
				}
				items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
			}
			_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
			})
			if err != nil {
				return nil, fmt.Errorf("error saving submission scoring subtasks: %w", err)
			}
			start = end
		}
		batchSize = 25
		start = 0
		for start < len(evalScoringSubtaskRows) {
			end := min(start+batchSize, len(evalScoringSubtaskRows))
			batch := evalScoringSubtaskRows[start:end]
			items := make([]types.WriteRequest, len(batch))
			for i := range len(batch) {
				item, err := attributevalue.MarshalMap(batch[i])
				if err != nil {
					return nil, fmt.Errorf("error marshalling eval scoring subtask row: %w", err)
				}
				items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
			}
			_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
			})
			if err != nil {
				return nil, fmt.Errorf("error saving evaluation scoring subtasks: %w", err)
			}
			start = end
		}
	case "testgroup":
		// PUT EVALUATION, SUBMISSION SCORING TESTGROUP ROWS
		evalScoringTestgroupRows := make([]*EvalScoringTestgroupRow, 0)
		submScoringTestgroupRows := make([]*SubmScoringTestgroupRow, 0)
		for _, testGroup := range task.TestGroupInformation {
			tgTests := 0
			for _, test := range task.Tests {
				if test.TestGroup != nil && *test.TestGroup == testGroup.TestGroupID {
					tgTests++
				}
			}
			evalRow := &EvalScoringTestgroupRow{
				SubmUuid:         submUuid.String(),
				SortKey:          fmt.Sprintf("eval#%s#scoring#testgroup#%02d", evalUuid.String(), testGroup.TestGroupID),
				StatementSubtask: testGroup.Subtask,
				TestgroupScore:   testGroup.Score,
				AcceptedTests:    0,
				WrongTests:       0,
				UntestedTests:    tgTests,
				Version:          1,
			}
			evalScoringTestgroupRows = append(evalScoringTestgroupRows, evalRow)

			submRow := &SubmScoringTestgroupRow{
				SubmUuid:         submUuid.String(),
				SortKey:          fmt.Sprintf("subm#scoring#testgroup#%02d", testGroup.TestGroupID),
				StatementSubtask: testGroup.Subtask,
				TestgroupScore:   testGroup.Score,
				AcceptedTests:    0,
				WrongTests:       0,
				CurrentEvalUuid:  evalUuid.String(),
				Version:          1,
				UntestedTests:    tgTests,
				Gsi1Pk:           1,
				Gsi1SortKey:      fmt.Sprintf("%s#%s#scoring#testgroup#%02d", createdAt.Format(time.RFC3339), submUuid.String(), testGroup.TestGroupID),
			}

			submScoringTestgroupRows = append(submScoringTestgroupRows, submRow)
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
					return nil, fmt.Errorf("error marshalling scoring testgroup row: %w", err)
				}
				items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
			}
			_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
			})
			if err != nil {
				return nil, fmt.Errorf("error saving submission scoring testgroups: %w", err)
			}
			start = end
		}
		batchSize = 25
		start = 0
		for start < len(evalScoringTestgroupRows) {
			end := min(start+batchSize, len(evalScoringTestgroupRows))
			batch := evalScoringTestgroupRows[start:end]
			items := make([]types.WriteRequest, len(batch))
			for i := range len(batch) {
				item, err := attributevalue.MarshalMap(batch[i])
				if err != nil {
					return nil, fmt.Errorf("error marshalling eval scoring testgroup row: %w", err)
				}
				items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
			}
			_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
			})
			if err != nil {
				return nil, fmt.Errorf("error saving evaluation scoring testgroups: %w", err)
			}
			start = end
		}
	}

	// PUT EVALUATION DETAILS ROW
	evalDetailsRow := &EvalDetailsRow{
		SubmUuid:                   submUuid.String(),
		SortKey:                    fmt.Sprintf("eval#%s#details", evalUuid.String()),
		EvalUuid:                   evalUuid.String(),
		EvaluationStage:            "waiting",
		TestlibCheckerCode:         task.TestlibCheckerCode,
		SystemInformation:          nil,
		SubmCompileStdout:          nil,
		SubmCompileStderr:          nil,
		SubmCompileExitCode:        nil,
		SubmCompileCpuTimeMillis:   nil,
		SubmCompileWallTimeMillis:  nil,
		SubmCompileMemoryKibiBytes: nil,
		ProgrammingLang: EvalDetailsProgrammingLang{
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
		return nil, fmt.Errorf("error marshalling eval details row: %w", err)
	}
	_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &s.submTableName, Item: item})
	if err != nil {
		return nil, fmt.Errorf("error saving evaluation details: %w", err)
	}

	// PUT EVALUATION TEST ROWS
	evaluationTestRows := make([]*EvalTestRow, 0)
	for _, test := range task.Tests {
		evaluationTestRows = append(evaluationTestRows, &EvalTestRow{
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
				return nil, fmt.Errorf("error marshalling evaluation test row: %w", err)
			}
			items[i] = types.WriteRequest{PutRequest: &types.PutRequest{Item: item}}
		}
		_, err = s.ddbClient.BatchWriteItem(context.TODO(), &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{s.submTableName: items},
		})
		if err != nil {
			return nil, fmt.Errorf("error saving evaluation tests: %w", err)
		}
		start = end
	}

	// PUT SUBMISSION DETAILS ROW
	submDetailsRow := &SubmDetailsRow{
		SubmUuid:          submUuid.String(),
		SortKey:           "subm#details",
		Content:           *subm,
		AuthorUuid:        user.UUID,
		TaskId:            task.PublishedTaskID,
		ProgLangId:        lang.ID,
		CurrentEvalUuid:   evalUuid.String(),
		CurrentEvalStatus: "waiting",
		Gsi1Pk:            1,
		Gsi1SortKey:       fmt.Sprintf("%s#%s#details", createdAt.Format(time.RFC3339), submUuid.String()),
		CreatedAtRfc3339:  createdAt.UTC().Format(time.RFC3339),
		Version:           1,
	}
	item, err = attributevalue.MarshalMap(submDetailsRow)
	if err != nil {
		return nil, fmt.Errorf("error marshalling submission details row: %w", err)
	}
	_, err = s.ddbClient.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &s.submTableName, Item: item})
	if err != nil {
		return nil, fmt.Errorf("error saving submission details: %w", err)
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
		return nil, fmt.Errorf("error marshalling eval request: %w", err)
	}
	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.submQueueUrl),
		MessageBody: aws.String(string(jsonReq)),
	})
	if err != nil {
		return nil, fmt.Errorf("error sending message to evaluation queue: %w", err)
	}

	var evalScoringTestgroups []*TestGroupResult = nil
	if scoringMethod == "testgroup" {
		evalScoringTestgroups = make([]*TestGroupResult, 0)
		for _, testGroup := range task.TestGroupInformation {
			tgTests := 0
			for _, test := range task.Tests {
				if test.TestGroup != nil && *test.TestGroup == testGroup.TestGroupID {
					tgTests++
				}
			}
			evalScoringTestgroups = append(evalScoringTestgroups, &TestGroupResult{
				TestGroupID:      testGroup.TestGroupID,
				TestGroupScore:   testGroup.Score,
				StatementSubtask: testGroup.Subtask,
				AcceptedTests:    0,
				WrongTests:       0,
				UntestedTests:    tgTests,
			})
		}
	}
	var evalScoringTests *TestsResult = nil
	if scoringMethod == "tests" {
		evalScoringTests = &TestsResult{
			Accepted: 0,
			Wrong:    0,
			Untested: len(task.Tests),
		}
	}
	var evalScoringSubtasks []*SubtaskResult = nil
	if scoringMethod == "subtask" {
		evalScoringSubtasks = make([]*SubtaskResult, 0)
		for _, subtask := range task.SubtaskScores {
			stTestCount := 0
			for _, test := range task.Tests {
				for _, testSt := range test.Subtasks {
					if testSt == subtask.SubtaskID {
						stTestCount++
					}
				}
			}
			evalScoringSubtasks = append(evalScoringSubtasks, &SubtaskResult{
				SubtaskID:     subtask.SubtaskID,
				SubtaskScore:  subtask.Score,
				AcceptedTests: 0,
				WrongTests:    0,
				UntestedTests: stTestCount,
			})
		}
	}

	res := &Submission{
		SubmUUID:              submDetailsRow.SubmUuid,
		Submission:            submDetailsRow.Content,
		Username:              user.Username,
		CreatedAt:             createdAt.Format(time.RFC3339),
		EvalStatus:            "waiting",
		EvalScoringTestgroups: evalScoringTestgroups,
		EvalScoringTests:      evalScoringTests,
		EvalScoringSubtasks:   evalScoringSubtasks,
		PLangID:               lang.ID,
		PLangDisplayName:      lang.FullName,
		PLangMonacoID:         lang.MonacoId,
		TaskName:              task.TaskFullName,
		TaskID:                task.PublishedTaskID,
	}

	s.createdSubmChan <- res

	s.evalUuidToSubmUuid[evalUuid.String()] = submUuid.String()

	return res, nil
}

func determineScoringMethod(task *task.TaskSubmEvalData) string {
	if len(task.SubtaskScores) > 0 {
		return "subtask"
	}
	if len(task.TestGroupInformation) > 0 {
		return "testgroup"
	}
	return "tests"
}

// CreateSubmission implements submissions.Service.
func (s *SubmissionSrvc) CreateSubmission(ctx context.Context, p *CreateSubmissionPayload) (res *Submission, err error) {
	submContent := SubmissionContent{Value: p.Submission}

	for _, v := range []Validatable{&submContent} {
		err := v.IsValid()
		if err != nil {
			return nil, err
		}
	}

	userByUsername, err := s.userSrvc.GetUserByUsername(ctx, &user.GetUserByUsernamePayload{Username: p.Username})
	if err != nil {
		log.Errorf(ctx, err, "error getting user: %+v", err.Error())
		if e, ok := err.(*srvcerr.Error); ok && e.ErrorCode() == user.ErrCodeUserNotFound {
			return nil, newErrUserNotFound()
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}

	claims, ok := ctx.Value(auth.CtxJwtClaimsKey).(*auth.JwtClaims)
	if !ok {
		return nil, newErrJwtTokenMissing()
	}

	if claims == nil {
		return nil, newErrJwtTokenMissing()
	}

	log.Printf(ctx, "%+v", claims)

	if claims.UUID != userByUsername.UUID {
		return nil, newErrUnauthorizedUsernameMismatch()
	}

	taskEvalData, err := s.taskSrvc.GetTaskSubmEvalData(ctx, &task.GetTaskSubmEvalDataPayload{
		TaskID: p.TaskCodeID,
	})
	if err != nil {
		if e, ok := err.(*task.Error); ok && e.ErrorCode() == task.ErrTaskNotFoundCode {
			return nil, newErrTaskNotFound()
		}
		return nil, fmt.Errorf("error getting task: %w", err)
	}

	langs := getHardcodedLanguageList()
	var foundPLang *ProgrammingLang = nil
	for _, lang := range langs {
		if lang.ID == p.ProgrammingLangID {
			foundPLang = &lang
		}
	}

	if foundPLang == nil {
		return nil, newErrInvalidProgLang()
	}

	return s.createSubmissionWithValidatedInput(&submContent.Value, userByUsername, taskEvalData, foundPLang)
}
