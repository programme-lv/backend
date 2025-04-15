package http

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/planglist"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submsrvc"
	"github.com/programme-lv/backend/task/srvc"
	"github.com/programme-lv/backend/user"
	"golang.org/x/sync/singleflight"
)

type SubmHttpHandler struct {
	submSrvc submsrvc.SubmSrvcClient
	taskSrvc srvc.TaskSrvcClient
	userSrvc *user.UserSrvc

	// solution submission rate limit
	lastSubmTime map[string]time.Time // username -> last submission time
	rateLock     sync.Mutex

	// submCache and singleflight for preventing submCache stampedes
	submCache *cache.Cache
	sfGroup   singleflight.Group
}

func NewSubmHttpHandler(
	submSrvc submsrvc.SubmSrvcClient,
	taskSrvc srvc.TaskSrvcClient,
	userSrvc *user.UserSrvc,
) *SubmHttpHandler {
	// Create a cache with 1 second default expiration and 1 minute cleanup interval
	c := cache.New(1*time.Second, 1*time.Minute)
	return &SubmHttpHandler{
		submSrvc:     submSrvc,
		taskSrvc:     taskSrvc,
		userSrvc:     userSrvc,
		lastSubmTime: make(map[string]time.Time),
		submCache:    c,
		// singleflight.Group doesn't need initialization
	}
}

func (h *SubmHttpHandler) mapSubm(
	ctx context.Context,
	s domain.Subm,
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
	s domain.Subm,
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
	taskNames, err := h.taskSrvc.ResolveNames(ctx, []string{shortID})
	if err != nil {
		return "", fmt.Errorf("failed to resolve task name: %w", err)
	}
	return taskNames[0], nil
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

func (h *SubmHttpHandler) getEval(ctx context.Context, evalUuid uuid.UUID) (domain.Eval, error) {
	return h.submSrvc.GetEval(ctx, evalUuid)
}
