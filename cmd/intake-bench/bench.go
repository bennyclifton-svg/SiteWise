package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sitewise/internal/eval"
)

// collector holds microsecond samples per budget path. It is the intake
// Observer, so it is called from concurrent filings.
type collector struct {
	mu      sync.Mutex
	samples map[string][]int64
}

func newCollector() *collector {
	return &collector{samples: map[string][]int64{}}
}

func (c *collector) Add(path string, d time.Duration) {
	c.mu.Lock()
	c.samples[path] = append(c.samples[path], d.Microseconds())
	c.mu.Unlock()
}

func (c *collector) Snapshot() map[string][]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string][]int64, len(c.samples))
	for k, v := range c.samples {
		out[k] = append([]int64(nil), v...)
	}
	return out
}

// stateReplayer serves a recorded response for the document a request is
// about, keyed by its state, after waiting the recorded provider latency.
// Bench documents get fresh ids, so a supersession question can differ from
// the recording; its answer is then rejected by validation and stays blank.
// Only the timing claim depends on the recording. A miss fails the bench: an
// instant error would make filings look faster than the provider is.
type stateReplayer struct {
	byState map[string]eval.Record
	misses  atomic.Int64
}

func newStateReplayer(records []eval.Record) (*stateReplayer, error) {
	r := &stateReplayer{byState: map[string]eval.Record{}}
	for _, rec := range records {
		key, err := stateKey([]byte(rec.Request))
		if err != nil {
			return nil, fmt.Errorf("recording for %s: %w", rec.Case, err)
		}
		r.byState[key] = rec
	}
	return r, nil
}

func stateKey(body []byte) (string, error) {
	var req struct {
		State json.RawMessage `json:"state"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", err
	}
	if len(req.State) == 0 {
		return "", errors.New("request has no state")
	}
	return eval.SHA256Hex(req.State), nil
}

func (r *stateReplayer) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	key, err := stateKey(body)
	if err != nil {
		return nil, err
	}
	rec, ok := r.byState[key]
	if !ok {
		r.misses.Add(1)
		return nil, eval.ErrReplayMiss
	}
	t := time.NewTimer(time.Duration(rec.LatencyUS) * time.Microsecond)
	defer t.Stop()
	select {
	case <-t.C:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	if rec.Status == 0 {
		return nil, fmt.Errorf("recorded transport error: %s", rec.Error)
	}
	return &http.Response{
		StatusCode: rec.Status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(rec.Response)),
		Request:    req,
	}, nil
}

// degrading stalls every nth request until its deadline, standing in for a
// provider that does not answer. The client's deadline turns it into grey
// fields, which is the degraded path the gate must include.
type degrading struct {
	next  http.RoundTripper
	every int64
	n     atomic.Int64
	stall atomic.Int64
}

func (d *degrading) RoundTrip(req *http.Request) (*http.Response, error) {
	if d.every > 0 && d.n.Add(1)%d.every == 0 {
		d.stall.Add(1)
		<-req.Context().Done()
		return nil, req.Context().Err()
	}
	return d.next.RoundTrip(req)
}

// lastByte reports when the transport has read the final upload byte, which
// is where the intake gate starts.
type lastByte struct {
	r    io.Reader
	left int64
	at   time.Time
	done chan struct{}
}

func newLastByte(body []byte) *lastByte {
	return &lastByte{r: bytes.NewReader(body), left: int64(len(body)), done: make(chan struct{})}
}

func (l *lastByte) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.left -= int64(n)
	if l.left <= 0 && l.at.IsZero() {
		l.at = time.Now()
		close(l.done)
	}
	return n, err
}

// event is one server-sent event line set.
type event struct {
	ID         int64  `json:"id"`
	Kind       string `json:"kind"`
	DocumentID string `json:"document_id"`
}

// readEvents parses an event stream and calls on for each data line until
// the stream ends.
func readEvents(r io.Reader, on func(event, time.Time)) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			return err
		}
		on(ev, time.Now())
	}
	return sc.Err()
}

// arrivals is when each document's first terminal event reached the client.
type arrivals struct {
	mu    sync.Mutex
	at    map[string]time.Time
	kind  map[string]string
	wait  map[string]chan struct{}
	maxID int64
}

func newArrivals() *arrivals {
	return &arrivals{at: map[string]time.Time{}, kind: map[string]string{}, wait: map[string]chan struct{}{}}
}

// terminal events end one filing attempt: filed, not filed, or failed.
var terminal = map[string]bool{"filing": true, "not_filed": true, "filing_failed": true}

func (a *arrivals) record(ev event, t time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ev.ID > a.maxID {
		a.maxID = ev.ID
	}
	if !terminal[ev.Kind] || ev.DocumentID == "" {
		return
	}
	if _, seen := a.at[ev.DocumentID]; seen {
		return
	}
	a.at[ev.DocumentID] = t
	a.kind[ev.DocumentID] = ev.Kind
	if ch, ok := a.wait[ev.DocumentID]; ok {
		close(ch)
		delete(a.wait, ev.DocumentID)
	}
}

// await returns the arrival time and kind for a document, or ctx's error.
func (a *arrivals) await(ctx context.Context, documentID string) (time.Time, string, error) {
	a.mu.Lock()
	if t, ok := a.at[documentID]; ok {
		k := a.kind[documentID]
		a.mu.Unlock()
		return t, k, nil
	}
	ch, ok := a.wait[documentID]
	if !ok {
		ch = make(chan struct{})
		a.wait[documentID] = ch
	}
	a.mu.Unlock()
	select {
	case <-ch:
		a.mu.Lock()
		defer a.mu.Unlock()
		return a.at[documentID], a.kind[documentID], nil
	case <-ctx.Done():
		return time.Time{}, "", ctx.Err()
	}
}

func (a *arrivals) cursor() int64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.maxID
}
