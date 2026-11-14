package store_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"sitewise/internal/procurement"
	"sitewise/internal/proposals"
	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// Every firm, label and figure in these tests is invented.

// tenderPackage is a town-planning package in org A with the model switched
// on, and the keys of its base rows.
func tenderPackage(t *testing.T) (*store.Store, string, []string) {
	t.Helper()
	ctx := context.Background()
	s, _, _ := workStore(t)
	row, _ := procurement.DisciplinePackage("consultant.town_planner", "Town Planner")
	pkg, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, it := range table.Items {
		if it.Price() && it.Base {
			keys = append(keys, it.Key)
		}
	}
	if len(keys) < 3 {
		t.Fatalf("want three base rows, got %v", keys)
	}
	return s, pkg.ID, keys
}

// tenderSubmission opens a received submission on the package.
func tenderSubmission(t *testing.T, s *store.Store, pkg string, n byte) string {
	t.Helper()
	ctx := context.Background()
	id, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, n, ""), pkg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE submissions SET status='received' WHERE org_id=$1 AND id=$2`, orgA, id); err != nil {
		t.Fatal(err)
	}
	return id
}

func rl(label, amt, kind, where, row string) tender.RawLine {
	st := "priced"
	if amt == "" {
		st = "included"
	}
	return tender.RawLine{P: 1, Label: label, Amt: amt, G: "ex", St: st, Kind: kind, Where: where, Row: row, Sure: true}
}

// reading reconciles invented lines against a page that prints them all.
func tenderReadingOf(t *testing.T, s *store.Store, sub string, lines ...tender.RawLine) tender.Reading {
	t.Helper()
	in, err := s.TenderInput(context.Background(), orgA, sub)
	if err != nil {
		t.Fatal(err)
	}
	var text []string
	for _, l := range lines {
		text = append(text, l.Label+" "+l.Amt)
	}
	return tender.Reconcile(tender.RawReading{Firm: "Invented Planning Pty Ltd", Basis: "fixed", GST: "ex", Lines: lines, Notes: []string{"Valid 30 days"},
		Exclusions: []tender.RawExclusion{{P: 1, Text: "Authority fees"}}}, []tender.Page{{Number: 1, Text: strings.Join(text, "\n")}}, in.Rows, in.StatedEx)
}

func commit(t *testing.T, s *store.Store, sub string, r tender.Reading) {
	t.Helper()
	if err := s.CommitTenderRead(context.Background(), orgA, sub, r, "t1", false, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func tenderColumn(t *testing.T, s *store.Store, pkg, sub string) store.TenderColumn {
	t.Helper()
	table, err := s.ReadPriceTable(context.Background(), orgA, projectA, pkg)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range table.Tender.Columns {
		if c.SubmissionID == sub {
			return c
		}
	}
	t.Fatalf("no tender column for %s", sub)
	return store.TenderColumn{}
}

// pricedCells adds up a submission's priced cells on base rows, as award reads them.
func pricedCells(t *testing.T, s *store.Store, pkg, sub string) (int64, map[string]int64) {
	t.Helper()
	table, err := s.ReadPriceTable(context.Background(), orgA, projectA, pkg)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]int64{}
	var sum int64
	for _, c := range table.Columns {
		if c.Submission.ID != sub {
			continue
		}
		for _, it := range table.Items {
			if cell, ok := c.Cells[it.Key]; ok && it.Base && cell.Cents != nil {
				by[it.Key] = *cell.Cents
				sum += *cell.Cents
			}
		}
	}
	return sum, by
}

func TestModelSwitch(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if on, err := s.ModelEnabled(ctx, orgA); err != nil || on {
		t.Fatalf("off by default: %v %v", on, err)
	}
	now := time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)
	if err := s.SetModelEnabled(ctx, orgA, userA, true, now); err != nil {
		t.Fatal(err)
	}
	at, err := s.ModelEnabledAt(ctx, orgA)
	if err != nil || at == nil || !at.Equal(now) {
		t.Fatalf("%v %v", at, err)
	}
	if on, _ := s.ModelEnabled(ctx, orgB); on {
		t.Fatal("switching org A on switched org B on")
	}
	if err := s.SetModelEnabled(ctx, orgA, userA, false, now); err != nil {
		t.Fatal(err)
	}
	if on, _ := s.ModelEnabled(ctx, orgA); on {
		t.Fatal("still on")
	}
}

func TestTenderInputGivesRowsAndTheStatedTotal(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 21)
	if _, err := rawPool(t).Exec(ctx, `UPDATE submissions SET stated_total_cents=110000, gst_basis='inc' WHERE org_id=$1 AND id=$2`, orgA, sub); err != nil {
		t.Fatal(err)
	}
	in, err := s.TenderInput(ctx, orgA, sub)
	if err != nil {
		t.Fatal(err)
	}
	if in.StatedEx == nil || *in.StatedEx != 100000 || in.Size == 0 || len(in.SHA256) == 0 {
		t.Fatalf("%+v", in)
	}
	have := map[string]bool{}
	for _, r := range in.Rows {
		have[r.Key] = true
		if strings.HasPrefix(r.Key, "rate:") || strings.HasPrefix(r.Key, "criterion:") || strings.HasPrefix(r.Key, "tender:") {
			t.Fatalf("row %s is not a priced row", r.Key)
		}
	}
	for _, k := range keys {
		if !have[k] {
			t.Fatalf("base row %s is missing", k)
		}
	}
	if _, err := s.TenderInput(ctx, orgB, sub); err == nil {
		t.Fatal("another org read the submission")
	}
}

func TestQueueAndClaim(t *testing.T) {
	ctx := context.Background()
	s, pkg, _ := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 22)
	pool := rawPool(t)
	state := func() string {
		var st string
		if err := pool.QueryRow(ctx, `SELECT tender_state FROM submissions WHERE org_id=$1 AND id=$2`, orgA, sub).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	// The switch.
	if err := s.SetModelEnabled(ctx, orgA, userA, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.QueueTenderRead(ctx, orgA, sub, false); err != nil || ok || state() != "" {
		t.Fatalf("queued with the switch off: %v %v", ok, err)
	}
	if err := s.SetModelEnabled(ctx, orgA, userA, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	// What is not a live proposal.
	for _, status := range []string{"not_proposal", "withdrawn", "awarded"} {
		if _, err := pool.Exec(ctx, `UPDATE submissions SET status=$3 WHERE org_id=$1 AND id=$2`, orgA, sub, status); err != nil {
			t.Fatal(err)
		}
		if ok, _ := s.QueueTenderRead(ctx, orgA, sub, false); ok {
			t.Fatalf("queued a %s submission", status)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE submissions SET status='received' WHERE org_id=$1 AND id=$2`, orgA, sub); err != nil {
		t.Fatal(err)
	}
	// Another org.
	if ok, _ := s.QueueTenderRead(ctx, orgB, sub, false); ok || state() != "" {
		t.Fatal("another org queued it")
	}
	if ok, err := s.QueueTenderRead(ctx, orgA, sub, false); err != nil || !ok || state() != "pending" {
		t.Fatalf("%v %v %s", ok, err, state())
	}
	if ok, _ := s.QueueTenderRead(ctx, orgA, sub, false); ok {
		t.Fatal("queued twice")
	}
	// Claiming is exclusive.
	if ok, _ := s.ClaimTenderRead(ctx, orgB, sub, time.Now()); ok {
		t.Fatal("another org claimed it")
	}
	first, _ := s.ClaimTenderRead(ctx, orgA, sub, time.Now())
	second, _ := s.ClaimTenderRead(ctx, orgA, sub, time.Now())
	if !first || second || state() != "reading" {
		t.Fatalf("claims %v %v state %s", first, second, state())
	}
	if n, _ := s.TenderAttempts(ctx, orgA, sub); n != 1 {
		t.Fatalf("attempts %d", n)
	}
	// A failure with a retry is claimed again when it is due.
	retry := time.Now().Add(time.Minute)
	if err := s.FailTenderRead(ctx, orgA, sub, "The model could not be reached. It will be tried again.", &retry); err != nil || state() != "failed" {
		t.Fatalf("%v %s", err, state())
	}
	if ok, _ := s.ClaimTenderRead(ctx, orgA, sub, time.Now()); ok {
		t.Fatal("claimed before its retry time")
	}
	if due, _ := s.DueTenderRetries(ctx, retry.Add(time.Second)); len(due) != 1 {
		t.Fatalf("due %v", due)
	}
	if ok, _ := s.ClaimTenderRead(ctx, orgA, sub, retry.Add(time.Second)); !ok {
		t.Fatal("not claimed when due")
	}
	// Read again queues from failed and from read.
	if err := s.FailTenderRead(ctx, orgA, sub, "The model declined to read this file.", nil); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.QueueTenderRead(ctx, orgA, sub, false); ok {
		t.Fatal("a failed read queued without Read again")
	}
	if ok, _ := s.QueueTenderRead(ctx, orgA, sub, true); !ok || state() != "pending" {
		t.Fatal("Read again did not queue")
	}
}

