package pgrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/srvc"
)

type pgProfileRepo struct {
	pool *pgxpool.Pool
}

func NewPgProfileRepo(pool *pgxpool.Pool) *pgProfileRepo {
	return &pgProfileRepo{pool: pool}
}

func (r *pgProfileRepo) SaveProfile(ctx context.Context, profile domain.SolutionProfile) error {
	runs, err := json.Marshal(profile.Runs)
	if err != nil {
		return fmt.Errorf("marshal profile runs: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO task_solution_profiles (
			id, task_id, fname, lang_id, status, error,
			expected_points, possible_points, requested_runs, tests_version,
			solution_sha256, runs, created_at, finished_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			error = EXCLUDED.error,
			expected_points = EXCLUDED.expected_points,
			possible_points = EXCLUDED.possible_points,
			requested_runs = EXCLUDED.requested_runs,
			tests_version = EXCLUDED.tests_version,
			solution_sha256 = EXCLUDED.solution_sha256,
			runs = EXCLUDED.runs,
			finished_at = EXCLUDED.finished_at
	`,
		profile.ID,
		profile.TaskShortID,
		profile.Fname,
		profile.LangID,
		string(profile.Status),
		profile.Error,
		profile.ExpectedPoints,
		profile.PossiblePoints,
		profile.RequestedRuns,
		profile.TestsVersion,
		profile.SolutionSha256,
		runs,
		profile.CreatedAt,
		profile.FinishedAt,
	)
	if err != nil {
		return fmt.Errorf("save solution profile: %w", err)
	}
	return nil
}

func (r *pgProfileRepo) GetProfile(ctx context.Context, id uuid.UUID) (domain.SolutionProfile, error) {
	const query = `
		SELECT id, task_id, fname, lang_id, status, error,
			expected_points, possible_points, requested_runs, tests_version,
			solution_sha256, runs, created_at, finished_at
		FROM task_solution_profiles
		WHERE id = $1
	`
	return r.scanProfile(r.pool.QueryRow(ctx, query, id))
}

func (r *pgProfileRepo) LatestFinishedProfile(ctx context.Context, taskID, fname string) (domain.SolutionProfile, error) {
	const query = `
		SELECT id, task_id, fname, lang_id, status, error,
			expected_points, possible_points, requested_runs, tests_version,
			solution_sha256, runs, created_at, finished_at
		FROM task_solution_profiles
		WHERE task_id = $1 AND fname = $2 AND status = 'finished'
		ORDER BY created_at DESC
		LIMIT 1
	`
	return r.scanProfile(r.pool.QueryRow(ctx, query, taskID, fname))
}

func (r *pgProfileRepo) ListProfiles(ctx context.Context, taskID, fname string, limit int) ([]domain.SolutionProfile, error) {
	const query = `
		SELECT id, task_id, fname, lang_id, status, error,
			expected_points, possible_points, requested_runs, tests_version,
			solution_sha256, runs, created_at, finished_at
		FROM task_solution_profiles
		WHERE task_id = $1 AND fname = $2
		ORDER BY created_at DESC
		LIMIT $3
	`
	rows, err := r.pool.Query(ctx, query, taskID, fname, limit)
	if err != nil {
		return nil, fmt.Errorf("query solution profiles: %w", err)
	}
	defer rows.Close()

	var profiles []domain.SolutionProfile
	for rows.Next() {
		profile, scanErr := r.scanProfile(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate solution profiles: %w", err)
	}
	return profiles, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (r *pgProfileRepo) scanProfile(row rowScanner) (domain.SolutionProfile, error) {
	var profile domain.SolutionProfile
	var status string
	var runs []byte

	err := row.Scan(
		&profile.ID,
		&profile.TaskShortID,
		&profile.Fname,
		&profile.LangID,
		&status,
		&profile.Error,
		&profile.ExpectedPoints,
		&profile.PossiblePoints,
		&profile.RequestedRuns,
		&profile.TestsVersion,
		&profile.SolutionSha256,
		&runs,
		&profile.CreatedAt,
		&profile.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.SolutionProfile{}, srvc.ErrProfileNotFoundInRepo
		}
		return domain.SolutionProfile{}, fmt.Errorf("scan solution profile: %w", err)
	}

	profile.Status = domain.ProfileStatus(status)
	if len(runs) > 0 {
		if err := json.Unmarshal(runs, &profile.Runs); err != nil {
			return domain.SolutionProfile{}, fmt.Errorf("unmarshal profile runs: %w", err)
		}
	}
	return profile, nil
}
