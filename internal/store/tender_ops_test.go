package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/store"
	"sitewise/internal/tender"
)

// Every firm, label and figure in these tests is invented.

func tsp(v string) *string { return &v }
func tbp(v bool) *bool     { return &v }

func top(name string) store.TenderOp { return store.TenderOp{Op: tender.Op{Op: name}} }

func tenderVersion(t *testing.T, s *store.Store, pkg string) int64 {
	t.Helper()
	table, err := s.ReadPriceTable(context.Background(), orgA, projectA, pkg)
	if err != nil {
		t.Fatal(err)
	}
	return table.Tender.Version
}

// ops applies a batch at the package's current version.
func ops(t *testing.T, s *store.Store, pkg string, batch ...store.TenderOp) (string, error) {
	t.Helper()
	return s.ApplyTenderOps(context.Background(), orgA, projectA, pkg, userA, tenderVersion(t, s, pkg), batch)
}

func mustOps(t *testing.T, s *store.Store, pkg string, batch ...store.TenderOp) string {
	t.Helper()
	change, err := ops(t, s, pkg, batch...)
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func refusedOps(t *testing.T, s *store.Store, pkg, words string, batch ...store.TenderOp) {
	t.Helper()
	_, err := ops(t, s, pkg, batch...)
	if !errors.Is(err, store.ErrInvalidInput) || !strings.Contains(err.Error(), words) {
		t.Fatalf("want a refusal saying %q, got %v", words, err)
	}
}

func lineNamed(t *testing.T, col store.TenderColumn, label string) tender.Line {
	t.Helper()
	for _, l := range col.Lines {
		if l.Label == label {
			return l
		}
	}
	t.Fatalf("no line %q", label)
	return tender.Line{}
}

// twoFirms is a package with two read tenders: A prices three rows and
// offers an option and an extra; B prices two rows and excludes the third.
func twoFirms(t *testing.T) (*store.Store, string, []string, string, string) {
	t.Helper()
	s, pkg, keys := tenderPackage(t)
	a, b := tenderSubmission(t, s, pkg, 41), tenderSubmission(t, s, pkg, 42)
	extra := rl("Asbestos removal", "$1,500.00", "ps", "outside", keys[0])
	commit(t, s, a, tenderReadingOf(t, s, a,
		rl("Advice", "$10,000.00", "base", "sum", keys[0]),
		rl("Report", "$60,000.00", "base", "sum", keys[1]),
		rl("Lodgement", "$5,000.00", "base", "sum", keys[2]),
		rl("Option: extra meeting", "$1,800.00", "option", "outside", ""),
		extra,
		rl("Total", "$75,000.00", "total", "inside", "")))
	excluded := tender.RawLine{P: 1, Label: "Lodgement", G: "ex", St: "excluded", Kind: "base", Where: "sum", Row: keys[2], Sure: true}
	commit(t, s, b, tenderReadingOf(t, s, b,
		rl("Advice", "$12,000.00", "base", "sum", keys[0]),
		rl("Report", "$50,000.00", "base", "sum", keys[1]),
		excluded,
		rl("Total", "$62,000.00", "total", "inside", "")))
	return s, pkg, keys, a, b
}

func TestCheckColumn(t *testing.T) {
	ctx := context.Background()
	s, pkg, _, a, _ := twoFirms(t)
	before := tenderVersion(t, s, pkg)
	check := top("check_column")
	check.Submission, check.Checked = a, tbp(true)
	if change := mustOps(t, s, pkg, check); change != "" {
		t.Fatalf("a column check is not an undoable change: %q", change)
	}
	if col := tenderColumn(t, s, pkg, a); !col.Checked || col.CheckedAt == nil || tenderVersion(t, s, pkg) <= before {
		t.Fatalf("%+v", col)
	}
	check.Checked = tbp(false)
	mustOps(t, s, pkg, check)
	if tenderColumn(t, s, pkg, a).Checked {
		t.Fatal("still checked")
	}
	// Stale version, unknown operation, another org.
	if _, err := s.ApplyTenderOps(ctx, orgA, projectA, pkg, userA, before, []store.TenderOp{check}); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	refusedOps(t, s, pkg, "operation 1", top("fly"))
	if _, err := s.ApplyTenderOps(ctx, orgB, projectA, pkg, userB, tenderVersion(t, s, pkg), []store.TenderOp{check}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org: %v", err)
	}
	// A column that has not been read, and one whose lines add to too much.
	unread := tenderSubmission(t, s, pkg, 46)
	check.Submission, check.Checked = unread, tbp(true)
	refusedOps(t, s, pkg, "have not been read yet", check)
	add := top("add_line")
	add.Submission, add.Label, add.Amount, add.Row = a, tsp("Counted twice"), tsp("999999"), tsp("")
	mustOps(t, s, pkg, add)
	check.Submission = a
	refusedOps(t, s, pkg, "Lines add to more than the tender sum", check)
}

func TestLineEditsKeepTheColumnTotalAndUndo(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys, a, b := twoFirms(t)
	col := tenderColumn(t, s, pkg, a)
	advice := lineNamed(t, col, "Advice")

	// Move: two cells change by the line's amount, the sum does not.
	move := top("move_line")
	move.Line, move.Row = advice.ID, tsp(keys[1])
	change := mustOps(t, s, pkg, move)
	sum, by := pricedCells(t, s, pkg, a)
	if change == "" || sum != 7500000 || by[keys[0]] != 0 || by[keys[1]] != 7000000 {
		t.Fatalf("change %q sum %d cells %v", change, sum, by)
	}
	if l := lineNamed(t, tenderColumn(t, s, pkg, a), "Advice"); l.DecidedBy != "user" {
		t.Fatalf("%+v", l)
	}
	// Undo puts it back; the version moves on.
	v := tenderVersion(t, s, pkg)
	areas, err := s.UndoChange(ctx, orgA, projectA, userA, change)
	if err != nil || len(areas) != 1 || areas[0] != "procurement" {
		t.Fatalf("%v %v", areas, err)
	}
	if _, by := pricedCells(t, s, pkg, a); by[keys[0]] != 1000000 || by[keys[1]] != 6000000 || tenderVersion(t, s, pkg) <= v {
		t.Fatalf("after undo: %v", by)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change); !errors.Is(err, store.ErrChangeUndone) {
		t.Fatalf("second undo: %v", err)
	}
	if _, err := s.UndoChange(ctx, orgB, projectA, userB, change); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's undo: %v", err)
	}

	// Move to no row grows Not broken down.
	move.Row = tsp("")
	mustOps(t, s, pkg, move)
	if _, by := pricedCells(t, s, pkg, a); by[tender.SystemRow] != 1000000 {
		t.Fatalf("%v", by)
	}
	// Correct an amount: Not broken down takes the difference, not the sum.
	set := top("set_line")
	set.Line, set.Amount = advice.ID, tsp("8000.00")
	mustOps(t, s, pkg, set)
	col = tenderColumn(t, s, pkg, a)
	if sum, by := pricedCells(t, s, pkg, a); *col.Sum != 7500000 || sum != 7500000 || by[tender.SystemRow] != 1000000 {
		t.Fatalf("sum %d cells %v", sum, by)
	}
	// Set the tender sum: it is the user's, and the system row takes the difference.
	setSum := top("set_tender_sum")
	setSum.Submission, setSum.Amount = a, tsp("80000")
	sumChange := mustOps(t, s, pkg, setSum)
	col = tenderColumn(t, s, pkg, a)
	if *col.Sum != 8000000 || col.SumBy != "user" || col.SumHow != "set by you" {
		t.Fatalf("%+v", col)
	}
	if sum, _ := pricedCells(t, s, pkg, a); sum != 8000000 {
		t.Fatalf("cells add to %d", sum)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, sumChange); err != nil {
		t.Fatal(err)
	}
	if col = tenderColumn(t, s, pkg, a); *col.Sum != 7500000 || col.SumBy != "rule" {
		t.Fatalf("the reading's sum did not come back: %+v", col)
	}

	// A batch is all or nothing, and names the operation at fault.
	good, bad := top("move_line"), top("move_line")
	good.Line, good.Row = lineNamed(t, col, "Report").ID, tsp(keys[0])
	bad.Line, bad.Row = lineNamed(t, col, "Lodgement").ID, tsp("stage:nowhere")
	refusedOps(t, s, pkg, "operation 2", good, bad)
	if l := lineNamed(t, tenderColumn(t, s, pkg, a), "Report"); l.Row != keys[1] {
		t.Fatal("a refused batch changed a line")
	}
	// Two firms in one batch.
	other := top("move_line")
	other.Line, other.Row = lineNamed(t, tenderColumn(t, s, pkg, b), "Advice").ID, tsp(keys[1])
	both := mustOps(t, s, pkg, good, other)
	if _, by := pricedCells(t, s, pkg, b); by[keys[1]] != 6200000 {
		t.Fatalf("%v", by)
	}
	// Undo after a later edit of the same line is refused.
	again := top("move_line")
	again.Line, again.Row = good.Line, tsp(keys[2])
	mustOps(t, s, pkg, again)
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, both); !errors.Is(err, store.ErrChangedSince) {
		t.Fatalf("undo over a later edit: %v", err)
	}
	// An edit does not uncheck a checked column.
	check := top("check_column")
	check.Submission, check.Checked = b, tbp(true)
	mustOps(t, s, pkg, check)
	other.Row = tsp(keys[0])
	mustOps(t, s, pkg, other)
	if !tenderColumn(t, s, pkg, b).Checked {
		t.Fatal("an edit unchecked the column")
	}
}

