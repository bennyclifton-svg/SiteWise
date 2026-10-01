package intake_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/latency"
)

func TestCandidateMultipleNumbers(t *testing.T) {
	filename := "A-101 Rev C - Ground Floor.pdf"
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Drawing No.", Source: identity.Source{Page: 1}},
		{Text: "A-101", Source: identity.Source{Page: 1}},
		{Text: "S-201", Source: identity.Source{Page: 1, Rotation: 270}},
		{Text: "221102", Source: identity.Source{Page: 1}},
	}}
	got := intake.Harvest(filename, text)

	for _, want := range []string{"A-101", "S-201", "221102"} {
		c, ok := findCandidate(got, intake.FieldNumber, want)
		if !ok {
			t.Fatalf("missing number %s in %+v", want, displays(got, intake.FieldNumber))
		}
		if want == "A-101" && filename[c.Provenance.Start:c.Provenance.End] != want && text.Runs[1].Text[c.Provenance.Start:c.Provenance.End] != want {
			t.Fatalf("A-101 provenance does not cover the literal: %+v", c.Provenance)
		}
	}
	s201, ok := findCandidate(got, intake.FieldNumber, "S-201")
	if !ok || s201.Provenance.Page != 1 || s201.Provenance.Rotation != 270 {
		t.Fatalf("S-201 provenance: %+v", s201)
	}
	if c, ok := findCandidate(got, intake.FieldNumber, "C"); ok {
		t.Fatalf("revision harvested as a number: %+v", c)
	}
	number := fieldResult(t, intake.Decide(got), intake.FieldNumber)
	if number.Settled || number.Rule != intake.RuleAmbiguous {
		t.Fatalf("multiple numbers settled: %+v", number)
	}
}

func TestCandidateTitleBlockDisagreesWithFilename(t *testing.T) {
	filename := "A-101 [B] Site Plan.pdf"
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Drawing No.", Source: identity.Source{Page: 1}},
		{Text: "A-102", Source: identity.Source{Page: 1}},
	}}
	got := intake.Harvest(filename, text)
	fileNumber, ok := findOrigin(got, intake.FieldNumber, "A-101", intake.OriginFilename)
	if !ok {
		t.Fatalf("filename number missing: %+v", displays(got, intake.FieldNumber))
	}
	if filename[fileNumber.Provenance.Start:fileNumber.Provenance.End] != "A-101" {
		t.Fatalf("filename byte span: %+v", fileNumber.Provenance)
	}
	block, ok := findOrigin(got, intake.FieldNumber, "A-102", intake.OriginText)
	if !ok || !block.Provenance.Labeled || block.Provenance.Page != 1 {
		t.Fatalf("title-block number: %+v", block)
	}
	number := fieldResult(t, intake.Decide(got), intake.FieldNumber)
	if number.Settled {
		t.Fatalf("disagreement settled: %+v", number)
	}
	rev := fieldResult(t, intake.Decide(got), intake.FieldRevision)
	if !rev.Settled || rev.Display != "B" || rev.Rule != intake.RuleUnique {
		t.Fatalf("agreed revision: %+v", rev)
	}
}

func TestCandidateMissingRevision(t *testing.T) {
	filename := "A-101 Site Plan.pdf"
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Drawing No.", Source: identity.Source{Page: 1}},
		{Text: "A-101", Source: identity.Source{Page: 1}},
		{Text: "Site Plan", Source: identity.Source{Page: 1, Heading: true}},
	}}
	got := intake.Harvest(filename, text)
	if len(displays(got, intake.FieldRevision)) != 0 {
		t.Fatalf("invented revision: %+v", displays(got, intake.FieldRevision))
	}
	rev := fieldResult(t, intake.Decide(got), intake.FieldRevision)
	if rev.Settled || rev.Display != "" || rev.Rule != intake.RuleMissing {
		t.Fatalf("missing revision: %+v", rev)
	}
	number := fieldResult(t, intake.Decide(got), intake.FieldNumber)
	if !number.Settled || number.Display != "A-101" || number.Normalized != "A-101" {
		t.Fatalf("agreed number: %+v", number)
	}
}

