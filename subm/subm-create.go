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
	err = s.ddbSubmTable.BatchSaveEvaluationTestRows(ctx, evaluationTestRows)
	if err != nil {
		log.Printf(ctx, "error saving submission evaluation test rows: %+v", err)
		return nil, submgen.InternalError("error saving submission evaluation test rows")
	}

	// TODO: extract the scoring logic into seperated section of the code
	var scores []ScoreGroup = make([]ScoreGroup, 0)

	// if it has subtasks, then those are the groups
	// otherwise it's just one big group

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
			if _, ok := subtaskToTests[subtask]; !ok {
				subtaskToTests[subtask] = make([]*taskgen.TaskEvalTestInformation, 0)
			}
			subtaskToTests[subtask] = append(subtaskToTests[subtask], test)
		}
	}

	if len(subtaskToTests) > 0 {
		// it follows that groups are subtasks

		// possible score is calculated as:
		// if the subtask has a score specified then that
		// else if there are testgroups then the sum of their scores
		// else the no of tests that belong to this subtask

		possible := 0
		for subtask, subtaskTests := range subtaskToTests {
			foundSubtaskScore := false
			for _, subtaskScore := range taskEvalData.SubtaskScores {
				if subtaskScore.SubtaskID == subtask {
					possible = subtaskScore.Score
					foundSubtaskScore = true
					break
				}
			}
			if foundSubtaskScore {
				scores = append(scores, ScoreGroup{
					Received: 0,
					Possible: possible,
					Finished: false,
				})
				continue
			}
			foundTestGroup := false
			for _, testGroupInfos := range taskEvalData.TestGroupInformation {
				if testGroupInfos.Subtask == subtask {
					possible += testGroupInfos.Score
					foundTestGroup = true
				}
			}
			if foundTestGroup {
				scores = append(scores, ScoreGroup{
					Received: 0,
					Possible: possible,
					Finished: false,
				})
				continue
			}
			scores = append(scores, ScoreGroup{
				Received: 0,
				Possible: len(subtaskTests),
				Finished: false,
			})
		}
	} else {
		// it follows that all tests are in one group
		// testgroups can't exist without subtasks
		possible := len(taskEvalData.Tests)
		scores = append(scores, ScoreGroup{
			Received: 0,
			Possible: possible,
			Finished: false,
		})
	}

	submDetailsRow := &SubmissionDetailsRow{
		SubmUuid:   submUuid.String(),
		SortKey:    "details",
		Content:    submContent.String(),
		AuthorUuid: userByUsername.UUID,
		TaskUuid:   p.TaskCodeID,
		ProgLangId: foundPLang.ID,
		EvalResult: &SubmDetailsRowEvaluation{
			EvalUuid: evalUuid.String(),
			Status:   "waiting",
			Scores:   scores,
		},
		Gsi1Pk:           1,
		CreatedAtRfc3339: createdAt.UTC().Format(time.RFC3339),
		Version:          0,
	}
	err = s.ddbSubmTable.SaveSubmissionDetails(ctx, submDetailsRow)
	if err != nil {
		log.Printf(ctx, "error saving submission details: %+v", err)
		return nil, submgen.InternalError("error saving submission details")
	}

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
