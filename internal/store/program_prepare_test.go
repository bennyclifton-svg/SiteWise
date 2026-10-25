package store_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	engine "sitewise/internal/program"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// Test fixture only: the fragment, its duration and its label are invented
// for these tests and are not library content.
const questionLibrarySrc = `[{design: test-fixture-only, anchor: "# Test"}]`

var questionLibrary = map[string]string{
	"fragments.yaml": `version: 1
status: draft
sources: ` + questionLibrarySrc + `
fragments:
  - id: fr.flood-check
    label: Test flood check
    stage: construction
    systems: [structure]
    applies_when: {det: flood_hazard, is: true}
    activities:
      - {key: check, label: Flood check, systems: [structure], duration: du.check}
    status: draft
    sources: ` + questionLibrarySrc + `
`,
	"durations.yaml": `version: 1
status: draft
sources: ` + questionLibrarySrc + `
durations:
  - id: du.check
    version: 1
    basis: fixed
    unit_days: working
    low: "3"
    typical: "5"
    high: "8"
    geography: Test
    limitations: []
    status: reviewed
    sources: ` + questionLibrarySrc + `
`,
}

var (
	questionCatOnce sync.Once
	questionCat     *knowledge.Catalog
	questionCatErr  error
)

// questionCatalog is the repository knowledge with the test program
// library above: one fragment whose predicate reads flood_hazard. An
// unanswered flood site omits it, and the program asks no profile question.
func questionCatalog(t *testing.T) *knowledge.Catalog {
	t.Helper()
	questionCatOnce.Do(func() {
		repo := filepath.Join("..", "..", "knowledge")
		root, err := os.MkdirTemp("", "program-question-")
		if err != nil {
			questionCatErr = err
			return
		}
		defer os.RemoveAll(root)
		err = filepath.WalkDir(repo, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(repo, path)
			if d.IsDir() {
				return os.MkdirAll(filepath.Join(root, rel), 0o755)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(root, rel), body, 0o600)
		})
		for name, body := range questionLibrary {
			if err == nil {
				err = os.WriteFile(filepath.Join(root, "program", name), []byte(body), 0o600)
			}
		}
		if err == nil {
			questionCat, err = knowledge.Load(root)
		}
		questionCatErr = err
	})
	if questionCatErr != nil {
		t.Fatal(questionCatErr)
	}
	return questionCat
}

func setValue(t *testing.T, s *store.Store, part, key string, w store.UserWrite) {
	t.Helper()
	if w.Scope == "" {
		w.Scope = "project"
	}
	if _, err := s.SetUserValue(context.Background(), orgA, projectA, part, userA, key, w); err != nil {
		t.Fatalf("%s: %v", key, err)
	}
}

