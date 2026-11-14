package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/invoices"
	"sitewise/internal/tender"
)

// TenderOp is one named edit to a package's tender tables (contract C6):
// a line edit (tender.Op), a column check, or an add or sub.
type TenderOp struct {
	tender.Op
	Checked    *bool   `json:"checked,omitempty"`
	Adjustment string  `json:"adjustment,omitempty"`
	Applied    *bool   `json:"applied,omitempty"`
	Source     string  `json:"source,omitempty"`
	Note       *string `json:"note,omitempty"`
}

// maxTenderOps is the most operations one batch may hold.
const maxTenderOps = 200

// tenderInputError is an edit the manager must fix. Its words are for the
// screen, so it prints nothing but them.
type tenderInputError struct{ msg string }

func (e *tenderInputError) Error() string        { return e.msg }
func (e *tenderInputError) Is(target error) bool { return target == ErrInvalidInput }

func tenderInvalid(format string, args ...any) error {
	return &tenderInputError{msg: fmt.Sprintf(format, args...)}
}

// ApplyTenderOps applies a batch to one package's tender tables, or none of
// it, and returns the id of the change that can be undone ("" when the
// batch held nothing undoable, such as a column check). It never calls a
// model: every figure it stores was typed by the manager, printed by the
// firm, or computed in Go from the other firms' cells.
func (s *Store) ApplyTenderOps(ctx context.Context, org, project, pkg, actor string, version int64, ops []TenderOp) (string, error) {
	if len(ops) == 0 || len(ops) > maxTenderOps {
		return "", tenderInvalid("Send between 1 and %d operations.", maxTenderOps)
	}
	changeID := ""
	for _, op := range ops {
		if op.Op.Op != "check_column" {
			changeID = NewChangeID()
			ctx = WithChange(ctx, Change{ID: changeID, OrgID: org, ProjectID: project, Actor: actor, Via: "screen", Area: "procurement"})
			break
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return "", err
	}
	var current int64
	err = tx.QueryRow(ctx, `SELECT tender_version FROM packages WHERE org_id=$1::uuid AND project_id::text=$2 AND id::text=$3 FOR UPDATE`, org, project, pkg).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if current != version {
		return "", ErrVersionConflict
	}
	b := &tenderBatch{s: s, tx: tx, org: org, project: project, pkg: pkg, actor: actor, lines: map[string][]tender.Line{}, before: map[string][]tender.Line{}}
	if err := b.load(ctx); err != nil {
		return "", err
	}
	for i, op := range ops {
		if err := b.apply(ctx, op); err != nil {
			var oe *tender.OpError
			var ie *tenderInputError
			switch {
			case errors.As(err, &oe):
				return "", tenderInvalid("operation %d: %s", i+1, oe.Msg)
			case errors.As(err, &ie):
				return "", tenderInvalid("operation %d: %s", i+1, ie.msg)
			}
			return "", err
		}
	}
	if err := b.flush(ctx); err != nil {
		return "", err
	}
	if err := syncSuggestions(ctx, tx, org, project, pkg); err != nil {
		return "", err
	}
	if changeID != "" {
		// The change exists only if a captured row was written.
		raw, _ := json.Marshal(ops)
		tag, err := tx.Exec(ctx, `UPDATE project_changes SET ops=$3::jsonb WHERE org_id=$1::uuid AND id=$2::uuid`, org, changeID, string(raw))
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 {
			changeID = ""
		}
	}
	if err := touchTender(ctx, tx, s, org, project, pkg, "", "edited"); err != nil {
		return "", err
	}
	return changeID, tx.Commit(ctx)
}

// tenderBatch is one batch in flight: the package's tender-read
// submissions and, for those an operation touches, their lines.
type tenderBatch struct {
	s                        *Store
	tx                       pgx.Tx
	org, project, pkg, actor string
	items                    []ScheduleItem
	rows                     map[string]bool
	stored                   map[string]tenderStored
	status                   map[string]string
	lines, before            map[string][]tender.Line
	dirty                    []string
}

func (b *tenderBatch) load(ctx context.Context) error {
	var err error
	if b.items, err = readSchedule(ctx, b.tx, b.org, b.pkg); err != nil {
		return err
	}
	b.rows = map[string]bool{}
	for _, r := range baseTenderRows(b.items) {
		b.rows[r.Key] = true
	}
	b.status = map[string]string{}
	rows, err := b.tx.Query(ctx, `SELECT id::text, status FROM submissions WHERE org_id=$1::uuid AND project_id::text=$2 AND package_id::text=$3 FOR UPDATE`, b.org, b.project, b.pkg)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			rows.Close()
			return err
		}
		b.status[id] = status
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	b.stored, err = readTenderStored(ctx, b.tx, b.org, ids)
	return err
}

