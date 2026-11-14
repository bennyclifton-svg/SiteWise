package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sitewise/internal/llm"
	"sitewise/internal/tender"
)

// record is one exchange with the model. Answers are recorded at the
// llm.Asker seam, not at HTTP: the SDK streams, and a streamed exchange is
// awkward to replay. The file holds real prices and is never committed.
type record struct {
	Case      string    `json:"case"`
	Prompt    string    `json:"prompt"` // tender.PromptVersion
	Model     string    `json:"model"`
	Request   string    `json:"request"` // sha256 of the request: model, system, schema, every part
	Text      string    `json:"text"`
	Stop      string    `json:"stop"`
	InTokens  int64     `json:"in_tokens"`
	OutTokens int64     `json:"out_tokens"`
	LatencyMS int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	At        time.Time `json:"at"`
}

var errReplayMiss = errors.New("no recorded answer for this request")

// requestKey hashes everything the model is sent.
func requestKey(req llm.Request) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	enc.Encode([]any{req.Model, req.System, req.Schema, req.MaxTokens})
	for _, p := range req.Parts {
		enc.Encode([]string{p.Text, p.ImageJPEG})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func loadRecords(path string) (map[string]record, error) {
	out := map[string]record{}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	for sc.Scan() {
		var r record
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.Error == "" {
			out[r.Request] = r // the latest answer for a request wins
		}
	}
	return out, sc.Err()
}

// replayer answers from the file. With a live asker behind it, a request
// that has no recording (or every request, when fresh) is asked and
// recorded.
type replayer struct {
	mu      sync.Mutex
	path    string
	records map[string]record
	live    llm.Asker
	fresh   bool
	caseID  string
	asked   int
	lastKey string
}

// forgetLast drops the answer just replayed, so the same request is asked
// again: used when the model listed nothing.
func (r *replayer) forgetLast() {
	r.mu.Lock()
	delete(r.records, r.lastKey)
	r.mu.Unlock()
}

func newReplayer(path string, live llm.Asker, fresh bool) (*replayer, error) {
	records, err := loadRecords(path)
	if err != nil {
		return nil, err
	}
	return &replayer{path: path, records: records, live: live, fresh: fresh}, nil
}

func (r *replayer) Ask(ctx context.Context, req llm.Request) (llm.Response, error) {
	key := requestKey(req)
	r.mu.Lock()
	rec, ok := r.records[key]
	id := r.caseID
	r.lastKey = key
	r.mu.Unlock()
	if ok && !(r.fresh && r.live != nil) {
		return llm.Response{Text: rec.Text, StopReason: rec.Stop, InTokens: rec.InTokens, OutTokens: rec.OutTokens, Latency: time.Duration(rec.LatencyMS) * time.Millisecond}, nil
	}
	if r.live == nil {
		return llm.Response{}, errReplayMiss
	}
	res, err := r.live.Ask(ctx, req)
	rec = record{Case: id, Prompt: tender.PromptVersion, Model: req.Model, Request: key, Text: res.Text, Stop: res.StopReason,
		InTokens: res.InTokens, OutTokens: res.OutTokens, LatencyMS: res.Latency.Milliseconds(), At: time.Now().UTC()}
	if err != nil {
		rec.Error = "failed" // never the provider's own words: they may quote the request
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asked++
	if werr := appendRecord(r.path, rec); werr != nil {
		return res, werr
	}
	if err == nil {
		r.records[key] = rec
	}
	return res, err
}

func appendRecord(path string, rec record) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(rec)
	_, err = f.Write(append(b, '\n'))
	return err
}
