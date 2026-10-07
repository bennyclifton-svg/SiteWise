package reports

import (
	"encoding/json"
	"sitewise/internal/delivery"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
	"strings"
	"testing"
)

func TestRFTPMPSharedRecordsAndExplicitProgress(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	date := "2026-10-07"
	s := Snapshot{ProjectName: "Shared project", ProjectBasis: json.RawMessage(`{"actor":"owner"}`), PartLabels: map[string]string{"roof": "Roof"}, Package: procurement.Package{ID: "works", Kind: "works", Title: "Roof package", Origin: "user"},
		Works:      []works.Item{{ID: "retain", PartID: "roof", Action: "retain", Title: "Existing roof drains", Inclusion: "included", Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated"}, {ID: "other", Action: "new", Title: "Unrelated package works", Inclusion: "included", Origin: "user"}},
		Scope:      []ScopeRecord{{ID: "responsibility", PackageID: "works", ScopeContent: procurement.ScopeContent{ItemKind: "responsibility", WorkItemID: "retain", Role: "protect", Inclusion: "included", UserText: "Protect drainage during installation"}, Origin: "user"}},
		Delivery:   []delivery.Item{{ID: "approval", Content: delivery.Content{Kind: "approval", Title: "Authority assessment", Status: "submitted", AsOf: &date, Details: json.RawMessage(`{"authority":"Council","submitted_on":"2026-10-07"}`)}, Origin: "user"}, {ID: "progress", Content: delivery.Content{Kind: "activity", Title: "Roof investigation", Status: "in_progress", AsOf: &date}, Origin: "user"}},
		Documents:  []BriefValue{{ID: "drawing", Label: "Document revision", Text: "Roof drawing; revision C", Basis: json.RawMessage(`{"sources":[{"filename":"roof.pdf","revision":"C","file_sha256":"abc"}]}`)}},
		Commercial: []BriefValue{{ID: "line:stable-cost", Label: "Roof allowance", Text: "Pricing ID stable-cost; price to be returned.", Origin: "calculation", Basis: json.RawMessage(`{"budget":{"amount":"2500.00"},"stage_id":"stage-1"}`)}}}
	s.Packages = []procurement.Package{s.Package}
	for _, kind := range []string{"rft", "pmp"} {
		template, ok := cat.ReportTemplate("tpl."+kind, 1)
		if !ok {
			t.Fatal("missing template", kind)
		}
		current := s
		if kind == "pmp" {
			current.Package = procurement.Package{}
		}
		sections, err := Assemble(current, template, cat, nil, false)
		if err != nil {
			t.Fatal(kind, err)
		}
		sections, refs, err := IssueContent(sections, kind, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(sections) != 7 || len(refs) == 0 {
			t.Fatal("incomplete shared report")
		}
		byID := map[string]Block{}
		for _, section := range sections {
			for _, b := range section.Blocks {
				byID[b.ID] = b
			}
		}
		if !strings.Contains(byID["work:retain"].Text, "Retain:") || !strings.Contains(byID["scope:responsibility"].Text, "protect") {
			t.Fatal("retain duty missing", kind)
		}
		if !strings.Contains(byID["delivery:approval"].Text, "Status: submitted") || strings.Contains(byID["delivery:approval"].Text, "Status: approved") {
			t.Fatal("submission promoted to approval")
		}
		if !strings.Contains(byID["delivery:progress"].Text, "in progress") || !strings.Contains(byID["delivery:progress"].Text, date) {
			t.Fatal("explicit progress/as-of missing")
		}
		if !strings.Contains(byID["document:drawing"].Text, "revision C") {
			t.Fatal("exact revision missing")
		}
		if kind == "rft" {
			if _, ok := byID["work:other"]; ok {
				t.Fatal("unrelated work included")
			}
			raw, _ := json.Marshal(sections)
			if strings.Contains(string(raw), "2500.00") {
				t.Fatal("budget disclosed by default")
			}
		} else if !strings.Contains(byID["cost:line:stable-cost"].Text, "2500.00") {
			t.Fatal("PMP budget unavailable")
		}
	}
	s.Commercial = nil
	s.Package = procurement.Package{}
	template, _ := cat.ReportTemplate("tpl.pmp", 1)
	sections, err := Assemble(s, template, cat, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, section := range sections {
		for _, b := range section.Blocks {
			if b.ID == "cost:unavailable" {
				found = strings.Contains(b.Text, "not available")
			}
		}
	}
	if !found {
		t.Fatal("missing cost turned into zero")
	}
}
