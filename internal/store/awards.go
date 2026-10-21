package store

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/costs"
	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
)

// ErrAwardLocked refuses an undo once an invoice is allocated to one of the
// award's lines: the ledger rests on them.
var ErrAwardLocked = errors.New("an invoice is allocated to this award's lines")

// AwardLine is one cost plan line the PM chose: the schedule items (or
// offered extras, by price id) it holds and the name it carries.
type AwardLine struct {
	Label    string   `json:"label"`
	Keys     []string `json:"item_keys"`
	PriceIDs []string `json:"price_ids"`
}

// AwardInput is the award as confirmed. Total is the running total the PM
// saw; the award is refused unless code computes the same to the cent.
type AwardInput struct {
	SubmissionID string      `json:"submission_id"`
	Lines        []AwardLine `json:"lines"`
	Total        string      `json:"total"`
}

// AwardLineView is one line as previewed or written.
type AwardLineView struct {
	CostItemID string   `json:"cost_item_id,omitempty"`
	Label      string   `json:"label"`
	Keys       []string `json:"item_keys"`
	PriceIDs   []string `json:"price_ids"`
	Cents      int64    `json:"-"`
	Invoiced   bool     `json:"invoiced"`
}

// AwardView is a package's live award, or a preview of one.
type AwardView struct {
	ID           string          `json:"id,omitempty"`
	SubmissionID string          `json:"submission_id"`
	FirmName     string          `json:"firm_name"`
	ParentLabel  string          `json:"parent_label"`
	ParentItemID string          `json:"parent_item_id,omitempty"`
	Budget       *int64          `json:"-"`
	Lines        []AwardLineView `json:"lines"`
	Total        int64           `json:"-"`
	CreatedAt    *time.Time      `json:"created_at,omitempty"`
	Locked       bool            `json:"locked"`
}

