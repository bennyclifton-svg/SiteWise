package store

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/invoices"
)

// A contract's breakdown (docs/plans/2026-11-26-payment-schedules.md,
// slice 3): the lines a payment schedule prints for a live award, each
// rolling up to one of the award's cost lines. Go seeds it when the
// contract is let: one line per cost line, or the awarded firm's own
// tender lines where they add up to the cost line to the cent. The PM
// adds, renames, moves, retires and marks lines final; each edit has one
// Undo. No Jev and no model: a tender's lines and kinds are a proposal the
// PM accepted by pressing Award or Rebuild.

// BreakdownLine is one line of a contract's breakdown.
type BreakdownLine struct {
	ID, AwardID, CostLineID string
	Section, Label, Note    string
	Source                  string // award, tender or user
	Amount                  *int64 // total let, or the provisional sum; nil unknown
	Final, Retired          bool
	FinalOn                 string // the day it was marked final, in Sydney
	Ordinal                 int
	Version                 int64
	// Stored is false on a contract whose breakdown is not stored: its
	// lines are then its cost lines, and ID is the cost line's.
	Stored bool
}

// key is the ledger key of claim rows on the line; "" keys as a cost line.
func (b BreakdownLine) key() string {
	if b.Stored {
		return invoices.BreakdownKey(b.ID)
	}
	return ""
}

// BreakdownGap is a cost line whose breakdown does not add up to its
// contract value: Diff is the breakdown less the contract line.
type BreakdownGap struct {
	CostLineID, Label string
	Diff              int64
}

// ErrBreakdownIssued: a payment schedule is issued on the contract, so
// "less prior" rests on its lines and they cannot be rebuilt.
var ErrBreakdownIssued = errors.New("a payment schedule is issued on this contract, so its breakdown can no longer be rebuilt")

// BreakdownInput adds or edits a breakdown line. A nil field is left as it
// is. An empty Amount is unknown.
type BreakdownInput struct {
	CostLine *string `json:"cost_line"`
	Section  *string `json:"section"`
	Label    *string `json:"label"`
	Note     *string `json:"note"`
	Amount   *string `json:"amount"`
	Final    *bool   `json:"final"`
	Retired  *bool   `json:"retired"`
	// Rebuild (add only) replaces the breakdown with the tender's lines.
	Rebuild bool `json:"rebuild"`
}

