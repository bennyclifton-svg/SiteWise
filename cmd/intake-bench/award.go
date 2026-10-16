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
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"sitewise/internal/eval"
	"sitewise/internal/invoices/sample"
	"sitewise/internal/jev"
	psample "sitewise/internal/proposals/sample"
	"sitewise/internal/store"
)

// proposalModel stands in for System One on proposal questions in a
// replayed bench: it answers as psample.Jev does, after a latency drawn from
// the recorded live proposal calls (data/eval/proposal/private), or 1.5 s
// when that private recording is absent.
type proposalModel struct {
	mu  sync.Mutex
	lat []time.Duration
	i   int
}

func newProposalModel(repo string) *proposalModel {
	m := &proposalModel{}
	if f, err := os.Open(filepath.Join(repo, "data", "eval", "proposal", "private", "recording.jsonl")); err == nil {
		if records, err := eval.ReadRecordings(f); err == nil {
			for _, r := range records {
				if r.Error == "" && r.Status == http.StatusOK && r.LatencyUS > 0 {
					m.lat = append(m.lat, time.Duration(r.LatencyUS)*time.Microsecond)
				}
			}
		}
		f.Close()
	}
	if len(m.lat) == 0 {
		m.lat = []time.Duration{1500 * time.Millisecond}
	}
	sort.Slice(m.lat, func(i, j int) bool { return m.lat[i] < m.lat[j] })
	return m
}

func (m *proposalModel) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	m.mu.Lock()
	d := m.lat[m.i%len(m.lat)]
	m.i++
	m.mu.Unlock()
	return psample.Jev{Delay: d}.Ask(ctx, call)
}

// submissionWatch closes a submission's channel when its read lands.
type submissionWatch struct {
	mu   sync.Mutex
	read map[string]chan struct{}
}

func (w *submissionWatch) wait(id string) chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch, ok := w.read[id]
	if !ok {
		ch = make(chan struct{})
		w.read[id] = ch
	}
	return ch
}

