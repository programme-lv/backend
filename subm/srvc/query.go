package srvc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/plang"
	"github.com/programme-lv/backend/subm/domain"
	tasksrvc "github.com/programme-lv/backend/task/srvc"
	usersrvc "github.com/programme-lv/backend/user"
	"github.com/programme-lv/backend/user/auth"
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

// CountSubms returns the total number of submissions
func (s *submSrvc) CountSubms(ctx context.Context, search string, author *uuid.UUID) (int, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("counting submissions")

	authorIds := make([]string, 0)
	taskIds := make([]string, 0)
	langIds := make([]string, 0)

	if search != "" {
		// get all possible task matches
		var searchTasksByNameErr *srvcerror.Error
		taskIds, searchTasksByNameErr = s.taskSrvc.SearchTasksByName(ctx, search)
		if searchTasksByNameErr != nil {
			return 0, fmt.Errorf("failed to search tasks by name: %w", searchTasksByNameErr)
		}
		taskIds = append(taskIds, search)

		// get all possible user matches
		authorId, err := s.userSrvc.GetUserByUsername(ctx, search)
		if err != nil && !srvcerror.Is(err, usersrvc.ErrCodeUserNotFound) {
			return 0, fmt.Errorf("failed to get user by username: %w", err)
		}
		if authorId.UUID != uuid.Nil {
			authorIdStr := authorId.UUID.String()
			authorIds = append(authorIds, authorIdStr)
		}
		if _, err := uuid.Parse(search); err == nil {
			authorIds = append(authorIds, search)
		}

		// get all programming languages
		langIds, err = plang.SearchProgrLangByName(search)
		if err != nil {
			return 0, fmt.Errorf("failed to get programming language by id: %w", err)
		}
		langIds = append(langIds, search)
	}
	count, err := s.submRepo.CountSubms(ctx, author, authorIds, taskIds, langIds)
	if err != nil {
		log.Error("failed to count submissions", "error", err)
		return 0, err
	}

	return count, nil
}

func (s *submSrvc) ViewSubm(ctx context.Context, submUuid uuid.UUID) (domain.Subm, error) {
	log := ctxlog.FromContext(ctx)

	subm, err := s.submRepo.GetSubm(ctx, submUuid)
	if err != nil {
		log.Error("failed to get submission", "error", err)
		return domain.Subm{}, fmt.Errorf("failed to get submission: %w", err)
	}

	userHasSolvedTheTask := false
	userUUID, err := auth.GetUserUuidFromCtx(ctx)
	if err == nil && userUUID != subm.AuthorUUID {
		userMaxScores, err := s.GetMaxScorePerTask(ctx, userUUID)
		if err != nil {
			log.Error("failed to get user scores", "error", err)
			return domain.Subm{}, fmt.Errorf("failed to get user scores: %w", err)
		}
		userScore, ok := userMaxScores[subm.TaskShortID]
		if ok {
			userHasSolvedTheTask = userScore.Received >= userScore.Possible
		}
	}

	if !userHasSolvedTheTask && subm.AuthorUUID != userUUID {
		subm.Content = ""
	}

	return subm, nil
}

type ListSubmsParams struct {
	Limit  int
	Offset int
	Search string
	Author *uuid.UUID // Optional author UUID to filter by
}

func (s *submSrvc) ListSubms(ctx context.Context, filter ListSubmsParams) ([]domain.Subm, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("listing submissions", "limit", filter.Limit, "offset", filter.Offset)

	authorIds := make([]string, 0)
	taskIds := make([]string, 0)
	langIds := make([]string, 0)

	if filter.Search != "" {
		// get all possible task matches
		var searchTasksByNameErr *srvcerror.Error
		taskIds, searchTasksByNameErr = s.taskSrvc.SearchTasksByName(ctx, filter.Search)
		if searchTasksByNameErr != nil {
			return nil, fmt.Errorf("failed to search tasks by name: %w", searchTasksByNameErr)
		}
		taskIds = append(taskIds, filter.Search)

		// get all possible user matches
		authorId, err := s.userSrvc.GetUserByUsername(ctx, filter.Search)
		if err != nil && !srvcerror.Is(err, usersrvc.ErrCodeUserNotFound) {
			return nil, fmt.Errorf("failed to get user by username: %w", err)
		}
		if authorId.UUID != uuid.Nil {
			authorIdStr := authorId.UUID.String()
			authorIds = append(authorIds, authorIdStr)
		}
		if _, err := uuid.Parse(filter.Search); err == nil {
			authorIds = append(authorIds, filter.Search)
		}

		// get all programming languages
		langIds, err = plang.SearchProgrLangByName(filter.Search)
		if err != nil {
			return nil, fmt.Errorf("failed to get programming language by id: %w", err)
		}
		langIds = append(langIds, filter.Search)
	}
	return s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset, filter.Search, filter.Author, authorIds, taskIds, langIds)
}

func (s *submSrvc) GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error) {
	if eval, ok := s.inProgrEval[uuid]; ok {
		return eval, nil
	}
	return s.evalRepo.GetEval(ctx, uuid)
}