func TestProgramGenInputReadsProfileWorksAndPackages(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	cat := b.Catalog
	setValue(t, s, part, "hdr.cond.planning", store.UserWrite{Value: strPtr("da")})
	setValue(t, s, part, "hdr.cond.procurement_route", store.UserWrite{Value: strPtr("traditional")})
	setValue(t, s, part, "hdr.work_type", store.UserWrite{Value: strPtr("refurb")})
	setValue(t, s, part, "det.state", store.UserWrite{Value: strPtr("NSW"), Scope: cat.KeyScope("det.state")})
	// An explicit unknown and an assumption are not applied values.
	setValue(t, s, part, "det.flood_hazard", store.UserWrite{State: "unknown", Scope: cat.KeyScope("det.flood_hazard")})
	setValue(t, s, part, "det.sprinklered", store.UserWrite{Value: strPtr("true"), Origin: "assumption", Scope: cat.KeyScope("det.sprinklered")})
	setValue(t, s, part, "fact.consent_date", store.UserWrite{Value: strPtr("12 March 2027"), Scope: cat.KeyScope("fact.consent_date")})
	setValue(t, s, part, "fact.consent_lapse_date", store.UserWrite{Value: strPtr("sometime soon"), Scope: cat.KeyScope("fact.consent_lapse_date")})
	setValue(t, s, part, "plan.target_completion", store.UserWrite{Value: strPtr("2028-06-30"), Scope: cat.KeyScope("plan.target_completion")})
	for _, it := range []works.Item{
		{PartID: part, SystemID: "structure", Action: "remove", Title: "Old frame"},
		{PartID: part, SystemID: "envelope", Action: "retain", Title: "Keep the facade"},
		{PartID: part, SystemID: "hydraulic.gas", Action: "investigate", Title: "Trace the gas line"},
	} {
		if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, it); err != nil {
			t.Fatal(err)
		}
	}
	stage := func(id, label string) procurement.Stage {
		return procurement.Stage{StageID: id, Label: label, NovationPhase: "none", Origin: "user"}
	}
	arch, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned",
		Stages: []procurement.Stage{stage("concept_design", "Concept"), stage("construction", "Site visits")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Cancelled engineer", LifecycleStatus: "cancelled",
		Stages: []procurement.Stage{stage("concept_design", "Concept")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "works", WorksScope: "head_contract", Title: "Builder", LifecycleStatus: "planned"}); err != nil {
		t.Fatal(err)
	}

	gi, err := store.ProgramGenInputForTest(ctx, s, orgA, projectA, cat)
	if err != nil {
		t.Fatal(err)
	}
	if gi.Conditions["planning"] != "da" || gi.Conditions["procurement_route"] != "traditional" {
		t.Fatalf("conditions %v", gi.Conditions)
	}
	if len(gi.WorkTypes) != 1 || gi.WorkTypes[0] != "refurb" {
		t.Fatalf("work types %v", gi.WorkTypes)
	}
	if gi.Values["state"] != "NSW" {
		t.Fatalf("values %v", gi.Values)
	}
	if _, ok := gi.Values["flood_hazard"]; ok {
		t.Fatal("an explicit unknown was read as a value")
	}
	if _, ok := gi.Values["sprinklered"]; ok {
		t.Fatal("an assumption was read as a stated value")
	}
	if gi.Facts.ConsentDate != "2027-03-12" || gi.Facts.ConsentLapseDate != "" {
		t.Fatalf("facts %+v", gi.Facts)
	}
	if gi.Planning["target_completion"] != "2028-06-30" {
		t.Fatalf("planning %v", gi.Planning)
	}
	actions := map[string]string{}
	for _, it := range gi.WorkItems {
		actions[it.System] = it.Action
	}
	if len(gi.WorkItems) != 3 || actions["structure"] != "remove" || actions["envelope"] != "retain" || actions["hydraulic.gas"] != "investigate" {
		t.Fatalf("work items %+v", gi.WorkItems)
	}
	// What the work items state about their systems: a removed frame was on
	// site and is not in the finished building; a retained facade is both.
	if gi.Present["structure"] || !gi.Existing["structure"] || !gi.Present["envelope"] || !gi.Existing["envelope"] {
		t.Fatalf("present %v existing %v", gi.Present, gi.Existing)
	}
	if _, ok := gi.Present["hydraulic.gas"]; ok {
		t.Fatal("an investigation stated the system's presence")
	}
	if len(gi.Packages) != 2 || gi.Packages[0].Title != "Architect" || gi.Packages[1].Title != "Builder" {
		t.Fatalf("packages %+v", gi.Packages)
	}
	if got := gi.Packages[0].Stages; len(got) != 2 || got[0].ID != arch.Stages[0].ID || got[0].CatalogueStageID != "concept_design" || got[1].Label != "Site visits" {
		t.Fatalf("stages %+v", got)
	}
	if len(gi.Packages[1].Stages) != 0 {
		t.Fatalf("works package stages %+v", gi.Packages[1].Stages)
	}

	// Generation over these inputs: the retained facade brings nothing, the
	// investigation brings one activity, and the program has no row for the
	// cancelled package.
	st, created, err := s.PrepareProgram(ctx, orgA, projectA, "program", cat)
	if err != nil || !created {
		t.Fatalf("prepare %v %v", created, err)
	}
	if !hasRow(st, "Trace the gas line") || hasRow(st, "Keep the facade") || hasRow(st, "Cancelled engineer: Concept") {
		t.Fatalf("rows %s", labels(st))
	}
}

