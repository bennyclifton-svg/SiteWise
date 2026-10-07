package reports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/works"
)

func TestUnforeseenReportScopeAndTreatment(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	const record = "uc.heritage-wall-opening-conflicts-with-fabric"
	s := Snapshot{ProjectName: "Test", ProjectBasis: json.RawMessage(`{"actor":"owner"}`), Package: procurement.Package{ID: "a", Kind: "works", Title: "Walls", Origin: "user"},
		Works:     []works.Item{{ID: "group", IsGroup: true, Inclusion: "included"}, {ID: "wall", ParentID: "group", PartID: "one", SystemID: "envelope.external-walls", Action: "alter", Inclusion: "included", Origin: "user"}, {ID: "other", PartID: "two", SystemID: "envelope.external-walls", Action: "alter", Inclusion: "included", Origin: "user"}},
		Scope:     []ScopeRecord{{ID: "scope", PackageID: "a", Origin: "user", ScopeContent: procurement.ScopeContent{WorkItemID: "group", Inclusion: "included", ItemKind: "responsibility", Role: "install", UserText: "Install"}}},
		Proposals: []works.Proposal{{Key: "ours", RecordKind: "uc", RecordID: record, Label: "Inspect opening", State: "accepted", Reason: works.ProposalReason{Triggers: []works.ProposalTrigger{{WorkItemID: "wall"}}}}, {Key: "theirs", RecordKind: "uc", RecordID: record, Label: "Unrelated risk", State: "proposed", Reason: works.ProposalReason{Triggers: []works.ProposalTrigger{{WorkItemID: "other"}}}}, {Key: "dismissed", RecordKind: "uc", RecordID: record, State: "dismissed"}}}
	for _, kind := range []string{"rft", "pmp"} {
		current := s
		if kind == "pmp" {
			current.Package = procurement.Package{}
		}
		template, _ := cat.ReportTemplate("tpl."+kind, 1)
		sections, err := Assemble(current, template, cat, nil, false)
		if err != nil {
			t.Fatal(err)
		}
		blocks := map[string]Block{}
		for _, section := range sections {
			for _, b := range section.Blocks {
				blocks[b.ID] = b
			}
		}
		b, ok := blocks["latent:ours"]
		if !ok || !b.Provisional {
			t.Fatalf("%s omitted accepted but unresolved latent review", kind)
		}
		for _, text := range []string{"Potential effects: cost, programme, compliance", "Heritage fabric assessment", "Provisional sum", "not allocated or priced", "accepted"} {
			if !strings.Contains(b.Text, text) {
				t.Fatalf("%s missing %q in %s", kind, text, b.Text)
			}
		}
		if !strings.Contains(string(b.Basis), `"contract"`) || !strings.Contains(string(b.Basis), `"de_risk"`) {
			t.Fatal("treatment not frozen")
		}
		_, other := blocks["latent:theirs"]
		if other != (kind == "pmp") {
			t.Fatalf("%s package scoping incorrect", kind)
		}
		if _, ok := blocks["latent:dismissed"]; ok {
			t.Fatal("dismissal ignored")
		}
		if _, ok := blocks["proposal:theirs"]; ok {
			t.Fatal("duplicate or unrelated proposal")
		}
	}
	if !proposalInPackage(works.Proposal{TargetPartID: "one"}, s, cat) || proposalInPackage(works.Proposal{TargetPartID: "two"}, s, cat) {
		t.Fatal("part-specific review escaped package")
	}
}

func TestChangesSinceIssueIncludesSavedTable(t *testing.T) {
	before := []Section{{Blocks: []Block{{ID: "activity", Label: "Dates", Text: "Owner wording", Table: &Table{Columns: []string{"Target", "Current"}, Rows: [][]string{{"2026-10-01", "2026-10-02"}}}}}}}
	after := []Section{{Blocks: []Block{{ID: "activity", Label: "Dates", Text: "Owner wording", Table: &Table{Columns: []string{"Target", "Current"}, Rows: [][]string{{"2026-10-01", "2026-10-03"}}}}}}}
	changes := ChangesSinceIssue(before, after)
	if len(changes) != 1 || changes[0].Kind != "changed" || !strings.Contains(changes[0].After, "2026-10-03") {
		t.Fatalf("table change hidden: %#v", changes)
	}
}

