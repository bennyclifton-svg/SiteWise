package jobs_test

import (
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/jobs"
)

func TestProfileSplitHeadingsKeepDescriptionsTablesAndExclusions(t *testing.T) {
	text := "3.0\nDevelopment Outline\n3.01\nDescription\nThe extension comprises a warehouse tenancy with a mezzanine office.\n3.02\nArea Schedule\nTenancy\nWarehouse Area\nOffice Area\nTotal Building Area\nTenancy 01\n1,920 m2\n199 m2\n2,135 m2\nAll areas are GLA.\n10.03 Natural Gas\n\nNot applicable."
	src := jobs.SourceFromText(identity.Text{Format: "pdf", PageCount: 1, Runs: []identity.Run{{Text: text, Source: identity.Source{Page: 1}}}})
	var description, table, exclusion bool
	for _, u := range src.Units {
		if u.Body != src.Source[0].Text[u.Start:u.End] {
			t.Fatal("source offsets changed")
		}
		if strings.Contains(u.Body, "The extension") {
			description = strings.Contains(u.Section, "Description")
			call := jobs.LabelCall(loadKnowledge(t), jobs.Passage{Text: u.Body, Section: u.Section, Context: u.Context, Kind: "design_brief", Ordinal: 30})
			if _, ok := call.Questions["hdr.subclass"]; !ok {
				t.Error("description not routed")
			}
		}
		if strings.Contains(u.Body, "2,135 m2") {
			table = strings.Contains(u.Body, "Total Building Area") && strings.Contains(u.Body, "1,920 m2")
		}
		if strings.Contains(u.Body, "Not applicable") {
			exclusion = strings.Contains(u.Body, "Natural Gas")
			call := jobs.LabelCall(loadKnowledge(t), jobs.Passage{Text: u.Body, Section: u.Section, Kind: "specification", Ordinal: 40})
			if _, ok := call.Questions["det.gas_supply"]; !ok {
				t.Error("gas exclusion not asked")
			}
		}
		if u.Section == "199 m2" {
			t.Error("quantity became heading")
		}
	}
	if !description || !table || !exclusion {
		t.Fatalf("description=%v table=%v exclusion=%v: %+v", description, table, exclusion, src.Units)
	}
}

func TestProfileRoutesLateDescriptionWithoutAHeading(t *testing.T) {
	call := jobs.LabelCall(loadKnowledge(t), jobs.Passage{Text: "The development comprises a warehouse with an associated office.", Kind: "design_brief", Ordinal: 60})
	if _, ok := call.Questions["hdr.building_class"]; !ok {
		t.Fatal("late project description skipped")
	}
}

func TestAncillaryUseAndGeneralPowerDoNotClassifyWholeProject(t *testing.T) {
	for _, p := range []jobs.Passage{
		{Text: "• General business administration activities including clerical work.", Section: "3.03 Proposed Use", Kind: "design_brief", Ordinal: 30},
		{Text: "One socket outlet per 10m2 of office GLA.", Section: "7.04 General Power", Kind: "design_brief", Ordinal: 100},
	} {
		call := jobs.LabelCall(loadKnowledge(t), p)
		for _, key := range []string{"hdr.building_class", "hdr.subclass"} {
			if _, ok := call.Questions[key]; ok {
				t.Errorf("component offered %s: %s", key, p.Text)
			}
		}
	}
}

func TestExplicitServiceExclusionIsAskedEvenWithoutFamilyLabel(t *testing.T) {
	p := jobs.Passage{Text: "Not used.", Section: "NON POTABLE WATER SERVICES", Ordinal: 40}
	call, ok := jobs.EvidenceCall(loadKnowledge(t), p)
	if !ok {
		t.Fatal("explicit service skipped because family label absent")
	}
	if _, ok := call.Questions["sys.hydraulic.non-potable-water.presence"]; !ok {
		t.Fatal("exclusion question absent")
	}
}
