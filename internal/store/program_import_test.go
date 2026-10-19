package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/jev"
	engine "sitewise/internal/program"
	"sitewise/internal/programread"
	"sitewise/internal/store"
)

type greyAsker struct{}

func (greyAsker) Ask(context.Context, jev.Call) (jev.Result, error) {
	return jev.Result{}, context.DeadlineExceeded
}

// importHarvest is a builder's file: Slab, Framing and Roof match the
// sample program by rule; "Site establishment" matches nothing.
func importHarvest() programread.Harvest {
	d := func(v int) *engine.Duration { return &engine.Duration{Low: v, High: v} }
	return programread.Harvest{Filename: "Smith program.xlsx", Checks: []string{"Date order assumed day/month in column Start"}, Rows: []programread.Row{
		{ID: "r1", FileID: "1", Label: "Site establishment", StartRaw: "1/03/2027", FinishRaw: "5/03/2027", Start: "2027-03-01", Finish: "2027-03-05"},
		{ID: "r2", FileID: "2", Label: "Slab", StartRaw: "8/04/2027", FinishRaw: "21/04/2027", DurationRaw: "10 days", Start: "2027-04-08", Finish: "2027-04-21", Duration: d(10)},
		{ID: "r3", FileID: "3", Label: "Framing", StartRaw: "22/04/2027", FinishRaw: "12/05/2027", Start: "2027-04-22", Finish: "2027-05-12"},
		{ID: "r4", FileID: "4", Label: "Roof", StartRaw: "26/05/2027", FinishRaw: "1/06/2027", Start: "2027-05-26", Finish: "2027-06-01",
			PredLinks: []programread.PredecessorLink{{Row: "r2", Type: "FS", Lag: 2, LagUnit: "working"}}},
	}}
}

func readImport(t *testing.T, s *store.Store) (string, store.ProgramImportCommitted) {
	t.Helper()
	ctx := context.Background()
	id, err := s.OpenProgramImport(ctx, orgA, projectA, docA)
	if err != nil {
		t.Fatal(err)
	}
	refs, err := s.ClaimProgramImports(ctx, orgA, docA)
	if err != nil || len(refs) != 1 || refs[0].ImportID != id {
		t.Fatalf("claim %v %v", refs, err)
	}
	in, err := s.ProgramImportForReading(ctx, orgA, id)
	if err != nil {
		t.Fatal(err)
	}
	if in.PerWeek != 5 || len(in.Targets) != 8 || in.Targets[0].Tier != programread.TierStage || in.Targets[0].CatalogueStageID != "design" {
		t.Fatalf("input %+v", in)
	}
	out, err := programread.Read(ctx, greyAsker{}, programread.Thresholds{QuestionVersion: programread.QuestionVersion}, importHarvest(), in.Targets)
	if err != nil {
		t.Fatal(err)
	}
	done, ok, err := s.CommitProgramImport(ctx, orgA, id, out, "", false, time.Now())
	if err != nil || !ok {
		t.Fatalf("commit %v %v", ok, err)
	}
	return id, done
}

func readingIDs(v store.ProgramView, target, field string) []string {
	var out []string
	for _, r := range v.Readings {
		if string(r.TargetActivityID) == target && (field == "" || r.Field == field) {
			out = append(out, r.ID)
		}
	}
	return out
}

