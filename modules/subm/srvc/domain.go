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
		eval.Tests[u.TestID-1].Finished = true
		if u.Subm != nil {
			if u.Subm.IsOomKilled || u.Subm.MemKiB >= int64(eval.MemLimKiB) {
				eval.Tests[u.TestID-1].Mle = true
			} else if u.Subm.ExitCode != 0 || u.Subm.StdErr != "" || u.Subm.Signal != nil {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.CpuMs >= int64(eval.CpuLimMs) {
				eval.Tests[u.TestID-1].Tle = true
			} else if u.Checker != nil {
				if u.Checker.ExitCode == 0 {
					eval.Tests[u.TestID-1].Ac = true
				} else {
					eval.Tests[u.TestID-1].Wa = true
				}
			}
			cpuMs := int(u.Subm.CpuMs)
			eval.Tests[u.TestID-1].CpuMs = &cpuMs
			memKiB := int(u.Subm.MemKiB)
			eval.Tests[u.TestID-1].MemKiB = &memKiB
		}
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
