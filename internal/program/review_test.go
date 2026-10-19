package program

import (
	"strings"
	"testing"
)

// liveFrom stores a plan as the store would: ids "id:<source key>", every
// row proposed and from the library.
func liveFrom(p Plan) ([]LiveRow, []LiveLink) {
	var rows []LiveRow
	for _, r := range p.Rows {
		parent := ""
		if r.ParentKey != "" {
			parent = "id:" + r.ParentKey
		}
		rows = append(rows, LiveRow{ID: "id:" + r.SourceKey, SourceKey: r.SourceKey, ParentID: parent, Kind: r.Kind,
			Label: r.Label, CatalogueStageID: r.CatalogueStageID, PackageID: r.PackageID, Duration: r.Duration,
			Low: r.Low, High: r.High, DurationOrigin: r.DurationOrigin, Origin: OriginLibrary, ReviewStatus: ReviewProposed})
	}
	var links []LiveLink
	for _, l := range p.Links {
		links = append(links, LiveLink{ID: "id:" + l.SourceKey, PredID: "id:" + l.PredKey, SuccID: "id:" + l.SuccKey,
			Type: l.Type, Lag: l.Lag, LagUnit: l.LagUnit, Origin: l.Origin, SourceKey: l.SourceKey})
	}
	return rows, links
}

func findChange(cs []Change, kind, key string) (Change, bool) {
	for _, c := range cs {
		if c.Kind == kind && c.SourceKey == key {
			return c, true
		}
	}
	return Change{}, false
}

func reviewInput() GenInput {
	return GenInput{WorkTypes: []string{"new"}, Values: map[string]string{"flood_hazard": "true"},
		WorkItems: []WorkItem{
			{ID: "slab", System: "substructure", Action: "new", Title: "Slab"},
			{ID: "frame", System: "structure", Action: "new", Title: "Frame"}},
		Packages: []Package{
			{ID: "arch", Title: "Architect", Kind: "services", Stages: []PackageStage{{ID: "a1", CatalogueStageID: "construction", Label: "Construction"}}},
			{ID: "eng", Title: "Engineer", Kind: "services", Stages: []PackageStage{{ID: "e1", CatalogueStageID: "construction", Label: "Construction"}}}}}
}

func TestDiffNoChange(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	p := mustGenerate(t, cat, reviewInput())
	rows, links := liveFrom(p)
	if cs := Diff(rows, links, p); len(cs) != 0 {
		t.Fatalf("unchanged program has changes: %+v", cs)
	}
}

func TestDiffDefaultsAndSummaries(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	before := mustGenerate(t, cat, reviewInput())
	rows, links := liveFrom(before)
	// The user edited the frame row and accepted the engineer's row.
	for i := range rows {
		switch rows[i].SourceKey {
		case "fragment:fr.frame:frame:frame":
			rows[i].Label, rows[i].Origin, rows[i].ReviewStatus = "Frame (steel)", OriginUser, ReviewAccepted
		case "package:eng:e1":
			rows[i].ReviewStatus = ReviewAccepted
		}
	}
	// A user-made row with no source key is never proposed for removal.
	rows = append(rows, LiveRow{ID: "mine", ParentID: "id:skeleton:sk.new:construction", Kind: KindActivity,
		Label: "My row", Origin: OriginUser, ReviewStatus: ReviewAccepted})

	in := reviewInput()
	in.WorkItems = in.WorkItems[:1]           // frame removed from scope
	in.Packages = []Package{in.Packages[0]}   // engineer removed in Procurement
	in.Packages[0].Title = "Architect (lead)" // renamed
	in.Packages = append(in.Packages, Package{ID: "qs", Title: "Quantity surveyor", Kind: "services"})
	after := mustGenerate(t, cat, in)
	cs := Diff(rows, links, after)

	frame, ok := findChange(cs, ChangeRemove, "fragment:fr.frame:frame:frame")
	if !ok || frame.DefaultChoice != ChoiceKeep || !frame.Edited || frame.Source != SourceWorkItems ||
		!strings.Contains(frame.Summary, "its work item is no longer in scope; you edited “Frame (steel)”") {
		t.Fatalf("edited removal %+v", frame)
	}
	eng, ok := findChange(cs, ChangeRemove, "package:eng:e1")
	if !ok || eng.DefaultChoice != ChoiceKeep || eng.PackageID != "eng" || eng.Source != SourceProcurement {
		t.Fatalf("package removal %+v", eng)
	}
	if got := RemovedPackageRows(cs); len(got) != 1 || got[0] != "id:package:eng:e1" {
		t.Fatalf("removed package rows %v", got)
	}
	ren, ok := findChange(cs, ChangeRename, "package:arch:a1")
	if !ok || ren.DefaultChoice != ChoiceAdopt || ren.Ops[0]["op"] != "update" || ren.Ops[0]["label"] != "Architect (lead): Construction" {
		t.Fatalf("rename %+v", ren)
	}
	add, ok := findChange(cs, ChangeAdd, "package:qs:")
	if !ok || add.DefaultChoice != ChoiceAdopt || add.Ops[0]["op"] != "add" || add.Ops[0]["parent_id"] != "id:skeleton:sk.new:construction" ||
		add.Ops[0]["source_key"] != "package:qs:" || add.Ops[0]["origin"] != OriginLibrary {
		t.Fatalf("add %+v", add)
	}
	// The new row's chain link travels with its add.
	linked := false
	for _, op := range add.Ops[1:] {
		if op["op"] == "link_add" && op["succ_id"] == "package:qs:" {
			linked = true
		}
	}
	if !linked {
		t.Fatalf("add has no link op: %+v", add.Ops)
	}
	for _, c := range cs {
		if c.TargetActivityID == "mine" {
			t.Fatal("user row proposed for removal")
		}
		if c.Kind == ChangeLink && c.Ops[0]["op"] == "link_delete" {
			for _, id := range c.Ops[0]["ids"].([]string) {
				if strings.Contains(id, "fr.frame") || strings.Contains(id, "package:eng") {
					t.Fatalf("link of a removed row listed separately: %s", id)
				}
			}
		}
	}
}

func TestDiffMatchesBySourceKeyOnly(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	p := mustGenerate(t, cat, reviewInput())
	rows, links := liveFrom(p)
	// Same label, different key: a removal and an addition, never a rename.
	for i := range rows {
		if rows[i].SourceKey == "package:arch:a1" {
			rows[i].SourceKey = "package:arch:old"
		}
	}
	cs := Diff(rows, links, p)
	if _, ok := findChange(cs, ChangeRemove, "package:arch:old"); !ok {
		t.Fatalf("no removal: %+v", cs)
	}
	if _, ok := findChange(cs, ChangeAdd, "package:arch:a1"); !ok {
		t.Fatalf("no addition: %+v", cs)
	}
	for _, c := range cs {
		if c.Kind == ChangeRename {
			t.Fatalf("label match made a rename: %+v", c)
		}
	}
}

func TestDiffDuration(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	in := reviewInput()
	p := mustGenerate(t, cat, in)
	rows, links := liveFrom(p)
	key := "fragment:fr.slab:slab:form"
	in.DocumentDurations = map[string]Reading{key: {Days: 9, ReadingID: "r"}}
	cs := Diff(rows, links, mustGenerate(t, cat, in))
	c, ok := findChange(cs, ChangeDuration, key)
	if !ok || c.DefaultChoice != ChoiceAdopt || c.Ops[0]["duration_days"] == nil || *(c.Ops[0]["duration_days"].(*int)) != 9 ||
		c.Summary != "Form slab: duration 5 days to 9 days." {
		t.Fatalf("duration change %+v", c)
	}
}
