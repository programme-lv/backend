package pgrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/logger"
	"github.com/programme-lv/backend/subm/domain"
)

type pgEvalRepo struct {
	pool *pgxpool.Pool
}

func NewPgEvalRepo(pool *pgxpool.Pool) *pgEvalRepo {
	return &pgEvalRepo{pool: pool}
}

func (r *pgEvalRepo) StoreEval(ctx context.Context, eval domain.Eval) error {
	log := logger.FromContext(ctx)
	log.Debug("storing evaluation", "eval_uuid", eval.UUID, "subm_uuid", eval.SubmUUID)

	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		log.Debug("failed to begin transaction", "error", err)
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

	log.Debug("executing upsert query", "query", evaluationUpsertQuery)
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
		log.Debug("failed to upsert evaluation", "error", err)
		return fmt.Errorf("failed to upsert evaluation: %w", err)
	}

	// Delete existing related data first to avoid duplicates
	deleteQueries := []string{
		`DELETE FROM subtasks WHERE evaluation_uuid = $1`,
		`DELETE FROM test_groups WHERE evaluation_uuid = $1`,
		`DELETE FROM tests WHERE evaluation_uuid = $1`,
	}
	for _, query := range deleteQueries {
		log.Debug("executing delete query", "query", query)
		_, err = tx.Exec(ctx, query, eval.UUID)
		if err != nil {
			log.Debug("failed to delete existing data", "error", err)
			return fmt.Errorf("failed to delete existing data: %w", err)
		}
	}

	// Insert Subtasks
	log.Debug("inserting subtasks", "count", len(eval.Subtasks))
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
			log.Debug("failed to insert subtask", "error", err)
			return fmt.Errorf("failed to insert subtask: %w", err)
		}
	}

	// Insert TestGroups
	log.Debug("inserting test groups", "count", len(eval.Groups))
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
			log.Debug("failed to insert test group", "error", err)
			return fmt.Errorf("failed to insert test group: %w", err)
		}
	}

	// Insert Tests
	log.Debug("inserting tests", "count", len(eval.Tests))
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
			log.Debug("failed to insert test", "error", err)
			return fmt.Errorf("failed to insert test: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Debug("failed to commit transaction", "error", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug("evaluation stored successfully", "eval_uuid", eval.UUID)
	return nil
}

