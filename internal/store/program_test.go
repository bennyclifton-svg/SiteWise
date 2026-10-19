package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
)

const userA2 = "11111111-1111-4111-8111-aaaaaaaaaab2"

func ip(v int) *int { return &v }

func dur(v int) store.OptInt { return store.OptInt{Set: true, Value: &v} }

// sampleProgram is three stages on a Monday start:
//
//	Design        Concept design 5d → Developed design 10d   [0,15)
//	Approvals     a leaf stage of 20d                         [15,35)
//	Construction  Slab 10d → Framing 15d → Roof 5d            [35,65)
//
// joined by chain links, everything library and proposed.
func sampleProgram() store.ProgramCreate {
	row := func(key, parent, kind, label, catalogue string, d *int) store.ProgramRowInput {
		r := store.ProgramRowInput{Key: key, ParentKey: parent, Kind: kind, Label: label, CatalogueStageID: catalogue, DurationDays: d, ReviewStatus: "proposed", Origin: "library"}
		if d != nil {
			r.DurationOrigin = "library"
		}
		return r
	}
	chain := func(a, b string) store.ProgramLinkInput {
		return store.ProgramLinkInput{PredKey: a, SuccKey: b, Origin: "chain"}
	}
	return store.ProgramCreate{Surface: "program", SkeletonID: "sk.generic", LibraryVersion: "test", StartDate: "2027-03-01", StartOrigin: "user",
		Rows: []store.ProgramRowInput{
			row("design", "", "stage", "Design", "design", nil),
			row("concept", "design", "activity", "Concept design", "", ip(5)),
			row("developed", "design", "activity", "Developed design", "", ip(10)),
			row("approvals", "", "stage", "Approvals", "other", ip(20)),
			row("construction", "", "stage", "Construction", "construction", nil),
			row("slab", "construction", "activity", "Slab", "", ip(10)),
			row("framing", "construction", "activity", "Framing", "", ip(15)),
			row("roof", "construction", "activity", "Roof", "", ip(5)),
		},
		Links: []store.ProgramLinkInput{chain("design", "approvals"), chain("approvals", "construction"), chain("concept", "developed"), chain("slab", "framing"), chain("framing", "roof")},
	}
}