// awardPlan resolves the confirmed lines against the stored cells: every
// figure comes from the submission, never from the request.
func awardPlan(ctx context.Context, tx pgx.Tx, org, project, pkg string, in AwardInput) (AwardView, Submission, *Firm, string, error) {
	var v AwardView
	sub, err := readSubmission(ctx, tx, org, in.SubmissionID)
	if err != nil {
		return v, sub, nil, "", err
	}
	if sub.ProjectID != project || sub.PackageID != pkg {
		return v, sub, nil, "", ErrNotFound
	}
	isBuilder, err := builderPackage(ctx, tx, org, pkg)
	if err != nil {
		return v, sub, nil, "", err
	}
	if sub.Status != SubmissionReceived && sub.Status != SubmissionNeedsReview {
		return v, sub, nil, "", ErrInvalidInput
	}
	var firm *Firm
	if sub.FirmID == "" {
		return v, sub, nil, "", ErrInvalidInput
	}
	f, err := readFirm(ctx, tx, org, sub.FirmID)
	if err != nil {
		return v, sub, nil, "", err
	}
	firm = &f
	var title string
	if err := tx.QueryRow(ctx, `SELECT title FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3 AND retired_at IS NULL`, org, project, pkg).Scan(&title); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return v, sub, nil, "", ErrNotFound
		}
		return v, sub, nil, "", err
	}
	rows, err := tx.Query(ctx, `SELECT id::text, COALESCE(item_key,''), label, amount_cents, state, decided_by FROM submission_prices WHERE org_id=$1::uuid AND submission_id::text=$2`, org, sub.ID)
	if err != nil {
		return v, sub, nil, "", err
	}
	type priced struct {
		label string
		cents *int64
		state string
		by    string
	}
	byKey, byID := map[string]priced{}, map[string]priced{}
	for rows.Next() {
		var id, key, label, state, by string
		var c *int64
		if err := rows.Scan(&id, &key, &label, &c, &state, &by); err != nil {
			rows.Close()
			return v, sub, nil, "", err
		}
		p := priced{label, c, state, by}
		if key != "" {
			byKey[key] = p
		} else {
			byID[id] = p
		}
	}
	rows.Close()
	if len(in.Lines) == 0 || len(in.Lines) > 100 {
		return v, sub, nil, "", ErrInvalidInput
	}
	quote, err := quotedReview(ctx, tx, org, sub.ID)
	if err != nil {
		return v, sub, nil, "", err
	}
	if _, confirmed := byKey["comparison:appointment"]; !isBuilder && (quote != nil || confirmed) {
		for _, line := range in.Lines {
			if len(line.PriceIDs) != 0 || len(line.Keys) != 1 || line.Keys[0] != "comparison:appointment" || byKey["comparison:appointment"].by != proposals.ByUser {
				return v, sub, nil, "", ErrInvalidInput
			}
		}
	}
	used := map[string]bool{}
	schedule, err := readSchedule(ctx, tx, org, pkg)
	if err != nil {
		return v, sub, nil, "", err
	}
	feeKeys := map[string]bool{}
	for _, it := range schedule {
		if !it.Retired && it.Price() {
			feeKeys[it.Key] = true
		}
	}
	for _, l := range in.Lines {
		if isBuilder && len(l.PriceIDs) > 0 {
			return v, sub, nil, "", ErrInvalidInput
		}
		label := strings.TrimSpace(l.Label)
		if label == "" || len(label) > 200 || len(l.Keys)+len(l.PriceIDs) == 0 {
			return v, sub, nil, "", ErrInvalidInput
		}
		line := AwardLineView{Label: label, Keys: []string{}, PriceIDs: []string{}}
		take := func(id string, p priced, ok bool) error {
			if isBuilder && p.by != proposals.ByUser {
				return ErrInvalidInput
			}
			if !ok || used[id] || p.state != proposals.StatePriced || p.cents == nil {
				return ErrInvalidInput
			}
			used[id] = true
			line.Cents += *p.cents
			return nil
		}
		for _, k := range l.Keys {
			if !feeKeys[k] {
				return v, sub, nil, "", ErrInvalidInput
			}
			p, ok := byKey[k]
			if err := take("k:"+k, p, ok); err != nil {
				return v, sub, nil, "", err
			}
			line.Keys = append(line.Keys, k)
		}
		for _, id := range l.PriceIDs {
			p, ok := byID[id]
			if err := take("p:"+id, p, ok); err != nil {
				return v, sub, nil, "", err
			}
			line.PriceIDs = append(line.PriceIDs, id)
		}
		v.Lines = append(v.Lines, line)
		v.Total += line.Cents
	}
	if in.Total != "" {
		c, ok, err := invoices.ParseCents(in.Total)
		if err != nil || !ok || c != v.Total {
			return v, sub, nil, "", ErrInvalidInput
		}
	}
	v.SubmissionID, v.FirmName, v.ParentLabel = sub.ID, firm.Name, title+": "+firm.Name
	return v, sub, firm, title, nil
}

// PreviewAward checks the PM's lines and returns what would be written.
func (s *Store) PreviewAward(ctx context.Context, org, project, pkg string, in AwardInput) (AwardView, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AwardView{}, err
	}
	defer tx.Rollback(ctx)
	v, _, _, _, err := awardPlan(ctx, tx, org, project, pkg, in)
	if err != nil {
		return v, err
	}
	parent, err := packageFeeLine(ctx, tx, org, project, pkg)
	if err != nil {
		return v, err
	}
	if parent != nil {
		v.Budget = budgetCents(*parent)
	}
	return v, nil
}

// packageFeeLine is the package's top-level fee line in the draft, the
// generated one first. Nil when there is none.
func packageFeeLine(ctx context.Context, tx pgx.Tx, org, project, pkg string) (*costs.Item, error) {
	p, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil || p.Status != "draft" {
		return nil, err
	}
	var found []costs.Item
	for _, it := range p.Items {
		if it.LineKind == "fee" && it.PackageID == pkg && it.ParentItemID == "" {
			found = append(found, it)
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].SourceKey == "fee:"+pkg && found[j].SourceKey != "fee:"+pkg })
	if len(found) == 0 {
		return nil, nil
	}
	return &found[0], nil
}

