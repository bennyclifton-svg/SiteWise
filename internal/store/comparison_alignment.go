package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/proposals"
)

type ComparisonRowInput struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Section string `json:"section"`
	Unit    string `json:"unit"`
}

// Comparison overrides do not alter the RFP's wording or program stage IDs.
func (s *Store) SetComparisonRow(ctx context.Context, org, project, pkg string, in ComparisonRowInput) error {
	in.Label = strings.TrimSpace(in.Label)
	if in.Key == "comparison:appointment" || in.Label == "" || len(in.Label) > 500 || (in.Section != "base" && in.Section != "option" && in.Section != "rate") {
		return ErrInvalidInput
	}
	if in.Section == "rate" {
		switch in.Unit {
		case "hour", "meeting", "inspection", "approval", "report", "review", "revision":
		default:
			return ErrInvalidInput
		}
	} else {
		in.Unit = ""
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockComparison(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	if err = s.ensureSchedule(ctx, tx, org, project, pkg); err != nil {
		return err
	}
	kind := in.Section
	if in.Section == "base" {
		kind = "stage"
	}
	if in.Key == "" {
		_, err = tx.Exec(ctx, `INSERT INTO schedule_items(org_id,project_id,package_id,key,kind,label,base,ordinal,source,comparison_custom,comparison_label,comparison_kind,comparison_unit)
 SELECT $1::uuid,$2::uuid,$3::uuid,$4,$5,$6,$7,COALESCE(max(ordinal),0)+1,'generic',true,$6,$5,$8 FROM schedule_items WHERE org_id=$1::uuid AND package_id=$3::uuid`, org, project, pkg, "comparison:"+newID(), kind, in.Label, in.Section == "base", in.Unit)
	} else {
		var tag interface{ RowsAffected() int64 }
		tag, err = tx.Exec(ctx, `UPDATE schedule_items SET comparison_label=$4,comparison_kind=$5,comparison_unit=$7,updated_at=now() WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id=$3::uuid AND key=$6 AND retired_at IS NULL AND kind <> 'criterion' AND ((kind='rate')=($5='rate'))`, org, project, pkg, in.Label, kind, in.Key, in.Unit)
		if err == nil && tag.RowsAffected() != 1 {
			return ErrNotFound
		}
	}
	if err != nil {
		return err
	}
	if err = procurementEvent(ctx, tx, s, org, project, "", "edited"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func lockComparison(ctx context.Context, tx pgx.Tx, org, project, pkg string) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3 FOR UPDATE`, org, project, pkg).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	var awarded bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM awards WHERE org_id=$1::uuid AND package_id=$2::uuid AND undone_at IS NULL)`, org, pkg).Scan(&awarded); err != nil {
		return err
	}
	if awarded {
		return ErrConflict
	}
	return nil
}

type AlignmentInput struct {
	Target   string   `json:"target"`
	PriceIDs []string `json:"price_ids"`
	Version  int64    `json:"version"`
	Undo     bool     `json:"undo"`
	Exclude  bool     `json:"exclude"`
}

// AlignSubmission moves known fees, never duplicates them. Original rows and
// decisions are saved atomically for undo; source descriptions remain on the fee.
func (s *Store) AlignSubmission(ctx context.Context, org, project, id string, in AlignmentInput) error {
	if in.Target == "comparison:appointment" || !in.Undo && ((!in.Exclude && in.Target == "") || len(in.PriceIDs) == 0 || len(in.PriceIDs) > 30) {
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	sub, err := readSubmission(ctx, tx, org, id)
	if err != nil {
		return err
	}
	if sub.ProjectID != project {
		return ErrNotFound
	}
	if err = lockComparison(ctx, tx, org, project, sub.PackageID); err != nil {
		return err
	}
	sub, err = lockSubmission(ctx, tx, org, project, id)
	if err != nil {
		return err
	}
	if sub.Version != in.Version {
		return fmt.Errorf("%w: this proposal changed; reload the comparison and try again", ErrConflict)
	}
	if sub.Status != SubmissionReceived && sub.Status != SubmissionNeedsReview {
		return ErrConflict
	}
	if in.Undo {
		var prices, decs []byte
		var version int64
		err = tx.QueryRow(ctx, `SELECT prices,decisions,version FROM comparison_alignment WHERE org_id=$1::uuid AND submission_id=$2::uuid`, org, id).Scan(&prices, &decs, &version)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if version != sub.Version {
			return ErrConflict
		}
		if _, err = tx.Exec(ctx, `DELETE FROM submission_prices WHERE org_id=$1::uuid AND submission_id=$2::uuid`, org, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO submission_prices SELECT * FROM jsonb_populate_recordset(NULL::submission_prices,$1::jsonb)`, prices); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM submission_decisions WHERE org_id=$1::uuid AND submission_id=$2::uuid`, org, id); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO submission_decisions SELECT * FROM jsonb_populate_recordset(NULL::submission_decisions,$1::jsonb)`, decs); err != nil {
			return err
		}
		// Keep the marker: older user mappings may still exist after this undo.
		if _, err = tx.Exec(ctx, `UPDATE comparison_alignment SET version=0 WHERE org_id=$1::uuid AND submission_id=$2::uuid`, org, id); err != nil {
			return err
		}
		return s.finishSubmissionWrite(ctx, tx, org, project, id)
	}
	items, err := readSchedule(ctx, tx, org, sub.PackageID)
	if err != nil {
		return err
	}
	labels := map[string]string{}
	valid := map[string]bool{}
	bases := map[string]bool{}
	rates := map[string]bool{}
	for _, it := range items {
		if !it.Retired && (it.Price() || it.Kind == "rate") {
			rates[it.Key] = it.Kind == "rate"
			labels[it.Key] = it.Label
			valid[it.Key] = true
			bases[it.Key] = it.Base
		}
	}
	if !in.Exclude && (!valid[in.Target] || (rates[in.Target] && len(in.PriceIDs) != 1)) {
		return ErrInvalidInput
	}
	rows, err := tx.Query(ctx, `SELECT id::text,COALESCE(item_key,''),label,amount_cents,state,page,passage,band FROM submission_prices WHERE org_id=$1::uuid AND submission_id=$2::uuid`, org, id)
	if err != nil {
		return err
	}
	var source []proposals.Cell
	var occupied string
	for rows.Next() {
		var c proposals.Cell
		if err = rows.Scan(&c.ID, &c.Key, &c.Label, &c.Cents, &c.State, &c.Page, &c.Passage, &c.Band); err != nil {
			rows.Close()
			return err
		}
		source = append(source, c)
		if c.Key == in.Target && c.State == proposals.StatePriced {
			occupied = c.ID
		}
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}
	selected := map[string]bool{}
	for _, pid := range in.PriceIDs {
		if selected[pid] {
			return ErrInvalidInput
		}
		selected[pid] = true
	}
	if occupied != "" && !selected[occupied] {
		return fmt.Errorf("%w: select the destination's existing fee as well, or choose an unpriced row", ErrConflict)
	}
	if !in.Exclude {
		if err := checkBuilderAlignment(ctx, tx, org, sub.PackageID, source, selected); err != nil {
			return err
		}
	}
	var total int64
	var parts []string
	found := 0
	for _, c := range source {
		if !selected[c.ID] {
			continue
		}
		found++
		if c.Key == "comparison:appointment" {
			return ErrInvalidInput
		}
		if c.State != proposals.StatePriced || c.Cents == nil || *c.Cents < 0 || c.Band == proposals.BandBlank || c.Band == proposals.BandGrey || (!in.Exclude && c.Key != "" && (!valid[c.Key] || (rates[c.Key] && !rates[in.Target]))) {
			return ErrInvalidInput
		}
		if total > math.MaxInt64-*c.Cents {
			return ErrInvalidInput
		}
		total += *c.Cents
		parts = append(parts, fmt.Sprintf("%s: %s ex GST (page %d). %s", c.Label, centsText(c.Cents), c.Page, c.Passage))
	}
	if found != len(selected) {
		return ErrNotFound
	}
	// Full source snapshots remain in history. The displayed passage is bounded
	// by the existing cell limit; it explicitly says when the PDF is needed.
	passage := strings.Join(parts, "\n")
	if len(passage) > 2000 {
		passage = clipText(passage, 1940) + "\nMore source text: see proposal PDF."
	}
	_, err = tx.Exec(ctx, `INSERT INTO comparison_alignment(org_id,submission_id,prices,decisions,version)
 VALUES($1::uuid,$2::uuid,COALESCE((SELECT jsonb_agg(to_jsonb(p)) FROM submission_prices p WHERE org_id=$1::uuid AND submission_id=$2::uuid),'[]'),COALESCE((SELECT jsonb_agg(to_jsonb(d)) FROM submission_decisions d WHERE org_id=$1::uuid AND submission_id=$2::uuid),'[]'),$3)
 ON CONFLICT(org_id,submission_id) DO UPDATE SET prices=EXCLUDED.prices,decisions=EXCLUDED.decisions,version=EXCLUDED.version`, org, id, sub.Version+1)
	if err != nil {
		return err
	}
	for _, c := range source {
		if selected[c.ID] && in.Exclude {
			if _, err = tx.Exec(ctx, `UPDATE submission_prices SET state='excluded',amount_cents=NULL,band='green',decided_by='user',basis='Set aside by you: not part of this comparison' WHERE org_id=$1::uuid AND id=$2::uuid`, org, c.ID); err != nil {
				return err
			}
			if c.Key != "" {
				if err = upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: proposals.PricePrefix + c.Key, Value: proposals.StateExcluded}, true); err != nil {
					return err
				}
			}
			continue
		}
		if !selected[c.ID] || c.Key == in.Target {
			continue
		}
		if c.Key == "" {
			_, err = tx.Exec(ctx, `DELETE FROM submission_prices WHERE org_id=$1::uuid AND id=$2::uuid`, org, c.ID)
		} else {
			state := proposals.StateIncluded
			if bases[c.Key] && !bases[in.Target] {
				state = proposals.StateExcluded
			}
			_, err = tx.Exec(ctx, `UPDATE submission_prices SET state=$4,amount_cents=NULL,band='green',decided_by='user',basis=$3 WHERE org_id=$1::uuid AND id=$2::uuid`, org, c.ID, clipText("Moved to "+labels[in.Target], 500), state)
			if err == nil {
				err = upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: proposals.PricePrefix + c.Key, Value: state}, true)
			}
		}
		if err != nil {
			return err
		}
	}
	if in.Exclude {
		return s.finishSubmissionWrite(ctx, tx, org, project, id)
	}
	_, err = tx.Exec(ctx, `INSERT INTO submission_prices(org_id,id,project_id,submission_id,item_key,label,amount_cents,state,band,decided_by,basis,passage)
 VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,'priced','green','user','Aligned by you',$8)
 ON CONFLICT(org_id,submission_id,item_key) WHERE item_key IS NOT NULL DO UPDATE SET amount_cents=EXCLUDED.amount_cents,state='priced',band='green',decided_by='user',basis=EXCLUDED.basis,passage=EXCLUDED.passage,label=EXCLUDED.label`, org, newID(), project, id, in.Target, labels[in.Target], total, passage)
	if err != nil {
		return err
	}
	if err = upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: proposals.PricePrefix + in.Target, Value: fmt.Sprintf("cents:%d", total)}, true); err != nil {
		return err
	}
	return s.finishSubmissionWrite(ctx, tx, org, project, id)
}
