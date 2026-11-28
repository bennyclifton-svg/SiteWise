package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
)

// The variation register (docs/plans/2026-11-17-variations-progress-claims.md,
// Task 3). A variation belongs to a live contract (an award) and to one of
// its lines; every write is captured for Undo, bumps the cost revision and
// tells open views of the cost plan to refresh.

var (
	// ErrInvalidVariation is input the PM must fix; its message says what.
	ErrInvalidVariation = errors.New("invalid variation")
	// ErrVariationNumber: the contract already has a variation with that number.
	ErrVariationNumber = errors.New("that variation number is already used on this contract")
)

func variationInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidVariation, fmt.Sprintf(format, args...))
}

// Contract is one live award as the Variations tab shows it.
type Contract struct {
	AwardID   string
	PackageID string
	Title     string
	Kind      string // the package's kind: works, services or supply
	Firm      string
	// Lines are the award's cost lines in award order, with the approved
	// contract each holds (nil when unknown).
	Lines []ContractLine
	// Breakdown is what a payment schedule prints: the stored lines, or the
	// cost lines while none is stored. Gaps are the cost lines whose
	// breakdown does not add up to their contract value.
	Breakdown []BreakdownLine
	Gaps      []BreakdownGap
	// The contract's own periods; nil days is not entered.
	ScheduleDays  *int
	ScheduleBasis string
	PaymentDays   *int
	PaymentBasis  string
	// What a payment schedule prints of the contract, typed once: its own
	// date, and its retention as a rate (basis points) of a base. "" and
	// nil are not entered; SiteWise has no default rate.
	ContractDate string
	// ActApplies is the PM's switch for naming the Act; nil is the default
	// by kind (ActOn).
	ActApplies    *bool
	RetentionBP   *int
	RetentionBase string
	// The awarded firm's ABN and address, as the firm's record holds them.
	FirmABN, FirmEmail string
}

// ContractLine is one cost line a contract holds.
type ContractLine struct {
	ID     string
	Label  string
	Amount *int64
}

// Claims are taken on every live contract, consultants included (payment
// schedules plan, owner decision 7). A services contract's claim is its own
// tax invoice unless the PM says otherwise.
func (c Contract) Claims() bool { return true }

// Services is a consultant's or other services contract.
func (c Contract) Services() bool { return c.Kind != "works" }

// ActOn says the contract's schedules name the Security of Payment Act:
// the PM's switch where set, else on for works and off for services (owner
// decision 8).
func (c Contract) ActOn() bool {
	if c.ActApplies != nil {
		return *c.ActApplies
	}
	return !c.Services()
}

