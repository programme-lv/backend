package subm

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/auth"
	submgen "github.com/programme-lv/backend/gen/submissions"
	taskgen "github.com/programme-lv/backend/gen/tasks"
	usergen "github.com/programme-lv/backend/gen/users"
	"goa.design/clue/log"
)

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

	createdAt := time.Now()
	submUuid := uuid.New()
	evalUuid := uuid.New()

	evaluationDetailsRow := &SubmissionEvaluationDetailsRow{
		SubmUuid:           submUuid.String(),
		SortKey:            fmt.Sprintf("evaluation#%s", evalUuid.String()),
		EvaluationStage:    "waiting",
		TestlibCheckerCode: taskEvalData.TestlibCheckerCode,
		SystemInformation:  nil,
		SubmCompileData:    nil,
		ProgrammingLang: SubmEvalDetailsProgrammingLang{
			PLangId:        foundPLang.ID,
			DisplayName:    foundPLang.FullName,
			SubmCodeFname:  foundPLang.CodeFilename,
			CompileCommand: foundPLang.CompileCmd,
			CompiledFname:  foundPLang.CompiledFilename,
			ExecCommand:    foundPLang.ExecuteCmd,
		},
		CreatedAtRfc3339: createdAt.UTC().Format(time.RFC3339),
		Version:          0,
	}
	err = s.ddbSubmTable.SaveSubmissionEvaluationDetails(ctx, evaluationDetailsRow)
	if err != nil {
		return nil, submgen.InternalError("error saving submission evaluation details")
	}

	evaluationTestRows := make([]*SubmissionEvaluationTestRow, 0)
	for _, test := range taskEvalData.Tests {
		evaluationTestRows = append(evaluationTestRows, &SubmissionEvaluationTestRow{
			SubmUuid:            submUuid.String(),
			SortKey:             fmt.Sprintf("evaluation#%s#test#%04d", evalUuid.String(), test.TestID),
			FullInputS3Uri:      test.FullInputS3URI,
			FullAnswerS3Uri:     test.FullAnswerS3URI,
			Reached:             false,
			Ignored:             false,
			Finished:            false,
			InputTrimmed:        nil,
			AnswerTrimmed:       nil,
			SubmTestRuntimeData: nil,
			CheckerRuntimeData:  nil,
			Subtasks:            test.Subtasks,
			TestGroup:           test.TestGroup,
		})
	}
	// TODO: batch save evaluation test rows

	var scores []ScoreGroup
	var scoreGrouopExample = ScoreGroup{
		Received: 0,
		Possible: 100,
		Finished: false,
	}
	// if it has subtasks, then those are the groups
	// else if it has test groups, then those are the groups
	// else it's just one group

	/*

		type TaskEvalTestInformation struct {
			// Test ID
			TestID int
			// Full input S3 URI
			FullInputS3URI string
			// Full answer S3 URI
			FullAnswerS3URI string
			// Subtasks that the test is part of
			Subtasks []int
			// Test group that the test is part of
			TestGroup *int
		}
	*/
	subtaskToTests := make(map[int][]*taskgen.TaskEvalTestInformation)
	for _, test := range taskEvalData.Tests {
		for _, subtask := range test.Subtasks {
			subtaskToTests[subtask] = append(subtaskToTests[subtask], test)
		}
	}

	if len(subtaskToTests) > 0 {
		// groups are subtasks
		// possible score is calculated as
		// if there are testgroups then by their points
		// else by the number of tests

	}

	// TODO: calculate scores

	submDetailsRow := &SubmissionDetailsRow{
		SubmUuid:   submUuid.String(),
		SortKey:    fmt.Sprintf("details"),
		Content:    submContent.String(),
		AuthorUuid: userByUsername.UUID,
		TaskUuid:   p.TaskCodeID,
		ProgLangId: foundPLang.ID,
		EvalResult: &SubmDetailsRowEvaluation{
			EvalUuid: evalUuid.String(),
			Status:   "waiting",
			Scores:   scores,
		},
		CreatedAtRfc3339: createdAt.UTC().Format(time.RFC3339),
		Version:          0,
	}
	err = s.ddbSubmTable.SaveSubmissionDetails(ctx, submDetailsRow)
	if err != nil {
		return nil, submgen.InternalError("error saving submission details")
	}

	// err = s.ddbSubmTable.Save(ctx, row)
	// if err != nil {
	// 	// TODO: automatically retry with exponential backoff on version conflict
	// 	if dynamo.IsCondCheckFailed(err) {
	// 		log.Errorf(ctx, err, "version conflict saving user")
	// 		return nil, submgen.InternalError("version conflict saving user")
	// 	} else {
	// 		log.Errorf(ctx, err, "error saving user")
	// 		return nil, submgen.InternalError("error saving user")
	// 	}
	// }

	// TODO: enqueue message to SQS
	messageBody := fmt.Sprintf("Message %s", submDetailsRow.SubmUuid)
	_, err = s.sqsClient.SendMessage(context.TODO(), &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.submQueueUrl),
		MessageBody: aws.String(messageBody),
	})
	if err != nil {
		fmt.Printf("failed to send message %s, %v\n", submDetailsRow.SubmUuid, err)
	}

	createdAtRfc3339 := createdAt.Format(time.RFC3339)

	res = &submgen.Submission{
		UUID:       submDetailsRow.SubmUuid,
		Submission: submDetailsRow.Content,
		Username:   userByUsername.Username,
		CreatedAt:  createdAtRfc3339,
		Evaluation: nil,
		Language: &submgen.SubmProgrammingLang{
			ID:       foundPLang.ID,
			FullName: foundPLang.FullName,
			MonacoID: foundPLang.MonacoId,
		},
		Task: &submgen.SubmTask{
			Name: taskEvalData.TaskFullName,
			Code: taskEvalData.PublishedTaskID,
		},
	}

	return res, nil
}