func TestCandidateDateLikeNumbers(t *testing.T) {
	filename := "Report 12-03-2024.pdf"
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Date", Source: identity.Source{Page: 1}},
		{Text: "12/03/2024", Source: identity.Source{Page: 1}},
		{Text: "2024-03-12", Source: identity.Source{Page: 1}},
		{Text: "15.12.23", Source: identity.Source{Page: 1}},
		{Text: "Drawing No.", Source: identity.Source{Page: 1}},
		{Text: "WTJ23410", Source: identity.Source{Page: 1}},
	}}
	got := intake.Harvest(filename, text)
	for _, banned := range []string{"12-03-2024", "12/03/2024", "2024-03-12", "15.12.23", "2024"} {
		if _, ok := findCandidate(got, intake.FieldNumber, banned); ok {
			t.Fatalf("date harvested as number %s: %+v", banned, displays(got, intake.FieldNumber))
		}
	}
	for _, want := range []string{"12-03-2024", "12/03/2024", "2024-03-12", "15.12.23"} {
		if _, ok := findCandidate(got, intake.FieldDate, want); !ok {
			t.Fatalf("missing date %s in %+v", want, displays(got, intake.FieldDate))
		}
	}
	dated, ok := findCandidate(got, intake.FieldDate, "12/03/2024")
	if !ok || dated.Display != "12/03/2024" || dated.Normalized != "12/03/2024" {
		t.Fatalf("date literal: %+v", dated)
	}
	number := fieldResult(t, intake.Decide(got), intake.FieldNumber)
	if !number.Settled || number.Display != "WTJ23410" {
		t.Fatalf("document number: %+v candidates %v", number, displays(got, intake.FieldNumber))
	}
	if fieldResult(t, intake.Decide(got), intake.FieldRevision).Settled {
		t.Fatal("date became a revision")
	}
}

func TestCandidateCellProvenance(t *testing.T) {
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Drawing No.", Source: identity.Source{Sheet: "Cover", Cell: "A2", Row: 2, Col: 1}},
		{Text: "A-101", Source: identity.Source{Sheet: "Cover", Cell: "B2", Row: 2, Col: 2, Merge: "B2:C2"}},
	}}
	got := intake.Harvest("register.xlsx", text)
	c, ok := findOrigin(got, intake.FieldNumber, "A-101", intake.OriginText)
	if !ok {
		t.Fatal("missing cell number")
	}
	if c.Provenance.Sheet != "Cover" || c.Provenance.Cell != "B2" || c.Provenance.Merge != "B2:C2" || c.Provenance.Row != 2 || c.Provenance.Col != 2 {
		t.Fatalf("cell provenance: %+v", c.Provenance)
	}
	if text.Runs[1].Text[c.Provenance.Start:c.Provenance.End] != "A-101" {
		t.Fatalf("cell byte span: %+v", c.Provenance)
	}
}

func TestCandidatePSeriesTokensAreRevisions(t *testing.T) {
	got := intake.Harvest("sketch.pdf", identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Issued P10 not P2", Source: identity.Source{Page: 1}},
	}})
	for _, want := range []string{"P10", "P2"} {
		c, ok := findCandidate(got, intake.FieldRevision, want)
		if !ok || c.Normalized != want {
			t.Fatalf("P-series %s: %+v", want, c)
		}
		if _, ok := findCandidate(got, intake.FieldNumber, want); ok {
			t.Fatalf("%s harvested as a number", want)
		}
	}
	ordered, ok := intake.Sequence([]string{"P10", "P2"})
	if !ok || strings.Join(ordered, ",") != "P2,P10" {
		t.Fatalf("sequence: %v %v", ordered, ok)
	}
}

