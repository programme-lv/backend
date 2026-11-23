package srvc

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCommunityStatsRepo struct {
	calls int
	ids   []string
	stats map[string]TaskCommunityStats
	err   error
}

func (f *fakeCommunityStatsRepo) TaskCommunityStats(
	_ context.Context,
	taskShortIDs []string,
) (map[string]TaskCommunityStats, error) {
	f.calls++
	f.ids = append([]string(nil), taskShortIDs...)
	return f.stats, f.err
}

func TestCommunityStatsService_EmptyInputSkipsRepo(t *testing.T) {
	repo := &fakeCommunityStatsRepo{}
	service := NewCommunityStatsService(repo)

	stats, err := service.TaskCommunityStats(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, stats)
	assert.Zero(t, repo.calls)
}

func TestCommunityStatsService_ReturnsRepoResult(t *testing.T) {
	want := map[string]TaskCommunityStats{
		"task-a": {
			UniqueSolvers:   1,
			UniqueAttempted: 2,
			Verdicts:        map[string]int{CommunityVerdictAccepted: 1, CommunityVerdictWrongAnswer: 1},
		},
	}
	repo := &fakeCommunityStatsRepo{stats: want}
	service := NewCommunityStatsService(repo)

	got, err := service.TaskCommunityStats(context.Background(), []string{"task-a"})
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, []string{"task-a"}, repo.ids)
}

func TestCommunityStatsService_NilResultBecomesEmptyMap(t *testing.T) {
	service := NewCommunityStatsService(&fakeCommunityStatsRepo{})

	got, err := service.TaskCommunityStats(context.Background(), []string{"task-a"})
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestCommunityStatsService_RepoErrorIsSanitized(t *testing.T) {
	repo := &fakeCommunityStatsRepo{err: assert.AnError}
	service := NewCommunityStatsService(repo)

	ctx := ctxlog.WithLogger(context.Background(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	stats, err := service.TaskCommunityStats(ctx, []string{"task-a"})
	require.Error(t, err)
	assert.Nil(t, stats)
	assert.True(t, srvcerror.Is(err, srvcerror.ErrCodeInternalServerError))
	assert.NotContains(t, err.Error(), assert.AnError.Error())
}
