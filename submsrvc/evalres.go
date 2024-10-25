package submsrvc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/tester/sqsgath"
)

func (s *SubmissionSrvc) StartProcessingSubmEvalResults(ctx context.Context) (err error) {
	submEvalResQueueUrl := s.resSqsUrl
	throtleChan := make(chan struct{}, 100)
	for i := 0; i < 100; i++ {
		throtleChan <- struct{}{}
	}
	for {
		output, err := s.sqsClient.ReceiveMessage(context.TODO(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(submEvalResQueueUrl),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     5,
		})
		if err != nil {
			log.Printf("failed to receive messages, %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, message := range output.Messages {
			_, err = s.sqsClient.DeleteMessage(context.TODO(), &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(submEvalResQueueUrl),
				ReceiptHandle: message.ReceiptHandle,
			})
			if err != nil {
				log.Printf("failed to delete message, %v\n", err)
			}

			var header sqsgath.Header
			err = json.Unmarshal([]byte(*message.Body), &header)
			if err != nil {
				log.Printf("failed to unmarshal message: %v\n", err)
				continue
			}

			switch header.MsgType {
			case sqsgath.MsgTypeStartedEvaluation:
				startedEvaluation := sqsgath.StartedEvaluation{}
				err = json.Unmarshal([]byte(*message.Body), &startedEvaluation)
				if err != nil {
					log.Printf("failed to unmarshal StartedEvaluation message: %v\n", err)
				} else {
					s.handleStartedEvaluation(&startedEvaluation)
				}
			case sqsgath.MsgTypeStartedCompilation:
				startedCompilation := sqsgath.StartedCompilation{}
				err = json.Unmarshal([]byte(*message.Body), &startedCompilation)
				if err != nil {
					log.Printf("failed to unmarshal StartedCompilation message: %v\n", err)
				} else {
					s.handleStartedCompilation(&startedCompilation)
				}
			case sqsgath.MsgTypeFinishedCompilation:
				finishedCompilation := sqsgath.FinishedCompilation{}
				err = json.Unmarshal([]byte(*message.Body), &finishedCompilation)
				if err != nil {
					log.Printf("failed to unmarshal FinishedCompilation message: %v\n", err)
				} else {
					s.handleFinishedCompilation(&finishedCompilation)
				}
			case sqsgath.MsgTypeStartedTesting:
				startedTesting := sqsgath.StartedTesting{}
				err = json.Unmarshal([]byte(*message.Body), &startedTesting)
				if err != nil {
					log.Printf("failed to unmarshal StartedTesting message: %v\n", err)
				} else {
					s.handleStartedTesting(&startedTesting)
				}
			case sqsgath.MsgTypeReachedTest:
				reachedTest := sqsgath.ReachedTest{}
				err = json.Unmarshal([]byte(*message.Body), &reachedTest)
				if err != nil {
					log.Printf("failed to unmarshal ReachedTest message: %v\n", err)
				} else {
					s.handleReachedTest(&reachedTest)
				}
			case sqsgath.MsgTypeIgnoredTest:
				ignoredTest := sqsgath.IgnoredTest{}
				err = json.Unmarshal([]byte(*message.Body), &ignoredTest)
				if err != nil {
					log.Printf("failed to unmarshal IgnoredTest message: %v\n", err)
				} else {
					s.handleIgnoredTest(&ignoredTest)
				}
			case sqsgath.MsgTypeFinishedTest:
				finishedTest := sqsgath.FinishedTest{}
				err = json.Unmarshal([]byte(*message.Body), &finishedTest)
				if err != nil {
					log.Printf("failed to unmarshal FinishedTest message: %v\n", err)
				} else {
					s.handleFinishedTest(&finishedTest)
				}
			case sqsgath.MsgTypeFinishedTesting:
				finishedTesting := sqsgath.FinishedTesting{}
				err = json.Unmarshal([]byte(*message.Body), &finishedTesting)
				if err != nil {
					log.Printf("failed to unmarshal FinishedTesting message: %v\n", err)
				} else {
					s.handleFinishedTesting(&finishedTesting)
				}
			case sqsgath.MsgTypeFinishedEvaluation:
				finishedEvaluation := sqsgath.FinishedEvaluation{}
				err = json.Unmarshal([]byte(*message.Body), &finishedEvaluation)
				if err != nil {
					log.Printf("failed to unmarshal FinishedEvaluation message: %v\n", err)
				} else {
					s.handleFinishedEvaluation(&finishedEvaluation)
				}
			}

			<-throtleChan
			throtleChan <- struct{}{}
		}
	}
}

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
	_, err = updStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
	}

	submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
	if err != nil {
		log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
		return
	}
	s.evalStageUpd <- &SubmEvalStageUpdate{
		SubmUuid: submUuid.String(),
		EvalUuid: evalUuid.String(),
		NewStage: "received",
	}
}

