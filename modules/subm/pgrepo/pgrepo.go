package pgrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/srvc"
)

type pgEvalRepo struct {
	pool *pgxpool.Pool
}

func NewPgEvalRepo(pool *pgxpool.Pool) *pgEvalRepo {
	return &pgEvalRepo{pool: pool}
}

func (r *pgEvalRepo) StoreEval(ctx context.Context, eval domain.Eval) error {

	// Start a transaction
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Calculate score info from evaluation
	scoreInfo := eval.CalculateScore()

	// Upsert Evaluation
	evaluationUpsertQuery := `
		INSERT INTO evaluations (
			uuid, subm_uuid, stage, score_unit, checker, interactor,
			cpu_lim_ms, mem_lim_kib, error_type, error_message, created_at,
			received_score, possible_score, scorebar_green, scorebar_red,
			scorebar_gray, scorebar_yellow, scorebar_purple,
			cpu_max_ms, mem_max_kib, exceeded_cpu, exceeded_mem
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
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
			created_at = EXCLUDED.created_at,
			received_score = EXCLUDED.received_score,
			possible_score = EXCLUDED.possible_score,
			scorebar_green = EXCLUDED.scorebar_green,
			scorebar_red = EXCLUDED.scorebar_red,
			scorebar_gray = EXCLUDED.scorebar_gray,
			scorebar_yellow = EXCLUDED.scorebar_yellow,
			scorebar_purple = EXCLUDED.scorebar_purple,
			cpu_max_ms = EXCLUDED.cpu_max_ms,
			mem_max_kib = EXCLUDED.mem_max_kib,
			exceeded_cpu = EXCLUDED.exceeded_cpu,
			exceeded_mem = EXCLUDED.exceeded_mem
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
		scoreInfo.ReceivedScore,
		scoreInfo.PossibleScore,
		scoreInfo.ScoreBar.Green,
		scoreInfo.ScoreBar.Red,
		scoreInfo.ScoreBar.Gray,
		scoreInfo.ScoreBar.Yellow,
		scoreInfo.ScoreBar.Purple,
		scoreInfo.MaxCpuMs,
		scoreInfo.MaxMemKiB,
		scoreInfo.ExceededCpu,
		scoreInfo.ExceededMem,
	)
	if err != nil {
		return fmt.Errorf("upsert evaluation: %w", err)
	}

	// Delete existing related data first to avoid duplicates
	deleteQueries := []string{
		`DELETE FROM eval_subtasks WHERE evaluation_uuid = $1`,
		`DELETE FROM eval_test_groups WHERE evaluation_uuid = $1`,
		`DELETE FROM eval_test_results WHERE evaluation_uuid = $1`,
	}
	for _, query := range deleteQueries {
		_, err = tx.Exec(ctx, query, eval.UUID)
		if err != nil {
			return fmt.Errorf("delete existing data: %w", err)
		}
	}

	// Insert Subtasks
	for _, subtask := range eval.Subtasks {
		subtaskInsertQuery := `
			INSERT INTO eval_subtasks (
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
			return fmt.Errorf("insert subtask: %w", err)
		}
	}

	// Insert TestGroups
	for _, group := range eval.Groups {
		groupInsertQuery := `
			INSERT INTO eval_test_groups (
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
			return fmt.Errorf("insert test group: %w", err)
		}
	}

	// Insert Tests
	for i, test := range eval.Tests {
		testInsertQuery := `
			INSERT INTO eval_test_results (
				evaluation_uuid, test_id, ac, wa, tle, mle, re, ig, reached, finished,
				inp_sha256, ans_sha256, cpu_ms, mem_kib
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		`
		_, err = tx.Exec(ctx, testInsertQuery,
			eval.UUID,
			i+1,
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
			test.CpuMs,
			test.MemKiB,
		)
		if err != nil {
			return fmt.Errorf("insert test: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *pgEvalRepo) GetEval(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error) {

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
			return domain.Eval{}, fmt.Errorf("evaluation not found: %w", err)
		}
		return domain.Eval{}, fmt.Errorf("query evaluation: %w", err)
	}

	// Handle EvaluationError
	if errorType != nil {
		et := domain.EvalErrorType(*errorType)
		eval.Error = &domain.EvalError{
			Type:    et,
			Message: errorMessage,
		}
	}

	// Fetch Subtasks
	subtasksQuery := `
		SELECT points, description, st_tests
		FROM eval_subtasks
		WHERE evaluation_uuid = $1
	`
	subtaskRows, err := r.pool.Query(ctx, subtasksQuery, evalUUID)
	if err != nil {
		return domain.Eval{}, fmt.Errorf("query subtasks: %w", err)
	}
	defer subtaskRows.Close()

	for subtaskRows.Next() {
		var st domain.Subtask
		err := subtaskRows.Scan(&st.Points, &st.Description, &st.StTests)
		if err != nil {
			return domain.Eval{}, fmt.Errorf("scan subtask: %w", err)
		}
		eval.Subtasks = append(eval.Subtasks, st)
	}
	if err := subtaskRows.Err(); err != nil {
		return domain.Eval{}, fmt.Errorf("error iterating subtasks: %w", err)
	}

	// Fetch TestGroups
	testGroupsQuery := `
		SELECT points, subtasks, tg_tests
		FROM eval_test_groups
		WHERE evaluation_uuid = $1
	`
	groupRows, err := r.pool.Query(ctx, testGroupsQuery, evalUUID)
	if err != nil {
		return domain.Eval{}, fmt.Errorf("query test groups: %w", err)
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var tg domain.TestGroup
		err := groupRows.Scan(&tg.Points, &tg.Subtasks, &tg.TgTests)
		if err != nil {
			return domain.Eval{}, fmt.Errorf("scan test group: %w", err)
		}
		eval.Groups = append(eval.Groups, tg)
	}
	if err := groupRows.Err(); err != nil {
		return domain.Eval{}, fmt.Errorf("error iterating test groups: %w", err)
	}

	// Fetch Tests
	testsQuery := `
		SELECT ac, wa, tle, mle, re, ig, reached, finished, inp_sha256, ans_sha256, cpu_ms, mem_kib
		FROM eval_test_results
		WHERE evaluation_uuid = $1
		ORDER BY id ASC
	`
	testRows, err := r.pool.Query(ctx, testsQuery, evalUUID)
	if err != nil {
		return domain.Eval{}, fmt.Errorf("query tests: %w", err)
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
			&test.CpuMs,
			&test.MemKiB,
		)
		if err != nil {
			return domain.Eval{}, fmt.Errorf("scan test: %w", err)
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
		return domain.Eval{}, fmt.Errorf("error iterating tests: %w", err)
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
	pool            *pgxpool.Pool
	generateShortID func() (string, error)
}

func NewPgSubmRepo(pool *pgxpool.Pool) *pgSubmRepo {
	return &pgSubmRepo{pool: pool, generateShortID: domain.RandomShortID}
}

const maxShortIDAttempts = 8

const submSelectCols = `uuid, short_id, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at`

func scanSubm(row interface{ Scan(dest ...any) error }) (domain.Subm, error) {
	var s domain.Subm
	err := row.Scan(
		&s.UUID,
		&s.ShortID,
		&s.Content,
		&s.AuthorUUID,
		&s.TaskShortID,
		&s.LangShortID,
		&s.CurrEvalUUID,
		&s.CreatedAt,
	)
	return s, err
}

func isShortIDTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "submissions_short_id_key"
}

// StoreSubm inserts a new submission. If ShortID is empty, a random one is generated
// and written back onto subm. Unique short_id collisions are retried.
func (r *pgSubmRepo) StoreSubm(ctx context.Context, subm *domain.Subm) error {

	provided := subm.ShortID != ""
	for attempt := 0; attempt < maxShortIDAttempts; attempt++ {
		if !provided {
			id, genErr := r.generateShortID()
			if genErr != nil {
				return fmt.Errorf("generate short id: %w", genErr)
			}
			subm.ShortID = id
		}

		err := r.insertSubm(ctx, *subm)
		if err == nil {
			return nil
		}
		if isShortIDTaken(err) {
			if provided {
				return domain.ErrShortIDTaken
			}
			continue
		}
		return fmt.Errorf("insert submission: %w", err)
	}

	return fmt.Errorf("insert submission: short id collisions")
}

func (r *pgSubmRepo) insertSubm(ctx context.Context, subm domain.Subm) error {
	submissionInsertQuery := `
		INSERT INTO submissions (
			uuid, short_id, content, author_uuid, task_shortid, lang_shortid, curr_eval_uuid, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	var currEvalId *uuid.UUID
	if subm.CurrEvalUUID != uuid.Nil {
		currEvalId = &subm.CurrEvalUUID
	}

	_, err := r.pool.Exec(ctx, submissionInsertQuery,
		subm.UUID,
		subm.ShortID,
		subm.Content,
		subm.AuthorUUID,
		subm.TaskShortID,
		subm.LangShortID,
		currEvalId,
		subm.CreatedAt,
	)
	return err
}

func (r *pgSubmRepo) AssignEval(ctx context.Context, submUuid uuid.UUID, evalUuid uuid.UUID) error {

	updateQuery := `
		UPDATE submissions 
		SET curr_eval_uuid = $1
		WHERE uuid = $2
	`

	_, err := r.pool.Exec(ctx, updateQuery, evalUuid, submUuid)
	if err != nil {
		return fmt.Errorf("assign evaluation to submission: %w", err)
	}

	return nil
}

// GetSubm retrieves a SubmissionEntity by UUID
func (r *pgSubmRepo) GetSubm(ctx context.Context, id uuid.UUID) (domain.Subm, error) {

	submissionQuery := `SELECT ` + submSelectCols + `
		FROM submissions
		WHERE uuid = $1
	`

	s, err := scanSubm(r.pool.QueryRow(ctx, submissionQuery, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subm{}, domain.ErrNotFound
		}
		return domain.Subm{}, fmt.Errorf("query submission: %w", err)
	}

	return s, nil
}

func (r *pgSubmRepo) GetSubmByShortID(ctx context.Context, shortID string) (domain.Subm, error) {

	submissionQuery := `SELECT ` + submSelectCols + `
		FROM submissions
		WHERE short_id = $1
	`

	s, err := scanSubm(r.pool.QueryRow(ctx, submissionQuery, shortID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Subm{}, domain.ErrNotFound
		}
		return domain.Subm{}, fmt.Errorf("query submission: %w", err)
	}

	return s, nil
}

// ListSubms retrieves all SubmissionEntities from the database
func (r *pgSubmRepo) ListSubms(ctx context.Context, limit int, offset int, search string, authorUuid *uuid.UUID, taskShortID string, authorIds []string, taskIds []string, langIds []string, includeAdmin bool) ([]domain.Subm, error) {

	var rows pgx.Rows
	var err error

	// Build WHERE conditions based on filters
	hasFilters := authorUuid != nil || taskShortID != "" || len(authorIds) > 0 || len(taskIds) > 0 || len(langIds) > 0 || !includeAdmin

	if hasFilters {
		var conditions []string
		var args []interface{}
		argIndex := 1

		// Add search-based filters if provided (these are OR'd together)
		var searchConditions []string
		if len(authorIds) > 0 {
			searchConditions = append(searchConditions, fmt.Sprintf("s.author_uuid=any($%d)", argIndex))
			args = append(args, pq.Array(authorIds))
			argIndex++
		}
		if len(taskIds) > 0 {
			searchConditions = append(searchConditions, fmt.Sprintf("s.task_shortid=any($%d)", argIndex))
			args = append(args, pq.Array(taskIds))
			argIndex++
		}
		if len(langIds) > 0 {
			searchConditions = append(searchConditions, fmt.Sprintf("s.lang_shortid=any($%d)", argIndex))
			args = append(args, pq.Array(langIds))
			argIndex++
		}

		if len(searchConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(searchConditions, " OR ")+")")
		}

		if authorUuid != nil {
			conditions = append(conditions, fmt.Sprintf("s.author_uuid = $%d", argIndex))
			args = append(args, *authorUuid)
			argIndex++
		}

		if taskShortID != "" {
			conditions = append(conditions, fmt.Sprintf("s.task_shortid = $%d", argIndex))
			args = append(args, taskShortID)
			argIndex++
		}

		// Exclude admin submissions unless includeAdmin is true
		if !includeAdmin {
			conditions = append(conditions, "u.username != 'admin'")
		}

		whereClause := strings.Join(conditions, " AND ")

		submissionsQuery := fmt.Sprintf(`SELECT s.uuid, s.short_id, s.content, s.author_uuid, s.task_shortid, s.lang_shortid, s.curr_eval_uuid, s.created_at
			FROM submissions s
			INNER JOIN users u ON s.author_uuid = u.uuid
			WHERE %s
			ORDER BY s.created_at DESC
			LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

		args = append(args, limit, offset)
		rows, err = r.pool.Query(ctx, submissionsQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("query submissions: %w", err)
		}
		defer rows.Close()
	} else {
		submissionsQuery := `SELECT ` + submSelectCols + `
			FROM submissions
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2`

		rows, err = r.pool.Query(ctx, submissionsQuery, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("query submissions: %w", err)
		}
		defer rows.Close()
	}

	var submissions []domain.Subm
	for rows.Next() {
		subm, scanErr := scanSubm(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan submission: %w", scanErr)
		}
		submissions = append(submissions, subm)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions: %w", err)
	}

	return submissions, nil
}

// CountSubms returns the total number of submissions in the database
func (r *pgSubmRepo) CountSubms(ctx context.Context, authorUuid *uuid.UUID, taskShortID string, authorIds []string, taskIds []string, langIds []string, includeAdmin bool) (int, error) {

	var count int

	// Build WHERE conditions based on filters
	hasFilters := authorUuid != nil || taskShortID != "" || len(authorIds) > 0 || len(taskIds) > 0 || len(langIds) > 0 || !includeAdmin

	if hasFilters {
		var conditions []string
		var args []interface{}
		argIndex := 1

		// Add specific author filter if provided
		if authorUuid != nil {
			conditions = append(conditions, fmt.Sprintf("s.author_uuid = $%d", argIndex))
			args = append(args, *authorUuid)
			argIndex++
		}

		if taskShortID != "" {
			conditions = append(conditions, fmt.Sprintf("s.task_shortid = $%d", argIndex))
			args = append(args, taskShortID)
			argIndex++
		}

		// Add search-based filters if provided (these are OR'd together)
		var searchConditions []string
		if len(authorIds) > 0 {
			searchConditions = append(searchConditions, fmt.Sprintf("s.author_uuid=any($%d)", argIndex))
			args = append(args, pq.Array(authorIds))
			argIndex++
		}
		if len(taskIds) > 0 {
			searchConditions = append(searchConditions, fmt.Sprintf("s.task_shortid=any($%d)", argIndex))
			args = append(args, pq.Array(taskIds))
			argIndex++
		}
		if len(langIds) > 0 {
			searchConditions = append(searchConditions, fmt.Sprintf("s.lang_shortid=any($%d)", argIndex))
			args = append(args, pq.Array(langIds))
			argIndex++
		}

		if len(searchConditions) > 0 {
			conditions = append(conditions, "("+strings.Join(searchConditions, " OR ")+")")
		}

		// Exclude admin submissions unless includeAdmin is true
		if !includeAdmin {
			conditions = append(conditions, "u.username != 'admin'")
		}

		whereClause := strings.Join(conditions, " AND ")
		countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM submissions s INNER JOIN users u ON s.author_uuid = u.uuid WHERE %s`, whereClause)

		err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count submissions with filters: %w", err)
		}
		return count, nil
	} else {
		countQuery := `SELECT COUNT(*) FROM submissions`
		err := r.pool.QueryRow(ctx, countQuery).Scan(&count)
		if err != nil {
			return 0, fmt.Errorf("count submissions: %w", err)
		}
	}

	return count, nil
}