// readContracts reads every live award with its lines from the draft plan.
func readContracts(ctx context.Context, q querier, org, project string) ([]Contract, error) {
	rows, err := q.Query(ctx, `
SELECT a.id::text, a.package_id::text, p.title, p.kind, COALESCE(f.name, ''), a.schedule_days, a.schedule_basis, a.payment_days, a.payment_basis,
 COALESCE(a.contract_date::text, ''), a.retention_bp, COALESCE(a.retention_base, ''), COALESCE(f.abn, ''), COALESCE(f.email, ''), a.act_applies
FROM awards a JOIN packages p ON p.org_id=a.org_id AND p.id=a.package_id
JOIN submissions s ON s.org_id=a.org_id AND s.id=a.submission_id LEFT JOIN firms f ON f.org_id=s.org_id AND f.id=s.firm_id
WHERE a.org_id=$1::uuid AND a.project_id=$2::uuid AND a.undone_at IS NULL
ORDER BY p.title, a.id`, org, project)
	if err != nil {
		return nil, err
	}
	var out []Contract
	index := map[string]int{}
	for rows.Next() {
		var c Contract
		var sd, pd *int16
		var bp *int32
		if err := rows.Scan(&c.AwardID, &c.PackageID, &c.Title, &c.Kind, &c.Firm, &sd, &c.ScheduleBasis, &pd, &c.PaymentBasis,
			&c.ContractDate, &bp, &c.RetentionBase, &c.FirmABN, &c.FirmEmail, &c.ActApplies); err != nil {
			rows.Close()
			return nil, err
		}
		if sd != nil {
			n := int(*sd)
			c.ScheduleDays = &n
		}
		if pd != nil {
			n := int(*pd)
			c.PaymentDays = &n
		}
		if bp != nil {
			n := int(*bp)
			c.RetentionBP = &n
		}
		index[c.AwardID] = len(out)
		out = append(out, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}
	// Each line once, in the award's order, with its label and contract
	// amount as the draft holds them.
	lines, err := q.Query(ctx, `
SELECT s.award_id::text, s.cost_item_id::text, COALESCE(r.label, s.label), v.amount::text, min(s.ordinal) AS ord
FROM cost_line_sources s JOIN awards a ON a.org_id=s.org_id AND a.id=s.award_id AND a.undone_at IS NULL
LEFT JOIN cost_plans cp ON cp.org_id=s.org_id AND cp.project_id=s.project_id
LEFT JOIN cost_item_revisions r ON r.org_id=s.org_id AND r.plan_version_id=cp.draft_version_id AND r.cost_item_id=s.cost_item_id
LEFT JOIN cost_values v ON v.org_id=s.org_id AND v.plan_version_id=cp.draft_version_id AND v.cost_item_id=s.cost_item_id AND v.metric='commitment' AND v.value_state='known'
WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid
GROUP BY s.award_id, s.cost_item_id, COALESCE(r.label, s.label), v.amount
ORDER BY s.award_id, ord, s.cost_item_id`, org, project)
	if err != nil {
		return nil, err
	}
	defer lines.Close()
	for lines.Next() {
		var award, id, label string
		var amount *string
		var ord int
		if err := lines.Scan(&award, &id, &label, &amount, &ord); err != nil {
			return nil, err
		}
		i, ok := index[award]
		if !ok {
			continue
		}
		l := ContractLine{ID: id, Label: label}
		if amount != nil {
			if c, ok, err := invoices.ParseCents(*amount); err == nil && ok {
				l.Amount = &c
			}
		}
		out[i].Lines = append(out[i].Lines, l)
	}
	if err := lines.Err(); err != nil {
		return nil, err
	}
	lines.Close()
	stored, err := readBreakdown(ctx, q, org, project)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].setBreakdown(stored[out[i].AwardID])
	}
	return out, nil
}

