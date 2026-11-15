package store

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/proposals"
	"sitewise/internal/tender"
)

const tenderLineColumns = `id::text, ordinal, page, label, printed, amount_cents, gst, state, kind, placed, heading, COALESCE(row_key,''), note, counted, flags, decided_by, COALESCE(split_of::text,''), slot`

func scanTenderLine(row pgx.Rows, sub *string) (tender.Line, error) {
	var l tender.Line
	var ordinal, page int32
	var slot *int32
	var flags []byte
	dest := []any{&l.ID, &ordinal, &page, &l.Label, &l.Printed, &l.Cents, &l.GST, &l.State, &l.Kind, &l.Placed, &l.Heading, &l.Row, &l.Note, &l.Counted, &flags, &l.DecidedBy, &l.SplitOf, &slot}
	if sub != nil {
		dest = append([]any{sub}, dest...)
	}
	if err := row.Scan(dest...); err != nil {
		return l, err
	}
	l.Ordinal, l.Page = int(ordinal), int(page)
	if slot != nil {
		v := int(*slot)
		l.Slot = &v
	}
	if err := json.Unmarshal(flags, &l.Flags); err != nil {
		return l, err
	}
	return l, nil
}

// tenderLines is every stored line of the given submissions, in order.
func tenderLines(ctx context.Context, q querier, org string, subs []string, lock bool) (map[string][]tender.Line, error) {
	out := map[string][]tender.Line{}
	if len(subs) == 0 {
		return out, nil
	}
	sql := `SELECT submission_id::text, ` + tenderLineColumns + ` FROM tender_lines WHERE org_id=$1::uuid AND submission_id::text = ANY($2::text[]) ORDER BY submission_id, ordinal, id`
	if lock {
		sql += ` FOR UPDATE`
	}
	rows, err := q.Query(ctx, sql, org, subs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sub string
		l, err := scanTenderLine(rows, &sub)
		if err != nil {
			return nil, err
		}
		out[sub] = append(out[sub], l)
	}
	return out, rows.Err()
}

func insertTenderLine(ctx context.Context, tx pgx.Tx, org, project, sub string, l tender.Line) error {
	flags, _ := json.Marshal(flagsOrEmpty(l.Flags))
	_, err := tx.Exec(ctx, `INSERT INTO tender_lines (org_id,id,project_id,submission_id,ordinal,page,label,printed,amount_cents,gst,state,kind,placed,heading,row_key,note,counted,flags,decided_by,split_of,slot)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15,''),$16,$17,$18::jsonb,$19,NULLIF($20,'')::uuid,$21)`,
		org, l.ID, project, sub, l.Ordinal, max(l.Page, 0), clipText(l.Label, 500), clipText(l.Printed, 80), l.Cents, gstColumn(l.GST), l.State, l.Kind, l.Placed,
		clipText(l.Heading, 300), l.Row, clipText(l.Note, 300), l.Counted, string(flags), l.DecidedBy, l.SplitOf, l.Slot)
	return err
}

func updateTenderLine(ctx context.Context, tx pgx.Tx, org string, l tender.Line) error {
	flags, _ := json.Marshal(flagsOrEmpty(l.Flags))
	_, err := tx.Exec(ctx, `UPDATE tender_lines SET ordinal=$3, page=$4, label=$5, printed=$6, amount_cents=$7, gst=$8, state=$9, kind=$10, placed=$11, heading=$12,
 row_key=NULLIF($13,''), note=$14, counted=$15, flags=$16::jsonb, decided_by=$17, split_of=NULLIF($18,'')::uuid, slot=$19, updated_at=now()
WHERE org_id=$1::uuid AND id=$2::uuid`, org, l.ID, l.Ordinal, max(l.Page, 0), clipText(l.Label, 500), clipText(l.Printed, 80), l.Cents, gstColumn(l.GST), l.State, l.Kind, l.Placed,
		clipText(l.Heading, 300), l.Row, clipText(l.Note, 300), l.Counted, string(flags), l.DecidedBy, l.SplitOf, l.Slot)
	return err
}

func flagsOrEmpty(f []tender.Flag) []tender.Flag {
	if f == nil {
		return []tender.Flag{}
	}
	return f
}

// tenderStored is the tender-pass state kept on a submission.
type tenderStored struct {
	ID, PackageID, State, Error, Basis, SumHow, SumBy, ReadHow string
	Sum, ReadSum                                               *int64
	Flags                                                      []tender.Flag
	Notes                                                      []string
	Exclusions                                                 []tender.RawExclusion
	CheckedAt                                                  *time.Time
}

