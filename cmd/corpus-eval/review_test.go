package main

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestReviewQueueDistinguishesUnknownBlankAndWrongGreen(t *testing.T) {
	tr := trace{Entry: entry{SHA256: "hash", Corpus: "test"}, Text: identity.Text{TextLayer: true}, Decisions: []intake.Decision{{Field: "number", Value: "A102", Band: intake.BandGreen}}}
	g := goldCase{Fields: map[string]label{"number": {"A101", "sheet"}, "date": {"2023-11-06", "issue row"}}}
	got := reviewFailures(tr, g, true)
	if len(got) != 2 {
		t.Fatalf("review count: %+v", got)
	}
	reasons := map[string]string{}
	for _, r := range got {
		reasons[r.Field] = r.Reason
	}
	if reasons["number"] != "wrong_green" || reasons["date"] != "blank" {
		t.Fatal(reasons)
	}
	unknown := reviewFailures(tr, goldCase{}, false)
	if len(unknown) != 1 || unknown[0].Reason != "unlabelled" {
		t.Fatal(unknown)
	}
	tr.Error = "no text"
	if got := reviewFailures(tr, goldCase{NotFiled: "no text"}, true); len(got) != 0 {
		t.Fatal(got)
	}
}
