package program

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"testing"

	"sitewise/internal/xlsx"
)

func dur(n int) *int { return &n }

// exportInput is two stages listed out of grid order, with a milestone, an
// unknown duration and one link of each lag shape the typed format has.
func exportInput(start string) Input {
	return Input{
		Calendar: Calendar{Start: start},
		Activities: []Activity{
			{ID: "b2", ParentID: "s2", Ordinal: 1, Label: "Frame", Duration: dur(2)},
			{ID: "s2", Ordinal: 1, Kind: KindStage, Label: "Build"},
			{ID: "a2", ParentID: "s1", Ordinal: 1, Label: "Detail", Duration: dur(3)},
			{ID: "s1", Ordinal: 0, Kind: KindStage, Label: "Design"},
			{ID: "m", ParentID: "s1", Ordinal: 2, Kind: KindMilestone, Label: "=Design signed off", Duration: dur(0)},
			{ID: "a1", ParentID: "s1", Ordinal: 0, Label: "Concept", Duration: dur(5)},
			{ID: "b1", ParentID: "s2", Ordinal: 0, Kind: KindHoldPoint, Label: "Footing inspection"},
		},
		Links: []Link{
			{ID: "l1", Pred: "a1", Succ: "a2", Type: LinkFS},
			{ID: "l2", Pred: "a1", Succ: "m", Type: LinkSS, Lag: 2, LagUnit: LagWorking},
			{ID: "l3", Pred: "a2", Succ: "b2", Type: LinkFS, Lag: 3, LagUnit: LagCalendar},
			{ID: "l4", Pred: "m", Succ: "b2", Type: LinkFF, Lag: -1},
			{ID: "l5", Pred: "s1", Succ: "s2"},
		},
	}
}

var exportOrigins = map[string]string{"s1": "library", "s2": "library", "a1": "user", "a2": "library", "m": "document", "b1": "library", "b2": "user"}

type linkKey struct {
	pred, succ, typ string
	lag             int
	unit            string
}

func wantLinks(in Input, rows map[string]bool) []linkKey {
	var out []linkKey
	for _, l := range in.Links {
		if !rows[l.Pred] || !rows[l.Succ] {
			continue
		}
		typ, unit := l.Type, l.LagUnit
		if typ == "" {
			typ = LinkFS
		}
		if unit == "" {
			unit = LagWorking
		}
		out = append(out, linkKey{l.Pred, l.Succ, typ, l.Lag, unit})
	}
	return sortLinks(out)
}

func sortLinks(s []linkKey) []linkKey {
	sort.Slice(s, func(i, j int) bool { return fmt.Sprint(s[i]) < fmt.Sprint(s[j]) })
	return s
}

// reimport reads a CSV export back the way the grid would: the Predecessors
// column through ParsePredecessors, row numbers through the ID column.
func reimport(t *testing.T, raw []byte, ids []string) []linkKey {
	t.Helper()
	recs, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(recs[0], "|") != strings.Join(ExportHeader, "|") {
		t.Fatalf("header %v", recs[0])
	}
	recs = recs[1:]
	if len(recs) != len(ids) {
		t.Fatalf("%d rows, want %d", len(recs), len(ids))
	}
	idOf := map[int]string{}
	for i, rec := range recs {
		n, err := strconv.Atoi(rec[0])
		if err != nil || n != i+1 {
			t.Fatalf("row %d has ID %q", i+1, rec[0])
		}
		idOf[n] = ids[i]
	}
	var got []linkKey
	for i, rec := range recs {
		refs, err := ParsePredecessors(rec[6], 5)
		if err != nil {
			t.Fatalf("row %d predecessors %q: %v", i+1, rec[6], err)
		}
		for _, r := range refs {
			pred, ok := idOf[r.Row]
			if !ok {
				t.Fatalf("row %d refers to row %d, which is not exported", i+1, r.Row)
			}
			got = append(got, linkKey{pred, ids[i], r.Type, r.Lag, r.LagUnit})
		}
	}
	return sortLinks(got)
}

func exportCSV(t *testing.T, rows []ExportRow) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := WriteExportCSV(&b, rows); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestExportExpandedReimportsLinks(t *testing.T) {
	in := exportInput("2027-03-01")
	rows := ExportRows(in, Compute(in), exportOrigins, ViewExpanded)
	ids := []string{"s1", "a1", "a2", "m", "s2", "b1", "b2"}
	var names []string
	for _, r := range rows {
		names = append(names, r.Name)
	}
	if got := strings.Join(names, "|"); got != "Design|Concept|Detail|=Design signed off|Build|Footing inspection|Frame" {
		t.Fatalf("grid order %s", got)
	}
	if rows[6].Predecessors != "3FS+3cd, 4FF-1d" || rows[3].Predecessors != "2SS+2d" || rows[2].Predecessors != "2" || rows[4].Predecessors != "1" {
		t.Fatalf("predecessors %q %q %q %q", rows[6].Predecessors, rows[3].Predecessors, rows[2].Predecessors, rows[4].Predecessors)
	}
	all := map[string]bool{}
	for _, id := range ids {
		all[id] = true
	}
	if got, want := reimport(t, exportCSV(t, rows), ids), wantLinks(in, all); fmt.Sprint(got) != fmt.Sprint(want) || len(got) != len(in.Links) {
		t.Fatalf("re-imported links\n got %v\nwant %v", got, want)
	}
}