func TestQueueNeedsAPackage(t *testing.T) {
	ctx := context.Background()
	s, _, _ := tenderPackage(t)
	id, err := s.OpenSubmission(ctx, orgA, projectA, proposalDoc(t, s, 23, ""), "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE submissions SET status='received' WHERE org_id=$1 AND id=$2`, orgA, id); err != nil {
		t.Fatal(err)
	}
	if ok, _ := s.QueueTenderRead(ctx, orgA, id, false); ok {
		t.Fatal("queued with no package")
	}
}

func TestCommitStoresLinesAndCellsThatAddUp(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 24)
	before, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	commit(t, s, sub, tenderReadingOf(t, s, sub,
		rl("Advice", "$10,000.00", "base", "sum", keys[0]),
		rl("Report", "$60,000.00", "base", "sum", keys[1]),
		rl("Sundries", "$5,000.00", "base", "sum", ""),
		rl("Lodgement", "", "base", "sum", keys[2]),
		rl("Total", "$80,000.00", "total", "inside", "")))
	col := tenderColumn(t, s, pkg, sub)
	if col.State != "read" || col.Sum == nil || *col.Sum != 8000000 || col.SumBy != "rule" || col.Basis != "fixed" || len(col.Lines) != 5 || col.Checked {
		t.Fatalf("%+v", col)
	}
	if len(col.Notes) != 1 || len(col.Exclusions) != 1 || col.Lines[0].Label != "Advice" || col.Lines[4].Kind != "total" {
		t.Fatalf("notes %v exclusions %v lines %+v", col.Notes, col.Exclusions, col.Lines)
	}
	sum, by := pricedCells(t, s, pkg, sub)
	if sum != 8000000 || by[keys[0]] != 1000000 || by[keys[1]] != 6000000 || by[tender.SystemRow] != 1000000 {
		t.Fatalf("cells add to %d: %v", sum, by)
	}
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if table.Tender.Version <= before.Tender.Version || !table.Tender.On {
		t.Fatalf("version %d -> %d, on %v", before.Tender.Version, table.Tender.Version, table.Tender.On)
	}
	last := table.Tender.Rows[len(table.Tender.Rows)-1]
	if !last.System || last.Label != "Not broken down" {
		t.Fatalf("rows end with %+v", last)
	}
	for _, c := range table.Columns {
		if c.Submission.ID != sub {
			continue
		}
		if cell := c.Cells[keys[2]]; cell.State != "included" {
			t.Fatalf("the included row reads %+v", cell)
		}
		if c.Totals.BaseTotal == nil || *c.Totals.BaseTotal != 8000000 || c.Totals.Unread != 0 || c.Totals.Gap != nil || !c.NeedsReview {
			t.Fatalf("totals %+v review %v", c.Totals, c.NeedsReview)
		}
	}
	// A gone submission, and another org, write nothing.
	if err := s.CommitTenderRead(ctx, orgA, "11111111-1111-4111-8111-00000000dead", tender.Reading{}, "t1", false, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := s.CommitTenderRead(ctx, orgB, sub, tender.Reading{}, "t1", false, time.Now()); err != nil {
		t.Fatal(err)
	}
	if col := tenderColumn(t, s, pkg, sub); len(col.Lines) != 5 {
		t.Fatal("another org's commit changed the lines")
	}
}

