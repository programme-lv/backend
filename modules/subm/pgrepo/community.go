package pgrepo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"github.com/programme-lv/backend/modules/subm/srvc"
)

// pgCommunityStatsRepo serves read-only aggregate community stats. It never
// loads code or per-test rows into Go: the query returns one row per task.
type pgCommunityStatsRepo struct {
	pool *pgxpool.Pool
}

// NewPgCommunityStatsRepo returns a community stats repository backed by pool.
func NewPgCommunityStatsRepo(pool *pgxpool.Pool) *pgCommunityStatsRepo {
	return &pgCommunityStatsRepo{pool: pool}
}

// communityStatsQuery computes one representative verdict per (task, user).
//
// Only the current evaluation of a submission is read (s.curr_eval_uuid), so a
// rejudge replaces the old result and stale evaluations never count. Only
// finished evaluations count; waiting, compiling and testing submissions are
// ignored even when they are the newest submission for the user. Admin
// submissions are excluded the same way as the public submission listing, and
// the users join excludes rows without a live user (submissions also cascade
// on user deletion).
//
// Per task and user:
//   - accepted if ANY current finished evaluation is error-free and has a full
//     positive score (possible > 0 and received == possible, in tenths with a
//     legacy integer fallback); this wins over later failures or pending work.
//   - otherwise the latest COMPLETED submission represents the user, ordered
//     by submission created_at and then submission uuid, never by evaluation
//     finish time.
//   - that evaluation is classified as: compilation -> compile_error,
//     internal -> internal_error, tester_unavailable -> tester_unavailable,
//     any other error -> unknown; error-free full score -> accepted; positive
//     aggregate score -> partially_correct; otherwise finished scored tests
//     (not examples, not ignored) select runtime_error > memory_limit_exceeded
//     > time_limit_exceeded > wrong_answer; a non-accepted test with a positive
//     partial score -> partially_correct; otherwise unknown.
//
// Counts are aggregated in Postgres, so the caller receives no user-level rows.
const communityStatsQuery = `
WITH target_tasks AS (
    SELECT DISTINCT unnest($1::text[]) AS task_shortid
),
completed AS (
    SELECT
        s.task_shortid,
        s.author_uuid,
        s.uuid AS subm_uuid,
        s.created_at AS subm_created_at,
        e.uuid AS eval_uuid,
        e.error_type,
        COALESCE(e.received_score_tenths, e.received_score * 10) AS received_tenths,
        COALESCE(e.possible_score_tenths, e.possible_score * 10) AS possible_tenths
    FROM submissions s
    INNER JOIN target_tasks t ON t.task_shortid = s.task_shortid
    INNER JOIN users u ON u.uuid = s.author_uuid
    INNER JOIN evaluations e ON e.uuid = s.curr_eval_uuid
    WHERE e.stage = 'finished'
      AND u.username <> 'admin'
),
accepted_users AS (
    SELECT DISTINCT task_shortid, author_uuid
    FROM completed
    WHERE error_type IS NULL
      AND possible_tenths > 0
      AND received_tenths = possible_tenths
),
latest_completed AS (
    SELECT DISTINCT ON (task_shortid, author_uuid)
        task_shortid,
        author_uuid,
        eval_uuid,
        error_type,
        received_tenths,
        possible_tenths
    FROM completed
    ORDER BY task_shortid, author_uuid, subm_created_at DESC, subm_uuid DESC
),
latest_result AS (
    SELECT
        l.task_shortid,
        l.author_uuid,
        l.error_type,
        l.received_tenths,
        l.possible_tenths,
        COALESCE(f.has_re, FALSE) AS has_re,
        COALESCE(f.has_mle, FALSE) AS has_mle,
        COALESCE(f.has_tle, FALSE) AS has_tle,
        COALESCE(f.has_wa, FALSE) AS has_wa,
        COALESCE(f.has_partial, FALSE) AS has_partial
    FROM latest_completed l
    LEFT JOIN LATERAL (
        SELECT
            bool_or(tr.re) AS has_re,
            bool_or(tr.mle) AS has_mle,
            bool_or(tr.tle) AS has_tle,
            bool_or(tr.wa) AS has_wa,
            bool_or(NOT tr.ac AND tr.score > 0) AS has_partial
        FROM eval_test_results tr
        WHERE tr.evaluation_uuid = l.eval_uuid
          AND NOT tr.example
          AND NOT tr.ig
          AND tr.finished
    ) f ON TRUE
),
classified AS (
    SELECT
        lr.task_shortid,
        lr.author_uuid,
        CASE
            WHEN lr.error_type = 'compilation' THEN 'compile_error'
            WHEN lr.error_type = 'internal' THEN 'internal_error'
            WHEN lr.error_type = 'tester_unavailable' THEN 'tester_unavailable'
            WHEN lr.error_type IS NOT NULL THEN 'unknown'
            WHEN lr.possible_tenths > 0 AND lr.received_tenths = lr.possible_tenths THEN 'accepted'
            WHEN lr.received_tenths > 0 THEN 'partially_correct'
            WHEN lr.has_re THEN 'runtime_error'
            WHEN lr.has_mle THEN 'memory_limit_exceeded'
            WHEN lr.has_tle THEN 'time_limit_exceeded'
            WHEN lr.has_wa THEN 'wrong_answer'
            WHEN lr.has_partial THEN 'partially_correct'
            ELSE 'unknown'
        END AS verdict
    FROM latest_result lr
),
user_result AS (
    SELECT
        c.task_shortid,
        CASE WHEN a.author_uuid IS NOT NULL THEN 'accepted' ELSE c.verdict END AS verdict
    FROM classified c
    LEFT JOIN accepted_users a
      ON a.task_shortid = c.task_shortid
     AND a.author_uuid = c.author_uuid
)
SELECT
    task_shortid,
    COUNT(*)::int AS unique_attempted,
    COUNT(*) FILTER (WHERE verdict = 'accepted')::int AS unique_solvers,
    COUNT(*) FILTER (WHERE verdict = 'partially_correct')::int AS partially_correct,
    COUNT(*) FILTER (WHERE verdict = 'wrong_answer')::int AS wrong_answer,
    COUNT(*) FILTER (WHERE verdict = 'time_limit_exceeded')::int AS time_limit_exceeded,
    COUNT(*) FILTER (WHERE verdict = 'memory_limit_exceeded')::int AS memory_limit_exceeded,
    COUNT(*) FILTER (WHERE verdict = 'runtime_error')::int AS runtime_error,
    COUNT(*) FILTER (WHERE verdict = 'compile_error')::int AS compile_error,
    COUNT(*) FILTER (WHERE verdict = 'internal_error')::int AS internal_error,
    COUNT(*) FILTER (WHERE verdict = 'tester_unavailable')::int AS tester_unavailable,
    COUNT(*) FILTER (WHERE verdict = 'unknown')::int AS unknown
FROM user_result
GROUP BY task_shortid
`

