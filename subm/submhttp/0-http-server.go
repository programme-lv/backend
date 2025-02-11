package submhttp

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/subm/submdomain"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/task"
	"github.com/programme-lv/backend/usersrvc"
)

type SubmHttpHandler struct {
	submSrvc submsrvc.SubmSrvcClient
	taskSrvc *task.TaskSrvc
	userSrvc *usersrvc.UserSrvc

	// solution submission rate limit
	lastSubmTime map[string]time.Time // username -> last submission time
	rateLock     sync.Mutex
}

func NewSubmHttpHandler(
	submSrvc submsrvc.SubmSrvcClient,
	taskSrvc *task.TaskSrvc,
	userSrvc *usersrvc.UserSrvc,
) *SubmHttpHandler {
	return &SubmHttpHandler{
		submSrvc:     submSrvc,
		taskSrvc:     taskSrvc,
		userSrvc:     userSrvc,
		lastSubmTime: make(map[string]time.Time),
	}
}

func (h *SubmHttpHandler) mapSubm(
	ctx context.Context,
	s submdomain.Subm,
) (*DetailedSubmView, error) {
	return mapSubm(
		ctx,
		s,
		h.getTaskFullName,
		h.getUsername,
		h.getPrLang,
		h.getEval,
	)
}

func (h *SubmHttpHandler) mapSubmListEntry(
	ctx context.Context,
	s submdomain.Subm,
) (SubmListEntry, error) {
	return mapSubmListEntry(
		ctx,
		s,
		h.getTaskFullName,
		h.getUsername,
		h.getPrLang,
		h.getEval,
	)
}

func (h *SubmHttpHandler) getTaskFullName(ctx context.Context, shortID string) (string, error) {
	task, err := h.taskSrvc.GetTask(ctx, shortID)
	if err != nil {
		return "", err
	}
	return task.FullName, nil
}

func (h *SubmHttpHandler) getUsername(ctx context.Context, userUuid uuid.UUID) (string, error) {
	user, err := h.userSrvc.GetUserByUUID(ctx, userUuid)
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

func (h *SubmHttpHandler) getPrLang(ctx context.Context, shortID string) (PrLang, error) {
	plang, err := planglist.GetProgrLangById(shortID)
	if err != nil {
		return PrLang{}, err
	}
	return PrLang{
		ShortID:  plang.ID,
		Display:  plang.FullName,
		MonacoID: plang.MonacoId,
	}, nil
}

func (h *SubmHttpHandler) getEval(ctx context.Context, evalUuid uuid.UUID) (submdomain.Eval, error) {
	return h.submSrvc.GetEval(ctx, evalUuid)
}
