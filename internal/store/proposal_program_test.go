package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/procurement"
	"sitewise/internal/programread"
	"sitewise/internal/rfp"
	"sitewise/internal/store"
)

type noJev struct{ t *testing.T }

func (n noJev) Ask(context.Context, jev.Call) (jev.Result, error) {
	n.t.Fatal("Jev must not be asked")
	return jev.Result{}, nil
}

// r7Response is the firm's returned response form: the R7 sheet filled in.
func r7Response(rows ...[]string) identity.Text {
	t := identity.Text{Format: "xlsx", TextLayer: true}
	all := append([][]string{rfp.R7Columns}, rows...)
	for i, row := range all {
		for j, cell := range row {
			if cell != "" {
				t.Runs = append(t.Runs, identity.Run{Text: cell, Source: identity.Source{Sheet: "R7 Program", Row: i + 3, Col: j + 1}})
			}
		}
	}
	return t
}

// awardable is an architect package on a program, with a firm's priced
// proposal ready to award.
func awardable(t *testing.T) (s *store.Store, pkg procurement.Package, submission string, lines []store.AwardLine) {
	t.Helper()
	ctx := context.Background()
	s, b, _ := workStore(t)
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned",
		Stages: []procurement.Stage{{StageID: "concept_design", Label: "Concept", NovationPhase: "none", Origin: "user"},
			{StageID: "construction", Label: "Site visits", NovationPhase: "none", Origin: "user"}}})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.PrepareProgram(ctx, orgA, projectA, "program", b.Catalog); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareCostPlan(ctx, orgA, projectA, userA); err != nil {
		t.Fatal(err)
	}
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	hasFee := false
	for _, it := range plan.Items {
		hasFee = hasFee || it.LineKind == "fee" && it.PackageID == pkg.ID
	}
	if !hasFee {
		if _, err := s.CreateCostItem(ctx, orgA, projectA, userA, store.CostItemInput{CostWrite: store.CostWrite{PlanVersionID: plan.ID, Version: plan.Version},
			Content: costs.Content{Label: "Architect", LineKind: "fee", PackageID: pkg.ID, Posting: true, Origin: "user", Meaning: "allowance"}}); err != nil {
			t.Fatal(err)
		}
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, it := range table.Items {
		if it.Kind == "stage" && it.Base {
			keys = append(keys, it.Key)
		}
	}
	if len(keys) != 1 {
		t.Fatalf("stage keys %v", keys)
	}
	submission, err = s.OpenSubmission(ctx, orgA, projectA, docA, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	// Read without a retained quote, so the award may name the stage price
	// directly (the quoted-fee rule is tested in quoted_fees_test.go).
	readWithQuote(t, s, submission, pageText(
		[]string{"Smith Architects Pty Ltd"}, []string{"ABN 88 071 762 537"}, []string{"1. FEE PROPOSAL"},
		[]string{"Lump sum fee", "$5,500"},
	), answers{"price:" + keys[0]: "a1", "firm_name": "c1"}, false)
	return s, pkg, submission, []store.AwardLine{{Label: "Architect", Keys: keys}}
}

func proposalReadings(t *testing.T, s *store.Store) store.ProgramView {
	t.Helper()
	ctx := context.Background()
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	v, err := s.ProgramView(ctx, orgA, projectA, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestProposalProgramReadingsShowOnlyOnceAwarded(t *testing.T) {
	ctx := context.Background()
	s, pkg, id, lines := awardable(t)
	if ok, err := s.QueueProposalProgram(ctx, orgA, id); err != nil || !ok {
		t.Fatalf("queue %v %v", ok, err)
	}
	if refs, err := s.PendingProposalPrograms(ctx); err != nil || len(refs) != 1 || refs[0].SubmissionID != id {
		t.Fatalf("pending %v %v", refs, err)
	}
	// A stage that has activities takes its length from them, so it takes
	// no reading: with the library's activities in both stages nothing is
	// read (owner direction 25 October 2026: packages add no rows).
	if in, err := s.ProposalProgramForReading(ctx, orgA, id); err != nil || in.NotRead == "" || len(in.Stages) != 0 {
		t.Fatalf("stages with activities read %+v %v", in, err)
	}
	// Emptied stages take the consultant's weeks themselves.
	emptyArchitectStages(t, s)
	in, err := s.ProposalProgramForReading(ctx, orgA, id)
	if err != nil {
		t.Fatal(err)
	}
	stage := map[string]programread.Stage{}
	for _, ps := range in.Stages {
		stage[ps.Label] = ps
	}
	// Packages add no rows (owner direction 25 October 2026), so a
	// reading targets the program stage by catalogue id.
	if in.NotRead != "" || in.PerWeek != 5 || len(in.Stages) != 2 || stage["Concept"].TargetActivityID != "" || stage["Concept"].CatalogueStageID != "concept_design" {
		t.Fatalf("input %+v", in)
	}
	// A well-completed R7 sheet: settled by rule, zero Jev questions.
	out, err := programread.ReadStageWeeks(ctx, noJev{t}, programread.Thresholds{QuestionVersion: programread.QuestionVersion}, programread.StageWeeksInput{
		Filename: "response.xlsx", PerWeek: in.PerWeek, Stages: in.Stages, Text: r7Response([]string{"Concept", "3", "", "Survey"}, []string{"Site visits", "40", "", ""})})
	if err != nil || out.Asked || len(out.Readings) != 2 {
		t.Fatalf("read %+v %v", out, err)
	}
	done, ok, err := s.CommitProposalProgram(ctx, orgA, id, in, out, false, time.Now())
	if err != nil || !ok || done.Readings != 2 {
		t.Fatalf("commit %+v %v %v", done, ok, err)
	}
	// Before award, nothing changes in Program.
	v := proposalReadings(t, s)
	if len(v.Readings) != 0 || v.ReadingsPending != 0 {
		t.Fatalf("readings before award %+v", v.Readings)
	}
	if _, err := s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: lines, Total: "5500.00"}); err != nil {
		t.Fatal(err)
	}
	v = proposalReadings(t, s)
	if len(v.Readings) != 2 || v.ReadingsPending != 2 {
		t.Fatalf("readings after award %+v", v.Readings)
	}
	var concept, visitsReading store.ProgramReadingView
	for _, r := range v.Readings {
		switch {
		case strings.Contains(r.Text, "Concept"):
			concept = r
		case strings.Contains(r.Text, "Site visits"):
			visitsReading = r
		}
	}
	if concept.Source != "proposal_r7" || concept.Text != "From the awarded proposal: Concept 3 weeks" || concept.CatalogueStageID != "concept_design" ||
		concept.Field != programread.FieldDuration || concept.Value != "3" || !concept.Applies || concept.DecidedBy != "rule" {
		t.Fatalf("concept %+v", concept)
	}
	// Another org sees nothing.
	if _, err := s.LoadProgram(ctx, orgB, projectA); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign program %v", err)
	}
	// Undoing the award hides them again.
	if err := s.UndoAward(ctx, orgA, projectA, pkg.ID, userA); err != nil {
		t.Fatal(err)
	}
	if v = proposalReadings(t, s); len(v.Readings) != 0 || v.ReadingsPending != 0 {
		t.Fatalf("readings after undo %+v", v.Readings)
	}
	st, _ := s.LoadProgram(ctx, orgA, projectA)
	if _, err := s.AcceptProgram(ctx, orgA, projectA, userA, st.Program.Version, []string{concept.ID}, false); err == nil {
		t.Fatal("a hidden reading was accepted")
	}
	// Awarded again: accept writes the duration in working days; the
	// user's own value on Site visits is never overwritten.
	if _, err := s.Award(ctx, orgA, projectA, pkg.ID, userA, store.AwardInput{SubmissionID: id, Lines: lines, Total: "5500.00"}); err != nil {
		t.Fatal(err)
	}
	visits := string(visitsReading.TargetActivityID)
	user := 7
	res, err := s.ApplyProgramOps(ctx, orgA, projectA, userA, st.Program.ID, st.Program.Version, []store.ProgramOp{{Op: "update", ID: visits, DurationDays: store.OptInt{Set: true, Value: &user}}})
	if err != nil {
		t.Fatal(err)
	}
	res, err = s.AcceptProgram(ctx, orgA, projectA, userA, res.Program.Version, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range res.Activities {
		switch a.ID {
		case string(concept.TargetActivityID):
			if a.DurationDays == nil || *a.DurationDays != 15 || a.DurationOrigin != "document" {
				t.Fatalf("concept %+v", a)
			}
		case visits:
			if *a.DurationDays != 7 || a.DurationOrigin != "user" {
				t.Fatalf("user value overwritten %+v", a)
			}
		}
	}
	if v = proposalReadings(t, s); len(v.Readings) != 1 || string(v.Readings[0].TargetActivityID) != visits {
		t.Fatalf("site visits reading should stay proposed: %+v", v.Readings)
	}
	// A second read never proposes an accepted stage again.
	if ok, _ := s.QueueProposalProgram(ctx, orgA, id); !ok {
		t.Fatal("re-queue")
	}
	if _, ok, err := s.CommitProposalProgram(ctx, orgA, id, in, out, false, time.Now()); err != nil || !ok {
		t.Fatal(err)
	}
	if v = proposalReadings(t, s); len(v.Readings) != 1 {
		t.Fatalf("accepted reading proposed again: %+v", v.Readings)
	}
}

// emptyArchitectStages deletes the activities of the two program stages the
// architect's package stages sit in, so each stage takes a reading itself.
func emptyArchitectStages(t *testing.T, s *store.Store) {
	t.Helper()
	prog, err := s.LoadProgram(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	emptied := map[string]bool{}
	for _, a := range prog.Activities {
		if a.Kind == "stage" && (a.CatalogueStageID == "concept_design" || a.CatalogueStageID == "construction") {
			emptied[a.ID] = true
		}
	}
	var kids []string
	for _, a := range prog.Activities {
		if emptied[string(a.ParentID)] {
			kids = append(kids, a.ID)
		}
	}
	if len(kids) > 0 {
		apply(t, s, prog, store.ProgramOp{Op: "delete", IDs: kids})
	}
}

func TestProposalProgramGreyIsRetried(t *testing.T) {
	ctx := context.Background()
	s, _, id, _ := awardable(t)
	emptyArchitectStages(t, s)
	if ok, err := s.QueueProposalProgram(ctx, orgA, id); err != nil || !ok {
		t.Fatal(err)
	}
	if ok, err := s.QueueProposalProgram(ctx, orgB, id); err != nil || ok {
		t.Fatalf("another org queued %v %v", ok, err)
	}
	if _, err := s.ProposalProgramForReading(ctx, orgB, id); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org read %v", err)
	}
	in, err := s.ProposalProgramForReading(ctx, orgA, id)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	out := programread.StageOutcome{Grey: []string{in.Stages[0].ID}}
	if _, ok, err := s.CommitProposalProgram(ctx, orgA, id, in, out, false, now); err != nil || !ok {
		t.Fatal(err)
	}
	if due, _ := s.DueProposalProgramRechecks(ctx, now); len(due) != 0 {
		t.Fatal("retried at once")
	}
	due, err := s.DueProposalProgramRechecks(ctx, now.Add(31*time.Second))
	if err != nil || len(due) != 1 {
		t.Fatalf("due %v %v", due, err)
	}
	if ok, err := s.ClaimProposalProgramRecheck(ctx, due[0], now.Add(31*time.Second)); err != nil || !ok {
		t.Fatal(err)
	}
	if ok, _ := s.ClaimProposalProgramRecheck(ctx, due[0], now.Add(31*time.Second)); ok {
		t.Fatal("claimed twice")
	}
	if _, ok, err := s.CommitProposalProgram(ctx, orgA, id, in, programread.StageOutcome{}, true, now); err != nil || !ok {
		t.Fatal(err)
	}
}
