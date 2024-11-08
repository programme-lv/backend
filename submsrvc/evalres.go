package submsrvc

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
)

func (s *SubmissionSrvc) StartProcessingSubmEvalResults(ctx context.Context) (err error) {
	evalHandlers := make(map[uuid.UUID]*evalHandler)
	for {
		msgs, err := s.evalSrvc.Receive()
		if err != nil {
			log.Printf("failed to receive messages, %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		// map events by evaluation uuids into handlers
		handlerEvents := make(map[*evalHandler][]evalsrvc.Msg)
		for _, msg := range msgs {
			submUuid, err := s.getSubmUuidFromEvalUuid(msg.EvalId)
			if err != nil {
				log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
				continue
			}

			handler, ok := evalHandlers[msg.EvalId]
			if !ok {
				handler = &evalHandler{
					postgres: s.postgres,
					evalUuid: msg.EvalId,
					submUuid: submUuid,
				}
				evalHandlers[msg.EvalId] = handler
			}
			handlerEvents[handler] = append(handlerEvents[handler], msg)
		}

		for handler, events := range handlerEvents {
			// map events by feedback type
			eventsByType := make(map[string][]evalsrvc.Msg)
			for _, event := range events {
				eventsByType[event.Data.Type()] = append(eventsByType[event.Data.Type()], event)
			}

			// process equal events in batches
			for t, events := range eventsByType {
				switch t {
				case evalsrvc.MsgTypeStartedEvaluation:
					go handler.startedEvaluation(events[0].Data.(evalsrvc.StartedEvaluation))
				case evalsrvc.MsgTypeStartedCompilation:
					go handler.startedCompiling(events[0].Data.(evalsrvc.StartedCompiling))
				case evalsrvc.MsgTypeFinishedCompilation:
					go handler.finishedCompiling(events[0].Data.(evalsrvc.FinishedCompiling))
				case evalsrvc.MsgTypeStartedTesting:
					go handler.startedTesting(events[0].Data.(evalsrvc.StartedTesting))
				case evalsrvc.MsgTypeReachedTest:
					e := make([]evalsrvc.ReachedTest, len(events))
					for i, event := range events {
						e[i] = event.Data.(evalsrvc.ReachedTest)
					}
					go handler.reachedTests(e)
				case evalsrvc.MsgTypeIgnoredTest:
					e := make([]evalsrvc.IgnoredTest, len(events))
					for i, event := range events {
						e[i] = event.Data.(evalsrvc.IgnoredTest)
					}
					go handler.ignoredTests(e)
				case evalsrvc.MsgTypeFinishedTest:
					e := make([]evalsrvc.FinishedTest, len(events))
					for i, event := range events {
						e[i] = event.Data.(evalsrvc.FinishedTest)
					}
					for _, event := range e {
						go handler.finishedTest(event)
					}
				case evalsrvc.MsgTypeFinishedEvaluation:
					go handler.finishedEvaluation(events[0].Data.(evalsrvc.FinishedEvaluation))
				}
			}
		}
	}
}

type evalHandler struct {
	postgres *sqlx.DB
	evalUuid uuid.UUID
	submUuid uuid.UUID
	stageUpd chan *EvalStageUpd
	groupUpd chan *TGroupScoreUpd
	tSetUpd  chan *TSetScoreUpd
}

func (h *evalHandler) startedEvaluation(e evalsrvc.StartedEvaluation) {
	sysInfo := e.SysInfo
	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.SystemInformation).
		SET(postgres.String("received"), postgres.String(sysInfo)).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
	_, err := updStmt.Exec(h.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
		return
	}
	h.stageUpd <- &EvalStageUpd{
		SubmUuid: h.submUuid.String(),
		EvalUuid: h.evalUuid.String(),
		NewStage: "received",
	}
}

func (h *evalHandler) startedCompiling(_ evalsrvc.StartedCompiling) {
	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.EvaluationStage).
		SET(postgres.String("compiling")).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
	_, err := updStmt.Exec(h.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
		return
	}
	h.stageUpd <- &EvalStageUpd{
		SubmUuid: h.submUuid.String(),
		EvalUuid: h.evalUuid.String(),
		NewStage: "compiling",
	}
}