// readVariations reads a project's variations with their linked documents.
func readVariations(ctx context.Context, q querier, org, project string) ([]invoices.Variation, error) {
	rows, err := q.Query(ctx, `
SELECT v.id::text, v.package_id::text, v.award_id::text, v.cost_item_id::text, v.number, v.title, v.details, v.status,
 v.claimed_cents, v.estimate_cents, v.approved_cents, v.time_claimed_days, COALESCE(to_char(v.submitted_on,'YYYY-MM-DD'),''),
 COALESCE(to_char(v.decided_on,'YYYY-MM-DD'),''), v.version, v.created_at,
 COALESCE((SELECT array_agg(d.document_id::text ORDER BY d.document_id) FROM variation_documents d WHERE d.org_id=v.org_id AND d.variation_id=v.id), '{}'),
 COALESCE((SELECT array_agg(doc.filename ORDER BY d.document_id) FROM variation_documents d JOIN documents doc ON doc.org_id=d.org_id AND doc.id=d.document_id WHERE d.org_id=v.org_id AND d.variation_id=v.id), '{}')
FROM variations v WHERE v.org_id=$1::uuid AND v.project_id=$2::uuid
ORDER BY v.package_id, v.created_at, v.id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []invoices.Variation
	for rows.Next() {
		var v invoices.Variation
		if err := rows.Scan(&v.ID, &v.PackageID, &v.AwardID, &v.LineID, &v.Number, &v.Title, &v.Details, &v.Status, &v.Claimed, &v.Estimate,
			&v.Approved, &v.TimeDays, &v.SubmittedOn, &v.DecidedOn, &v.Version, &v.CreatedAt, &v.Documents, &v.DocumentNames); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// VariationInput is a new variation or an edit. A nil field is left as it
// is; an empty string clears a value to unknown. Money is as typed, ex GST.
type VariationInput struct {
	Package     string   `json:"package"`
	Line        *string  `json:"line"`
	Number      *string  `json:"number"`
	Title       *string  `json:"title"`
	Details     *string  `json:"details"`
	Status      *string  `json:"status"`
	Claimed     *string  `json:"claimed"`
	Estimate    *string  `json:"estimate"`
	Approved    *string  `json:"approved"`
	TimeDays    *string  `json:"time_claimed_days"`
	SubmittedOn *string  `json:"submitted_on"`
	DecidedOn   *string  `json:"decided_on"`
	Link        []string `json:"link_documents"`
	Unlink      []string `json:"unlink_documents"`
	Version     int64    `json:"version"`
}

var variationNumber = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ./-]{0,19}$`)

// AddVariation records a variation on a package's live contract. The
// number is the PM's or the next V01; the line is the PM's pick, else the
// contract's only line.
func (s *Store) AddVariation(ctx context.Context, org, project, actor string, in VariationInput, now time.Time) (invoices.Variation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return invoices.Variation{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return invoices.Variation{}, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return invoices.Variation{}, err
	}
	contracts, err := readContracts(ctx, tx, org, project)
	if err != nil {
		return invoices.Variation{}, err
	}
	var c *Contract
	for i := range contracts {
		if contracts[i].PackageID == in.Package {
			c = &contracts[i]
		}
	}
	if c == nil {
		return invoices.Variation{}, ErrNotFound
	}
	v := invoices.Variation{ID: newID(), PackageID: c.PackageID, AwardID: c.AwardID, Status: invoices.VariationSubmitted}
	if in.Line == nil || strings.TrimSpace(*in.Line) == "" {
		if len(c.Lines) != 1 {
			return v, variationInvalid("pick the contract line this variation changes")
		}
		v.LineID = c.Lines[0].ID
	}
	if in.Number == nil || strings.TrimSpace(*in.Number) == "" {
		var n int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(substring(number FROM '^[Vv]([0-9]{1,4})$')::int), 0) + 1 FROM variations WHERE org_id=$1::uuid AND package_id=$2::uuid`, org, c.PackageID).Scan(&n); err != nil {
			return v, err
		}
		v.Number = fmt.Sprintf("V%02d", n)
	}
	if in.Title == nil || strings.TrimSpace(*in.Title) == "" {
		return v, variationInvalid("a variation needs a description")
	}
	if err := applyVariationInput(&v, in, *c); err != nil {
		return v, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO variations (org_id, id, project_id, package_id, award_id, cost_item_id, number, title, details, status, claimed_cents, estimate_cents,
 approved_cents, time_claimed_days, submitted_on, decided_on, created_by, created_at, updated_at)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6::uuid,$7,$8,$9,$10,$11,$12,$13,$14,NULLIF($15,'')::date,NULLIF($16,'')::date,$17::uuid,$18,$18)`,
		org, v.ID, project, v.PackageID, v.AwardID, v.LineID, v.Number, v.Title, v.Details, v.Status, v.Claimed, v.Estimate, v.Approved, v.TimeDays,
		v.SubmittedOn, v.DecidedOn, actor, now); err != nil {
		return v, variationWriteError(err)
	}
	if err := linkVariationDocuments(ctx, tx, org, project, v.ID, in.Link, in.Unlink); err != nil {
		return v, err
	}
	v.Version = 1
	return v, s.finishClaimsWrite(ctx, tx, org, project)
}

// EditVariation changes one variation. Approving needs an amount (zero or
// negative is allowed); leaving approved clears it, so the amount and the
// status never disagree.
func (s *Store) EditVariation(ctx context.Context, org, project, actor, id string, in VariationInput, now time.Time) (invoices.Variation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return invoices.Variation{}, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return invoices.Variation{}, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return invoices.Variation{}, err
	}
	all, err := readVariations(ctx, tx, org, project)
	if err != nil {
		return invoices.Variation{}, err
	}
	var v *invoices.Variation
	for i := range all {
		if all[i].ID == id {
			v = &all[i]
		}
	}
	if v == nil {
		return invoices.Variation{}, ErrNotFound
	}
	if in.Version != 0 && in.Version != v.Version {
		return *v, ErrVersionConflict
	}
	contracts, err := readContracts(ctx, tx, org, project)
	if err != nil {
		return *v, err
	}
	var c Contract
	for _, k := range contracts {
		if k.AwardID == v.AwardID {
			c = k
		}
	}
	if c.AwardID == "" {
		return *v, variationInvalid("its contract's award was undone; the variation can no longer change")
	}
	if in.Package != "" && in.Package != v.PackageID {
		return *v, variationInvalid("a variation stays on its contract")
	}
	if err := applyVariationInput(v, in, c); err != nil {
		return *v, err
	}
	if _, err := tx.Exec(ctx, `
UPDATE variations SET cost_item_id=$3::uuid, number=$4, title=$5, details=$6, status=$7, claimed_cents=$8, estimate_cents=$9, approved_cents=$10,
 time_claimed_days=$11, submitted_on=NULLIF($12,'')::date, decided_on=NULLIF($13,'')::date, updated_at=$14, version=version+1
WHERE org_id=$1::uuid AND id=$2::uuid`, org, v.ID, v.LineID, v.Number, v.Title, v.Details, v.Status, v.Claimed, v.Estimate, v.Approved,
		v.TimeDays, v.SubmittedOn, v.DecidedOn, now); err != nil {
		return *v, variationWriteError(err)
	}
	if err := linkVariationDocuments(ctx, tx, org, project, v.ID, in.Link, in.Unlink); err != nil {
		return *v, err
	}
	v.Version++
	return *v, s.finishClaimsWrite(ctx, tx, org, project)
}

