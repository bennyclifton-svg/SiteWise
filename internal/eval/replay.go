package eval

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ErrReplayMiss means a request has no recording. Replay never falls back to
// the network: a changed question or state needs a new live recording.
var ErrReplayMiss = errors.New("no recorded jev response for this request")

// Record is one exact Jev exchange. Request is the body as sent; it holds the
// identity text of a private document and must stay out of Git. Headers,
// including the credential, are never recorded.
type Record struct {
	Case          string    `json:"case"`
	RequestSHA256 string    `json:"request_sha256"`
	Request       string    `json:"request"`
	Status        int       `json:"status"`
	Response      string    `json:"response,omitempty"`
	Error         string    `json:"error,omitempty"`
	LatencyUS     int64     `json:"latency_us"`
	RecordedAt    time.Time `json:"recorded_at"`
}

// Recorder is a round tripper that forwards to Next and appends each exchange
// to a JSON-lines log.
type Recorder struct {
	next http.RoundTripper
	mu   sync.Mutex
	out  io.Writer
	cur  string
	err  error
}

// NewRecorder records every exchange through next to out.
func NewRecorder(next http.RoundTripper, out io.Writer) *Recorder {
	return &Recorder{next: next, out: out}
}

// SetCase labels the following exchanges.
func (r *Recorder) SetCase(id string) {
	r.mu.Lock()
	r.cur = id
	r.mu.Unlock()
}

// Err is the first write failure, if any.
func (r *Recorder) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// RoundTrip forwards the request and records it with the full response body.
// Only evaluation POSTs are recorded; the client's warm-up HEAD is not.
func (r *Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPost {
		return r.next.RoundTrip(req)
	}
	body, err := readBody(req)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	resp, err := r.next.RoundTrip(req)
	rec := Record{RequestSHA256: hexSHA(body), Request: string(body), RecordedAt: start.UTC()}
	if err != nil {
		rec.Error = err.Error()
		rec.LatencyUS = time.Since(start).Microseconds()
		r.write(rec)
		return nil, err
	}
	payload, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	rec.LatencyUS = time.Since(start).Microseconds()
	rec.Status = resp.StatusCode
	rec.Response = string(payload)
	if readErr != nil {
		rec.Error = readErr.Error()
	}
	r.write(rec)
	resp.Body = io.NopCloser(bytes.NewReader(payload))
	return resp, readErr
}

func (r *Recorder) write(rec Record) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rec.Case = r.cur
	line, err := json.Marshal(rec)
	if err == nil {
		line = append(line, '\n')
		_, err = r.out.Write(line)
	}
	if err != nil && r.err == nil {
		r.err = err
	}
}

// ReadRecordings parses a JSON-lines log. The last record for a request wins.
func ReadRecordings(in io.Reader) ([]Record, error) {
	var out []Record
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for n := 1; sc.Scan(); n++ {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("recording line %d: %w", n, err)
		}
		if rec.RequestSHA256 != hexSHA([]byte(rec.Request)) {
			return nil, fmt.Errorf("recording line %d: request hash does not match its body", n)
		}
		out = append(out, rec)
	}
	return out, sc.Err()
}

// Replayer serves recorded responses by exact request hash. With delay set it
// waits the recorded latency first, honouring the request's deadline, so the
// client's own timing and degraded handling see the recorded provider time.
type Replayer struct {
	byHash map[string]Record
	delay  bool
	mu     sync.Mutex
	misses int
	served int
}

// NewReplayer indexes records by request hash.
func NewReplayer(records []Record, delay bool) *Replayer {
	by := make(map[string]Record, len(records))
	for _, rec := range records {
		by[rec.RequestSHA256] = rec
	}
	return &Replayer{byHash: by, delay: delay}
}

// Misses is how many requests had no recording.
func (p *Replayer) Misses() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.misses
}

// Lookup returns the record for a request body.
func (p *Replayer) Lookup(body []byte) (Record, bool) {
	rec, ok := p.byHash[hexSHA(body)]
	return rec, ok
}

// RoundTrip answers from the recording or fails with ErrReplayMiss.
func (p *Replayer) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := readBody(req)
	if err != nil {
		return nil, err
	}
	rec, ok := p.Lookup(body)
	p.mu.Lock()
	if ok {
		p.served++
	} else {
		p.misses++
	}
	p.mu.Unlock()
	if !ok {
		return nil, ErrReplayMiss
	}
	if p.delay {
		if err := wait(req.Context(), time.Duration(rec.LatencyUS)*time.Microsecond); err != nil {
			return nil, err
		}
	}
	if rec.Error != "" && rec.Status == 0 {
		return nil, fmt.Errorf("recorded transport error: %s", rec.Error)
	}
	return &http.Response{
		StatusCode:    rec.Status,
		Status:        http.StatusText(rec.Status),
		Header:        http.Header{"Content-Type": {"application/json"}},
		Body:          io.NopCloser(bytes.NewReader([]byte(rec.Response))),
		ContentLength: int64(len(rec.Response)),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Request:       req,
	}, nil
}

func readBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
