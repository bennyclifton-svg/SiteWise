package store_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

func walshText(t *testing.T, file string) identity.Text {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "proposals", "testdata", "walsh", file))
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(body), int64(len(body)), proposals.ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func walshDocument(t *testing.T, s *store.Store, id, file, filename string) {
	t.Helper()
	ctx := context.Background()
	if err := s.CreateFile(ctx, orgA, store.File{ID: file, ProjectID: projectA, SHA256: bytes.Repeat([]byte{id[len(id)-1]}, 32), ByteSize: 8, MediaType: "text/markdown"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDocument(ctx, orgA, store.Document{ID: id, ProjectID: projectA, FileID: file, Filename: filename, Status: "filed"}); err != nil {
		t.Fatal(err)
	}
}

// Walsh, 2026-10-27: the engagement letter (3 March) accepts the firm's fee
// proposal (18 February). Whichever is read first, the proposal is
// superseded by the letter, never "unsuccessful", and the letter keeps its
// own date and the PI figures it states.
func TestWalshLetterSupersedesTheProposalItAccepts(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, p)
	if err != nil {
		t.Fatal(err)
	}
	letterDoc, proposalDoc := "11111111-1111-4111-8111-aaaaaaaaab31", "11111111-1111-4111-8111-aaaaaaaaab32"
	walshDocument(t, s, letterDoc, "11111111-1111-4111-8111-aaaaaaaaab41", "01-engagement-letter-atelier-north.md")
	walshDocument(t, s, proposalDoc, "11111111-1111-4111-8111-aaaaaaaaab42", "02-fee-proposal-atelier-north.md")
	letter, err := s.OpenSubmission(ctx, orgA, projectA, letterDoc, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readInto(t, s, letter, walshText(t, "01-engagement-letter-atelier-north.md"), answers{})
	proposal, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readInto(t, s, proposal, walshText(t, "02-fee-proposal-atelier-north.md"), answers{})

	pool := rawPool(t)
	status := func(id string) (st, by, on string) {
		if err := pool.QueryRow(ctx, `SELECT status, COALESCE(superseded_by::text,''), received_on::text FROM submissions WHERE org_id=$1 AND id=$2`, orgA, id).Scan(&st, &by, &on); err != nil {
			t.Fatal(err)
		}
		return
	}
	if st, by, on := status(proposal); st != "superseded" || by != letter || on != "2026-02-18" {
		t.Fatalf("proposal %s by %s on %s; want superseded by the letter, dated 2026-02-18", st, by, on)
	}
	if st, _, on := status(letter); st == "superseded" || on != "2026-03-03" {
		t.Fatalf("letter %s on %s", st, on)
	}
	var limit *int64
	var expires *string
	if err := pool.QueryRow(ctx, `SELECT pi_limit_cents, pi_expires_on::text FROM submissions WHERE org_id=$1 AND id=$2`, orgA, letter).Scan(&limit, &expires); err != nil {
		t.Fatal(err)
	}
	if limit == nil || *limit != 500000000 || expires == nil || *expires != "2026-06-30" {
		t.Fatalf("PI %v %v", limit, expires)
	}
	board, _, err := s.ProcurementBoard(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var card *store.FirmCard
	for i := range board[pkg.ID] {
		if board[pkg.ID][i].SubmissionID == letter {
			card = &board[pkg.ID][i]
		}
	}
	if card == nil || card.PIExpiresOn != "2026-06-30" || !card.PIExpired {
		t.Fatalf("board card %+v", card)
	}

	// Once the letter is awarded, the Profile names the appointed firm.
	if _, err := pool.Exec(ctx, `UPDATE submissions SET status='awarded' WHERE org_id=$1 AND id=$2`, orgA, letter); err != nil {
		t.Fatal(err)
	}
	appointed, err := s.AppointedFirms(ctx, orgA, projectA)
	if err != nil || len(appointed) != 1 {
		t.Fatalf("appointed %+v %v", appointed, err)
	}
	if a := appointed[0]; a.Package != "Architect" || a.Firm != "Atelier North Pty Ltd" || a.Contact != "Lucia Ferraro" || a.PIExpiresOn != "2026-06-30" {
		t.Fatalf("appointed firm %+v", a)
	}
	if other, err := s.AppointedFirms(ctx, orgB, projectA); err != nil || len(other) != 0 {
		t.Fatalf("another org sees the appointment: %+v %v", other, err)
	}
}

// Walsh, 2026-10-27: the builder's preliminary-cost email dropped on the
// head contract row is not a tender, but it names an invited builder. The
// row shows the firm as named; nothing is priced or awarded from it.
func TestWalshBuilderEmailNamesAnInvitedFirm(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, _ := procurement.DisciplinePackage("trade.main_works", "Head contract")
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, p)
	if err != nil {
		t.Fatal(err)
	}
	doc := "11111111-1111-4111-8111-aaaaaaaaab33"
	walshDocument(t, s, doc, "11111111-1111-4111-8111-aaaaaaaaab43", "04-email-builder-preliminary-cost-advice.md")
	id, err := s.OpenSubmission(ctx, orgA, projectA, doc, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readInto(t, s, id, walshText(t, "04-email-builder-preliminary-cost-advice.md"), answers{proposals.FieldKind: proposals.KindNone})
	board, _, err := s.ProcurementBoard(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	cards := board[pkg.ID]
	if len(cards) != 1 || cards[0].Status != store.SubmissionNotProposal || cards[0].FirmName != "Deluca Building Pty Ltd" {
		t.Fatalf("named builder cards %+v", cards)
	}
	if cards[0].BaseTotal != nil || cards[0].StatedTotal != nil {
		t.Fatalf("a named firm carries no price: %+v", cards[0])
	}
}
