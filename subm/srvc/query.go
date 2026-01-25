package srvc

import (
	"context"
	"errors"
	"fmt"

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
func (s *submSrvc) GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, *srvcerror.Error) {
	taskExistsCache := make(map[string]bool)
	h := getMaxScorePerTaskHandler{
		listSubmJoinEval: s.submRepo.ListShallowSubmsJoinEval,
		doesTaskExist: func(ctx context.Context, taskShortID string) (bool, *srvcerror.Error) {
			if exists, ok := taskExistsCache[taskShortID]; ok {
				return exists, nil
			}
			_, err := s.taskSrvc.GetTaskFullNames(ctx, []string{taskShortID})
			if err != nil {
				if errors.Is(err, tasksrvc.ErrSomeTaskNotFound) {
					taskExistsCache[taskShortID] = false
					return false, nil
				}
				return false, err
			}
			taskExistsCache[taskShortID] = true
			return true, nil
		},
		getFullEval: s.GetEval,
	}
	return h.Handle(ctx, userUUID)
}

type getMaxScorePerTaskHandler struct {
	listSubmJoinEval func(ctx context.Context, authorUuid *uuid.UUID) ([]ShallowSubmJoinEvalDto, error)
	doesTaskExist    func(ctx context.Context, taskShortID string) (bool, *srvcerror.Error)
	getFullEval      func(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error)
}

func (h getMaxScorePerTaskHandler) Handle(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, *srvcerror.Error) {
	log := ctxlog.FromContext(ctx).With("handler", "get max score per task")

	submJoinEvalList, err := h.listSubmJoinEval(ctx, &userUUID)
	if err != nil {
		action := "list shallow subms joined with evals"
		log.Error(action, "user uuid", userUUID, "error", err)
		return nil, srvcerror.InternalServerError()
	}

	userSubmsWithScoreInfo := make([]domain.SubmJoinScoreInfo, 0)
	for _, submJoinEval := range submJoinEvalList {
		doesTaskExist, err := h.doesTaskExist(ctx, submJoinEval.Subm.TaskShortID)
		if err != nil {
			action := "check if task exists"
			log.Error(action, "user uuid", userUUID, "error", err)
			return nil, srvcerror.InternalServerError()
		}
		if !doesTaskExist {
			continue
		}

		if submJoinEval.Eval.ScoreInfo == nil {
			log.Error("no score info", "eval_uuid", submJoinEval.Eval.UUID)
			fullEval, err := h.getFullEval(ctx, submJoinEval.Eval.UUID)
			if err != nil {
				action := "get full eval"
				log.Error(action, "user uuid", userUUID, "error", err)
				return nil, srvcerror.InternalServerError()
			}

			userSubmsWithScoreInfo = append(userSubmsWithScoreInfo, domain.SubmJoinScoreInfo{
				SubmUuid:    submJoinEval.Subm.UUID,
				TaskShortID: submJoinEval.Subm.TaskShortID,
				CreatedAt:   submJoinEval.Subm.CreatedAt,
				ScoreInfo:   fullEval.CalculateScore(),
			})
		} else {
			userSubmsWithScoreInfo = append(userSubmsWithScoreInfo, domain.SubmJoinScoreInfo{
				SubmUuid:    submJoinEval.Subm.UUID,
				TaskShortID: submJoinEval.Subm.TaskShortID,
				CreatedAt:   submJoinEval.Subm.CreatedAt,
				ScoreInfo:   *submJoinEval.Eval.ScoreInfo,
			})
		}

	}

	return domain.CalcMaxScores(userSubmsWithScoreInfo), nil

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
		var searchProgrLangByNameErr error
		langIds, searchProgrLangByNameErr = plang.SearchProgrLangByName(search)
		if searchProgrLangByNameErr != nil {
			return 0, fmt.Errorf("failed to search programming languages by name: %w", searchProgrLangByNameErr)
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
		var searchProgrLangByNameErr error
		langIds, searchProgrLangByNameErr = plang.SearchProgrLangByName(filter.Search)
		if searchProgrLangByNameErr != nil {
			return nil, fmt.Errorf("failed to search programming languages by name: %w", searchProgrLangByNameErr)
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
