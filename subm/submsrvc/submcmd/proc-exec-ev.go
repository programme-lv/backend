package submcmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/execsrvc"
	decorator "github.com/programme-lv/backend/srvccqs"
	subm "github.com/programme-lv/backend/subm/domain"
)

type ProcExecEvCmd decorator.CmdHandler[ProcExecEvParams]

type ProcExecEvParams struct {
	Eval  subm.Eval
	Event execsrvc.Event
}

type ProcExecEvCmdHandler struct {
	StoreEval     func(ctx context.Context, eval subm.Eval) error
	BcastEvalUpd  func(eval subm.Eval)
	GetEvalByUuid func(ctx context.Context, uuid uuid.UUID) (subm.Eval, error)
	InProgrEval   map[uuid.UUID]subm.Eval
}

func (h *ProcExecEvCmdHandler) Handle(ctx context.Context, p ProcExecEvParams) error {
	latestEval, ok := h.InProgrEval[p.Eval.UUID]
	if !ok {
		return fmt.Errorf("eval not found in in-memory cache")
	}
	slog.Info("received event", "event", fmt.Sprintf("%+v", p.Event.Type()))

	eval := applyExecEventToEval(latestEval, p.Event)

	final := false
	final = final || p.Event.Type() == execsrvc.InternalServerErrorType
	final = final || p.Event.Type() == execsrvc.CompilationErrorType
	final = final || p.Event.Type() == execsrvc.FinishedTestingType

	if final {
		err := h.StoreEval(ctx, eval)
		if err != nil {
			slog.Error("failed to store evaluation", "error", err)
			return err
		} else {
			slog.Info("stored evaluation", "eval", fmt.Sprintf("%+v", eval))
		}
		delete(h.InProgrEval, p.Eval.UUID)
	} else {
		finishedTests := 0
		for _, test := range eval.Tests {
			if test.Finished {
				finishedTests++
			}
		}
		slog.Info("test progress", "finished", finishedTests, "total", len(eval.Tests))
		h.InProgrEval[p.Eval.UUID] = eval
	}

	h.BcastEvalUpd(eval)
	return nil
}

func applyExecEventToEval(eval subm.Eval, event execsrvc.Event) subm.Eval {
	switch u := event.(type) {
	case execsrvc.ReceivedSubmission:
	case execsrvc.StartedCompiling:
		eval.Stage = subm.EvalStageCompiling
	case execsrvc.StartedTesting:
		eval.Stage = subm.EvalStageTesting
	case execsrvc.FinishedTesting:
		eval.Stage = subm.EvalStageFinished
	case execsrvc.InternalServerError:
		eval.Stage = subm.EvalStageFinished
		eval.Error = &subm.EvalError{
			Type:    subm.ErrorTypeInternal,
			Message: u.ErrorMsg,
		}
	case execsrvc.CompilationError:
		eval.Stage = subm.EvalStageFinished
		eval.Error = &subm.EvalError{
			Type:    subm.ErrorTypeCompilation,
			Message: u.ErrorMsg,
		}
	case execsrvc.ReachedTest:
		if u.TestId > len(eval.Tests) {
			slog.Error("reached test out of bounds", "test_id", u.TestId, "eval", fmt.Sprintf("%+v", eval))
			return eval
		}
		eval.Tests[u.TestId-1].Reached = true
	case execsrvc.FinishedTest:
		if u.TestID > len(eval.Tests) {
			slog.Error("finished test out of bounds", "test_id", u.TestID, "eval", fmt.Sprintf("%+v", eval))
			return eval
		}
		eval.Tests[u.TestID-1].Finished = true
		if u.Subm != nil {
			if u.Subm.ExitCode != 0 {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.StdErr != "" {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.Signal != nil {
				eval.Tests[u.TestID-1].Re = true
			} else if u.Subm.CpuMs > int64(eval.CpuLimMs) {
				eval.Tests[u.TestID-1].Tle = true
			} else if u.Subm.MemKiB > int64(eval.MemLimKiB) {
				eval.Tests[u.TestID-1].Mle = true
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
	case execsrvc.IgnoredTest:
		if u.TestId > len(eval.Tests) {
			slog.Error("ignored test out of bounds", "test_id", u.TestId, "eval", fmt.Sprintf("%+v", eval))
			return eval
		}
		eval.Tests[u.TestId-1].Ig = true
	}
	return eval
}
