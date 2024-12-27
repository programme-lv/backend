package http

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

func (httpserver *HttpServer) listenToSubmUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	ch, err := httpserver.submSrvc.ListenToNewSubmCreated(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type SubmissionListUpdate struct {
		SubmCreated *Submission `json:"subm_created"`
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
		case update, ok := <-ch:
			if !ok {
				return
			}
			message := SubmissionListUpdate{
				SubmCreated: mapSubm(update),
			}
			marshalled, err := json.Marshal(message)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			safeWrite("data: " + string(marshalled) + "\n\n")
		}
	}
}
