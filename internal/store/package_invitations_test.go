package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// boardRow is one package's cards on the Procurement board, as firm name and
// status, in the order Firm 1, 2, 3 shows them.
func boardRow(t *testing.T, s *store.Store, pkg string) []string {
	t.Helper()
	board, _, err := s.ProcurementBoard(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	out := []string{}
	for _, c := range board[pkg] {
		out = append(out, c.FirmName+":"+c.Status)
	}
	return out
}

func invite(t *testing.T, s *store.Store, pkg, name string) store.Invitation {
	t.Helper()
	inv, err := s.InviteFirm(context.Background(), orgA, projectA, pkg, "", name)
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

func plannerRow(t *testing.T, s *store.Store) procurement.Package {
	t.Helper()
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(context.Background(), orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	return pkg
}

func sameRow(got []string, want ...string) bool {
	return strings.Join(got, "|") == strings.Join(want, "|")
}

// The PM names the firms the RFP goes to before any quote arrives. A name
// already in the directory is that firm, and naming it twice is one cell.
func TestInvitedFirmsShowOnTheRowBeforeAnyProposal(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := plannerRow(t, s)
	known, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Beta Planning Pty Ltd", ABN: "88071762537"})
	if err != nil {
		t.Fatal(err)
	}
	alpha := invite(t, s, pkg.ID, "  Alpha Planning ")
	if alpha.FirmName != "Alpha Planning" || alpha.PackageID != pkg.ID || alpha.FirmID == "" {
		t.Fatalf("invitation %+v", alpha)
	}
	if beta := invite(t, s, pkg.ID, "beta planning"); beta.FirmID != known || beta.FirmName != "Beta Planning Pty Ltd" {
		t.Fatalf("typed name did not reuse the directory firm: %+v", beta)
	}
	invite(t, s, pkg.ID, "Alpha Planning")
	if again, err := s.InviteFirm(ctx, orgA, projectA, pkg.ID, alpha.FirmID, ""); err != nil || again.FirmID != alpha.FirmID {
		t.Fatalf("invite by id: %+v %v", again, err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning Pty Ltd:invited") {
		t.Fatalf("board row %v", got)
	}
	board, _, err := s.ProcurementBoard(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	card := board[pkg.ID][0]
	if card.SubmissionID != "" || card.FirmID != alpha.FirmID || card.InvitedOn != store.SydneyDate(time.Now()) || card.BaseTotal != nil || card.Flags == nil {
		t.Fatalf("invited card %+v", card)
	}
	firms, err := s.ReadFirms(ctx, orgA, "planning")
	if err != nil || len(firms) != 2 {
		t.Fatalf("directory has %d planning firms (%v), want 2", len(firms), err)
	}
	// Nothing to price: the price table has no column for an invitation.
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil || len(table.Columns) != 0 {
		t.Fatalf("price table columns %d (%v), want none", len(table.Columns), err)
	}
}

func TestInviteRefusesBadInputOtherOrgsAndRetiredRows(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := plannerRow(t, s)
	for _, c := range []struct{ firm, name string }{{"", ""}, {"", "   "}, {"", strings.Repeat("x", 201)}, {"not-a-uuid", ""}, {newUUID(t), "Both"}} {
		if _, err := s.InviteFirm(ctx, orgA, projectA, pkg.ID, c.firm, c.name); !errors.Is(err, store.ErrInvalidInput) {
			t.Errorf("invite firm %q name %q: %v, want invalid input", c.firm, c.name, err)
		}
	}
	if _, err := s.InviteFirm(ctx, orgA, projectA, pkg.ID, newUUID(t), ""); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("unknown firm: %v", err)
	}
	if _, err := s.InviteFirm(ctx, orgB, projectB, pkg.ID, "", "Intruder"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("another org's package: %v", err)
	}
	if _, err := s.InviteFirm(ctx, orgA, projectA, newUUID(t), "", "Nowhere"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("missing package: %v", err)
	}
	other, err := s.CreateFirm(ctx, orgB, store.FirmFields{Name: "Other Org Firm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.InviteFirm(ctx, orgA, projectA, pkg.ID, other, ""); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("another org's firm: %v", err)
	}
	group, _ := combinedGroup(t, s, "consultant.structural", "consultant.civil")
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, group.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InviteFirm(ctx, orgA, projectA, group.ID, "", "Too Late"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("retired row: %v", err)
	}
	if err := s.RemoveInvitation(ctx, orgA, projectA, pkg.ID, other); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("remove a firm never invited: %v", err)
	}
}

// A quote dropped on the invited firm's cell is that firm's, whatever the
// read finds; it stays in the firm's cell, between the other invitations.
func TestQuoteDroppedOnAnInvitedCellKeepsThatFirmAndCell(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := plannerRow(t, s)
	invite(t, s, pkg.ID, "Alpha Planning")
	beta := invite(t, s, pkg.ID, "Beta Planning")
	invite(t, s, pkg.ID, "Gamma Planning")
	id, err := s.OpenSubmissionFor(ctx, orgA, projectA, docA, pkg.ID, beta.FirmID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning:reading", "Gamma Planning:invited") {
		t.Fatalf("while reading %v", got)
	}
	readInto(t, s, id, pageText([]string{"Printed Planners Pty Ltd"}, []string{"Total fee excluding GST", "$5,000"}), answers{})
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning:received", "Gamma Planning:invited") {
		t.Fatalf("after read %v", got)
	}
	if _, err := s.OpenSubmissionFor(ctx, orgA, projectA, docA, pkg.ID, newUUID(t), time.Now()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown firm on upload: %v", err)
	}
}

// A quote dropped anywhere on the row whose read finds an invited firm
// takes that firm's cell; detached, the cell shows the invitation again.
func TestReadQuoteFillsTheInvitedFirmsCell(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := plannerRow(t, s)
	invite(t, s, pkg.ID, "Alpha Planning")
	beta := invite(t, s, pkg.ID, "Beta Planning")
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning:invited", ":reading") {
		t.Fatalf("while reading %v", got)
	}
	readInto(t, s, id, pageText([]string{"Beta Planning"}, []string{"Total fee excluding GST", "$5,000"}), answers{})
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning:received") {
		t.Fatalf("after read %v", got)
	}
	none := ""
	if err := s.EditSubmission(ctx, orgA, projectA, id, store.SubmissionEdit{PackageID: &none}); err != nil {
		t.Fatal(err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning:invited") {
		t.Fatalf("after detach %v", got)
	}
	if err := s.RemoveInvitation(ctx, orgA, projectA, pkg.ID, beta.FirmID); err != nil {
		t.Fatal(err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited") {
		t.Fatalf("after remove %v", got)
	}
	if err := s.RemoveInvitation(ctx, orgA, projectA, pkg.ID, beta.FirmID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second remove: %v", err)
	}
}

// A firm's revised quote keeps the cell its first quote took.
func TestRevisedQuoteKeepsItsFirmsCell(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := plannerRow(t, s)
	alpha, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Alpha Planning"})
	if err != nil {
		t.Fatal(err)
	}
	beta, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Beta Planning"})
	if err != nil {
		t.Fatal(err)
	}
	for n, f := range []string{alpha, beta, alpha} {
		id, err := s.OpenSubmissionFor(ctx, orgA, projectA, proposalDoc(t, s, byte(n+1), ""), pkg.ID, f, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		readInto(t, s, id, pageText([]string{"Total fee excluding GST", "$5,000"}), answers{})
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:received", "Beta Planning:received") {
		t.Fatalf("revised quote moved its firm: %v", got)
	}
}

// Combined engagements: an invitation made on a member before or after
// combining is the group's; the member row shows none, and it can be
// removed from the group's row.
func TestInvitationsOnCombinedConsultantsAreTheGroups(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.structural", "Structural engineer")
	first, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	row, _ = procurement.DisciplinePackage("consultant.civil", "Civil engineer")
	second, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	before := invite(t, s, first.ID, "Early Engineers")
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{first.ID, second.ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	if after := invite(t, s, second.ID, "Late Engineers"); after.PackageID != group.ID {
		t.Fatalf("invitation on a grouped member stayed on it: %+v", after)
	}
	if got := boardRow(t, s, group.ID); !sameRow(got, "Early Engineers:invited", "Late Engineers:invited") {
		t.Fatalf("group row %v", got)
	}
	if got := boardRow(t, s, first.ID); len(got) != 0 {
		t.Fatalf("grouped member shows %v", got)
	}
	if err := s.RemoveInvitation(ctx, orgA, projectA, group.ID, before.FirmID); err != nil {
		t.Fatal(err)
	}
	if got := boardRow(t, s, group.ID); !sameRow(got, "Late Engineers:invited") {
		t.Fatalf("after remove %v", got)
	}
}

// A misspelt invited firm is corrected in its cell: a name only this cell
// uses is renamed, so the misspelling leaves the directory; a firm with
// details or other uses keeps its name and the cell takes the corrected firm.
func TestInvitedFirmNameIsCorrectedInItsCell(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg := plannerRow(t, s)
	invite(t, s, pkg.ID, "Alpha Planning")
	typo := invite(t, s, pkg.ID, "Smth Planing")
	invite(t, s, pkg.ID, "Gamma Planning")

	fixed, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, typo.FirmID, "", " Smith Planning Pty Ltd ")
	if err != nil || fixed.FirmID != typo.FirmID || fixed.FirmName != "Smith Planning Pty Ltd" {
		t.Fatalf("correct typed name: %+v %v", fixed, err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Smith Planning Pty Ltd:invited", "Gamma Planning:invited") {
		t.Fatalf("board row %v", got)
	}
	if firms, _ := s.ReadFirms(ctx, orgA, "Smth"); len(firms) != 0 {
		t.Fatalf("misspelling still in the directory: %+v", firms)
	}

	// A directory firm chosen instead takes the same cell; the bare name goes.
	known, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Beta Planning Pty Ltd", ABN: "88071762537"})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, typo.FirmID, known, ""); err != nil || got.FirmID != known {
		t.Fatalf("correct to directory firm: %+v %v", got, err)
	}
	if got := boardRow(t, s, pkg.ID); !sameRow(got, "Alpha Planning:invited", "Beta Planning Pty Ltd:invited", "Gamma Planning:invited") {
		t.Fatalf("board row %v", got)
	}
	if firms, _ := s.ReadFirms(ctx, orgA, "Smith Planning"); len(firms) != 0 {
		t.Fatalf("replaced bare name still in the directory: %+v", firms)
	}
	// A firm with details is not renamed: the cell takes a new firm.
	if got, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, known, "", "Beta Planning Group"); err != nil || got.FirmID == known {
		t.Fatalf("correct a detailed firm: %+v %v", got, err)
	}
	if firms, _ := s.ReadFirms(ctx, orgA, "88071762537"); len(firms) != 1 || firms[0].Name != "Beta Planning Pty Ltd" {
		t.Fatalf("detailed firm changed: %+v", firms)
	}

	// A name invited on another row is that row's too: it keeps its name there.
	other, _ := procurement.DisciplinePackage("consultant.heritage", "Heritage")
	pkg2, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, other)
	if err != nil {
		t.Fatal(err)
	}
	shared := invite(t, s, pkg2.ID, "Delta Consult")
	invite(t, s, pkg.ID, "Delta Consult")
	if got, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, shared.FirmID, "", "Delta Heritage"); err != nil || got.FirmID == shared.FirmID {
		t.Fatalf("correct a shared name: %+v %v", got, err)
	}
	if got := boardRow(t, s, pkg2.ID); !sameRow(got, "Delta Consult:invited") {
		t.Fatalf("other row changed: %v", got)
	}

	// The same firm twice on one row is refused; so are bad input and strangers.
	if _, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, shared.FirmID, "", "x"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("firm no longer on the row: %v", err)
	}
	if _, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, typo.FirmID, "", "Alpha Planning"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("replaced firm: %v", err)
	}
	gamma := boardFirm(t, s, pkg.ID, "Gamma Planning")
	if _, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, gamma, "", "alpha planning"); !errors.Is(err, store.ErrConflict) {
		t.Errorf("a firm already on the row: %v", err)
	}
	if _, err := s.CorrectInvitation(ctx, orgA, projectA, pkg.ID, gamma, "", "  "); !errors.Is(err, store.ErrInvalidInput) {
		t.Errorf("blank name: %v", err)
	}
	if _, err := s.CorrectInvitation(ctx, orgB, projectB, pkg.ID, gamma, "", "Intruder"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("another org: %v", err)
	}
	if got := boardRow(t, s, pkg.ID); len(got) != 4 || got[2] != "Gamma Planning:invited" {
		t.Fatalf("board row after refusals %v", got)
	}
}

func boardFirm(t *testing.T, s *store.Store, pkg, name string) string {
	t.Helper()
	board, _, err := s.ProcurementBoard(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range board[pkg] {
		if c.FirmName == name {
			return c.FirmID
		}
	}
	t.Fatalf("%s not on the row", name)
	return ""
}
