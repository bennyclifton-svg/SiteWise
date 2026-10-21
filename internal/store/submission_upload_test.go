package store_test

import (
	"context"
	"errors"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"testing"
	"time"
)

func TestAttachProposalKeepsColumnFirmAndUserPrices(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Selected Architects"})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	key := ""
	for _, it := range table.Items {
		if it.Base && it.Price() {
			key = it.Key
			break
		}
	}
	if err = s.SetSubmissionCell(ctx, orgA, projectA, sub, key, store.CellInput{State: "priced", Amount: "1234"}); err != nil {
		t.Fatal(err)
	}
	if err = s.CheckSubmissionUpload(ctx, orgB, projectA, pkg.ID, sub); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org: %v", err)
	}
	if _, err = s.AttachSubmissionDocument(ctx, orgA, projectA, pkg.ID, sub, docB, time.Now()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong source org: %v", err)
	}
	id, err := s.AttachSubmissionDocument(ctx, orgA, projectA, pkg.ID, sub, docA, time.Now())
	if err != nil || id != sub {
		t.Fatalf("attach %s %v", id, err)
	}
	readInto(t, s, sub, pageText([]string{"Other Architects Pty Ltd"}, []string{"Design fee", "$8,000"}), answers{"firm_name": "c1"})
	table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 1 || table.Columns[0].Submission.ID != sub || table.Columns[0].Submission.DocumentID != docA || table.Columns[0].Firm.ID != firm {
		t.Fatalf("lost source/firm: %+v", table.Columns)
	}
	if p := table.Columns[0].Cells[key]; p.Cents == nil || *p.Cents != 123400 || p.DecidedBy != "user" {
		t.Fatalf("lost user price: %+v", p)
	}
	if err = s.CheckSubmissionUpload(ctx, orgA, projectA, pkg.ID, sub); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("would overwrite attached source: %v", err)
	}
}
