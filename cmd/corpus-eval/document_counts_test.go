package main

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestAbsentDateDoesNotCountAsFullyPopulated(t *testing.T) {
	g := goldCase{Fields: map[string]label{}}
	tr := trace{Text: identity.Text{TextLayer: true}}
	for _, f := range []string{"number", "revision", "title", "date", "kind", "discipline"} {
		v := "value"
		if f == "date" {
			v = ""
		}
		g.Fields[f] = label{Value: v, Evidence: "independently checked"}
		tr.Decisions = append(tr.Decisions, intake.Decision{Field: f, Value: v})
	}
	s := summary{Metrics: map[string]counts{}}
	score(&s, tr, g)
	if s.FullyLabelledDocuments != 1 || s.FullyCorrectDocuments != 1 || s.FullyPopulatedCorrectDocuments != 0 {
		t.Fatalf("document counts: %+v", s)
	}
	if s.Metrics["date"].CorrectAbsent != 1 {
		t.Fatal("missing correct abstention")
	}
	delete(g.Fields, "title")
	s = summary{Metrics: map[string]counts{}}
	score(&s, tr, g)
	if s.FullyLabelledDocuments != 0 || s.FullyCorrectDocuments != 0 {
		t.Fatal("partially labelled document counted complete")
	}
}
