package profile_test

import (
	"reflect"
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/works"
)

func TestProposalInputsInheritanceAndEligibility(t *testing.T) {
	cat := repoCatalog(t)
	parts := input().Parts
	rows := []profile.Row{
		{PartID: whole, Key: "hdr.work_type", Value: "new", Band: "user", Origin: "user"},
		{PartID: "p-b", Key: "hdr.work_type", Value: "refurb", Band: "user", Origin: "user"},
		{PartID: whole, Key: "det.existing_building_year", Value: "2000", Band: "amber", Origin: "document", Meaning: "stated"},
		{PartID: whole, Key: "det.existing_building", Value: "true", Band: "user", Origin: "user"},
		{PartID: "p-b", Key: "det.existing_building_year", Band: "user", Origin: "user", ValueState: profile.StateUnknown},
		{PartID: whole, Key: "det.bal", Value: "BAL-LOW", Band: "planning", Origin: "assumption"},
	}
	in, err := profile.ProposalInputs(rows, parts, nil, cat)
	if err != nil {
		t.Fatal(err)
	}
	p := in.Parts["p-b"]
	if p.Values["existing_building_year"].Value != "" || p.Values["existing_building_year"].Origin != "user" {
		t.Fatal("unknown lost", p.Values)
	}
	if p.Values["existing_building"].Value != "true" {
		t.Fatal("inheritance lost", p.Values)
	}
	if p.Values["bal"].Value != "" || p.Values["bal"].Origin != "assumption" {
		t.Fatal("assumption applied", p.Values)
	}
	if !reflect.DeepEqual(p.WorkTypes, []string{"new", "refurb"}) {
		t.Fatal(p.WorkTypes)
	}
	for _, meaning := range []string{"requirement", "allowance", "forecast"} {
		rows[3].Meaning = meaning
		in, err = profile.ProposalInputs(rows, parts, nil, cat)
		if err != nil {
			t.Fatal(err)
		}
		if in.Parts["p-b"].Values["existing_building"].Value != "" {
			t.Fatalf("%s applied", meaning)
		}
	}
}

func TestProposalPresenceSeparatesExistingAndCompleted(t *testing.T) {
	cat := repoCatalog(t)
	parts := input().Parts
	rows := []profile.Row{{PartID: whole, Key: "sys.fire-active.hydrants.presence", Value: "not_included", Band: "user", Origin: "user"}, {PartID: whole, Key: "sys.fire-active.fire-water.existing", Value: "present", Band: "user", Origin: "user"}}
	items := []works.Item{{ID: "hydrant", PartID: "p-b", SystemID: "fire-active.hydrants", Action: "new", Inclusion: "included"}}
	in, err := profile.ProposalInputs(rows, parts, items, cat)
	if err != nil {
		t.Fatal(err)
	}
	if in.Existing("p-b", "fire-active.hydrants") != knowledge.Unknown {
		t.Fatal("new work proved existing hydrants")
	}
	if in.Parts["p-b"].PresentState("fire-active.hydrants") != knowledge.True {
		t.Fatal("new work did not supersede absent presence")
	}
	if in.Existing("p-b", "fire-active.fire-water") != knowledge.True {
		t.Fatal("existing site evidence lost")
	}
	if in.Parts["p-b"].PresentState("electrical") != knowledge.Unknown {
		t.Fatal("missing presence became absent")
	}
	rows = append(rows, profile.Row{PartID: "p-b", Key: "sys.fire-active.fire-water.existing", Band: "user", ValueState: profile.StateUnknown})
	in, err = profile.ProposalInputs(rows, parts, items, cat)
	if err != nil {
		t.Fatal(err)
	}
	if in.Existing("p-b", "fire-active.fire-water") != knowledge.Unknown {
		t.Fatal("part unknown failed to shadow site evidence")
	}
	items[0].Action = "remove"
	in, err = profile.ProposalInputs(rows, parts, items, cat)
	if err != nil {
		t.Fatal(err)
	}
	if in.Parts["p-b"].PresentState("fire-active.hydrants") != knowledge.False {
		t.Fatal("removed system still present")
	}
}

func TestProposalInputsRejectUnknownPart(t *testing.T) {
	cat := repoCatalog(t)
	if _, err := profile.ProposalInputs(nil, input().Parts, []works.Item{{PartID: "wrong"}}, cat); err == nil {
		t.Fatal("foreign part accepted")
	}
}
