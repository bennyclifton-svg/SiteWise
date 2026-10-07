package reports

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

func TestRFPAssemblySavedScopeAndDraftBoundaries(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	template, ok := cat.ReportTemplate("tpl.rfp-capex", 1)
	if !ok {
		t.Fatal("template missing")
	}
	pkg := procurement.Package{ID: "services", Kind: "services", Title: "Fire engineering", Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated", LifecycleStatus: "planned", Stages: []procurement.Stage{{ID: "stage", Label: "Design", Origin: "user", NovationPhase: "none"}}}
	s := Snapshot{ProjectID: "project", ProjectName: "Upgrade", ProjectBasis: json.RawMessage(`{"actor":"owner"}`), Package: pkg, PartLabels: map[string]string{"warehouse": "Warehouse A"}, Brief: []BriefValue{{ID: "floor_area", Label: "Floor area", Unknown: true, Origin: "document", Basis: json.RawMessage(`{}`)}}}
	s.Works = []works.Item{{ID: "replace", PartID: "warehouse", Title: "Replace sprinklers", Action: "replace", Inclusion: "included", Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated"}, {ID: "survey", Title: "Survey fire water", Action: "investigate", Inclusion: "included", ReviewStatus: "proposed", Origin: "calculation", Meaning: "stated"}, {ID: "excluded", Title: "Excluded roof works", Action: "repair", Inclusion: "excluded", Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated"}}
	s.Scope = []ScopeRecord{{ID: "scope", PackageID: pkg.ID, ScopeContent: procurement.ScopeContent{ItemKind: "responsibility", WorkItemID: "replace", Role: "design", Inclusion: "included", UserText: "Design replacement sprinklers", Deliverable: "Design drawings"}, Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated"}, {ID: "foreign", PackageID: "other-package", ScopeContent: procurement.ScopeContent{UserText: "Other package secret"}}}
	s.Delivery = []delivery.Item{{ID: "risk", Content: delivery.Content{Kind: "risk", Title: "Existing water supply", Status: "open", Details: json.RawMessage(`{"likelihood":"Unknown","consequence":"Capacity may be insufficient"}`)}, Origin: "user", ReviewStatus: "accepted_for_planning", Meaning: "stated"}}
	s.Proposals = []works.Proposal{{Key: "ic.open||||0", State: "open", Kind: "obligation", Label: "Coordinate shutdown", Draft: true}, {Key: "ic.dismissed||||0", State: "dismissed", Label: "Dismissed proposal secret"}}
	sections, err := AssembleRFP(s, template, cat, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) != 7 {
		t.Fatal("missing required sections", sections)
	}
	blocks := map[string]Block{}
	for _, section := range sections {
		if !section.Essential {
			t.Fatal("essential flag lost")
		}
		for _, b := range section.Blocks {
			blocks[b.ID] = b
			if b.ContentSHA256 == "" {
				t.Fatal("unhashed target", b)
			}
		}
	}
	if blocks["brief:floor_area"].Text != "Not established" || !blocks["work:survey"].Provisional || !blocks["proposal:ic.open||||0"].Provisional {
		t.Fatal("unknown/proposed state lost", blocks)
	}
	if !strings.Contains(blocks["scope:scope"].Text, "Design drawings") || !strings.Contains(blocks["work:replace"].Text, "Warehouse A") || !strings.Contains(blocks["work:excluded"].Label, "excluded") {
		t.Fatal("scope detail lost", blocks)
	}
	if !blocks["clause:brief:cl.rfp-draft-status"].Provisional {
		t.Fatal("draft clause became approved")
	}
	raw, _ := json.Marshal(sections)
	for _, secret := range []string{"Other package secret", "Dismissed proposal secret"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("unrelated or dismissed content leaked")
		}
	}
	edit := Edit{TargetID: "scope:scope", Text: "Protected engineering scope", BaseContentSHA256: blocks["scope:scope"].ContentSHA256, UserID: "owner", Version: 1}
	s.Scope[0].Deliverable = "Updated drawings"
	sections, err = AssembleRFP(s, template, cat, []Edit{edit}, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, section := range sections {
		for _, b := range section.Blocks {
			if b.ID == edit.TargetID {
				found = true
				if !b.Conflict || b.Text != edit.Text {
					t.Fatal("edit overwritten", b)
				}
			}
		}
	}
	if !found {
		t.Fatal("edit omitted")
	}
	s.PendingDocuments = 1
	if _, err := AssembleRFP(s, template, cat, nil, false); !errors.Is(err, ErrReadingIncomplete) {
		t.Fatalf("pending silently assembled: %v", err)
	}
	if _, err := AssembleRFP(s, template, cat, nil, true); err != nil {
		t.Fatal("explicit last completed refused", err)
	}
	s.PendingDocuments = 0
	s.FailedDocuments = 1
	if _, err := AssembleRFP(s, template, cat, nil, false); !errors.Is(err, ErrReadingIncomplete) {
		t.Fatal("failed reading hidden", err)
	}
	now := time.Now()
	s.Package.RetiredAt = &now
	if _, err := AssembleRFP(s, template, cat, nil, true); err == nil {
		t.Fatal("retired package assembled")
	}
}

func TestContentHashSurvivesJSONBPersistence(t *testing.T) {
	one := Block{ID: "one", Text: "text", Basis: json.RawMessage(`{"version":1234567890123456789,"actor":"owner"}`)}
	two := one
	two.Basis = json.RawMessage(`{ "actor": "owner", "version": 1234567890123456789 }`)
	a, err := ContentHash(one)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ContentHash(two)
	if err != nil || a != b {
		t.Fatalf("JSON order changed hash %s %s %v", a, b, err)
	}
}