func newProgram(t *testing.T, s *store.Store) store.ProgramState {
	t.Helper()
	ctx := context.Background()
	if _, err := s.CreateProgram(ctx, orgA, projectA, sampleProgram()); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func rowID(t *testing.T, st store.ProgramState, label string) string {
	t.Helper()
	for _, a := range st.Activities {
		if a.Label == label {
			return a.ID
		}
	}
	t.Fatalf("no row %q", label)
	return ""
}

func rowOf(t *testing.T, st store.ProgramState, label string) store.ProgramActivity {
	t.Helper()
	for _, a := range st.Activities {
		if a.Label == label {
			return a
		}
	}
	t.Fatalf("no row %q", label)
	return store.ProgramActivity{}
}

func hasRow(st store.ProgramState, label string) bool {
	for _, a := range st.Activities {
		if a.Label == label {
			return true
		}
	}
	return false
}

func linkBetween(st store.ProgramState, pred, succ string) (store.ProgramLink, bool) {
	for _, l := range st.Links {
		if l.PredID == pred && l.SuccID == succ {
			return l, true
		}
	}
	return store.ProgramLink{}, false
}

func times(t *testing.T, st store.ProgramState, label string) engine.Times {
	t.Helper()
	tm, ok := st.Computed.Activities.Get(rowID(t, st, label))
	if !ok {
		t.Fatalf("no times for %q", label)
	}
	return tm
}

func labels(st store.ProgramState) string {
	var out []string
	for _, a := range st.Activities {
		out = append(out, a.Label)
	}
	return strings.Join(out, ",")
}

func apply(t *testing.T, s *store.Store, st store.ProgramState, ops ...store.ProgramOp) store.ProgramResult {
	t.Helper()
	res, err := s.ApplyProgramOps(context.Background(), orgA, projectA, userA, st.Program.ID, st.Program.Version, ops)
	if err != nil {
		t.Fatalf("ops %+v: %v", ops, err)
	}
	if res.Program.Version != st.Program.Version+1 {
		t.Fatalf("version %d after %d: one bump per batch", res.Program.Version, st.Program.Version)
	}
	return res
}

// refuse runs a batch that must be refused naming field, and checks that
// nothing was written.
func refuse(t *testing.T, s *store.Store, st store.ProgramState, field, message string, ops ...store.ProgramOp) *store.ProgramOpError {
	t.Helper()
	_, err := s.ApplyProgramOps(context.Background(), orgA, projectA, userA, st.Program.ID, st.Program.Version, ops)
	var oe *store.ProgramOpError
	if !errors.As(err, &oe) {
		t.Fatalf("ops %+v: want a refusal, got %v", ops, err)
	}
	if oe.Field != field || !strings.Contains(oe.Message, message) {
		t.Fatalf("ops %+v: refusal %q on %q, want %q on %q", ops, oe.Message, oe.Field, message, field)
	}
	now, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if now.Program.Version != st.Program.Version || labels(now) != labels(st) || len(now.Links) != len(st.Links) {
		t.Fatalf("refused batch wrote: version %d→%d", st.Program.Version, now.Program.Version)
	}
	return oe
}

func TestProgramCreateLoadAndPackageStages(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if _, err := s.LoadProgram(ctx, orgA, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("no program yet: %v", err)
	}
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned", Stages: []procurement.Stage{{StageID: "design", Label: "Design", NovationPhase: "none", Origin: "user"}}})
	if err != nil {
		t.Fatal(err)
	}
	in := sampleProgram()
	in.PackageStages = map[string]string{pkg.Stages[0].ID: "design"}
	created, err := s.CreateProgram(ctx, orgA, projectA, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateProgram(ctx, orgA, projectA, sampleProgram()); !errors.Is(err, store.ErrProgramExists) {
		t.Fatalf("second program: %v", err)
	}
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if got := labels(st); got != "Design,Concept design,Developed design,Approvals,Construction,Slab,Framing,Roof" {
		t.Fatalf("outline order %s", got)
	}
	if st.Program.ID != created.ProgramID || st.Program.StageRevision != 1 || st.Program.Version != 1 || len(st.Links) != 5 {
		t.Fatalf("program %+v links %d", st.Program, len(st.Links))
	}
	if f := st.Computed.Finish; f.TypicalIndex == nil || *f.TypicalIndex != 65 || *f.Typical != "2027-05-28" || !f.Complete {
		t.Fatalf("finish %+v", f)
	}
	if tm := times(t, st, "Approvals"); tm.ES != 15 || tm.EF != 35 {
		t.Fatalf("approvals %+v", tm)
	}
	// The package stage points at its program stage and pins revision 1.
	conn, err := pgx.Connect(ctx, testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var stage string
	var revision int64
	if err := conn.QueryRow(ctx, `SELECT program_stage_id::text,stage_revision FROM package_stages WHERE org_id=$1 AND id=$2`, orgA, pkg.Stages[0].ID).Scan(&stage, &revision); err != nil {
		t.Fatal(err)
	}
	if stage != created.IDs["design"] || revision != 1 {
		t.Fatalf("package stage link %s %d", stage, revision)
	}
	// The package still reads as before.
	if again, err := s.ReadPackages(ctx, orgA, projectA); err != nil || len(again) != 1 || again[0].Stages[0].ID != pkg.Stages[0].ID {
		t.Fatalf("packages after link: %v %+v", err, again)
	}
	// JSON keys follow plan 8.2.
	raw, _ := json.Marshal(st)
	// Deleting the project removes the program, though a package stage
	// points at one of its stages.
	defer func() {
		if _, err := s.DeleteProject(ctx, orgA, projectA); err != nil {
			t.Fatal(err)
		}
		if _, err := s.LoadProgram(ctx, orgA, projectA); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("program survived its project: %v", err)
		}
	}()
	for _, key := range []string{`"program":{"id"`, `"stage_revision":1`, `"working_week":31`, `"duration_days":null`, `"pred_id"`, `"lag_days":0`, `"lag_unit":"working"`, `"calendar":{"days":[]}`, `"computed":{"status"`, `"constraint_date":null`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("response has no %s", key)
		}
	}
}

func TestProgramCreateRefusesBadInput(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	cases := map[string]func(*store.ProgramCreate){
		"stage under a stage":   func(c *store.ProgramCreate) { c.Rows[3].ParentKey = "design" },
		"activity at top level": func(c *store.ProgramCreate) { c.Rows[1].ParentKey = "" },
		"third tier":            func(c *store.ProgramCreate) { c.Rows[2].ParentKey = "concept"; c.Rows[2].Kind = "activity" },
		"unknown link row":      func(c *store.ProgramCreate) { c.Links[0].SuccKey = "nowhere" },
		"loop": func(c *store.ProgramCreate) {
			c.Links = append(c.Links, store.ProgramLinkInput{PredKey: "roof", SuccKey: "slab", Origin: "user"})
		},
		"chain with lag": func(c *store.ProgramCreate) { c.Links[0].LagDays = 2 },
		"milestone with length": func(c *store.ProgramCreate) {
			c.Rows[7].Kind = "milestone"
		},
		"surface": func(c *store.ProgramCreate) { c.Surface = "gantt" },
		"over the link limit": func(c *store.ProgramCreate) {
			c.Links = make([]store.ProgramLinkInput, store.MaxProgramLinks+1)
		},
	}
	for name, edit := range cases {
		in := sampleProgram()
		edit(&in)
		if _, err := s.CreateProgram(ctx, orgA, projectA, in); !errors.Is(err, store.ErrInvalidInput) {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if _, err := s.LoadProgram(ctx, orgA, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("a refused create wrote a program")
	}
	if _, err := s.CreateProgram(ctx, orgA, projectB, sampleProgram()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's project: %v", err)
	}
}

func TestProgramOpValidation(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	slab := rowID(t, st, "Slab")
	tooMany := make([]store.ProgramOp, store.MaxProgramOps+1)
	for i := range tooMany {
		tooMany[i] = store.ProgramOp{Op: "update", ID: slab, Label: sp("Slab")}
	}
	cases := []struct {
		field, message string
		ops            []store.ProgramOp
	}{
		{"ops", "No changes", nil},
		{"ops", store.MsgTooManyOps, tooMany},
		{"op", "Unknown edit", []store.ProgramOp{{Op: "rename"}}},
		{"label", store.MsgLabel, []store.ProgramOp{{Op: "update", ID: slab, Label: sp("   ")}}},
		{"label", store.MsgLabel, []store.ProgramOp{{Op: "update", ID: slab, Label: sp(strings.Repeat("x", 201))}}},
		{"duration_days", store.MsgDuration, []store.ProgramOp{{Op: "update", ID: slab, DurationDays: dur(3651)}}},
		{"duration_days", store.MsgDuration, []store.ProgramOp{{Op: "resize", ID: slab, DurationDays: dur(-1)}}},
		{"duration_days", store.MsgDuration, []store.ProgramOp{{Op: "resize", ID: slab}}},
		{"lag_days", engine.MsgLinkLag, []store.ProgramOp{{Op: "link_add", PredID: slab, SuccID: rowID(t, st, "Roof"), LagDays: ip(366)}}},
		{"lag_days", engine.MsgLinkLag, []store.ProgramOp{{Op: "link_add", PredID: slab, SuccID: rowID(t, st, "Roof"), LagDays: ip(-366)}}},
		{"type", engine.MsgLinkType, []store.ProgramOp{{Op: "link_add", PredID: slab, SuccID: rowID(t, st, "Roof"), Type: sp("XX")}}},
		{"lag_unit", engine.MsgLinkLagUnit, []store.ProgramOp{{Op: "link_add", PredID: slab, SuccID: rowID(t, st, "Roof"), LagUnit: sp("hours")}}},
		{"start_date", store.MsgDate, []store.ProgramOp{{Op: "set_start", StartDate: sp("2027-3-1")}}},
		{"start_date", store.MsgDate, []store.ProgramOp{{Op: "set_start", StartDate: sp("2027-02-30")}}},
		{"day", store.MsgDate, []store.ProgramOp{{Op: "calendar_day", Day: "1 March 2027", Kind: sp("rdo"), Label: sp("RDO")}}},
		{"kind", "Day kind", []store.ProgramOp{{Op: "calendar_day", Day: "2027-03-08", Kind: sp("holiday"), Label: sp("Labour Day")}}},
		{"label", store.MsgLabel, []store.ProgramOp{{Op: "calendar_day", Day: "2027-03-08", Kind: sp("public_holiday")}}},
		{"constraint_date", store.MsgDate, []store.ProgramOp{{Op: "update", ID: slab, ConstraintKind: sp("snet")}}},
		{"constraint_date", "takes no date", []store.ProgramOp{{Op: "update", ID: slab, ConstraintKind: sp("asap"), ConstraintDate: sp("2027-03-01")}}},
		{"constraint_kind", "Constraint", []store.ProgramOp{{Op: "update", ID: slab, ConstraintKind: sp("alap")}}},
		{"id", store.MsgNotInProgram, []store.ProgramOp{{Op: "update", ID: "not-an-id", Label: sp("x")}}},
		{"id", store.MsgNotInProgram, []store.ProgramOp{{Op: "update", ID: "33333333-3333-4333-8333-333333333333", Label: sp("x")}}},
		{"temp_id", "temp_id", []store.ProgramOp{{Op: "add", Label: sp("Stage")}}},
		{"temp_id", "temp_id", []store.ProgramOp{{Op: "add", TempID: "n", Label: sp("A")}, {Op: "add", TempID: "n", Label: sp("B")}}},
		{"kind", "Kind", []store.ProgramOp{{Op: "add", TempID: "n", Label: sp("A"), Kind: sp("task")}}},
		{"parent_id", "under a stage", []store.ProgramOp{{Op: "add", TempID: "n", Label: sp("A"), Kind: sp("activity")}}},
		{"duration_days", "0 days", []store.ProgramOp{{Op: "add", TempID: "n", ParentID: rowID(t, st, "Design"), Label: sp("Gate"), Kind: sp("milestone"), DurationDays: dur(3)}}},
		{"catalogue_stage_id", "Only a stage", []store.ProgramOp{{Op: "update", ID: slab, CatalogueStageID: sp("other")}}},
		{"catalogue_stage_id", "not a catalogue stage", []store.ProgramOp{{Op: "update", ID: rowID(t, st, "Design"), CatalogueStageID: sp("st.made-up")}}},
		{"days", "at most", []store.ProgramOp{{Op: "move", IDs: []string{slab}, Days: ip(4000)}}},
		{"ids", "Choose at least", []store.ProgramOp{{Op: "link_chain", IDs: []string{slab}}}},
		{"id", store.MsgLinkNotFound, []store.ProgramOp{{Op: "link_update", ID: slab, LagDays: ip(1)}}},
	}
	for _, c := range cases {
		refuse(t, s, st, c.field, c.message, c.ops...)
	}
	// A refusal anywhere in the batch writes nothing from earlier ops.
	refuse(t, s, st, "label", store.MsgLabel, store.ProgramOp{Op: "update", ID: slab, Label: sp("Ground slab")}, store.ProgramOp{Op: "update", ID: slab, Label: sp("")})
	// A row of another org's program is not in this program.
	if _, err := s.CreateProgram(context.Background(), orgB, projectB, sampleProgram()); err != nil {
		t.Fatal(err)
	}
	other, err := s.LoadProgram(context.Background(), orgB, projectB)
	if err != nil {
		t.Fatal(err)
	}
	refuse(t, s, st, "id", store.MsgNotInProgram, store.ProgramOp{Op: "update", ID: rowID(t, other, "Slab"), Label: sp("Mine")})
	refuse(t, s, st, "succ_id", store.MsgNotInProgram, store.ProgramOp{Op: "link_add", PredID: slab, SuccID: rowID(t, other, "Roof")})
}

func TestProgramTwoTiers(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	design, slab := rowID(t, st, "Design"), rowID(t, st, "Slab")
	refuse(t, s, st, "ids", store.MsgTwoTiers, store.ProgramOp{Op: "indent", IDs: []string{slab}})
	refuse(t, s, st, "ids", store.MsgTwoTiers, store.ProgramOp{Op: "indent", IDs: []string{rowID(t, st, "Construction")}})
	refuse(t, s, st, "ids", "no stage above", store.ProgramOp{Op: "indent", IDs: []string{design}})
	refuse(t, s, st, "parent_id", store.MsgTwoTiers, store.ProgramOp{Op: "add", TempID: "x", ParentID: slab, Label: sp("Pour")})
	refuse(t, s, st, "parent_id", store.MsgTwoTiers, store.ProgramOp{Op: "add", TempID: "x", ParentID: design, Kind: sp("stage"), Label: sp("Sub-stage")})
	refuse(t, s, st, "ids", store.MsgTwoTiers, store.ProgramOp{Op: "outdent", IDs: []string{design}})
	refuse(t, s, st, "ids", store.MsgTwoTiers, store.ProgramOp{Op: "reorder", IDs: []string{rowID(t, st, "Construction")}, ParentID: design})
	refuse(t, s, st, "parent_id", "Outdent", store.ProgramOp{Op: "reorder", IDs: []string{slab}})

	// A leaf stage indents into the stage above, at its end.
	res := apply(t, s, st, store.ProgramOp{Op: "indent", IDs: []string{rowID(t, st, "Approvals")}})
	approvals := rowOf(t, res.ProgramState, "Approvals")
	if approvals.Kind != "activity" || string(approvals.ParentID) != design || approvals.CatalogueStageID != "" || *approvals.DurationDays != 20 {
		t.Fatalf("indented %+v", approvals)
	}
	if got := labels(res.ProgramState); got != "Design,Concept design,Developed design,Approvals,Construction,Slab,Framing,Roof" {
		t.Fatalf("order %s", got)
	}
	if res.Program.StageRevision != 2 {
		t.Fatalf("a stage removed by indent bumps the revision: %d", res.Program.StageRevision)
	}
	// Chains follow: Design → Construction joins the stages, Developed →
	// Approvals continues Design's activities.
	if _, ok := linkBetween(res.ProgramState, design, rowID(t, res.ProgramState, "Construction")); !ok {
		t.Fatal("stage chain not rejoined")
	}
	if l, ok := linkBetween(res.ProgramState, rowID(t, res.ProgramState, "Developed design"), approvals.ID); !ok || l.Origin != "chain" {
		t.Fatal("activity chain not extended")
	}
	if tm := times(t, res.ProgramState, "Approvals"); tm.ES != 15 || tm.EF != 35 {
		t.Fatalf("approvals moved %+v", tm)
	}
	// And outdents back to a stage directly after its stage.
	res2 := apply(t, s, res.ProgramState, store.ProgramOp{Op: "outdent", IDs: []string{approvals.ID}})
	back := rowOf(t, res2.ProgramState, "Approvals")
	if back.Kind != "stage" || back.ParentID != "" || back.CatalogueStageID != "other" || res2.Program.StageRevision != 3 {
		t.Fatalf("outdented %+v rev %d", back, res2.Program.StageRevision)
	}
	if got := labels(res2.ProgramState); got != "Design,Concept design,Developed design,Approvals,Construction,Slab,Framing,Roof" {
		t.Fatalf("order %s", got)
	}
}

func TestProgramDeleteHealsChains(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	slab, framing, roof := rowID(t, st, "Slab"), rowID(t, st, "Framing"), rowID(t, st, "Roof")
	res := apply(t, s, st, store.ProgramOp{Op: "delete", IDs: []string{framing}})
	if len(res.HealedLinks) != 1 || res.HealedLinks[0].PredID != slab || res.HealedLinks[0].SuccID != roof || res.HealedLinks[0].Origin != "chain" {
		t.Fatalf("healed %+v", res.HealedLinks)
	}
	if hasRow(res.ProgramState, "Framing") || res.Computed.Cycle != nil {
		t.Fatal("framing still live or a loop")
	}
	if tm := times(t, res.ProgramState, "Roof"); tm.ES != 45 {
		t.Fatalf("roof %+v", tm)
	}
	if res.Program.StageRevision != 1 {
		t.Fatal("deleting an activity is not a stage change")
	}
	// Deleting a stage takes its activities and their links, and heals the
	// stage chain around it.
	res2 := apply(t, s, res.ProgramState, store.ProgramOp{Op: "delete", IDs: []string{rowID(t, st, "Approvals")}})
	if l, ok := linkBetween(res2.ProgramState, rowID(t, st, "Design"), rowID(t, st, "Construction")); !ok || l.Origin != "chain" || res2.Program.StageRevision != 2 {
		t.Fatalf("stage chain not healed: %v rev %d", ok, res2.Program.StageRevision)
	}
	res3 := apply(t, s, res2.ProgramState, store.ProgramOp{Op: "delete", IDs: []string{rowID(t, st, "Design")}})
	if hasRow(res3.ProgramState, "Concept design") || hasRow(res3.ProgramState, "Developed design") || len(res3.HealedLinks) != 0 {
		t.Fatalf("stage delete left children or healed into nothing: %s %+v", labels(res3.ProgramState), res3.HealedLinks)
	}
	for _, l := range res3.Links {
		if l.PredID == rowID(t, st, "Concept design") || l.SuccID == rowID(t, st, "Developed design") {
			t.Fatal("a retired row kept a live link")
		}
	}

	// No heal when a link already joins the two rows.
	s2 := profileStore(t)
	st2 := newProgram(t, s2)
	st2 = apply(t, s2, st2, store.ProgramOp{Op: "link_add", PredID: rowID(t, st2, "Slab"), SuccID: rowID(t, st2, "Roof"), Type: sp("SS"), LagDays: ip(3)}).ProgramState
	res4 := apply(t, s2, st2, store.ProgramOp{Op: "delete", IDs: []string{rowID(t, st2, "Framing")}})
	if len(res4.HealedLinks) != 0 {
		t.Fatalf("healed over an existing link %+v", res4.HealedLinks)
	}
	if l, _ := linkBetween(res4.ProgramState, rowID(t, st2, "Slab"), rowID(t, st2, "Roof")); l.Type != "SS" || l.LagDays != 3 {
		t.Fatalf("existing link changed %+v", l)
	}
	// A hold point needs a reason, kept on the retired row.
	hold := apply(t, s2, res4.ProgramState, store.ProgramOp{Op: "add", TempID: "h", ParentID: rowID(t, st2, "Construction"), Kind: sp("hold_point"), Label: sp("Hold point: slab inspection")})
	h := hold.TempIDs["h"]
	refuse(t, s2, hold.ProgramState, "reason", store.MsgHoldPointReason, store.ProgramOp{Op: "delete", IDs: []string{h}})
	apply(t, s2, hold.ProgramState, store.ProgramOp{Op: "delete", IDs: []string{h}, Reason: sp("Certifier waived it")})
	conn, err := pgx.Connect(context.Background(), testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	var reason string
	if err := conn.QueryRow(context.Background(), `SELECT dismissed_reason FROM program_activities WHERE org_id=$1 AND id=$2 AND retired_at IS NOT NULL`, orgA, h).Scan(&reason); err != nil || reason != "Certifier waived it" {
		t.Fatalf("hold point reason %q %v", reason, err)
	}
}

func TestProgramReorderRewritesOnlyChains(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	design, approvals, construction := rowID(t, st, "Design"), rowID(t, st, "Approvals"), rowID(t, st, "Construction")
	developed, slab := rowID(t, st, "Developed design"), rowID(t, st, "Slab")
	st = apply(t, s, st, store.ProgramOp{Op: "link_add", PredID: developed, SuccID: slab}).ProgramState
	user, _ := linkBetween(st, developed, slab)
	designApprovals, _ := linkBetween(st, design, approvals)

	res := apply(t, s, st, store.ProgramOp{Op: "reorder", IDs: []string{construction}})
	if got := labels(res.ProgramState); got != "Construction,Slab,Framing,Roof,Design,Concept design,Developed design,Approvals" {
		t.Fatalf("stages carry their activities: %s", got)
	}
	if res.Program.StageRevision != 2 {
		t.Fatal("a stage reorder bumps the revision")
	}
	if len(res.ChainRemoved) != 1 || res.ChainRemoved[0].PredID != approvals || res.ChainRemoved[0].SuccID != construction {
		t.Fatalf("removed %+v", res.ChainRemoved)
	}
	// Construction → Design would loop through the user's Developed → Slab.
	if len(res.ChainAdded) != 0 {
		t.Fatalf("added %+v", res.ChainAdded)
	}
	if l, ok := linkBetween(res.ProgramState, developed, slab); !ok || l.Version != user.Version || l.Origin != "user" {
		t.Fatal("a user link changed on reorder")
	}
	if l, _ := linkBetween(res.ProgramState, design, approvals); l.Version != designApprovals.Version {
		t.Fatal("a chain between rows still neighbours changed")
	}
	if len(res.BackwardLinks) != 1 || res.BackwardLinks[0].ID != user.ID || !containsText(res.Toasts, "Slab still waits for Developed design.") {
		t.Fatalf("backward %+v toasts %v", res.BackwardLinks, res.Toasts)
	}

	// Inside a stage: Roof before Framing re-sequences Slab → Roof → Framing.
	framing, roof := rowID(t, st, "Framing"), rowID(t, st, "Roof")
	res2 := apply(t, s, res.ProgramState, store.ProgramOp{Op: "reorder", IDs: []string{roof}, ParentID: construction, AfterID: slab})
	if got := labels(res2.ProgramState); !strings.HasPrefix(got, "Construction,Slab,Roof,Framing,") {
		t.Fatalf("order %s", got)
	}
	for _, p := range [][2]string{{slab, roof}, {roof, framing}} {
		if l, ok := linkBetween(res2.ProgramState, p[0], p[1]); !ok || l.Origin != "chain" {
			t.Fatalf("missing chain %v", p)
		}
	}
	if _, ok := linkBetween(res2.ProgramState, framing, roof); ok {
		t.Fatal("old chain kept")
	}
	if res2.Program.StageRevision != 2 || !containsText(res2.Toasts, "Moved Roof after Slab; sequence updated.") {
		t.Fatalf("activity reorder: rev %d toasts %v", res2.Program.StageRevision, res2.Toasts)
	}

	// link_chain skips pairs already linked and pairs that would loop.
	concept := rowID(t, st, "Concept design")
	res3 := apply(t, s, res2.ProgramState, store.ProgramOp{Op: "link_chain", IDs: []string{approvals, concept, framing, roof, slab}})
	if len(res3.Linked) != 1 || res3.Linked[0].PredID != concept || res3.Linked[0].SuccID != approvals || res3.Linked[0].Origin != "user" {
		t.Fatalf("linked %+v", res3.Linked)
	}
	// link_add refuses a loop, naming it.
	oe := refuse(t, s, res3.ProgramState, "succ_id", "would make a loop", store.ProgramOp{Op: "link_add", PredID: framing, SuccID: concept})
	if len(oe.Cycle) < 3 || !strings.Contains(oe.Message, "→") {
		t.Fatalf("cycle %v %q", oe.Cycle, oe.Message)
	}
	refuse(t, s, res3.ProgramState, "succ_id", engine.MsgLinkDuplicate, store.ProgramOp{Op: "link_add", PredID: developed, SuccID: slab})
	refuse(t, s, res3.ProgramState, "succ_id", engine.MsgLinkOwnStage, store.ProgramOp{Op: "link_add", PredID: design, SuccID: concept})
	refuse(t, s, res3.ProgramState, "succ_id", engine.MsgLinkIntoStage, store.ProgramOp{Op: "link_add", PredID: approvals, SuccID: design, Type: sp("FF")})
}

func containsText(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestProgramMove(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	concept, framing, roof := rowID(t, st, "Concept design"), rowID(t, st, "Framing"), rowID(t, st, "Roof")
	// All or none: Framing is held by Slab, so Concept design does not move either.
	oe := refuse(t, s, st, "ids", "Concept design is held by the program start; Framing is held by Slab.", store.ProgramOp{Op: "move", IDs: []string{concept, framing}, Days: ip(-3)})
	if len(oe.Held) != 2 || oe.Held[1] != framing {
		t.Fatalf("held %v", oe.Held)
	}
	refuse(t, s, st, "ids", "Held by Slab", store.ProgramOp{Op: "move", IDs: []string{framing}, Days: ip(-1)})
	// Moving only the free row is fine.
	st = apply(t, s, st, store.ProgramOp{Op: "move", IDs: []string{concept}, Days: ip(0)}).ProgramState
	refuse(t, s, st, "ids", "Held by the program start", store.ProgramOp{Op: "move", IDs: []string{concept}, Days: ip(-1)})

	// Later sets a start-no-earlier constraint on the date moved to.
	res := apply(t, s, st, store.ProgramOp{Op: "move", IDs: []string{roof}, Days: ip(5)})
	r := rowOf(t, res.ProgramState, "Roof")
	if r.ConstraintKind != "snet" || r.ConstraintDate != "2027-05-31" || r.Origin != "user" || r.ReviewStatus != "accepted" {
		t.Fatalf("roof %+v", r)
	}
	if tm := times(t, res.ProgramState, "Roof"); tm.ES != 65 {
		t.Fatalf("roof %+v", tm)
	}
	// Earlier lowers it, then removes it once the link holds the bar.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "move", IDs: []string{roof}, Days: ip(-2)})
	if r = rowOf(t, res.ProgramState, "Roof"); r.ConstraintKind != "snet" || times(t, res.ProgramState, "Roof").ES != 63 {
		t.Fatalf("lowered %+v", r)
	}
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "move", IDs: []string{roof}, Days: ip(-3)})
	if r = rowOf(t, res.ProgramState, "Roof"); r.ConstraintKind != "asap" || r.ConstraintDate != "" || times(t, res.ProgramState, "Roof").ES != 60 {
		t.Fatalf("removed %+v", r)
	}
	// A stage with activities moves them together; links still drive the
	// later ones, so only the first keeps a constraint.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "move", IDs: []string{rowID(t, st, "Construction")}, Days: ip(2)})
	if times(t, res.ProgramState, "Slab").ES != 37 || times(t, res.ProgramState, "Roof").ES != 62 {
		t.Fatal("stage move did not shift its activities")
	}
	if rowOf(t, res.ProgramState, "Slab").ConstraintKind != "snet" || rowOf(t, res.ProgramState, "Framing").ConstraintKind != "asap" {
		t.Fatal("constraints after a stage move")
	}
	if res.Program.StageRevision != 1 {
		t.Fatal("a move is not a stage change")
	}
	// One row to a date.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "move", ID: concept, StartDate: sp("2027-03-06")})
	if tm := times(t, res.ProgramState, "Concept design"); tm.ES != 5 || rowOf(t, res.ProgramState, "Concept design").ConstraintDate != "2027-03-08" {
		t.Fatalf("to a Saturday lands on Monday: %+v", tm)
	}
}

// TestProgramMoveRelative: dragging works without a start date (plan 13.1,
// 13.3 Q2). Constraint dates count from engine.RelativeEpoch, and set_start
// keeps every constraint on the same working-day index; undo restores them.
func TestProgramMoveRelative(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st := newProgram(t, s)
	roof, concept := rowID(t, st, "Roof"), rowID(t, st, "Concept design")
	st = apply(t, s, st, store.ProgramOp{Op: "set_start"}).ProgramState
	if st.Program.StartDate != "" {
		t.Fatal("start not cleared")
	}

	// Later by 5: Roof 60 → 65 = 13 weeks after the epoch Monday.
	res := apply(t, s, st, store.ProgramOp{Op: "move", IDs: []string{roof}, Days: ip(5)},
		// A Saturday in the epoch's first week lands on Monday, index 5.
		store.ProgramOp{Op: "move", ID: concept, StartDate: sp("2001-01-06")})
	if r := rowOf(t, res.ProgramState, "Roof"); r.ConstraintKind != "snet" || r.ConstraintDate != "2001-04-02" || times(t, res.ProgramState, "Roof").ES != 65 {
		t.Fatalf("relative move %+v %+v", r, times(t, res.ProgramState, "Roof"))
	}
	if r := rowOf(t, res.ProgramState, "Concept design"); r.ConstraintDate != "2001-01-08" || times(t, res.ProgramState, "Concept design").ES != 5 {
		t.Fatalf("relative move to a date %+v", r)
	}
	moved := res.ProgramState

	// A holiday is kept but ignored without a start date.
	st = apply(t, s, moved, store.ProgramOp{Op: "calendar_day", Day: "2027-04-14", Kind: sp("public_holiday"), Label: sp("Shutdown")}).ProgramState
	if times(t, st, "Roof").ES != 65 || len(st.Calendar.Days) != 1 {
		t.Fatal("a relative program ignores holidays")
	}

	// Setting a Wednesday start keeps each bar on its index: Concept 5 is
	// Wed 10 Mar; Roof 65 is 13 weeks on, Wed 2 Jun, plus the Wed 14 Apr
	// holiday = Thu 3 Jun.
	st = apply(t, s, st, store.ProgramOp{Op: "set_start", StartDate: sp("2027-03-03")}).ProgramState
	if r := rowOf(t, st, "Roof"); r.ConstraintDate != "2027-06-03" || times(t, st, "Roof").ES != 65 {
		t.Fatalf("roof after set_start %+v %+v", r, times(t, st, "Roof"))
	}
	if r := rowOf(t, st, "Concept design"); r.ConstraintDate != "2027-03-10" || r.Origin != "user" || times(t, st, "Concept design").ES != 5 {
		t.Fatalf("concept after set_start %+v", r)
	}
	if r := rowOf(t, st, "Slab"); r.ConstraintKind != "asap" || r.ConstraintDate != "" {
		t.Fatalf("an unconstrained row gains nothing: %+v", r)
	}

	// Undo puts the program back to relative with the epoch dates.
	u, err := s.UndoProgram(ctx, orgA, projectA, userA, st.Program.Version)
	if err != nil {
		t.Fatal(err)
	}
	st = u.ProgramState
	if st.Program.StartDate != "" || rowOf(t, st, "Roof").ConstraintDate != "2001-04-02" || rowOf(t, st, "Concept design").ConstraintDate != "2001-01-08" || times(t, st, "Roof").ES != 65 {
		t.Fatalf("undo set_start %+v roof %+v", st.Program, rowOf(t, st, "Roof"))
	}

	// A Monday start: index 65 is Mon 31 May plus the holiday = Tue 1 Jun.
	// Clearing the start converts back to the epoch.
	st = apply(t, s, st, store.ProgramOp{Op: "set_start", StartDate: sp("2027-03-01")}).ProgramState
	if rowOf(t, st, "Roof").ConstraintDate != "2027-06-01" || times(t, st, "Roof").ES != 65 {
		t.Fatalf("monday start %+v", rowOf(t, st, "Roof"))
	}
	st = apply(t, s, st, store.ProgramOp{Op: "set_start"}).ProgramState
	if rowOf(t, st, "Roof").ConstraintDate != "2001-04-02" || times(t, st, "Roof").ES != 65 {
		t.Fatalf("cleared start %+v", rowOf(t, st, "Roof"))
	}
}

func TestProgramFirstAndLastChild(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	approvals := rowID(t, st, "Approvals")
	res := apply(t, s, st, store.ProgramOp{Op: "add", TempID: "lodge", ParentID: approvals, Label: sp("Lodgement")})
	lodge := rowOf(t, res.ProgramState, "Lodgement")
	if lodge.DurationDays == nil || *lodge.DurationDays != 20 || rowOf(t, res.ProgramState, "Approvals").DurationDays != nil {
		t.Fatalf("first child %+v", lodge)
	}
	if !containsText(res.Toasts, "Approvals's 20 days moved to Lodgement.") || res.TempIDs["lodge"] != lodge.ID {
		t.Fatalf("toasts %v temp %v", res.Toasts, res.TempIDs)
	}
	if res.Program.StageRevision != 1 {
		t.Fatal("adding an activity is not a stage change")
	}
	// The stage kept its links: Construction still starts at 35.
	if times(t, res.ProgramState, "Slab").ES != 35 {
		t.Fatal("stage links lost")
	}
	refuse(t, s, res.ProgramState, "duration_days", store.MsgStageSpan, store.ProgramOp{Op: "resize", ID: approvals, DurationDays: dur(10)})
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "resize", ID: lodge.ID, DurationDays: dur(25)})
	// Removing the last activity hands the stage its last span.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "delete", IDs: []string{lodge.ID}})
	a := rowOf(t, res.ProgramState, "Approvals")
	if a.DurationDays == nil || *a.DurationDays != 25 || a.DurationOrigin != "calculation" || !containsText(res.Toasts, "Approvals has no activities now; it keeps its span of 25 days.") {
		t.Fatalf("last child %+v %v", a, res.Toasts)
	}
	// A milestone resized above 0 becomes an activity.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "add", TempID: "m", ParentID: rowID(t, st, "Design"), Kind: sp("milestone"), Label: sp("Design freeze")})
	m := res.TempIDs["m"]
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "resize", ID: m, DurationDays: dur(2)})
	if rowOf(t, res.ProgramState, "Design freeze").Kind != "activity" {
		t.Fatal("milestone stayed a milestone")
	}
}

