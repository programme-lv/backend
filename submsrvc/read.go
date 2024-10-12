package submsrvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
)

func (s *SubmissionSrvc) GetSubmission(ctx context.Context, submUuid string) (*FullSubmission, error) {
	panic("not implemented")
}

func (s *SubmissionSrvc) ListSubmissions(ctx context.Context) ([]*Submission, error) {
	// get all submissions ever
	selectSubmStmt := postgres.SELECT(table.Submissions.AllColumns, table.Evaluations.AllColumns).
		FROM(
			table.Submissions.
				INNER_JOIN(table.Evaluations, table.Submissions.CurrentEvalUUID.EQ(table.Evaluations.EvalUUID)).
				INNER_JOIN(table.EvaluationTestset, table.Evaluations.EvalUUID.EQ(table.EvaluationTestset.EvalUUID)),
		)

	type SubmJoinEvalModel struct {
		model.Submissions
		model.Evaluations
		model.EvaluationTestset
	}

	var submJoinEval []SubmJoinEvalModel
	if err := selectSubmStmt.QueryContext(ctx, s.postgres, &submJoinEval); err != nil {
		return nil, fmt.Errorf("failed to get submissions: %w", err)
	}

	selectSubtasks := postgres.SELECT(table.Submissions.SubmUUID, table.EvaluationSubtasks.AllColumns).
		FROM(table.Submissions.
			INNER_JOIN(table.EvaluationSubtasks, table.Submissions.CurrentEvalUUID.EQ(table.EvaluationSubtasks.EvalUUID)),
		)

	type SubmSubtaskModel struct {
		model.Submissions
		model.EvaluationSubtasks
	}

	var subtasks []SubmSubtaskModel
	if err := selectSubtasks.QueryContext(ctx, s.postgres, &subtasks); err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}

	submUUIDToSubtask := make(map[uuid.UUID][]SubmSubtaskModel)
	for _, subtask := range subtasks {
		submUUIDToSubtask[subtask.SubmUUID] = append(submUUIDToSubtask[subtask.SubmUUID], subtask)
	}

	selectTestGroups := postgres.SELECT(table.Submissions.SubmUUID, table.EvaluationTestgroups.AllColumns).
		FROM(table.Submissions.
			INNER_JOIN(table.EvaluationTestgroups, table.Submissions.CurrentEvalUUID.EQ(table.EvaluationTestgroups.EvalUUID)),
		)

	type SubmTestGroupModel struct {
		model.Submissions
		model.EvaluationTestgroups
	}

	var submTestGroups []SubmTestGroupModel
	if err := selectTestGroups.QueryContext(ctx, s.postgres, &submTestGroups); err != nil {
		return nil, fmt.Errorf("failed to get test groups: %w", err)
	}

	submUUIDToTestGroups := make(map[uuid.UUID][]SubmTestGroupModel)
	for _, testGroup := range submTestGroups {
		submUUIDToTestGroups[testGroup.SubmUUID] = append(submUUIDToTestGroups[testGroup.SubmUUID], testGroup)
	}

	authorUUIDs := make([]uuid.UUID, len(submJoinEval))
	for i, subm := range submJoinEval {
		authorUUIDs[i] = subm.Submissions.AuthorUUID
	}
	// request usernames for authors from user service
	usernames, err := s.userSrvc.GetUsernames(ctx, authorUUIDs)
	if err != nil {
		return nil, err
	}

	taskShortIDs := make([]string, len(submJoinEval))
	for i, subm := range submJoinEval {
		taskShortIDs[i] = subm.Submissions.TaskID
	}

	taskFullNames, err := s.taskSrvc.GetTaskFullNames(ctx, taskShortIDs)
	if err != nil {
		return nil, err
	}

	languages, err := s.ListProgrammingLanguages(ctx)
	if err != nil {
		return nil, err
	}
	langIDToFullName := make(map[string]string)
	for _, lang := range languages {
		langIDToFullName[lang.ID] = lang.FullName
	}
	langIDToMonacoID := make(map[string]string)
	for _, lang := range languages {
		langIDToMonacoID[lang.ID] = lang.MonacoId
	}

	submissions := make([]*Submission, len(submJoinEval))
	for i, subm := range submJoinEval {
		subtasks := make([]Subtask, 0)
		subtaskModels, hasSubtasks := submUUIDToSubtask[subm.Submissions.SubmUUID]
		if hasSubtasks {
			for _, subtask := range subtaskModels {
				description := ""
				if subtask.EvaluationSubtasks.Description != nil {
					description = *subtask.EvaluationSubtasks.Description
				}
				subtasks = append(subtasks, Subtask{
					SubtaskID:   int(subtask.EvaluationSubtasks.SubtaskID),
					Points:      int(subtask.EvaluationSubtasks.SubtaskPoints),
					Accepted:    int(subtask.EvaluationSubtasks.Accepted),
					Wrong:       int(subtask.EvaluationSubtasks.Wrong),
					Untested:    int(subtask.EvaluationSubtasks.Untested),
					Description: description,
				})
			}
		}
		testGroups := make([]TestGroup, 0)
		testGroupModels, hasTestGroups := submUUIDToTestGroups[subm.Submissions.SubmUUID]
		if hasTestGroups {
			for _, testGroup := range testGroupModels {
				subtaskArrayStrPtr := testGroup.EvaluationTestgroups.StatementSubtasks
				subtaskArray := []int{}
				if subtaskArrayStrPtr != nil {
					// {1,2,3}
					subtaskArrayStr := strings.Trim(*subtaskArrayStrPtr, "{}")
					subtaskArrayStrs := strings.Split(subtaskArrayStr, ",")
					for _, subtaskStr := range subtaskArrayStrs {
						subtask, err := strconv.Atoi(subtaskStr)
						if err != nil {
							return nil, fmt.Errorf("failed to convert subtask string to int: %w", err)
						}
						subtaskArray = append(subtaskArray, subtask)
					}
				}
				testGroups = append(testGroups, TestGroup{
					TestGroupID: int(testGroup.EvaluationTestgroups.TestgroupID),
					Points:      int(testGroup.EvaluationTestgroups.TestgroupPoints),
					Accepted:    int(testGroup.EvaluationTestgroups.Accepted),
					Wrong:       int(testGroup.EvaluationTestgroups.Wrong),
					Untested:    int(testGroup.EvaluationTestgroups.Untested),
					Subtasks:    subtaskArray,
				})
			}
		}
		submissions[i] = &Submission{
			UUID:    subm.Submissions.SubmUUID,
			Content: subm.Submissions.Content,
			Author: Author{
				UUID:     subm.Submissions.AuthorUUID,
				Username: usernames[i],
			},
			Task: Task{
				ShortID:  subm.Submissions.TaskID,
				FullName: taskFullNames[i],
			},
			Lang: Lang{
				ShortID:  subm.Evaluations.LangID,
				Display:  langIDToFullName[subm.Evaluations.LangID],
				MonacoID: langIDToMonacoID[subm.Evaluations.LangID],
			},
			CurrEval: Evaluation{
				UUID:       subm.Evaluations.EvalUUID,
				Stage:      subm.Evaluations.EvaluationStage,
				CreatedAt:  subm.Evaluations.CreatedAt,
				Subtasks:   subtasks,
				TestGroups: testGroups,
				TestSet: &TestSet{
					Accepted: int(subm.EvaluationTestset.Accepted),
					Wrong:    int(subm.EvaluationTestset.Wrong),
					Untested: int(subm.EvaluationTestset.Untested),
				},
			},
			CreatedAt: subm.Submissions.CreatedAt,
		}
	}

	return submissions, nil
}
