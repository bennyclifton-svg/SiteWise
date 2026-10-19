package httpapi

import (
	"context"
	"sync"
	"testing"
	"time"
)

// The review is debounced per project (plan 7.4): a burst of kicks makes one
// run, after the last kick, with the latest session; projects do not share
// a timer; Stop drops what has not started.
func TestProgramReviewDebounce(t *testing.T) {
	var mu sync.Mutex
	runs := map[string][]string{}
	r := &programReviewer{delay: 80 * time.Millisecond, pending: map[string]*pendingReview{},
		run: func(_ context.Context, session, project string) {
			mu.Lock()
			runs[project] = append(runs[project], session)
			mu.Unlock()
		}}
	began := time.Now()
	for i, s := range []string{"s1", "s2", "s3", "s4", "s5"} {
		r.Kick(s, "p1")
		if i == 0 {
			r.Kick("other", "p2")
		}
		time.Sleep(20 * time.Millisecond)
	}
	r.Wait()
	if waited := time.Since(began); waited < 160*time.Millisecond {
		t.Fatalf("ran before the burst went quiet: %v", waited)
	}
	if got := runs["p1"]; len(got) != 1 || got[0] != "s5" {
		t.Fatalf("p1 runs %v", got)
	}
	if got := runs["p2"]; len(got) != 1 {
		t.Fatalf("p2 runs %v", got)
	}
	r.Kick("s6", "p1")
	r.Wait()
	if got := runs["p1"]; len(got) != 2 {
		t.Fatalf("a later kick did not run again: %v", got)
	}
	r.Kick("s7", "p1")
	r.Stop()
	if got := runs["p1"]; len(got) != 2 {
		t.Fatalf("stop ran a pending review: %v", got)
	}
	var nilReviewer *programReviewer
	nilReviewer.Kick("s", "p") // no review configured: nothing happens
	nilReviewer.Stop()
}