func (h *evalHandler) finishedCompiling(x evalsrvc.FinishedCompiling) {
	now := time.Now()
	runtimeData := model.RuntimeData{
		Stdout:            &x.RuntimeData.StdOut,
		Stderr:            &x.RuntimeData.StdErr,
		ExitCode:          x.RuntimeData.ExitCode,
		CPUTimeMillis:     x.RuntimeData.CpuMs,
		WallTimeMillis:    x.RuntimeData.WallMs,
		MemoryKibiBytes:   x.RuntimeData.MemKiB,
		CtxSwitchesForced: &x.RuntimeData.CtxSwV,
		ExitSignal:        &x.RuntimeData.ExitCode,
		CreatedAt:         &now,
	}
	insertStmt := table.RuntimeData.
		INSERT(table.RuntimeData.MutableColumns).
		MODEL(runtimeData).
		RETURNING(table.RuntimeData.ID)
	err := insertStmt.Query(h.postgres, &runtimeData)
	if err != nil {
		log.Printf("failed to insert runtime data: %v", err)
		return
	}

	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.CompileRuntimeID).
		SET(postgres.Int32(runtimeData.ID)).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
	_, err = updStmt.Exec(h.postgres)
	if err != nil {
		log.Printf("failed to update evaluation compile runtime id: %v", err)
	}
}

func (h *evalHandler) startedTesting(_ evalsrvc.StartedTesting) {
	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.EvaluationStage).
		SET(postgres.String("testing")).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
	_, err := updStmt.Exec(h.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
	}
	h.stageUpd <- &EvalStageUpd{
		SubmUuid: h.submUuid.String(),
		EvalUuid: h.evalUuid.String(),
		NewStage: "testing",
	}
}

func (h *evalHandler) reachedTests(x []evalsrvc.ReachedTest) {
	for _, test := range x {
		go func(test evalsrvc.ReachedTest) {
			if test.Ans == nil || test.In == nil {
				log.Printf("reached test with nil input or answer")
			}
			updateStmt := table.EvaluationTests.
				UPDATE(
					table.EvaluationTests.Reached,
					table.EvaluationTests.InputTrimmed,
					table.EvaluationTests.AnswerTrimmed,
				).
				SET(
					postgres.Bool(true),
					postgres.String(*test.In),
					postgres.String(*test.Ans),
				).
				WHERE(
					table.EvaluationTests.EvalUUID.EQ(
						postgres.UUID(h.evalUuid),
					).AND(table.EvaluationTests.TestID.EQ(
						postgres.Int64(test.TestId),
					)),
				)
			_, err := updateStmt.Exec(h.postgres)
			if err != nil {
				log.Printf("failed to update evaluation test reached: %v", err)
			}
		}(test)
	}
}

func (h *evalHandler) ignoredTests(x []evalsrvc.IgnoredTest) {
	ids := make([]postgres.Expression, len(x))
	for i, test := range x {
		ids[i] = postgres.Int(test.TestId)
	}
	updStmt := table.EvaluationTests.
		UPDATE(table.EvaluationTests.Ignored).
		SET(postgres.Bool(true)).
		WHERE(
			table.EvaluationTests.EvalUUID.EQ(
				postgres.UUID(h.evalUuid),
			).AND(table.EvaluationTests.TestID.IN(
				ids...,
			)),
		)
	_, err := updStmt.Exec(h.postgres)
	if err != nil {
		log.Printf("failed to update evaluation test reached: %v", err)
	}
}