func TestCandidateHarvestBudget(t *testing.T) {
	name, text := denseIdentity()
	samples := make([]int64, 0, 20)
	for i := 0; i < 20; i++ {
		samples = append(samples, timedCall(func() { intake.Harvest(name, text) }))
	}
	assertBudget(t, "candidate_harvesting", samples, 5_000, 10_000)
}

func TestRuleCatalogIsCopiedNotInvented(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "data", "intake")
	cat, err := intake.LoadCatalog(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Disciplines) != 57 {
		t.Fatalf("disciplines: %d", len(cat.Disciplines))
	}
	raw, err := os.ReadFile(filepath.Join(dir, "disciplines.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"participant_type", "request_kind", "workspace_slug", "pmp_label", "picker_visible"} {
		if strings.Contains(string(raw), banned) {
			t.Fatalf("doctrine field %s copied", banned)
		}
	}
	civil := findDiscipline(t, cat, "consultant.civil")
	if !contains(civil.Aliases, "Stormwater") {
		t.Fatalf("civil aliases: %v", civil.Aliases)
	}
	clerk := filepath.Join(repoRoot(t), "..", "clerk", "data", "taxonomy", "disciplines.json")
	body, err := os.ReadFile(clerk)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if hex.EncodeToString(sum[:]) != cat.DisciplineSourceSHA256 {
		t.Fatalf("source hash %s", cat.DisciplineSourceSHA256)
	}
	if cat.KindListComplete || cat.KindDesignCount != 16 || len(cat.Kinds) != 12 || cat.KindReconciliation == "" {
		t.Fatalf("kind reconciliation: complete=%v design=%d found=%d", cat.KindListComplete, cat.KindDesignCount, len(cat.Kinds))
	}
	var wantKinds = []string{
		"design_brief", "drawing", "specification", "report", "certificate", "correspondence",
		"contract", "commercial", "schedule", "statutory_instrument", "photo", "unknown",
	}
	for i, id := range wantKinds {
		if cat.Kinds[i].ID != id || cat.Kinds[i].Label == "" {
			t.Fatalf("kind %d: %+v", i, cat.Kinds[i])
		}
	}
	if cat.LifecycleReconciliation == "" || !hasArea(cat, "design") || !hasArea(cat, "construction") {
		t.Fatalf("lifecycle: %+v", cat.Lifecycle)
	}
	kindRaw, err := os.ReadFile(filepath.Join(dir, "kinds.json"))
	if err != nil {
		t.Fatal(err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(kindRaw, &probe); err != nil {
		t.Fatal(err)
	}
	if _, ok := probe["reconciliation"]; !ok {
		t.Fatal("kinds.json has no reconciliation")
	}
}

func TestRuleBudget(t *testing.T) {
	name, text := denseIdentity()
	candidates := intake.Harvest(name, text)
	samples := make([]int64, 0, 20)
	for i := 0; i < 20; i++ {
		samples = append(samples, timedCall(func() { intake.Decide(candidates) }))
	}
	assertBudget(t, "deterministic_field_rules", samples, 1_000, 1_000)
}

func TestRuleSettlesSheetLiteral(t *testing.T) {
	filename := "a-101 site plan.pdf"
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "Drawing No. A-101", Source: identity.Source{Page: 1}},
		{Text: "Title", Source: identity.Source{Page: 1}},
		{Text: "Ground Floor", Source: identity.Source{Page: 1}},
	}}
	got := intake.Harvest(filename, text)
	number := fieldResult(t, intake.Decide(got), intake.FieldNumber)
	if !number.Settled || number.Display != "A-101" || number.Rule != intake.RuleUnique {
		t.Fatalf("sheet literal: %+v", number)
	}
	title := fieldResult(t, intake.Decide(got), intake.FieldTitle)
	if title.Settled {
		t.Fatalf("title disagreement settled: %+v candidates %v", title, displays(got, intake.FieldTitle))
	}
}

