package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"sitewise/internal/programread"
	"sitewise/internal/store"
)

const holdIface = "if.envelope-frame-before-roof"

// holdProgram is the sample program with one generated hold point in
// Construction, as generation makes it (source key hold:<interface>).
func holdProgram(t *testing.T, s *store.Store) (store.ProgramState, string) {
	t.Helper()
	ctx := context.Background()
	in := sampleProgram()
	in.Rows = append(in.Rows, store.ProgramRowInput{Key: "hold", ParentKey: "construction", Kind: "hold_point", Label: "Hold point: frame inspection",
		InterfaceID: holdIface, SourceKey: "hold:" + holdIface, ReviewStatus: "proposed", Origin: "library",
		DurationOrigin: "calculation", DurationSource: json.RawMessage(`{"hold_point":"` + holdIface + `"}`)})
	if _, err := s.CreateProgram(ctx, orgA, projectA, in); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	return st, rowID(t, st, "Hold point: frame inspection")
}

func released(t *testing.T, st store.ProgramState, id string) map[string]any {
	t.Helper()
	for _, a := range st.Activities {
		if a.ID != id {
			continue
		}
		var src map[string]any
		if err := json.Unmarshal(a.DurationSource, &src); err != nil {
			t.Fatal(err)
		}
		if src["hold_point"] != holdIface {
			t.Fatalf("hold point source lost: %s", a.DurationSource)
		}
		r, _ := src["released"].(map[string]any)
		return r
	}
	t.Fatalf("no row %s", id)
	return nil
}

