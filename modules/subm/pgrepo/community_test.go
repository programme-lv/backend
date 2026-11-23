//go:build integration

package pgrepo

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/common/testutil"
	"github.com/programme-lv/backend/modules/subm/srvc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The community stats query has one representative result per (task, user).
// These tests cover each clause of that classification against a real,
// isolated Postgres database from testutil.

type communityTest struct {
	db *pgxpool.Pool
}

func newCommunityTest(t *testing.T) *communityTest {
	t.Helper()
	return &communityTest{db: testutil.MustGetMigratedTestPostgresDb(t)}
}

func (c *communityTest) user(t *testing.T, username string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := c.db.Exec(context.Background(), `
		INSERT INTO users (uuid, firstname, lastname, username, email, bcrypt_pwd)
		VALUES ($1, 'Test', 'User', $2, $3, '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX')
	`, id, username, fmt.Sprintf("%s-%s@example.com", username, id))
	require.NoError(t, err)
	return id
}

type communityEvalSpec struct {
	stage string

	errorType    *string
	errorMessage *string

	// Tenths are the current score columns. When both are nil, the legacy
	// integer columns are used instead, as the production query does.
	receivedTenths *int
	possibleTenths *int

	receivedScore *int
	possibleScore *int

	createdAt time.Time
	tests     []communityTestResult
}

type communityTestResult struct {
	ac       bool
	wa       bool
	tle      bool
	mle      bool
	re       bool
	ig       bool
	reached  bool
	finished bool
	example  bool
	score    int
}

func (c *communityTest) eval(t *testing.T, submUUID uuid.UUID, spec communityEvalSpec) uuid.UUID {
	t.Helper()
	evalUUID := uuid.New()
	createdAt := spec.createdAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	_, err := c.db.Exec(context.Background(), `
		INSERT INTO evaluations (
			uuid, subm_uuid, stage, score_unit, checker, interactor,
			cpu_lim_ms, mem_lim_kib, error_type, error_message, created_at,
			received_score, possible_score, received_score_tenths, possible_score_tenths
		) VALUES (
			$1, $2, $3, 'test', 'diff', NULL,
			1000, 262144, $4, $5, $6, $7, $8, $9, $10
		)
	`, evalUUID, submUUID, spec.stage, spec.errorType, spec.errorMessage, createdAt,
		spec.receivedScore, spec.possibleScore, spec.receivedTenths, spec.possibleTenths)
	require.NoError(t, err)

	for i, result := range spec.tests {
		_, err = c.db.Exec(context.Background(), `
			INSERT INTO eval_test_results (
				evaluation_uuid, test_id, ac, wa, tle, mle, re, ig,
				reached, finished, score, example
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`, evalUUID, i+1, result.ac, result.wa, result.tle, result.mle,
			result.re, result.ig, result.reached, result.finished, result.score, result.example)
		require.NoError(t, err)
	}
	return evalUUID
}

var communityShortIDCounter int64

func (c *communityTest) subm(
	t *testing.T,
	submUUID uuid.UUID,
	author uuid.UUID,
	task string,
	evalUUID uuid.UUID,
	createdAt time.Time,
) {
	t.Helper()
	id := atomic.AddInt64(&communityShortIDCounter, 1)
	shortID := fmt.Sprintf("%06d", id)
	_, err := c.db.Exec(context.Background(), `
		INSERT INTO submissions (
			uuid, short_id, content, author_uuid, task_shortid,
			lang_shortid, curr_eval_uuid, created_at
		) VALUES ($1, $2, 'code', $3, $4, 'cpp17', $5, $6)
	`, submUUID, shortID, author, task, evalUUID, createdAt)
	require.NoError(t, err)
}

func communityInt(v int) *int       { return &v }
func communityPtr(v string) *string { return &v }

func communityFinished(receivedTenths, possibleTenths int, tests ...communityTestResult) communityEvalSpec {
	return communityEvalSpec{
		stage:          "finished",
		receivedTenths: communityInt(receivedTenths),
		possibleTenths: communityInt(possibleTenths),
		tests:          tests,
	}
}

func communityFullScore() communityTestResult {
	return communityTestResult{ac: true, reached: true, finished: true, score: 100}
}

func communityWrongAnswer() communityTestResult {
	return communityTestResult{wa: true, reached: true, finished: true}
}

