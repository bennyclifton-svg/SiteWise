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
	"strconv"
	"strings"
	"sync"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/xlsx"
)

// programModel stands in for System One on program_row questions in a
// replayed bench: after a latency drawn like the proposal model's (the
// recorded live proposal calls, or 1.5 s), it answers none to every
// question. Program thresholds ship uncalibrated, so any answer would stay
// blank anyway; the bench measures the path, not accuracy.
type programModel struct{ proposal *proposalModel }

func (m programModel) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	m.proposal.mu.Lock()
	d := m.proposal.lat[m.proposal.i%len(m.proposal.lat)]
	m.proposal.i++
	m.proposal.mu.Unlock()
	select {
	case <-time.After(d):
	case <-ctx.Done():
		return jev.Result{}, ctx.Err()
	}
	conf := 0.9
	answers := make(map[string]jev.Answer, len(call.Questions))
	for id := range call.Questions {
		answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "none", Confidence: &conf}
	}
	return jev.Result{Model: "jev-1.13.0", Answers: answers, QuestionVersion: call.QuestionVersion}, nil
}

// importWatch closes a document's channel when its program readings land.
type importWatch struct {
	mu   sync.Mutex
	read map[string]chan struct{}
}

func (w *importWatch) wait(doc string) chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch, ok := w.read[doc]
	if !ok {
		ch = make(chan struct{})
		w.read[doc] = ch
	}
	return ch
}

func (w *importWatch) seen(doc string) {
	ch := w.wait(doc)
	w.mu.Lock()
	defer w.mu.Unlock()
	select {
	case <-ch:
	default:
		close(ch)
	}
}

// programImport measures program_import (plan 11.1): a program file dropped
// on Program, from the upload to its readings written, which is filing
// plus one background program_row fan-out, the same shape as proposal_read.
func (a *apiClient) programImport(ctx context.Context, n int) error {
	project, err := a.createProject(ctx, "Program import benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/program"
	out, err := a.call(ctx, http.MethodPost, base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK)
	if err != nil {
		return err
	}
	var v struct {
		Program struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"program"`
		Activities []struct {
			Label string `json:"label"`
		} `json:"activities"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return err
	}
	watch := &importWatch{read: map[string]chan struct{}{}}
	stop, err := a.programStream(ctx, project, watch)
	if err != nil {
		return err
	}
	defer stop()
	for i := 0; i < n; i++ {
		file, err := programBenchFile(v.Activities, i)
		if err != nil {
			return err
		}
		start := time.Now()
		resp, err := a.do(ctx, http.MethodPost, base+"/import?name="+url.QueryEscape(fmt.Sprintf(benchFilePrefix+"builder program %02d.xlsx", i)), bytes.NewReader(file), int64(len(file)))
		if err != nil {
			return err
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf("program import %d: %s", resp.StatusCode, raw)
		}
		var up struct {
			DocumentID string `json:"document_id"`
		}
		if err := json.Unmarshal(raw, &up); err != nil {
			return err
		}
		select {
		case <-watch.wait(up.DocumentID):
		case <-time.After(30 * time.Second):
			return errors.New("program import not read within 30 s")
		case <-ctx.Done():
			return ctx.Err()
		}
		a.samples.Add("program_import", time.Since(start))
	}
	return nil
}

// programBenchFile is a builder's program of the generated rows: every
// row in a builder's words, and every second one also by its own label
// (settled by rule), so every sample is rules plus one fan-out. i makes the bytes, and
// so the document, new each time.
func programBenchFile(acts []struct {
	Label string `json:"label"`
}, i int) ([]byte, error) {
	txt := func(s string) xlsx.Cell { return xlsx.Cell{Text: s} }
	rows := [][]xlsx.Cell{{txt("Bench program " + strconv.Itoa(i))}, {txt("ID"), txt("Task Name"), txt("Duration"), txt("Start"), txt("Finish"), txt("Predecessors")}}
	day := 1
	for k, act := range acts {
		labels := []string{act.Label + " works package " + strconv.Itoa(i)}
		if k%2 == 1 {
			labels = append([]string{act.Label}, labels...)
		}
		for _, label := range labels {
			id := strconv.Itoa(len(rows) - 1)
			pred := ""
			if k > 0 {
				pred = strconv.Itoa(len(rows) - 2)
			}
			rows = append(rows, []xlsx.Cell{txt(id), txt(label), txt("5 days"), txt(fmt.Sprintf("%d/03/2027", day)), txt(fmt.Sprintf("%d/03/2027", day+4)), txt(pred)})
			day = day%20 + 1
		}
	}
	var b bytes.Buffer
	err := xlsx.WriteXLSX(&b, []xlsx.Sheet{{Name: "Program", Rows: rows}})
	return b.Bytes(), err
}

func (a *apiClient) programStream(ctx context.Context, project string, w *importWatch) (func(), error) {
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
					DocumentID string `json:"document_id"`
				} `json:"payload"`
			}
			if json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev) == nil && ev.Kind == "program.readings" {
				w.seen(ev.Payload.DocumentID)
			}
		}
	}()
	return func() { cancel(); resp.Body.Close(); <-done }, nil
}
