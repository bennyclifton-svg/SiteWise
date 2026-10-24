package store_test

import (
	"context"
	"errors"
	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
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

func TestSubmissionRereadKeepsManualCellsAndOtherSourceFromSameFirm(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.architect", "Architect")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	firm, err := s.CreateFirm(ctx, orgA, store.FirmFields{Name: "Example Studio"})
	if err != nil {
		t.Fatal(err)
	}
	older, err := s.AddFirmSubmission(ctx, orgA, projectA, pkg.ID, firm, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readInto(t, s, id, pageText([]string{"Total fee excluding GST", "$5,000"}), answers{})
	if err = s.SetSubmissionCell(ctx, orgA, projectA, id, "comparison:appointment", store.CellInput{State: "priced", Amount: "1234", Note: "User scope"}); err != nil {
		t.Fatal(err)
	}
	in, err := s.SubmissionForReading(ctx, orgA, id, false)
	if err != nil {
		t.Fatal(err)
	}
	pc := in.Context
	pc.Lookup = s.FirmLookup(ctx, orgA)
	out, err := proposals.Read(ctx, answers{}, greenAll(), in.Filename, pageText([]string{"Example Studio"}, []string{"Total fee excluding GST", "$5,000"}), pc, in.Keep)
	if err != nil {
		t.Fatal(err)
	}
	ok, err := s.CommitSubmissionRead(ctx, orgA, id, out, true, time.Now())
	if err != nil || !ok {
		t.Fatalf("reread: %v %v", ok, err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(table.Columns) != 2 {
		t.Fatalf("reread silently superseded another source: %+v", table.Columns)
	}
	for _, col := range table.Columns {
		if col.Submission.ID == older && col.Submission.Status != store.SubmissionReceived {
			t.Fatal("other source changed")
		}
		if col.Submission.ID == id {
			cell := col.Cells["comparison:appointment"]
			if cell.Cents == nil || *cell.Cents != 123400 || cell.DecidedBy != "user" || col.Submission.PackageID != pkg.ID {
				t.Fatal("manual values changed")
			}
		}
	}
}