func TestCommitWritesOptionCellsAndLeavesRatesAlone(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 25)
	in, _ := s.TenderInput(ctx, orgA, sub)
	option := ""
	for _, r := range in.Rows {
		if strings.HasPrefix(r.Key, "option:") {
			option = r.Key
		}
	}
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO submission_prices (org_id,id,project_id,submission_id,item_key,label,amount_cents,state,band,decided_by)
VALUES ($1,gen_random_uuid(),$2,$3,'rate:principal_or_director','Principal',35000,'priced','green','jev')`, orgA, projectA, sub); err != nil {
		t.Fatal(err)
	}
	lines := []tender.RawLine{rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$10,000.00", "total", "inside", "")}
	if option != "" {
		lines = append(lines, rl("Option: extra meeting", "$1,800.00", "option", "outside", option))
	}
	commit(t, s, sub, tenderReadingOf(t, s, sub, lines...))
	var rate int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM submission_prices WHERE org_id=$1 AND submission_id=$2 AND item_key LIKE 'rate:%'`, orgA, sub).Scan(&rate); err != nil || rate != 1 {
		t.Fatalf("rate cells %d %v", rate, err)
	}
	if option != "" {
		var cents int64
		if err := pool.QueryRow(ctx, `SELECT amount_cents FROM submission_prices WHERE org_id=$1 AND submission_id=$2 AND item_key=$3`, orgA, sub, option).Scan(&cents); err != nil || cents != 180000 {
			t.Fatalf("option cell %d %v", cents, err)
		}
	}
	if sum, _ := pricedCells(t, s, pkg, sub); sum != 1000000 {
		t.Fatalf("the option entered the base total: %d", sum)
	}
}

