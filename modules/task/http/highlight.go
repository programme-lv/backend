package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/programme-lv/backend/common/jsonresp"
	"github.com/programme-lv/backend/modules/task/highlight"
)

type highlightStore interface {
	List(context.Context) ([]highlight.Tag, error)
	Create(context.Context, string, string, ...highlight.Aliases) (highlight.Tag, error)
	Edit(context.Context, string, string, bool, int64, ...highlight.Aliases) (highlight.Tag, error)
	Get(context.Context, string) (highlight.Assignment, error)
	Replace(context.Context, string, []string, int64) (highlight.Assignment, error)
	Batch(context.Context, []string) (map[string][]highlight.Label, error)
}

// WithHighlightStore enables editorial highlight management and public labels.
func WithHighlightStore(s highlightStore) HandlerOption {
	return func(h *taskHttpHandler) { h.highlights = s }
}

// optionalHighlightString distinguishes omission from an explicit empty value.
type optionalHighlightString struct{ value *string }

func (o *optionalHighlightString) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return fmt.Errorf("highlight alias must be a string")
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.value = &value
	return nil
}

type createHighlightRequest struct {
	Abbreviation optionalHighlightString `json:"abbreviation"`
	EnglishLabel optionalHighlightString `json:"english_label"`
	ID           *string                 `json:"id"`
	Label        *string                 `json:"label"`
}
type editHighlightRequest struct {
	Abbreviation     optionalHighlightString `json:"abbreviation"`
	EnglishLabel     optionalHighlightString `json:"english_label"`
	Label            *string                 `json:"label"`
	Archived         *bool                   `json:"archived"`
	ExpectedRevision *int64                  `json:"expected_revision"`
}
type replaceHighlightsRequest struct {
	HighlightedTags  []string `json:"highlighted_tags"`
	ExpectedRevision *int64   `json:"expected_revision"`
}

func (h *taskHttpHandler) highlightError(ctx context.Context, err error) jsonresp.HttpStatusCoder {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, highlight.ErrInvalid):
		return jsonresp.ErrHttpBadRequest.WithMsg("invalid highlight input")
	case errors.Is(err, highlight.ErrConflict):
		return jsonresp.HttpError{ErrCode: "conflict", ErrMsg: "highlight revision conflict or duplicate ID", HttpStatus: http.StatusConflict}
	case errors.Is(err, highlight.ErrNotFound):
		return jsonresp.HttpError{ErrCode: "not_found", ErrMsg: "highlight or task not found", HttpStatus: http.StatusNotFound}
	default:
		h.logger(ctx).Error("highlight database operation", "error", err)
		return jsonresp.ErrInternalServerError
	}
}
func (h *taskHttpHandler) listHighlights(ctx context.Context) ([]highlight.Tag, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return nil, jsonresp.ErrInternalServerError
	}
	v, e := h.highlights.List(ctx)
	return v, h.highlightError(ctx, e)
}
func (h *taskHttpHandler) createHighlight(ctx context.Context, r createHighlightRequest) (highlight.Tag, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return highlight.Tag{}, jsonresp.ErrInternalServerError
	}
	if r.ID == nil || r.Label == nil {
		return highlight.Tag{}, jsonresp.ErrHttpBadRequest
	}
	v, e := h.highlights.Create(ctx, *r.ID, *r.Label, highlight.Aliases{Abbreviation: r.Abbreviation.value, EnglishLabel: r.EnglishLabel.value})
	return v, h.highlightError(ctx, e)
}
func (h *taskHttpHandler) editHighlight(ctx context.Context, r editHighlightRequest) (highlight.Tag, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return highlight.Tag{}, jsonresp.ErrInternalServerError
	}
	if r.Label == nil || r.Archived == nil || r.ExpectedRevision == nil {
		return highlight.Tag{}, jsonresp.ErrHttpBadRequest
	}
	v, e := h.highlights.Edit(ctx, chi.URLParamFromCtx(ctx, "tagId"), *r.Label, *r.Archived, *r.ExpectedRevision, highlight.Aliases{Abbreviation: r.Abbreviation.value, EnglishLabel: r.EnglishLabel.value})
	return v, h.highlightError(ctx, e)
}
func (h *taskHttpHandler) getHighlights(ctx context.Context) (highlight.Assignment, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return highlight.Assignment{}, jsonresp.ErrInternalServerError
	}
	v, e := h.highlights.Get(ctx, chi.URLParamFromCtx(ctx, "taskId"))
	return v, h.highlightError(ctx, e)
}
func (h *taskHttpHandler) replaceHighlights(ctx context.Context, r replaceHighlightsRequest) (highlight.Assignment, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return highlight.Assignment{}, jsonresp.ErrInternalServerError
	}
	if r.HighlightedTags == nil || r.ExpectedRevision == nil {
		return highlight.Assignment{}, jsonresp.ErrHttpBadRequest
	}
	v, e := h.highlights.Replace(ctx, chi.URLParamFromCtx(ctx, "taskId"), r.HighlightedTags, *r.ExpectedRevision)
	return v, h.highlightError(ctx, e)
}
func (h *taskHttpHandler) addHighlights(ctx context.Context, previews []TaskPreview) ([]TaskPreview, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return previews, nil
	}
	ids := make([]string, len(previews))
	for i, p := range previews {
		ids[i] = p.ShortId
	}
	tags, err := h.highlights.Batch(ctx, ids)
	if err != nil {
		return nil, h.highlightError(ctx, err)
	}
	result := append([]TaskPreview{}, previews...)
	for i := range result {
		result[i].HighlightedTags = tags[result[i].ShortId]
	}
	return result, nil
}
func (h *taskHttpHandler) addViewHighlights(ctx context.Context, t Task) (Task, jsonresp.HttpStatusCoder) {
	if h.highlights == nil {
		return t, nil
	}
	tags, err := h.highlights.Batch(ctx, []string{t.ShortTaskID})
	if err != nil {
		return Task{}, h.highlightError(ctx, err)
	}
	t.HighlightedTags = tags[t.ShortTaskID]
	return t, nil
}
