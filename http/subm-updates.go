package http

import (
	"fmt"
	"net/http"
	"time"
)

func (httpserver *HttpServer) listenToSubmUpdates(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a channel for client disconnection
	clientGone := r.Context().Done()

	// Create a flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Send events periodically
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-clientGone:
			// Client disconnected
			return
		case t := <-ticker.C:
			// Send an event
			fmt.Fprintf(w, "event: message\n")
			fmt.Fprintf(w, "data: The time is %v\n\n", t)
			flusher.Flush()
		}
	}
}
