package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func sseHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming unsupported"})
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ch := svc.StreamHub().Register()
		defer svc.StreamHub().Unregister(ch)

		_, _ = fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"ok\"}\n\n")
		flusher.Flush()

		heartbeat := time.NewTicker(20 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case msg := <-ch:
				_, _ = fmt.Fprintf(w, "event: status\ndata: %s\n\n", msg)
				flusher.Flush()
			case <-heartbeat.C:
				_, _ = fmt.Fprint(w, "event: heartbeat\ndata: {}\n\n")
				flusher.Flush()
			case <-r.Context().Done():
				return
			}
		}
	}
}
