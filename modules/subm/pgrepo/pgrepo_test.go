//go:build integration

package pgrepo

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/programme-lv/backend/common/testutil"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var existingAuthorUuid = uuid.New()
var existingEvalUuid = uuid.New()

func newSampleDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	db := testutil.MustGetMigratedTestPostgresDb(t)
	ctx := context.Background()
	_, err := db.Exec(ctx, `
		INSERT INTO users (
			uuid, firstname, lastname, username, email, bcrypt_pwd
		) VALUES (
			$1, 'Test', 'User', 'testuser', 'test@example.com', '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
		)
	`, existingAuthorUuid)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		INSERT INTO evaluations (
			uuid, stage, score_unit, checker, interactor,
			cpu_lim_ms, mem_lim_kib, error_type, error_message, created_at
		) VALUES (
			$1, 'finished', 'test', 'diff', NULL,
			1000, 262144, NULL, NULL, NOW()
		)
	`, existingEvalUuid)
	require.NoError(t, err)
	return db
}

func sampleSubmWithoutEval() domain.Subm {
	return domain.Subm{
		UUID:         uuid.New(),
		Content:      "Sample submission content",
		AuthorUUID:   existingAuthorUuid,
		TaskShortID:  "task_123",
		LangShortID:  "py_x.y.z",
		CurrEvalUUID: uuid.Nil,
		CreatedAt:    time.Now(),
	}
}

func stringPtr(s string) *string { return &s }

func sampleEval() domain.Eval {
	cpuMs1 := 150
	memKiB1 := 10240
	cpuMs2 := 200
	memKiB2 := 15360
	return domain.Eval{
		UUID:      uuid.New(),
		SubmUUID:  uuid.New(),
		Stage:     domain.EvalStageFinished,
		ScoreUnit: domain.ScoreUnitTest,
		Error:     nil,
		Subtasks: []domain.Subtask{
			{Points: 10, Description: "Sample subtask 1", StTests: []int{1, 2}},
			{Points: 20, Description: "Sample subtask 2", StTests: []int{3, 4}},
		},
		Groups: []domain.TestGroup{
			{Points: 15, Subtasks: []int{1}, TgTests: []int{1, 2}},
			{Points: 25, Subtasks: []int{2}, TgTests: []int{3, 4}},
		},
		Tests: []domain.Test{
			{Ac: true, Reached: true, Finished: true, InpSha256: "input_hash_1", AnsSha256: "answer_hash_1", CpuMs: &cpuMs1, MemKiB: &memKiB1},
			{Ac: true, Reached: true, Finished: true, InpSha256: "input_hash_2", AnsSha256: "answer_hash_2", CpuMs: &cpuMs2, MemKiB: &memKiB2},
			{Wa: true, Reached: true, Finished: true, InpSha256: "input_hash_3", AnsSha256: "answer_hash_3"},
			{Tle: true, Reached: true, Finished: true, InpSha256: "input_hash_4", AnsSha256: "answer_hash_4"},
		},
		Checker:   stringPtr("diff"),
		CpuLimMs:  1000,
		MemLimKiB: 262144,
		CreatedAt: time.Now(),
	}
}

func TestSubmRepo_StoreWithEval_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(newSampleDB(t))

	sample := sampleSubmWithoutEval()
	sample.CurrEvalUUID = existingEvalUuid

	require.NoError(t, repo.StoreSubm(context.Background(), &sample))
	stored, err := repo.GetSubm(context.Background(), sample.UUID)
	require.NoError(t, err)
	require.WithinDuration(t, sample.CreatedAt, stored.CreatedAt, time.Millisecond)
	sample.CreatedAt = time.Time{}
	stored.CreatedAt = time.Time{}
	require.Equal(t, sample, stored)
}

func TestSubmRepo_Get_ValidUUID(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(newSampleDB(t))

	sample := sampleSubmWithoutEval()
	require.NoError(t, repo.StoreSubm(context.Background(), &sample))

	got, err := repo.GetSubm(context.Background(), sample.UUID)
	require.NoError(t, err)
	assert.Equal(t, sample.UUID, got.UUID)
	assert.Equal(t, sample.Content, got.Content)
}

func TestSubmRepo_List_MultipleEntries(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(newSampleDB(t))

	entities := make([]domain.Subm, 5)
	for i := range entities {
		entities[i] = sampleSubmWithoutEval()
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].CreatedAt.After(entities[j].CreatedAt)
	})
	for _, e := range entities {
		require.NoError(t, repo.StoreSubm(context.Background(), &e))
	}

	listed, err := repo.ListSubms(context.Background(), 3, 1, "", nil, []string{}, []string{}, []string{}, true)
	require.NoError(t, err)
	require.Len(t, listed, 3)

	expected := entities[1:4]
	for i, listedOne := range listed {
		require.Equal(t, expected[i].UUID, listedOne.UUID)
		if i > 0 {
			equal := listed[i].CreatedAt.Equal(listed[i-1].CreatedAt)
			before := listed[i].CreatedAt.Before(listed[i-1].CreatedAt)
			require.True(t, equal || before)
		}
	}
}

func TestEvalRepo_Store_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgEvalRepo(newSampleDB(t))

	sample := sampleEval()
	require.NoError(t, repo.StoreEval(context.Background(), sample))

	stored, err := repo.GetEval(context.Background(), sample.UUID)
	require.NoError(t, err)
	require.WithinDuration(t, sample.CreatedAt, stored.CreatedAt, time.Millisecond)
	sample.CreatedAt = time.Time{}
	stored.CreatedAt = time.Time{}

	require.Equal(t, sample.UUID, stored.UUID)
	require.Equal(t, sample.SubmUUID, stored.SubmUUID)
	require.Equal(t, sample.Stage, stored.Stage)
	require.Equal(t, sample.ScoreUnit, stored.ScoreUnit)
	require.Equal(t, sample.CpuLimMs, stored.CpuLimMs)
	require.Equal(t, sample.MemLimKiB, stored.MemLimKiB)
	require.Equal(t, *sample.Checker, *stored.Checker)
	require.Equal(t, len(sample.Subtasks), len(stored.Subtasks))
	require.Equal(t, len(sample.Groups), len(stored.Groups))
	require.Equal(t, len(sample.Tests), len(stored.Tests))
	require.Equal(t, *sample.Tests[0].CpuMs, *stored.Tests[0].CpuMs)
	require.Nil(t, stored.Tests[2].CpuMs)
}

func TestSubmRepo_ListShallowSubmsJoinEval_WithCompleteScoreInfo(t *testing.T) {
	t.Parallel()
	db := newSampleDB(t)
	evalRepo := NewPgEvalRepo(db)
	submRepo := NewPgSubmRepo(db)
	ctx := context.Background()

	subm := sampleSubmWithoutEval()
	require.NoError(t, submRepo.StoreSubm(ctx, &subm))

	eval := sampleEval()
	eval.SubmUUID = subm.UUID
	require.NoError(t, evalRepo.StoreEval(ctx, eval))

	_, err := db.Exec(ctx, `
		UPDATE evaluations
		SET received_score = $1, possible_score = $2, scorebar_green = $3, scorebar_red = $4,
		    scorebar_gray = $5, scorebar_yellow = $6, scorebar_purple = $7,
		    cpu_max_ms = $8, mem_max_kib = $9, exceeded_cpu = $10, exceeded_mem = $11
		WHERE uuid = $12
	`, 50, 100, 50, 30, 15, 5, 0, 1500, 65536, false, false, eval.UUID)
	require.NoError(t, err)
	require.NoError(t, submRepo.AssignEval(ctx, subm.UUID, eval.UUID))

	result, err := submRepo.ListShallowSubmsJoinEval(ctx, &existingAuthorUuid)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, subm.UUID, result[0].Subm.UUID)
	require.Equal(t, eval.UUID, result[0].Eval.UUID)
	require.NotNil(t, result[0].Eval.ScoreInfo)
	require.Equal(t, 50, result[0].Eval.ScoreInfo.ReceivedScore)
	require.Equal(t, 100, result[0].Eval.ScoreInfo.PossibleScore)
	require.Equal(t, 50, result[0].Eval.ScoreInfo.ScoreBar.Green)
}

func TestSubmRepo_GetByShortID(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(newSampleDB(t))
	ctx := context.Background()

	sample := sampleSubmWithoutEval()
	require.NoError(t, repo.StoreSubm(ctx, &sample))
	require.True(t, domain.ValidShortID(sample.ShortID))

	got, err := repo.GetSubmByShortID(ctx, sample.ShortID)
	require.NoError(t, err)
	assert.Equal(t, sample.UUID, got.UUID)
	assert.Equal(t, sample.ShortID, got.ShortID)

	_, err = repo.GetSubmByShortID(ctx, "zzzzzz")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestSubmRepo_StoreSubm_DuplicateShortID(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(newSampleDB(t))
	ctx := context.Background()

	first := sampleSubmWithoutEval()
	first.ShortID = "cccccc"
	require.NoError(t, repo.StoreSubm(ctx, &first))

	second := sampleSubmWithoutEval()
	second.ShortID = "cccccc"
	err := repo.StoreSubm(ctx, &second)
	require.ErrorIs(t, err, domain.ErrShortIDTaken)
}

func TestSubmRepo_StoreSubm_RetriesGeneratedShortID(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(newSampleDB(t))
	ctx := context.Background()

	first := sampleSubmWithoutEval()
	first.ShortID = "dddddd"
	require.NoError(t, repo.StoreSubm(ctx, &first))

	orig := repo.generateShortID
	t.Cleanup(func() { repo.generateShortID = orig })
	calls := 0
	repo.generateShortID = func() (string, error) {
		calls++
		if calls == 1 {
			return "dddddd", nil
		}
		return "eeeeee", nil
	}

	second := sampleSubmWithoutEval()
	second.ShortID = ""
	require.NoError(t, repo.StoreSubm(ctx, &second))
	assert.Equal(t, "eeeeee", second.ShortID)
	assert.Equal(t, 2, calls)
}
