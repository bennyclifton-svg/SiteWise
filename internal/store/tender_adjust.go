package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/tender"
)

// Adjustment amount states and sources as stored.
const (
	AmountSuggested = "suggested"
	AmountApplied   = "applied"

	SourceFirm    = "firm"
	SourceHighest = "others_highest"
	SourceAverage = "others_average"
	SourceTyped   = "typed"
	SourceNone    = "none"
)

type storedAdjustment struct {
	TenderAdjustment
	Key       string
	Dismissed bool
	Ordinal   int
}

// readAdjustments is every add or sub of a package, dismissed ones too,
// with each firm's amount, in table order.
func readAdjustments(ctx context.Context, q querier, org, pkg string) ([]storedAdjustment, error) {
	rows, err := q.Query(ctx, `SELECT id::text, ordinal, label, note, origin, reason, COALESCE(row_key,''), COALESCE(suggestion_key,''), dismissed_at IS NOT NULL
FROM tender_adjustments WHERE org_id=$1::uuid AND package_id::text=$2 ORDER BY ordinal, id`, org, pkg)
	if err != nil {
		return nil, err
	}
	var out []storedAdjustment
	at := map[string]int{}
	for rows.Next() {
		a := storedAdjustment{}
		var ordinal int32
		if err := rows.Scan(&a.ID, &ordinal, &a.Label, &a.Note, &a.Origin, &a.Reason, &a.Row, &a.Key, &a.Dismissed); err != nil {
			rows.Close()
			return nil, err
		}
		a.Ordinal, a.Amounts, a.Others = int(ordinal), map[string]TenderAmount{}, map[string]tender.Range{}
		at[a.ID] = len(out)
		out = append(out, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(out) == 0 {
		return out, err
	}
	rows, err = q.Query(ctx, `SELECT m.adjustment_id::text, m.submission_id::text, m.amount_cents, m.state, m.source, COALESCE(m.line_id::text,''), COALESCE(l.page,0)
FROM tender_adjustment_amounts m JOIN tender_adjustments a ON a.org_id=m.org_id AND a.id=m.adjustment_id
LEFT JOIN tender_lines l ON l.org_id=m.org_id AND l.id=m.line_id
WHERE m.org_id=$1::uuid AND a.package_id::text=$2`, org, pkg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var adj, sub string
		var m TenderAmount
		var page int32
		if err := rows.Scan(&adj, &sub, &m.Cents, &m.State, &m.Source, &m.LineID, &page); err != nil {
			return nil, err
		}
		m.Page = int(page)
		out[at[adj]].Amounts[sub] = m
	}
	return out, rows.Err()
}

// suggestColumns turns the package's tender-read columns into what
// tender.Suggest and tender.Others read.
func suggestColumns(cols []TenderColumn) []tender.SuggestColumn {
	var out []tender.SuggestColumn
	for _, c := range cols {
		if c.State != TenderRead {
			continue
		}
		sc := tender.SuggestColumn{ID: c.SubmissionID, Lines: c.Lines, Cells: map[string]tender.Cell{}, Marks: map[string]string{}}
		for key, cell := range c.Cells {
			if cell.State != "" {
				sc.Cells[key] = tender.Cell{State: cell.State, Cents: cell.Cents, Lines: cell.Lines}
			}
			if cell.Mark != "" {
				sc.Marks[key] = cell.Mark
			}
		}
		out = append(out, sc)
	}
	return out
}

func liveSubmissions(ctx context.Context, q querier, org, project, pkg string) ([]string, error) {
	rows, err := q.Query(ctx, `SELECT id::text FROM submissions WHERE org_id=$1::uuid AND project_id::text=$2 AND package_id::text=$3
AND status IN ('reading','received','needs_review','awarded','unsuccessful') ORDER BY received_on, created_at`, org, project, pkg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func sameCents(a, b *int64) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

// syncSuggestions brings the package's system adjustments in step with
// tender.Suggest. It never changes an applied amount, a user's row or a
// dismissed row, and it never applies anything.
func syncSuggestions(ctx context.Context, tx pgx.Tx, org, project, pkg string) error {
	items, err := readSchedule(ctx, tx, org, pkg)
	if err != nil {
		return err
	}
	subs, err := liveSubmissions(ctx, tx, org, project, pkg)
	if err != nil {
		return err
	}
	cols, _, err := packageTenderColumns(ctx, tx, org, items, subs)
	if err != nil {
		return err
	}
	optionRows := map[string]string{}
	for _, it := range items {
		if !it.Retired && it.Kind == "option" {
			optionRows[it.Key] = it.Label
		}
	}
	suggestions := tender.Suggest(baseTenderRows(items), optionRows, suggestColumns(cols))
	stored, err := readAdjustments(ctx, tx, org, pkg)
	if err != nil {
		return err
	}
	byKey := map[string]*storedAdjustment{}
	next := 0
	for i := range stored {
		next = max(next, stored[i].Ordinal+1)
		if stored[i].Key != "" {
			byKey[stored[i].Key] = &stored[i]
		}
	}
	wanted := map[string]map[string]bool{} // adjustment id -> submission ids Suggest still gives
	for _, s := range suggestions {
		a := byKey[s.Key]
		if a != nil && a.Dismissed {
			continue // the manager removed it; it stays away
		}
		id := ""
		if a == nil {
			id = newID()
			if _, err := tx.Exec(ctx, `INSERT INTO tender_adjustments (org_id,id,project_id,package_id,ordinal,label,origin,reason,row_key,suggestion_key)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,'system',$7,NULLIF($8,''),$9)`, org, id, project, pkg, next, clipText(s.Label, 300), s.Reason, s.Row, s.Key); err != nil {
				return err
			}
			next++
		} else {
			id = a.ID
			if a.Label != clipText(s.Label, 300) || a.Row != s.Row {
				if _, err := tx.Exec(ctx, `UPDATE tender_adjustments SET label=$3, row_key=NULLIF($4,''), updated_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, id, clipText(s.Label, 300), s.Row); err != nil {
					return err
				}
			}
		}
		wanted[id] = map[string]bool{}
		for sub, amt := range s.Amounts {
			wanted[id][sub] = true
			var have *TenderAmount
			if a != nil {
				if m, ok := a.Amounts[sub]; ok {
					have = &m
				}
			}
			switch {
			case have == nil:
				if _, err := tx.Exec(ctx, `INSERT INTO tender_adjustment_amounts (org_id,adjustment_id,submission_id,project_id,amount_cents,state,source,line_id)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,'suggested',$6,NULLIF($7,'')::uuid)`, org, id, sub, project, amt.Cents, amt.Source, amt.LineID); err != nil {
					return err
				}
			case have.State == AmountApplied:
				// The manager's acceptance stands, whatever the lines now say.
			case !sameCents(have.Cents, amt.Cents) || have.Source != amt.Source || have.LineID != amt.LineID:
				if _, err := tx.Exec(ctx, `UPDATE tender_adjustment_amounts SET amount_cents=$4, source=$5, line_id=NULLIF($6,'')::uuid, updated_at=now()
WHERE org_id=$1::uuid AND adjustment_id=$2::uuid AND submission_id=$3::uuid`, org, id, sub, amt.Cents, amt.Source, amt.LineID); err != nil {
					return err
				}
			}
		}
	}
	// A suggestion the lines no longer give goes; a row left with no amounts goes too.
	for i := range stored {
		a := &stored[i]
		if a.Origin != "system" || a.Dismissed {
			continue
		}
		left := 0
		for sub, m := range a.Amounts {
			if m.State == AmountSuggested && !wanted[a.ID][sub] {
				if _, err := tx.Exec(ctx, `DELETE FROM tender_adjustment_amounts WHERE org_id=$1::uuid AND adjustment_id=$2::uuid AND submission_id=$3::uuid`, org, a.ID, sub); err != nil {
					return err
				}
				continue
			}
			left++
		}
		for sub := range wanted[a.ID] {
			if _, had := a.Amounts[sub]; !had {
				left++
			}
		}
		if left == 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM tender_adjustments WHERE org_id=$1::uuid AND id=$2::uuid`, org, a.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// withAdjustments lays the package's adds and subs over its tender
// columns: each column's like-for-like total (its tender sum plus what the
// manager applied), the gaps an applied add has answered, the ranges the
// other firms priced a gap row at, and the count of suggestions waiting.
func withAdjustments(ctx context.Context, q querier, org, pkg string, cols []TenderColumn) ([]TenderAdjustment, int, error) {
	stored, err := readAdjustments(ctx, q, org, pkg)
	if err != nil {
		return nil, 0, err
	}
	at := map[string]*TenderColumn{}
	for i := range cols {
		at[cols[i].SubmissionID] = &cols[i]
	}
	sug := suggestColumns(cols)
	out := []TenderAdjustment{}
	suggested := 0
	for _, a := range stored {
		if a.Dismissed {
			continue
		}
		for sub, m := range a.Amounts {
			col := at[sub]
			if col == nil || col.State != TenderRead {
				delete(a.Amounts, sub) // an earlier or withdrawn submission's amount is not shown
				continue
			}
			switch {
			case m.State == AmountApplied && m.Cents != nil:
				if col.LikeForLike != nil {
					*col.LikeForLike += *m.Cents
				}
				col.AppliedCents += *m.Cents
				col.AppliedRows++
				if a.Reason == tender.ReasonGap && col.Gaps > 0 {
					col.Gaps-- // answered: it no longer bars the firm from "lowest"
				}
			case m.State == AmountSuggested:
				suggested++
				if a.Reason == tender.ReasonGap && a.Row != "" {
					if r := tender.Others(sug, sub, a.Row); r.Count > 0 {
						a.Others[sub] = r
					}
				}
			}
		}
		out = append(out, a.TenderAdjustment)
	}
	return out, suggested, nil
}