func TestSplitThroughTheStore(t *testing.T) {
	s, pkg, keys, a, _ := twoFirms(t)
	report := lineNamed(t, tenderColumn(t, s, pkg, a), "Report")
	split := top("split_line")
	split.Line, split.Parts, split.RestRow = report.ID, []tender.Part{{Row: keys[0], Amount: "20000"}}, tsp(keys[2])
	change := mustOps(t, s, pkg, split)
	sum, by := pricedCells(t, s, pkg, a)
	if sum != 7500000 || by[keys[0]] != 3000000 || by[keys[1]] != 0 || by[keys[2]] != 4500000 {
		t.Fatalf("sum %d cells %v", sum, by)
	}
	if _, err := s.UndoChange(context.Background(), orgA, projectA, userA, change); err != nil {
		t.Fatal(err)
	}
	if col := tenderColumn(t, s, pkg, a); len(col.Lines) != 6 {
		t.Fatalf("%d lines after undoing the split", len(col.Lines))
	}
	if _, by := pricedCells(t, s, pkg, a); by[keys[1]] != 6000000 {
		t.Fatalf("%v", by)
	}
}

func adjustmentNamed(t *testing.T, s *store.Store, pkg, prefix string) (store.TenderAdjustment, bool) {
	t.Helper()
	table, err := s.ReadPriceTable(context.Background(), orgA, projectA, pkg)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range table.Tender.Adjustments {
		if strings.HasPrefix(a.Label, prefix) {
			return a, true
		}
	}
	return store.TenderAdjustment{}, false
}