func TestProgramImportReadingsAcceptUndoDismissAdd(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st := newProgram(t, s)
	slab, framing, roof, construction := rowID(t, st, "Slab"), rowID(t, st, "Framing"), rowID(t, st, "Roof"), rowID(t, st, "Construction")

	// The user's own value is never overwritten (plan section 2, rule 14).
	res, err := s.ApplyProgramOps(ctx, orgA, projectA, userA, st.Program.ID, st.Program.Version, []store.ProgramOp{{Op: "update", ID: framing, DurationDays: dur(12)}})
	if err != nil {
		t.Fatal(err)
	}
	importID, done := readImport(t, s)
	if done.ReadingsPending != 7 {
		t.Fatalf("pending %d", done.ReadingsPending)
	}
	st, _ = s.LoadProgram(ctx, orgA, projectA)
	v, err := s.ProgramView(ctx, orgA, projectA, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.ReadingsPending != 7 || len(v.Imports) != 1 {
		t.Fatalf("view %+v", v.ProgramExtras)
	}
	imp := v.Imports[0]
	if imp.Text != "3 readings from plan.pdf" || imp.Unmatched != 1 || imp.UnmatchedRows[0].Label != "Site establishment" {
		t.Fatalf("import %+v", imp)
	}
	if len(imp.Links) != 1 || imp.Links[0].PredID != slab || imp.Links[0].SuccID != roof || imp.Links[0].LagDays != 2 {
		t.Fatalf("links %+v", imp.Links)
	}
	found := false
	for _, c := range v.Checks {
		found = found || c.Code == "date_order_assumed"
	}
	if !found {
		t.Fatalf("checks %+v", v.Checks)
	}
	for _, r := range v.Readings {
		if string(r.TargetActivityID) == framing && r.Field == "duration" {
			t.Fatal("no duration reading without a duration cell")
		}
		if string(r.TargetActivityID) == framing && r.Field == "start" && !r.Applies {
			t.Fatal("framing start applies: its constraint is the default")
		}
	}

	// Accept Slab's start: the whole Slab group applies.
	// A read never bumps the program version.
	if _, err = s.AcceptProgram(ctx, orgA, projectA, userA, res.Program.Version, readingIDs(v, slab, "start"), false); err != nil {
		t.Fatal(err)
	}
	st, _ = s.LoadProgram(ctx, orgA, projectA)
	a := rowOf(t, st, "Slab")
	if a.DurationDays == nil || *a.DurationDays != 10 || a.DurationOrigin != "document" || a.ConstraintKind != "snet" || a.ConstraintDate != "2027-04-08" {
		t.Fatalf("slab %+v", a)
	}
	v, _ = s.ProgramView(ctx, orgA, projectA, st, nil)
	if len(readingIDs(v, slab, "")) != 0 || v.ReadingsPending != 4 {
		t.Fatalf("slab readings still proposed: %d", v.ReadingsPending)
	}

	// Undo puts the row and the readings back.
	res, err = s.UndoProgram(ctx, orgA, projectA, userA, st.Program.Version)
	if err != nil {
		t.Fatal(err)
	}
	a = rowOf(t, res.ProgramState, "Slab")
	if *a.DurationDays != 10 || a.DurationOrigin != "library" || a.ConstraintKind != "asap" {
		t.Fatalf("undo %+v", a)
	}
	v, _ = s.ProgramView(ctx, orgA, projectA, res.ProgramState, nil)
	if v.ReadingsPending != 7 {
		t.Fatalf("undo pending %d", v.ReadingsPending)
	}

	// Accept all: Framing keeps the user's 12 days and its readings stay
	// proposed; Roof gets its dates and the file's Slab → Roof link.
	res, err = s.AcceptProgram(ctx, orgA, projectA, userA, res.Program.Version, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if f := rowOf(t, res.ProgramState, "Framing"); *f.DurationDays != 12 || f.DurationOrigin != "user" || f.ConstraintKind != "asap" {
		t.Fatalf("framing %+v", f)
	}
	if r := rowOf(t, res.ProgramState, "Roof"); *r.DurationDays != 5 || r.ConstraintDate != "2027-05-26" {
		t.Fatalf("roof %+v", r)
	}
	if l, ok := linkBetween(res.ProgramState, slab, roof); !ok || l.Origin != "document" || l.LagDays != 2 {
		t.Fatalf("file link %+v %v", l, ok)
	}
	v, _ = s.ProgramView(ctx, orgA, projectA, res.ProgramState, nil)
	if len(readingIDs(v, framing, "")) != 2 {
		t.Fatalf("framing readings %+v", v.Readings)
	}

	// Dismiss Framing's readings.
	res, err = s.DismissProgramReadings(ctx, orgA, projectA, userA, res.Program.Version, readingIDs(v, framing, "start"))
	if err != nil {
		t.Fatal(err)
	}
	v, _ = s.ProgramView(ctx, orgA, projectA, res.ProgramState, nil)
	if v.ReadingsPending != 0 {
		t.Fatalf("pending after dismiss %d", v.ReadingsPending)
	}

	// The unmatched row is added in one click under Construction.
	res, err = s.AddProgramImportRows(ctx, orgA, projectA, userA, res.Program.Version, []store.ProgramRowAdd{{ImportID: importID, Row: "r1", ParentID: construction}})
	if err != nil {
		t.Fatal(err)
	}
	site := rowOf(t, res.ProgramState, "Site establishment")
	if site.Origin != "document" || *site.DurationDays != 5 || site.ConstraintDate != "2027-03-01" || string(site.ParentID) != construction {
		t.Fatalf("added %+v", site)
	}
	if _, err := s.AddProgramImportRows(ctx, orgA, projectA, userA, res.Program.Version, []store.ProgramRowAdd{{ImportID: importID, Row: "r1", ParentID: construction}}); err == nil {
		t.Fatal("added the same row twice")
	}
	v, _ = s.ProgramView(ctx, orgA, projectA, res.ProgramState, nil)
	// Site establishment is in the program now; Framing's match was
	// dismissed, so its row is free to add instead.
	if v.Imports[0].Unmatched != 1 || v.Imports[0].UnmatchedRows[0].Label != "Framing" {
		t.Fatalf("unmatched %+v", v.Imports[0])
	}

	// A re-read never proposes again what the user decided.
	refsIn, err := s.ProgramImportForReading(ctx, orgA, importID)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := programread.Read(ctx, greyAsker{}, programread.Thresholds{QuestionVersion: programread.QuestionVersion}, importHarvest(), refsIn.Targets)
	if _, ok, err := s.CommitProgramImport(ctx, orgA, importID, out, "", true, time.Now()); err != nil || !ok {
		t.Fatalf("recheck %v %v", ok, err)
	}
	if n, _ := s.ImportReadingsPending(ctx, orgA, st.Program.ID); n != 0 {
		t.Fatalf("re-read proposed decided readings again: %d", n)
	}
}

func TestProgramImportIsOrgScoped(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	newProgram(t, s)
	if _, err := s.OpenProgramImport(ctx, orgB, projectA, docA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other org opened an import: %v", err)
	}
	id, err := s.OpenProgramImport(ctx, orgA, projectA, docA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ProgramImportForReading(ctx, orgB, id); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other org read an import: %v", err)
	}
	if refs, err := s.ClaimProgramImports(ctx, orgB, docA); err != nil || len(refs) != 0 {
		t.Fatalf("other org claimed: %v %v", refs, err)
	}
}