func TestPrepareProgramIsIdempotentAndLinksPackageStages(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	if _, _, err := s.PrepareProgram(ctx, orgA, projectA, "nowhere", b.Catalog); err == nil {
		t.Fatal("unknown surface accepted")
	}
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned",
		Stages: []procurement.Stage{{StageID: "concept_design", Label: "Concept", NovationPhase: "none", Origin: "user"},
			{StageID: "construction", Label: "Site visits", NovationPhase: "none", Origin: "user"}}})
	if err != nil {
		t.Fatal(err)
	}
	// Two surfaces prepare at once: one program, created once.
	var wg sync.WaitGroup
	results := make([]store.ProgramState, 2)
	createdBy := make([]bool, 2)
	errs := make([]error, 2)
	for i, surface := range []string{"procurement", "cost_plan"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i], createdBy[i], errs[i] = s.PrepareProgram(ctx, orgA, projectA, surface, b.Catalog)
		}()
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || results[0].Program.ID != results[1].Program.ID || createdBy[0] == createdBy[1] {
		t.Fatalf("concurrent prepare %v %v %v", errs, createdBy, results[0].Program.ID != results[1].Program.ID)
	}
	st := results[0]
	again, created, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog)
	if err != nil || created || again.Program.ID != st.Program.ID || again.Program.Version != st.Program.Version {
		t.Fatalf("second prepare %v %v", created, err)
	}
	// No Profile: sk.generic's stages, all proposed library rows, and no
	// row per package (owner direction 25 October 2026).
	var stages []string
	for _, a := range st.Activities {
		if a.Kind == engine.KindStage {
			stages = append(stages, string(a.CatalogueStageID))
		}
		if a.PackageID != "" || a.ReviewStatus != "proposed" || a.Origin != "library" {
			t.Fatalf("row %+v", a)
		}
	}
	if strings.Join(stages, ",") != strings.Join(b.Catalog.GenericStages(), ",") || st.Program.SkeletonID != "sk.generic" {
		t.Fatalf("stages %v", stages)
	}
	if st.Program.LibraryVersion != b.Catalog.LibraryVersion() || (st.Program.CreatedBySurface != "procurement" && st.Program.CreatedBySurface != "cost_plan") {
		t.Fatalf("program %+v", st.Program)
	}
	if st.Computed.Status.Total == 0 || st.Computed.Status.Known > st.Computed.Status.Total {
		t.Fatalf("status %+v", st.Computed.Status)
	}
	conn, err := pgx.Connect(ctx, testDSN(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, ps := range pkg.Stages {
		var stageID string
		if err := conn.QueryRow(ctx, `SELECT program_stage_id::text FROM package_stages WHERE org_id=$1 AND id=$2`, orgA, ps.ID).Scan(&stageID); err != nil {
			t.Fatal(err)
		}
		// The package stage is placed in the program stage of the same
		// catalogue id; no row is made for it.
		var stage store.ProgramActivity
		for _, a := range st.Activities {
			if a.ID == stageID {
				stage = a
			}
		}
		if stage.Kind != engine.KindStage || string(stage.CatalogueStageID) != ps.StageID {
			t.Fatalf("package stage %s: stage %+v (%s)", ps.Label, stage, stageID)
		}
	}
	// Another org never sees or creates this program.
	if _, _, err := s.PrepareProgram(ctx, orgB, projectA, "program", b.Catalog); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign prepare %v", err)
	}
	view, err := s.ProgramView(ctx, orgA, projectA, st, b.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	// The empty library references no unknown fact: no question.
	if view.Question != nil || view.Skeleton.ID != "sk.generic" || view.Skeleton.Label == "" || view.Skeleton.Reason == "" || view.ChangesPending != 0 || view.ReadingsPending != 0 {
		t.Fatalf("extras %+v", view.ProgramExtras)
	}
}

func TestProgramQuestionAnswerAndNotSure(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	cat := questionCatalog(t)
	if _, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "structure", Action: "new", Title: "Frame"}); err != nil {
		t.Fatal(err)
	}
	st, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", cat)
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.ProgramView(ctx, orgA, projectA, st, cat)
	if err != nil {
		t.Fatal(err)
	}
	if view.Question != nil {
		t.Fatalf("program asked a profile question: %+v", view.Question)
	}
	if st.Program.StartDate == "" || st.Program.StartOrigin != "profile" {
		t.Fatalf("start %q origin %q", st.Program.StartDate, st.Program.StartOrigin)
	}
	for _, a := range st.Activities {
		if strings.Contains(a.Label, "flood") {
			t.Fatalf("unanswered flood hazard added %+v", a)
		}
	}
}
