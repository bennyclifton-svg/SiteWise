package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
)

type answers map[string]string

func (a answers) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	res := jev.Result{Answers: map[string]jev.Answer{}}
	conf := 0.99
	for f := range c.Questions {
		if v, ok := a[f]; ok {
			res.Answers[f] = jev.Answer{Type: jev.TypeChoice, Choice: v, Confidence: &conf}
		}
	}
	return res, nil
}

func pageText(rows ...[]string) identity.Text {
	t := identity.Text{TextLayer: true, Format: "pdf"}
	y := 800.0
	for _, row := range rows {
		x := 40.0
		for _, cell := range row {
			t.Runs = append(t.Runs, identity.Run{Text: cell, Source: identity.Source{Page: 1, X: x, Y: y, Width: 90, Height: 10}})
			x += 120
		}
		y -= 14
	}
	return t
}

func greenAll() proposals.Thresholds {
	lo, hi := 2, 255
	g := 0.9
	th := proposals.Thresholds{QuestionVersion: proposals.QuestionVersion, Questions: map[string][]proposals.Threshold{}}
	for _, f := range []string{"price", "rate", "criterion", "firm", "kind", "package", "gst_basis", "stated_total"} {
		th.Questions[f] = []proposals.Threshold{{Options: &lo, MaxOptions: &hi, Green: &g, Amber: &g}}
	}
	return th
}

// readInto runs one read the way the server does and commits it.
func readInto(t *testing.T, s *store.Store, id string, text identity.Text, ask proposals.Asker) {
	readWithQuote(t, s, id, text, ask, true)
}

func readWithQuote(t *testing.T, s *store.Store, id string, text identity.Text, ask proposals.Asker, quoted bool) {
	t.Helper()
	ctx := context.Background()
	in, err := s.SubmissionForReading(ctx, orgA, id, false)
	if err != nil {
		t.Fatal(err)
	}
	pc := in.Context
	pc.Lookup = s.FirmLookup(ctx, orgA)
	out, err := proposals.Read(ctx, ask, greenAll(), in.Filename, text, pc, in.Keep)
	if err != nil {
		t.Fatal(err)
	}
	if !quoted {
		// Exercise historical readings stored before source quotes were retained.
		out.Quote = nil
	}
	ok, err := s.CommitSubmissionRead(ctx, orgA, id, out, false, time.Now())
	if err != nil || !ok {
		t.Fatalf("commit %v %v", ok, err)
	}
}