func budgetCents(it costs.Item) *int64 {
	v, ok := it.Values["budget"]
	if !ok || v.ValueState != "known" || v.Amount == nil {
		return nil
	}
	c, ok, err := invoices.ParseCents(string(*v.Amount))
	if err != nil || !ok {
		return nil
	}
	return &c
}

type parentBefore struct {
	Label   string                 `json:"label"`
	Posting bool                   `json:"posting"`
	Values  map[string]costs.Value `json:"values"`
	Content costs.Content          `json:"content"`
	// Package state the award changed, restored by undo.
	Lifecycle   string `json:"lifecycle_status"`
	SupplierABN string `json:"supplier_abn"`
}

// Award writes the firm's breakdown into the cost plan in one transaction:
// the package's fee line becomes a group that keeps its budget, each chosen
// line becomes a child holding its contract value, each child names the
// schedule items it holds, the package is appointed to the firm, and the
// other submissions are unsuccessful.
func (s *Store) Award(ctx context.Context, org, project, pkg, actor string, in AwardInput) (AwardView, error) {
	tx, p, err := s.costTxMode(ctx, org, project, actor, CostWrite{}, true)
	if err != nil {
		return AwardView{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockComparison(ctx, tx, org, project, pkg); err != nil {
		return AwardView{}, err
	}
	var live bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM awards WHERE org_id=$1::uuid AND package_id::text=$2 AND undone_at IS NULL)`, org, pkg).Scan(&live); err != nil {
		return AwardView{}, err
	}
	if live {
		return AwardView{}, ErrConflict
	}
	v, sub, firm, _, err := awardPlan(ctx, tx, org, project, pkg, in)
	if err != nil {
		return v, err
	}
	parent, err := packageFeeLine(ctx, tx, org, project, pkg)
	if err != nil {
		return v, err
	}
	var before parentBefore
	if err := tx.QueryRow(ctx, `SELECT lifecycle_status, COALESCE(supplier_abn,'') FROM packages WHERE org_id=$1::uuid AND id::text=$2`, org, pkg).Scan(&before.Lifecycle, &before.SupplierABN); err != nil {
		return v, err
	}
	created := parent == nil
	parentID := ""
	if created {
		parentID = newID()
		c := costs.Content{Label: v.ParentLabel, LineKind: "fee", PackageID: pkg, Posting: false, Origin: "user", Meaning: "stated"}
		if err := s.insertCostItem(ctx, tx, org, project, p.ID, actor, parentID, c); err != nil {
			return v, err
		}
	} else {
		if !parent.Posting {
			// A line already split into its own parts is the PM's structure;
			// award does not rebuild it.
			return v, ErrConflict
		}
		parentID = parent.ID
		before.Label, before.Posting, before.Values, before.Content = parent.Label, parent.Posting, parent.Values, parent.Content
		v.Budget = budgetCents(*parent)
		if _, err := tx.Exec(ctx, `DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric<>'budget'`, org, p.ID, parentID); err != nil {
			return v, err
		}
		if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET posting=false, label=$4, quantity=NULL, rate=NULL, unit=NULL, rate_basis=NULL, version=version+1
WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, p.ID, parentID, clipText(v.ParentLabel, 500)); err != nil {
			return v, err
		}
	}
	v.ParentItemID = parentID
	// A group with no budget holds an unknown one: the contract shows, the
	// budget stays blank and no variance is computed.
	if parent == nil || parent.Values["budget"].ValueState == "" {
		if err := writeCostValue(ctx, tx, org, project, p.ID, parentID, actor, "budget", costs.Value{ValueState: "unknown", Origin: "user", Meaning: "allowance"}); err != nil {
			return v, err
		}
	}
	code := ""
	if parent != nil {
		code = parent.Code
	}
	if code == "" {
		code = "A"
	}
	raw, _ := json.Marshal(before)
	awardID := newID()
	if _, err := tx.Exec(ctx, `INSERT INTO awards (org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,parent_before,created_by)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8::jsonb,$9::uuid)`, org, awardID, project, pkg, sub.ID, parentID, created, string(raw), actor); err != nil {
		return v, err
	}
	rationale := "Awarded to " + firm.Name + " (" + sub.Filename + ")"
	for i := range v.Lines {
		l := &v.Lines[i]
		id := newID()
		// Codes keep the PM's order under the group: F01.1, F01.2, ...
		c := costs.Content{ParentItemID: parentID, Code: clipText(code+"."+strconv.Itoa(i+1), 100), Label: l.Label, LineKind: "fee", PackageID: pkg, Posting: true, Origin: "user", Meaning: "stated", Rationale: rationale}
		if err := s.insertCostItem(ctx, tx, org, project, p.ID, actor, id, c); err != nil {
			return v, err
		}
		amount := costs.Money(invoices.FormatCents(l.Cents))
		if err := writeCostValue(ctx, tx, org, project, p.ID, id, actor, "commitment", costs.Value{ValueState: "known", Amount: &amount, Origin: "user", Meaning: "stated", Rationale: rationale}); err != nil {
			return v, err
		}
		for _, k := range l.Keys {
			if _, err := tx.Exec(ctx, `INSERT INTO cost_line_sources (org_id,id,project_id,award_id,cost_item_id,item_key,label,ordinal)
SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,COALESCE((SELECT label FROM schedule_items WHERE org_id=$1::uuid AND package_id=$7::uuid AND key=$6),''),$8`, org, newID(), project, awardID, id, k, pkg, i); err != nil {
				return v, err
			}
		}
		for _, pid := range l.PriceIDs {
			if _, err := tx.Exec(ctx, `INSERT INTO cost_line_sources (org_id,id,project_id,award_id,cost_item_id,price_id,label,ordinal)
SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,COALESCE((SELECT label FROM submission_prices WHERE org_id=$1::uuid AND id=$6::uuid),''),$7`, org, newID(), project, awardID, id, pid, i); err != nil {
				return v, err
			}
		}
		l.CostItemID = id
	}
	abn := firm.ABN
	if abn != "" && !invoices.ValidABN(abn) {
		abn = ""
	}
	if _, err := tx.Exec(ctx, `UPDATE packages SET lifecycle_status='appointed', supplier_abn=COALESCE(NULLIF($3,''), supplier_abn), version=version+1 WHERE org_id=$1::uuid AND id::text=$2`, org, pkg, abn); err != nil {
		return v, err
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status=CASE WHEN id=$3::uuid THEN 'awarded' ELSE 'unsuccessful' END, updated_at=now(), version=version+1
WHERE org_id=$1::uuid AND package_id::text=$2 AND (id=$3::uuid OR status IN ('received','needs_review'))`, org, pkg, sub.ID); err != nil {
		return v, err
	}
	if err := BumpRevision(ctx, tx, org, project, "packages"); err != nil {
		return v, err
	}
	if err := procurementEvent(ctx, tx, s, org, project, sub.ID, "awarded"); err != nil {
		return v, err
	}
	if _, err := s.finishCost(ctx, tx, org, project, p.ID); err != nil {
		return v, err
	}
	return v, nil
}

