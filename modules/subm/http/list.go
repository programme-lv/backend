package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/subm/domain"
	"github.com/programme-lv/backend/modules/subm/srvc"
	"github.com/programme-lv/backend/modules/user/auth"
)

// PaginatedResponse represents a paginated response with data and pagination metadata
type PaginatedResponse struct {
	Page       []SubmListEntry `json:"page"`
	Pagination Pagination      `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Total   int  `json:"total"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"hasMore"`
}

const (
	defaultSubmListLimit = 30
	maxSubmListLimit     = 100
	maxTaskIDQueryLen    = 50
)

func (h *SubmHttpHandler) GetSubmList(w http.ResponseWriter, r *http.Request) {
	log := ctxlog.FromContext(r.Context())
	w.Header().Set("Cache-Control", "no-store")

	limit := parseSubmListLimit(r.URL.Query().Get("limit"))
	offset := 0

	offsetStr := r.URL.Query().Get("offset")
	if offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	search := r.URL.Query().Get("search")
	if len(search) > 100 {
		jsonresp.BadRequest(w, "Search query too long")
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if len(taskID) > maxTaskIDQueryLen {
		jsonresp.BadRequest(w, "task_id too long")
		return
	}

	var author *uuid.UUID
	if parseMineQuery(r.URL.Query().Get("mine")) {
		userUUID, err := auth.GetUserUuidFromCtx(r.Context())
		if err != nil {
			jsonresp.HandleErrorWithContext(r.Context(), w, ErrJwtTokenMissing)
			return
		}
		author = &userUUID
	}

	includeAdmin := false
	if auth.IsAdmin(r.Context()) {
		includeAdmin = true
	}

	authorKey := ""
	if author != nil {
		authorKey = author.String()
	}
	cacheKey := fmt.Sprintf("subm_list:%d:%d:%s:%t:%s:%s", limit, offset, search, includeAdmin, authorKey, taskID)

	filter := srvc.ListSubmsParams{
		Limit:        limit,
		Offset:       offset,
		Search:       search,
		Author:       author,
		TaskShortID:  taskID,
		IncludeAdmin: includeAdmin,
	}

	if cachedResponse, found := h.submCache.Get(cacheKey); found {
		if response, ok := cachedResponse.(PaginatedResponse); ok {
			jsonresp.Success(w, response)
			return
		}
	}

	result, err, _ := h.sfGroup.Do(cacheKey, func() (interface{}, error) {
		if cachedResponse, found := h.submCache.Get(cacheKey); found {
			if response, ok := cachedResponse.(PaginatedResponse); ok {
				return response, nil
			}
		}

		totalCount, countSubmsErr := h.submSrvc.CountSubms(r.Context(), filter)
		if countSubmsErr != nil {
			return nil, countSubmsErr
		}

		subms, err := h.submSrvc.ListSubms(r.Context(), filter)
		if err != nil {
			return nil, err
		}

		mapSubmList := func(subms []domain.Subm) []SubmListEntry {
			response := make([]SubmListEntry, 0)
			for _, subm := range subms {
				entry, err := h.mapSubmListEntry(r.Context(), subm)
				if err != nil {
					log.Warn("map subm list entry", "error", err)
					continue
				}
				response = append(response, entry)
			}
			return response
		}

		submEntries := mapSubmList(subms)

		hasMore := offset+len(submEntries) < totalCount
		paginatedResponse := PaginatedResponse{
			Page: submEntries,
			Pagination: Pagination{
				Total:   totalCount,
				Offset:  offset,
				Limit:   limit,
				HasMore: hasMore,
			},
		}

		h.submCache.Set(cacheKey, paginatedResponse, 0)

		return paginatedResponse, nil
	})

	if err != nil {
		jsonresp.HandleErrorWithContext(r.Context(), w, err)
		return
	}

	response, ok := result.(PaginatedResponse)
	if !ok {
		log.Error("submission list cache value has unexpected type")
		jsonresp.InternalError(w)
		return
	}
	jsonresp.Success(w, response)
}

func parseMineQuery(raw string) bool {
	return raw == "1" || raw == "true"
}

func parseSubmListLimit(raw string) int {
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultSubmListLimit
	}
	return min(parsed, maxSubmListLimit)
}