func TestProgramAddUpdateAndTempIDs(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	res := apply(t, s, st,
		store.ProgramOp{Op: "add", TempID: "s", AfterID: rowID(t, st, "Approvals"), Label: sp("Tender")},
		store.ProgramOp{Op: "add", TempID: "a", ParentID: "s", Label: sp("Tender period"), DurationDays: dur(15)},
		store.ProgramOp{Op: "add", TempID: "b", ParentID: "s", Label: sp("Evaluation"), DurationDays: dur(5)},
		store.ProgramOp{Op: "link_add", PredID: "b", SuccID: rowID(t, st, "Slab"), Type: sp("FS"), LagDays: ip(2)},
	)
	if got := labels(res.ProgramState); got != "Design,Concept design,Developed design,Approvals,Tender,Tender period,Evaluation,Construction,Slab,Framing,Roof" {
		t.Fatalf("order %s", got)
	}
	tender := rowOf(t, res.ProgramState, "Tender")
	if tender.CatalogueStageID != "other" || tender.Origin != "user" || tender.ReviewStatus != "accepted" || res.Program.StageRevision != 2 {
		t.Fatalf("new stage %+v rev %d", tender, res.Program.StageRevision)
	}
	// The new stage joined the stage chain, the activities their stage's.
	if _, ok := linkBetween(res.ProgramState, rowID(t, st, "Approvals"), tender.ID); !ok {
		t.Fatal("new stage not chained after Approvals")
	}
	if _, ok := linkBetween(res.ProgramState, tender.ID, rowID(t, st, "Construction")); !ok {
		t.Fatal("new stage not chained before Construction")
	}
	if _, ok := linkBetween(res.ProgramState, rowID(t, st, "Approvals"), rowID(t, st, "Construction")); ok {
		t.Fatal("old stage chain kept")
	}
	if _, ok := linkBetween(res.ProgramState, res.TempIDs["a"], res.TempIDs["b"]); !ok {
		t.Fatal("activities not chained")
	}
	// Tender ends at 35+20=55; Slab waits 2 days after Evaluation.
	if times(t, res.ProgramState, "Slab").ES != 57 {
		t.Fatalf("slab %+v", times(t, res.ProgramState, "Slab"))
	}
	// Editing a library row makes it the user's; renaming an activity, a
	// duration and a constraint are not stage changes.
	res = apply(t, s, res.ProgramState,
		store.ProgramOp{Op: "update", ID: rowID(t, st, "Slab"), Label: sp(" Ground slab "), DurationDays: dur(12)},
		store.ProgramOp{Op: "update", ID: rowID(t, st, "Roof"), ConstraintKind: sp("mso"), ConstraintDate: sp("2027-09-01")},
	)
	slab := rowOf(t, res.ProgramState, "Ground slab")
	if slab.Origin != "user" || slab.ReviewStatus != "accepted" || *slab.DurationDays != 12 || slab.DurationOrigin != "user" || res.Program.StageRevision != 2 {
		t.Fatalf("edited %+v rev %d", slab, res.Program.StageRevision)
	}
	if r := rowOf(t, res.ProgramState, "Roof"); r.ConstraintKind != "mso" || r.ConstraintDate != "2027-09-01" {
		t.Fatalf("constraint %+v", r)
	}
	// duration_days null makes it unknown, and unknown spreads.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "update", ID: slab.ID, DurationDays: store.OptInt{Set: true}})
	if rowOf(t, res.ProgramState, "Ground slab").DurationDays != nil || !times(t, res.ProgramState, "Roof").Unknown || res.Computed.Status.Known != res.Computed.Status.Total-1 {
		t.Fatalf("unknown %+v", res.Computed.Status)
	}
	// Link edits and deletes; an edited chain becomes the user's.
	chain, _ := linkBetween(res.ProgramState, rowID(t, st, "Framing"), rowID(t, st, "Roof"))
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "link_update", ID: chain.ID, Type: sp("SS"), LagDays: ip(-2), LagUnit: sp("calendar")})
	if l, _ := linkBetween(res.ProgramState, rowID(t, st, "Framing"), rowID(t, st, "Roof")); l.Type != "SS" || l.LagDays != -2 || l.LagUnit != "calendar" || l.Origin != "user" || l.Version != 2 {
		t.Fatalf("link %+v", l)
	}
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "link_delete", IDs: []string{chain.ID}})
	if _, ok := linkBetween(res.ProgramState, rowID(t, st, "Framing"), rowID(t, st, "Roof")); ok {
		t.Fatal("link not deleted")
	}
	// Calendar days and the start date.
	res = apply(t, s, res.ProgramState,
		store.ProgramOp{Op: "calendar_day", Day: "2027-03-08", Kind: sp("public_holiday"), Label: sp("Labour Day")},
		store.ProgramOp{Op: "set_start", StartDate: sp("2027-03-02")},
	)
	if len(res.Calendar.Days) != 1 || res.Calendar.Days[0].Origin != "user" || res.Program.StartDate != "2027-03-02" || res.Program.StartOrigin != "user" {
		t.Fatalf("calendar %+v start %q", res.Calendar, res.Program.StartDate)
	}
	if tm := times(t, res.ProgramState, "Concept design"); tm.Start != "2027-03-02" || tm.Finish != "2027-03-09" {
		t.Fatalf("holiday skipped %+v", tm)
	}
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "calendar_day", Day: "2027-03-08"})
	if len(res.Calendar.Days) != 0 || res.Program.StageRevision != 2 {
		t.Fatal("calendar day not removed, or a stage change")
	}
}

