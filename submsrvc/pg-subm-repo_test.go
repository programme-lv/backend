package submsrvc

import (
	"sort"
	"testing"

	"context"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"github.com/peterldowns/pgtestdb/migrators/golangmigrator"
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
	gm := golangmigrator.New("../migrate")
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
	assert.Equal(t, 24, version)
	assert.False(t, dirty)
}

// getSampleSubmEntityWithoutEval creates a SubmissionEntity with sample data.
func getSampleSubmEntityWithoutEval() SubmissionEntity {
	return SubmissionEntity{
		UUID:        uuid.New(),
		Content:     "Sample submission content",
		AuthorUUID:  existingAuthorUuid, // author must pre-exist in the db
		TaskShortID: "task_123",
		LangShortID: "py_x.y.z",
		CurrEvalID:  uuid.Nil,
		CreatedAt:   time.Now(),
	}
}

// TestSubmRepo_StoreWithoutEval_Success tests successful storage of a SubmissionEntity without an evaluation.
func TestSubmRepo_StoreWithoutEval_Success(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	sampleEntity := getSampleSubmEntityWithoutEval()

	err := repo.Store(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid SubmissionEntity")

	// Retrieve the stored entity
	storedEntity, err := repo.Get(context.Background(), sampleEntity.UUID)
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
	sampleEntity.CurrEvalID = existingEvalUuid

	err := repo.Store(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid SubmissionEntity")

	// Retrieve the stored entity
	storedEntity, err := repo.Get(context.Background(), sampleEntity.UUID)
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
	err := repo.Store(context.Background(), sampleEntity)
	assert.Nil(t, err, "expected no error when storing valid SubmissionEntity")

	// Retrieve the entity
	retrievedEntity, err := repo.Get(context.Background(), sampleEntity.UUID)
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

	retrievedEntity, err := repo.Get(context.Background(), nonExistentUUID)
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
	entities := make([]SubmissionEntity, numEntries)
	for i := 0; i < numEntries; i++ {
		entities[i] = getSampleSubmEntityWithoutEval()
	}
	sort.Slice(entities, func(i, j int) bool {
		return entities[i].CreatedAt.After(entities[j].CreatedAt)
	})

	for i := 0; i < numEntries; i++ {
		err := repo.Store(context.Background(), entities[i])
		require.Nil(t, err, "expected no error when storing SubmissionEntity")
	}

	// List all entities
	listedEntities, err := repo.List(context.Background(), 3, 1)
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
	listedEntities, err := repo.List(context.Background(), 100, 0)
	assert.Nil(t, err, "expected no error when listing with no SubmissionEntities")
	assert.Empty(t, listedEntities, "expected no SubmissionEntities to be listed")
}

// TestSubmRepo_Store_MissingFields tests storing SubmissionEntities with missing required fields.
func TestSubmRepo_Store_MissingFields(t *testing.T) {
	t.Parallel()
	repo := NewPgSubmRepo(NewSampleDB(t))

	// Example: Missing Content and AuthorUUID
	invalidEntities := []SubmissionEntity{
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
		err := repo.Store(context.Background(), entity)
		assert.NotNil(t, err, "expected error when storing SubmissionEntity with missing fields")
	}
}
