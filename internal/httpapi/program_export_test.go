package httpapi_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"mime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

func exportInt(n int) *int { return &n }

// exportProgram is two stages with a milestone, a hold point of unknown
// duration and one link of each lag shape the typed format writes.
func exportProgram(start string) store.ProgramCreate {
	row := func(key, parent, kind, label string, d *int, origin string) store.ProgramRowInput {
		r := store.ProgramRowInput{Key: key, ParentKey: parent, Kind: kind, Label: label, DurationDays: d, ReviewStatus: "proposed", Origin: origin}
		if kind == "stage" {
			r.CatalogueStageID = "other"
		}
		if d != nil {
			r.DurationOrigin = "user"
		}
		return r
	}
	origin := ""
	if start != "" {
		origin = "user"
	}
	return store.ProgramCreate{Surface: "program", LibraryVersion: "export-test", StartDate: start, StartOrigin: origin,
		Rows: []store.ProgramRowInput{
			row("s1", "", "stage", "Design", nil, "library"),
			row("a1", "s1", "activity", "Concept", exportInt(5), "user"),
			row("a2", "s1", "activity", "Detail", exportInt(3), "library"),
			row("m", "s1", "milestone", "Design signed off", exportInt(0), "library"),
			row("s2", "", "stage", "Build", nil, "library"),
			row("b1", "s2", "hold_point", "Footing inspection", nil, "library"),
			row("b2", "s2", "activity", "Frame", exportInt(2), "user"),
		},
		Links: []store.ProgramLinkInput{
			{PredKey: "a1", SuccKey: "a2", Type: "FS", Origin: "chain"},
			{PredKey: "a1", SuccKey: "m", Type: "SS", LagDays: 2, LagUnit: "working", Origin: "user"},
			{PredKey: "a2", SuccKey: "b2", Type: "FS", LagDays: 3, LagUnit: "calendar", Origin: "user"},
			{PredKey: "m", SuccKey: "b2", Type: "FF", LagDays: -1, LagUnit: "working", Origin: "user"},
			{PredKey: "s1", SuccKey: "s2", Type: "FS", Origin: "chain"},
		},
	}
}