func TestProgramStageRevision(t *testing.T) {
	s, _, _ := workStore(t)
	st := newProgram(t, s)
	design := rowID(t, st, "Design")
	steps := []struct {
		name string
		bump bool
		ops  []store.ProgramOp
	}{
		{"add activity", false, []store.ProgramOp{{Op: "add", TempID: "x", ParentID: design, Label: sp("Brief"), DurationDays: dur(2)}}},
		{"rename activity", false, []store.ProgramOp{{Op: "update", ID: rowID(t, st, "Slab"), Label: sp("Slab on ground")}}},
		{"resize", false, []store.ProgramOp{{Op: "resize", ID: rowID(t, st, "Roof"), DurationDays: dur(6)}}},
		{"move", false, []store.ProgramOp{{Op: "move", IDs: []string{rowID(t, st, "Roof")}, Days: ip(1)}}},
		{"link", false, []store.ProgramOp{{Op: "link_add", PredID: rowID(t, st, "Concept design"), SuccID: rowID(t, st, "Slab")}}},
		{"reorder activities", false, []store.ProgramOp{{Op: "reorder", IDs: []string{rowID(t, st, "Developed design")}, ParentID: design}}},
		{"start", false, []store.ProgramOp{{Op: "set_start", StartDate: sp("2027-04-05")}}},
		{"calendar", false, []store.ProgramOp{{Op: "calendar_day", Day: "2027-04-26", Kind: sp("public_holiday"), Label: sp("Anzac Day")}}},
		{"rename stage to the same name", false, []store.ProgramOp{{Op: "update", ID: design, Label: sp("Design")}}},
		{"remap to the same stage", false, []store.ProgramOp{{Op: "update", ID: design, CatalogueStageID: sp("design")}}},
		{"rename stage", true, []store.ProgramOp{{Op: "update", ID: design, Label: sp("Design development")}}},
		{"add stage", true, []store.ProgramOp{{Op: "add", TempID: "st", Label: sp("Handover")}}},
		{"reorder stages", true, []store.ProgramOp{{Op: "reorder", IDs: []string{rowID(t, st, "Approvals")}}}},
		{"remap stage", true, []store.ProgramOp{{Op: "update", ID: rowID(t, st, "Approvals"), CatalogueStageID: sp("design")}}},
		{"delete stage", true, []store.ProgramOp{{Op: "delete", IDs: []string{rowID(t, st, "Approvals")}}}},
	}
	for _, step := range steps {
		before := st.Program.StageRevision
		st = apply(t, s, st, step.ops...).ProgramState
		if got := st.Program.StageRevision - before; (got == 1) != step.bump || got > 1 {
			t.Fatalf("%s: stage revision moved by %d", step.name, got)
		}
		if step.name == "remap stage" && rowOf(t, st, "Approvals").CatalogueStageID != "design" {
			t.Fatal("remap not stored")
		}
	}
	refuse(t, s, st, "catalogue_stage_id", "not a catalogue stage", store.ProgramOp{Op: "update", ID: design, CatalogueStageID: sp("st.nowhere")})
}

