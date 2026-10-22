package srvc

import (
	"fmt"
	"log/slog"

	"github.com/programme-lv/backend/modules/exec"
	"github.com/programme-lv/backend/modules/subm/domain"
)

func applyExecEventToEval(eval domain.Eval, event exec.Event) domain.Eval {
	switch u := event.(type) {
	case exec.ReceivedSubmission:
	case exec.StartedCompiling:
		eval.Stage = domain.EvalStageCompiling
	case exec.FinishedTesting:
		eval.Stage = domain.EvalStageFinished
	case exec.InternalServerError:
		eval.Stage = domain.EvalStageFinished
		eval.Error = &domain.EvalError{
			Type:    domain.ErrorTypeInternal,
			Message: u.ErrorMsg,
		}
	case exec.CompilationError:
		eval.Stage = domain.EvalStageFinished
		eval.Error = &domain.EvalError{
			Type:    domain.ErrorTypeCompilation,
			Message: u.ErrorMsg,
		}
	case exec.ReachedTest:
		eval.Stage = domain.EvalStageTesting
		if u.TestId > len(eval.Tests) {
			slog.Error("reached test out of bounds", "test_id", u.TestId, "eval", fmt.Sprintf("%+v", eval))
			return eval
		}
		eval.Tests[u.TestId-1].Reached = true
	case exec.FinishedTest:
		eval.Stage = domain.EvalStageTesting
		if u.TestID > len(eval.Tests) {
			slog.Error("finished test out of bounds", "test_id", u.TestID, "eval", fmt.Sprintf("%+v", eval))
			return eval
		}
		test := &eval.Tests[u.TestID-1]
		test.Finished = true
		applyTestResult(test, eval, u.Subm, u.Checker)
	case exec.IgnoredTest:
		eval.Stage = domain.EvalStageTesting
		if u.TestId > len(eval.Tests) {
			slog.Error("ignored test out of bounds", "test_id", u.TestId, "eval", fmt.Sprintf("%+v", eval))
			return eval
		}
		eval.Tests[u.TestId-1].Ig = true
	}
	return eval
}

func applyTestResult(test *domain.Test, eval domain.Eval, subm, checker *exec.RunData) {
	if subm == nil {
		return
	}
	cpuMs := int(subm.CpuMs)
	test.CpuMs = &cpuMs
	memKiB := int(subm.MemKiB)
	test.MemKiB = &memKiB

	if subm.IsOomKilled || subm.MemKiB >= int64(eval.MemLimKiB) {
		test.Mle = true
		return
	}
	if subm.ExitCode != 0 || subm.Signal != nil {
		test.Re = true
		return
	}
	if subm.CpuMs >= int64(eval.CpuLimMs) {
		test.Tle = true
		return
	}
	if eval.Interactor != nil {
		percent := interactorPercent(checker)
		test.Score = percent
		test.Ac = percent == 100
		test.Wa = percent == 0
		return
	}
	if subm.StdErr != "" {
		test.Re = true
		return
	}
	if checker == nil || checker.ExitCode == 0 {
		test.Score = 100
		test.Ac = true
		return
	}
	test.Wa = true
}

// interactorPercent maps the interactor exit status to a test percentage.
// Interactors are compiled with PC_BASE_EXIT_CODE=50, so _pc(p) exits
// with 50+p.
func interactorPercent(checker *exec.RunData) int {
	if checker == nil {
		return 0
	}
	if checker.ExitCode == 0 {
		return 100
	}
	if checker.ExitCode >= 50 && checker.ExitCode <= 150 {
		return int(checker.ExitCode - 50)
	}
	return 0
}
