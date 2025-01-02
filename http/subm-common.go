package http

import (
	"time"

	"github.com/programme-lv/backend/submsrvc"
)

type Submission struct {
	SubmUUID  string    `json:"subm_uuid"`
	Content   string    `json:"content"`
	Username  string    `json:"username"`
	CurrEval  *SubmEval `json:"curr_eval"`
	PrLang    PrLang    `json:"pr_lang"`
	TaskName  string    `json:"task_name"`
	TaskID    string    `json:"task_id"`
	CreatedAt string    `json:"created_at"`
}

type PrLang struct {
	ShortID  string `json:"short_id"`
	Display  string `json:"display"`
	MonacoID string `json:"monaco_id"`
}

type SubmEval struct {
	EvalUUID     string      `json:"eval_uuid"`
	EvalStage    string      `json:"eval_stage"`
	ScoreUnit    string      `json:"score_unit"`
	EvalError    string      `json:"eval_error"`
	ErrorMsg     string      `json:"error_msg"`
	Subtasks     []Subtask   `json:"subtasks"`
	TestGroups   []TestGroup `json:"test_groups"`
	TestVerdicts []string    `json:"test_verdicts"` // q,ac,wa,tle,mle,re,ig
}

type Subtask struct {
	Points      int    `json:"points"`
	Description string `json:"description"`
	StTests     []int  `json:"st_tests"`
}

type TestGroup struct {
	Points   int   `json:"points"`
	Subtasks []int `json:"subtasks"`
	TgTests  []int `json:"tg_tests"`
}

func mapSubm(subm submsrvc.Submission) *Submission {
	var currEval *SubmEval
	if subm.CurrEval != nil {
		mapped := mapSubmEval(*subm.CurrEval)
		currEval = &mapped
	}
	return &Submission{
		SubmUUID: subm.UUID.String(),
		Content:  subm.Content,
		Username: subm.Author.Username,
		CurrEval: currEval,
		PrLang: PrLang{
			ShortID:  subm.Lang.ShortID,
			Display:  subm.Lang.Display,
			MonacoID: subm.Lang.MonacoID,
		},
		TaskName:  subm.Task.FullName,
		TaskID:    subm.Task.ShortID,
		CreatedAt: subm.CreatedAt.Format(time.RFC3339),
	}
}

func mapSubmEval(eval submsrvc.Evaluation) SubmEval {
	errType := ""
	errMsg := ""
	if eval.Error != nil {
		errType = string(eval.Error.Type)
		if eval.Error.Message != nil {
			errMsg = *eval.Error.Message
		}
	}

	subtasks := []Subtask{}
	for _, subtask := range eval.Subtasks {
		subtasks = append(subtasks, Subtask{
			Points:      subtask.Points,
			Description: subtask.Description,
			StTests:     subtask.StTests,
		})
	}

	testGroups := []TestGroup{}
	for _, testGroup := range eval.Groups {
		testGroups = append(testGroups, TestGroup{
			Points:   testGroup.Points,
			Subtasks: testGroup.Subtasks,
			TgTests:  testGroup.TgTests,
		})
	}

	testVerdicts := []string{}
	for _, test := range eval.Tests {
		if test.Finished {
			if test.Ac {
				testVerdicts = append(testVerdicts, "ac") // accepted
			} else if test.Wa {
				testVerdicts = append(testVerdicts, "wa") // wrong answer
			} else if test.Tle {
				testVerdicts = append(testVerdicts, "tle") // time limit exceeded
			} else if test.Mle {
				testVerdicts = append(testVerdicts, "mle") // memory limit exceeded
			} else if test.Re {
				testVerdicts = append(testVerdicts, "re") // runtime error
			} else if test.Ig {
				testVerdicts = append(testVerdicts, "ig") // ignored
			} else {
				testVerdicts = append(testVerdicts, "") // unknown
			}
		} else if test.Reached {
			testVerdicts = append(testVerdicts, "t") // testing
		} else {
			testVerdicts = append(testVerdicts, "q") // queued
		}
	}

	return SubmEval{
		EvalUUID:     eval.UUID.String(),
		EvalStage:    eval.Stage,
		ScoreUnit:    eval.ScoreUnit,
		EvalError:    errType,
		ErrorMsg:     errMsg,
		Subtasks:     subtasks,
		TestGroups:   testGroups,
		TestVerdicts: testVerdicts,
	}
}
