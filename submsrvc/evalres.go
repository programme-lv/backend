package submsrvc

import (
	"log"
	"strings"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/tester/sqsgath"
)

func (s *SubmissionSrvc) handleStartedEvaluation(x *sqsgath.StartedEvaluation) {
	logStartedEvaluation(x)

	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.SystemInformation).
		SET(postgres.String("received"), postgres.String(x.SystemInfo)).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
	log.Printf("update statement: %s", updStmt.DebugSql())
	_, err = updStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
	}
}

func logStartedEvaluation(x *sqsgath.StartedEvaluation) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("StartedTime: %s", x.StartedTime)
	log.Printf("SystemInfo: %.50s...", strings.ReplaceAll(x.SystemInfo, "\n", "\\n")[:50])
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleStartedCompilation(x *sqsgath.StartedCompilation) {
	logStartedCompilation(x)

	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.EvaluationStage).
		SET(postgres.String("compiling")).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
	log.Printf("update statement: %s", updStmt.DebugSql())
	_, err = updStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
	}
}

func logStartedCompilation(x *sqsgath.StartedCompilation) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleFinishedCompilation(x *sqsgath.FinishedCompilation) {
	logFinishedCompilation(x)

	now := time.Now()
	runtimeData := model.RuntimeData{
		Stdout:            x.RuntimeData.Stdout,
		Stderr:            x.RuntimeData.Stderr,
		ExitCode:          x.RuntimeData.ExitCode,
		CPUTimeMillis:     x.RuntimeData.CpuTimeMillis,
		WallTimeMillis:    x.RuntimeData.WallTimeMillis,
		MemoryKibiBytes:   x.RuntimeData.MemoryKibiBytes,
		CtxSwitchesForced: &x.RuntimeData.ContextSwitchesForced,
		ExitSignal:        x.RuntimeData.ExitSignal,
		IsolateStatus:     &x.RuntimeData.IsolateStatus,
		CreatedAt:         &now,
	}

	// insert runtime data and return its id
	insertStmt := table.RuntimeData.
		INSERT(table.RuntimeData.MutableColumns).
		MODEL(runtimeData).
		RETURNING(table.RuntimeData.ID)

	err := insertStmt.Query(s.postgres, &runtimeData)
	if err != nil {
		log.Printf("failed to insert runtime data: %v", err)
		return
	}

	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.CompileRuntimeID).
		SET(postgres.Int32(runtimeData.ID)).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
	_, err = updStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation compile runtime id: %v", err)
	}
}

func logFinishedCompilation(x *sqsgath.FinishedCompilation) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	if x.RuntimeData.Stdout != nil {
		log.Printf("Stdout: %.50s...", strings.ReplaceAll(*x.RuntimeData.Stdout, "\n", "\\n"))
	} else {
		log.Printf("Stdout: nil")
	}
	if x.RuntimeData.Stderr != nil {
		log.Printf("Stderr: %.50s...", strings.ReplaceAll(*x.RuntimeData.Stderr, "\n", "\\n"))
	} else {
		log.Printf("Stderr: nil")
	}
	log.Printf("ExitCode: %d", x.RuntimeData.ExitCode)
	log.Printf("CpuTimeMillis: %d", x.RuntimeData.CpuTimeMillis)
	log.Printf("WallTimeMillis: %d", x.RuntimeData.WallTimeMillis)
	log.Printf("MemoryKibiBytes: %d", x.RuntimeData.MemoryKibiBytes)
	log.Printf("ContextSwitchesVoluntary: %d", x.RuntimeData.ContextSwitchesVoluntary)
	log.Printf("ContextSwitchesForced: %d", x.RuntimeData.ContextSwitchesForced)
	log.Printf("ExitSignal: %v", x.RuntimeData.ExitSignal)
	log.Printf("IsolateStatus: %s", x.RuntimeData.IsolateStatus)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleStartedTesting(x *sqsgath.StartedTesting) {
	logStartedTesting(x)
}

func logStartedTesting(x *sqsgath.StartedTesting) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleReachedTest(x *sqsgath.ReachedTest) {
	logReachedTest(x)
}

func logReachedTest(x *sqsgath.ReachedTest) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("TestId: %d", x.TestId)
	if x.Input != nil {
		log.Printf("Input: %.50s...", strings.ReplaceAll(*x.Input, "\n", "\\n"))
	} else {
		log.Printf("Input: nil")
	}
	if x.Answer != nil {
		log.Printf("Answer: %.50s...", strings.ReplaceAll(*x.Answer, "\n", "\\n"))
	} else {
		log.Printf("Answer: nil")
	}
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleIgnoredTest(x *sqsgath.IgnoredTest) {
	logIgnoredTest(x)
}

func logIgnoredTest(x *sqsgath.IgnoredTest) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("TestId: %d", x.TestId)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleFinishedTest(x *sqsgath.FinishedTest) {
	logFinishedTest(x)
}

func logFinishedTest(x *sqsgath.FinishedTest) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("TestId: %d", x.TestId)
	if x.Submission != nil {
		log.Printf("Submission Stdout: %.50s...", strings.ReplaceAll(*x.Submission.Stdout, "\n", "\\n"))
		log.Printf("Submission Stderr: %.50s...", strings.ReplaceAll(*x.Submission.Stderr, "\n", "\\n"))
		log.Printf("Submission ExitCode: %d", x.Submission.ExitCode)
		log.Printf("Submission CpuTimeMillis: %d", x.Submission.CpuTimeMillis)
		log.Printf("Submission MemoryKibiBytes: %d", x.Submission.MemoryKibiBytes)
	}
	if x.Checker != nil {
		log.Printf("Checker Stdout: %.50s...", strings.ReplaceAll(*x.Checker.Stdout, "\n", "\\n"))
		log.Printf("Checker Stderr: %.50s...", strings.ReplaceAll(*x.Checker.Stderr, "\n", "\\n"))
		log.Printf("Checker ExitCode: %d", x.Checker.ExitCode)
		log.Printf("Checker CpuTimeMillis: %d", x.Checker.CpuTimeMillis)
		log.Printf("Checker MemoryKibiBytes: %d", x.Checker.MemoryKibiBytes)
	}
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleFinishedTesting(x *sqsgath.FinishedTesting) {
	logFinishedTesting(x)
}

func logFinishedTesting(x *sqsgath.FinishedTesting) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleFinishedEvaluation(x *sqsgath.FinishedEvaluation) {
	logFinishedEvaluation(x)
}

func logFinishedEvaluation(x *sqsgath.FinishedEvaluation) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	if x.Error != nil {
		log.Printf("Error: %s", *x.Error)
	} else {
		log.Printf("Error: nil")
	}
	log.Printf("--------------------------------")
}