func readTenderStored(ctx context.Context, q querier, org string, subs []string) (map[string]tenderStored, error) {
	out := map[string]tenderStored{}
	if len(subs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `SELECT id::text, COALESCE(package_id::text,''), tender_state, tender_error, tender_basis, tender_sum_cents, tender_sum_how, tender_sum_by,
 tender_sum_read_cents, tender_sum_read_how, tender_flags, tender_notes, tender_exclusions, tender_checked_at
FROM submissions WHERE org_id=$1::uuid AND id::text = ANY($2::text[])`, org, subs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st tenderStored
		var flags, notes, excl []byte
		if err := rows.Scan(&st.ID, &st.PackageID, &st.State, &st.Error, &st.Basis, &st.Sum, &st.SumHow, &st.SumBy, &st.ReadSum, &st.ReadHow, &flags, &notes, &excl, &st.CheckedAt); err != nil {
			return nil, err
		}
		if json.Unmarshal(flags, &st.Flags) != nil || json.Unmarshal(notes, &st.Notes) != nil || json.Unmarshal(excl, &st.Exclusions) != nil {
			return nil, errors.New("stored tender reading is malformed")
		}
		out[st.ID] = st
	}
	return out, rows.Err()
}

// reading rebuilds a submission's reading from what is stored and brings
// its gap, remainder and column flags in step with the lines.
func (st tenderStored) reading(lines []tender.Line) tender.Reading {
	r := tender.Reading{Basis: st.Basis, TenderSum: st.Sum, SumHow: st.SumHow, Lines: lines, Exclusions: st.Exclusions, Notes: st.Notes, Flags: append([]tender.Flag{}, st.Flags...)}
	tender.Recount(&r)
	return r
}