func TestProgramExportRoute(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Export: Smith/Jones")
	base := "/api/projects/" + project + "/program/export"

	// No program yet, and bad parameters.
	if resp := m.do(t, "GET", base, nil); resp.StatusCode != 404 {
		t.Fatalf("no program: %d", resp.StatusCode)
	}
	for _, q := range []string{"?format=pdf", "?view=gantt"} {
		if resp := m.do(t, "GET", base+q, nil); resp.StatusCode != 400 {
			t.Fatalf("%s: %d", q, resp.StatusCode)
		}
	}
	created, err := a.store.CreateProgram(ctx, m.orgID, project, exportProgram("2027-03-01"))
	if err != nil {
		t.Fatal(err)
	}
	st, err := a.store.LoadProgram(ctx, m.orgID, project)
	if err != nil {
		t.Fatal(err)
	}

	// The done criterion: the CSV's Predecessors column, read back through
	// the grid parser against the ID column, gives every stored link
	// between exported rows with its type, lag and unit.
	type link struct {
		pred, succ, typ string
		lag             int
		unit            string
	}
	reimport := func(raw []byte) ([]link, [][]string) {
		recs, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		recs = recs[1:]
		labelOf := map[int]string{}
		for _, r := range recs {
			n, _ := strconv.Atoi(r[0])
			labelOf[n] = r[2]
		}
		var out []link
		for _, r := range recs {
			refs, err := engine.ParsePredecessors(r[6], 5)
			if err != nil {
				t.Fatalf("%q: %v", r[6], err)
			}
			for _, ref := range refs {
				pred, ok := labelOf[ref.Row]
				if !ok {
					t.Fatalf("row %s refers to row %d, not exported", r[0], ref.Row)
				}
				out = append(out, link{pred, r[2], ref.Type, ref.Lag, ref.LagUnit})
			}
		}
		sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]) < fmt.Sprint(out[j]) })
		return out, recs
	}
	stored := func(keep map[string]bool) []link {
		label := map[string]string{}
		for _, act := range st.Activities {
			label[act.ID] = act.Label
		}
		var out []link
		for _, l := range st.Links {
			if keep[label[l.PredID]] && keep[label[l.SuccID]] {
				out = append(out, link{label[l.PredID], label[l.SuccID], l.Type, l.LagDays, l.LagUnit})
			}
		}
		sort.Slice(out, func(i, j int) bool { return fmt.Sprint(out[i]) < fmt.Sprint(out[j]) })
		return out
	}

	resp := m.do(t, "GET", base+"?format=csv", nil)
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/csv") {
		t.Fatalf("csv: %d %s", resp.StatusCode, raw)
	}
	_, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if params["filename"] != "Export  Smith Jones program.csv" {
		t.Fatalf("filename %q", params["filename"])
	}
	got, recs := reimport(raw)
	all := map[string]bool{}
	for _, act := range st.Activities {
		all[act.Label] = true
	}
	if want := stored(all); fmt.Sprint(got) != fmt.Sprint(want) || len(got) != 5 {
		t.Fatalf("expanded links\n got %v\nwant %v", got, want)
	}
	if len(recs) != 7 || recs[0][2] != "Design" || recs[1][2] != "Concept" || recs[4][2] != "Build" {
		t.Fatalf("grid order %v", recs)
	}
	// The hold point's duration is unknown: blank cells, never a guess.
	if r := recs[5]; r[1] != "Hold point" || r[3] != "" || r[4] != "" || r[5] != "" || r[7] != "" || r[8] != "" {
		t.Fatalf("unknown row %v", r)
	}
	if r := recs[1]; r[3] != "2027-03-01" || r[4] != "2027-03-05" || r[5] != "5" || r[9] != "User" {
		t.Fatalf("concept %v", r)
	}

	// Collapsed: stages only; the stage link survives, activity links are
	// omitted, and no reference points at a row that is not exported.
	got, recs = reimport(m.raw(t, "GET", base+"?format=csv&view=collapsed"))
	if len(recs) != 2 || recs[1][6] != "1" {
		t.Fatalf("collapsed %v", recs)
	}
	if want := stored(map[string]bool{"Design": true, "Build": true}); fmt.Sprint(got) != fmt.Sprint(want) || len(got) != 1 {
		t.Fatalf("collapsed links %v want %v", got, want)
	}

	// Excel is the default and opens as a workbook with the same rows.
	resp = m.do(t, "GET", base, nil)
	xl, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	_, params, _ = mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if resp.StatusCode != 200 || params["filename"] != "Export  Smith Jones program.xlsx" {
		t.Fatalf("xlsx: %d %q", resp.StatusCode, params["filename"])
	}
	z, err := zip.NewReader(bytes.NewReader(xl), int64(len(xl)))
	if err != nil {
		t.Fatalf("workbook does not open: %v", err)
	}
	var sheet []byte
	for _, f := range z.File {
		if f.Name == "xl/worksheets/sheet1.xml" {
			rc, _ := f.Open()
			sheet, _ = io.ReadAll(rc)
			rc.Close()
		}
	}
	if !bytes.Contains(sheet, []byte("3FS+3cd, 4FF-1d")) || !bytes.Contains(sheet, []byte(`<row r="8">`)) || bytes.Contains(sheet, []byte(`<row r="9">`)) {
		t.Fatalf("sheet: %.600s", sheet)
	}

	// Nothing about the export writes: the program version is unchanged.
	after, err := a.store.LoadProgram(ctx, m.orgID, project)
	if err != nil || after.Program.Version != st.Program.Version || created.ProgramID != st.Program.ID {
		t.Fatalf("export changed the program: %v %d", err, after.Program.Version)
	}
}

