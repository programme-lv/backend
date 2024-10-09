package submsrvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/gen/postgres/public/model"
	"github.com/programme-lv/backend/gen/postgres/public/table"
	"github.com/programme-lv/backend/user"
)

type CreateSubmissionParams struct {
	Submission string
	Username   string
	ProgLangID string
	TaskCodeID string
}

func (s *SubmissionSrvc) CreateSubmission(ctx context.Context,
	params *CreateSubmissionParams) (*Submission, error) {
	// validate & retrieve USER
	user, err := s.userSrvc.GetUserByUsername(ctx,
		&user.GetUserByUsernamePayload{Username: params.Username})
	if err != nil {
		return nil, err
	}
	userUuid, err := uuid.Parse(user.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user UUID: %w", err)
	}

	// validate & retrieve PROGRAMMING LANGUAGE
	languages, err := s.ListProgrammingLanguages(ctx)
	if err != nil {
		return nil, err
	}
	var language *ProgrammingLang
	for _, l := range languages {
		if l.ID == params.ProgLangID {
			language = &l
			break
		}
	}

	if language == nil {
		return nil, fmt.Errorf("programming language not found")
	}

	// validate & retrieve TASK
	task, err := s.taskSrvc.GetTask(ctx, params.TaskCodeID)
	if err != nil {
		return nil, err
	}

	submission := model.Submissions{
		Content:    params.Submission,
		AuthorUUID: userUuid,
		TaskID:     task.ShortId,
		ProgLangID: language.ID,
	}
	insertStmt := table.Submissions.
		INSERT(
			table.Submissions.Content,
			table.Submissions.AuthorUUID,
			table.Submissions.TaskID,
			table.Submissions.ProgLangID,
		).
		MODEL(&submission).
		RETURNING(
			table.Submissions.SubmUUID,
			table.Submissions.CreatedAt,
		)

	var insertedSubmissions []model.Submissions

	err = insertStmt.QueryContext(ctx, s.postgres,
		&insertedSubmissions)
	if err != nil {
		return nil, fmt.Errorf("failed to insert submission: %w", err)
	}

	if len(insertedSubmissions) == 0 {
		return nil, errors.New("failed to insert submission")
	}

	insertedSubmission := insertedSubmissions[0]

	return &Submission{
		UUID:    insertedSubmission.SubmUUID,
		Content: insertedSubmission.Content,
		Author: Author{
			UUID:     insertedSubmission.AuthorUUID,
			Username: user.Username,
		},
		Task: Task{
			ShortID:  task.ShortId,
			FullName: task.FullName,
		},
		Lang: Lang{
			ShortID:  language.ID,
			Display:  language.FullName,
			MonacoID: language.MonacoId,
		},
		CreatedAt: insertedSubmission.CreatedAt,
		CurrEval: Evaluation{
			UUID: uuid.Nil,
		},
	}, nil
}