func (s *SubmissionSrvc) getSubmUuidFromEvalUuid(evalUuid uuid.UUID) (uuid.UUID, error) {
	// Check if evalUuid exists in s.evalUuidToSubmUuid sync.Map
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
	_, err = updStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
	}

	submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
	if err != nil {
		log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
		return
	}

	s.evalStageUpd <- &SubmEvalStageUpdate{
		SubmUuid: submUuid.String(),
		EvalUuid: evalUuid.String(),
		NewStage: "compiling",
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

	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	updStmt := table.Evaluations.
		UPDATE(table.Evaluations.EvaluationStage).
		SET(postgres.String("testing")).
		WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
	_, err = updStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation stage: %v", err)
	}

	submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
	if err != nil {
		log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
		return
	}

	s.evalStageUpd <- &SubmEvalStageUpdate{
		SubmUuid: submUuid.String(),
		EvalUuid: evalUuid.String(),
		NewStage: "testing",
	}
}

func logStartedTesting(x *sqsgath.StartedTesting) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleReachedTest(x *sqsgath.ReachedTest) {
	logReachedTest(x)

	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	if x.Input != nil && x.Answer != nil {
		updateStmt := table.EvaluationTests.
			UPDATE(table.EvaluationTests.Reached, table.EvaluationTests.InputTrimmed, table.EvaluationTests.AnswerTrimmed).
			SET(postgres.Bool(true), postgres.String(*x.Input), postgres.String(*x.Answer)).
			WHERE(
				table.EvaluationTests.EvalUUID.EQ(postgres.UUID(evalUuid)).
					AND(table.EvaluationTests.TestID.EQ(postgres.Int32(int32(x.TestId)))),
			)
		_, err = updateStmt.Exec(s.postgres)
		if err != nil {
			log.Printf("failed to update evaluation test reached: %v", err)
		}
	} else {
		log.Printf("reached test with nil input or answer")
	}
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
	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	updateStmt := table.EvaluationTests.
		UPDATE(table.EvaluationTests.Ignored).
		SET(postgres.Bool(true)).
		WHERE(
			table.EvaluationTests.EvalUUID.EQ(postgres.UUID(evalUuid)).
				AND(table.EvaluationTests.TestID.EQ(postgres.Int32(int32(x.TestId)))),
		)
	_, err = updateStmt.Exec(s.postgres)
	if err != nil {
		log.Printf("failed to update evaluation test ignored: %v", err)
	}
}

func logIgnoredTest(x *sqsgath.IgnoredTest) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	log.Printf("TestId: %d", x.TestId)
	log.Printf("--------------------------------")
}

