package main

import (
	"fmt"

	"sitewise/internal/knowledge"
	"sitewise/internal/rfp"
	"sitewise/internal/tender"
)

// rowsFor is the default price schedule of one RFP or RFT, or of several as
// a combined engagement, as the rows a tender pass is offered. No profile
// facts and no settings: a tender that arrives before its RFP is issued is
// read into the same default rows.
func rowsFor(cat *knowledge.Catalog, ids []string) ([]tender.Row, error) {
	var rows []tender.Row
	for _, id := range ids {
		d, ok := cat.RFP(id)
		if !ok {
			return nil, fmt.Errorf("no RFP knowledge for %q", id)
		}
		sk, ok := cat.RFPSkeletonFor(d)
		if !ok {
			return nil, fmt.Errorf("no RFP skeleton for %q", id)
		}
		for _, it := range rfp.Schedule(rfp.Resolve(d, sk, rfp.DriverSet{Facts: map[string]rfp.Fact{}, Text: map[string]string{}}, rfp.Settings{})) {
			if !it.Price() || !it.Base {
				continue
			}
			row := tender.Row{Key: it.Key, Label: it.Label, Detail: it.Detail}
			if len(ids) > 1 {
				row.Key, row.Label = id+":"+it.Key, d.Title+": "+it.Label
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}