// column checks that a submission is a tender-read column of this package
// that may still be edited.
func (b *tenderBatch) column(sub string, edit bool) error {
	st, ok := b.stored[sub]
	switch {
	case !ok || !liveColumn(b.status[sub]):
		return tenderInvalid("That firm is not in this package's table.")
	case st.State != TenderRead:
		return tenderInvalid("This firm's prices have not been read yet.")
	case edit && b.status[sub] == SubmissionAwarded:
		return tenderInvalid("This firm is awarded. Undo the award before changing its lines.")
	}
	return nil
}

// of returns the lines of a submission, loading them on first use.
func (b *tenderBatch) of(ctx context.Context, sub string) ([]tender.Line, error) {
	if l, ok := b.lines[sub]; ok {
		return l, nil
	}
	loaded, err := tenderLines(ctx, b.tx, b.org, []string{sub}, true)
	if err != nil {
		return nil, err
	}
	b.lines[sub], b.before[sub] = loaded[sub], append([]tender.Line{}, loaded[sub]...)
	return b.lines[sub], nil
}

// owner is the submission a line belongs to, when it is in this package.
func (b *tenderBatch) owner(ctx context.Context, line string) (string, error) {
	for sub, lines := range b.lines {
		for _, l := range lines {
			if l.ID == line {
				return sub, nil
			}
		}
	}
	var sub string
	err := b.tx.QueryRow(ctx, `SELECT submission_id::text FROM tender_lines WHERE org_id=$1::uuid AND id::text=$2`, b.org, line).Scan(&sub)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", tenderInvalid("That line is no longer there. Reload the table.")
	}
	if err != nil {
		return "", err
	}
	if _, ok := b.stored[sub]; !ok {
		return "", tenderInvalid("That line is not in this package.")
	}
	return sub, nil
}

