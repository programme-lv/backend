package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/programme-lv/backend/common/jsonresp"
)

func (h *UserHttpHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	userUUID, ok := requireUserUUID(w, r)
	if !ok {
		return
	}

	var request struct {
		HideTaskTags *bool `json:"hide_task_tags"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.HideTaskTags == nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	if err := h.userSrvc.UpdatePreferences(r.Context(), userUUID, *request.HideTaskTags); err != nil {
		jsonresp.HandleSrvcError(slog.Default(), w, err)
		return
	}
	jsonresp.Success(w, struct {
		HideTaskTags bool `json:"hide_task_tags"`
	}{*request.HideTaskTags})
}