func (w *submissionWatch) seen(id, state string) {
	if id == "" || state == "reading" || state == "" {
		return
	}
	ch := w.wait(id)
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (a *apiClient) procurementStream(ctx context.Context, project string, w *submissionWatch) (func(), error) {
	resp, err := a.do(ctx, http.MethodGet, "/projects/"+project+"/documents", nil, 0)
	if err != nil {
		return nil, err
	}
	var list struct {
		Cursor int64 `json:"cursor"`
	}
	err = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	resp, err = a.do(ctx, http.MethodGet, "/events?after="+strconv.FormatInt(list.Cursor, 10), nil, 0)
	if err != nil || resp.StatusCode != http.StatusOK {
		cancel()
		if err == nil {
			resp.Body.Close()
			err = fmt.Errorf("event stream status %d", resp.StatusCode)
		}
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64<<10), 4<<20)
		for sc.Scan() {
			line := sc.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var ev struct {
				Kind    string `json:"kind"`
				Payload struct {
					SubmissionID string `json:"submission_id"`
					State        string `json:"state"`
				} `json:"payload"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev) == nil && ev.Kind == store.EventProcurement {
				w.seen(ev.Payload.SubmissionID, ev.Payload.State)
			}
		}
	}()
	return func() { cancel(); resp.Body.Close(); <-done }, nil
}

// awardDisciplines each get one package of five proposals: the price table
// budget is for five firms.
var awardDisciplines = []string{"consultant.town_planner", "consultant.traffic", "consultant.civil", "consultant.structural", "consultant.access", "consultant.arborist", "consultant.geotechnical", "consultant.surveyor"}

// procurementAward measures proposal read (upload to the read landing),
// the price table with five firms, a cell write, award and the report.
func (a *apiClient) procurementAward(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Procurement award benchmark")
	if err != nil {
		return err
	}
	watch := &submissionWatch{read: map[string]chan struct{}{}}
	stop, err := a.procurementStream(ctx, project, watch)
	if err != nil {
		return err
	}
	defer stop()
	type pkgRows struct {
		id     string
		labels []string
		keys   []string
	}
	var pkgs []pkgRows
	for _, d := range awardDisciplines {
		body, _ := json.Marshal(map[string]string{"discipline_id": d})
		out, err := a.call(ctx, http.MethodPost, "/projects/"+project+"/procurement/rows", body, http.StatusCreated)
		if err != nil {
			return err
		}
		var p struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(out, &p); err != nil {
			return err
		}
		out, err = a.call(ctx, http.MethodGet, "/projects/"+project+"/procurement/packages/"+p.ID+"/prices", nil, http.StatusOK)
		if err != nil {
			return err
		}
		var t struct {
			Items []struct {
				Key   string `json:"key"`
				Label string `json:"label"`
				Base  bool   `json:"base"`
			} `json:"items"`
		}
		if err := json.Unmarshal(out, &t); err != nil {
			return err
		}
		r := pkgRows{id: p.ID}
		for _, it := range t.Items {
			if it.Base {
				r.labels = append(r.labels, it.Label)
				r.keys = append(r.keys, it.Key)
			}
		}
		pkgs = append(pkgs, r)
	}
	var subs []string
	firm := 0
	for i := 0; i < max(n, 5*len(pkgs)) && i < 5*len(pkgs); i++ {
		p := pkgs[i%len(pkgs)]
		firm++
		var lines []psample.Line
		for j, l := range p.labels {
			lines = append(lines, psample.Line{Label: l, Cents: int64(100000 + 1000*j + 37*firm)})
		}
		pdf := psample.PDF(psample.Proposal{Firm: fmt.Sprintf("Bench Firm %02d Pty Ltd", firm), ABN: sample.ABNFor(500 + firm), Email: fmt.Sprintf("office@benchfirm%02d.test", firm), Lines: lines, Rates: []psample.Line{{Label: "Director", Cents: 40000}}})
		start := time.Now()
		resp, err := a.do(ctx, http.MethodPost, "/projects/"+project+"/procurement/packages/"+p.id+"/submissions?name="+url.QueryEscape(fmt.Sprintf(benchFilePrefix+"fee proposal (Bench Firm %02d).pdf", firm)), bytes.NewReader(pdf), int64(len(pdf)))
		if err != nil {
			return err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("proposal upload %d: %s", resp.StatusCode, raw)
		}
		var up struct {
			SubmissionID string `json:"submission_id"`
		}
		if err := json.Unmarshal(raw, &up); err != nil {
			return err
		}
		select {
		case <-watch.wait(up.SubmissionID):
		case <-time.After(30 * time.Second):
			return errors.New("proposal not read within 30 s")
		case <-ctx.Done():
			return ctx.Err()
		}
		a.samples.Add("proposal_read", time.Since(start))
		if i%len(pkgs) == 0 {
			subs = append(subs, up.SubmissionID)
		}
	}
	first := pkgs[0]
	base := "/projects/" + project + "/procurement/packages/" + first.id
	for i := 0; i < n; i++ {
		if _, err := a.timed(ctx, "price_table", http.MethodGet, base+"/prices", nil, http.StatusOK); err != nil {
			return err
		}
		amount := strconv.Itoa(1000 + i)
		if _, err := a.timed(ctx, "submission_write", http.MethodPut, "/projects/"+project+"/submissions/"+subs[0]+"/cells/"+url.PathEscape(first.keys[0]), []byte(`{"state":"priced","amount":"`+amount+`"}`), http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "procurement_report", http.MethodGet, base+"/report", nil, http.StatusOK); err != nil {
			return err
		}
		award, _ := json.Marshal(map[string]any{"submission_id": subs[i%len(subs)], "lines": []map[string]any{{"label": "Stages", "item_keys": first.keys[:min(3, len(first.keys))]}}})
		if _, err := a.timed(ctx, "award_write", http.MethodPost, base+"/award", award, http.StatusCreated); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "award_write", http.MethodDelete, base+"/award", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

// call is one untimed request whose status must be want.
func (a *apiClient) call(ctx context.Context, method, path string, body []byte, want int) ([]byte, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	resp, err := a.do(ctx, method, path, r, int64(len(body)))
	if err != nil {
		return nil, err
	}
	out, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != want {
		return nil, fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, bytes.TrimSpace(out))
	}
	return out, nil
}
