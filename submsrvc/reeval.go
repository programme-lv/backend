package submsrvc

import (
	"context"
	"fmt"
	"log"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/planglist"
)

type ReevaluateResponse struct {
	SubmUuid    uuid.UUID
	OldEvalUuid uuid.UUID
	NewEvalUuid uuid.UUID
}

func (s *SubmissionSrvc) ReevaluateSubmission(ctx context.Context, submUuidStr string) (*ReevaluateResponse, error) {
	submUuid, err := uuid.Parse(submUuidStr)
	if err != nil {
		format := "failed to parse submission UUID: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	log.Println("reevaluating submission", submUuid)
	// check if the submission exists
	selectSubmStmt := postgres.SELECT(table.Submissions.AllColumns).FROM(
		table.Submissions,
	).WHERE(
		table.Submissions.SubmUUID.EQ(postgres.UUID(submUuid)),
	)

	var subms []model.Submissions
	if err := selectSubmStmt.Query(s.postgres, &subms); err != nil {
		format := "failed to query submissions: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrSubmissionNotFound().SetDebug(errMsg)
	}
	if len(subms) == 0 {
		return nil, ErrSubmissionNotFound()
	}
	if len(subms) > 1 {
		format := "multiple submissions found with the same UUID: %s"
		errMsg := fmt.Errorf(format, submUuid)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	subm := subms[0]

	task, err := s.taskSrvc.GetTask(ctx, subm.TaskID)
	if err != nil {
		format := "failed to get task: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrTaskNotFound().SetDebug(errMsg)
	}

	lang, err := planglist.GetProgrammingLanguageById(subm.ProgLangID)
	if err != nil {
		format := "failed to get programming language: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	evalUuid, err := s.InsertNewEvaluation(ctx, &task, lang)
	if err != nil {
		format := "failed to insert new evaluation: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	// update submission with new eval UUID
	updateStmt := table.Submissions.UPDATE(table.Submissions.AllColumns).SET(
		table.Submissions.CurrentEvalUUID.SET(postgres.UUID(evalUuid)),
	).WHERE(
		table.Submissions.SubmUUID.EQ(postgres.UUID(submUuid)),
	)
	if _, err := updateStmt.Exec(s.postgres); err != nil {
		format := "failed to update submission: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	s.evalUuidToSubmUuid.Store(evalUuid, submUuid)

	req := evalsrvc.Request{
		Code:       subm.Content,
		Tests:      evalReqTests(&task),
		Checker:    task.CheckerPtr(),
		Interactor: task.InteractorPtr(),
		CpuMs:      task.CpuMillis(),
		MemKiB:     task.MemoryKiB(),
		LangId:     subm.ProgLangID,
	}

	// enqueue evaluation
	if _, err := s.evalSrvc.Enqueue(req, evalUuid); err != nil {
		format := "failed to enqueue evaluation: %w"
		errMsg := fmt.Errorf(format, err)
		return nil, ErrInternalSE().SetDebug(errMsg)
	}

	oldEvalUuid := uuid.Nil
	if subm.CurrentEvalUUID != nil {
		oldEvalUuid = *subm.CurrentEvalUUID
	}

	return &ReevaluateResponse{
		SubmUuid:    submUuid,
		OldEvalUuid: oldEvalUuid,
		NewEvalUuid: evalUuid,
	}, nil
}