func readBreakdown(ctx context.Context, q querier, org, project string) (map[string][]BreakdownLine, error) {
	rows, err := q.Query(ctx, `
SELECT b.award_id::text, b.id::text, b.cost_item_id::text, b.section, b.ordinal, b.label, b.note, b.amount_cents, b.final_at IS NOT NULL, b.source, b.retired_at IS NOT NULL, b.version,
 COALESCE(to_char(b.final_at AT TIME ZONE 'Australia/Sydney', 'YYYY-MM-DD'), '')
FROM contract_breakdown_lines b JOIN awards a ON a.org_id=b.org_id AND a.id=b.award_id AND a.undone_at IS NULL
WHERE b.org_id=$1::uuid AND b.project_id=$2::uuid ORDER BY b.award_id, b.ordinal, b.id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]BreakdownLine{}
	for rows.Next() {
		b := BreakdownLine{Stored: true}
		if err := rows.Scan(&b.AwardID, &b.ID, &b.CostLineID, &b.Section, &b.Ordinal, &b.Label, &b.Note, &b.Amount, &b.Final, &b.Source, &b.Retired, &b.Version, &b.FinalOn); err != nil {
			return nil, err
		}
		out[b.AwardID] = append(out[b.AwardID], b)
	}
	return out, rows.Err()
}

// setBreakdown gives a contract its breakdown. With none stored, the
// breakdown is the contract's cost lines.
func (c *Contract) setBreakdown(stored []BreakdownLine) {
	c.Breakdown, c.Gaps = nil, nil
	if len(stored) == 0 {
		for i, l := range c.Lines {
			c.Breakdown = append(c.Breakdown, BreakdownLine{ID: l.ID, AwardID: c.AwardID, CostLineID: l.ID, Section: invoices.SectionWorks, Label: l.Label, Amount: l.Amount, Source: "award", Ordinal: i})
		}
		return
	}
	c.Breakdown = stored
	sums, unknown := map[string]int64{}, map[string]bool{}
	for _, b := range stored {
		switch {
		case b.Retired:
		case b.Amount == nil:
			unknown[b.CostLineID] = true
		default:
			sums[b.CostLineID] += *b.Amount
		}
	}
	for _, l := range c.Lines {
		if l.Amount == nil || unknown[l.ID] || sums[l.ID] == *l.Amount {
			continue
		}
		c.Gaps = append(c.Gaps, BreakdownGap{CostLineID: l.ID, Label: l.Label, Diff: sums[l.ID] - *l.Amount})
	}
}

// wholeLines maps each cost line to the stored line that stands for the
// whole of it: where a claim row that names only its cost line sits.
func (c Contract) wholeLines() map[string]string {
	out := map[string]string{}
	for _, b := range c.Breakdown {
		if b.Stored && b.Source == "award" && out[b.CostLineID] == "" {
			out[b.CostLineID] = b.ID
		}
	}
	return out
}

// placeRows resolves each row's breakdown line: the one it was placed on,
// else the line that stands for its whole cost line.
func (c Contract) placeRows(rows []invoices.ClaimRow) {
	whole := c.wholeLines()
	for i := range rows {
		if rows[i].BreakdownID == "" && rows[i].LineID != "" && rows[i].VariationID == "" && rows[i].Kind != invoices.RowRetention {
			rows[i].BreakdownID = whole[rows[i].LineID]
		}
	}
}

// liveAward is a package's live award, locked for a breakdown write.
func liveAward(ctx context.Context, tx pgx.Tx, org, project, pkg string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM awards WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 AND undone_at IS NULL FOR UPDATE`, org, project, pkg).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

func awardContract(ctx context.Context, tx pgx.Tx, org, project, awardID string) (Contract, error) {
	contracts, err := readContracts(ctx, tx, org, project)
	if err != nil {
		return Contract{}, err
	}
	for _, c := range contracts {
		if c.AwardID == awardID {
			return c, nil
		}
	}
	return Contract{}, ErrNotFound
}

// seedBreakdown stores an award's breakdown when it has none. Every cost
// line gets one line, in the award's order. With split, a cost line is
// instead split into the awarded firm's counted tender lines on its rows
// when they add up to its contract value (invoices.BreakdownFromTender).
func seedBreakdown(ctx context.Context, tx pgx.Tx, org, project, awardID string, split bool) error {
	c, err := awardContract(ctx, tx, org, project, awardID)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, b := range c.Breakdown {
		if b.Stored {
			return nil
		}
	}
	parts := map[string][]invoices.TenderPart{}
	if split {
		rows, err := tx.Query(ctx, `
SELECT s.cost_item_id::text, t.label, t.kind, t.amount_cents
FROM cost_line_sources s JOIN awards a ON a.org_id=s.org_id AND a.id=s.award_id
JOIN tender_lines t ON t.org_id=s.org_id AND t.submission_id=a.submission_id AND t.row_key=s.item_key AND t.counted AND t.amount_cents IS NOT NULL
WHERE s.org_id=$1::uuid AND s.award_id=$2::uuid AND s.item_key IS NOT NULL
ORDER BY s.cost_item_id, t.ordinal, t.id`, org, awardID)
		if err != nil {
			return err
		}
		for rows.Next() {
			var line string
			var p invoices.TenderPart
			if err := rows.Scan(&line, &p.Label, &p.Kind, &p.Amount); err != nil {
				rows.Close()
				return err
			}
			parts[line] = append(parts[line], p)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	ordinal := 0
	for _, l := range c.Lines {
		lines, ok := invoices.BreakdownFromTender(l.Amount, parts[l.ID])
		source := "tender"
		if !ok {
			lines, source = []invoices.BreakdownLine{{Section: invoices.SectionWorks, Label: l.Label, Amount: l.Amount}}, "award"
		}
		for _, b := range lines {
			if err := insertBreakdownLine(ctx, tx, org, project, awardID, l.ID, b.Section, breakdownLabel(b.Label), "", b.Amount, source, ordinal); err != nil {
				return err
			}
			ordinal++
		}
	}
	return nil
}

func breakdownLabel(s string) string {
	s = clipText(strings.TrimSpace(s), 500)
	if s == "" {
		return "Contract line"
	}
	return s
}

func insertBreakdownLine(ctx context.Context, tx pgx.Tx, org, project, awardID, costLine, section, label, note string, amount *int64, source string, ordinal int) error {
	_, err := tx.Exec(ctx, `INSERT INTO contract_breakdown_lines (org_id,id,project_id,award_id,cost_item_id,section,ordinal,label,note,amount_cents,source)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8,$9,$10,$11)`, org, newID(), project, awardID, costLine, section, ordinal, label, note, amount, source)
	return err
}

// breakdownTx opens a breakdown write on a package's live contract and
// makes sure its breakdown is stored, so the write has rows to work on.
func (s *Store) breakdownTx(ctx context.Context, org, project, actor, pkg string) (pgx.Tx, Contract, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, Contract{}, err
	}
	fail := func(err error) (pgx.Tx, Contract, error) {
		_ = tx.Rollback(ctx)
		return nil, Contract{}, err
	}
	if err := lockProject(ctx, tx, org, project); err != nil {
		return fail(err)
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return fail(err)
	}
	awardID, err := liveAward(ctx, tx, org, project, pkg)
	if err != nil {
		return fail(err)
	}
	if err := seedBreakdown(ctx, tx, org, project, awardID, false); err != nil {
		return fail(err)
	}
	c, err := awardContract(ctx, tx, org, project, awardID)
	if err != nil {
		return fail(err)
	}
	return tx, c, nil
}

func breakdownSection(s string) (string, error) {
	switch s {
	case invoices.SectionWorks, invoices.SectionProvisional:
		return s, nil
	}
	return "", claimInvalid("a breakdown line is works or a provisional sum")
}

func breakdownAmount(s string) (*int64, error) {
	c, ok, err := invoices.ParseCents(s)
	if err != nil || ok && c < 0 {
		return nil, claimInvalid("the sum is not an amount")
	}
	if !ok {
		return nil, nil
	}
	return &c, nil
}

// AddBreakdownLine adds a line to a contract's breakdown. With no cost line
// named it rolls up to the cost line of the last line in its section, else
// the contract's first cost line.
func (s *Store) AddBreakdownLine(ctx context.Context, org, project, actor, pkg string, in BreakdownInput) error {
	if in.Rebuild {
		return s.rebuildBreakdown(ctx, org, project, actor, pkg)
	}
	section := invoices.SectionWorks
	if in.Section != nil {
		var err error
		if section, err = breakdownSection(*in.Section); err != nil {
			return err
		}
	}
	if in.Label == nil || strings.TrimSpace(*in.Label) == "" || len(*in.Label) > 500 {
		return claimInvalid("a breakdown line needs a description of up to 500 characters")
	}
	note := ""
	if in.Note != nil {
		if note = strings.TrimSpace(*in.Note); len(note) > 300 {
			return claimInvalid("a note is at most 300 characters")
		}
	}
	var amount *int64
	if in.Amount != nil {
		var err error
		if amount, err = breakdownAmount(*in.Amount); err != nil {
			return err
		}
	}
	tx, c, err := s.breakdownTx(ctx, org, project, actor, pkg)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if len(c.Lines) == 0 {
		return claimInvalid("this contract has no cost line to hold a breakdown")
	}
	costLine, next := c.Lines[0].ID, 0
	for _, b := range c.Breakdown {
		if b.Ordinal >= next {
			next = b.Ordinal + 1
		}
		if b.Section == section && !b.Retired {
			costLine = b.CostLineID
		}
	}
	if in.CostLine != nil && *in.CostLine != "" {
		costLine = ""
		for _, l := range c.Lines {
			if l.ID == *in.CostLine {
				costLine = l.ID
			}
		}
		if costLine == "" {
			return claimInvalid("that cost line is not on this contract")
		}
	}
	if err := insertBreakdownLine(ctx, tx, org, project, c.AwardID, costLine, section, strings.TrimSpace(*in.Label), note, amount, "user", next); err != nil {
		return err
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
}

// EditBreakdownLine renames a line, types its note or sum, moves it between
// works and provisional sums, marks a provisional sum final, or retires it.
// A retired line is kept: issued schedules print it.
func (s *Store) EditBreakdownLine(ctx context.Context, org, project, actor, pkg, lineID string, in BreakdownInput) error {
	tx, c, err := s.breakdownTx(ctx, org, project, actor, pkg)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var b *BreakdownLine
	for i := range c.Breakdown {
		// A line named by its cost line, from a screen that read the
		// contract before its breakdown was stored.
		if c.Breakdown[i].ID == lineID || c.Breakdown[i].Source == "award" && c.Breakdown[i].CostLineID == lineID {
			b = &c.Breakdown[i]
			break
		}
	}
	if b == nil {
		return ErrNotFound
	}
	if in.Label != nil {
		if strings.TrimSpace(*in.Label) == "" || len(*in.Label) > 500 {
			return claimInvalid("a breakdown line needs a description of up to 500 characters")
		}
		b.Label = strings.TrimSpace(*in.Label)
	}
	if in.Note != nil {
		if b.Note = strings.TrimSpace(*in.Note); len(b.Note) > 300 {
			return claimInvalid("a note is at most 300 characters")
		}
	}
	if in.Amount != nil {
		if b.Amount, err = breakdownAmount(*in.Amount); err != nil {
			return err
		}
	}
	if in.Section != nil {
		if b.Section, err = breakdownSection(*in.Section); err != nil {
			return err
		}
	}
	if in.Final != nil {
		b.Final = *in.Final
	}
	if b.Final && b.Section != invoices.SectionProvisional {
		if in.Final != nil && *in.Final {
			return claimInvalid("only a provisional sum is marked final")
		}
		// Moved to works: it is no longer a sum to settle.
		b.Final = false
	}
	if in.Retired != nil {
		b.Retired = *in.Retired
	}
	tag, err := tx.Exec(ctx, `UPDATE contract_breakdown_lines SET label=$3, note=$4, amount_cents=$5, section=$6,
 final_at=CASE WHEN NOT $7 THEN NULL ELSE COALESCE(final_at, now()) END,
 retired_at=CASE WHEN NOT $8 THEN NULL ELSE COALESCE(retired_at, now()) END, version=version+1
WHERE org_id=$1::uuid AND id=$2::uuid`, org, b.ID, b.Label, b.Note, b.Amount, b.Section, b.Final, b.Retired)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
}

// rebuildBreakdown replaces a contract's breakdown with the awarded firm's
// own tender lines, where they add up. It is refused once a schedule is
// issued on the contract. Rows of open claims are placed again by label.
func (s *Store) rebuildBreakdown(ctx context.Context, org, project, actor, pkg string) error {
	tx, c, err := s.breakdownTx(ctx, org, project, actor, pkg)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var issued bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued'
JOIN invoices i ON i.org_id=r.org_id AND i.id=r.invoice_id
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.kind='payment_schedule' AND i.package_id::text=$3)`, org, project, pkg).Scan(&issued); err != nil {
		return err
	}
	if issued {
		return ErrBreakdownIssued
	}
	if _, err := tx.Exec(ctx, `UPDATE progress_claim_lines SET breakdown_line_id=NULL, version=version+1
WHERE org_id=$1::uuid AND breakdown_line_id IN (SELECT id FROM contract_breakdown_lines WHERE org_id=$1::uuid AND award_id=$2::uuid)`, org, c.AwardID); err != nil {
		return claimWriteError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM contract_breakdown_lines WHERE org_id=$1::uuid AND award_id=$2::uuid`, org, c.AwardID); err != nil {
		return err
	}
	if err := seedBreakdown(ctx, tx, org, project, c.AwardID, true); err != nil {
		return err
	}
	// A row sits on the new line whose words are its own, when one is.
	if _, err := tx.Exec(ctx, `UPDATE progress_claim_lines l SET breakdown_line_id=b.id, version=l.version+1
FROM contract_breakdown_lines b
WHERE l.org_id=$1::uuid AND b.org_id=l.org_id AND b.award_id=$2::uuid AND b.cost_item_id=l.cost_item_id AND b.source<>'award'
 AND lower(btrim(b.label))=lower(btrim(l.label)) AND l.breakdown_line_id IS NULL
 AND (SELECT count(*) FROM contract_breakdown_lines o WHERE o.org_id=b.org_id AND o.award_id=b.award_id AND o.cost_item_id=b.cost_item_id AND lower(btrim(o.label))=lower(btrim(b.label)))=1`, org, c.AwardID); err != nil {
		return claimWriteError(err)
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
}
