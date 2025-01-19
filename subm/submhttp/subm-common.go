package submhttp

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/subm"
	"github.com/programme-lv/backend/subm/submqueries"
)

type Subm struct {
	SubmUUID  string `json:"subm_uuid"`
	Content   string `json:"content"`
	Username  string `json:"username"`
	CurrEval  *Eval  `json:"curr_eval"`
	PrLang    PrLang `json:"pr_lang"`
	TaskName  string `json:"task_name"`
	TaskID    string `json:"task_id"`
	CreatedAt string `json:"created_at"`
}

type PrLang struct {
	ShortID  string `json:"short_id"`
	Display  string `json:"display"`
	MonacoID string `json:"monaco_id"`
}

type Eval struct {
	EvalUUID   string      `json:"eval_uuid"`
	EvalStage  string      `json:"eval_stage"`
	ScoreUnit  string      `json:"score_unit"`
	EvalError  string      `json:"eval_error"`
	ErrorMsg   string      `json:"error_msg"`
	Subtasks   []Subtask   `json:"subtasks"`
	TestGroups []TestGroup `json:"test_groups"`
	Verdicts   string      `json:"verdicts"` // q,ac,wa,tle,mle,re,ig -> "QAWTMRI"
}

type Subtask struct {
	Points      int    `json:"points"`
	Description string `json:"description"`
	// StTests     []int  `json:"st_tests"`
	StTests [][]int `json:"st_tests"`
}

type TestGroup struct {
	Points   int   `json:"points"`
	Subtasks []int `json:"subtasks"`
	// TgTests  []int `json:"tg_tests"`
	TgTests [][]int `json:"tg_tests"`
}

// func mapSubm(subm subm.SubmView) *Submission {
// 	var currEval *SubmEval
// 	if subm.CurrEval != nil {
// 		mapped := mapSubmEval(*subm.CurrEval)
// 		currEval = &mapped
// 	}
// 	return &Submission{
// 		SubmUUID: subm.UUID.String(),
// 		Content:  subm.Content,
// 		Username: subm.Author.Username,
// 		CurrEval: currEval,
// 		PrLang: PrLang{
// 			ShortID:  subm.Lang.ShortID,
// 			Display:  subm.Lang.Display,
// 			MonacoID: subm.Lang.MonacoID,
// 		},
// 		TaskName:  subm.Task.FullName,
// 		TaskID:    subm.Task.ShortID,
// 		CreatedAt: subm.CreatedAt.Format(time.RFC3339),
// 	}
// }

func (h SubmHttpServer) mapSubm(
	ctx context.Context,
	s subm.Subm,
) (*Subm, error) {
	return mapSubm(
		ctx,
		s,
		func(ctx context.Context, shortID string) (string, error) {
			task, err := h.taskSrvc.GetTask(ctx, shortID)
			if err != nil {
				return "", err
			}
			return task.FullName, nil
		},
		func(ctx context.Context, userUuid uuid.UUID) (string, error) {
			user, err := h.userSrvc.GetUserByUUID(ctx, userUuid)
			if err != nil {
				return "", err
			}
			return user.Username, nil
		},
		func(ctx context.Context, shortID string) (PrLang, error) {
			plang, err := planglist.GetProgrLangById(shortID)
			if err != nil {
				return PrLang{}, err
			}
			return PrLang{
				ShortID:  plang.ID,
				Display:  plang.FullName,
				MonacoID: plang.MonacoId,
			}, nil
		},
		func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error) {
			return h.submSrvc.GetEvalQuery.Handle(ctx, submqueries.GetEvalParams{
				EvalUUID: evalUuid,
			})
		},
	)
}

func mapSubm(
	ctx context.Context,
	subm subm.Subm,
	getTaskName func(ctx context.Context, shortID string) (string, error),
	getUsername func(ctx context.Context, userUuid uuid.UUID) (string, error),
	getPrLang func(ctx context.Context, shortID string) (PrLang, error),
	getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error),
) (*Subm, error) {
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
	return &Subm{
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

func mapSubmEval(eval subm.Eval) Eval {
	errType := ""
	errMsg := ""
	if eval.Error != nil {
		errType = string(eval.Error.Type)
		if eval.Error.Message != nil {
			errMsg = *eval.Error.Message
		}
	}

	// subtasks := []Subtask{}
	// for _, subtask := range eval.Subtasks {
	// 	subtasks = append(subtasks, Subtask{
	// 		Points:      subtask.Points,
	// 		Description: subtask.Description,
	// 		StTests:     subtask.StTests,
	// 	})
	// }

	// testGroups := []TestGroup{}
	// for _, testGroup := range eval.Groups {
	// 	testGroups = append(testGroups, TestGroup{
	// 		Points:   testGroup.Points,
	// 		Subtasks: testGroup.Subtasks,
	// 		TgTests:  testGroup.TgTests,
	// 	})
	// }

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

	return Eval{
		EvalUUID:   eval.UUID.String(),
		EvalStage:  string(eval.Stage),
		ScoreUnit:  string(eval.ScoreUnit),
		EvalError:  errType,
		ErrorMsg:   errMsg,
		Subtasks:   subtasks,
		TestGroups: testGroups,
		Verdicts:   verdicts,
	}
}
