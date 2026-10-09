package reports

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

// An RFP is read by a consultant pricing services. Review prompts for the
// project manager belong in an internal pre-issue list, not the issued body.
func TestRFPBodyKeepsReviewItemsInternal(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	template, _ := cat.ReportTemplate("tpl.rfp-capex", 1)
	pkg := procurement.Package{ID: "arch", Kind: "services", Title: "Architect", Origin: "user", LifecycleStatus: "planned", Stages: []procurement.Stage{{ID: "concept", Label: "Concept design", NovationPhase: "pre"}}}
	s := Snapshot{ProjectName: "Petersham", ProjectBasis: json.RawMessage(`{}`), Package: pkg, ProfileStale: []string{"doc"}}
	s.Brief = []BriefValue{
		{ID: "a", Label: "Whole project — cond.access constraints", Text: "urban_constrained", Basis: json.RawMessage(`{}`)},
		{ID: "b", Label: "Whole project — Sewage disposal", Unknown: true, Basis: json.RawMessage(`{}`)},
		{ID: "c", Label: "Whole project — cond.planning", Text: "da", Basis: json.RawMessage(`{}`)},
		{ID: "d", Label: "Whole project — scale.storeys", Text: "5", Basis: json.RawMessage(`{}`)},
		{ID: "e", Label: "Whole project — scale.total storeys", Text: "5", Basis: json.RawMessage(`{}`)},
		{ID: "f", Label: "Whole project — Building type", Text: "apartments", Basis: json.RawMessage(`{}`)},
	}
	for _, id := range []string{"w1", "w2", "w3"} {
		s.Works = append(s.Works, works.Item{ID: id, Title: "Element " + id, Action: "new", Inclusion: "included", ReviewStatus: "proposed"})
		s.Proposals = append(s.Proposals, works.Proposal{Key: "cap|" + id, State: "open", Kind: "investigation", Label: "Capacity of the existing supply"})
		s.Gaps = append(s.Gaps, procurement.Gap{WorkItemID: id, Role: "design", State: "not_yet_accepted"})
	}
	sections, err := Assemble(s, template, cat, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	var body, review strings.Builder
	var reviewSection *Section
	for i, section := range sections {
		var raw []byte
		for _, b := range section.Blocks {
			raw = append(raw, b.Label+": "+b.Text+"\n"...)
		}
		if section.Internal {
			review.Write(raw)
			reviewSection = &sections[i]
		} else {
			body.Write(raw)
		}
	}
	if reviewSection == nil {
		t.Fatal("no internal pre-issue review section")
	}
	for _, leak := range []string{"Unaccepted proposal", "Allocation", "Not established", "Reading status", "cond.", "urban_constrained", "Element w1. Action"} {
		if strings.Contains(body.String(), leak) {
			t.Fatalf("body still carries %q: %s", leak, body.String())
		}
	}
	for _, want := range []string{"Access constraints", "Urban constrained", "Development application", "Element w1, Element w2, Element w3"} {
		if !strings.Contains(body.String(), want) {
			t.Fatalf("body lost %q: %s", want, body.String())
		}
	}
	if strings.Contains(body.String(), "Total storeys") || !strings.Contains(body.String(), "Building type: Apartments") {
		t.Fatalf("repeated total or lower-case value: %s", body.String())
	}
	if n := strings.Count(review.String(), "Capacity of the existing supply"); n != 1 {
		t.Fatalf("repeated proposal listed %d times, want once with a count", n)
	}
	for _, want := range []string{"3 work items", "Sewage disposal", "No services or deliverables", "needs updating"} {
		if !strings.Contains(review.String(), want) {
			t.Fatalf("review lost %q: %s", want, review.String())
		}
	}
	issued, _, err := IssueContent(sections, "rfp", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range issued {
		if section.Internal {
			t.Fatal("internal review section issued")
		}
	}
}