func likeForLike(t *testing.T, s *store.Store, pkg, sub string) int64 {
	t.Helper()
	col := tenderColumn(t, s, pkg, sub)
	if col.LikeForLike == nil {
		t.Fatal("no like-for-like total")
	}
	return *col.LikeForLike
}

func TestSuggestionsApplyAndLikeForLike(t *testing.T) {
	ctx := context.Background()
	s, pkg, keys, a, b := twoFirms(t)
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if len(table.Tender.Adjustments) != 3 || table.Tender.Suggested != 3 {
		t.Fatalf("%d adjustments, %d suggested: %+v", len(table.Tender.Adjustments), table.Tender.Suggested, table.Tender.Adjustments)
	}
	gap, _ := adjustmentNamed(t, s, pkg, "Allowance for")
	option, _ := adjustmentNamed(t, s, pkg, "Option:")
	extra, _ := adjustmentNamed(t, s, pkg, "Extra to the price:")
	if gap.Reason != "gap" || gap.Row != keys[2] || gap.Amounts[b].Cents != nil || gap.Amounts[b].Source != "none" || gap.Others[b].Count != 1 || gap.Others[b].High != 500000 {
		t.Fatalf("gap %+v", gap)
	}
	if m := option.Amounts[a]; m.Cents == nil || *m.Cents != 180000 || m.State != "suggested" || m.Source != "firm" || m.Page != 1 {
		t.Fatalf("option %+v", option)
	}
	if likeForLike(t, s, pkg, a) != 7500000 || likeForLike(t, s, pkg, b) != 6200000 {
		t.Fatal("a suggestion counted before it was applied")
	}
	// B has an unanswered gap, so A is the lowest though B's sum is lower.
	if !tenderColumn(t, s, pkg, a).Lowest || tenderColumn(t, s, pkg, b).Lowest {
		t.Fatal("a firm with a gap was marked lowest")
	}

	// Apply the firm's own figure, and take it back.
	apply := top("apply_adjustment")
	apply.Adjustment, apply.Submission, apply.Applied = extra.ID, a, tbp(true)
	change := mustOps(t, s, pkg, apply)
	if likeForLike(t, s, pkg, a) != 7650000 {
		t.Fatalf("like for like %d", likeForLike(t, s, pkg, a))
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change); err != nil {
		t.Fatal(err)
	}
	if likeForLike(t, s, pkg, a) != 7500000 {
		t.Fatal("undo did not un-apply")
	}
	mustOps(t, s, pkg, apply)
	// The suggestion follows its line while suggested; an applied amount does not move.
	setOption := top("set_line")
	setOption.Line, setOption.Amount = option.Amounts[a].LineID, tsp("2000")
	setExtra := top("set_line")
	setExtra.Line, setExtra.Amount = extra.Amounts[a].LineID, tsp("9999")
	mustOps(t, s, pkg, setOption, setExtra)
	option, _ = adjustmentNamed(t, s, pkg, "Option:")
	extra, _ = adjustmentNamed(t, s, pkg, "Extra to the price:")
	if *option.Amounts[a].Cents != 200000 || *extra.Amounts[a].Cents != 150000 || extra.Amounts[a].State != "applied" {
		t.Fatalf("option %+v extra %+v", option.Amounts[a], extra.Amounts[a])
	}

	// A gap needs the manager's choice.
	fill := top("apply_adjustment")
	fill.Adjustment, fill.Submission, fill.Applied = gap.ID, b, tbp(true)
	refusedOps(t, s, pkg, "Choose the highest, the average, or type a figure", fill)
	fill.Source = "others_highest"
	mustOps(t, s, pkg, fill)
	gap, _ = adjustmentNamed(t, s, pkg, "Allowance for")
	if m := gap.Amounts[b]; *m.Cents != 500000 || m.Source != "others_highest" || m.State != "applied" || likeForLike(t, s, pkg, b) != 6700000 {
		t.Fatalf("%+v", m)
	}
	// Answered, the gap no longer bars B from being the lowest.
	if !tenderColumn(t, s, pkg, b).Lowest {
		t.Fatal("B has the lowest like-for-like total and no open gap")
	}
	fill.Applied = tbp(false)
	mustOps(t, s, pkg, fill)
	fill.Applied, fill.Source, fill.Amount = tbp(true), "typed", tsp("4000")
	mustOps(t, s, pkg, fill)
	if likeForLike(t, s, pkg, b) != 6600000 {
		t.Fatalf("typed gap: %d", likeForLike(t, s, pkg, b))
	}

	// The manager's own row, typed amounts, rename, note, removal.
	add := top("add_adjustment")
	add.Label = tsp("Site shed")
	mustOps(t, s, pkg, add)
	shed, _ := adjustmentNamed(t, s, pkg, "Site shed")
	typeA, typeB := top("set_adjustment_amount"), top("set_adjustment_amount")
	typeA.Adjustment, typeA.Submission, typeA.Amount = shed.ID, a, tsp("-5000")
	typeB.Adjustment, typeB.Submission, typeB.Amount = shed.ID, b, tsp("1200")
	mustOps(t, s, pkg, typeA, typeB)
	if likeForLike(t, s, pkg, a) != 7150000 || likeForLike(t, s, pkg, b) != 6720000 {
		t.Fatalf("a %d b %d", likeForLike(t, s, pkg, a), likeForLike(t, s, pkg, b))
	}
	typeB.Amount = nil
	rename := top("set_adjustment")
	rename.Adjustment, rename.Label, rename.Note = shed.ID, tsp("Site shed, by owner"), tsp("Agreed at the interview")
	mustOps(t, s, pkg, typeB, rename)
	shed, _ = adjustmentNamed(t, s, pkg, "Site shed, by owner")
	if _, has := shed.Amounts[b]; has || shed.Note != "Agreed at the interview" {
		t.Fatalf("%+v", shed)
	}
	renameSystem := top("set_adjustment")
	renameSystem.Adjustment, renameSystem.Label = option.ID, tsp("Renamed")
	refusedOps(t, s, pkg, "keeps its name", renameSystem)

	// Take a tendered line out: a subtraction, applied; the first table is unchanged.
	take := top("take_out_line")
	take.Line = lineNamed(t, tenderColumn(t, s, pkg, a), "Lodgement").ID
	mustOps(t, s, pkg, take)
	less, ok := adjustmentNamed(t, s, pkg, "Less: Lodgement")
	if !ok || *less.Amounts[a].Cents != -500000 || less.Amounts[a].State != "applied" || less.Reason != "taken_out" {
		t.Fatalf("%+v", less)
	}
	if sum, _ := pricedCells(t, s, pkg, a); sum != 7500000 || likeForLike(t, s, pkg, a) != 6650000 {
		t.Fatalf("first table %d, like for like %d", sum, likeForLike(t, s, pkg, a))
	}

	// Remove a user's row (deleted) and a system row (dismissed, and it stays away after a re-read).
	rmShed, rmOption := top("remove_adjustment"), top("remove_adjustment")
	rmShed.Adjustment, rmOption.Adjustment = shed.ID, option.ID
	mustOps(t, s, pkg, rmShed, rmOption)
	if _, ok := adjustmentNamed(t, s, pkg, "Site shed"); ok {
		t.Fatal("the user's row is still there")
	}
	commit(t, s, a, tenderReadingOf(t, s, a,
		rl("Advice", "$10,000.00", "base", "sum", keys[0]),
		rl("Report", "$60,000.00", "base", "sum", keys[1]),
		rl("Lodgement", "$5,000.00", "base", "sum", keys[2]),
		rl("Option: extra meeting", "$1,800.00", "option", "outside", ""),
		rl("Total", "$75,000.00", "total", "inside", "")))
	if _, ok := adjustmentNamed(t, s, pkg, "Option:"); ok {
		t.Fatal("a dismissed suggestion came back after a re-read")
	}
	// The applied extra outlives the line it came from.
	extra, ok = adjustmentNamed(t, s, pkg, "Extra to the price:")
	if !ok || extra.Amounts[a].State != "applied" || *extra.Amounts[a].Cents != 150000 {
		t.Fatalf("the applied amount did not survive the re-read: %+v", extra)
	}
}