// applyVariationInput validates the fields at the boundary and sets them.
func applyVariationInput(v *invoices.Variation, in VariationInput, c Contract) error {
	if in.Line != nil && strings.TrimSpace(*in.Line) != "" {
		line := strings.TrimSpace(*in.Line)
		found := false
		for _, l := range c.Lines {
			found = found || l.ID == line
		}
		if !found {
			return variationInvalid("that line is not one of %s's contract lines", c.Title)
		}
		v.LineID = line
	}
	if in.Number != nil && strings.TrimSpace(*in.Number) != "" {
		n := strings.TrimSpace(*in.Number)
		if !variationNumber.MatchString(n) {
			return variationInvalid("a variation number is up to 20 letters, digits, spaces, dots, slashes or dashes")
		}
		v.Number = n
	}
	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" || len(t) > 200 {
			return variationInvalid("a description is 1 to 200 characters")
		}
		v.Title = t
	}
	if in.Details != nil {
		if len(*in.Details) > 4000 {
			return variationInvalid("details are at most 4,000 characters")
		}
		v.Details = strings.TrimSpace(*in.Details)
	}
	for _, m := range []struct {
		raw  *string
		into **int64
		name string
	}{{in.Claimed, &v.Claimed, "claimed"}, {in.Estimate, &v.Estimate, "estimate"}, {in.Approved, &v.Approved, "approved"}} {
		if m.raw == nil {
			continue
		}
		c, ok, err := invoices.ParseCents(*m.raw)
		if err != nil {
			return variationInvalid("the %s amount is not an amount", m.name)
		}
		if ok {
			*m.into = &c
		} else {
			*m.into = nil
		}
	}
	if in.TimeDays != nil {
		t := strings.TrimSpace(*in.TimeDays)
		if t == "" {
			v.TimeDays = nil
		} else {
			n, err := strconv.Atoi(t)
			if err != nil || n < 0 || n > 3650 {
				return variationInvalid("time claimed is a whole number of days")
			}
			v.TimeDays = &n
		}
	}
	for _, d := range []struct {
		raw  *string
		into *string
	}{{in.SubmittedOn, &v.SubmittedOn}, {in.DecidedOn, &v.DecidedOn}} {
		if d.raw == nil {
			continue
		}
		t := strings.TrimSpace(*d.raw)
		if t != "" {
			if p, err := time.Parse("2006-01-02", t); err != nil || p.Year() < 2000 || p.Year() > 2099 {
				return variationInvalid("a date is YYYY-MM-DD")
			}
		}
		*d.into = t
	}
	if in.Status != nil {
		st := strings.TrimSpace(*in.Status)
		known := false
		for _, k := range invoices.VariationStatuses {
			known = known || k == st
		}
		if !known {
			return variationInvalid("a status is proposed, submitted, approved, rejected or withdrawn")
		}
		v.Status = st
	}
	switch {
	case v.Status == invoices.VariationApproved && v.Approved == nil:
		return variationInvalid("approving needs the approved amount")
	case v.Status != invoices.VariationApproved && in.Approved != nil && v.Approved != nil:
		return variationInvalid("an approved amount is set by approving the variation")
	case v.Status != invoices.VariationApproved:
		v.Approved = nil
	}
	if len(in.Link)+len(in.Unlink) > 50 {
		return variationInvalid("link at most 50 documents at once")
	}
	return nil
}

func variationWriteError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrVariationNumber
	}
	return err
}

