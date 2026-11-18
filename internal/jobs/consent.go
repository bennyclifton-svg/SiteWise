package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	intakevocab "sitewise/data/intake"
	"sitewise/internal/consent"
	"sitewise/internal/store"
)

// The consent read (docs/plans/2026-11-17-consent-conditions.md, Task 4),
// in the pattern of the specification import: Go splits the notice, then
// one background Jev fan-out per condition, sixteen in flight, cached by
// source. Every pick is a proposal on a row "to check".

// errNoConditions is a document with no numbered conditions; the strip says
// so in one line.
var errNoConditions = errors.New("No numbered conditions found")

func (w *Worker) consent(ctx context.Context, job store.ClaimedJob) error {
	pages, err := w.consentPages(ctx, job)
	if err != nil {
		return err
	}
	n := consent.Split(pages)
	switch {
	case len(n.Conditions) == 0:
		return errNoConditions
	case len(n.Conditions) > consent.MaxConditions:
		return fmt.Errorf("the notice has more than %d numbered conditions; split the document", consent.MaxConditions)
	}
	notice := store.ConsentNotice{DocumentID: job.DocumentID, ConsentRef: n.ConsentRef, ModRef: n.ModRef, Modification: n.Modification,
		Determined: n.Determined, Conditions: len(n.Conditions), Gaps: n.Gaps}
	if notice.Gaps == nil {
		notice.Gaps = []string{}
	}
	prior, err := w.Store.ConsentPrior(ctx, job.OrgID, job.DocumentID, notice)
	if err != nil {
		return err
	}
	systems := w.consentSystems()
	labels := disciplineLabels()
	rows := make([]store.ConsentRow, len(n.Conditions))
	var mu sync.Mutex
	var askErr error
	err = eachPassage(ctx, len(n.Conditions), func(ctx context.Context, i int) error {
		c := n.Conditions[i]
		call := consent.Call(c, systems)
		print := consent.Fingerprint(call)
		var reading consent.Reading
		if prior[c.Number] == print {
			// Read before with the same state and questions: no call. The
			// store keeps the row's picks.
			reading = consent.Reading{Fingerprint: print, QuestionVersion: consent.QuestionVersion, Fields: map[string]consent.Field{}}
			rows[i] = conditionRow(c, reading, labels, true)
			return nil
		}
		result, err := w.ask(ctx, call)
		if err != nil {
			mu.Lock()
			if askErr == nil {
				askErr = err
			}
			mu.Unlock()
		}
		reading = consent.Apply(c, call, result, err != nil)
		if err != nil {
			// A timeout leaves the picks grey and the fingerprint empty, so
			// the retry asks again.
			reading.Fingerprint = ""
		}
		rows[i] = conditionRow(c, reading, labels, false)
		return nil
	})
	if err != nil {
		return err
	}
	if _, err := w.Store.UpsertConditions(ctx, job, notice, keepCached(rows, prior)); err != nil {
		return err
	}
	// Rule fields are written either way; a failed call fails the job so it
	// retries, and only the grey picks are asked again.
	return askErr
}

// keepCached drops conditions whose reading was cached: their rows are
// already written with the same words and picks.
func keepCached(rows []store.ConsentRow, prior map[string]string) []store.ConsentRow {
	out := rows[:0:0]
	for _, r := range rows {
		if r.Fingerprint != "" && prior[r.Number] == r.Fingerprint {
			continue
		}
		out = append(out, r)
	}
	return out
}

func conditionRow(c consent.Condition, r consent.Reading, labels map[string]string, cached bool) store.ConsentRow {
	raw, _ := json.Marshal(r)
	row := store.ConsentRow{Number: c.Number, Heading: c.Heading, Text: consent.Kept(c), StageHeading: c.StageHeading, Stage: c.Stage,
		Page: c.Page, Deferred: c.Deferred, Marker: c.Marker, MarkerBy: c.MarkerBy, Reading: raw, Fingerprint: r.Fingerprint}
	if cached {
		return row
	}
	if f := r.Fields["timing"]; f.Band == "amber" && row.Stage == "" {
		row.Stage = f.Value
	}
	if f := r.Fields["party"]; f.Band == "amber" {
		row.Discipline = f.Value
		row.OwnerText = consent.PartyLabel(f.Value, labels)
	}
	if f := r.Fields["system"]; f.Band == "amber" && f.Value != consent.NoSystem {
		row.SystemID = f.Value
	}
	if f := r.Fields["evidence"]; f.Band == "amber" {
		yes := f.Value == "true"
		row.EvidenceNeeded = &yes
	}
	return row
}

// consentPages are the notice's stored pages (OCR text for a scan), or a
// fresh extraction when filing stored none.
func (w *Worker) consentPages(ctx context.Context, job store.ClaimedJob) ([]consent.Page, error) {
	src, err := w.Store.SpecificationSource(ctx, job.OrgID, job.DocumentID)
	if errors.Is(err, store.ErrNotFound) && w.Source != nil {
		fresh, e := w.Source(ctx, job.OrgID, job.DocumentID)
		if e != nil {
			return nil, e
		}
		src, err = fresh.Source, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]consent.Page, len(src))
	for i, p := range src {
		out[i] = consent.Page{Page: p.Page, Text: p.Text}
	}
	return out, nil
}

// consentSystems are the catalogue's top-level systems in building order.
func (w *Worker) consentSystems() []consent.System {
	if w.Catalog == nil {
		return nil
	}
	var out []consent.System
	for _, s := range w.Catalog.BuildingOrderSystems() {
		out = append(out, consent.System{ID: s.ID, Label: s.Label})
	}
	return out
}

func disciplineLabels() map[string]string {
	var file struct {
		Disciplines []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"disciplines"`
	}
	_ = json.Unmarshal(intakevocab.Disciplines, &file)
	out := map[string]string{}
	for _, d := range file.Disciplines {
		out[d.ID] = d.Label
	}
	return out
}

// queueConsentRead queues the read of a notice filing recognised: filed as
// a Statutory Instrument, with a notice marker and at least three numbered
// conditions (owner decision 2). Anything else waits for Read conditions.
func (w *Worker) queueConsentRead(ctx context.Context, job store.ClaimedJob, src store.DocumentSource) error {
	kind, err := w.Store.DocumentKind(ctx, job.OrgID, job.DocumentID)
	if err != nil || kind != "statutory_instrument" {
		return err
	}
	pages := make([]consent.Page, len(src.Source))
	for i, p := range src.Source {
		pages[i] = consent.Page{Page: p.Page, Text: p.Text}
	}
	if !consent.Detect(pages) {
		return nil
	}
	return w.Store.QueueConsentAfterFiling(ctx, job.OrgID, job.DocumentID)
}