// UndoAward reverses a package's live award while no invoice rests on its
// lines: the children go, the fee line is restored exactly as it was, the
// package and the other submissions return to where they were.
func (s *Store) UndoAward(ctx context.Context, org, project, pkg, actor string) error {
	tx, p, err := s.costTxMode(ctx, org, project, actor, CostWrite{}, true)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var awardID, sub, parentID string
	var created bool
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT id::text, submission_id::text, parent_item_id::text, parent_created, parent_before FROM awards
WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 AND undone_at IS NULL FOR UPDATE`, org, project, pkg).Scan(&awardID, &sub, &parentID, &created, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	children, err := awardChildren(ctx, tx, org, awardID)
	if err != nil {
		return err
	}
	var invoiced bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invoice_allocations WHERE org_id=$1::uuid AND cost_item_id::text = ANY($2::text[]))`, org, children).Scan(&invoiced); err != nil {
		return err
	}
	if invoiced {
		return ErrAwardLocked
	}
	var before parentBefore
	_ = json.Unmarshal(raw, &before)
	for _, q := range []string{
		`DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id::text = ANY($3::text[])`,
		`DELETE FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id::text = ANY($3::text[])`,
	} {
		if _, err := tx.Exec(ctx, q, org, p.ID, children); err != nil {
			return err
		}
	}
	if created {
		var held bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM invoice_allocations WHERE org_id=$1::uuid AND cost_item_id=$2::uuid)`, org, parentID).Scan(&held); err != nil {
			return err
		}
		if !held {
			for _, q := range []string{
				`DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`,
				`DELETE FROM cost_item_revisions WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`,
			} {
				if _, err := tx.Exec(ctx, q, org, p.ID, parentID); err != nil {
					return err
				}
			}
		}
	} else {
		if _, err := tx.Exec(ctx, `UPDATE cost_item_revisions SET posting=true, label=$4, quantity=$5::text::numeric, rate=$6::text::numeric, unit=NULLIF($7,''), rate_basis=NULLIF($8,''), version=version+1
WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid`, org, p.ID, parentID, before.Label, before.Content.Quantity, before.Content.Rate, before.Content.Unit, before.Content.RateBasis); err != nil {
			return err
		}
		if _, had := before.Values["budget"]; !had {
			if _, err := tx.Exec(ctx, `DELETE FROM cost_values WHERE org_id=$1::uuid AND plan_version_id=$2::uuid AND cost_item_id=$3::uuid AND metric='budget'`, org, p.ID, parentID); err != nil {
				return err
			}
		}
		for m, val := range before.Values {
			if m == "budget" {
				continue
			}
			if _, err := tx.Exec(ctx, `INSERT INTO cost_values(org_id,project_id,plan_version_id,cost_item_id,metric,value_state,amount,low,high,as_of,origin,meaning,rationale,provenance)
VALUES($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7::text::numeric,$8::text::numeric,$9::text::numeric,$10::text::date,$11,$12,$13,$14::jsonb)
ON CONFLICT(org_id,plan_version_id,cost_item_id,metric) DO NOTHING`, org, project, p.ID, parentID, m, val.ValueState, val.Amount, val.Low, val.High, val.AsOf, val.Origin, val.Meaning, val.Rationale, jsonOrEmpty(val.Provenance)); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE awards SET undone_at=now() WHERE org_id=$1::uuid AND id=$2::uuid`, org, awardID); err != nil {
		return err
	}
	lifecycle := before.Lifecycle
	if lifecycle == "" {
		lifecycle = "procuring"
	}
	if _, err := tx.Exec(ctx, `UPDATE packages SET lifecycle_status=$3, supplier_abn=NULLIF($4,''), version=version+1 WHERE org_id=$1::uuid AND id::text=$2`, org, pkg, lifecycle, before.SupplierABN); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE submissions SET status='received', updated_at=now(), version=version+1 WHERE org_id=$1::uuid AND package_id::text=$2 AND status IN ('awarded','unsuccessful')`, org, pkg); err != nil {
		return err
	}
	if err := BumpRevision(ctx, tx, org, project, "packages"); err != nil {
		return err
	}
	if err := procurementEvent(ctx, tx, s, org, project, sub, "award_undone"); err != nil {
		return err
	}
	_, err = s.finishCost(ctx, tx, org, project, p.ID)
	return err
}

func jsonOrEmpty(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func awardChildren(ctx context.Context, tx pgx.Tx, org, awardID string) ([]string, error) {
	rows, err := tx.Query(ctx, `SELECT DISTINCT cost_item_id::text FROM cost_line_sources WHERE org_id=$1::uuid AND award_id=$2::uuid`, org, awardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// ReadAward is the package's live award with its lines as the draft holds
// them, or nil.
func (s *Store) ReadAward(ctx context.Context, org, project, pkg string) (*AwardView, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var v AwardView
	var created time.Time
	err = tx.QueryRow(ctx, `SELECT a.id::text, a.submission_id::text, a.parent_item_id::text, a.created_at, COALESCE(f.name, s.filename)
