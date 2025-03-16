package pgrepo

import (
	"sort"
	"testing"

	"context"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewDB returns a connection pool to a unique and isolated test database,
// fully migrated and ready for testing
func NewDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "proglv", // local dev pg user
		Password:   "proglv", // local dev pg password
		Host:       "localhost",
		Port:       "5433",
		Options:    "sslmode=disable",
	}
	gm := golangmigrator.New("../../migrate")
	config := pgtestdb.Custom(t, conf, gm)

	pool, err := pgxpool.New(ctx, config.URL())
	if err != nil {
		t.Fatalf("Failed to create connection pool: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

var existingAuthorUuid = uuid.New() // author pre-existing in the db
var existingEvalUuid = uuid.New()   // evaluation pre-existing in the db

// NewSampleDB adds a sample author to result of NewDB
func NewSampleDB(t *testing.T) *pgxpool.Pool {
	// create a sample author in the db
	db := NewDB(t)
	ctx := context.Background()
	_, err := db.Exec(ctx, `
		INSERT INTO users (
			uuid, firstname, lastname, username, email, bcrypt_pwd
		) VALUES (
			$1, 'Test', 'User', 'testuser', 'test@example.com', '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX'
		)
	`, existingAuthorUuid)
	if err != nil {
		t.Fatalf("Failed to create sample author: %v", err)
	}

	// create a sample evaluation in the db
	_, err = db.Exec(ctx, `
		INSERT INTO evaluations (
			uuid, stage, score_unit, checker, interactor,
			cpu_lim_ms, mem_lim_kib, error_type, error_message, created_at
		) VALUES (
			$1, 'finished', 'test', 'diff', NULL,
			1000, 262144, NULL, NULL, NOW()
		)
	`, existingEvalUuid)
	if err != nil {
		t.Fatalf("Failed to create sample evaluation: %v", err)
	}
	return db
}

func TestPgDbSchemaVersion(t *testing.T) {
	t.Parallel()

	db := NewSampleDB(t)

	var version int
	var dirty bool
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := db.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	assert.Nil(t, err)
	assert.Equal(t, 31, version)
	assert.False(t, dirty)
}

// getSampleSubmEntityWithoutEval creates a SubmissionEntity with sample data.
func getSampleSubmEntityWithoutEval() domain.Subm {
	return domain.Subm{
		UUID:         uuid.New(),
		Content:      "Sample submission content",
		AuthorUUID:   existingAuthorUuid, // author must pre-exist in the db
		TaskShortID:  "task_123",
		LangShortID:  "py_x.y.z",
		CurrEvalUUID: uuid.Nil,
		CreatedAt:    time.Now(),
	}
}

// TestSubmRepo_StoreWithoutEval_Success tests successful storage of a SubmissionEntity without an evaluation.
func TestSubmRepo_StoreWithoutEval_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	sampleEntity := getSampleSubmEntityWithoutEval()

	err := repo.StoreSubm(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid SubmissionEntity")

	// Retrieve the stored entity
	storedEntity, err := repo.GetSubm(context.Background(), sampleEntity.UUID)
	require.Nil(t, err, "expected no error when retrieving stored SubmissionEntity")
	require.NotNil(t, storedEntity)

	// compare submission created at with a 1ms precision
	require.WithinDuration(t, sampleEntity.CreatedAt, storedEntity.CreatedAt, 1*time.Millisecond)
	sampleEntity.CreatedAt = time.Time{}
	storedEntity.CreatedAt = time.Time{}

	require.Equal(t, sampleEntity, storedEntity)
}

// TestSubmRepo_StoreWithEval_Success tests successful storage of a SubmissionEntity with an evaluation.
func TestSubmRepo_StoreWithEval_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	sampleEntity := getSampleSubmEntityWithoutEval()
	sampleEntity.CurrEvalUUID = existingEvalUuid

	err := repo.StoreSubm(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid SubmissionEntity")

	// Retrieve the stored entity
	storedEntity, err := repo.GetSubm(context.Background(), sampleEntity.UUID)
	require.Nil(t, err, "expected no error when retrieving stored SubmissionEntity")
	require.NotNil(t, storedEntity)

	// compare submission created at with a 1ms precision
	require.WithinDuration(t, sampleEntity.CreatedAt, storedEntity.CreatedAt, 1*time.Millisecond)
	sampleEntity.CreatedAt = time.Time{}
	storedEntity.CreatedAt = time.Time{}

	require.Equal(t, sampleEntity, storedEntity)
}

// TestSubmRepo_Get_ValidUUID tests retrieving a SubmissionEntity with a valid UUID.
func TestSubmRepo_Get_ValidUUID(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	sampleEntity := getSampleSubmEntityWithoutEval()

	// Store the entity first
	err := repo.StoreSubm(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid SubmissionEntity")

	// Retrieve the entity
	retrievedEntity, err := repo.GetSubm(context.Background(), sampleEntity.UUID)
	assert.Nil(t, err, "expected no error when retrieving SubmissionEntity")
	assert.NotNil(t, retrievedEntity, "expected retrieved SubmissionEntity to be not nil")
	assert.Equal(t, sampleEntity.UUID, retrievedEntity.UUID, "UUIDs should match")
	assert.Equal(t, sampleEntity.Content, retrievedEntity.Content, "Content should match")
	// Add more assertions as needed
}

// TestSubmRepo_Get_InvalidUUID tests retrieving a SubmissionEntity with an invalid UUID.
func TestSubmRepo_Get_InvalidUUID(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	nonExistentUUID := uuid.New()

	retrievedEntity, err := repo.GetSubm(context.Background(), nonExistentUUID)
	assert.NotNil(t, err, "expected error when retrieving SubmissionEntity with non-existent UUID")
	assert.Empty(t, retrievedEntity, "expected retrieved SubmissionEntity to be empty for non-existent UUID")
}

// TestSubmRepo_List_MultipleEntries tests listing multiple SubmissionEntities.
// require that submissions are sorted by created at with the newest first
func TestSubmRepo_List_MultipleEntries(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	// Create and store multiple entities
	numEntries := 5
	entities := make([]domain.Subm, numEntries)
	for i := 0; i < numEntries; i++ {
		entities[i] = getSampleSubmEntityWithoutEval()
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].CreatedAt.After(entities[j].CreatedAt)
	})

	for i := 0; i < numEntries; i++ {
		err := repo.StoreSubm(context.Background(), entities[i])
		require.Nil(t, err, "expected no error when storing SubmissionEntity")
	}

	// List all entities
	listedEntities, err := repo.ListSubms(context.Background(), 3, 1)
	require.Nil(t, err, "expected no error when listing SubmissionEntities")
	require.Len(t, listedEntities, 3, "expected number of listed entities to match stored entries")

	expected := entities[1:4]

	for i, listed := range listedEntities {
		require.Less(t, i, len(expected), "listed entity index should be less than expected number of entities")
		require.Equal(t, expected[i].UUID, listed.UUID, "UUIDs should match")
		require.Equal(t, expected[i].Content, listed.Content, "Content should match for listed entity")
		if i > 0 {
			equal := listedEntities[i].CreatedAt.Equal(listedEntities[i-1].CreatedAt)
			before := listedEntities[i].CreatedAt.Before(listedEntities[i-1].CreatedAt)
			require.True(t, equal || before, "created at should be in descending order")
		}
	}
}

// TestSubmRepo_List_NoEntries tests listing SubmissionEntities when none are stored.
func TestSubmRepo_List_NoEntries(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	// Ensure repository is empty by not storing any entities
	listedEntities, err := repo.ListSubms(context.Background(), 100, 0)
	assert.Nil(t, err, "expected no error when listing with no SubmissionEntities")
	assert.Empty(t, listedEntities, "expected no SubmissionEntities to be listed")
}

// TestSubmRepo_Store_MissingFields tests storing SubmissionEntities with missing required fields.
func TestSubmRepo_Store_MissingFields(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	// Example: Missing Content and AuthorUUID
	invalidEntities := []domain.Subm{
		{
			UUID:        uuid.New(),
			Content:     "",
			AuthorUUID:  uuid.New(),
			TaskShortID: "TASK123",
			LangShortID: "GO",
			CreatedAt:   time.Now(),
		},
		{
			UUID:        uuid.New(),
			Content:     "Valid Content",
			AuthorUUID:  uuid.Nil, // Assuming AuthorUUID cannot be nil
			TaskShortID: "TASK123",
			LangShortID: "GO",
			CreatedAt:   time.Now(),
		},
		// Add more variations as needed
	}

	for _, entity := range invalidEntities {
		err := repo.StoreSubm(context.Background(), entity)
		assert.NotNil(t, err, "expected error when storing SubmissionEntity with missing fields")
	}
}

// getSampleEvalEntity creates an Eval entity with sample data.
func getSampleEvalEntity() domain.Eval {
	submUUID := uuid.New()
	cpuMs1 := 150
	memKiB1 := 10240
	cpuMs2 := 200
	memKiB2 := 15360
	return domain.Eval{
		UUID:      uuid.New(),
		SubmUUID:  submUUID,
		Stage:     domain.EvalStageFinished,
		ScoreUnit: domain.ScoreUnitTest,
		Error:     nil,
		Subtasks: []domain.Subtask{
			{
				Points:      10,
				Description: "Sample subtask 1",
				StTests:     []int{1, 2},
			},
			{
				Points:      20,
				Description: "Sample subtask 2",
				StTests:     []int{3, 4},
			},
		},
		Groups: []domain.TestGroup{
			{
				Points:   15,
				Subtasks: []int{1},
				TgTests:  []int{1, 2},
			},
			{
				Points:   25,
				Subtasks: []int{2},
				TgTests:  []int{3, 4},
			},
		},
		Tests: []domain.Test{
			{
				Ac:        true,
				Wa:        false,
				Tle:       false,
				Mle:       false,
				Re:        false,
				Ig:        false,
				Reached:   true,
				Finished:  true,
				InpSha256: "input_hash_1",
				AnsSha256: "answer_hash_1",
				CpuMs:     &cpuMs1,
				MemKiB:    &memKiB1,
			},
			{
				Ac:        true,
				Wa:        false,
				Tle:       false,
				Mle:       false,
				Re:        false,
				Ig:        false,
				Reached:   true,
				Finished:  true,
				InpSha256: "input_hash_2",
				AnsSha256: "answer_hash_2",
				CpuMs:     &cpuMs2,
				MemKiB:    &memKiB2,
			},
			{
				Ac:        false,
				Wa:        true,
				Tle:       false,
				Mle:       false,
				Re:        false,
				Ig:        false,
				Reached:   true,
				Finished:  true,
				InpSha256: "input_hash_3",
				AnsSha256: "answer_hash_3",
				CpuMs:     nil,
				MemKiB:    nil,
			},
			{
				Ac:        false,
				Wa:        false,
				Tle:       true,
				Mle:       false,
				Re:        false,
				Ig:        false,
				Reached:   true,
				Finished:  true,
				InpSha256: "input_hash_4",
				AnsSha256: "answer_hash_4",
				CpuMs:     nil,
				MemKiB:    nil,
			},
		},
		Checker:    stringPtr("diff"),
		Interactor: nil,
		CpuLimMs:   1000,
		MemLimKiB:  262144,
		CreatedAt:  time.Now(),
	}
}

// Helper function to create a string pointer
func stringPtr(s string) *string {
	return &s
}

// getSampleEvalEntityWithError creates an Eval entity with an error.
func getSampleEvalEntityWithError() domain.Eval {
	eval := getSampleEvalEntity()
	errorMessage := "Compilation error: undefined reference"
	eval.Error = &domain.EvalError{
		Type:    domain.ErrorTypeCompilation,
		Message: &errorMessage,
	}
	return eval
}

// TestEvalRepo_Store_Success tests successful storage of an Eval entity.
func TestEvalRepo_Store_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgEvalRepo(NewSampleDB(t))

	sampleEntity := getSampleEvalEntity()

	err := repo.StoreEval(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid Eval entity")

	// Retrieve the stored entity
	storedEntity, err := repo.GetEval(context.Background(), sampleEntity.UUID)
	require.Nil(t, err, "expected no error when retrieving stored Eval entity")
	require.NotNil(t, storedEntity)

	// Compare created_at with a 1ms precision
	require.WithinDuration(t, sampleEntity.CreatedAt, storedEntity.CreatedAt, 1*time.Millisecond)
	sampleEntity.CreatedAt = time.Time{}
	storedEntity.CreatedAt = time.Time{}

	// Compare the entities
	require.Equal(t, sampleEntity.UUID, storedEntity.UUID)
	require.Equal(t, sampleEntity.SubmUUID, storedEntity.SubmUUID)
	require.Equal(t, sampleEntity.Stage, storedEntity.Stage)
	require.Equal(t, sampleEntity.ScoreUnit, storedEntity.ScoreUnit)
	require.Equal(t, sampleEntity.Error, storedEntity.Error)
	require.Equal(t, sampleEntity.CpuLimMs, storedEntity.CpuLimMs)
	require.Equal(t, sampleEntity.MemLimKiB, storedEntity.MemLimKiB)

	// Compare checker and interactor
	if sampleEntity.Checker == nil {
		require.Nil(t, storedEntity.Checker)
	} else {
		require.NotNil(t, storedEntity.Checker)
		require.Equal(t, *sampleEntity.Checker, *storedEntity.Checker)
	}

	if sampleEntity.Interactor == nil {
		require.Nil(t, storedEntity.Interactor)
	} else {
		require.NotNil(t, storedEntity.Interactor)
		require.Equal(t, *sampleEntity.Interactor, *storedEntity.Interactor)
	}

	// Compare subtasks
	require.Equal(t, len(sampleEntity.Subtasks), len(storedEntity.Subtasks))
	for i, subtask := range sampleEntity.Subtasks {
		require.Equal(t, subtask.Points, storedEntity.Subtasks[i].Points)
		require.Equal(t, subtask.Description, storedEntity.Subtasks[i].Description)
		require.Equal(t, subtask.StTests, storedEntity.Subtasks[i].StTests)
	}

	// Compare test groups
	require.Equal(t, len(sampleEntity.Groups), len(storedEntity.Groups))
	for i, group := range sampleEntity.Groups {
		require.Equal(t, group.Points, storedEntity.Groups[i].Points)
		require.Equal(t, group.Subtasks, storedEntity.Groups[i].Subtasks)
		require.Equal(t, group.TgTests, storedEntity.Groups[i].TgTests)
	}

	// Compare tests
	require.Equal(t, len(sampleEntity.Tests), len(storedEntity.Tests))
	for i, test := range sampleEntity.Tests {
		require.Equal(t, test.Ac, storedEntity.Tests[i].Ac)
		require.Equal(t, test.Wa, storedEntity.Tests[i].Wa)
		require.Equal(t, test.Tle, storedEntity.Tests[i].Tle)
		require.Equal(t, test.Mle, storedEntity.Tests[i].Mle)
		require.Equal(t, test.Re, storedEntity.Tests[i].Re)
		require.Equal(t, test.Ig, storedEntity.Tests[i].Ig)
		require.Equal(t, test.Reached, storedEntity.Tests[i].Reached)
		require.Equal(t, test.Finished, storedEntity.Tests[i].Finished)
		require.Equal(t, test.InpSha256, storedEntity.Tests[i].InpSha256)
		require.Equal(t, test.AnsSha256, storedEntity.Tests[i].AnsSha256)

		// Compare the new fields
		if test.CpuMs == nil {
			require.Nil(t, storedEntity.Tests[i].CpuMs)
		} else {
			require.NotNil(t, storedEntity.Tests[i].CpuMs)
			require.Equal(t, *test.CpuMs, *storedEntity.Tests[i].CpuMs)
		}

		if test.MemKiB == nil {
			require.Nil(t, storedEntity.Tests[i].MemKiB)
		} else {
			require.NotNil(t, storedEntity.Tests[i].MemKiB)
			require.Equal(t, *test.MemKiB, *storedEntity.Tests[i].MemKiB)
		}
	}
}

// TestEvalRepo_StoreWithError_Success tests successful storage of an Eval entity with an error.
func TestEvalRepo_StoreWithError_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgEvalRepo(NewSampleDB(t))

	sampleEntity := getSampleEvalEntityWithError()

	err := repo.StoreEval(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing Eval entity with error")

	// Retrieve the stored entity
	storedEntity, err := repo.GetEval(context.Background(), sampleEntity.UUID)
	require.Nil(t, err, "expected no error when retrieving stored Eval entity")
	require.NotNil(t, storedEntity)

	// Verify error was stored correctly
	require.NotNil(t, storedEntity.Error)
	require.Equal(t, sampleEntity.Error.Type, storedEntity.Error.Type)
	require.NotNil(t, storedEntity.Error.Message)
	require.Equal(t, *sampleEntity.Error.Message, *storedEntity.Error.Message)
}

// TestEvalRepo_StoreWithExecData_Success tests successful storage of an Eval entity with execution data.
func TestEvalRepo_StoreWithExecData_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgEvalRepo(NewSampleDB(t))

	// Create a sample entity with specific execution data
	sampleEntity := getSampleEvalEntity()

	// Set specific values for CPU and memory usage
	cpuMs := 123
	memKiB := 45678

	// Update the first test with these values
	sampleEntity.Tests[0].CpuMs = &cpuMs
	sampleEntity.Tests[0].MemKiB = &memKiB

	// Make sure the second test has nil values to test null handling
	sampleEntity.Tests[1].CpuMs = nil
	sampleEntity.Tests[1].MemKiB = nil

	err := repo.StoreEval(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing Eval entity with execution data")

	// Retrieve the stored entity
	storedEntity, err := repo.GetEval(context.Background(), sampleEntity.UUID)
	require.Nil(t, err, "expected no error when retrieving stored Eval entity")
	require.NotNil(t, storedEntity)

	// Verify execution data was stored correctly
	require.NotNil(t, storedEntity.Tests[0].CpuMs)
	require.Equal(t, cpuMs, *storedEntity.Tests[0].CpuMs)
	require.NotNil(t, storedEntity.Tests[0].MemKiB)
	require.Equal(t, memKiB, *storedEntity.Tests[0].MemKiB)

	// Verify null values were handled correctly
	require.Nil(t, storedEntity.Tests[1].CpuMs)
	require.Nil(t, storedEntity.Tests[1].MemKiB)
}

// TestSubmRepo_Store_MissingFields tests storing SubmissionEntities with missing required fields.
