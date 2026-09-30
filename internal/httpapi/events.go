package httpapi

import (
	"context"
	"net/http"

	"sitewise/internal/events"
)

// streamEvents serves the org's durable event log from the client's cursor.
// EventSource sends Last-Event-ID on its own reconnects; a fresh page passes
// the cursor from its list response as ?after=.
func streamEvents(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	cursor := r.Header.Get("Last-Event-ID")
	if cursor == "" {
		cursor = r.URL.Query().Get("after")
	}
	after, err := events.LastEventID(cursor)
	if err != nil {
		http.Error(w, "bad cursor", http.StatusBadRequest)
		return
	}
	if _, ok := w.(http.Flusher); !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	// The retry hint keeps a dropped stream's reconnect quick without a tight loop.
	_, _ = w.Write([]byte("retry: 2000\n\n"))
	w.(http.Flusher).Flush()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		select {
		case <-deps.Closing:
			cancel()
		case <-ctx.Done():
		}
	}()
	if err := deps.Broker.Serve(ctx, w, session.OrgID, after); err != nil && ctx.Err() == nil {
		deps.Log.Printf("event stream ended org=%s: %v", session.OrgID, err)
	}
}