func holdView(t *testing.T, s *store.Store, st store.ProgramState) []store.ProgramHoldPointView {
	t.Helper()
	v, err := s.ProgramView(context.Background(), orgA, projectA, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	return v.HoldPoints
}

func TestProgramHoldPointReleaseOnlyOnClick(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st, hold := holdProgram(t, s)

	targets, err := s.DocumentHoldPoints(ctx, orgA, docA)
	if err != nil || len(targets) != 1 || targets[0].ActivityID != hold || targets[0].InterfaceID != holdIface || targets[0].ProgramID != st.Program.ID {
		t.Fatalf("targets %+v %v", targets, err)
	}
	// Another org cannot see this program's hold points through its own id.
	if other, err := s.DocumentHoldPoints(ctx, orgB, docA); err != nil || len(other) != 0 {
		t.Fatalf("cross-org targets %+v %v", other, err)
	}
	if len(holdView(t, s, st)) != 0 {
		t.Fatal("hold point listed with no evidence")
	}

	// A reading is written as a proposal and changes no program row.
	passage := newUUIDString()
	if err := s.ReplaceHoldPointReadings(ctx, orgA, docA, []store.HoldPointReading{{ProgramID: st.Program.ID, ActivityID: hold, InterfaceID: holdIface, PassageID: passage, Band: programread.BandGreen, Probability: 0.93}}); err != nil {
		t.Fatal(err)
	}
	after, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if after.Program.Version != st.Program.Version || released(t, after, hold) != nil || rowOf(t, after, "Hold point: frame inspection").ReviewStatus != "proposed" {
		t.Fatal("a reading released the hold point without a click")
	}
	hv := holdView(t, s, after)
	if len(hv) != 1 || hv[0].ActivityID != hold || len(hv[0].Evidence) != 1 || hv[0].Released != nil || hv[0].Limit != programread.HoldPointLimit {
		t.Fatalf("view %+v", hv)
	}
	ev := hv[0].Evidence[0]
	if !strings.HasPrefix(ev.Text, "Evidence found: ") || ev.DocumentID != docA || ev.PassageID != passage || ev.Band != "green" || ev.DateLabel != "document dated" || len(ev.DocumentDate) != 10 {
		t.Fatalf("evidence %+v", ev)
	}
	// Replacing drops the document's old proposals; nothing stays twice.
	if err := s.ReplaceHoldPointReadings(ctx, orgA, docA, []store.HoldPointReading{{ProgramID: st.Program.ID, ActivityID: hold, InterfaceID: holdIface, PassageID: passage, Band: programread.BandAmber, Probability: 0.7}}); err != nil {
		t.Fatal(err)
	}
	hv = holdView(t, s, after)
	if len(hv) != 1 || len(hv[0].Evidence) != 1 || hv[0].Evidence[0].Band != "amber" {
		t.Fatalf("replaced view %+v", hv)
	}
	reading := hv[0].Evidence[0].ReadingID

	// Evidence for another row is refused; nothing is written.
	if _, err := s.ReleaseHoldPoint(ctx, orgA, projectA, userA, after.Program.Version, rowID(t, after, "Slab"), reading); !isOpErr(err, "id") {
		t.Fatalf("released a non-hold row: %v", err)
	}
	if _, err := s.ReleaseHoldPoint(ctx, orgA, projectA, userA, after.Program.Version, hold, newUUIDString()); !isOpErr(err, "reading_id") {
		t.Fatalf("released with unknown evidence: %v", err)
	}

	// The click: one batch, one version, evidence accepted with it.
	res, err := s.ReleaseHoldPoint(ctx, orgA, projectA, userA, after.Program.Version, hold, reading)
	if err != nil {
		t.Fatal(err)
	}
	if res.Program.Version != after.Program.Version+1 {
		t.Fatalf("version %d after %d", res.Program.Version, after.Program.Version)
	}
	rel := released(t, res.ProgramState, hold)
	if rel == nil || rel["by"] != "user" || rel["reading_id"] != reading || rel["document_id"] != docA || rel["passage_id"] != passage || len(rel["document_date"].(string)) != 10 {
		t.Fatalf("release %+v", rel)
	}
	if rowOf(t, res.ProgramState, "Hold point: frame inspection").ReviewStatus != "accepted" {
		t.Fatal("released hold point still proposed")
	}
	hv = holdView(t, s, res.ProgramState)
	if len(hv) != 1 || hv[0].Released == nil || hv[0].Released.DocumentID != docA || hv[0].Released.DateLabel != "document dated" || hv[0].Released.DocumentTitle == "" || len(hv[0].Evidence) != 0 {
		t.Fatalf("released view %+v", hv)
	}
	if targets, _ := s.DocumentHoldPoints(ctx, orgA, docA); len(targets) != 0 {
		t.Fatal("a released hold point is still read for evidence")
	}
	if _, err := s.ReleaseHoldPoint(ctx, orgA, projectA, userA, res.Program.Version, hold, ""); !isOpErr(err, "id") {
		t.Fatalf("released twice: %v", err)
	}
	// A 0-day edit keeps the release.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "update", ID: hold, DurationDays: dur(0)})
	if released(t, res.ProgramState, hold) == nil {
		t.Fatal("a 0-day edit cleared the release")
	}

	// Undo the edit, then the release: the release goes, the evidence is
	// proposed again.
	if res, err = s.UndoProgram(ctx, orgA, projectA, userA, res.Program.Version); err != nil {
		t.Fatal(err)
	}
	if res, err = s.UndoProgram(ctx, orgA, projectA, userA, res.Program.Version); err != nil {
		t.Fatal(err)
	}
	if released(t, res.ProgramState, hold) != nil || rowOf(t, res.ProgramState, "Hold point: frame inspection").ReviewStatus != "proposed" {
		t.Fatal("undo left the release")
	}
	hv = holdView(t, s, res.ProgramState)
	if len(hv) != 1 || hv[0].Released != nil || len(hv[0].Evidence) != 1 || hv[0].Evidence[0].ReadingID != reading {
		t.Fatalf("undone view %+v", hv)
	}

	// Released without evidence, by the user's request alone; undone.
	res, err = s.ReleaseHoldPoint(ctx, orgA, projectA, userA, res.Program.Version, hold, "")
	if err != nil {
		t.Fatal(err)
	}
	if rel := released(t, res.ProgramState, hold); rel == nil || rel["by"] != "user" || rel["document_id"] != nil {
		t.Fatalf("manual release %+v", rel)
	}
	hv = holdView(t, s, res.ProgramState)
	if len(hv) != 1 || hv[0].Released == nil || hv[0].Released.DocumentID != "" || hv[0].Released.DateLabel != "" || len(hv[0].Evidence) != 1 {
		t.Fatalf("manual view %+v", hv)
	}
	if res, err = s.UndoProgram(ctx, orgA, projectA, userA, res.Program.Version); err != nil {
		t.Fatal(err)
	}
	if released(t, res.ProgramState, hold) != nil {
		t.Fatal("undo left the manual release")
	}
}

func isOpErr(err error, field string) bool {
	var oe *store.ProgramOpError
	return errors.As(err, &oe) && oe.Field == field
}