func insertRunData(pg *sqlx.DB, rd *evalsrvc.RunData) (int32, error) {
	now := time.Now()
	res := model.RuntimeData{
		Stdout:            &rd.StdOut,
		Stderr:            &rd.StdErr,
		ExitCode:          rd.ExitCode,
		CPUTimeMillis:     rd.CpuMs,
		WallTimeMillis:    rd.WallMs,
		MemoryKibiBytes:   rd.MemKiB,
		CtxSwitchesForced: &rd.CtxSwF,
		ExitSignal:        rd.Signal,
		CreatedAt:         &now,
	}
	insertStmt := table.RuntimeData.
		INSERT(table.RuntimeData.MutableColumns).
		MODEL(res).
		RETURNING(table.RuntimeData.ID)

	err := insertStmt.Query(pg, &res)
	if err != nil {
		log.Printf("failed to insert runtime data: %v", err)
		return 0, err
	}
	return res.ID, nil
}

func (h *evalHandler) finishedTest(x evalsrvc.FinishedTest) {
	if x.Subm == nil {
		log.Printf("finished test with nil submission")
		return
	}

	submDataId, err := insertRunData(h.postgres, x.Subm)
	if err != nil {
		log.Printf("failed to insert submission runtime data: %v", err)
		return
	}

	var checkerDataId int32
	if x.Checker != nil {
		checkerDataId, err = insertRunData(h.postgres, x.Checker)
		if err != nil {
			log.Printf("failed to insert checker runtime data: %v", err)
			return
		}
	}

	accepted := (x.Checker != nil && x.Checker.ExitCode == 0)

	var updateStmt postgres.UpdateStatement

	if checkerDataId != 0 {
		updateStmt = table.EvaluationTests.
			UPDATE(
				table.EvaluationTests.SubmRuntimeID,
				table.EvaluationTests.CheckerRuntimeID,
				table.EvaluationTests.Reached,
				table.EvaluationTests.Finished,
				table.EvaluationTests.Accepted,
			).SET(
			postgres.Int32(submDataId),
			postgres.Int32(checkerDataId),
			postgres.Bool(true),
			postgres.Bool(true),
			postgres.Bool(accepted),
		)
	} else {
		updateStmt = table.EvaluationTests.
			UPDATE(
				table.EvaluationTests.SubmRuntimeID,
				table.EvaluationTests.Reached,
				table.EvaluationTests.Finished,
				table.EvaluationTests.Accepted,
			).SET(
			postgres.Int32(submDataId),
			postgres.Bool(true),
			postgres.Bool(true),
			postgres.Bool(accepted),
		)
	}

	updateStmt = updateStmt.WHERE(
		table.EvaluationTests.EvalUUID.EQ(postgres.UUID(h.evalUuid)).
			AND(table.EvaluationTests.TestID.EQ(postgres.Int64(x.TestID))),
	)

	_, err = updateStmt.Exec(h.postgres)
	if err != nil {
		log.Printf("failed to update evaluation test: %v", err)
	}

	// update SUBTASK aggregate results
	go func() {
		var updSubtasksStmt postgres.UpdateStatement
		if accepted {
			updSubtasksStmt = table.EvaluationSubtasks.
				UPDATE(table.EvaluationSubtasks.Accepted, table.EvaluationSubtasks.Untested)
			updSubtasksStmt = updSubtasksStmt.SET(
				table.EvaluationSubtasks.Accepted.ADD(postgres.Int(1)),
				table.EvaluationSubtasks.Untested.SUB(postgres.Int(1)),
			)
		} else {
			updSubtasksStmt = table.EvaluationSubtasks.
				UPDATE(table.EvaluationSubtasks.Untested, table.EvaluationSubtasks.Wrong)
			updSubtasksStmt = updSubtasksStmt.SET(
				table.EvaluationSubtasks.Untested.SUB(postgres.Int(1)),
				table.EvaluationSubtasks.Wrong.ADD(postgres.Int(1)),
			)
		}

		updSubtasksStmt = updSubtasksStmt.
			WHERE(table.EvaluationSubtasks.EvalUUID.EQ(postgres.UUID(h.evalUuid)).
				AND(table.EvaluationSubtasks.SubtaskID.IN(postgres.SELECT(
					postgres.Raw(fmt.Sprintf("unnest(%s)",
						table.EvaluationTests.Subtasks.Name()))).
					FROM(table.EvaluationTests).
					WHERE(
						table.EvaluationTests.EvalUUID.EQ(postgres.UUID(h.evalUuid)).
							AND(table.EvaluationTests.TestID.EQ(postgres.Int64(x.TestID))),
					),
				),
				)).RETURNING(table.EvaluationSubtasks.AllColumns)
		var subtasks []model.EvaluationSubtasks
		err = updSubtasksStmt.Query(h.postgres, &subtasks)
		if err != nil {
			log.Printf("failed to update evaluation subtasks: %v", err)
		}
	}()

	// update test GROUP aggregate results
	go func() {
		var updTestGroupsStmt postgres.UpdateStatement
		if accepted {
			updTestGroupsStmt = table.EvaluationTestgroups.
				UPDATE(table.EvaluationTestgroups.Accepted, table.EvaluationTestgroups.Untested)
			updTestGroupsStmt = updTestGroupsStmt.SET(
				table.EvaluationTestgroups.Accepted.ADD(postgres.Int(1)),
				table.EvaluationTestgroups.Untested.SUB(postgres.Int(1)),
			)
		} else {
			updTestGroupsStmt = table.EvaluationTestgroups.
				UPDATE(table.EvaluationTestgroups.Untested, table.EvaluationTestgroups.Wrong)
			updTestGroupsStmt = updTestGroupsStmt.SET(
				table.EvaluationTestgroups.Untested.SUB(postgres.Int(1)),
				table.EvaluationTestgroups.Wrong.ADD(postgres.Int(1)),
			)
		}

		updTestGroupsStmt = updTestGroupsStmt.
			WHERE(table.EvaluationTestgroups.EvalUUID.EQ(postgres.UUID(h.evalUuid)).
				AND(
					table.EvaluationTestgroups.TestgroupID.IN(
						postgres.SELECT(postgres.Raw(fmt.Sprintf("unnest(%s)",
							table.EvaluationTests.Testgroups.Name()))).
							FROM(table.EvaluationTests).
							WHERE(
								table.EvaluationTests.EvalUUID.EQ(postgres.UUID(h.evalUuid)).
									AND(table.EvaluationTests.TestID.EQ(postgres.Int64(x.TestID))),
							),
					),
				)).RETURNING(table.EvaluationTestgroups.AllColumns)
		var testGroups []model.EvaluationTestgroups
		err = updTestGroupsStmt.Query(h.postgres, &testGroups)
		if err != nil {
			log.Printf("failed to update evaluation test groups: %v", err)
		}

		for _, testGroup := range testGroups {
			h.groupUpd <- &TGroupScoreUpd{
				SubmUUID:      h.submUuid.String(),
				EvalUUID:      h.evalUuid.String(),
				TestGroupID:   int(testGroup.TestgroupID),
				AcceptedTests: int(testGroup.Accepted),
				WrongTests:    int(testGroup.Wrong),
				UntestedTests: int(testGroup.Untested),
			}
		}
	}()

	// update test SET aggregate results
	go func() {
		var updTestSetStmt postgres.UpdateStatement
		if accepted {
			updTestSetStmt = table.EvaluationTestset.
				UPDATE(table.EvaluationTestset.Accepted, table.EvaluationTestset.Untested)
			updTestSetStmt = updTestSetStmt.SET(
				table.EvaluationTestset.Accepted.ADD(postgres.Int(1)),
				table.EvaluationTestset.Untested.SUB(postgres.Int(1)),
			)
		} else {
			updTestSetStmt = table.EvaluationTestset.
				UPDATE(table.EvaluationTestset.Untested, table.EvaluationTestset.Wrong)
			updTestSetStmt = updTestSetStmt.SET(
				table.EvaluationTestset.Untested.SUB(postgres.Int(1)),
				table.EvaluationTestset.Wrong.ADD(postgres.Int(1)),
			)
		}

		updTestSetStmt = updTestSetStmt.
			WHERE(table.EvaluationTestset.EvalUUID.EQ(postgres.UUID(h.evalUuid))).
			RETURNING(table.EvaluationTestset.AllColumns)
		var testSet model.EvaluationTestset
		err = updTestSetStmt.Query(h.postgres, &testSet)
		if err != nil {
			log.Printf("failed to update evaluation test set: %v", err)
		}

		h.tSetUpd <- &TSetScoreUpd{
			SubmUuid: h.submUuid.String(),
			EvalUuid: h.evalUuid.String(),
			Accepted: int(testSet.Accepted),
			Wrong:    int(testSet.Wrong),
			Untested: int(testSet.Untested),
		}
	}()
}