func findCandidate(cands []intake.Candidate, field, display string) (intake.Candidate, bool) {
	for _, c := range cands {
		if c.Field == field && c.Display == display {
			return c, true
		}
	}
	return intake.Candidate{}, false
}

func findOrigin(cands []intake.Candidate, field, display, origin string) (intake.Candidate, bool) {
	for _, c := range cands {
		if c.Field == field && c.Display == display && c.Provenance.Origin == origin {
			return c, true
		}
	}
	return intake.Candidate{}, false
}

func displays(cands []intake.Candidate, field string) []string {
	var out []string
	for _, c := range cands {
		if c.Field == field {
			out = append(out, c.Display)
		}
	}
	return out
}

func fieldResult(t *testing.T, results []intake.Result, field string) intake.Result {
	t.Helper()
	for _, r := range results {
		if r.Field == field {
			return r
		}
	}
	t.Fatalf("no result for %s", field)
	return intake.Result{}
}

func findDiscipline(t *testing.T, cat intake.Catalog, id string) intake.Discipline {
	t.Helper()
	for _, d := range cat.Disciplines {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("missing %s", id)
	return intake.Discipline{}
}

func hasArea(cat intake.Catalog, id string) bool {
	for _, area := range cat.Lifecycle {
		if area.ID == id && area.Label != "" {
			return true
		}
	}
	return false
}

func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func denseIdentity() (string, identity.Text) {
	runs := make([]identity.Run, 400)
	for i := range runs {
		runs[i] = identity.Run{
			Text:   "Drawing No. A-101 Rev P10 Date 12/03/2024 Ground Floor Plan Project 221102",
			Source: identity.Source{Page: 1, Sheet: "Cover", Cell: "A1"},
		}
	}
	return "221102_A-101 [P10] - Ground Floor Plan.pdf", identity.Text{TextLayer: true, Runs: runs}
}

func timedCall(fn func()) int64 {
	const n = 32
	start := time.Now()
	for i := 0; i < n; i++ {
		fn()
	}
	return time.Since(start).Microseconds() / n
}

func assertBudget(t *testing.T, name string, samples []int64, p50us, p90us int64) {
	t.Helper()
	p50, err := latency.Percentile(samples, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	p90, err := latency.Percentile(samples, 0.9)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s p50=%dus p90=%dus", name, p50, p90)
	if p50 > p50us || p90 > p90us {
		t.Fatalf("%s p50 %dus p90 %dus exceeds %d/%d", name, p50, p90, p50us, p90us)
	}
}

func run(text string) identity.Run { return identity.Run{Text: text, Source: identity.Source{Page: 1}} }

func TestCandidateRejectsImpossibleDates(t *testing.T) {
	got := intake.Harvest("sheet.pdf", identity.Text{TextLayer: true, Runs: []identity.Run{
		run("90/90/90"), run("31/02/2023x"), run("13/13/2023"), run("2023-14-01"), run("05.12.2023"),
	}})
	dates := displays(got, intake.FieldDate)
	if len(dates) != 1 || dates[0] != "05.12.2023" {
		t.Fatalf("only a calendar date is a date: %v", dates)
	}
}

func TestCandidateCaptionIsNotATitle(t *testing.T) {
	got := intake.Harvest("sheet.pdf", identity.Text{TextLayer: true, Runs: []identity.Run{
		run("SHEET TITLE"), run("DRAWN"),
		run("Drawing Title:"), run("Drawing No:"),
		run("Title"), run("ARCHITECT"),
	}})
	if titles := displays(got, intake.FieldTitle); len(titles) != 0 {
		t.Fatalf("title-block captions harvested as titles: %v", titles)
	}
}

func TestCandidatePrelimInBodyTextIsNotARevision(t *testing.T) {
	got := intake.Harvest("sheet.pdf", identity.Text{TextLayer: true, Runs: []identity.Run{
		run("P23 CONDUIT TO PIT"), run("P2"),
	}})
	revs := displays(got, intake.FieldRevision)
	if len(revs) != 1 || revs[0] != "P2" {
		t.Fatalf("a P-token inside a note is not a revision; a standalone one may be: %v", revs)
	}
}

func TestCandidateSplitsRevisionSuffixedFilenameNumber(t *testing.T) {
	// A stem that is only <sheet>-<two digits> is the sheet at that revision.
	got := intake.Harvest("S204-03.pdf", identity.Text{TextLayer: true})
	if n := displays(got, intake.FieldNumber); len(n) != 1 || n[0] != "S204" {
		t.Fatalf("the whole-stem form is split, not offered fused: %v", n)
	}
	if r := displays(got, intake.FieldRevision); len(r) != 1 || r[0] != "03" {
		t.Fatalf("the suffix is the revision: %v", r)
	}
	c, _ := findCandidate(got, intake.FieldNumber, "S204")
	if "S204-03.pdf"[c.Provenance.Start:c.Provenance.End] != "S204" {
		t.Fatalf("provenance covers the literal: %+v", c.Provenance)
	}

	// Inside a longer name code offers both readings and does not choose.
	got = intake.Harvest("S204-03 Shoring Plan.pdf", identity.Text{TextLayer: true})
	for _, want := range []string{"S204", "S204-03"} {
		if _, ok := findCandidate(got, intake.FieldNumber, want); !ok {
			t.Fatalf("missing %s: %v", want, displays(got, intake.FieldNumber))
		}
	}
	if number := fieldResult(t, intake.Decide(got), intake.FieldNumber); number.Settled {
		t.Fatalf("code does not choose between S204 and S204-03: %+v", number)
	}
}

func TestCandidateStandaloneShortSheetNumber(t *testing.T) {
	got := intake.Harvest("M07 - MECHANICAL - ROOF PLAN - [B2].pdf", identity.Text{TextLayer: true, Runs: []identity.Run{
		run("M07"), run("NOTE M07 IS SHOWN"),
	}})
	n := displays(got, intake.FieldNumber)
	if len(n) != 2 || n[0] != "M07" || n[1] != "M07" {
		t.Fatalf("a short sheet number counts from the filename's lead token and a standalone cell only: %v", n)
	}
}

func TestCandidateShortFilenameIsNotATitle(t *testing.T) {
	got := intake.Harvest("M07-ME~1.PDF", identity.Text{TextLayer: true})
	if titles := displays(got, intake.FieldTitle); len(titles) != 0 {
		t.Fatalf("an 8.3 short name is not a title: %v", titles)
	}
	if _, ok := findCandidate(got, intake.FieldNumber, "M07"); !ok {
		t.Fatalf("its lead sheet number still counts: %v", displays(got, intake.FieldNumber))
	}
}

func TestCandidateTitleNextToTheSheetNumberCell(t *testing.T) {
	got := intake.Harvest("sheet.pdf", identity.Text{TextLayer: true, Runs: []identity.Run{
		run("GENERAL NOTE ONE APPLIES TO ALL WORK SHOWN ON THIS DRAWING AND SHALL BE READ WITH THE SPECIFICATION"),
		run("PLANT ROOM - MECHANICAL LAYOUT"),
		run("2024-03-01"),
		run("M-207"),
		run("DRAWN"),
		run("1:100"),
		run("FAR AWAY TEXT ONE"), run("FAR AWAY TEXT TWO"), run("FAR AWAY TEXT THREE"), run("FAR AWAY TEXT FOUR"),
	}})
	titles := displays(got, intake.FieldTitle)
	if len(titles) != 1 || titles[0] != "PLANT ROOM - MECHANICAL LAYOUT" {
		t.Fatalf("a title-like cell beside the sheet number is a title candidate; notes and distant cells are not: %v", titles)
	}
}
