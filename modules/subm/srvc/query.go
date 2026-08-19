package srvc

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/srvcerror"
	"github.com/programme-lv/backend/modules/plang"
	"github.com/programme-lv/backend/modules/subm/domain"
	tasksrvc "github.com/programme-lv/backend/modules/task/srvc"
	usersrvc "github.com/programme-lv/backend/modules/user"
	"github.com/programme-lv/backend/modules/user/auth"
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
			_, err := s.taskSrvc.ResolveNames(ctx, []string{taskShortID})
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
	getFullEval      func(ctx context.Context, evalUUID uuid.UUID) (domain.Eval, srvcerror.E)
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

// CountSubms returns the total number of submissions matching filter.
func (s *submSrvc) CountSubms(ctx context.Context, filter ListSubmsParams) (int, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("query", "count submissions")

	authorIds, taskIds, langIds, resolveErr := s.resolveSearchIDs(ctx, filter.Search)
	if resolveErr != nil {
		return 0, resolveErr
	}
	count, countErr := s.submRepo.CountSubms(ctx, filter.Author, filter.TaskShortID, authorIds, taskIds, langIds, filter.IncludeAdmin)
	if countErr != nil {
		log.Error("count submissions", "error", countErr)
		return 0, ErrInternal
	}

	return count, nil
}

func (s *submSrvc) ViewSubm(ctx context.Context, submUuid uuid.UUID) (domain.Subm, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("query", "view submission")

	subm, err := s.submRepo.GetSubm(ctx, submUuid)
	if err != nil {
		return mapGetSubmErr(log, err, "subm_uuid", submUuid.String())
	}

	return s.redactSubmContent(ctx, subm)
}

func (s *submSrvc) ViewSubmByShortID(ctx context.Context, shortID string) (domain.Subm, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("query", "view submission")

	subm, err := s.submRepo.GetSubmByShortID(ctx, shortID)
	if err != nil {
		return mapGetSubmErr(log, err, "short_id", shortID)
	}

	return s.redactSubmContent(ctx, subm)
}

func mapGetSubmErr(log *slog.Logger, err error, idKey, idVal string) (domain.Subm, srvcerror.E) {
	if errors.Is(err, domain.ErrNotFound) {
		log.Warn("submission not found", idKey, idVal)
		return domain.Subm{}, ErrSubmissionNotFound
	}
	log.Error("get submission", "error", err, idKey, idVal)
	return domain.Subm{}, srvcerror.InternalServerError()
}

func (s *submSrvc) redactSubmContent(ctx context.Context, subm domain.Subm) (domain.Subm, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("query", "view submission")

	userHasSolvedTheTask := false
	userUUID, authErr := auth.GetUserUuidFromCtx(ctx)
	if authErr == nil && userUUID != subm.AuthorUUID {
		userMaxScores, maxScoreErr := s.GetMaxScorePerTask(ctx, userUUID)
		if maxScoreErr != nil {
			log.Error("get user scores", "error", maxScoreErr)
			return domain.Subm{}, srvcerror.InternalServerError()
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
	Author *uuid.UUID // optional; AND with other filters
	// TaskShortID is an exact task short id AND filter. Distinct from Search,
	// which ORs fuzzy matches across task, user, and language.
	TaskShortID string

	IncludeAdmin bool // whether to include admin submissions
}

func (s *submSrvc) resolveSearchIDs(ctx context.Context, search string) (authorIds, taskIds, langIds []string, err srvcerror.E) {
	authorIds = make([]string, 0)
	taskIds = make([]string, 0)
	langIds = make([]string, 0)
	if search == "" {
		return authorIds, taskIds, langIds, nil
	}

	log := ctxlog.FromContext(ctx)

	var searchTasksByNameErr srvcerror.E
	taskIds, searchTasksByNameErr = s.taskSrvc.SearchTasksByName(ctx, search)
	if searchTasksByNameErr != nil {
		return nil, nil, nil, searchTasksByNameErr
	}
	taskIds = append(taskIds, search)

	authorId, userErr := s.userSrvc.GetUserByUsername(ctx, search)
	if userErr != nil && !errors.Is(userErr, usersrvc.ErrUserNotFound) {
		return nil, nil, nil, userErr
	}
	if authorId.UUID != uuid.Nil {
		authorIds = append(authorIds, authorId.UUID.String())
	}
	if _, parseErr := uuid.Parse(search); parseErr == nil {
		authorIds = append(authorIds, search)
	}

	var searchProgrLangByNameErr error
	langIds, searchProgrLangByNameErr = plang.SearchProgrLangByName(search)
	if searchProgrLangByNameErr != nil {
		log.Error("search programming languages by name", "error", searchProgrLangByNameErr)
		return nil, nil, nil, srvcerror.InternalServerError()
	}
	langIds = append(langIds, search)
	return authorIds, taskIds, langIds, nil
}

func (s *submSrvc) ListSubms(ctx context.Context, filter ListSubmsParams) ([]domain.Subm, srvcerror.E) {
	log := ctxlog.FromContext(ctx).With("query", "list submissions")
	log.Debug("listing submissions", "limit", filter.Limit, "offset", filter.Offset)

	authorIds, taskIds, langIds, resolveErr := s.resolveSearchIDs(ctx, filter.Search)
	if resolveErr != nil {
		return nil, resolveErr
	}

	subms, listErr := s.submRepo.ListSubms(ctx, filter.Limit, filter.Offset, filter.Search, filter.Author, filter.TaskShortID, authorIds, taskIds, langIds, filter.IncludeAdmin)
	if listErr != nil {
		log.Error("list submissions from repo", "error", listErr)
		return nil, srvcerror.InternalServerError()
	}
	return subms, nil
}

func (s *submSrvc) GetEval(ctx context.Context, uuid uuid.UUID) (domain.Eval, srvcerror.E) {
	if eval, ok := s.inProgrEval[uuid]; ok {
		return eval, nil
	}
	eval, err := s.evalRepo.GetEval(ctx, uuid)
	if err != nil {
		log := ctxlog.FromContext(ctx).With("query", "get evaluation")
		log.Error("get evaluation from repo", "error", err)
		return domain.Eval{}, srvcerror.InternalServerError()
	}
	return eval, nil
}