func TestApplySuggestedAndRefusals(t *testing.T) {
	s, pkg, _, a, b := twoFirms(t)
	all := top("apply_suggested")
	all.Submission = a
	mustOps(t, s, pkg, all)
	if likeForLike(t, s, pkg, a) != 7500000+180000+150000 {
		t.Fatalf("%d", likeForLike(t, s, pkg, a))
	}
	all.Submission = b
	mustOps(t, s, pkg, all)
	gap, _ := adjustmentNamed(t, s, pkg, "Allowance for")
	if gap.Amounts[b].State != "suggested" {
		t.Fatal("apply_suggested filled a gap by itself")
	}
	// A submission of another package is refused.
	other, otherPkg, _ := tenderPackage2(t, s)
	amount := top("set_adjustment_amount")
	amount.Adjustment, amount.Submission, amount.Amount = gap.ID, other, tsp("100")
	refusedOps(t, s, pkg, "not in this package", amount)
	_ = otherPkg
	take := top("take_out_line")
	take.Line = lineNamed(t, tenderColumn(t, s, pkg, a), "Option: extra meeting").ID
	refusedOps(t, s, pkg, "Only a counted line can be taken out", take)
}

// tenderPackage2 is a second package in the same project with one read tender.
func tenderPackage2(t *testing.T, s *store.Store) (string, string, []string) {
	t.Helper()
	ctx := context.Background()
	pkg, keys := secondPackage(t, s)
	sub := tenderSubmission(t, s, pkg, 44)
	commit(t, s, sub, tenderReadingOf(t, s, sub, rl("Design", "$1,000.00", "base", "sum", keys[0]), rl("Total", "$1,000.00", "total", "inside", "")))
	_ = ctx
	return sub, pkg, keys
}

