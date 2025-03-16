package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/subm/domain"
)

func mapSubmListEntry(
	ctx context.Context,
	s domain.Subm,
	getTaskName func(ctx context.Context, shortID string) (string, error),
	getUsername func(ctx context.Context, userUuid uuid.UUID) (string, error),
	getPrLang func(ctx context.Context, shortID string) (PrLang, error),
	getEval func(ctx context.Context, evalUuid uuid.UUID) (domain.Eval, error),
) (SubmListEntry, error) {

	username, err := getUsername(ctx, s.AuthorUUID)
	if err != nil {
		slog.Default().Warn("failed to get username when mapping subm list entry", "error", err, "subm_uuid", s.UUID, "author_uuid", s.AuthorUUID)
		return SubmListEntry{}, err
	}

	taskName, err := getTaskName(ctx, s.TaskShortID)
	if err != nil {
		slog.Default().Warn("failed to get task name when mapping subm list entry", "error", err, "subm_uuid", s.UUID, "task_short_id", s.TaskShortID)
		return SubmListEntry{}, err
	}

	prLang, err := getPrLang(ctx, s.LangShortID)
	if err != nil {
		slog.Default().Warn("failed to get pr lang when mapping subm list entry", "error", err, "subm_uuid", s.UUID, "lang_short_id", s.LangShortID)
		return SubmListEntry{}, err
	}

	eval, err := getEval(ctx, s.CurrEvalUUID)
	if err != nil {
		slog.Default().Warn("failed to get eval when mapping subm list entry", "error", err, "subm_uuid", s.UUID, "curr_eval_uuid", s.CurrEvalUUID)
		return SubmListEntry{}, err
	}

	scoreInfo := eval.CalculateScore()
	status := string(eval.Stage)
	if eval.Error != nil {
		if eval.Error.Type == domain.ErrorTypeCompilation {
			status = "compile_error"
		} else if eval.Error.Type == domain.ErrorTypeInternal {
			status = "internal_error"
		} else {
			status = string(eval.Error.Type)
		}
	}

	return SubmListEntry{
		SubmUuid:   s.UUID.String(),
		Username:   username,
		TaskId:     s.TaskShortID,
		TaskName:   taskName,
		PrLangId:   prLang.ShortID,
		PrLangName: prLang.Display,
		ScoreInfo: ScoreInfo{
			ScoreBar: struct {
				Green  int `json:"green"`
				Red    int `json:"red"`
				Gray   int `json:"gray"`
				Yellow int `json:"yellow"`
				Purple int `json:"purple"`
			}{
				Green:  scoreInfo.ScoreBar.Green,
				Red:    scoreInfo.ScoreBar.Red,
				Gray:   scoreInfo.ScoreBar.Gray,
				Yellow: scoreInfo.ScoreBar.Yellow,
				Purple: scoreInfo.ScoreBar.Purple,
			},
			ReceivedScore: scoreInfo.ReceivedScore,
			PossibleScore: scoreInfo.PossibleScore,
			MaxCpuMs:      scoreInfo.MaxCpuMs,
			MaxMemKiB:     scoreInfo.MaxMemKiB,
			ExceededCpu:   scoreInfo.ExceededCpu,
			ExceededMem:   scoreInfo.ExceededMem,
		},
		Status:    status,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
	}, nil
}

func mapSubm(
	ctx context.Context,
	subm domain.Subm,
	getTaskName func(ctx context.Context, shortID string) (string, error),
	getUsername func(ctx context.Context, userUuid uuid.UUID) (string, error),
	getPrLang func(ctx context.Context, shortID string) (PrLang, error),
	getEval func(ctx context.Context, evalUuid uuid.UUID) (domain.Eval, error),
) (*DetailedSubmView, error) {
	taskName, err := getTaskName(ctx, subm.TaskShortID)
	if err != nil {
		return nil, err
	}
	username, err := getUsername(ctx, subm.AuthorUUID)
	if err != nil {
		return nil, err
	}
	prLang, err := getPrLang(ctx, subm.LangShortID)
	if err != nil {
		return nil, err
	}
	var currEval *Eval
	if subm.CurrEvalUUID != uuid.Nil {
		eval, err := getEval(ctx, subm.CurrEvalUUID)
		if err != nil {
			return nil, err
		}
		mapped := mapSubmEval(eval)
		currEval = &mapped
	}
	return &DetailedSubmView{
		SubmUUID:  subm.UUID.String(),
		Content:   subm.Content,
		Username:  username,
		CurrEval:  currEval,
		PrLang:    prLang,
		TaskName:  taskName,
		TaskID:    subm.TaskShortID,
		CreatedAt: subm.CreatedAt.Format(time.RFC3339),
	}, nil

}

