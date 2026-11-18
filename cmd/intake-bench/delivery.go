package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"sitewise/internal/chatfix"
	"sitewise/internal/delivery"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

// delivery samples the register (docs/plans/2026-11-17-rfis-review-comments.md,
// Task 7): an RFI created and edited (delivery_write), the register read
// with 1,000 rows and their readings (delivery_read), the read with a
// System filter and the also-open line (delivery_system_read), and the
// reply match on a filed document (register_match). n is at least ten, so
// a slow cached generic plan after the fifth call would show.
func (a *apiClient) delivery(ctx context.Context, st *store.Store, n int) error {
	n = max(n, 20)
	project, err := a.createProject(ctx, "Delivery endpoint benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/delivery"
	// 1,000 rows: RFIs on the envelope and the older kinds, each the
	// screen's own write, untimed.
	for i := 0; i < 1000-n; i++ {
		c := map[string]any{"kind": "rfi", "status": "open", "title": fmt.Sprintf("Seeded RFI %d", i), "system_id": "envelope", "details": map[string]any{"raised_on": "2026-11-02"}}
		if i%4 == 0 {
			c = map[string]any{"kind": "risk", "title": fmt.Sprintf("Seeded risk %d", i), "system_id": "substructure"}
		}
		body, _ := json.Marshal(c)
		resp, err := a.do(ctx, http.MethodPost, base, bytes.NewReader(body), int64(len(body)))
		if err != nil {
			return err
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("seeding the register: status %d", resp.StatusCode)
		}
	}
	for i := 0; i < n; i++ {
		body, _ := json.Marshal(map[string]any{"kind": "rfi", "title": fmt.Sprintf("RFI %d", i), "system_id": "envelope", "details": map[string]any{"question": "Which membrane?", "raised_on": "2026-11-20"}})
		raw, err := a.timed(ctx, "delivery_write", http.MethodPost, base, body, http.StatusCreated)
		if err != nil {
			return err
		}
		var item delivery.Item
		if err := json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if item.Number == nil || item.TargetDate == nil || *item.TargetDate != "2026-11-27" {
			return fmt.Errorf("invalid RFI create response")
		}
		raw, err = a.timed(ctx, "delivery_write", http.MethodPatch, base+"/"+item.ID, []byte(`{"version":1,"status":"open"}`), http.StatusOK)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &item); err != nil || item.Status != "open" {
			return fmt.Errorf("invalid RFI edit response")
		}
		raw, err = a.timed(ctx, "delivery_read", http.MethodGet, base, nil, http.StatusOK)
		if err != nil {
			return err
		}
		var view struct {
			Items []delivery.Item `json:"items"`
		}
		if err := json.Unmarshal(raw, &view); err != nil {
			return err
		}
		if len(view.Items) != 1000-n+i+1 {
			return fmt.Errorf("register read omitted rows: %d", len(view.Items))
		}
		if _, err := a.timed(ctx, "delivery_system_read", http.MethodGet, base+"?system=envelope", nil, http.StatusOK); err != nil {
			return err
		}
	}
	return registerMatch(ctx, st, a.samples, project, n)
}

// registerMatch files n replies that cite an open RFI and times the
// register job on each: claim, read the text, match, write the reading.
// It is Go's rule only; the bench calls no Jev.
func registerMatch(ctx context.Context, st *store.Store, samples *collector, project string, n int) error {
	w := &jobs.Worker{Store: st}
	for i := 0; i < n; i++ {
		ids, err := chatfix.SeedDocuments(ctx, st, benchOrg, project, []chatfix.Document{{Title: fmt.Sprintf("Reply %d", i), Kind: "drawing", Pages: []string{fmt.Sprintf("Issued in response to RFI-%03d.", i+1)}}})
		if err != nil {
			return err
		}
		for _, id := range ids {
			if err := st.QueueRegister(ctx, benchOrg, id); err != nil {
				return err
			}
		}
		for _, kind := range []string{store.JobKindFullText, store.JobKindRegister} {
			for {
				start := time.Now()
				job, err := st.ClaimJobSince(ctx, benchOrg, time.Minute, []string{kind}, time.Time{})
				if errors.Is(err, store.ErrIdle) {
					break
				}
				if err != nil {
					return err
				}
				if kind == store.JobKindRegister {
					if err := w.Perform(ctx, job); err != nil {
						return err
					}
				}
				if err := st.CompleteJob(ctx, job.OrgID, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: `{"kind":"` + kind + `","status":"done"}`}); err != nil {
					return err
				}
				if kind == store.JobKindRegister {
					samples.Add("register_match", time.Since(start))
				}
			}
		}
	}
	return nil
}