// TaskCommunityStats returns per-task aggregate community stats for the given
// task IDs. Tasks without finished attempts are absent from the result.
func (r *pgCommunityStatsRepo) TaskCommunityStats(
	ctx context.Context,
	taskShortIDs []string,
) (map[string]srvc.TaskCommunityStats, error) {
	if len(taskShortIDs) == 0 {
		return map[string]srvc.TaskCommunityStats{}, nil
	}

	rows, err := r.pool.Query(ctx, communityStatsQuery, pq.Array(taskShortIDs))
	if err != nil {
		return nil, fmt.Errorf("query task community stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]srvc.TaskCommunityStats)
	for rows.Next() {
		var (
			taskShortID      string
			uniqueAttempted  int
			uniqueSolvers    int
			partiallyCorrect int
			wrongAnswer      int
			timeLimit        int
			memoryLimit      int
			runtimeError     int
			compileError     int
			internalError    int
			testerUnavail    int
			unknown          int
		)
		if err := rows.Scan(
			&taskShortID,
			&uniqueAttempted,
			&uniqueSolvers,
			&partiallyCorrect,
			&wrongAnswer,
			&timeLimit,
			&memoryLimit,
			&runtimeError,
			&compileError,
			&internalError,
			&testerUnavail,
			&unknown,
		); err != nil {
			return nil, fmt.Errorf("scan task community stats: %w", err)
		}

		verdicts := make(map[string]int, 10)
		addVerdict(verdicts, srvc.CommunityVerdictAccepted, uniqueSolvers)
		addVerdict(verdicts, srvc.CommunityVerdictPartiallyCorrect, partiallyCorrect)
		addVerdict(verdicts, srvc.CommunityVerdictWrongAnswer, wrongAnswer)
		addVerdict(verdicts, srvc.CommunityVerdictTimeLimitExceeded, timeLimit)
		addVerdict(verdicts, srvc.CommunityVerdictMemoryLimitExceeded, memoryLimit)
		addVerdict(verdicts, srvc.CommunityVerdictRuntimeError, runtimeError)
		addVerdict(verdicts, srvc.CommunityVerdictCompileError, compileError)
		addVerdict(verdicts, srvc.CommunityVerdictInternalError, internalError)
		addVerdict(verdicts, srvc.CommunityVerdictTesterUnavailable, testerUnavail)
		addVerdict(verdicts, srvc.CommunityVerdictUnknown, unknown)

		stats[taskShortID] = srvc.TaskCommunityStats{
			UniqueSolvers:   uniqueSolvers,
			UniqueAttempted: uniqueAttempted,
			Verdicts:        verdicts,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task community stats: %w", err)
	}

	return stats, nil
}

func addVerdict(verdicts map[string]int, verdict string, count int) {
	if count > 0 {
		verdicts[verdict] = count
	}
}