func mapSubmEval(eval domain.Eval) Eval {
	errType := ""
	if eval.Error != nil {
		errType = string(eval.Error.Type)
	}

	subtasks := []Subtask{}
	for _, subtask := range eval.Subtasks {
		testRanges := [][]int{}
		if len(subtask.StTests) > 0 {
			start := subtask.StTests[0]
			prev := start
			for i := 1; i < len(subtask.StTests); i++ {
				if subtask.StTests[i] != prev+1 {
					testRanges = append(testRanges, []int{start, prev})
					start = subtask.StTests[i]
				}
				prev = subtask.StTests[i]
			}
			testRanges = append(testRanges, []int{start, prev})
		}
		subtasks = append(subtasks, Subtask{
			Points:      subtask.Points,
			Description: subtask.Description,
			StTests:     testRanges,
		})
	}

	testGroups := []TestGroup{}
	for _, testGroup := range eval.Groups {
		testRanges := [][]int{}
		if len(testGroup.TgTests) > 0 {
			start := testGroup.TgTests[0]
			prev := start
			for i := 1; i < len(testGroup.TgTests); i++ {
				if testGroup.TgTests[i] != prev+1 {
					testRanges = append(testRanges, []int{start, prev})
					start = testGroup.TgTests[i]
				}
				prev = testGroup.TgTests[i]
			}
			testRanges = append(testRanges, []int{start, prev})
		}
		testGroups = append(testGroups, TestGroup{
			Points:   testGroup.Points,
			Subtasks: testGroup.Subtasks,
			TgTests:  testRanges,
		})
	}

	verdicts := ""
	for _, test := range eval.Tests {
		if test.Finished {
			if test.Ac {
				verdicts += "A" // accepted
			} else if test.Wa {
				verdicts += "W" // wrong answer
			} else if test.Tle {
				verdicts += "T" // time limit exceeded
			} else if test.Mle {
				verdicts += "M" // memory limit exceeded
			} else if test.Re {
				verdicts += "R" // runtime error
			} else if test.Ig {
				verdicts += "I" // ignored
			} else {
				verdicts += "U" // unknown
			}
		} else if test.Reached {
			verdicts += "T" // testing
		} else {
			verdicts += "Q" // queued
		}
	}

	scoreInfo := eval.CalculateScore()

	return Eval{
		EvalUUID:   eval.UUID.String(),
		SubmUUID:   eval.SubmUUID.String(),
		EvalStage:  string(eval.Stage),
		ScoreUnit:  string(eval.ScoreUnit),
		EvalError:  errType,
		Subtasks:   subtasks,
		TestGroups: testGroups,
		Verdicts:   verdicts,
		ScoreInfo: ScoreInfo{
			ScoreBar: struct {
				Green  int `json:"green"`
				Red    int `json:"red"`
				Gray   int `json:"gray"`
				Yellow int `json:"yellow"`
				Purple int `json:"purple"`
			}{
				Green:  scoreInfo.ScoreBar.Green,
				Red:    scoreInfo.ScoreBar.Red,
				Gray:   scoreInfo.ScoreBar.Gray,
				Yellow: scoreInfo.ScoreBar.Yellow,
				Purple: scoreInfo.ScoreBar.Purple,
			},
			ReceivedScore: scoreInfo.ReceivedScore,
			PossibleScore: scoreInfo.PossibleScore,
			MaxCpuMs:      scoreInfo.MaxCpuMs,
			MaxMemKiB:     scoreInfo.MaxMemKiB,
			ExceededCpu:   scoreInfo.ExceededCpu,
			ExceededMem:   scoreInfo.ExceededMem,
		},
	}
}

func (h *SubmHttpHandler) mapMaxScore(ctx context.Context, taskShortID string, m domain.MaxScore) (MaxScore, error) {
	taskFullNames, err := h.taskSrvc.GetTaskFullNames(ctx, []string{taskShortID})
	if err != nil {
		return MaxScore{}, err
	}
	taskFullName := taskFullNames[0]
	if taskFullName == "" {
		taskFullName = taskShortID
	}
	return MaxScore{
		SubmUuid:     m.SubmUuid.String(),
		Received:     m.Received,
		Possible:     m.Possible,
		CreatedAt:    m.CreatedAt.Format(time.RFC3339),
		TaskFullName: taskFullName,
	}, nil
}
