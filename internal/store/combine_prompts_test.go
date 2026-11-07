package store_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Combined engagements, slice 5: prompts for rows made before groups, and a
// combined proposal on a split project waiting for the PM.

func servicesRows(t *testing.T, s *store.Store, project string, disciplines ...string) []procurement.Package {
	t.Helper()
	labels := map[string]string{procurement.ServicesFiling: "Services Engineer", "consultant.mechanical": "Mechanical", "consultant.electrical": "Electrical", "consultant.architect": "Architect", "consultant.access": "Access"}
	var out []procurement.Package
	for _, d := range disciplines {
		p, err := s.CreateDisciplinePackage(context.Background(), orgA, project, userA, mustRow(t, d, labels[d]))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func prompts(t *testing.T, s *store.Store) []procurement.CombinePrompt {
	t.Helper()
	o, err := s.ReadPackageProposals(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	return o.Prompts
}

func TestServicesCombinePrompt(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if got := prompts(t, s); len(got) != 0 {
		t.Fatalf("no rows, no prompt: %+v", got)
	}
	rows := servicesRows(t, s, projectA, procurement.ServicesFiling, "consultant.mechanical", "consultant.electrical")
	got := prompts(t, s)
	// The combined row first, then the services in Procurement's order.
	if len(got) != 1 || got[0].Key != procurement.PromptServices || !reflect.DeepEqual(got[0].PackageIDs, []string{rows[0].ID, rows[2].ID, rows[1].ID}) {
		t.Fatalf("services prompt: %+v", got)
	}
	// Combine is the PM's: the prompt's members, through the usual route.
	if _, err := s.CombinePackages(ctx, orgA, projectA, userA, got[0].PackageIDs, ""); err != nil {
		t.Fatal(err)
	}
	if got := prompts(t, s); len(got) != 0 {
		t.Fatalf("combined rows are not prompted again: %+v", got)
	}
}

func TestDismissCombinePrompt(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	servicesRows(t, s, projectA, procurement.ServicesFiling, "consultant.mechanical")
	if err := s.DismissCombinePrompt(ctx, orgB, projectA, userB, procurement.PromptServices); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org cannot dismiss: %v", err)
	}
	if err := s.DismissCombinePrompt(ctx, orgA, projectB, userA, procurement.PromptServices); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's project: %v", err)
	}
	if err := s.DismissCombinePrompt(ctx, orgA, projectA, userA, "default:services:Architect"); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("only a combine prompt is dismissed here: %v", err)
	}
	if len(prompts(t, s)) != 1 {
		t.Fatal("refusals leave the prompt")
	}
	for range 2 {
		if err := s.DismissCombinePrompt(ctx, orgA, projectA, userA, procurement.PromptServices); err != nil {
			t.Fatal(err)
		}
	}
	if got := prompts(t, s); len(got) != 0 {
		t.Fatalf("a dismissed prompt is remembered: %+v", got)
	}
	// The dismissal is not a dismissed suggestion: suggestions are unchanged.
	o, err := s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil || len(o.Items) != 2 {
		t.Fatalf("rows kept: %+v %v", o.Items, err)
	}
}

func TestCombinedProposalOnSplitProject(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	rows := servicesRows(t, s, projectA, "consultant.mechanical", "consultant.electrical")
	label := func(string) string { return "Services Engineer" }
	open := func(n byte) string {
		t.Helper()
		doc := proposalDoc(t, s, n, procurement.ServicesFiling)
		id, err := s.OpenSubmission(ctx, orgA, projectA, doc, "", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if _, created, err := s.EnsureProposalPackage(ctx, orgA, projectA, doc, label); err != nil || created {
			t.Fatalf("a split project gets no Services Engineer row: %v %v", created, err)
		}
		in, err := s.SubmissionForReading(ctx, orgA, id, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(in.Context.Packages) != 0 {
			t.Fatalf("no package is guessed for a combined filing: %+v", in.Context.Packages)
		}
		readWithQuote(t, s, id, feeText, answers{"firm_name": "c1"}, false)
		return id
	}
	sub := open(0x41)
	if pkg := submissionPackage(t, s, sub); pkg != "" {
		t.Fatalf("unplaced, not %q", pkg)
	}
	_, loose, err := s.ProcurementBoard(ctx, orgA, projectA)
	if err != nil || len(loose) != 1 || !loose[0].Combined {
		t.Fatalf("the board marks the combined proposal: %+v %v", loose, err)
	}
	ids := []string{rows[0].ID, rows[1].ID}
	groups := func() int {
		t.Helper()
		o, err := s.ReadPackageProposals(ctx, orgA, projectA)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for _, p := range o.Items {
			if p.IsGroup() && p.RetiredAt == nil {
				n++
			}
		}
		return n
	}
	// Refusals combine nothing.
	if _, err := s.CombineForProposal(ctx, orgB, projectA, userB, ids, "", sub); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org: %v", err)
	}
	const otherProject = "33333333-3333-4333-8333-555555555555"
	if err := s.CreateProject(ctx, orgA, otherProject, "Other project"); err != nil {
		t.Fatal(err)
	}
	other := servicesRows(t, s, otherProject, "consultant.mechanical", "consultant.electrical")
	if _, err := s.CombineForProposal(ctx, orgA, otherProject, userA, []string{other[0].ID, other[1].ID}, "", sub); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another project's proposal: %v", err)
	}
	if _, err := s.CombineForProposal(ctx, orgA, projectA, userA, ids, "", "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("not a uuid: %v", err)
	}
	placed := open(0x42)
	if err := s.EditSubmission(ctx, orgA, projectA, placed, store.SubmissionEdit{PackageID: &rows[0].ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CombineForProposal(ctx, orgA, projectA, userA, ids, "", placed); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("an already placed proposal: %v", err)
	}
	if n := groups(); n != 0 {
		t.Fatalf("refused, yet %d group(s) made", n)
	}
	group, err := s.CombineForProposal(ctx, orgA, projectA, userA, ids, "", sub)
	if err != nil || !group.IsGroup() {
		t.Fatalf("combine for this proposal: %+v %v", group, err)
	}
	if pkg := submissionPackage(t, s, sub); pkg != group.ID {
		t.Fatalf("the proposal is the group's: %q", pkg)
	}
	if _, loose, _ := s.ProcurementBoard(ctx, orgA, projectA); len(loose) != 0 {
		t.Fatalf("nothing left unplaced: %+v", loose)
	}
	// The PM's choice stands for the next read.
	in, err := s.SubmissionForReading(ctx, orgA, sub, false)
	if err != nil || in.Context.PackageID != group.ID {
		t.Fatalf("the next read keeps the group: %q %v", in.Context.PackageID, err)
	}
}

// A project with no separate services keeps the Services Engineer row a
// combined proposal makes, as before.
func TestCombinedProposalWithoutSplitServicesMakesItsRow(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	doc := proposalDoc(t, s, 0x43, procurement.ServicesFiling)
	if _, err := s.OpenSubmission(ctx, orgA, projectA, doc, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	id, created, err := s.EnsureProposalPackage(ctx, orgA, projectA, doc, func(string) string { return "Services Engineer" })
	if err != nil || !created || id == "" {
		t.Fatalf("Services Engineer row made: %q %v %v", id, created, err)
	}
}