func TestProgramExportRelative(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Relative export")
	if _, err := a.store.CreateProgram(ctx, m.orgID, project, exportProgram("")); err != nil {
		t.Fatal(err)
	}
	recs, err := csv.NewReader(bytes.NewReader(m.raw(t, "GET", "/api/projects/"+project+"/program/export?format=csv"))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if r := recs[2]; r[3] != "Week 1, day 1" || r[4] != "Week 1, day 5" {
		t.Fatalf("relative concept %v", r)
	}
}

// TestProgramExportTiming times the Excel export of a 1,000-activity,
// 2,000-link program through the real HTTP server (budget program_export,
// p50 1,000 ms, p90 2,000 ms, plan 11.1). The full bench does the same in
// cmd/intake-bench.
func TestProgramExportTiming(t *testing.T) {
	ctx := context.Background()
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Export timing")
	in := store.ProgramCreate{Surface: "program", LibraryVersion: "timing", StartDate: "2027-03-01", StartOrigin: "user"}
	key := func(s, j int) string { return fmt.Sprintf("a%02d.%02d", s, j) }
	for s := 0; s < 20; s++ {
		sk := fmt.Sprintf("s%02d", s)
		in.Rows = append(in.Rows, store.ProgramRowInput{Key: sk, Kind: "stage", Label: fmt.Sprintf("Stage %d", s+1), CatalogueStageID: "other", ReviewStatus: "proposed", Origin: "library"})
		if s > 0 {
			in.Links = append(in.Links, store.ProgramLinkInput{PredKey: fmt.Sprintf("s%02d", s-1), SuccKey: sk, Type: "FS", Origin: "chain"})
		}
		for j := 0; j < 49; j++ {
			in.Rows = append(in.Rows, store.ProgramRowInput{Key: key(s, j), ParentKey: sk, Kind: "activity", Label: fmt.Sprintf("Activity %d.%d", s+1, j+1),
				DurationDays: exportInt(1 + (s*7+j)%10), DurationOrigin: "user", ReviewStatus: "proposed", Origin: "library"})
			if j > 0 {
				in.Links = append(in.Links, store.ProgramLinkInput{PredKey: key(s, j-1), SuccKey: key(s, j), Type: "FS", Origin: "chain"})
			}
		}
	}
	for s := 1; s < 20 && len(in.Links) < 2000; s++ {
		for j := 0; j < 49 && len(in.Links) < 2000; j++ {
			in.Links = append(in.Links, store.ProgramLinkInput{PredKey: key(s-1, j), SuccKey: key(s, j), Type: "FS", LagDays: j % 3, LagUnit: "working", Origin: "user"})
		}
	}
	for s := 0; s < 20 && len(in.Links) < 2000; s++ {
		for j := 2; j < 49 && len(in.Links) < 2000; j++ {
			in.Links = append(in.Links, store.ProgramLinkInput{PredKey: key(s, j-2), SuccKey: key(s, j), Type: "SS", Origin: "user"})
		}
	}
	if len(in.Rows) != 1000 || len(in.Links) != 2000 {
		t.Fatalf("%d rows %d links", len(in.Rows), len(in.Links))
	}
	if _, err := a.store.CreateProgram(ctx, m.orgID, project, in); err != nil {
		t.Fatal(err)
	}
	path := "/api/projects/" + project + "/program/export?format=xlsx&view=expanded"
	var took []time.Duration
	for k := 0; k < 21; k++ {
		start := time.Now()
		raw := m.raw(t, "GET", path)
		took = append(took, time.Since(start))
		if !bytes.HasPrefix(raw, []byte("PK")) {
			t.Fatal("not a workbook")
		}
	}
	sort.Slice(took, func(i, j int) bool { return took[i] < took[j] })
	p50, p90 := took[len(took)/2], took[len(took)*9/10]
	t.Logf("program_export 1,000 activities: p50 %v p90 %v (budget 1,000 ms / 2,000 ms)", p50, p90)
	if p50 > time.Second || p90 > 2*time.Second {
		t.Fatalf("program_export over budget: p50 %v p90 %v", p50, p90)
	}
}