func (h *evalHandler) finishedEvaluation(x evalsrvc.FinishedEvaluation) {
	if x.CompileError {
		var updateStmt postgres.UpdateStatement
		if x.ErrorMsg != nil {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.ErrorMessage).
				SET(postgres.String("compile_error"), postgres.String(*x.ErrorMsg)).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
		} else {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage).
				SET(postgres.String("compile_error")).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
		}
		_, err := updateStmt.Exec(h.postgres)
		if err != nil {
			log.Printf("failed to update evaluation stage: %v", err)
		}

		h.stageUpd <- &EvalStageUpd{
			SubmUuid: h.submUuid.String(),
			EvalUuid: h.evalUuid.String(),
			NewStage: "compile_error",
		}
	} else if x.InternalError || (x.ErrorMsg != nil && *x.ErrorMsg != "") {
		var updateStmt postgres.UpdateStatement
		if x.ErrorMsg != nil {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.ErrorMessage).
				SET(postgres.String("internal_error"), postgres.String(*x.ErrorMsg)).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
		} else {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.ErrorMessage).
				SET(postgres.String("internal_error"), postgres.String("unknown error")).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
		}
		_, err := updateStmt.Exec(h.postgres)
		if err != nil {
			log.Printf("failed to update evaluation stage: %v", err)
		}

		h.stageUpd <- &EvalStageUpd{
			SubmUuid: h.submUuid.String(),
			EvalUuid: h.evalUuid.String(),
			NewStage: "internal_error",
		}
	} else {
		updateStmt := table.Evaluations.
			UPDATE(table.Evaluations.EvaluationStage).
			SET(postgres.String("finished")).
			WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(h.evalUuid)))
		_, err := updateStmt.Exec(h.postgres)
		if err != nil {
			log.Printf("failed to update evaluation stage: %v", err)
		}

		h.stageUpd <- &EvalStageUpd{
			SubmUuid: h.submUuid.String(),
			EvalUuid: h.evalUuid.String(),
			NewStage: "finished",
		}
	}
}

func (s *SubmissionSrvc) getSubmUuidFromEvalUuid(evalUuid uuid.UUID) (uuid.UUID, error) {
	if submUuid, ok := s.evalUuidToSubmUuid.Load(evalUuid); ok {
		return submUuid.(uuid.UUID), nil
	}

	// If not found, perform a database select
	var subm model.Submissions
	err := table.Submissions.
		SELECT(table.Submissions.SubmUUID).
		WHERE(table.Submissions.CurrentEvalUUID.EQ(postgres.UUID(evalUuid))).
		Query(s.postgres, &subm)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get subm_uuid from database: %v", err)
	}

	// Store the result in s.evalUuidToSubmUuid sync.Map
	s.evalUuidToSubmUuid.Store(evalUuid, subm.SubmUUID)

	return subm.SubmUUID, nil
}