func (r *pgSubmRepo) ListShallowSubmsJoinEval(ctx context.Context, authorUuid *uuid.UUID) ([]srvc.ShallowSubmJoinEvalDto, error) {

	query := `
		SELECT 
			s.uuid, s.short_id, s.author_uuid, s.task_shortid, s.lang_shortid, s.curr_eval_uuid, s.created_at,
			e.uuid, e.subm_uuid, e.stage, e.score_unit, e.error_type, e.error_message, 
			e.checker, e.interactor, e.cpu_lim_ms, e.mem_lim_kib, e.created_at,
			e.received_score, e.possible_score, e.scorebar_green, e.scorebar_red, e.scorebar_gray, 
			e.scorebar_yellow, e.scorebar_purple, e.cpu_max_ms, e.mem_max_kib, e.exceeded_cpu, e.exceeded_mem
		FROM submissions s
		INNER JOIN evaluations e ON s.curr_eval_uuid = e.uuid
		WHERE s.author_uuid = $1
		ORDER BY s.created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, authorUuid)
	if err != nil {
		return nil, fmt.Errorf("query submissions with evaluations: %w", err)
	}
	defer rows.Close()

	var result []srvc.ShallowSubmJoinEvalDto
	for rows.Next() {
		var dto srvc.ShallowSubmJoinEvalDto
		var errorType *string
		var errorMessage *string
		var checker *string
		var interactor *string

		// Score-related fields that may be NULL
		var receivedScore *int
		var possibleScore *int
		var scorebarGreen *int
		var scorebarRed *int
		var scorebarGray *int
		var scorebarYellow *int
		var scorebarPurple *int
		var cpuMaxMs *int
		var memMaxKiB *int
		var exceededCpu *bool
		var exceededMem *bool

		err := rows.Scan(
			// Submission fields
			&dto.Subm.UUID,
			&dto.Subm.ShortID,
			&dto.Subm.AuthorUUID,
			&dto.Subm.TaskShortID,
			&dto.Subm.LangShortID,
			&dto.Subm.CurrEvalUUID,
			&dto.Subm.CreatedAt,
			// Evaluation fields - non-NULL with INNER JOIN
			&dto.Eval.UUID,
			&dto.Eval.SubmUUID,
			&dto.Eval.Stage,
			&dto.Eval.ScoreUnit,
			&errorType,
			&errorMessage,
			&checker,
			&interactor,
			&dto.Eval.CpuLimMs,
			&dto.Eval.MemLimKiB,
			&dto.Eval.CreatedAt,
			// Score-related fields
			&receivedScore,
			&possibleScore,
			&scorebarGreen,
			&scorebarRed,
			&scorebarGray,
			&scorebarYellow,
			&scorebarPurple,
			&cpuMaxMs,
			&memMaxKiB,
			&exceededCpu,
			&exceededMem,
		)
		if err != nil {
			return nil, fmt.Errorf("scan submission and evaluation: %w", err)
		}

		// Handle EvaluationError
		if errorType != nil {
			et := domain.EvalErrorType(*errorType)
			dto.Eval.Error = &domain.EvalError{
				Type:    et,
				Message: errorMessage,
			}
		}

		dto.Eval.Checker = checker
		dto.Eval.Interactor = interactor

		// Set ScoreInfo only if all score-related columns are non-NULL
		if receivedScore != nil && possibleScore != nil && scorebarGreen != nil && scorebarRed != nil &&
			scorebarGray != nil && scorebarYellow != nil && scorebarPurple != nil &&
			cpuMaxMs != nil && memMaxKiB != nil && exceededCpu != nil && exceededMem != nil {
			dto.Eval.ScoreInfo = &domain.ScoreInfo{
				ScoreBar: domain.ScoreBarInfo{
					Green:  *scorebarGreen,
					Red:    *scorebarRed,
					Gray:   *scorebarGray,
					Yellow: *scorebarYellow,
					Purple: *scorebarPurple,
				},
				ReceivedScore: *receivedScore,
				PossibleScore: *possibleScore,
				MaxCpuMs:      *cpuMaxMs,
				MaxMemKiB:     *memMaxKiB,
				ExceededCpu:   *exceededCpu,
				ExceededMem:   *exceededMem,
			}
		}

		result = append(result, dto)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating submissions and evaluations: %w", err)
	}

	return result, nil
}
