package proposals

import "sitewise/internal/rfp"

// Cell is one firm's answer to one schedule item, as stored.
type Cell struct {
	ID        string `json:"id"`
	Key       string `json:"item_key"`
	Label     string `json:"label"`
	Cents     *int64 `json:"-"`
	State     string `json:"state"`
	Basis     string `json:"basis,omitempty"`
	Page      int    `json:"page,omitempty"`
	Passage   string `json:"passage,omitempty"`
	Band      string `json:"band"`
	DecidedBy string `json:"decided_by"`
}

// Column is one firm's totals in the price table. Every figure is computed
// here, never in the browser. BaseTotal is nil while any base item is unread
// or priced without a known amount; KnownBase is the sum of what is known.
type Column struct {
	BaseTotal    *int64
	KnownBase    int64
	NotPriced    int
	Unread       int
	OptionsTotal int64
	OthersTotal  int64
	// Gap is the stated total less the base total, when both are known and
	// they differ by more than a dollar.
	Gap    *int64
	Lowest bool
}

// Totals computes one firm's column from its cells. Options and items the
// firm offered unasked never enter the base total.
func Totals(items []rfp.ScheduleItem, cells map[string]Cell, others []Cell, stated *int64) Column {
	var c Column
	complete := true
	hasBase := false
	for _, it := range items {
		if !it.Price() {
			continue
		}
		hasBase = hasBase || it.Base
		cell, ok := cells[it.Key]
		switch {
		case !ok || cell.State == "":
			if it.Base {
				c.Unread++
				complete = false
			}
		case cell.State == StatePriced && cell.Cents != nil:
			if it.Base {
				c.KnownBase += *cell.Cents
			} else {
				c.OptionsTotal += *cell.Cents
			}
		case cell.State == StatePriced:
			if it.Base {
				complete = false
			}
		case (cell.State == StateNotPriced || cell.State == StateExcluded) && it.Base:
			c.NotPriced++
			complete = false
		}
	}
	for _, o := range others {
		if o.Cents != nil {
			c.OthersTotal += *o.Cents
		}
	}
	if complete && hasBase {
		v := c.KnownBase
		c.BaseTotal = &v
		if stated != nil {
			if gap := *stated - v; gap > 100 || gap < -100 {
				c.Gap = &gap
			}
		}
	}
	return c
}

// MarkLowest flags the lowest base total among complete columns: every base
// item priced or included, none not priced. A cheaper total with items not
// priced is not "lowest"; the screen says why.
func MarkLowest(cols []Column) {
	best := -1
	for i, c := range cols {
		if c.BaseTotal == nil || c.NotPriced > 0 {
			continue
		}
		if best < 0 || *c.BaseTotal < *cols[best].BaseTotal {
			best = i
		}
	}
	if best >= 0 {
		cols[best].Lowest = true
	}
}
