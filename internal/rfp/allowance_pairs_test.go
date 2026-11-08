package rfp

import (
	"sitewise/internal/reports"
	"strings"
	"testing"
)

// Allowances print as two pairs of columns (owner direction, 8 November
// 2026): Item | Qty | Item | Qty, read down the left pair then the right,
// each item carrying its detail. The printed RFP and the screen match.
func TestAllowancesPrintInTwoPairs(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	r := Resolve(d, sk, facts(t, "class=1a", "work_type=new", "procurement=traditional", "pathway=cdc"), Settings{})
	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	var table *reports.Table
	for _, s := range sections {
		for _, b := range s.Blocks {
			if b.ID == "rfp:allowances" {
				table = b.Table
			}
		}
	}
	if table == nil {
		t.Fatal("no allowance table")
	}
	if strings.Join(table.Columns, "|") != "Item|Qty|Item|Qty" {
		t.Fatalf("columns = %v", table.Columns)
	}
	n := len(r.Allowances)
	half := (n + 1) / 2
	if len(table.Rows) != half {
		t.Fatalf("%d allowances in %d rows", n, len(table.Rows))
	}
	for i, a := range r.Allowances {
		row, col := i%half, 0
		if i >= half {
			col = 2
		}
		item := table.Rows[row][col]
		if !strings.HasPrefix(item, a.Label) || (a.Detail != "" && !strings.Contains(strings.ToLower(item), strings.ToLower(a.Detail[1:]))) {
			t.Errorf("allowance %s printed as %q", a.Key, item)
		}
		if table.Rows[row][col+1] != allowanceQty(a) {
			t.Errorf("allowance %s qty %q", a.Key, table.Rows[row][col+1])
		}
	}
}