func TestSubmissionFromFilingToPriceTable(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateDecision(ctx, orgA, store.Decision{ID: "11111111-1111-4111-8111-aaaaaaaaaa32", DocumentID: docA, Field: "discipline", Value: "consultant.town_planner", Band: "amber", DecidedBy: "jev"}); err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil || len(table.Items) == 0 {
		t.Fatalf("schedule %+v %v", table.Items, err)
	}
	var stage, second string
	for _, it := range table.Items {
		if it.Kind == "stage" && it.Base {
			if stage == "" {
				stage = it.Key
			} else if second == "" {
				second = it.Key
			}
		}
	}
	id, err := s.OpenSubmission(ctx, orgA, projectA, docA, "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := s.OpenSubmission(ctx, orgA, projectA, docA, "", time.Now()); again != id {
		t.Fatal("one document opened two submissions")
	}
	text := pageText(
		[]string{"Mecone Pty Ltd"},
		[]string{"ABN 88 071 762 537"},
		[]string{"1. FEE PROPOSAL"},
		[]string{"Preliminary planning advice", "$3,000"},
		[]string{"Planning report", "$9,000"},
		[]string{"Total (excl. GST)", "$12,000"},
	)
	readInto(t, s, id, text, answers{"price:" + stage: "a1", "price:" + second: "a2", "firm_name": "c1"})
	table, err = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil || len(table.Columns) != 1 {
		t.Fatalf("columns %+v %v", table.Columns, err)
	}
	col := table.Columns[0]
	if col.Firm == nil || col.Firm.ABN != "88071762537" || col.Firm.Status != "draft" {
		t.Fatalf("firm %+v", col.Firm)
	}
	if c := col.Cells[stage]; c.Cents == nil || *c.Cents != 300000 || c.Band != "green" {
		t.Fatalf("cell %+v", c)
	}
	if col.Submission.Status != store.SubmissionReceived || col.Submission.StatedTotal == nil || *col.Submission.StatedTotal != 1200000 {
		t.Fatalf("submission %+v", col.Submission)
	}

	// A schedule correction is preserved, but does not silently define the
	// appointment scope or add a bundle to its component prices.
	if err := s.SetSubmissionCell(ctx, orgA, projectA, id, stage, store.CellInput{State: "priced", Amount: "3,500"}); err != nil {
		t.Fatal(err)
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if got := table.Columns[0]; got.Totals.BaseTotal != nil || got.Cells[stage].Cents == nil || *got.Cells[stage].Cents != 350000 {
		t.Fatalf("schedule correction altered confirmed comparison: %+v", got)
	}
	if err := s.SetSubmissionCell(ctx, orgA, projectA, id, "comparison:appointment", store.CellInput{State: "priced", Amount: "12500", Note: "Advice and planning report, corrected scope"}); err != nil {
		t.Fatal(err)
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if got := table.Columns[0].Totals.KnownBase; got != 1250000 {
		t.Fatalf("confirmed fee %d", got)
	}
	if err := s.SetSubmissionCell(ctx, orgA, projectA, id, stage, store.CellInput{State: "priced", Amount: "-1"}); !errors.Is(err, store.ErrInvalidInput) {
		t.Fatalf("negative amount: %v", err)
	}

	// A revised proposal from the same firm supersedes the first, which stays.
	if err := s.CreateFile(ctx, orgA, store.File{ID: "11111111-1111-4111-8111-aaaaaaaaaa13", ProjectID: projectA, SHA256: make([]byte, 32), ByteSize: 8, MediaType: "application/pdf"}); err != nil {
		t.Fatal(err)
	}
	doc2 := "11111111-1111-4111-8111-aaaaaaaaaa23"
	if err := s.CreateDocument(ctx, orgA, store.Document{ID: doc2, ProjectID: projectA, FileID: "11111111-1111-4111-8111-aaaaaaaaaa13", Filename: "Fee proposal rev B.pdf", Status: "filed"}); err != nil {
		t.Fatal(err)
	}
	id2, err := s.OpenSubmission(ctx, orgA, projectA, doc2, pkg.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	readInto(t, s, id2, text, answers{"price:" + stage: "a1"})
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if len(table.Columns) != 1 || table.Columns[0].Submission.ID != id2 || len(table.Earlier) != 1 || table.Earlier[0].SupersededBy != id2 {
		t.Fatalf("supersession: columns %d earlier %+v", len(table.Columns), table.Earlier)
	}
	board, loose, err := s.ProcurementBoard(ctx, orgA, projectA)
	if err != nil || len(board[pkg.ID]) != 1 || len(loose) != 0 || board[pkg.ID][0].FirmName == "" {
		t.Fatalf("board %+v loose %+v %v", board, loose, err)
	}

	// The firm is in the org's directory and matched by ABN next time.
	firms, err := s.ReadFirms(ctx, orgA, "88071762537")
	if err != nil || len(firms) != 1 {
		t.Fatalf("directory %+v %v", firms, err)
	}
	name := "Mecone"
	if err := s.EditSubmission(ctx, orgA, projectA, id2, store.SubmissionEdit{Firm: &store.FirmFields{Name: name, ABN: "88 071 762 537"}}); err != nil {
		t.Fatal(err)
	}
	if firms, _ = s.ReadFirms(ctx, orgA, ""); len(firms) != 1 || firms[0].Status != "confirmed" || firms[0].Name != name {
		t.Fatalf("confirmed firm %+v", firms)
	}

	// Another org sees none of it.
	if _, err := s.ReadPriceTable(ctx, orgB, projectA, pkg.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org table: %v", err)
	}
	if err := s.SetSubmissionCell(ctx, orgB, projectA, id2, stage, store.CellInput{State: "not_priced"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org cell: %v", err)
	}
	if firms, _ := s.ReadFirms(ctx, orgB, ""); len(firms) != 0 {
		t.Fatalf("cross-org directory %+v", firms)
	}
	if _, err := s.OpenSubmission(ctx, orgB, projectA, docA, "", time.Now()); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org open: %v", err)
	}

	// Deleting the proposal's document keeps the submission and its prices.
	if _, err := s.DeleteDocuments(ctx, orgA, projectA, []string{doc2}, userA); err != nil {
		t.Fatal(err)
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if len(table.Columns) != 1 || table.Columns[0].Submission.DocumentID != "" || table.Columns[0].Cells[stage].Cents == nil {
		t.Fatalf("after delete %+v", table.Columns)
	}
}
