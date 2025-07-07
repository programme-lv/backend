package http

import (
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/httpjson"
)

const taskPreviewListCacheKey = "task_preview_list"

type TaskPreview struct {
	ShortId          string             `json:"short_id"`
	FullName         string             `json:"full_name"`
	IllustrImg       *IllustrationImage `json:"illustr_img"`
	DifficultyRating int                `json:"difficulty_rating"`
	OriginOlympiad   string             `json:"origin_olympiad"`
	OriginNote       string             `json:"origin_note"`
	MdStatementStory string             `json:"md_statement_story"`
}

func (h *TaskHttpHandler) GetTaskPreviewList(w http.ResponseWriter, r *http.Request) {
	// Try to get from cache
	if cached, found := h.cache.Get(taskPreviewListCacheKey); found {
		if previews, ok := cached.([]TaskPreview); ok {
			httpjson.Success(w, previews)
			return
		}
	}

	// Use singleflight to prevent cache stampede
	result, err, _ := h.sfGroup.Do(taskPreviewListCacheKey, func() (interface{}, error) {
		if cached, found := h.cache.Get(taskPreviewListCacheKey); found {
			if previews, ok := cached.([]TaskPreview); ok {
				return previews, nil
			}
		}
		tasks, err := h.taskSrvc.ListTaskPreviews(r.Context())
		if err != nil {
			return nil, err
		}
		previews := make([]TaskPreview, 0, len(tasks))
		for _, t := range tasks {
			previews = append(previews, h.mapTaskPreview(t))
		}
		h.cache.Set(taskPreviewListCacheKey, previews, 0)
		return previews, nil
	})

	if err != nil {
		httpjson.HandleSrvcError(slog.Default(), w, err)
		return
	}

	previewList, _ := result.([]TaskPreview)
	httpjson.Success(w, previewList)
}
