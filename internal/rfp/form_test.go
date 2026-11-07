package rfp

import (
	"bytes"
	"context"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/xlsx"
)

// The response form is read back by the same extractor that reads a
// returned proposal: every price and rate key is a cell in column A.
func TestResponseFormRoundTrips(t *testing.T) {
	items := GenericSchedule([]Row{{Key: "r2", Label: "R2 Deliverables list", Text: "Each deliverable."}, {Key: "r3", Label: "R3 Hourly rates", Text: "Roles: Director · Planner."}})
	items = append(items, ScheduleItem{Key: "option:site_visit", Kind: ItemOption, Label: "Site visit"})
	var buf bytes.Buffer
	if err := xlsx.WriteXLSX(&buf, ResponseForm(FormHeader{Title: "Town planning RFP", Project: "Arthur St"}, items, nil)); err != nil {
		t.Fatal(err)
	}
	limits := identity.DefaultLimits()
	limits.MaxSheets = 12
	limits.MaxRows = 500
	text, err := identity.Extract(context.Background(), "xlsx", bytes.NewReader(buf.Bytes()), int64(buf.Len()), limits)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, run := range text.Runs {
		if run.Source.Col == 1 {
			found[run.Text] = true
		}
	}
	for _, item := range items {
		if (item.Price() || item.Kind == ItemRate) && !found[item.Key] {
			t.Errorf("form has no row for %s", item.Key)
		}
	}
	if !found["base_total"] || !found["options_total"] || !found["Firm"] || !found["ABN"] {
		t.Errorf("form is missing totals or firm fields: %v", found)
	}
}

// A combined engagement's form asks for a sum per discipline and the
// engagement total, every row keyed by its discipline.
func TestGroupResponseFormSumsEachDiscipline(t *testing.T) {
	e := engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil")
	fees := GroupSchedule(e)
	var buf bytes.Buffer
	if err := xlsx.WriteXLSX(&buf, GroupResponseForm(FormHeader{Title: "RFP — Structural and civil engineering", Project: "Arthur St"}, fees, e.Parts[0].Resolved.Returnables)); err != nil {
		t.Fatal(err)
	}
	limits := identity.DefaultLimits()
	limits.MaxSheets = 12
	limits.MaxRows = 500
	text, err := identity.Extract(context.Background(), "xlsx", bytes.NewReader(buf.Bytes()), int64(buf.Len()), limits)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	combined := false
	for _, run := range text.Runs {
		if run.Source.Col == 1 {
			found[run.Text] = true
			combined = combined || run.Text == CombinedStageNote
		}
	}
	for _, f := range fees {
		if !found["sum:"+f.Key] {
			t.Errorf("no sum for %s", f.Key)
		}
		for _, it := range append(f.Items, f.Rates...) {
			if !found[f.Key+":"+it.Key] {
				t.Errorf("form has no row for %s:%s", f.Key, it.Key)
			}
		}
	}
	if !found["engagement_total"] || !combined {
		t.Errorf("form needs the engagement total and the combined-stage note: %v", found)
	}
}