func TestAwardNeedsNoColumnCheck(t *testing.T) {
	ctx := context.Background()
	s, pkg, _, a, _ := twoFirms(t)
	if err := s.EditSubmission(ctx, orgA, projectA, a, store.SubmissionEdit{Firm: &store.FirmFields{Name: "Invented Planning Pty Ltd"}}); err != nil {
		t.Fatal(err)
	}
	table, _ := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	var lines []store.AwardLine
	for _, c := range table.Columns {
		if c.Submission.ID != a {
			continue
		}
		for _, it := range table.Items {
			if cell, ok := c.Cells[it.Key]; ok && it.Base && cell.Cents != nil {
				lines = append(lines, store.AwardLine{Label: it.Label, Keys: []string{it.Key}})
			}
		}
	}
	in := store.AwardInput{SubmissionID: a, Lines: lines}
	// The column was never marked checked: pressing Award is the acceptance.
	view, err := s.PreviewAward(ctx, orgA, projectA, pkg, in)
	if err != nil || view.Total != 7500000 {
		t.Fatalf("award total %d, want the tender sum: %v", view.Total, err)
	}
	if _, err := s.Award(ctx, orgA, projectA, pkg, userA, in); err != nil {
		t.Fatal(err)
	}
	// Awarded, its lines are locked.
	move := top("move_line")
	move.Line, move.Row = lineNamed(t, tenderColumn(t, s, pkg, a), "Advice").ID, tsp("")
	refusedOps(t, s, pkg, "Undo the award", move)
	_ = time.Now
}