func (r *pgEvalRepo) GetEval(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting evaluation", "eval_uuid", evalUUID)

	// Fetch Evaluation
	evalQuery := `
		SELECT uuid, subm_uuid, stage, score_unit, checker, interactor, cpu_lim_ms, mem_lim_kib,
			   error_type, error_message, created_at
		FROM evaluations
		WHERE uuid = $1
	`
	var eval domain.Eval
	var errorType *string
	var errorMessage *string

	log.Debug("executing evaluation query", "query", evalQuery)
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
			log.Debug("evaluation not found", "eval_uuid", evalUUID)
			return domain.Eval{}, fmt.Errorf("evaluation not found: %w", err)
		}
		log.Debug("failed to query evaluation", "error", err)
		return domain.Eval{}, fmt.Errorf("failed to query evaluation: %w", err)
	}

	// Handle EvaluationError
	if errorType != nil {
		et := domain.EvalErrorType(*errorType)
		eval.Error = &domain.EvalError{
			Type:    et,
			Message: errorMessage,
		}
		log.Debug("evaluation has error", "error_type", et, "error_message", errorMessage)
	}

	// Fetch Subtasks
	subtasksQuery := `
		SELECT points, description, st_tests
		FROM subtasks
		WHERE evaluation_uuid = $1
	`
	log.Debug("executing subtasks query", "query", subtasksQuery)
	subtaskRows, err := r.pool.Query(ctx, subtasksQuery, evalUUID)
	if err != nil {
		log.Debug("failed to query subtasks", "error", err)
		return domain.Eval{}, fmt.Errorf("failed to query subtasks: %w", err)
	}
	defer subtaskRows.Close()

	for subtaskRows.Next() {
		var st domain.Subtask
		err := subtaskRows.Scan(&st.Points, &st.Description, &st.StTests)
		if err != nil {
			log.Debug("failed to scan subtask", "error", err)
			return domain.Eval{}, fmt.Errorf("failed to scan subtask: %w", err)
		}
		eval.Subtasks = append(eval.Subtasks, st)
	}
	if err := subtaskRows.Err(); err != nil {
		log.Debug("error iterating subtasks", "error", err)
		return domain.Eval{}, fmt.Errorf("error iterating subtasks: %w", err)
	}
	log.Debug("fetched subtasks", "count", len(eval.Subtasks))

	// Fetch TestGroups
	testGroupsQuery := `
		SELECT points, subtasks, tg_tests
		FROM test_groups
		WHERE evaluation_uuid = $1
	`
	log.Debug("executing test groups query", "query", testGroupsQuery)
	groupRows, err := r.pool.Query(ctx, testGroupsQuery, evalUUID)
	if err != nil {
		log.Debug("failed to query test groups", "error", err)
		return domain.Eval{}, fmt.Errorf("failed to query test groups: %w", err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var tg domain.TestGroup
		err := groupRows.Scan(&tg.Points, &tg.Subtasks, &tg.TgTests)
		if err != nil {
			log.Debug("failed to scan test group", "error", err)
			return domain.Eval{}, fmt.Errorf("failed to scan test group: %w", err)
		}
		eval.Groups = append(eval.Groups, tg)
	}
	if err := groupRows.Err(); err != nil {
		log.Debug("error iterating test groups", "error", err)
		return domain.Eval{}, fmt.Errorf("error iterating test groups: %w", err)
	}
	log.Debug("fetched test groups", "count", len(eval.Groups))

	// Fetch Tests
	testsQuery := `
		SELECT ac, wa, tle, mle, re, ig, reached, finished, inp_sha256, ans_sha256
		FROM tests
		WHERE evaluation_uuid = $1
	`
	log.Debug("executing tests query", "query", testsQuery)
	testRows, err := r.pool.Query(ctx, testsQuery, evalUUID)
	if err != nil {
		log.Debug("failed to query tests", "error", err)
		return domain.Eval{}, fmt.Errorf("failed to query tests: %w", err)
	}
	defer testRows.Close()

	for testRows.Next() {
		var test domain.Test
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
			log.Debug("failed to scan test", "error", err)
			return domain.Eval{}, fmt.Errorf("failed to scan test: %w", err)
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
		log.Debug("error iterating tests", "error", err)
		return domain.Eval{}, fmt.Errorf("error iterating tests: %w", err)
	}
	log.Debug("fetched tests", "count", len(eval.Tests))

	log.Debug("evaluation retrieved successfully", "eval_uuid", evalUUID)
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
func (r *pgSubmRepo) StoreSubm(ctx context.Context, subm domain.Subm) error {
	log := logger.FromContext(ctx)
	log.Debug("storing submission", "subm_uuid", subm.UUID, "author_uuid", subm.AuthorUUID, "task_id", subm.TaskShortID)

	submissionInsertQuery := `
		INSERT INTO submissions (
			uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	var currEvalId *uuid.UUID
	if subm.CurrEvalUUID != uuid.Nil {
		currEvalId = &subm.CurrEvalUUID
	}

	log.Debug("executing insert query", "query", submissionInsertQuery)
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
		log.Debug("failed to insert submission", "error", err)
		return fmt.Errorf("failed to insert submission: %w", err)
	}

	log.Debug("submission stored successfully", "subm_uuid", subm.UUID)
	return nil
}

func (r *pgSubmRepo) AssignEval(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error {
	log := logger.FromContext(ctx)
	log.Debug("assigning evaluation to submission", "subm_uuid", submUuid, "eval_uuid", evalUuid)

	updateQuery := `
		UPDATE submissions 
		SET curr_eval_uuid = $1
		WHERE uuid = $2
	`
	log.Debug("executing update query", "query", updateQuery)

	_, err := r.pool.Exec(ctx, updateQuery, evalUuid, submUuid)
	if err != nil {
		log.Debug("failed to assign evaluation to submission", "error", err)
		return fmt.Errorf("failed to assign evaluation to submission: %w", err)
	}

	log.Debug("evaluation assigned successfully", "subm_uuid", submUuid, "eval_uuid", evalUuid)
	return nil
}

// GetSubm retrieves a SubmissionEntity by UUID
func (r *pgSubmRepo) GetSubm(ctx context.Context, id uuid.UUID) (domain.Subm, error) {
	log := logger.FromContext(ctx)
	log.Debug("getting submission", "subm_uuid", id)

	submissionQuery := `
		SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		FROM submissions
		WHERE uuid = $1
	`
	log.Debug("executing query", "query", submissionQuery)

	var s domain.Subm
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
			log.Debug("submission not found", "subm_uuid", id)
			return domain.Subm{}, fmt.Errorf("submission not found: %w", err)
		}
		log.Debug("failed to query submission", "error", err)
		return domain.Subm{}, fmt.Errorf("failed to query submission: %w", err)
	}

	log.Debug("submission retrieved successfully", "subm_uuid", id)
	return s, nil
}

// ListSubms retrieves all SubmissionEntities from the database
func (r *pgSubmRepo) ListSubms(ctx context.Context, limit int, offset int) ([]domain.Subm, error) {
	log := logger.FromContext(ctx)
	log.Debug("executing ListSubms query", "limit", limit, "offset", offset)

	submissionsQuery := `
			SELECT uuid, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
			FROM submissions
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
	`

	submissionQueryPretty := strings.ReplaceAll(submissionsQuery, "\t", " ")
	for strings.Contains(submissionQueryPretty, "  ") {
		submissionQueryPretty = strings.ReplaceAll(submissionQueryPretty, "  ", " ")
	}
	log.Debug("running SQL query", "query", submissionQueryPretty)

	rows, err := r.pool.Query(ctx, submissionsQuery, limit, offset)
	if err != nil {
		log.Debug("failed to query submissions", "error", err)
		return nil, fmt.Errorf("failed to query submissions: %w", err)
	}
	defer rows.Close()

	var submissions []domain.Subm
	for rows.Next() {
		var subm domain.Subm
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
			log.Debug("failed to scan submission", "error", err)
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, subm)
	}

	if err := rows.Err(); err != nil {
		log.Debug("error iterating submissions", "error", err)
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}

	log.Debug("successfully retrieved submissions", "count", len(submissions))
	return submissions, nil
}

// CountSubms returns the total number of submissions in the database
func (r *pgSubmRepo) CountSubms(ctx context.Context) (int, error) {
	log := logger.FromContext(ctx)
	log.Debug("executing CountSubms query")

	countQuery := `SELECT COUNT(*) FROM submissions`

	var count int
	err := r.pool.QueryRow(ctx, countQuery).Scan(&count)
	if err != nil {
		log.Debug("failed to count submissions", "error", err)
		return 0, fmt.Errorf("failed to count submissions: %w", err)
	}

	log.Debug("counted submissions", "count", count)
	return count, nil
}
