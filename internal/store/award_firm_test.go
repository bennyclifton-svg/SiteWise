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

// A proposal whose firm was never read cannot be awarded, and the refusal
// says why. Naming the firm on the proposal then lets the same award go
// through: the Chen certifier letter, 25 October 2026.
func TestAwardNeedsTheFirmAndSaysSo(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.certifier", "Certifier")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Placeholder"})
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	none := ""
	if err := s.EditSubmission(ctx, orgA, projectA, id, store.SubmissionEdit{FirmID: &none}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetSubmissionCell(ctx, orgA, projectA, id, "comparison:appointment", store.CellInput{State: "priced", Amount: "6800", Note: "Fee: owner direct, $6,800 + GST"}); err != nil {
		t.Fatal(err)
	}
	in := store.AwardInput{SubmissionID: id, Lines: []store.AwardLine{{Label: "Professional fee", Keys: []string{"comparison:appointment"}}}}
	_, err = s.PreviewAward(ctx, orgA, projectA, pkg.ID, in)
	if !errors.Is(err, store.ErrInvalidInput) || !strings.Contains(err.Error(), "firm") {
		t.Fatalf("award with no firm: %v, want an invalid-input refusal naming the firm", err)
	}
	if err := s.EditSubmission(ctx, orgA, projectA, id, store.SubmissionEdit{Firm: &store.FirmFields{Name: "Northgate Certifiers Pty Ltd"}}); err != nil {
		t.Fatal(err)
	}
	view, err := s.PreviewAward(ctx, orgA, projectA, pkg.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if view.Total != 680000 {
		t.Fatalf("award total %d, want 680000 cents", view.Total)
	}
}