func assertCommunityStats(
	t *testing.T,
	stats map[string]srvc.TaskCommunityStats,
	task string,
	solvers, attempted int,
	verdicts map[string]int,
) {
	t.Helper()
	got, ok := stats[task]
	require.Truef(t, ok, "expected stats for task %s", task)
	assert.Equal(t, solvers, got.UniqueSolvers, "unique_solvers")
	assert.Equal(t, attempted, got.UniqueAttempted, "unique_attempted")
	assert.Equal(t, verdicts, got.Verdicts, "verdicts")
	sum := 0
	for _, count := range got.Verdicts {
		sum += count
	}
	assert.Equal(t, got.UniqueAttempted, sum, "verdict sum must equal unique_attempted")
}

func TestCommunityStats_RepeatedAttemptsAndAcceptedEver(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	author := c.user(t, "solver")
	now := time.Now()

	firstEval := c.eval(t, uuid.New(), communityFinished(100, 100, communityFullScore()))
	c.subm(t, uuid.New(), author, "task-a", firstEval, now.Add(-time.Hour))

	// A later wrong answer does not take away an earlier accepted result.
	secondEval := c.eval(t, uuid.New(), communityFinished(0, 100, communityWrongAnswer()))
	c.subm(t, uuid.New(), author, "task-a", secondEval, now)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 1, 1, map[string]int{
		srvc.CommunityVerdictAccepted: 1,
	})
}

func TestCommunityStats_LatestCompletedIgnoresPending(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()
	author := c.user(t, "pending-user")
	finishedEval := c.eval(t, uuid.New(), communityFinished(0, 100, communityWrongAnswer()))
	c.subm(t, uuid.New(), author, "task-a", finishedEval, now.Add(-time.Hour))

	pendingEval := c.eval(t, uuid.New(), communityEvalSpec{stage: "testing"})
	c.subm(t, uuid.New(), author, "task-a", pendingEval, now)

	// A user with only pending work is not an attempt.
	pendingOnly := c.user(t, "pending-only")
	pendingOnlyEval := c.eval(t, uuid.New(), communityEvalSpec{stage: "waiting"})
	c.subm(t, uuid.New(), pendingOnly, "task-a", pendingOnlyEval, now)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 0, 1, map[string]int{
		srvc.CommunityVerdictWrongAnswer: 1,
	})
}

func TestCommunityStats_LatestUsesSubmissionOrder(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()
	author := c.user(t, "order-user")

	// The earlier submission has the later evaluation finish time. The
	// submission order must win.
	earlierEval := c.eval(t, uuid.New(), communityEvalSpec{
		stage:          "finished",
		receivedTenths: communityInt(0),
		possibleTenths: communityInt(100),
		createdAt:      now.Add(2 * time.Hour),
		tests:          []communityTestResult{communityWrongAnswer()},
	})
	c.subm(t, uuid.New(), author, "task-a", earlierEval, now.Add(-time.Hour))

	laterEval := c.eval(t, uuid.New(), communityEvalSpec{
		stage:          "finished",
		receivedTenths: communityInt(0),
		possibleTenths: communityInt(100),
		createdAt:      now.Add(-2 * time.Hour),
		tests:          []communityTestResult{{re: true, reached: true, finished: true}},
	})
	c.subm(t, uuid.New(), author, "task-a", laterEval, now)

	// Equal submission times fall back to submission uuid, descending.
	firstUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	secondUUID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	tieAuthor := c.user(t, "tie-user")
	tiedAt := now.Add(24 * time.Hour)
	tieWA := c.eval(t, firstUUID, communityEvalSpec{
		stage:          "finished",
		receivedTenths: communityInt(0),
		possibleTenths: communityInt(100),
		tests:          []communityTestResult{communityWrongAnswer()},
	})
	c.subm(t, firstUUID, tieAuthor, "task-a", tieWA, tiedAt)
	tieTLE := c.eval(t, secondUUID, communityEvalSpec{
		stage:          "finished",
		receivedTenths: communityInt(0),
		possibleTenths: communityInt(100),
		tests:          []communityTestResult{{tle: true, reached: true, finished: true}},
	})
	c.subm(t, secondUUID, tieAuthor, "task-a", tieTLE, tiedAt)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 0, 2, map[string]int{
		srvc.CommunityVerdictRuntimeError:      1,
		srvc.CommunityVerdictTimeLimitExceeded: 1,
	})
}

