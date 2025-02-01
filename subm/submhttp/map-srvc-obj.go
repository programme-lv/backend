package submhttp

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/subm"
)

func mapSubmListEntry(
	ctx context.Context,
	s subm.Subm,
	getTaskName func(ctx context.Context, shortID string) (string, error),
	getUsername func(ctx context.Context, userUuid uuid.UUID) (string, error),
	getPrLang func(ctx context.Context, shortID string) (PrLang, error),
	getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error),
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

	gotScore := 0
	maxScore := 0
	green := 0
	red := 0
	gray := 0
	yellow := 0
	purple := 0
	if eval.ScoreUnit == subm.ScoreUnitTestGroup {
		for _, testGroup := range eval.Groups {
			maxScore += testGroup.Points
		}
		if eval.Error == nil {
			for _, testGroup := range eval.Groups {
				allUncreached := true
				allAccepted := true
				hasWrong := false
				for _, testIdx := range testGroup.TgTests {
					test := eval.Tests[testIdx-1]
					if test.Reached {
						allUncreached = false
					}
					if !test.Ac {
						allAccepted = false
					}
					if test.Wa || test.Tle || test.Mle || test.Re {
						hasWrong = true
					}
				}
				if allUncreached {
					gray += testGroup.Points
				} else if allAccepted {
					green += testGroup.Points
					gotScore += testGroup.Points
				} else if hasWrong {
					red += testGroup.Points
				} else {
					yellow += testGroup.Points
				}
			}
		} else {
			purple = 100
		}
	} else if eval.ScoreUnit == subm.ScoreUnitTest {
		maxScore += len(eval.Tests)
		if eval.Error == nil {
			for _, test := range eval.Tests {
				if test.Ac {
					green += 1
					gotScore += 1
				} else if test.Wa || test.Tle || test.Mle || test.Re {
					red += 1
				} else if test.Reached {
					yellow += 1
				} else {
					gray += 1
				}
			}
		} else {
			purple = 100
		}
	}

	status := string(eval.Stage)
	if eval.Error != nil {
		status = string(eval.Error.Type)
	}

	// maxCpuMs := 0
	// maxMemMiB := 0

	// for _, test := range eval.Tests {
	// 	if test.CpuLimMs > maxCpuMs {
	// 		maxCpuMs = test.CpuLimMs
	// 	}
	// 	if test.MemLimKiB > maxMemMiB {
	// 		maxMemMiB = test.MemLimKiB
	// 	}
	// }

	// green red gray yellow purple should sum up to 100
	total := green + red + gray + yellow + purple
	green = green * 100 / total
	red = red * 100 / total
	yellow = yellow * 100 / total
	purple = purple * 100 / total
	gray = 100 - green - red - yellow - purple

	return SubmListEntry{
		SubmUuid:   s.UUID.String(),
		Username:   username,
		TaskId:     s.TaskShortID,
		TaskName:   taskName,
		PrLangId:   prLang.ShortID,
		PrLangName: prLang.Display,
		ScoreBar: struct {
			Green  int `json:"green"`
			Red    int `json:"red"`
			Gray   int `json:"gray"`
			Yellow int `json:"yellow"`
			Purple int `json:"purple"`
		}{
			Green:  green,
			Red:    red,
			Gray:   gray,
			Yellow: yellow,
			Purple: purple,
		},
		ReceivedScore: gotScore,
		PossibleScore: maxScore,
		Status:        status,
		CreatedAt:     s.CreatedAt.Format(time.RFC3339),
		MaxCpuMs:      0,
		MaxMemMiB:     0,
	}, nil
}

func mapSubm(
	ctx context.Context,
	subm subm.Subm,
	getTaskName func(ctx context.Context, shortID string) (string, error),
	getUsername func(ctx context.Context, userUuid uuid.UUID) (string, error),
	getPrLang func(ctx context.Context, shortID string) (PrLang, error),
	getEval func(ctx context.Context, evalUuid uuid.UUID) (subm.Eval, error),
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

func mapSubmEval(eval subm.Eval) Eval {
	errType := ""
	// errMsg := ""
	if eval.Error != nil {
		errType = string(eval.Error.Type)
		// if eval.Error.Message != nil {
		// 	errMsg = *eval.Error.Message
		// }
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
		EvalUUID:  eval.UUID.String(),
		EvalStage: string(eval.Stage),
		ScoreUnit: string(eval.ScoreUnit),
		EvalError: errType,
		// ErrorMsg:   errMsg,
		Subtasks:   subtasks,
		TestGroups: testGroups,
		Verdicts:   verdicts,
	}
}