func TestProgramUndoRedo(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st := newProgram(t, s)
	slab := rowID(t, st, "Slab")
	if _, err := s.UndoProgram(ctx, orgA, projectA, userA, st.Program.Version); err == nil || err.Error() != store.MsgNothingToUndo {
		t.Fatalf("nothing to undo: %v", err)
	}
	undo := func(st store.ProgramState, actor string) (store.ProgramState, error) {
		res, err := s.UndoProgram(ctx, orgA, projectA, actor, st.Program.Version)
		return res.ProgramState, err
	}
	redo := func(st store.ProgramState, actor string) (store.ProgramState, error) {
		res, err := s.RedoProgram(ctx, orgA, projectA, actor, st.Program.Version)
		return res.ProgramState, err
	}
	// Two renames of one row undo one at a time, back to the start.
	st = apply(t, s, st, store.ProgramOp{Op: "update", ID: slab, Label: sp("Slab A")}).ProgramState
	st = apply(t, s, st, store.ProgramOp{Op: "update", ID: slab, Label: sp("Slab B")}).ProgramState
	var err error
	if st, err = undo(st, userA); err != nil || !hasRow(st, "Slab A") {
		t.Fatalf("first undo %v %s", err, labels(st))
	}
	if st, err = undo(st, userA); err != nil || !hasRow(st, "Slab") {
		t.Fatalf("second undo %v %s", err, labels(st))
	}
	if r := rowOf(t, st, "Slab"); r.Origin != "library" || r.ReviewStatus != "proposed" || r.Version != 1 {
		t.Fatalf("undo restores the row exactly: %+v", r)
	}
	if st, err = redo(st, userA); err != nil || !hasRow(st, "Slab A") {
		t.Fatalf("redo %v %s", err, labels(st))
	}
	if st, err = redo(st, userA); err != nil || !hasRow(st, "Slab B") {
		t.Fatalf("redo again %v %s", err, labels(st))
	}
	if _, err = redo(st, userA); err == nil || err.Error() != store.MsgNothingToRedo {
		t.Fatalf("nothing to redo: %v", err)
	}

	// Another user's edit to the same row since blocks undo; theirs undoes.
	st = apply(t, s, st, store.ProgramOp{Op: "update", ID: slab, Label: sp("Slab C")}).ProgramState
	res, err := s.ApplyProgramOps(ctx, orgA, projectA, userA2, st.Program.ID, st.Program.Version, []store.ProgramOp{{Op: "update", ID: slab, Label: sp("Slab D")}})
	if err != nil {
		t.Fatal(err)
	}
	st = res.ProgramState
	var oe *store.ProgramOpError
	if _, err = undo(st, userA); !errors.As(err, &oe) || oe.Message != store.MsgUndoChanged {
		t.Fatalf("changed since: %v", err)
	}
	if st, err = undo(st, userA2); err != nil || !hasRow(st, "Slab C") {
		t.Fatalf("other user's undo %v", err)
	}
	if st, err = undo(st, userA); err != nil || !hasRow(st, "Slab B") {
		t.Fatalf("undo once the row is back: %v %s", err, labels(st))
	}
	// A new batch ends the redo run.
	st = apply(t, s, st, store.ProgramOp{Op: "update", ID: rowID(t, st, "Roof"), Label: sp("Roofing")}).ProgramState
	if _, err = redo(st, userA); err == nil || err.Error() != store.MsgNothingToRedo {
		t.Fatalf("redo after a new batch: %v", err)
	}

	// Undo restores a deleted stage, its activities and every link, and
	// bumps the stage revision again.
	before := st
	st = apply(t, s, st, store.ProgramOp{Op: "delete", IDs: []string{rowID(t, st, "Construction")}}).ProgramState
	if hasRow(st, "Roofing") {
		t.Fatal("not deleted")
	}
	rev := st.Program.StageRevision
	if st, err = undo(st, userA); err != nil {
		t.Fatal(err)
	}
	if labels(st) != labels(before) || len(st.Links) != len(before.Links) || st.Program.StageRevision != rev+1 {
		t.Fatalf("restored %s links %d rev %d", labels(st), len(st.Links), st.Program.StageRevision)
	}
	for i := range before.Links {
		if l, ok := linkBetween(st, before.Links[i].PredID, before.Links[i].SuccID); !ok || l.ID != before.Links[i].ID {
			t.Fatal("link not restored by id")
		}
	}
	// Undoing an add retires the row; redo brings the same row back.
	res2 := apply(t, s, st, store.ProgramOp{Op: "add", TempID: "n", ParentID: rowID(t, st, "Design"), Label: sp("Survey"), DurationDays: dur(3)})
	if st, err = undo(res2.ProgramState, userA); err != nil || hasRow(st, "Survey") {
		t.Fatalf("undo add %v", err)
	}
	if st, err = redo(st, userA); err != nil || rowID(t, st, "Survey") != res2.TempIDs["n"] {
		t.Fatalf("redo add %v", err)
	}
	// Calendar and start date undo too.
	st = apply(t, s, st, store.ProgramOp{Op: "calendar_day", Day: "2027-03-08", Kind: sp("public_holiday"), Label: sp("Labour Day")}, store.ProgramOp{Op: "set_start", StartDate: sp("2027-03-03")}).ProgramState
	if st, err = undo(st, userA); err != nil || len(st.Calendar.Days) != 0 || st.Program.StartDate != "2027-03-01" {
		t.Fatalf("undo calendar %v %+v", err, st.Program)
	}
	// Undo is version pinned like every write.
	if _, err = s.UndoProgram(ctx, orgA, projectA, userA, st.Program.Version-1); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale undo: %v", err)
	}
}

