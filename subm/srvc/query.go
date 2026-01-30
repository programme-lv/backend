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
func (s *submSrvc) GetMaxScorePerTask(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, srvcerror.E) {
	taskExistsCache := make(map[string]bool)
	h := getMaxScorePerTaskHandler{
		listSubmJoinEval: s.submRepo.ListShallowSubmsJoinEval,
		doesTaskExist: func(ctx context.Context, taskShortID string) (bool, srvcerror.E) {
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
	doesTaskExist    func(ctx context.Context, taskShortID string) (bool, srvcerror.E)
	getFullEval      func(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, error)
}

func (h getMaxScorePerTaskHandler) Handle(ctx context.Context, userUUID uuid.UUID) (map[string]domain.MaxScore, srvcerror.E) {
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
func (s *submSrvc) CountSubms(ctx context.Context, search string, author *uuid.UUID, includeAdmin bool) (int, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("query", "count submissions")

	authorIds := make([]string, 0)
	taskIds := make([]string, 0)
	langIds := make([]string, 0)

	if search != "" {
		var err srvcerror.E

		// get all possible task matches
		taskIds, err = s.taskSrvc.SearchTasksByName(ctx, search)
		if err != nil {
			log.Error("search tasks by name", "error", err)
			return 0, ErrInternal
		}
		taskIds = append(taskIds, search)

		// get all possible user matches
		var authorId usersrvc.User
		authorId, err = s.userSrvc.GetUserByUsername(ctx, search)
		if err != nil && !errors.Is(err, usersrvc.ErrUserNotFound) {
			log.Error("get user by username", "error", err)
			return 0, ErrInternal
		}

		if authorId.UUID != uuid.Nil {
			authorIdStr := authorId.UUID.String()
			authorIds = append(authorIds, authorIdStr)
		}
		if _, err := uuid.Parse(search); err == nil {
			authorIds = append(authorIds, search)
		}

		// get all programming languages
		var pLangSearchErr error
		langIds, pLangSearchErr = plang.SearchProgrLangByName(search)
		if pLangSearchErr != nil {
			log.Error("search progr langs by name", "error", pLangSearchErr)
			return 0, ErrInternal
		}
		langIds = append(langIds, search)
	}
	count, countErr := s.submRepo.CountSubms(ctx, author, authorIds, taskIds, langIds, includeAdmin)
	if countErr != nil {
		log.Error("failed to count submissions", "error", countErr)
		return 0, ErrInternal
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

	IncludeAdmin bool // whether to include admin submissions
}

func (s *submSrvc) ListSubms(ctx context.Context, filter ListSubmsParams) ([]domain.Subm, error) {
	log := ctxlog.FromContext(ctx)
	log.Debug("listing submissions", "limit", filter.Limit, "offset", filter.Offset)

	authorIds := make([]string, 0)
	taskIds := make([]string, 0)
	langIds := make([]string, 0)

	if filter.Search != "" {
		// get all possible task matches
		var searchTasksByNameErr srvcerror.E
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
	return s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset, filter.Search, filter.Author, authorIds, taskIds, langIds, filter.IncludeAdmin)
}

func (s *submSrvc) GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, error) {
	if eval, ok := s.inProgrEval[uuid]; ok {
		return eval, nil
	}
	return s.evalRepo.GetEval(ctx, uuid)
}
