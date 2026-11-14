package sample

import (
	"context"
	"testing"

	"sitewise/internal/tender"
)

// Every figure in this test is invented.

func TestModelListsLinesFromPageText(t *testing.T) {
	rows := []tender.Row{{Key: "stage:concept", Label: "Concept design"}, {Key: "stage:planning", Label: "Planning approval"}}
	text := "Invented Planning Pty Ltd\nConcept sketches\n$3,000.00\nPlanning report $9,000.00\nSundries $500.00\nSurvey (excluded)\n" +
		"Option: extra meeting $1,800.00\nSaving: fewer copies $200.00\nExtra: planning levy $700.00\nTotal (excl. GST)\n$12,500.00"
	pages := []tender.Page{{Number: 1, Text: text}}
	raw, info, err := tender.Pass(context.Background(), Model{}, nil, "claude-haiku-5-5", "file.docx", false, pages, rows)
	if err != nil || info.Stop != "end_turn" {
		t.Fatalf("%v %+v", err, info)
	}
	want := []struct{ label, kind, where, row, st string }{
		{"Concept sketches", "base", "sum", "stage:concept", "priced"},
		{"Planning report", "base", "sum", "stage:planning", "priced"},
		{"Sundries", "base", "sum", "", "priced"},
		{"Survey", "base", "sum", "", "excluded"},
		{"Option: extra meeting", "option", "outside", "", "priced"},
		{"Saving: fewer copies", "saving", "outside", "", "priced"},
		{"Extra: planning levy", "ps", "outside", "stage:planning", "priced"},
		{"Total (excl. GST)", "total", "inside", "", "priced"},
	}
	if len(raw.Lines) != len(want) {
		t.Fatalf("%d lines: %+v", len(raw.Lines), raw.Lines)
	}
	for i, w := range want {
		l := raw.Lines[i]
		if l.Label != w.label || l.Kind != w.kind || l.Where != w.where || l.Row != w.row || l.St != w.st || l.P != 1 {
			t.Fatalf("line %d: %+v", i, l)
		}
	}
	r := tender.Reconcile(raw, pages, rows, nil)
	if r.TenderSum == nil || *r.TenderSum != 1250000 || r.Gap != 0 {
		t.Fatalf("sum %v gap %d", r.TenderSum, r.Gap)
	}
}