func TestProgramVersionConflictAndAccept(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st := newProgram(t, s)
	slab := rowID(t, st, "Slab")
	next := apply(t, s, st, store.ProgramOp{Op: "update", ID: slab, Label: sp("Slab A")})
	res, err := s.ApplyProgramOps(ctx, orgA, projectA, userA, st.Program.ID, st.Program.Version, []store.ProgramOp{{Op: "update", ID: slab, Label: sp("Slab B")}})
	if !errors.Is(err, store.ErrVersionConflict) || res.Program.Version != next.Program.Version || !hasRow(res.ProgramState, "Slab A") {
		t.Fatalf("stale version: %v %d", err, res.Program.Version)
	}
	if _, err = s.ApplyProgramOps(ctx, orgA, projectA, userA, "33333333-3333-4333-8333-333333333333", next.Program.Version, []store.ProgramOp{{Op: "set_start"}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong program id: %v", err)
	}
	st = next.ProgramState
	acc, err := s.AcceptProgram(ctx, orgA, projectA, userA, st.Program.Version, []string{rowID(t, st, "Roof")}, false)
	if err != nil || rowOf(t, acc.ProgramState, "Roof").ReviewStatus != "accepted" || rowOf(t, acc.ProgramState, "Framing").ReviewStatus != "proposed" {
		t.Fatalf("accept one: %v", err)
	}
	if _, err = s.AcceptProgram(ctx, orgA, projectA, userA, acc.Program.Version, []string{"33333333-3333-4333-8333-333333333333"}, false); err == nil {
		t.Fatal("accepted an unknown id")
	}
	all, err := s.AcceptProgram(ctx, orgA, projectA, userA, acc.Program.Version, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range all.Activities {
		if a.ReviewStatus != "accepted" {
			t.Fatalf("%s still proposed", a.Label)
		}
	}
	back, err := s.UndoProgram(ctx, orgA, projectA, userA, all.Program.Version)
	if err != nil || rowOf(t, back.ProgramState, "Framing").ReviewStatus != "proposed" || rowOf(t, back.ProgramState, "Roof").ReviewStatus != "accepted" {
		t.Fatalf("undo accept all: %v", err)
	}
}

func TestProgramRowLimit(t *testing.T) {
	s := profileStore(t)
	in := store.ProgramCreate{Surface: "program", LibraryVersion: "test", Rows: []store.ProgramRowInput{{Key: "s", Kind: "stage", Label: "Works", CatalogueStageID: "other", ReviewStatus: "accepted", Origin: "user"}}}
	for i := 1; i < store.MaxProgramRows; i++ {
		in.Rows = append(in.Rows, store.ProgramRowInput{Key: "a" + itoa(i), ParentKey: "s", Kind: "activity", Label: "Task " + itoa(i), ReviewStatus: "accepted", Origin: "user"})
	}
	if _, err := s.CreateProgram(context.Background(), orgA, projectA, in); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil || len(st.Activities) != store.MaxProgramRows {
		t.Fatalf("%v %d", err, len(st.Activities))
	}
	refuse(t, s, st, "ops", store.MsgTooManyRows, store.ProgramOp{Op: "add", TempID: "x", ParentID: st.Activities[0].ID, Label: sp("One more")})
	// At the limit, a delete then an add still fits.
	apply(t, s, st, store.ProgramOp{Op: "delete", IDs: []string{st.Activities[1].ID}}, store.ProgramOp{Op: "add", TempID: "x", ParentID: st.Activities[0].ID, Label: sp("One more")})
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

func TestProgramOrgIsolation(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	st := newProgram(t, s)
	if _, err := s.LoadProgram(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("org B read A's program: %v", err)
	}
	if _, err := s.ApplyProgramOps(ctx, orgB, projectA, userB, st.Program.ID, st.Program.Version, []store.ProgramOp{{Op: "set_start"}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("org B wrote A's program: %v", err)
	}
	if _, err := s.UndoProgram(ctx, orgB, projectA, userB, st.Program.Version); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("org B undid A's program: %v", err)
	}
	if _, err := s.AcceptProgram(ctx, orgB, projectA, userB, st.Program.Version, nil, true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("org B accepted A's program: %v", err)
	}
	if _, err := s.CreateProgram(ctx, orgB, projectA, sampleProgram()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("org B created on A's project: %v", err)
	}
	again, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil || again.Program.Version != st.Program.Version {
		t.Fatalf("A's program changed: %v", err)
	}
	// Deleting the project takes the program, even with a package stage
	// pointing at a program stage.
	if _, err := s.DeleteProject(ctx, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadProgram(ctx, orgA, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("program survived its project: %v", err)
	}
}

func TestProgramReorderAcrossStages(t *testing.T) {
	s := profileStore(t)
	st := newProgram(t, s)
	design, approvals := rowID(t, st, "Design"), rowID(t, st, "Approvals")
	concept, developed := rowID(t, st, "Concept design"), rowID(t, st, "Developed design")
	// Design's two activities move into Approvals: Design keeps its last
	// span, Approvals takes its length from them.
	res := apply(t, s, st, store.ProgramOp{Op: "reorder", IDs: []string{developed, concept}, ParentID: approvals})
	if got := labels(res.ProgramState); got != "Design,Approvals,Concept design,Developed design,Construction,Slab,Framing,Roof" {
		t.Fatalf("order %s", got)
	}
	d, a := rowOf(t, res.ProgramState, "Design"), rowOf(t, res.ProgramState, "Approvals")
	if d.DurationDays == nil || *d.DurationDays != 15 || a.DurationDays != nil || res.Program.StageRevision != 1 {
		t.Fatalf("design %v approvals %v rev %d", d.DurationDays, a.DurationDays, res.Program.StageRevision)
	}
	if !containsText(res.Toasts, "Design has no activities now; it keeps its span of 15 days.") {
		t.Fatalf("toasts %v", res.Toasts)
	}
	if l, ok := linkBetween(res.ProgramState, concept, developed); !ok || l.Origin != "chain" {
		t.Fatal("the moved activities keep their chain")
	}
	if tm := times(t, res.ProgramState, "Concept design"); tm.ES != 15 {
		t.Fatalf("concept %+v", tm)
	}
	// A childless stage dropped into a stage becomes its activity.
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "reorder", IDs: []string{design}, ParentID: approvals, AfterID: developed})
	if r := rowOf(t, res.ProgramState, "Design"); r.Kind != "activity" || string(r.ParentID) != approvals || res.Program.StageRevision != 2 {
		t.Fatalf("demoted %+v rev %d", r, res.Program.StageRevision)
	}
	// A row linked to the stage it would join is refused.
	slab := rowID(t, st, "Slab")
	res = apply(t, s, res.ProgramState, store.ProgramOp{Op: "link_add", PredID: approvals, SuccID: slab})
	refuse(t, s, res.ProgramState, "ids", "remove that link first", store.ProgramOp{Op: "reorder", IDs: []string{slab}, ParentID: approvals})
}
