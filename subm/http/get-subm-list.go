package http

import (
	"net/http"
	"strconv"

	"github.com/programme-lv/backend/httpjson"
	"github.com/programme-lv/backend/logger"
	"github.com/programme-lv/backend/subm/domain"
	"github.com/programme-lv/backend/subm/submsrvc/submquery"
)

// PaginatedResponse represents a paginated response with data and pagination metadata
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
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

	// Get total count of submissions
	totalCount, err := h.submSrvc.CountSubms(r.Context())
	if err != nil {
		log.Error("failed to count submissions", "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
	}

	// Get paginated submissions
	subms, err := h.submSrvc.ListSubms(r.Context(), submquery.ListSubmsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		log.Error("failed to list submissions", "error", err)
		httpjson.HandleErrorWithContext(*r, w, err)
		return
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
	log.Info("returning submission list", "count", len(submEntries), "total", totalCount)

	// Create paginated response
	hasMore := offset+len(submEntries) < totalCount
	paginatedResponse := PaginatedResponse{
		Data: submEntries,
		Pagination: Pagination{
			Total:   totalCount,
			Offset:  offset,
			Limit:   limit,
			HasMore: hasMore,
		},
	}

	httpjson.WriteSuccessJson(w, paginatedResponse)
}