func (b *tenderBatch) apply(ctx context.Context, op TenderOp) error {
	switch name := op.Op.Op; {
	case name == "check_column":
		return b.check(ctx, op)
	case tender.LineOps[name]:
		sub := op.Submission
		if name != "add_line" && name != "set_tender_sum" {
			var err error
			if sub, err = b.owner(ctx, op.Line); err != nil {
				return err
			}
		}
		if err := b.column(sub, true); err != nil {
			return err
		}
		lines, err := b.of(ctx, sub)
		if err != nil {
			return err
		}
		next, err := tender.Apply(lines, b.rows, op.Op, newID)
		if err != nil {
			return err
		}
		b.lines[sub] = next
		if !containsString(b.dirty, sub) {
			b.dirty = append(b.dirty, sub)
		}
		return nil
	case name == "add_adjustment" || name == "set_adjustment" || name == "remove_adjustment" || name == "set_adjustment_amount" ||
		name == "apply_adjustment" || name == "take_out_line" || name == "apply_suggested":
		// Adds and subs read the cells, so the line edits before them land first.
		if err := b.flush(ctx); err != nil {
			return err
		}
		return b.adjust(ctx, op)
	}
	return tenderInvalid("Unknown operation %q.", op.Op.Op)
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// check records the manager's acceptance of one firm's reading. It is not
// an undoable change: it is their word, taken back with one more click.
func (b *tenderBatch) check(ctx context.Context, op TenderOp) error {
	if op.Checked == nil {
		return tenderInvalid("Say whether the column is checked.")
	}
	if err := b.column(op.Submission, false); err != nil {
		return err
	}
	if !*op.Checked {
		_, err := b.tx.Exec(ctx, `UPDATE submissions SET tender_checked_at=NULL, tender_checked_by=NULL WHERE org_id=$1::uuid AND id::text=$2`, b.org, op.Submission)
		return err
	}
	if err := b.flush(ctx); err != nil {
		return err
	}
	lines, err := b.of(ctx, op.Submission)
	if err != nil {
		return err
	}
	stored, err := readTenderStored(ctx, b.tx, b.org, []string{op.Submission})
	if err != nil {
		return err
	}
	r := stored[op.Submission].reading(lines)
	// Lines that add to more than the printed sum are counted twice
	// somewhere: the column cannot be accepted until that is sorted out.
	if r.TenderSum != nil && r.Gap < -100 {
		return tenderInvalid("Lines add to more than the tender sum. Take the repeated lines out of the count first.")
	}
	_, err = b.tx.Exec(ctx, `UPDATE submissions SET tender_checked_at=$3, tender_checked_by=$4::uuid WHERE org_id=$1::uuid AND id::text=$2`, b.org, op.Submission, time.Now(), b.actor)
	return err
}

// flush writes the difference each edited submission's lines now hold and
// brings its sum, flags and cells back in step.
func (b *tenderBatch) flush(ctx context.Context) error {
	for _, sub := range b.dirty {
		was := map[string]tender.Line{}
		for _, l := range b.before[sub] {
			was[l.ID] = l
		}
		for _, l := range b.lines[sub] {
			old, had := was[l.ID]
			delete(was, l.ID)
			switch {
			case !had:
				if err := insertTenderLine(ctx, b.tx, b.org, b.project, sub, l); err != nil {
					return err
				}
			case !reflect.DeepEqual(old, l):
				if err := updateTenderLine(ctx, b.tx, b.org, l); err != nil {
					return err
				}
			}
		}
		for id := range was {
			if _, err := b.tx.Exec(ctx, `DELETE FROM tender_lines WHERE org_id=$1::uuid AND id::text=$2`, b.org, id); err != nil {
				return err
			}
		}
		b.before[sub] = append([]tender.Line{}, b.lines[sub]...)
		if err := refreshTender(ctx, b.tx, b.org, b.project, b.pkg, sub, b.items); err != nil {
			return err
		}
	}
	b.dirty = nil
	return nil
}

type adjustmentRow struct {
	ID, Origin, Reason, Row string
	Dismissed               bool
}

func (b *tenderBatch) adjustment(ctx context.Context, id string) (adjustmentRow, error) {
	var a adjustmentRow
	err := b.tx.QueryRow(ctx, `SELECT id::text, origin, reason, COALESCE(row_key,''), dismissed_at IS NOT NULL FROM tender_adjustments
WHERE org_id=$1::uuid AND package_id::text=$2 AND id::text=$3 FOR UPDATE`, b.org, b.pkg, id).Scan(&a.ID, &a.Origin, &a.Reason, &a.Row, &a.Dismissed)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && a.Dismissed) {
		return a, tenderInvalid("That add or sub is no longer there. Reload the table.")
	}
	return a, err
}

func (b *tenderBatch) insertAdjustment(ctx context.Context, label, note, reason, row string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 300 {
		return "", tenderInvalid("Give the add or sub a name.")
	}
	if len(note) > 500 {
		return "", tenderInvalid("The note is too long.")
	}
	if row != "" && !b.rows[row] {
		return "", tenderInvalid("Choose a row of this schedule.")
	}
	id := newID()
	_, err := b.tx.Exec(ctx, `INSERT INTO tender_adjustments (org_id,id,project_id,package_id,ordinal,label,note,origin,reason,row_key)
SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,COALESCE(MAX(ordinal),-1)+1,$5,$6,'user',$7,NULLIF($8,'')
FROM tender_adjustments WHERE org_id=$1::uuid AND package_id=$4::uuid`, b.org, id, b.project, b.pkg, label, note, reason, row)
	return id, err
}

func (b *tenderBatch) putAmount(ctx context.Context, adj, sub string, cents int64, state, source, line string) error {
	_, err := b.tx.Exec(ctx, `INSERT INTO tender_adjustment_amounts (org_id,adjustment_id,submission_id,project_id,amount_cents,state,source,line_id)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,'')::uuid)
ON CONFLICT (org_id,adjustment_id,submission_id) DO UPDATE SET amount_cents=EXCLUDED.amount_cents, state=EXCLUDED.state, source=EXCLUDED.source,
 line_id=COALESCE(EXCLUDED.line_id, tender_adjustment_amounts.line_id), updated_at=now()`, b.org, adj, sub, b.project, cents, state, source, line)
	return err
}