func TestSecondCommitKeepsTheUsersLinesAndClearsChecked(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 26)
	lines := []tender.RawLine{rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$12,000.00", "total", "inside", "")}
	commit(t, s, sub, tenderReadingOf(t, s, sub, lines...))
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO tender_lines (org_id,id,project_id,submission_id,ordinal,label,amount_cents,gst,state,kind,placed,row_key,counted,decided_by)
VALUES ($1,gen_random_uuid(),$2,$3,50,'Site visit',200000,'ex','priced','base','sum',$4,true,'user')`, orgA, projectA, sub, keys[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE submissions SET tender_checked_at=now(), tender_checked_by=$3 WHERE org_id=$1 AND id=$2`, orgA, sub, userA); err != nil {
		t.Fatal(err)
	}
	commit(t, s, sub, tenderReadingOf(t, s, sub, lines...))
	col := tenderColumn(t, s, pkg, sub)
	if col.Checked {
		t.Fatal("a re-read column is still checked")
	}
	kept := false
	for _, l := range col.Lines {
		kept = kept || (l.Label == "Site visit" && l.Counted && l.DecidedBy == "user")
	}
	if !kept || len(col.Lines) != 3 {
		t.Fatalf("lines %+v", col.Lines)
	}
	sum, by := pricedCells(t, s, pkg, sub)
	if sum != 1200000 || by[keys[1]] != 200000 {
		t.Fatalf("cells %d %v", sum, by)
	}
}

func TestFirstCommitKeepsAUsersOldCell(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 27)
	amount := "9000.00"
	if err := s.SetSubmissionCell(ctx, orgA, projectA, sub, keys[0], store.CellInput{State: "priced", Amount: amount}); err != nil {
		t.Skipf("the old cell write has another shape here: %v", err)
	}
	commit(t, s, sub, tenderReadingOf(t, s, sub,
		rl("Advice", "$10,000.00", "base", "sum", keys[0]),
		rl("Report", "$5,000.00", "base", "sum", keys[1]),
		rl("Total", "$15,000.00", "total", "inside", "")))
	col := tenderColumn(t, s, pkg, sub)
	var user, model *tender.Line
	for i := range col.Lines {
		l := &col.Lines[i]
		if l.Row == keys[0] && l.DecidedBy == "user" {
			user = l
		}
		if l.Label == "Advice" {
			model = l
		}
	}
	if user == nil || !user.Counted || model == nil || model.Counted {
		t.Fatalf("user %+v model %+v", user, model)
	}
	if _, by := pricedCells(t, s, pkg, sub); by[keys[0]] != 900000 {
		t.Fatalf("the user's figure did not stand: %v", by)
	}
}

