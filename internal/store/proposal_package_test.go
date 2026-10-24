package store_test

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

func kavanaghMarkdown(t *testing.T, file string) identity.Text {
	t.Helper()
	body, err := os.ReadFile("../proposals/testdata/kavanagh/" + file)
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(body), int64(len(body)), proposals.ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

var disciplineLabels = map[string]string{"consultant.quantity_surveyor": "Quantity Surveyor", "consultant.architect": "Architect"}

func label(id string) string { return disciplineLabels[id] }

// Kavanagh, 24 October 2026: proposals filed before the project had any
// packages stayed "needs a package". A filed consultant proposal now makes
// its consultancy's row and lands under it with firm, fee and date.
func TestProposalFiledBeforePackagesLandsUnderItsConsultancy(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if err := s.CreateDecision(ctx, orgA, store.Decision{ID: "11111111-1111-4111-8111-aaaaaaaaaa41", DocumentID: docA, Field: "discipline", Value: "consultant.quantity_surveyor", Band: "amber", DecidedBy: "jev"}); err != nil {
		t.Fatal(err)
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	pkg, created, err := s.EnsureProposalPackage(ctx, orgA, projectA, docA, label)
	if err != nil || !created || pkg == "" {
		t.Fatalf("package %q created=%v err=%v", pkg, created, err)
	}
	if again, created, err := s.EnsureProposalPackage(ctx, orgA, projectA, docA, label); err != nil || created || again != "" {
		t.Fatalf("second ensure made another package: %q %v %v", again, created, err)
	}
	readInto(t, s, id, kavanaghMarkdown(t, "04-fee-proposal-vertex-cost-advisory.md"), answers{})

	board, loose, err := s.ProcurementBoard(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	if len(loose) != 0 || len(board[pkg]) != 1 {
		t.Fatalf("board %+v unassigned %+v", board, loose)
	}
	card := board[pkg][0]
	if card.FirmName != "Vertex Cost Advisory Pty Ltd" || card.StatedTotal == nil || *card.StatedTotal != 4500000 || card.ReceivedOn != "2026-01-24" {
		t.Fatalf("card %+v stated %v", card, card.StatedTotal)
	}
	pkgs, err := s.ReadPackages(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range pkgs {
		if p.ID == pkg && (p.Origin != "document" || p.Kind != "services" || p.DisciplineID != "consultant.quantity_surveyor" || len(p.Stages) == 0) {
			t.Fatalf("package %+v", p)
		}
	}
}

// The PM's removal of a consultancy stands: a later proposal for it waits
// for a package choice instead of bringing the row back.
func TestProposalDoesNotRecreateARemovedConsultancy(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	p, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	yes := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, p.ID, userA, procurement.PackagePatch{Retired: &yes, Version: p.Version}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDecision(ctx, orgA, store.Decision{ID: "11111111-1111-4111-8111-aaaaaaaaaa42", DocumentID: docA, Field: "discipline", Value: "consultant.architect", Band: "amber", DecidedBy: "jev"}); err != nil {
		t.Fatal(err)
	}
	if id, created, err := s.EnsureProposalPackage(ctx, orgA, projectA, docA, label); err != nil || created || id != "" {
		t.Fatalf("removed consultancy recreated: %q %v %v", id, created, err)
	}
}

// No discipline, no row: the proposal stays for the PM to place.
func TestProposalWithoutDisciplineMakesNoPackage(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if id, created, err := s.EnsureProposalPackage(ctx, orgA, projectA, docA, label); err != nil || created || id != "" {
		t.Fatalf("package without a discipline: %q %v %v", id, created, err)
	}
}
