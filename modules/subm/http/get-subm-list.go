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
	Page       interface{} `json:"page"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Total   int  `json:"total"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"hasMore"`
}

func (h *SubmHttpHandler) GetSubmList(w http.ResponseWriter, r *http.Request) {
	log := ctxlog.FromContext(r.Context())
	log.Info("getting submission list")
	w.Header().Set("Cache-Control", "no-store")

	// Parse pagination parameters from query string
	limit := 30 // Default limit
	offset := 0 // Default offset

	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

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

	onlyMyStr := r.URL.Query().Get("my")
	onlyMy := false
	if onlyMyStr == "true" {
		onlyMy = true
	}

	var authorUuid *uuid.UUID
	if onlyMy {
		userUuid, err := auth.GetUserUuidFromCtx(r.Context())
		if err == auth.ErrNoJwtClaims || err == auth.ErrEmptyJwtClaims {
			jsonresp.Unauthorized(w, "jwt claims are missing")
			return
		}
		if err != nil {
			log.Error("failed to get user uuid from context", "error", err)
			jsonresp.HandleErrorWithContext(r.Context(), w, err)
			return
		}
		authorUuid = &userUuid
	}

	includeAdmin := false
	if auth.IsAdmin(r.Context()) {
		includeAdmin = true
	}

	// Create a cache key based on pagination parameters
	authorUuidStr := ""
	if authorUuid != nil {
		authorUuidStr = authorUuid.String()
	}
	cacheKey := fmt.Sprintf("subm_list:%d:%d:%s:%s:%t", limit, offset, search, authorUuidStr, includeAdmin)

	// Try to get from cache first
	if cachedResponse, found := h.submCache.Get(cacheKey); found {
		if response, ok := cachedResponse.(PaginatedResponse); ok {
			log.Info("returning cached submission list", "limit", limit, "offset", offset, "search", search, "includeAdmin", includeAdmin)
			jsonresp.Success(w, response)
			return
		}
	}

	// If not in cache or invalid cache, use singleflight to prevent multiple concurrent requests
	// from all hitting the database at the same time
	result, err, _ := h.sfGroup.Do(cacheKey, func() (interface{}, error) {
		// Check cache again in case another request already populated it while we were waiting
		if cachedResponse, found := h.submCache.Get(cacheKey); found {
			if response, ok := cachedResponse.(PaginatedResponse); ok {
				return response, nil
			}
		}

		// Get total count of submissions
		totalCount, countSubmsErr := h.submSrvc.CountSubms(r.Context(), search, authorUuid, includeAdmin)
		if countSubmsErr != nil {
			log.Error("failed to count submissions", "error", countSubmsErr)
			return nil, countSubmsErr
		}

		// Get paginated submissions
		subms, err := h.submSrvc.ListSubms(r.Context(), srvc.ListSubmsParams{
			Limit:        limit,
			Offset:       offset,
			Search:       search,
			Author:       authorUuid,
			IncludeAdmin: includeAdmin,
		})
		if err != nil {
			log.Error("failed to list submissions", "error", err)
			return nil, err
		}

		log.Debug("submissions retrieved successfully", "count", len(subms))

		mapSubmList := func(subms []domain.Subm) []SubmListEntry {
			response := make([]SubmListEntry, 0)
			for _, subm := range subms {
				entry, err := h.mapSubmListEntry(r.Context(), subm)
				if err != nil {
					log.Warn("failed to map subm list entry", "error", err)
					continue
				}
				response = append(response, entry)
			}
			return response
		}

		submEntries := mapSubmList(subms)
		log.Info("processed submission list", "count", len(submEntries), "total", totalCount)

		// Create paginated response
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

		// Store in cache for future requests
		h.submCache.Set(cacheKey, paginatedResponse, 0) // Use default expiration time

		return paginatedResponse, nil
	})

	if err != nil {
		jsonresp.HandleErrorWithContext(r.Context(), w, err)
		return
	}

	response := result.(PaginatedResponse)
	log.Info("returning submission list", "count", len(response.Page.([]SubmListEntry)), "total", response.Pagination.Total)
	jsonresp.Success(w, response)
}