// adjust applies one operation of the second table. It changes only the
// two adjustment tables: the first table never moves.
func (b *tenderBatch) adjust(ctx context.Context, op TenderOp) error {
	text := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	switch op.Op.Op {
	case "add_adjustment":
		_, err := b.insertAdjustment(ctx, text(op.Label), text(op.Note), tender.ReasonOther, text(op.Row))
		return err
	case "set_adjustment":
		a, err := b.adjustment(ctx, op.Adjustment)
		if err != nil {
			return err
		}
		if op.Label == nil && op.Note == nil {
			return tenderInvalid("Nothing to change.")
		}
		if op.Label != nil {
			if a.Origin != "user" {
				return tenderInvalid("A suggested row keeps its name. Add a note instead.")
			}
			label := strings.TrimSpace(*op.Label)
			if label == "" || len(label) > 300 {
				return tenderInvalid("Give the add or sub a name.")
			}
			if _, err := b.tx.Exec(ctx, `UPDATE tender_adjustments SET label=$3, updated_at=now() WHERE org_id=$1::uuid AND id::text=$2`, b.org, a.ID, label); err != nil {
				return err
			}
		}
		if op.Note != nil {
			if len(*op.Note) > 500 {
				return tenderInvalid("The note is too long.")
			}
			if _, err := b.tx.Exec(ctx, `UPDATE tender_adjustments SET note=$3, updated_at=now() WHERE org_id=$1::uuid AND id::text=$2`, b.org, a.ID, *op.Note); err != nil {
				return err
			}
		}
		return nil
	case "remove_adjustment":
		a, err := b.adjustment(ctx, op.Adjustment)
		if err != nil {
			return err
		}
		if a.Origin == "user" {
			_, err = b.tx.Exec(ctx, `DELETE FROM tender_adjustments WHERE org_id=$1::uuid AND id::text=$2`, b.org, a.ID)
			return err
		}
		// A system row is kept, dismissed, so it does not come back; none of
		// its amounts may go on counting out of sight.
		if _, err := b.tx.Exec(ctx, `DELETE FROM tender_adjustment_amounts WHERE org_id=$1::uuid AND adjustment_id::text=$2`, b.org, a.ID); err != nil {
			return err
		}
		_, err = b.tx.Exec(ctx, `UPDATE tender_adjustments SET dismissed_at=now(), updated_at=now() WHERE org_id=$1::uuid AND id::text=$2`, b.org, a.ID)
		return err
	case "set_adjustment_amount":
		a, err := b.adjustment(ctx, op.Adjustment)
		if err != nil {
			return err
		}
		if err := b.column(op.Submission, false); err != nil {
			return err
		}
		if op.Amount == nil || strings.TrimSpace(*op.Amount) == "" {
			_, err := b.tx.Exec(ctx, `DELETE FROM tender_adjustment_amounts WHERE org_id=$1::uuid AND adjustment_id::text=$2 AND submission_id::text=$3`, b.org, a.ID, op.Submission)
			return err
		}
		c, ok, err := invoices.ParseCents(*op.Amount)
		if err != nil || !ok {
			return tenderInvalid("Type an amount, such as 5000 or -5000.")
		}
		return b.putAmount(ctx, a.ID, op.Submission, c, AmountApplied, SourceTyped, "")
	case "apply_adjustment":
		a, err := b.adjustment(ctx, op.Adjustment)
		if err != nil {
			return err
		}
		if err := b.column(op.Submission, false); err != nil {
			return err
		}
		if op.Applied == nil {
			return tenderInvalid("Say whether to apply it.")
		}
		var cents *int64
		var state, source, line string
		err = b.tx.QueryRow(ctx, `SELECT amount_cents, state, source, COALESCE(line_id::text,'') FROM tender_adjustment_amounts
WHERE org_id=$1::uuid AND adjustment_id::text=$2 AND submission_id::text=$3 FOR UPDATE`, b.org, a.ID, op.Submission).Scan(&cents, &state, &source, &line)
		if errors.Is(err, pgx.ErrNoRows) {
			return tenderInvalid("This firm has nothing on that row to apply.")
		}
		if err != nil {
			return err
		}
		if !*op.Applied {
			if a.Reason == tender.ReasonGap && a.Origin == "system" {
				// A gap goes back to having no figure.
				_, err := b.tx.Exec(ctx, `UPDATE tender_adjustment_amounts SET amount_cents=NULL, state='suggested', source='none', updated_at=now()
WHERE org_id=$1::uuid AND adjustment_id::text=$2 AND submission_id::text=$3`, b.org, a.ID, op.Submission)
				return err
			}
			if a.Origin != "system" || source != SourceFirm {
				return tenderInvalid("Clear the amount instead: it is your own figure.")
			}
			_, err := b.tx.Exec(ctx, `UPDATE tender_adjustment_amounts SET state='suggested', updated_at=now() WHERE org_id=$1::uuid AND adjustment_id::text=$2 AND submission_id::text=$3`, b.org, a.ID, op.Submission)
			return err
		}
		if cents != nil && op.Source == "" {
			// The firm's own printed figure.
			return b.putAmount(ctx, a.ID, op.Submission, *cents, AmountApplied, source, line)
		}
		// A gap: the manager chooses where the figure comes from.
		switch op.Source {
		case SourceTyped:
			if op.Amount == nil {
				return tenderInvalid("Type a figure.")
			}
			c, ok, err := invoices.ParseCents(*op.Amount)
			if err != nil || !ok {
				return tenderInvalid("Type a figure, such as 15000.")
			}
			return b.putAmount(ctx, a.ID, op.Submission, c, AmountApplied, SourceTyped, "")
		case SourceHighest, SourceAverage:
			if a.Row == "" {
				return tenderInvalid("This row is not tied to a schedule row. Type a figure.")
			}
			subs, err := liveSubmissions(ctx, b.tx, b.org, b.project, b.pkg)
			if err != nil {
				return err
			}
			cols, _, err := packageTenderColumns(ctx, b.tx, b.org, b.items, subs)
			if err != nil {
				return err
			}
			r := tender.Others(suggestColumns(cols), op.Submission, a.Row)
			if r.Count == 0 {
				return tenderInvalid("No other firm priced this row.")
			}
			// Computed in Go from the other firms' cells, on the manager's click.
			c := r.High
			if op.Source == SourceAverage {
				c = r.Average
			}
			return b.putAmount(ctx, a.ID, op.Submission, c, AmountApplied, op.Source, "")
		}
		return tenderInvalid("Choose the highest, the average, or type a figure.")
	case "take_out_line":
		sub, err := b.owner(ctx, op.Line)
		if err != nil {
			return err
		}
		if err := b.column(sub, false); err != nil {
			return err
		}
		lines, err := b.of(ctx, sub)
		if err != nil {
			return err
		}
		for _, l := range lines {
			if l.ID != op.Line {
				continue
			}
			if !l.Counted || l.Cents == nil {
				return tenderInvalid("Only a counted line can be taken out.")
			}
			id, err := b.insertAdjustment(ctx, clipText("Less: "+l.Label, 300), "", tender.ReasonTakenOut, l.Row)
			if err != nil {
				return err
			}
			// The firm's own printed figure, subtracted. The first table does not change.
			return b.putAmount(ctx, id, sub, -*l.Cents, AmountApplied, SourceFirm, l.ID)
		}
		return tenderInvalid("That line is no longer there. Reload the table.")
	case "apply_suggested":
		if err := b.column(op.Submission, false); err != nil {
			return err
		}
		// Every suggestion that carries the firm's own figure. Gaps are left:
		// each needs the manager's choice.
		_, err := b.tx.Exec(ctx, `UPDATE tender_adjustment_amounts m SET state='applied', updated_at=now()
FROM tender_adjustments a WHERE a.org_id=m.org_id AND a.id=m.adjustment_id AND a.package_id::text=$2 AND a.dismissed_at IS NULL
AND m.org_id=$1::uuid AND m.submission_id::text=$3 AND m.state='suggested' AND m.source='firm' AND m.amount_cents IS NOT NULL`, b.org, b.pkg, op.Submission)
		return err
	}
	return tenderInvalid("Unknown operation %q.", op.Op.Op)
}