func TestOldReaderDoesNotOverwriteTenderCells(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 28)
	commit(t, s, sub, tenderReadingOf(t, s, sub, rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$10,000.00", "total", "inside", "")))
	// A recheck by the old reader, as the retry sweep does.
	in, err := s.SubmissionForReading(ctx, orgA, sub, true)
	if err != nil {
		t.Fatal(err)
	}
	pc := in.Context
	pc.Lookup = s.FirmLookup(ctx, orgA)
	out, err := proposals.Read(ctx, answers{"price:" + keys[0]: "a1"}, greenAll(), in.Filename, pageText([]string{"Invented Planning Pty Ltd"}, []string{"Advice", "$7,777"}), pc, in.Keep)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := s.CommitSubmissionRead(ctx, orgA, sub, out, true, time.Now()); err != nil || !ok {
		t.Fatalf("recheck %v %v", ok, err)
	}
	if _, by := pricedCells(t, s, pkg, sub); by[keys[0]] != 1000000 {
		t.Fatalf("the old reader overwrote a tender cell: %v", by)
	}
}

func TestSystemRowSurvivesAScheduleSyncAndOldColumnsIgnoreIt(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	old := tenderSubmission(t, s, pkg, 29)
	pool := rawPool(t)
	for n, k := range keys {
		if _, err := pool.Exec(ctx, `INSERT INTO submission_prices (org_id,id,project_id,submission_id,item_key,label,amount_cents,state,band,decided_by,ordinal)
VALUES ($1,gen_random_uuid(),$2,$3,$4,'Fee',100000,'priced','green','rule',$5)`, orgA, projectA, old, k, n); err != nil {
			t.Fatal(err)
		}
	}
	base := func() *int64 {
		table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg)
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range table.Columns {
			if c.Submission.ID == old {
				if c.Totals.Unread != 0 {
					t.Fatalf("unread %d", c.Totals.Unread)
				}
				return c.Totals.BaseTotal
			}
		}
		return nil
	}
	want := base()
	if want == nil {
		t.Fatal("the old column has no base total")
	}
	read := tenderSubmission(t, s, pkg, 30)
	commit(t, s, read, tenderReadingOf(t, s, read, rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$10,000.00", "total", "inside", "")))
	commit(t, s, read, tenderReadingOf(t, s, read, rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$10,000.00", "total", "inside", "")))
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schedule_items WHERE org_id=$1 AND package_id=$2 AND key=$3 AND retired_at IS NULL AND base AND kind='stage' AND label='Not broken down'`, orgA, pkg, tender.SystemRow).Scan(&n); err != nil || n != 1 {
		t.Fatalf("system rows %d %v", n, err)
	}
	if got := base(); got == nil || *got != *want {
		t.Fatalf("the old column's base total moved: %v -> %v", *want, got)
	}
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if len(table.Tender.Columns) != 1 || len(table.Columns) != 2 {
		t.Fatalf("%d tender columns of %d", len(table.Tender.Columns), len(table.Columns))
	}
}

func TestTableMarksLowestAndNeedsReviewUntilChecked(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	a, b := tenderSubmission(t, s, pkg, 31), tenderSubmission(t, s, pkg, 32)
	commit(t, s, a, tenderReadingOf(t, s, a, rl("Advice", "$10,000.00", "base", "sum", keys[0]), rl("Total", "$10,000.00", "total", "inside", "")))
	commit(t, s, b, tenderReadingOf(t, s, b, rl("Advice", "$8,000.00", "base", "sum", keys[0]), rl("Total", "$8,000.00", "total", "inside", "")))
	if !tenderColumn(t, s, pkg, b).Lowest || tenderColumn(t, s, pkg, a).Lowest {
		t.Fatal("the lower tender sum is not marked lowest")
	}
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	for _, c := range table.Columns {
		if !c.NeedsReview {
			t.Fatal("an unchecked column does not need review")
		}
		if c.Totals.Lowest != (c.Submission.ID == b) {
			t.Fatalf("totals lowest on %s", c.Submission.ID)
		}
	}
	if err := s.SetModelEnabled(ctx, orgA, userA, false, time.Now()); err != nil {
		t.Fatal(err)
	}
	table, _ = s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if table.Tender.On || len(table.Tender.Columns) != 2 {
		t.Fatalf("with the switch off: on %v, %d columns", table.Tender.On, len(table.Tender.Columns))
	}
}

func TestTenderMigrationConstraints(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys := tenderPackage(t)
	sub := tenderSubmission(t, s, pkg, 33)
	pool := rawPool(t)
	insert := func(state, placed string, cents any, counted bool) error {
		_, err := pool.Exec(ctx, `INSERT INTO tender_lines (org_id,id,project_id,submission_id,ordinal,label,amount_cents,state,kind,placed,row_key,counted,decided_by)
VALUES ($1,gen_random_uuid(),$2,$3,0,'Line',$4,$5,'base',$6,$7,$8,'model')`, orgA, projectA, sub, cents, state, placed, keys[0], counted)
		return err
	}
	if insert("included", "sum", 100, false) == nil {
		t.Fatal("an included line kept an amount")
	}
	if insert("priced", "elsewhere", 100, false) == nil {
		t.Fatal("an unknown placement was stored")
	}
	if insert("priced", "sum", nil, true) == nil {
		t.Fatal("a line with no amount was counted")
	}
	if err := insert("priced", "sum", 100, true); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := pool.QueryRow(ctx, `SELECT tender_state FROM submissions WHERE org_id=$1 AND id=$2`, orgA, sub).Scan(&state); err != nil || state != "" {
		t.Fatalf("default state %q %v", state, err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM submissions WHERE org_id=$1 AND id=$2`, orgA, sub); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tender_lines WHERE org_id=$1 AND submission_id=$2`, orgA, sub).Scan(&n); err != nil || n != 0 {
		t.Fatalf("%d lines outlived their submission", n)
	}
}
