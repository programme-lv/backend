package submpgrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/subm/submdomain"
)

type pgEvalRepo struct {
	pool *pgxpool.Pool
}

func NewPgEvalRepo(pool *pgxpool.Pool) *pgEvalRepo {
	return &pgEvalRepo{pool: pool}
}

func (r *pgEvalRepo) StoreEval(ctx context.Context, eval submdomain.Eval) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Upsert Evaluation
	evaluationUpsertQuery := `
		INSERT INTO evaluations (
			uuid, subm_uuid, stage, score_unit, checker, interactor,
			cpu_lim_ms, mem_lim_kib, error_type, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (uuid) DO UPDATE SET
			subm_uuid = EXCLUDED.subm_uuid,
			stage = EXCLUDED.stage,
			score_unit = EXCLUDED.score_unit,
			checker = EXCLUDED.checker,
			interactor = EXCLUDED.interactor,
			cpu_lim_ms = EXCLUDED.cpu_lim_ms,
			mem_lim_kib = EXCLUDED.mem_lim_kib,
			error_type = EXCLUDED.error_type,
			error_message = EXCLUDED.error_message,
			created_at = EXCLUDED.created_at
	`
	var errorType *string
	var errorMessage *string
	if eval.Error != nil {
		et := string(eval.Error.Type)
		errorType = &et
		errorMessage = eval.Error.Message
	}

	_, err = tx.Exec(ctx, evaluationUpsertQuery,
		eval.UUID,
		eval.SubmUUID,
		eval.Stage,
		eval.ScoreUnit,
		eval.Checker,
		eval.Interactor,
		eval.CpuLimMs,
		eval.MemLimKiB,
		errorType,
		errorMessage,
		eval.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert evaluation: %w", err)
	}

	// Delete existing related data first to avoid duplicates
	deleteQueries := []string{
		`DELETE FROM subtasks WHERE evaluation_uuid = $1`,
		`DELETE FROM test_groups WHERE evaluation_uuid = $1`,
		`DELETE FROM tests WHERE evaluation_uuid = $1`,
	}
	for _, query := range deleteQueries {
		_, err = tx.Exec(ctx, query, eval.UUID)
		if err != nil {
			return fmt.Errorf("failed to delete existing data: %w", err)
		}
	}

	// Insert Subtasks
	for _, subtask := range eval.Subtasks {
		subtaskInsertQuery := `
			INSERT INTO subtasks (
				evaluation_uuid, points, description, st_tests
			) VALUES ($1, $2, $3, $4)
		`
		_, err = tx.Exec(ctx, subtaskInsertQuery,
			eval.UUID,
			subtask.Points,
			subtask.Description,
			subtask.StTests,
		)
		if err != nil {
			return fmt.Errorf("failed to insert subtask: %w", err)
		}
	}

	// Insert TestGroups
	for _, group := range eval.Groups {
		groupInsertQuery := `
			INSERT INTO test_groups (
				evaluation_uuid, points, subtasks, tg_tests
			) VALUES ($1, $2, $3, $4)
		`
		_, err = tx.Exec(ctx, groupInsertQuery,
			eval.UUID,
			group.Points,
			group.Subtasks,
			group.TgTests,
		)
		if err != nil {
			return fmt.Errorf("failed to insert test group: %w", err)
		}
	}

	// Insert Tests
	for _, test := range eval.Tests {
		testInsertQuery := `
			INSERT INTO tests (
				evaluation_uuid, ac, wa, tle, mle, re, ig, reached, finished,
				inp_sha256, ans_sha256
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`
		_, err = tx.Exec(ctx, testInsertQuery,
			eval.UUID,
			test.Ac,
			test.Wa,
			test.Tle,
			test.Mle,
			test.Re,
			test.Ig,
			test.Reached,
			test.Finished,
			nullableString(test.InpSha256),
			nullableString(test.AnsSha256),
		)
		if err != nil {
			return fmt.Errorf("failed to insert test: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *pgEvalRepo) GetEval(ctx context.Context, evalUUID uuid.UUID) (submdomain.Eval, error) {
	// Fetch Evaluation
	evalQuery := `
		SELECT uuid, subm_uuid, stage, score_unit, checker, interactor, cpu_lim_ms, mem_lim_kib,
			   error_type, error_message, created_at
		FROM evaluations
		WHERE uuid = $1
	`
	var eval submdomain.Eval
	var errorType *string
	var errorMessage *string
	err := r.pool.QueryRow(ctx, evalQuery, evalUUID).Scan(
		&eval.UUID,
		&eval.SubmUUID,
		&eval.Stage,
		&eval.ScoreUnit,
		&eval.Checker,
		&eval.Interactor,
		&eval.CpuLimMs,
		&eval.MemLimKiB,
		&errorType,
		&errorMessage,
		&eval.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return submdomain.Eval{}, fmt.Errorf("evaluation not found: %w", err)
		}
		return submdomain.Eval{}, fmt.Errorf("failed to query evaluation: %w", err)
	}

	// Handle EvaluationError
	if errorType != nil {
		et := submdomain.EvalErrorType(*errorType)
		eval.Error = &submdomain.EvalError{
			Type:    et,
			Message: errorMessage,
		}
	}

	// Fetch Subtasks
	subtasksQuery := `
		SELECT points, description, st_tests
		FROM subtasks
		WHERE evaluation_uuid = $1
	`
	subtaskRows, err := r.pool.Query(ctx, subtasksQuery, evalUUID)
	if err != nil {
		return submdomain.Eval{}, fmt.Errorf("failed to query subtasks: %w", err)
	}
	defer subtaskRows.Close()

	for subtaskRows.Next() {
		var st submdomain.Subtask
		err := subtaskRows.Scan(&st.Points, &st.Description, &st.StTests)
		if err != nil {
			return submdomain.Eval{}, fmt.Errorf("failed to scan subtask: %w", err)
		}
		eval.Subtasks = append(eval.Subtasks, st)
	}
	if err := subtaskRows.Err(); err != nil {
		return submdomain.Eval{}, fmt.Errorf("error iterating subtasks: %w", err)
	}

	// Fetch TestGroups
	testGroupsQuery := `
		SELECT points, subtasks, tg_tests
		FROM test_groups
		WHERE evaluation_uuid = $1
	`
	groupRows, err := r.pool.Query(ctx, testGroupsQuery, evalUUID)
	if err != nil {
		return submdomain.Eval{}, fmt.Errorf("failed to query test groups: %w", err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var tg submdomain.TestGroup
		err := groupRows.Scan(&tg.Points, &tg.Subtasks, &tg.TgTests)
		if err != nil {
			return submdomain.Eval{}, fmt.Errorf("failed to scan test group: %w", err)
		}
		eval.Groups = append(eval.Groups, tg)
	}
	if err := groupRows.Err(); err != nil {
		return submdomain.Eval{}, fmt.Errorf("error iterating test groups: %w", err)
	}

	// Fetch Tests
	testsQuery := `
		SELECT ac, wa, tle, mle, re, ig, reached, finished, inp_sha256, ans_sha256
		FROM tests
		WHERE evaluation_uuid = $1
	`
	testRows, err := r.pool.Query(ctx, testsQuery, evalUUID)
	if err != nil {
		return submdomain.Eval{}, fmt.Errorf("failed to query tests: %w", err)
	}
	defer testRows.Close()

	for testRows.Next() {
		var test submdomain.Test
		var inpSha256, ansSha256 *string
		err := testRows.Scan(
			&test.Ac,
			&test.Wa,
			&test.Tle,
			&test.Mle,
			&test.Re,
			&test.Ig,
			&test.Reached,
			&test.Finished,
			&inpSha256,
			&ansSha256,
		)
		if err != nil {
			return submdomain.Eval{}, fmt.Errorf("failed to scan test: %w", err)
		}
		if inpSha256 != nil {
			test.InpSha256 = *inpSha256
		}
		if ansSha256 != nil {
			test.AnsSha256 = *ansSha256
		}
		eval.Tests = append(eval.Tests, test)
	}
	if err := testRows.Err(); err != nil {
		return submdomain.Eval{}, fmt.Errorf("error iterating tests: %w", err)
	}

	return eval, nil
}

// Helper function to handle nullable strings
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type pgSubmRepo struct {
	pool *pgxpool.Pool
}

func NewPgSubmRepo(pool *pgxpool.Pool) *pgSubmRepo {
	return &pgSubmRepo{pool: pool}
}

// StoreSubm inserts a new SubmissionEntity into the database.
func (r *pgSubmRepo) StoreSubm(ctx context.Context, subm submdomain.Subm) error {
	submissionInsertQuery := `
		INSERT INTO submissions (
			uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var currEvalId *uuid.UUID
	if subm.CurrEvalUUID != uuid.Nil {
		currEvalId = &subm.CurrEvalUUID
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

// GetSubm retrieves a SubmissionEntity by UUID
func (r *pgSubmRepo) GetSubm(ctx context.Context, id uuid.UUID) (submdomain.Subm, error) {
	submissionQuery := `
		SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		FROM submissions
		WHERE uuid = $1
	`
	var s submdomain.Subm
	err := r.pool.QueryRow(ctx, submissionQuery, id).Scan(
		&s.UUID,
		&s.Content,
		&s.AuthorUUID,
		&s.TaskShortID,
		&s.LangShortID,
		&s.CurrEvalUUID,
		&s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return submdomain.Subm{}, fmt.Errorf("submission not found: %w", err)
		}
		return submdomain.Subm{}, fmt.Errorf("failed to query submission: %w", err)
	}

	return s, nil
}

// ListSubms retrieves all SubmissionEntities from the database
func (r *pgSubmRepo) ListSubms(ctx context.Context, limit int, offset int) ([]submdomain.Subm, error) {
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

	var submissions []submdomain.Subm
	for rows.Next() {
		var subm submdomain.Subm
		err := rows.Scan(
			&subm.UUID,
			&subm.Content,
			&subm.AuthorUUID,
			&subm.TaskShortID,
			&subm.LangShortID,
			&subm.CurrEvalUUID,
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