func TestRFTProposalMembershipUsesHierarchyExplicitInterfacesAndLiveRoles(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	group := works.Item{ID: "group", IsGroup: true, Inclusion: "included"}
	child := works.Item{ID: "child", ParentID: "group", PartID: "one", SystemID: "hydraulic.cold-water", Inclusion: "included"}
	scope := func(id, work, pkg, role string) ScopeRecord {
		return ScopeRecord{ID: id, PackageID: pkg, ScopeContent: procurement.ScopeContent{WorkItemID: work, Role: role, ItemKind: "responsibility", Inclusion: "included"}}
	}
	a, b := procurement.Package{ID: "A"}, procurement.Package{ID: "B"}
	s := Snapshot{Package: a, Packages: []procurement.Package{a, b}, Works: []works.Item{group, child}, Scope: []ScopeRecord{scope("parent", "group", "A", "install"), scope("child", "child", "B", "install")}}
	p := works.Proposal{TargetSystemID: "hydraulic", TargetPartID: "one", Reason: works.ProposalReason{Triggers: []works.ProposalTrigger{{WorkItemID: "outside"}}}}
	if proposalInPackage(p, s, cat) {
		t.Fatal("parent role included explicitly overridden child")
	}
	s.Package = b
	if !proposalInPackage(p, s, cat) {
		t.Fatal("ancestor target did not reach child-system package")
	}
	p.TargetPartID = "two"
	if proposalInPackage(p, s, cat) {
		t.Fatal("target system escaped part")
	}
	p.TargetPartID = "one"
	s.Package = a
	s.Scope = append(s.Scope, scope("protect", "group", "A", "protect"))
	if !proposalInPackage(p, s, cat) {
		t.Fatal("child install override removed inherited protect role")
	}
	s.Scope = s.Scope[:1]
	s.Works[0].Inclusion = "excluded"
	if proposalInPackage(p, s, cat) {
		t.Fatal("excluded group inherited")
	}
	s.Works[0].Inclusion = "included"
	now := time.Now()
	s.Works[0].RetiredAt = &now
	if proposalInPackage(p, s, cat) {
		t.Fatal("retired group inherited")
	}
	s.Works[0].RetiredAt = nil
	s.Scope = append(s.Scope, scope("child", "child", "B", "install"))
	s.Packages[1].RetiredAt = &now
	if !proposalInPackage(p, s, cat) {
		t.Fatal("retired package assignment shadowed live parent")
	}
	// An explicit non-role association stays valid independently of role inheritance.
	s.Packages[1].RetiredAt = nil
	s.Scope = append(s.Scope, scope("note", "child", "A", ""))
	s.Scope[2].ItemKind = "obligation"
	if !proposalInPackage(p, s, cat) {
		t.Fatal("non-role scope association lost")
	}
	s.Scope = []ScopeRecord{{ID: "interface", PackageID: "A", ScopeContent: procurement.ScopeContent{Inclusion: "included", ItemKind: "obligation", InterfaceIDs: []string{"if.explicit"}}}}
	p = works.Proposal{InterfaceID: "if.explicit", TargetSystemID: "electrical", TargetPartID: "different"}
	if !proposalInPackage(p, s, cat) {
		t.Fatal("explicit interface allocation lost")
	}
	s.Scope[0].Inclusion = "excluded"
	if proposalInPackage(p, s, cat) {
		t.Fatal("excluded interface included")
	}
	s.Scope[0].Inclusion = "included"
	s.Scope[0].RetiredAt = &now
	if proposalInPackage(p, s, cat) {
		t.Fatal("retired interface scope included")
	}
	s.Scope[0].RetiredAt = nil
	s.Packages[0].RetiredAt = &now
	if proposalInPackage(p, s, cat) {
		t.Fatal("retired package included")
	}
}
