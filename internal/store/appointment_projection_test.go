package store

import (
	"encoding/json"
	"testing"

	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

func TestAppointmentBundleAndUnknownAmounts(t *testing.T) {
	amount := func(s string) *string { return &s }
	base := int64(100000)
	items := []rfp.ScheduleItem{{Key: "stage:concept", Label: "Concept", Kind: "stage", Base: true}, {Key: "stage:documents", Label: "Documents", Kind: "stage", Base: true}, {Key: "stage:site", Label: "Site services", Kind: "stage", Base: true}}
	col := FirmColumn{Submission: Submission{GSTBasis: "ex", StatedTotal: &base}, Cells: map[string]proposals.Cell{}, Decisions: map[string]proposals.Decision{}, Quote: &proposals.QuoteReview{
		Fees:     []proposals.QuotedFee{{ID: "total", Label: "Total", Amount: amount("1000.00"), Role: proposals.RoleTotal}, {ID: "design", Label: "Design bundle", Amount: amount("800.00"), Role: proposals.RoleComponent}, {ID: "site", Label: "Site services", Amount: amount("200.00"), Role: proposals.RoleComponent}},
		Coverage: map[string]proposals.QuoteCoverage{"stage:concept": {FeeID: "design", Status: "Suggested coverage"}, "stage:documents": {FeeID: "design", Status: "Suggested coverage"}, "stage:site": {FeeID: "site", Status: "Suggested coverage"}},
	}}
	rows, total, _ := appointmentRows(items, col)
	if len(rows) != 2 || total == nil || *total != 100000 || len(rows[0].Keys) != 2 {
		t.Fatalf("bundle duplicated %+v %v", rows, total)
	}
	raw, _ := json.Marshal(AppointmentChoice{Included: false, Keys: []string{"stage:concept", "stage:documents"}})
	col.Decisions[appointmentPrefix+"quote:design"] = proposals.Decision{Value: string(raw), DecidedBy: proposals.ByUser}
	rows, total, _ = appointmentRows(items, col)
	if len(rows) != 2 || rows[0].Included || total == nil || *total != 20000 {
		t.Fatal("bundle did not move together")
	}
	// A mismatched breakdown stays a single package; no invented allocation.
	col.Decisions = map[string]proposals.Decision{}
	*col.Quote.Fees[1].Amount = "700.00"
	rows, total, _ = appointmentRows(items, col)
	if rows[0].ID != "fee" || total == nil || *total != 100000 {
		t.Fatalf("unreconciled source added to total %+v", rows)
	}
	// A known tax basis is required before the original total is offered.
	col.Submission.GSTBasis = "unknown"
	_, total, _ = appointmentRows(items, col)
	if total != nil {
		t.Fatal("unknown tax basis manufactured a fee")
	}
}
