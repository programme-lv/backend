package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/logger"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
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
	log := logger.FromContext(r.Context())
	log.Info("getting submission list")

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

	log.Debug("pagination parameters", "limit", limit, "offset", offset)

	// Create a cache key based on pagination parameters
	cacheKey := fmt.Sprintf("subm_list:%d:%d", limit, offset)

	// Try to get from cache first
	if cachedResponse, found := h.cache.Get(cacheKey); found {
		if response, ok := cachedResponse.(PaginatedResponse); ok {
			log.Info("returning cached submission list", "limit", limit, "offset", offset)
			httpjson.WriteSuccessJson(w, response)
			return
		}
	}

	// If not in cache or invalid cache, use singleflight to prevent multiple concurrent requests
	// from all hitting the database at the same time
	result, err, _ := h.sfGroup.Do(cacheKey, func() (interface{}, error) {
		// Check cache again in case another request already populated it while we were waiting
		if cachedResponse, found := h.cache.Get(cacheKey); found {
			if response, ok := cachedResponse.(PaginatedResponse); ok {
				return response, nil
			}
		}

		// Get total count of submissions
		totalCount, err := h.submSrvc.CountSubms(r.Context())
		if err != nil {
			log.Error("failed to count submissions", "error", err)
			return nil, err
		}

		// Get paginated submissions
		subms, err := h.submSrvc.ListSubms(r.Context(), submquery.ListSubmsParams{
			Limit:  limit,
			Offset: offset,
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
		h.cache.Set(cacheKey, paginatedResponse, 0) // Use default expiration time

		return paginatedResponse, nil
	})

	if err != nil {
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	response := result.(PaginatedResponse)
	log.Info("returning submission list", "count", len(response.Page.([]SubmListEntry)), "total", response.Pagination.Total)
	httpjson.WriteSuccessJson(w, response)
}