// The manager's layout is saved for everyone: a line's place in its row,
// undone with the change log, and the firms' order in the price table.
func TestSlotsAndFirmOrder(t *testing.T) {
	ctx := context.Background()
	s, pkg, _, a, b := twoFirms(t)
	// Word matching puts each firm's "Advice" in the same place.
	advA, advB := lineNamed(t, tenderColumn(t, s, pkg, a), "Advice"), lineNamed(t, tenderColumn(t, s, pkg, b), "Advice")
	if ca, cb := tenderColumn(t, s, pkg, a), tenderColumn(t, s, pkg, b); ca.Slots[advA.ID] != cb.Slots[advB.ID] {
		t.Fatalf("like lines apart: %v %v", ca.Slots, cb.Slots)
	}
	place := func(line string, at int) store.TenderOp {
		op := top("set_slot")
		op.Line, op.Slot = line, &at
		return op
	}
	change := mustOps(t, s, pkg, place(advA.ID, 0), place(advB.ID, 1))
	if got := tenderColumn(t, s, pkg, b).Slots[advB.ID]; got != 1 {
		t.Fatalf("saved place %d", got)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change); err != nil {
		t.Fatal(err)
	}
	if ca, cb := tenderColumn(t, s, pkg, a), tenderColumn(t, s, pkg, b); ca.Slots[advA.ID] != cb.Slots[advB.ID] {
		t.Fatal("undo did not give the row back to word matching")
	}
	refusedOps(t, s, pkg, "Choose a place in the row", place(advA.ID, 1000))

	order := top("order_columns")
	order.Order = []string{b, a}
	mustOps(t, s, pkg, order)
	table, err := s.ReadPriceTable(ctx, orgA, projectA, pkg)
	if err != nil || len(table.Columns) != 2 || table.Columns[0].Submission.ID != b || table.Tender.Columns[0].SubmissionID != b {
		t.Fatalf("firm order not kept: %v", err)
	}
	other, _, _ := tenderPackage2(t, s)
	order.Order = []string{a, other}
	refusedOps(t, s, pkg, "not in this package", order)
}
