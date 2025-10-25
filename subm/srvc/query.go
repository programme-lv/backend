package srvc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/subm/domain"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
)

// GetMaxScorePerTask implements SubmSrvcClient.
func (s *submSrvc) GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, error) {
	// Get all submissions
	// TODO: wtf is this?
	subms, err := s.submRepo.ListSubms(ctx, 10000, 0, "", nil, []string{}, []string{}, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}

	if len(subms) == 10000 {
		slog.Error("too many submissions", "user_uuid", userUUID)
	}

	taskExistsCache := make(map[string]bool)

	// Filter submissions by user and collect evaluations
	userSubmsWithEval := make([]domain.SubmJoinEval, 0)
	for _, subm := range subms {
		if subm.AuthorUUID != userUUID {
			continue
		}

		// Skip tasks that don't exist anymore
		if _, ok := taskExistsCache[subm.TaskShortID]; !ok {
			_, err := s.taskSrvc.GetTaskFullNames(ctx, []string{subm.TaskShortID})
			if srvcerror.Is(err, tasksrvc.ErrSomeTaskNotFound) {
				taskExistsCache[subm.TaskShortID] = false
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("could not resolve task full name: %w", err)
			}
			taskExistsCache[subm.TaskShortID] = true
		}

		// Skip submissions without evaluations
		if subm.CurrEvalUUID == uuid.Nil {
			continue
		}

		// Get the evaluation
		eval, err := s.GetEval(ctx, subm.CurrEvalUUID)
		if err != nil {
			slog.Error("failed to get evaluation", "error", err, "eval_uuid", subm.CurrEvalUUID)
			continue
		}

		userSubmsWithEval = append(userSubmsWithEval, domain.SubmJoinEval{
			Subm: subm,
			Eval: eval,
		})
	}

	// Calculate max scores using the domain logic
	return domain.CalcMaxScores(userSubmsWithEval), nil
}
