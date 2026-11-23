package srvc

import (
	"context"

	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
)

// Community verdict buckets exposed by GET /tasks community_stats.
const (
	CommunityVerdictAccepted            = "accepted"
	CommunityVerdictPartiallyCorrect    = "partially_correct"
	CommunityVerdictWrongAnswer         = "wrong_answer"
	CommunityVerdictTimeLimitExceeded   = "time_limit_exceeded"
	CommunityVerdictMemoryLimitExceeded = "memory_limit_exceeded"
	CommunityVerdictRuntimeError        = "runtime_error"
	CommunityVerdictCompileError        = "compile_error"
	CommunityVerdictInternalError       = "internal_error"
	CommunityVerdictTesterUnavailable   = "tester_unavailable"
	CommunityVerdictUnknown             = "unknown"
)

// TaskCommunityStats is aggregate, non-attributable community activity for one
// task. Each user contributes at most once to UniqueAttempted and Verdicts,
// and a user who has ever been accepted contributes to UniqueSolvers.
type TaskCommunityStats struct {
	UniqueSolvers   int
	UniqueAttempted int
	// Verdicts contains only positive counts. Its sum equals UniqueAttempted.
	Verdicts map[string]int
}

// CommunityStatsRepo is the read-only persistence port for community stats.
type CommunityStatsRepo interface {
	TaskCommunityStats(ctx context.Context, taskShortIDs []string) (map[string]TaskCommunityStats, error)
}

// CommunityStatsService is a narrow read-only service used by the task HTTP
// gateway. It is intentionally separate from SubmissionService so the task
// module does not need the submission use cases.
type CommunityStatsService interface {
	TaskCommunityStats(ctx context.Context, taskShortIDs []string) (map[string]TaskCommunityStats, srvcerror.E)
}

type communityStatsSrvc struct {
	repo CommunityStatsRepo
}

var _ CommunityStatsService = &communityStatsSrvc{}

// NewCommunityStatsService returns a read-only community stats service.
func NewCommunityStatsService(repo CommunityStatsRepo) *communityStatsSrvc {
	return &communityStatsSrvc{repo: repo}
}

// TaskCommunityStats returns stats for the requested task IDs. Unknown or
// unlisted tasks and tasks without completed attempts are absent from the
// result.
func (s *communityStatsSrvc) TaskCommunityStats(
	ctx context.Context,
	taskShortIDs []string,
) (map[string]TaskCommunityStats, srvcerror.E) {
	if len(taskShortIDs) == 0 {
		return map[string]TaskCommunityStats{}, nil
	}

	stats, err := s.repo.TaskCommunityStats(ctx, taskShortIDs)
	if err != nil {
		ctxlog.FromContext(ctx).
			With("module", "subm", "layer", "srvc").
			Error("query task community stats", "task_count", len(taskShortIDs), "error", err)
		return nil, srvcerror.InternalServerError()
	}
	if stats == nil {
		stats = map[string]TaskCommunityStats{}
	}
	return stats, nil
}
