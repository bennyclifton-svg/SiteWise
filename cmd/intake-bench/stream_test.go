package main

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// An event stream must outlive the client's per-request Timeout: the award
// step reads forty proposals over ~90 s on one stream, and a 60 s Timeout
// cut it silently, so a read that landed was never seen.
func TestEventStreamOutlivesClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: %d\n\n", i)
			w.(http.Flusher).Flush()
			select {
			case <-time.After(150 * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
	}))
	defer srv.Close()
	a := &apiClient{origin: srv.URL, http: &http.Client{Timeout: 100 * time.Millisecond}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := a.do(ctx, http.MethodGet, "/events?after=0", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	got := 0
	for sc.Scan() {
		if sc.Text() != "" {
			got++
		}
	}
	if got != 3 || sc.Err() != nil {
		t.Fatalf("stream delivered %d of 3 events, err %v", got, sc.Err())
	}
}