func TestCommunityStats_CurrentRejudgeOnly(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()

	// Historical full score is ignored: only s.curr_eval_uuid counts.
	rejudged := c.user(t, "rejudged-down")
	submUUID := uuid.New()
	_ = c.eval(t, submUUID, communityFinished(100, 100, communityFullScore()))
	currentEval := c.eval(t, submUUID, communityFinished(0, 100, communityWrongAnswer()))
	c.subm(t, submUUID, rejudged, "task-a", currentEval, now)

	// The opposite direction: the stale evaluation failed, the current one
	// is a fresh full score.
	fixed := c.user(t, "rejudged-up")
	submUUID2 := uuid.New()
	_ = c.eval(t, submUUID2, communityFinished(0, 100, communityWrongAnswer()))
	currentEval2 := c.eval(t, submUUID2, communityFinished(100, 100, communityFullScore()))
	c.subm(t, submUUID2, fixed, "task-a", currentEval2, now.Add(-time.Hour))

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 1, 2, map[string]int{
		srvc.CommunityVerdictAccepted:    1,
		srvc.CommunityVerdictWrongAnswer: 1,
	})
}

func TestCommunityStats_PartialTenthsAndLegacyFallback(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()

	// 9.9 / 10 is partial, not accepted.
	nearFull := c.user(t, "near-full")
	c.subm(t, uuid.New(), nearFull, "task-a",
		c.eval(t, uuid.New(), communityFinished(99, 100)), now)

	// 0.1 rounds to an integer received score of 0 but is still partial.
	tiny := c.user(t, "tiny")
	c.subm(t, uuid.New(), tiny, "task-a",
		c.eval(t, uuid.New(), communityFinished(1, 100)), now)

	// Legacy rows without tenths use received_score/possible_score * 10.
	legacyPartial := c.user(t, "legacy-partial")
	c.subm(t, uuid.New(), legacyPartial, "task-a", c.eval(t, uuid.New(), communityEvalSpec{
		stage:         "finished",
		receivedScore: communityInt(5),
		possibleScore: communityInt(10),
	}), now)

	legacyFull := c.user(t, "legacy-full")
	c.subm(t, uuid.New(), legacyFull, "task-a", c.eval(t, uuid.New(), communityEvalSpec{
		stage:         "finished",
		receivedScore: communityInt(10),
		possibleScore: communityInt(10),
	}), now)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 1, 4, map[string]int{
		srvc.CommunityVerdictAccepted:         1,
		srvc.CommunityVerdictPartiallyCorrect: 3,
	})
}

func TestCommunityStats_TestFlagPrecedence(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()
	withFlags := func(author string, flags communityTestResult) {
		flags.reached = true
		flags.finished = true
		evalUUID := c.eval(t, uuid.New(), communityFinished(0, 100, flags))
		c.subm(t, uuid.New(), c.user(t, author), "task-a", evalUUID, now)
	}

	withFlags("runtime", communityTestResult{re: true, mle: true, tle: true, wa: true})
	withFlags("memory", communityTestResult{mle: true, tle: true, wa: true})
	withFlags("time", communityTestResult{tle: true, wa: true})
	withFlags("wrong", communityTestResult{wa: true})

	partial := c.user(t, "partial-test")
	partialEval := c.eval(t, uuid.New(), communityFinished(0, 100,
		communityTestResult{reached: true, finished: true, score: 50}))
	c.subm(t, uuid.New(), partial, "task-a", partialEval, now)

	unknown := c.user(t, "no-terminal")
	unknownEval := c.eval(t, uuid.New(), communityFinished(0, 100))
	c.subm(t, uuid.New(), unknown, "task-a", unknownEval, now)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 0, 6, map[string]int{
		srvc.CommunityVerdictRuntimeError:        1,
		srvc.CommunityVerdictMemoryLimitExceeded: 1,
		srvc.CommunityVerdictTimeLimitExceeded:   1,
		srvc.CommunityVerdictWrongAnswer:         1,
		srvc.CommunityVerdictPartiallyCorrect:    1,
		srvc.CommunityVerdictUnknown:             1,
	})
}