func TestExportCollapsedKeepsStageLinksOnly(t *testing.T) {
	in := exportInput("2027-03-01")
	rows := ExportRows(in, Compute(in), exportOrigins, ViewCollapsed)
	if len(rows) != 2 || rows[0].Name != "Design" || rows[1].Name != "Build" || rows[0].Tier != "Stage" {
		t.Fatalf("collapsed rows %+v", rows)
	}
	// Activity links across the collapsed stages are omitted, never lifted:
	// only the stage-to-stage link remains, and it points at row 1.
	if rows[0].Predecessors != "" || rows[1].Predecessors != "1" {
		t.Fatalf("collapsed predecessors %q %q", rows[0].Predecessors, rows[1].Predecessors)
	}
	got := reimport(t, exportCSV(t, rows), []string{"s1", "s2"})
	if want := wantLinks(in, map[string]bool{"s1": true, "s2": true}); fmt.Sprint(got) != fmt.Sprint(want) || len(got) != 1 {
		t.Fatalf("collapsed links %v want %v", got, want)
	}
}

func TestExportDatesDurationsAndUnknowns(t *testing.T) {
	in := exportInput("2027-03-01") // a Monday
	out := Compute(in)
	rows := ExportRows(in, out, exportOrigins, ViewExpanded)
	byName := map[string]ExportRow{}
	for _, r := range rows {
		byName[r.Name] = r
	}
	concept := byName["Concept"]
	if concept.Start != "2027-03-01" || concept.Finish != "2027-03-05" || *concept.Duration != 5 || concept.Origin != "User" || concept.Tier != "Activity" || !concept.Dated {
		t.Fatalf("concept %+v", concept)
	}
	ms := byName["=Design signed off"]
	if ms.Tier != "Milestone" || *ms.Duration != 0 || ms.Finish != "" || ms.Start == "" || ms.Origin != "Document" {
		t.Fatalf("milestone %+v", ms)
	}
	// The hold point has no duration: it and its stage stay blank, never
	// guessed, while their links still export.
	for _, name := range []string{"Footing inspection", "Build"} {
		r := byName[name]
		if r.Start != "" || r.Finish != "" || r.Duration != nil || r.Float != nil || r.Critical != "" {
			t.Fatalf("%s should be unknown: %+v", name, r)
		}
	}
	if byName["Footing inspection"].Tier != "Hold point" {
		t.Fatal("hold point tier")
	}
	for _, r := range rows {
		if r.Duration == nil {
			continue
		}
		tm := out.Activities[indexOf(in, r.Name)]
		if *r.Float != tm.TF || (r.Critical == "Yes") != tm.Critical {
			t.Fatalf("%s float %d critical %s, engine %d %v", r.Name, *r.Float, r.Critical, tm.TF, tm.Critical)
		}
	}
	raw := string(exportCSV(t, rows))
	// Blank cells for unknowns, and the formula-looking name defused.
	if !strings.Contains(raw, "6,Hold point,Footing inspection,,,,,,,Library") || !strings.Contains(raw, "'=Design signed off") {
		t.Fatalf("csv:\n%s", raw)
	}
}

func indexOf(in Input, label string) int {
	for i, a := range in.Activities {
		if a.Label == label {
			return i
		}
	}
	return -1
}

func TestExportRelativeProgramShowsWeeks(t *testing.T) {
	in := exportInput("")
	rows := ExportRows(in, Compute(in), exportOrigins, ViewExpanded)
	if rows[1].Start != "Week 1, day 1" || rows[1].Finish != "Week 1, day 5" || rows[1].Dated {
		t.Fatalf("relative concept %+v", rows[1])
	}
	if rows[2].Start != "Week 2, day 1" {
		t.Fatalf("relative detail %+v", rows[2])
	}
}

func TestExportCycleLeavesEveryRowUnknown(t *testing.T) {
	in := exportInput("2027-03-01")
	in.Links = append(in.Links, Link{ID: "back", Pred: "a2", Succ: "a1", Type: LinkFS})
	out := Compute(in)
	if out.Cycle == nil {
		t.Fatal("expected a cycle")
	}
	for _, r := range ExportRows(in, out, exportOrigins, ViewExpanded) {
		if r.Duration != nil || r.Start != "" {
			t.Fatalf("%s computed through a loop: %+v", r.Name, r)
		}
	}
}

func TestExportWorkbookOpens(t *testing.T) {
	in := exportInput("2027-03-01")
	rows := ExportRows(in, Compute(in), exportOrigins, ViewExpanded)
	var b bytes.Buffer
	if err := xlsx.WriteXLSX(&b, []xlsx.Sheet{ExportSheet(rows)}); err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(b.Bytes()), int64(b.Len()))
	if err != nil {
		t.Fatal(err)
	}
	var sheet string
	for _, f := range z.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			rc, _ := f.Open()
			raw, _ := io.ReadAll(rc)
			rc.Close()
			sheet = string(raw)
		}
	}
	// Header, eight rows; the predecessor cell verbatim; Concept's start as
	// the Excel serial of 1 March 2027 (46447).
	for _, want := range []string{"Predecessors", `<row r="8">`, "3FS+3cd, 4FF-1d", `<c r="D3" s="4"><v>46447</v></c>`} {
		if !strings.Contains(sheet, want) {
			t.Fatalf("sheet has no %s", want)
		}
	}
	if strings.Contains(sheet, `<row r="9">`) {
		t.Fatal("extra row")
	}
}
