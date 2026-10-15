package jobs

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"sitewise/internal/specifications"
	"sitewise/internal/store"
)

func (w *Worker) specification(ctx context.Context, job store.ClaimedJob) error {
	if w.Ask == nil {
		return errors.New("specification reader unavailable")
	}
	i, err := w.Store.SpecificationImport(ctx, job.OrgID, job.DocumentID)
	if err != nil {
		return err
	}
	pages, err := w.Store.SpecificationSource(ctx, job.OrgID, job.DocumentID)
	missing := errors.Is(err, store.ErrNotFound)
	geometryMissing := false
	if err == nil && w.Source != nil {
		doc, e := w.Store.GetDocument(ctx, job.OrgID, job.DocumentID)
		if e != nil {
			return e
		}
		geometryMissing = strings.EqualFold(filepath.Ext(doc.Filename), ".pdf")
		for _, p := range pages {
			if len(p.Regions) > 0 {
				geometryMissing = false
				break
			}
		}
	}
	if (missing || geometryMissing) && w.Source != nil {
		source, e := w.Source(ctx, job.OrgID, job.DocumentID)
		if e != nil {
			return e
		}
		if missing {
			pages = source.Source
		} else {
			pages = sourceWithGeometry(pages, source.Source)
		}
		err = nil
	}
	if err != nil {
		return err
	}
	input := make([]specifications.Page, 0, len(pages))
	for _, p := range pages {
		page := specifications.Page{Page: p.Page, Location: p.Location, Text: p.Text}
		for _, r := range p.Regions {
			page.Regions = append(page.Regions, specifications.Region{Text: r.Text, X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
		}
		input = append(input, page)
	}
	candidates := specifications.Harvest(input)
	if len(candidates) == 0 {
		return errors.New("no readable specification text; add details manually or provide a text-bearing document")
	}
	if len(candidates) > 1000 {
		return errors.New("specification exceeds 1000 candidate items; split the document before importing")
	}
	snapshot, err := w.Store.ReadSpecifications(ctx, job.OrgID, i.ProjectID)
	if err != nil {
		return err
	}
	sections := specifications.Sections(w.Catalog, snapshot)
	return eachPassage(ctx, len(candidates), func(ctx context.Context, n int) error {
		c := candidates[n]
		call := specifications.WorkspaceImportCall(c, snapshot, sections)
		fingerprint := specifications.WorkspaceFingerprint(call)
		result, found, err := w.Store.CachedSpecificationReading(ctx, job.OrgID, job.DocumentID, c.Key, fingerprint)
		if err != nil {
			return err
		}
		if _, ok := result.Answers["item"]; !ok {
			found = false
		}
		if !found {
			result, err = w.Ask.Ask(ctx, call)
			if err != nil {
				return err
			}
			if _, ok := result.Answers["item"]; !ok {
				return fmt.Errorf("item reading unavailable for source page %d; retry import", c.Page)
			}
			if err = w.Store.SaveSpecificationReading(ctx, job, c.Key, fingerprint, result); err != nil {
				return err
			}
		}
		answer, ok := result.Answers["item"]
		if !ok {
			return fmt.Errorf("item reading unavailable for source page %d; retry import", c.Page)
		}
		if answer.Choice != "yes" {
			return nil
		}
		return w.Store.SaveSpecificationCandidate(ctx, job, i, c, result)
	})
}

// Older cached PDF sources have no run geometry. Refresh only native pages for
// this import; do not replace profile passages/facts or lose recovered OCR text.
func sourceWithGeometry(cached, fresh []store.SourcePage) []store.SourcePage {
	byPage := map[int]store.SourcePage{}
	for _, p := range fresh {
		if len(p.Regions) > 0 {
			byPage[p.Page] = p
		}
	}
	out := append([]store.SourcePage(nil), cached...)
	for i, p := range out {
		if native, ok := byPage[p.Page]; ok {
			out[i] = native
		}
	}
	return out
}