func (s *SubmissionSrvc) handleFinishedTest(x *sqsgath.FinishedTest) {
	logFinishedTest(x)
	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	if x.Submission == nil {
		log.Printf("finished test with nil submission")
		return
	}

	now := time.Now()

	// Create and insert submission runtime data
	submRuntimeData := model.RuntimeData{
		Stdout:            x.Submission.Stdout,
		Stderr:            x.Submission.Stderr,
		ExitCode:          x.Submission.ExitCode,
		CPUTimeMillis:     x.Submission.CpuTimeMillis,
		WallTimeMillis:    x.Submission.WallTimeMillis,
		MemoryKibiBytes:   x.Submission.MemoryKibiBytes,
		CtxSwitchesForced: &x.Submission.ContextSwitchesForced,
		ExitSignal:        x.Submission.ExitSignal,
		IsolateStatus:     &x.Submission.IsolateStatus,
		CreatedAt:         &now,
	}

	insertSubmissionStmt := table.RuntimeData.
		INSERT(table.RuntimeData.MutableColumns).
		MODEL(submRuntimeData).
		RETURNING(table.RuntimeData.ID)

	err = insertSubmissionStmt.Query(s.postgres, &submRuntimeData)
	if err != nil {
		log.Printf("failed to insert submission runtime data: %v", err)
		return
	}

	var accepted bool

	if x.Checker == nil {
		accepted = false
		updateStmt := table.EvaluationTests.
			UPDATE(
				table.EvaluationTests.SubmRuntimeID,
				table.EvaluationTests.Finished,
				table.EvaluationTests.Accepted,
			).
			SET(
				postgres.Int32(submRuntimeData.ID),
				postgres.Bool(true),
				postgres.Bool(accepted),
			).
			WHERE(
				table.EvaluationTests.EvalUUID.EQ(postgres.UUID(evalUuid)).
					AND(table.EvaluationTests.TestID.EQ(postgres.Int32(int32(x.TestId)))),
			)

		_, err = updateStmt.Exec(s.postgres)
		if err != nil {
			log.Printf("failed to update evaluation test: %v", err)
		}
	} else {
		checkerRuntimeData := model.RuntimeData{
			Stdout:            x.Checker.Stdout,
			Stderr:            x.Checker.Stderr,
			ExitCode:          x.Checker.ExitCode,
			CPUTimeMillis:     x.Checker.CpuTimeMillis,
			WallTimeMillis:    x.Checker.WallTimeMillis,
			MemoryKibiBytes:   x.Checker.MemoryKibiBytes,
			CtxSwitchesForced: &x.Checker.ContextSwitchesForced,
			ExitSignal:        x.Checker.ExitSignal,
			IsolateStatus:     &x.Checker.IsolateStatus,
			CreatedAt:         &now,
		}

		insertCheckerStmt := table.RuntimeData.
			INSERT(table.RuntimeData.MutableColumns).
			MODEL(checkerRuntimeData).
			RETURNING(table.RuntimeData.ID)

		err = insertCheckerStmt.Query(s.postgres, &checkerRuntimeData)
		if err != nil {
			log.Printf("failed to insert checker runtime data: %v", err)
			return
		}

		accepted = true
		if x.Checker.ExitCode != 0 {
			accepted = false
		}

		// Update EvaluationTests table
		updateStmt := table.EvaluationTests.
			UPDATE(
				table.EvaluationTests.SubmRuntimeID,
				table.EvaluationTests.CheckerRuntimeID,
				table.EvaluationTests.Finished,
				table.EvaluationTests.Accepted,
			).
			SET(
				postgres.Int32(submRuntimeData.ID),
				postgres.Int32(checkerRuntimeData.ID),
				postgres.Bool(true),
				postgres.Bool(accepted),
			).
			WHERE(
				table.EvaluationTests.EvalUUID.EQ(postgres.UUID(evalUuid)).
					AND(table.EvaluationTests.TestID.EQ(postgres.Int32(int32(x.TestId)))),
			)

		_, err = updateStmt.Exec(s.postgres)
		if err != nil {
			log.Printf("failed to update evaluation test: %v", err)
		}
	}

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
		WHERE(table.EvaluationSubtasks.EvalUUID.EQ(postgres.UUID(evalUuid)).
			AND(
				table.EvaluationSubtasks.SubtaskID.IN(
					postgres.SELECT(postgres.Raw(fmt.Sprintf("unnest(%s)",
						table.EvaluationTests.Subtasks.Name()))).
						FROM(table.EvaluationTests).
						WHERE(
							table.EvaluationTests.EvalUUID.EQ(postgres.UUID(evalUuid)).
								AND(table.EvaluationTests.TestID.EQ(postgres.Int32(int32(x.TestId)))),
						),
				),
			)).RETURNING(table.EvaluationSubtasks.AllColumns)
	var subtasks []model.EvaluationSubtasks
	err = updSubtasksStmt.Query(s.postgres, &subtasks)
	if err != nil {
		log.Printf("failed to update evaluation subtasks: %v", err)
	}

	// for _, subtask := range subtasks {
	// 	s. <- &SubtaskScoringUpdate{
	// 		SubmUUID:      submUuid.String(),
	// 		EvalUUID:      evalUuid.String(),
	// 		SubtaskID:     int(subtask.SubtaskID),
	// 		AcceptedTests: int(subtask.Accepted),
	// 		WrongTests:    int(subtask.Wrong),
	// 		UntestedTests: int(subtask.Untested),
	// 	}
	// }

	// Update EvaluationTestGroups table
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
		WHERE(table.EvaluationTestgroups.EvalUUID.EQ(postgres.UUID(evalUuid)).
			AND(
				table.EvaluationTestgroups.TestgroupID.IN(
					postgres.SELECT(postgres.Raw(fmt.Sprintf("unnest(%s)",
						table.EvaluationTests.Testgroups.Name()))).
						FROM(table.EvaluationTests).
						WHERE(
							table.EvaluationTests.EvalUUID.EQ(postgres.UUID(evalUuid)).
								AND(table.EvaluationTests.TestID.EQ(postgres.Int32(int32(x.TestId)))),
						),
				),
			)).RETURNING(table.EvaluationTestgroups.AllColumns)
	var testGroups []model.EvaluationTestgroups
	err = updTestGroupsStmt.Query(s.postgres, &testGroups)
	if err != nil {
		log.Printf("failed to update evaluation test groups: %v", err)
	}

	submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
	if err != nil {
		log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
		return
	}

	for _, testGroup := range testGroups {
		s.testGroupScoreUpd <- &TestGroupScoringUpdate{
			SubmUUID:      submUuid.String(),
			EvalUUID:      evalUuid.String(),
			TestGroupID:   int(testGroup.TestgroupID),
			AcceptedTests: int(testGroup.Accepted),
			WrongTests:    int(testGroup.Wrong),
			UntestedTests: int(testGroup.Untested),
		}
	}

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
		WHERE(table.EvaluationTestset.EvalUUID.EQ(postgres.UUID(evalUuid))).
		RETURNING(table.EvaluationTestset.AllColumns)
	var testSet model.EvaluationTestset
	err = updTestSetStmt.Query(s.postgres, &testSet)
	if err != nil {
		log.Printf("failed to update evaluation test set: %v", err)
	}

	s.testSetScoreUpd <- &TestSetScoringUpdate{
		SubmUuid: submUuid.String(),
		EvalUuid: evalUuid.String(),
		Accepted: int(testSet.Accepted),
		Wrong:    int(testSet.Wrong),
		Untested: int(testSet.Untested),
	}
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

	evalUuid, err := uuid.Parse(x.EvalUuid)
	if err != nil {
		log.Printf("failed to parse eval_uuid: %v", err)
		return
	}

	if x.CompileError {
		var updateStmt postgres.UpdateStatement
		if x.ErrorMessage != nil {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.ErrorMessage).
				SET(postgres.String("compile_error"), postgres.String(*x.ErrorMessage)).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
		} else {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage).
				SET(postgres.String("compile_error")).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
		}
		_, err = updateStmt.Exec(s.postgres)
		if err != nil {
			log.Printf("failed to update evaluation stage: %v", err)
		}

		submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
		if err != nil {
			log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
			return
		}

		s.evalStageUpd <- &SubmEvalStageUpdate{
			SubmUuid: submUuid.String(),
			EvalUuid: evalUuid.String(),
			NewStage: "compile_error",
		}
	} else if x.InternalError || (x.ErrorMessage != nil && *x.ErrorMessage != "") {
		var updateStmt postgres.UpdateStatement
		if x.ErrorMessage != nil {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.ErrorMessage).
				SET(postgres.String("internal_error"), postgres.String(*x.ErrorMessage)).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
		} else {
			updateStmt = table.Evaluations.
				UPDATE(table.Evaluations.EvaluationStage, table.Evaluations.ErrorMessage).
				SET(postgres.String("internal_error"), postgres.String("unknown error")).
				WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
		}
		_, err = updateStmt.Exec(s.postgres)
		if err != nil {
			log.Printf("failed to update evaluation stage: %v", err)
		}

		submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
		if err != nil {
			log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
			return
		}

		s.evalStageUpd <- &SubmEvalStageUpdate{
			SubmUuid: submUuid.String(),
			EvalUuid: evalUuid.String(),
			NewStage: "internal_error",
		}
	} else {
		updateStmt := table.Evaluations.
			UPDATE(table.Evaluations.EvaluationStage).
			SET(postgres.String("finished")).
			WHERE(table.Evaluations.EvalUUID.EQ(postgres.UUID(evalUuid)))
		_, err = updateStmt.Exec(s.postgres)
		if err != nil {
			log.Printf("failed to update evaluation stage: %v", err)
		}

		submUuid, err := s.getSubmUuidFromEvalUuid(evalUuid)
		if err != nil {
			log.Printf("failed to get subm_uuid from eval_uuid: %v", err)
			return
		}

		s.evalStageUpd <- &SubmEvalStageUpdate{
			SubmUuid: submUuid.String(),
			EvalUuid: evalUuid.String(),
			NewStage: "finished",
		}
	}
}

func logFinishedEvaluation(x *sqsgath.FinishedEvaluation) {
	log.Printf("EvalUuid: %.6s...", x.EvalUuid)
	log.Printf("MsgType: %s", x.MsgType)
	if x.ErrorMessage != nil {
		log.Printf("ErrorMessage: %s", *x.ErrorMessage)
	} else {
		log.Printf("ErrorMessage: nil")
	}
	log.Printf("CompileError: %t", x.CompileError)
	log.Printf("InternalError: %t", x.InternalError)
	log.Printf("--------------------------------")
}