// linkVariationDocuments adds and removes document links; each document
// must be in the project.
func linkVariationDocuments(ctx context.Context, tx pgx.Tx, org, project, variation string, link, unlink []string) error {
	for _, d := range link {
		if !uuidText.MatchString(d) {
			return ErrNotFound
		}
		tag, err := tx.Exec(ctx, `INSERT INTO variation_documents (org_id, project_id, variation_id, document_id)
SELECT $1::uuid, $2::uuid, $3::uuid, id FROM documents WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$4::uuid
ON CONFLICT DO NOTHING`, org, project, variation, d)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			var linked bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM variation_documents WHERE org_id=$1::uuid AND variation_id=$2::uuid AND document_id=$3::uuid)`, org, variation, d).Scan(&linked); err != nil {
				return err
			}
			if !linked {
				return ErrNotFound
			}
		}
	}
	for _, d := range unlink {
		if !uuidText.MatchString(d) {
			return ErrNotFound
		}
		if _, err := tx.Exec(ctx, `DELETE FROM variation_documents WHERE org_id=$1::uuid AND variation_id=$2::uuid AND document_id=$3::uuid`, org, variation, d); err != nil {
			return err
		}
	}
	return nil
}

var uuidText = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// finishClaimsWrite bumps the cost revision and tells open cost plan views
// (the grid and the ledger) to refresh, then commits.
func (s *Store) finishClaimsWrite(ctx context.Context, tx pgx.Tx, org, project string) error {
	if err := s.touchClaims(ctx, tx, org, project); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// touchClaims is the claims area's finisher, after a write and after Undo.
func (s *Store) touchClaims(ctx context.Context, tx pgx.Tx, org, project string) error {
	if err := BumpRevision(ctx, tx, org, project, "costs"); err != nil {
		return err
	}
	raw, _ := json.Marshal(map[string]any{"project_id": project, "revision": 0})
	if _, err := appendEvent(ctx, s.q.WithTx(tx), org, "costs", "", string(raw)); err != nil {
		return err
	}
	return invoiceEvent(ctx, tx, s, org, project, "", "")
}

// variationsHold says why an award cannot be undone: a variation or a
// progress claim names it.
func variationsHold(ctx context.Context, tx pgx.Tx, org, awardID, pkg string) (bool, error) {
	var held bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM variations WHERE org_id=$1::uuid AND award_id=$2::uuid)
 OR EXISTS(SELECT 1 FROM invoices WHERE org_id=$1::uuid AND package_id::text=$3 AND kind='progress_claim' AND status NOT IN ('void','superseded'))`, org, awardID, pkg).Scan(&held)
	return held, err
}

// attachVariations puts each draft line's variation sums on it as two
// values, which costs.Forecast and the grid's two variation columns read.
// Only a draft is read this way: a frozen baseline keeps what it froze.
func attachVariations(ctx context.Context, q querier, org, project string, p *costs.Plan) error {
	if p.Status != "draft" || len(p.Items) == 0 {
		return nil
	}
	vs, err := readVariations(ctx, q, org, project)
	if err != nil || len(vs) == 0 {
		return err
	}
	applyVariations(p.Items, vs)
	return nil
}

// applyVariations sets the two variation values on the lines that have any.
func applyVariations(items []costs.Item, vs []invoices.Variation) {
	byLine := invoices.VariationsByLine(vs)
	for i := range items {
		l, ok := byLine[items[i].ID]
		if !ok || !items[i].Posting {
			continue
		}
		if l.HasApproved {
			items[i].Values[costs.MetricApprovedVariations] = centsValue(l.Approved, "Approved variations on this line")
		}
		switch {
		case l.Unknown > 0:
			items[i].Values[costs.MetricForecastVariations] = costs.Value{ValueState: "unknown", Origin: "calculation", Rationale: fmt.Sprintf("%d open variation(s) with no estimate or claimed amount", l.Unknown)}
		case l.HasForecast:
			why := "Open variations at the PM's estimate"
			if l.ClaimedBasis > 0 {
				why = fmt.Sprintf("Open variations at the PM's estimate; %d at the claimed amount", l.ClaimedBasis)
			}
			items[i].Values[costs.MetricForecastVariations] = centsValue(l.Forecast, why)
		}
	}
}

func centsValue(c int64, why string) costs.Value {
	m := costs.Money(invoices.FormatCents(c))
	return costs.Value{ValueState: "known", Amount: &m, Origin: "calculation", Meaning: "stated", Rationale: why}
}

// VariationRegister is every live contract and every variation, read on
// one snapshot, for operations that name a contract, a line or a variation.
func (s *Store) VariationRegister(ctx context.Context, org, project string) ([]Contract, []invoices.Variation, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, org, project).Scan(&exists); err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, ErrNotFound
	}
	cs, err := readContracts(ctx, tx, org, project)
	if err != nil {
		return nil, nil, err
	}
	vs, err := readVariations(ctx, tx, org, project)
	if err != nil {
		return nil, nil, err
	}
	return cs, vs, tx.Commit(ctx)
}
