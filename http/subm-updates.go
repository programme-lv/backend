package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

func (httpserver *HttpServer) listenToSubmListUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	submCreatedCh, err := httpserver.submSrvc.ListenToNewSubmCreated(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	evalUpdateCh, err := httpserver.submSrvc.ListenToSubmListEvalUpdates(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type evalUpdateStruct struct {
		SubmUuid   string   `json:"subm_uuid"`
		EvalUpdate SubmEval `json:"new_eval"`
	}

	type SubmissionListUpdate struct {
		SubmCreated *Submission       `json:"subm_created"`
		EvalUpdate  *evalUpdateStruct `json:"eval_update"`
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
			message := SubmissionListUpdate{
				SubmCreated: mapSubm(submCreated),
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
			message := SubmissionListUpdate{
				SubmCreated: nil,
				EvalUpdate: &evalUpdateStruct{
					SubmUuid:   evalUpdate.SubmUuid.String(),
					EvalUpdate: mapSubmEval(evalUpdate.Eval),
				},
			}
			marshalled, err := json.Marshal(message)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			slog.Info("eval update", "message", string(marshalled))
			safeWrite("data: " + string(marshalled) + "\n\n")
		}
	}
}