// CommitTenderRead stores one reading: its lines, its tender sum and flags,
// and the price cells projected from the counted lines. Lines the user
// decided are kept. incomplete is true when the answer was cut off.
func (s *Store) CommitTenderRead(ctx context.Context, org, id string, r tender.Reading, prompt string, incomplete bool, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var project, pkg string
	err = tx.QueryRow(ctx, `SELECT project_id::text, COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1::uuid AND id::text=$2`, org, id).Scan(&project, &pkg)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && pkg == "") {
		return nil
	}
	if err != nil {
		return err
	}
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1::uuid AND id::text=$2 FOR UPDATE`, org, id).Scan(&pkg); err != nil || pkg == "" {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if err := s.ensureSchedule(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	if err := ensureTenderRow(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	items, err := readSchedule(ctx, tx, org, pkg)
	if err != nil {
		return err
	}
	baseRow := map[string]bool{}
	for _, it := range items {
		if !it.Retired && it.Price() && it.Base && it.Key != tender.SystemRow {
			baseRow[it.Key] = true
		}
	}
	existing, err := tenderLines(ctx, tx, org, []string{id}, true)
	if err != nil {
		return err
	}
	kept := []tender.Line{}
	// A line read again keeps the id it had: suggestions, applied adds and
	// subs and a dismissed suggestion all point at a line by its id.
	sameLine := func(l tender.Line) string { return strconv.Itoa(l.Page) + "|" + l.Label + "|" + l.Printed }
	oldIDs := map[string][]string{}
	for _, l := range existing[id] {
		if l.DecidedBy != proposals.ByUser {
			oldIDs[sameLine(l)] = append(oldIDs[sameLine(l)], l.ID)
		}
	}
	if len(existing[id]) == 0 {
		// The first reading: a figure the manager typed into a cell of the
		// old table becomes their line, so a later reading never overwrites it.
		if kept, err = userCellLines(ctx, tx, org, project, id, baseRow); err != nil {
			return err
		}
	} else {
		for _, l := range existing[id] {
			if l.DecidedBy == proposals.ByUser {
				kept = append(kept, l)
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM tender_lines WHERE org_id=$1::uuid AND submission_id::text=$2 AND decided_by <> 'user'`, org, id); err != nil {
			return err
		}
	}
	// A line the manager already moved, split or corrected stands in for the
	// same line read again; and their figure for a whole row stands for it.
	rowFigure := map[string]bool{}
	taken := make([]bool, len(r.Lines))
	maxOrdinal := len(r.Lines)
	for _, k := range kept {
		maxOrdinal = max(maxOrdinal, k.Ordinal+1)
		if k.Note == tender.RowFigureNote && k.Counted {
			rowFigure[k.Row] = true
			continue
		}
		if k.SplitOf != "" {
			continue // its original is kept too, and that is what matches
		}
		for i, l := range r.Lines {
			if !taken[i] && l.Page == k.Page && (l.Label == k.Label || (l.Cents != nil && k.Cents != nil && *l.Cents == *k.Cents && k.Printed == l.Printed)) {
				taken[i] = true
				break
			}
		}
	}
	for i, l := range r.Lines {
		if taken[i] {
			continue
		}
		l.ID, l.Ordinal = newID(), i
		if ids := oldIDs[sameLine(l)]; len(ids) > 0 {
			l.ID, oldIDs[sameLine(l)] = ids[0], ids[1:]
		}
		if l.DecidedBy == "" {
			l.DecidedBy = "model"
		}
		if l.Counted && rowFigure[l.Row] {
			l.Counted = false
			l.Flags = append(l.Flags, tender.Flag{Code: tender.FlagBreakdown, Text: "Your figure for this row stands."})
		}
		if err := insertTenderLine(ctx, tx, org, project, id, l); err != nil {
			return err
		}
	}
	flags := append([]tender.Flag{}, r.Flags...)
	if incomplete {
		flags = append(flags, tender.Flag{Code: tender.FlagIncomplete, Text: "The reading was cut off. Some lines may be missing."})
	}
	raw, _ := json.Marshal(flagsOrEmpty(flags))
	notes, _ := json.Marshal(stringsOrEmpty(r.Notes))
	excl := r.Exclusions
	if excl == nil {
		excl = []tender.RawExclusion{}
	}
	exclusions, _ := json.Marshal(excl)
	basis := r.Basis
	if basis == "" {
		basis = "unknown"
	}
	// The pass's own sum is kept beside the sum in force, so that undoing a
	// sum the manager set can put the pass's back (refreshTender).
	if _, err := tx.Exec(ctx, `UPDATE submissions SET tender_state='read', tender_error='', tender_retry_after=NULL, tender_prompt=$3, tender_basis=$4,
 tender_sum_read_cents=$5, tender_sum_read_how=$6, tender_flags=$7::jsonb, tender_notes=$8::jsonb, tender_exclusions=$9::jsonb, tender_read_at=$10,
 tender_checked_at=NULL, tender_checked_by=NULL, updated_at=now()
WHERE org_id=$1::uuid AND id::text=$2`, org, id, clipText(prompt, 40), basis, r.TenderSum, clipText(r.SumHow, 200), string(raw), string(notes), string(exclusions), now); err != nil {
		return err
	}
	if err := refreshTender(ctx, tx, org, project, pkg, id, items); err != nil {
		return err
	}
	// Suggestions are rebuilt from the new lines; applied amounts and the
	// manager's own rows are left alone.
	if err := syncSuggestions(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM comparison_alignment WHERE org_id=$1::uuid AND submission_id::text=$2`, org, id); err != nil {
		return err
	}
	if err := touchTender(ctx, tx, s, org, project, pkg, id, TenderRead); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// userCellLines turns each price cell the manager decided on a base row
// into their own line.
func userCellLines(ctx context.Context, tx pgx.Tx, org, project, sub string, baseRow map[string]bool) ([]tender.Line, error) {
	rows, err := tx.Query(ctx, `SELECT COALESCE(item_key,''), label, amount_cents, page FROM submission_prices
WHERE org_id=$1::uuid AND submission_id::text=$2 AND decided_by='user' AND amount_cents IS NOT NULL AND state='priced' ORDER BY ordinal, id`, org, sub)
	if err != nil {
		return nil, err
	}
	var out []tender.Line
	for rows.Next() {
		var l tender.Line
		var page int32
		var cents int64
		if err := rows.Scan(&l.Row, &l.Label, &cents, &page); err != nil {
			rows.Close()
			return nil, err
		}
		if !baseRow[l.Row] {
			continue
		}
		l.ID, l.Page, l.Cents, l.GST, l.State, l.Kind, l.Placed, l.Counted, l.DecidedBy, l.Note = newID(), int(page), &cents, "ex", tender.StatePriced, "base", "sum", true, proposals.ByUser, tender.RowFigureNote
		l.Ordinal = 100000 + len(out)
		if l.Label == "" {
			l.Label = "Your figure"
		}
		out = append(out, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, l := range out {
		if err := insertTenderLine(ctx, tx, org, project, sub, l); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// refreshTender brings everything derived from a submission's lines back
// in step with them: the tender sum, the flags and the price cells. It
// runs after every reading, every edit and every undo. items may be nil.
func refreshTender(ctx context.Context, tx pgx.Tx, org, project, pkg, sub string, items []ScheduleItem) error {
	if items == nil {
		var err error
		if items, err = readSchedule(ctx, tx, org, pkg); err != nil {
			return err
		}
	}
	stored, err := readTenderStored(ctx, tx, org, []string{sub})
	if err != nil {
		return err
	}
	st, ok := stored[sub]
	if !ok || st.State != TenderRead {
		return nil
	}
	lines, err := tenderLines(ctx, tx, org, []string{sub}, false)
	if err != nil {
		return err
	}
	st.Sum, st.SumHow, st.SumBy = st.ReadSum, st.ReadHow, "rule"
	if user := tender.UserTenderSum(lines[sub]); user != nil {
		st.Sum, st.SumHow, st.SumBy = user, "set by you", "user"
	}
	r := st.reading(lines[sub])
	flags, _ := json.Marshal(flagsOrEmpty(r.Flags))
	if _, err := tx.Exec(ctx, `UPDATE submissions SET tender_sum_cents=$3, tender_sum_how=$4, tender_sum_by=$5, tender_flags=$6::jsonb, updated_at=now()
WHERE org_id=$1::uuid AND id::text=$2`, org, sub, st.Sum, clipText(st.SumHow, 200), st.SumBy, string(flags)); err != nil {
		return err
	}
	return projectTenderCells(ctx, tx, org, project, sub, r, items)
}

// baseTenderRows is the schedule's live base rows other than the system
// row: the rows of the first table.
func baseTenderRows(items []ScheduleItem) []tender.Row {
	var rows []tender.Row
	for _, it := range items {
		if !it.Retired && it.Price() && it.Base && it.Key != tender.SystemRow {
			rows = append(rows, tender.Row{Key: it.Key, Label: it.Label, Detail: it.Detail})
		}
	}
	return rows
}

// projectTenderCells rewrites a tender-read submission's price cells from
// its lines (rule R9). It owns the cells on base rows, option rows and the
// system row, and no others (rates stay with the old reader).
func projectTenderCells(ctx context.Context, tx pgx.Tx, org, project, sub string, r tender.Reading, items []ScheduleItem) error {
	cells := tender.Cells(r, baseTenderRows(items))
	var owned []string
	for _, it := range items {
		if !it.Retired && it.Price() {
			owned = append(owned, it.Key)
		}
	}
	if _, err := tx.Exec(ctx, `DELETE FROM submission_prices WHERE org_id=$1::uuid AND submission_id::text=$2 AND item_key = ANY($3::text[])`, org, sub, owned); err != nil {
		return err
	}
	type behind struct {
		page          int
		lines, user   int
		optionCents   int64
		optionPresent bool
	}
	by := map[string]*behind{}
	at := func(key string) *behind {
		if by[key] == nil {
			by[key] = &behind{}
		}
		return by[key]
	}
	known := map[string]bool{}
	for _, it := range items {
		if !it.Retired && it.Price() && it.Base && it.Key != tender.SystemRow {
			known[it.Key] = true
		}
	}
	for _, l := range r.Lines {
		if l.Cents == nil {
			continue
		}
		if l.Counted {
			key := l.Row
			if !known[key] {
				key = tender.SystemRow
			}
			b := at(key)
			if b.lines == 0 || (l.Page > 0 && l.Page < b.page) || b.page == 0 {
				b.page = l.Page
			}
			b.lines++
			if l.DecidedBy == proposals.ByUser {
				b.user++
			}
			continue
		}
		if l.Placed == "outside" && l.Row != "" && !known[l.Row] {
			b := at(l.Row)
			b.optionCents, b.optionPresent = b.optionCents+*l.Cents, true
			if b.page == 0 || (l.Page > 0 && l.Page < b.page) {
				b.page = l.Page
			}
		}
	}
	for _, it := range items {
		if it.Retired || !it.Price() {
			continue
		}
		var cents *int64
		state, basis, band, decided, page := "", "", proposals.BandGreen, proposals.ByRule, 0
		b := by[it.Key]
		if b != nil {
			page = b.page
		}
		switch cell, ok := cells[it.Key]; {
		case ok:
			state, cents = cell.State, cell.Cents
			if state != tender.StatePriced {
				cents = nil
			}
			basis = cell.Note
			if state == tender.StatePriced {
				basis = strconv.Itoa(cell.Lines) + " lines"
				if cell.Lines == 1 {
					basis = "1 line"
				}
				if it.Key == tender.SystemRow && cell.Lines == 0 {
					basis = "not broken down"
				}
			}
			if cell.ToCheck {
				band = proposals.BandAmber
			}
			if b != nil && b.lines > 0 && b.user == b.lines {
				decided = proposals.ByUser
			}
		case !it.Base && b != nil && b.optionPresent:
			v := b.optionCents
			state, cents, basis = tender.StatePriced, &v, "offered outside the tender sum"
		default:
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO submission_prices (org_id,id,project_id,submission_id,item_key,label,amount_cents,state,basis,page,passage,band,decided_by,ordinal)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,$9,$10,'',$11,$12,$13)`,
			org, newID(), project, sub, it.Key, clipText(it.Label, 500), cents, state, clipText(basis, 500), max(page, 0), band, decided, it.Ordinal); err != nil {
			return err
		}
	}
	return nil
}

// isTenderKey reports whether a schedule key belongs to the tender tables.
func isTenderKey(key string) bool { return strings.HasPrefix(key, tenderPrefix) }
