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

// Store inserts a new SubmissionEntity along with its Evaluation and related entities into the database.
func (r *pgSubmRepo) Store(ctx context.Context, subm SubmissionEntity) error {
	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Safe to call, does nothing if already committed

	// Insert Submission
	submissionInsertQuery := `
		INSERT INTO submissions (
			uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var currEvalUUID *uuid.UUID
	if subm.CurrEval != nil {
		currEvalUUID = &subm.CurrEval.UUID
	}
	_, err = tx.Exec(ctx, submissionInsertQuery,
		subm.UUID,
		subm.Content,
		subm.AuthorUUID,
		subm.TaskShortID,
		subm.LangShortID,
		currEvalUUID,
		subm.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert submission: %w", err)
	}

	// If there's a current evaluation, insert it and its related entities
	if subm.CurrEval != nil {
		eval := subm.CurrEval

		// Insert Evaluation
		evaluationInsertQuery := `
			INSERT INTO evaluations (
				uuid, submission_uuid, stage, score_unit, checker, interactor,
				cpu_lim_ms, mem_lim_kib, error_type, error_message, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`
		var errorType *string
		var errorMessage *string
		if eval.Error != nil {
			et := string(eval.Error.Type)
			errorType = &et
			errorMessage = eval.Error.Message
		}

		_, err = tx.Exec(ctx, evaluationInsertQuery,
			eval.UUID,
			subm.UUID,
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
			return fmt.Errorf("failed to insert evaluation: %w", err)
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
					evaluation_uuid, ac, wa, tle, mle, re, ig, reached, finished
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
			)
			if err != nil {
				return fmt.Errorf("failed to insert test: %w", err)
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Get retrieves a SubmissionEntity by UUID, including its Evaluation and related entities.
func (r *pgSubmRepo) Get(ctx context.Context, id uuid.UUID) (*SubmissionEntity, error) {
	// Fetch Submission
	submissionQuery := `
		SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		FROM submissions
		WHERE uuid = $1
	`
	var subm SubmissionEntity
	var currEvalUUID *uuid.UUID
	err := r.pool.QueryRow(ctx, submissionQuery, id).Scan(
		&subm.UUID,
		&subm.Content,
		&subm.AuthorUUID,
		&subm.TaskShortID,
		&subm.LangShortID,
		&currEvalUUID,
		&subm.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("submission not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query submission: %w", err)
	}

	// If there's a current evaluation, fetch it along with related entities
	if currEvalUUID != nil {
		eval, err := r.fetchEvaluation(ctx, *currEvalUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch evaluation: %w", err)
		}
		subm.CurrEval = eval
	}

	return &subm, nil
}

// List retrieves all SubmissionEntities from the database, including their Evaluations.
func (r *pgSubmRepo) List(ctx context.Context) ([]SubmissionEntity, error) {
	submissionsQuery := `
		SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		FROM submissions
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, submissionsQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []SubmissionEntity
	for rows.Next() {
		var subm SubmissionEntity
		var currEvalUUID *uuid.UUID
		err := rows.Scan(
			&subm.UUID,
			&subm.Content,
			&subm.AuthorUUID,
			&subm.TaskShortID,
			&subm.LangShortID,
			&currEvalUUID,
			&subm.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		// If there's a current evaluation, fetch it
		if currEvalUUID != nil {
			eval, err := r.fetchEvaluation(ctx, *currEvalUUID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch evaluation: %w", err)
			}
			subm.CurrEval = eval
		}

		submissions = append(submissions, subm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}

	return submissions, nil
}

// fetchEvaluation retrieves an Evaluation and its related entities by UUID.
func (r *pgSubmRepo) fetchEvaluation(ctx context.Context, evalUUID uuid.UUID) (*Evaluation, error) {
	// Fetch Evaluation
	evalQuery := `
		SELECT uuid, stage, score_unit, checker, interactor, cpu_lim_ms, mem_lim_kib,
			   error_type, error_message, created_at
		FROM evaluations
		WHERE uuid = $1
	`
	var eval Evaluation
	var errorType *string
	var errorMessage *string
	err := r.pool.QueryRow(ctx, evalQuery, evalUUID).Scan(
		&eval.UUID,
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
			return nil, fmt.Errorf("evaluation not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query evaluation: %w", err)
	}

	// Handle EvaluationError
	if errorType != nil {
		et := EvaluationErrorType(*errorType)
		eval.Error = &EvaluationError{
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
		return nil, fmt.Errorf("failed to query subtasks: %w", err)
	}
	defer subtaskRows.Close()

	for subtaskRows.Next() {
		var st Subtask
		err := subtaskRows.Scan(&st.Points, &st.Description, &st.StTests)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subtask: %w", err)
		}
		eval.Subtasks = append(eval.Subtasks, st)
	}
	if err := subtaskRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating subtasks: %w", err)
	}

	// Fetch TestGroups
	testGroupsQuery := `
		SELECT points, subtasks, tg_tests
		FROM test_groups
		WHERE evaluation_uuid = $1
	`
	groupRows, err := r.pool.Query(ctx, testGroupsQuery, evalUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query test groups: %w", err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var tg TestGroup
		err := groupRows.Scan(&tg.Points, &tg.Subtasks, &tg.TgTests)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test group: %w", err)
		}
		eval.Groups = append(eval.Groups, tg)
	}
	if err := groupRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating test groups: %w", err)
	}

	// Fetch Tests
	testsQuery := `
		SELECT ac, wa, tle, mle, re, ig, reached, finished
		FROM tests
		WHERE evaluation_uuid = $1
	`
	testRows, err := r.pool.Query(ctx, testsQuery, evalUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tests: %w", err)
	}
	defer testRows.Close()

	for testRows.Next() {
		var test Test
		err := testRows.Scan(
			&test.Ac,
			&test.Wa,
			&test.Tle,
			&test.Mle,
			&test.Re,
			&test.Ig,
			&test.Reached,
			&test.Finished,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test: %w", err)
		}
		eval.Tests = append(eval.Tests, test)
	}
	if err := testRows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tests: %w", err)
	}

	return &eval, nil
}