FROM awards a JOIN submissions s ON s.org_id=a.org_id AND s.id=a.submission_id LEFT JOIN firms f ON f.org_id=s.org_id AND f.id=s.firm_id
WHERE a.org_id=$1::uuid AND a.project_id=$2::uuid AND a.package_id::text=$3 AND a.undone_at IS NULL`, org, project, pkg).Scan(&v.ID, &v.SubmissionID, &v.ParentItemID, &created, &v.FirmName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v.CreatedAt = &created
	plan, err := readCostPlan(ctx, tx, org, project, "")
	if err != nil {
		return nil, err
	}
	items := map[string]costs.Item{}
	for _, it := range plan.Items {
		items[it.ID] = it
	}
	if parent, ok := items[v.ParentItemID]; ok {
		v.ParentLabel = parent.Label
		v.Budget = budgetCents(parent)
	}
	rows, err := tx.Query(ctx, `SELECT s.cost_item_id::text, COALESCE(s.item_key,''), COALESCE(s.price_id::text,''),
 EXISTS (SELECT 1 FROM invoice_allocations i WHERE i.org_id=s.org_id AND i.cost_item_id=s.cost_item_id)
FROM cost_line_sources s WHERE s.org_id=$1::uuid AND s.award_id=$2::uuid ORDER BY s.ordinal, s.id`, org, v.ID)
	if err != nil {
		return nil, err
	}
	index := map[string]int{}
	for rows.Next() {
		var item, key, price string
		var invoiced bool
		if err := rows.Scan(&item, &key, &price, &invoiced); err != nil {
			rows.Close()
			return nil, err
		}
		i, ok := index[item]
		if !ok {
			it := items[item]
			line := AwardLineView{CostItemID: item, Label: it.Label, Keys: []string{}, PriceIDs: []string{}}
			if c := it.Values["commitment"]; c.Amount != nil {
				if n, ok, _ := invoices.ParseCents(string(*c.Amount)); ok {
					line.Cents = n
				}
			}
			index[item] = len(v.Lines)
			i = len(v.Lines)
			v.Lines = append(v.Lines, line)
			v.Total += line.Cents
		}
		if key != "" {
			v.Lines[i].Keys = append(v.Lines[i].Keys, key)
		}
		if price != "" {
			v.Lines[i].PriceIDs = append(v.Lines[i].PriceIDs, price)
		}
		v.Lines[i].Invoiced = v.Lines[i].Invoiced || invoiced
		v.Locked = v.Locked || invoiced
	}
	rows.Close()
	return &v, rows.Err()
}

// CreateFirm adds a confirmed firm to the directory, or returns the one
// that already has its ABN.
func (s *Store) CreateFirm(ctx context.Context, org string, f FirmFields) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var id string
	f.Name = strings.TrimSpace(f.Name)
	abn := invoices.NormalizeABN(f.ABN)
	if f.Name == "" || len(f.Name) > 200 || strings.TrimSpace(f.ABN) != "" && (abn == "" || !invoices.ValidABN(abn)) || len(f.Address) > 500 || len(f.Contact) > 200 || len(f.Email) > 254 || len(f.Phone) > 50 {
		return "", ErrInvalidInput
	}
	if abn != "" {
		if err := tx.QueryRow(ctx, `SELECT id::text FROM firms WHERE org_id=$1::uuid AND abn=$2`, org, abn).Scan(&id); err == nil {
			return id, nil
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}
	id = newID()
	if _, err := tx.Exec(ctx, `INSERT INTO firms (org_id,id,name,name_key,abn,address,contact,email,phone,status) VALUES ($1::uuid,$2::uuid,$3,$4,NULLIF($5,''),$6,$7,$8,$9,'confirmed')`,
		org, id, f.Name, nameKeyOrName(f.Name), abn, f.Address, f.Contact, f.Email, f.Phone); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// SubmissionPackage is the submission's package id, "" if none yet.
func (s *Store) SubmissionPackage(ctx context.Context, org, project, id string) (string, error) {
	var pkg string
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3`, org, project, id).Scan(&pkg)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return pkg, err
}
