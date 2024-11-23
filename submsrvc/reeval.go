package submsrvc

import (
	"context"
	"fmt"

	"github.com/go-jet/jet/v2/postgres"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/evalsrvc"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
)

func (s *SubmissionSrvc) ReevaluateSubmission(ctx context.Context, submUuid string) error {
	// check if the submission exists
	selectSubmStmt := postgres.SELECT(table.Submissions.AllColumns).WHERE(
		table.Submissions.SubmUUID.EQ(postgres.String(submUuid)),
	)

	var subms []model.Submissions
	if err := selectSubmStmt.Query(s.postgres, &subms); err != nil {
		format := "failed to query submissions: %w"
		errMsg := fmt.Errorf(format, err)
		return ErrSubmissionNotFound().SetDebug(errMsg)
	}
	if len(subms) == 0 {
		return ErrSubmissionNotFound()
	}
	if len(subms) > 1 {
		format := "multiple submissions found with the same UUID: %s"
		errMsg := fmt.Errorf(format, submUuid)
		return ErrInternalSE().SetDebug(errMsg)
	}

	subm := subms[0]

	task, err := s.taskSrvc.GetTask(ctx, subm.TaskID)
	if err != nil {
		format := "failed to get task: %w"
		errMsg := fmt.Errorf(format, err)
		return ErrTaskNotFound().SetDebug(errMsg)
	}

	evalUuid, err := uuid.NewV7()
	if err != nil {
		format := "failed to generate UUID: %w"
		errMsg := fmt.Errorf(format, err)
		return ErrInternalSE().SetDebug(errMsg)
	}

	// update submission with new eval UUID
	updateStmt := table.Submissions.UPDATE(table.Submissions.AllColumns).SET(
		table.Submissions.CurrentEvalUUID.SET(postgres.String(evalUuid.String())),
	).WHERE(
		table.Submissions.SubmUUID.EQ(postgres.String(submUuid)),
	)
	if _, err := updateStmt.Exec(s.postgres); err != nil {
		format := "failed to update submission: %w"
		errMsg := fmt.Errorf(format, err)
		return ErrInternalSE().SetDebug(errMsg)
	}

	req := evalsrvc.Request{
		Code:       subm.Content,
		Tests:      evalReqTests(&task),
		Checker:    task.CheckerPtr(),
		Interactor: task.InteractorPtr(),
		CpuMs:      task.CpuMillis(),
		MemKiB:     task.MemoryKiB(),
	}

	// enqueue evaluation
	if _, err := s.evalSrvc.Enqueue(req, evalUuid); err != nil {
		format := "failed to enqueue evaluation: %w"
		errMsg := fmt.Errorf(format, err)
		return ErrInternalSE().SetDebug(errMsg)
	}

	return nil
}
