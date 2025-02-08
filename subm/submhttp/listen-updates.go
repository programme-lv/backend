package submhttp

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

func (h *SubmHttpHandler) ListenToSubmListUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	submCreatedCh, err := h.submSrvc.SubscribeNewSubms(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	evalUpdateCh, err := h.submSrvc.SubscribeEvalUpds(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type SubmissionListUpdate struct {
		SubmCreated *SubmListEntry `json:"subm_created"`
		EvalUpdate  *Eval          `json:"eval_update"`
	}

	var writeMutex sync.Mutex
	safeWrite := func(data string) {
		writeMutex.Lock()
		defer writeMutex.Unlock()
		io.WriteString(w, data)
		flusher.Flush()
	}

	keepAliveTicker := time.NewTicker(15 * time.Second)
	defer keepAliveTicker.Stop()

	for {
		select {
		case <-keepAliveTicker.C:
			safeWrite(": keep-alive\n\n")
		case submCreated, ok := <-submCreatedCh:
			if !ok {
				return
			}
			entry, err := h.mapSubmListEntry(r.Context(), submCreated)
			if err != nil {
				slog.Default().Warn("failed to map subm list entry", "error", err, "subm_uuid", submCreated.UUID)
				continue
			}
			message := SubmissionListUpdate{
				SubmCreated: &entry,
			}
			marshalled, err := json.Marshal(message)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			safeWrite("data: " + string(marshalled) + "\n\n")
		case evalUpdate, ok := <-evalUpdateCh:
			if !ok {
				return
			}
			mappedEval := mapSubmEval(evalUpdate)
			message := SubmissionListUpdate{
				EvalUpdate: &mappedEval,
			}
			marshalled, err := json.Marshal(message)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// slog.Info("eval update", "message", string(marshalled))
			safeWrite("data: " + string(marshalled) + "\n\n")
		}
	}
}
