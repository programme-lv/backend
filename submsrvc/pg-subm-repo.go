package submsrvc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgSubmRepo struct {
	pool *pgxpool.Pool
}

func NewPgSubmRepo(pool *pgxpool.Pool) *pgSubmRepo {
	return &pgSubmRepo{pool: pool}
}

// Store inserts a new SubmissionEntity into the database.
func (r *pgSubmRepo) Store(ctx context.Context, subm SubmissionEntity) error {
	submissionInsertQuery := `
		INSERT INTO submissions (
			uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var currEvalId *uuid.UUID
	if subm.CurrEvalID != uuid.Nil {
		currEvalId = &subm.CurrEvalID
	}
	_, err := r.pool.Exec(ctx, submissionInsertQuery,
		subm.UUID,
		subm.Content,
		subm.AuthorUUID,
		subm.TaskShortID,
		subm.LangShortID,
		currEvalId,
		subm.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert submission: %w", err)
	}

	return nil
}

func (r *pgSubmRepo) AssignEval(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error {
	updateQuery := `
		UPDATE submissions 
		SET curr_eval_uuid = $1
		WHERE uuid = $2
	`
	_, err := r.pool.Exec(ctx, updateQuery, evalUuid, submUuid)
	if err != nil {
		return fmt.Errorf("failed to assign evaluation to submission: %w", err)
	}
	return nil
}

// Get retrieves a SubmissionEntity by UUID
func (r *pgSubmRepo) Get(ctx context.Context, id uuid.UUID) (SubmissionEntity, error) {
	submissionQuery := `
		SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		FROM submissions
		WHERE uuid = $1
	`
	var subm SubmissionEntity
	err := r.pool.QueryRow(ctx, submissionQuery, id).Scan(
		&subm.UUID,
		&subm.Content,
		&subm.AuthorUUID,
		&subm.TaskShortID,
		&subm.LangShortID,
		&subm.CurrEvalID,
		&subm.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SubmissionEntity{}, fmt.Errorf("submission not found: %w", err)
		}
		return SubmissionEntity{}, fmt.Errorf("failed to query submission: %w", err)
	}

	return subm, nil
}

// List retrieves all SubmissionEntities from the database
func (r *pgSubmRepo) List(ctx context.Context, limit int, offset int) ([]SubmissionEntity, error) {
	submissionsQuery := `
			SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
			FROM submissions
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
	`
	rows, err := r.pool.Query(ctx, submissionsQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []SubmissionEntity
	for rows.Next() {
		var subm SubmissionEntity
		err := rows.Scan(
			&subm.UUID,
			&subm.Content,
			&subm.AuthorUUID,
			&subm.TaskShortID,
			&subm.LangShortID,
			&subm.CurrEvalID,
			&subm.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, subm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}

	return submissions, nil
}
