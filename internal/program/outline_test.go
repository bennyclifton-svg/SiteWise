package program

import (
	"strings"
	"testing"
)

// Owner direction 12 November 2026 (docs/plans/2026-11-12-program-outline.md):
// a first program is 20 to 30 rows, an unknown NCC class gives one generic
// set of activities rather than an "If …" copy per class, every stage has
// work in it and the stages follow one another.

func outlineHouse(class string) GenInput {
	in := chenHouse()
	in.Outline = true
	in.Values = map[string]string{"state": "VIC"}
	if class != "" {
		in.Values["ncc_class"] = class
	}
	return in
}

func checkOutline(t *testing.T, p Plan) {
	t.Helper()
	if n := len(p.Rows); n < 20 || n > 30 {
		for _, r := range p.Rows {
			t.Logf("%s | %s", r.ParentKey, r.Label)
		}
		t.Fatalf("outline has %d rows, want 20 to 30", n)
	}
	kids := map[string]int{}
	for _, r := range p.Rows {
		if strings.HasPrefix(r.Label, "If ") {
			t.Fatalf("conditional row in the outline: %s", r.Label)
		}
		if r.Kind == KindHoldPoint || r.WorkItemID != "" {
			t.Fatalf("trade-level row in the outline: %s (%s)", r.Label, r.SourceKey)
		}
		if r.Kind != KindStage {
			kids[r.ParentKey]++
			if r.Duration == nil {
				t.Fatalf("no duration: %s", r.Label)
			}
		}
	}
	out := Compute(p.Input(Calendar{}))
	prevFinish := -1
	for _, r := range p.Rows {
		if r.Kind != KindStage {
			continue
		}
		if kids[r.SourceKey] == 0 {
			t.Fatalf("stage %s is empty", r.Label)
		}
		tm, ok := out.Activities.Get(r.SourceKey)
		if !ok || tm.Unknown || tm.EF <= tm.ES {
			t.Fatalf("stage %s has no span: %+v", r.Label, tm)
		}
		if tm.ES < prevFinish {
			t.Fatalf("stage %s starts at %d before the previous stage finishes at %d", r.Label, tm.ES, prevFinish)
		}
		prevFinish = tm.EF
	}
}

func construction(t *testing.T, p Plan) Row {
	t.Helper()
	for _, r := range p.Rows {
		if r.FragmentID == fragmentConstruction {
			return r
		}
	}
	t.Fatal("no Construction bar")
	return Row{}
}

func TestOutlineUnknownClassIsGeneric(t *testing.T) {
	cat := testCatalog(t, nil, "")
	p := mustGenerate(t, cat, outlineHouse(""))
	checkOutline(t, p)
	c := construction(t, p)
	if c.DurationOrigin != DurationCalculation || *c.Duration < 20 {
		t.Fatalf("Construction not sized from the trade chain: %v %v", c.DurationOrigin, *c.Duration)
	}
	if note, _ := c.DurationSource["assumed"].(string); !strings.Contains(note, "Class 1a") {
		t.Fatalf("assumed class not said: %q", note)
	}
	found := false
	for _, r := range p.Rows {
		if r.Label == "Prepare the concept design" {
			found = r.DurationSource["assumed"] == noteFallback
		}
	}
	if !found {
		t.Fatal("generic concept design missing or not marked assumed")
	}
}

func TestOutlineKnownClassUsesItsOwnLengths(t *testing.T) {
	cat := testCatalog(t, nil, "")
	p := mustGenerate(t, cat, outlineHouse("1a"))
	checkOutline(t, p)
	for _, r := range p.Rows {
		if strings.Contains(r.FragmentID, "-generic") {
			t.Fatalf("Class 1a took a generic fallback: %s", r.FragmentID)
		}
	}
	if c := construction(t, p); c.DurationOrigin != DurationCalculation {
		t.Fatalf("Construction origin %s", c.DurationOrigin)
	}
}

func TestOutlineConstructionDurationFromProfile(t *testing.T) {
	cat := testCatalog(t, nil, "")
	in := outlineHouse("")
	in.Planning = map[string]string{"construction_duration": "40"}
	p := mustGenerate(t, cat, in)
	c := construction(t, p)
	if c.DurationOrigin != DurationProfile || *c.Duration != 200 {
		t.Fatalf("Construction %v %v, want 200 days from the Profile", c.DurationOrigin, *c.Duration)
	}
}

// The detailed program is unchanged by the outline library: no outline-only
// fragment appears in it.
func TestDetailedProgramHasNoOutlineRows(t *testing.T) {
	cat := testCatalog(t, nil, "")
	in := outlineHouse("1a")
	in.Outline = false
	p := mustGenerate(t, cat, in)
	for _, r := range p.Rows {
		if r.FragmentID == fragmentConstruction || strings.Contains(r.FragmentID, "-generic") || r.FragmentID == "fr.for-construction-issue" {
			t.Fatalf("outline-only row in the detailed program: %s", r.FragmentID)
		}
	}
}
