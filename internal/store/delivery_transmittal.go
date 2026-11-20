package store

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
	"sitewise/internal/rfp"
)

// rfiSourceIDs are the documents attached to a register row, in the order
// attached, each once.
func rfiSourceIDs(it delivery.Item) []string {
	var p struct {
		Sources []struct {
			DocumentID string `json:"document_id"`
		} `json:"sources"`
	}
	if json.Unmarshal(it.Provenance, &p) != nil {
		return nil
	}
	seen, ids := map[string]bool{}, []string{}
	for _, s := range p.Sources {
		if s.DocumentID != "" && !seen[s.DocumentID] {
			seen[s.DocumentID] = true
			ids = append(ids, s.DocumentID)
		}
	}
	return ids
}

// rfiTransmittals are each RFI's attached documents as the RFP transmittal
// prints them: the same saved identity values, read in one query for the
// whole register. A document deleted since is left out.
func rfiTransmittals(ctx context.Context, tx pgx.Tx, org, project string, items []delivery.Item) (map[string][][]string, error) {
	byItem, all := map[string][]string{}, []string{}
	for _, it := range items {
		if it.Kind != "rfi" {
			continue
		}
		if ids := rfiSourceIDs(it); len(ids) > 0 {
			byItem[it.ID] = ids
			all = append(all, ids...)
		}
	}
	out := map[string][][]string{}
	if len(all) == 0 {
		return out, nil
	}
	values, err := reportDocumentScheduleSelection(ctx, tx, org, project, all)
	if err != nil {
		return nil, err
	}
	rows := map[string][]string{}
	for _, d := range rfp.DocumentRows(values) {
		rows[d.ID] = []string{d.Number, d.Title, d.Revision, d.Date, d.Discipline}
	}
	for item, ids := range byItem {
		for _, id := range ids {
			if row, ok := rows[id]; ok {
				out[item] = append(out[item], row)
			}
		}
	}
	return out, nil
}