func TestCommunityStats_ErrorsAndExamples(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()
	withError := func(author, errorType string) {
		evalUUID := c.eval(t, uuid.New(), communityEvalSpec{
			stage:          "finished",
			errorType:      communityPtr(errorType),
			errorMessage:   communityPtr("boom"),
			receivedTenths: communityInt(0),
			possibleTenths: communityInt(100),
			tests:          []communityTestResult{{wa: true, reached: true, finished: true}},
		})
		c.subm(t, uuid.New(), c.user(t, author), "task-a", evalUUID, now)
	}

	withError("compile", "compilation")
	withError("internal", "internal")
	withError("unavailable", "tester_unavailable")
	withError("other-error", "something_else")

	// Example and ignored rows must not supply a terminal verdict or a
	// partial score.
	examplesOnly := c.user(t, "examples-only")
	examplesEval := c.eval(t, uuid.New(), communityFinished(0, 100,
		communityTestResult{example: true, re: true, reached: true, finished: true, score: 100},
		communityTestResult{ig: true, wa: true, reached: true, finished: true, score: 100},
	))
	c.subm(t, uuid.New(), examplesOnly, "task-a", examplesEval, now)

	// Full score requires a positive possible score. 0/0 is not a solve,
	// even when a test row claims acceptance.
	zeroPossible := c.user(t, "zero-possible")
	zeroEval := c.eval(t, uuid.New(), communityFinished(0, 0, communityFullScore()))
	c.subm(t, uuid.New(), zeroPossible, "task-a", zeroEval, now)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assertCommunityStats(t, stats, "task-a", 0, 6, map[string]int{
		srvc.CommunityVerdictCompileError:      1,
		srvc.CommunityVerdictInternalError:     1,
		srvc.CommunityVerdictTesterUnavailable: 1,
		srvc.CommunityVerdictUnknown:           3,
	})
}

func TestCommunityStats_MultipleTasksFilteringAndAdmin(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	now := time.Now()
	admin := c.user(t, "admin")
	regular := c.user(t, "regular")
	other := c.user(t, "other")
	deleted := c.user(t, "deleted")

	adminFull := c.eval(t, uuid.New(), communityFinished(100, 100, communityFullScore()))
	c.subm(t, uuid.New(), admin, "task-a", adminFull, now)

	regularWA := c.eval(t, uuid.New(), communityFinished(0, 100, communityWrongAnswer()))
	c.subm(t, uuid.New(), regular, "task-a", regularWA, now)

	otherFull := c.eval(t, uuid.New(), communityFinished(100, 100, communityFullScore()))
	c.subm(t, uuid.New(), other, "task-b", otherFull, now)

	deletedFull := c.eval(t, uuid.New(), communityFinished(100, 100, communityFullScore()))
	c.subm(t, uuid.New(), deleted, "task-a", deletedFull, now)
	_, err := c.db.Exec(ctx, `DELETE FROM users WHERE uuid = $1`, deleted)
	require.NoError(t, err)

	pending := c.eval(t, uuid.New(), communityEvalSpec{stage: "compiling"})
	c.subm(t, uuid.New(), regular, "task-c", pending, now)

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a", "task-b", "task-c", "task-missing"})
	require.NoError(t, err)
	require.Len(t, stats, 2)
	assertCommunityStats(t, stats, "task-a", 0, 1, map[string]int{
		srvc.CommunityVerdictWrongAnswer: 1,
	})
	assertCommunityStats(t, stats, "task-b", 1, 1, map[string]int{
		srvc.CommunityVerdictAccepted: 1,
	})

	filtered, err := repo.TaskCommunityStats(ctx, []string{"task-b"})
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assertCommunityStats(t, filtered, "task-b", 1, 1, map[string]int{
		srvc.CommunityVerdictAccepted: 1,
	})

	empty, err := repo.TaskCommunityStats(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

func TestCommunityStats_NoCompletedAttempts(t *testing.T) {
	c := newCommunityTest(t)
	ctx := context.Background()
	repo := NewPgCommunityStatsRepo(c.db)

	author := c.user(t, "waiting")
	waitingEval := c.eval(t, uuid.New(), communityEvalSpec{stage: "waiting"})
	c.subm(t, uuid.New(), author, "task-a", waitingEval, time.Now())

	stats, err := repo.TaskCommunityStats(ctx, []string{"task-a"})
	require.NoError(t, err)
	assert.Empty(t, stats)
}
