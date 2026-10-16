package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/costs"
	"sitewise/internal/invoices"
)

// The invoice ledger (docs/plans/2026-10-16-invoice-intake-implementation.md).
// Every row is scoped to org and project; every write takes the project's
// invoice lock, recomputes the invoice from its decisions in the same
// transaction and appends an "invoices" event for live views.

// EventInvoices tells open views of a project's cost plan to refresh.
const EventInvoices = "invoices"

// MaxInvoiceReadAttempts bounds background re-checks after a grey read.
const MaxInvoiceReadAttempts = 3

// InvoiceUpload is one stored file to read as an invoice.
type InvoiceUpload struct {
	SHA256    []byte
	Size      int64
	MediaType string
	Filename  string
	UserID    string
	// NotRead stores the file without reading it (unsupported format).
	NotRead string
}

// InvoiceCreated is the ledger row an upload made, or the one it already had.
type InvoiceCreated struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Created bool   `json:"created"`
}

func lockInvoices(ctx context.Context, tx pgx.Tx, orgID, projectID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || '/' || $2 || '/invoices', 0))`, orgID, projectID)
	return err
}

func invoiceEvent(ctx context.Context, tx pgx.Tx, s *Store, orgID, projectID, invoiceID, status string) error {
	payload, _ := json.Marshal(map[string]string{"project_id": projectID, "invoice_id": invoiceID, "status": status})
	_, err := appendEvent(ctx, s.q.WithTx(tx), orgID, EventInvoices, "", string(payload))
	return err
}

// CreateInvoice stores the association and a ledger row in one transaction.
// The same bytes in the same project are one invoice: a re-drop returns the
// existing row ("Already filed") and makes nothing new.
func (s *Store) CreateInvoice(ctx context.Context, orgID, projectID string, in InvoiceUpload, now time.Time) (InvoiceCreated, error) {
	if len(in.SHA256) != 32 || in.Size < 0 || strings.TrimSpace(in.Filename) == "" || len(in.Filename) > 255 {
		return InvoiceCreated{}, invoices.ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return InvoiceCreated{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockInvoices(ctx, tx, orgID, projectID); err != nil {
		return InvoiceCreated{}, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)`, orgID, projectID).Scan(&exists); err != nil {
		return InvoiceCreated{}, err
	}
	if !exists {
		return InvoiceCreated{}, ErrNotFound
	}
	var fileID string
	var size int64
	err = tx.QueryRow(ctx, `
INSERT INTO files (org_id, id, project_id, sha256, byte_size, media_type)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6)
ON CONFLICT (org_id, project_id, sha256) DO UPDATE SET sha256 = EXCLUDED.sha256
RETURNING id::text, byte_size`, orgID, newID(), projectID, in.SHA256, in.Size, in.MediaType).Scan(&fileID, &size)
	if err != nil {
		return InvoiceCreated{}, err
	}
	if size != in.Size {
		return InvoiceCreated{}, ErrMetadata
	}
	var out InvoiceCreated
	err = tx.QueryRow(ctx, `SELECT id::text, status FROM invoices WHERE org_id=$1::uuid AND project_id=$2::uuid AND file_id=$3::uuid`, orgID, projectID, fileID).Scan(&out.ID, &out.Status)
	if err == nil {
		return out, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return InvoiceCreated{}, err
	}
	out = InvoiceCreated{ID: newID(), Status: invoices.StatusReading, Created: true}
	if in.NotRead != "" {
		out.Status = invoices.StatusNotRead
	}
	var user any
	if in.UserID != "" {
		user = in.UserID
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO invoices (org_id, id, project_id, file_id, filename, status, not_read_reason, period, uploaded_by, created_at, updated_at)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7, $8::date, $9::uuid, $10, $10)`,
		orgID, out.ID, projectID, fileID, in.Filename, out.Status, in.NotRead, invoices.PeriodOf(now), user, now); err != nil {
		return InvoiceCreated{}, err
	}
	if out.Status == invoices.StatusNotRead {
		if _, err := tx.Exec(ctx, `INSERT INTO invoice_decisions (org_id, project_id, invoice_id, field, value, band, decided_by) VALUES ($1::uuid,$2::uuid,$3::uuid,'period',$4,'green','rule')`, orgID, projectID, out.ID, invoices.PeriodOf(now)); err != nil {
			return InvoiceCreated{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO invoice_allocations (org_id, id, project_id, invoice_id, decided_by, band) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,'rule','blank')`, orgID, newID(), projectID, out.ID); err != nil {
			return InvoiceCreated{}, err
		}
	}
	if err := invoiceEvent(ctx, tx, s, orgID, projectID, out.ID, out.Status); err != nil {
		return InvoiceCreated{}, err
	}
	return out, tx.Commit(ctx)
}

// InvoiceReadInput is what one read needs, read in one snapshot.
type InvoiceReadInput struct {
	ProjectID string
	Filename  string
	SHA256    []byte
	Size      int64
	Status    string
	Attempts  int
	CreatedAt time.Time
	// Keep are decisions the read must not replace: the user's, and on a
	// re-check every answer that is not grey.
	Keep    map[string]invoices.Decision
	Context invoices.Context
}

// InvoiceForReading loads an invoice, its file and the project context.
func (s *Store) InvoiceForReading(ctx context.Context, orgID, invoiceID string, recheck bool) (InvoiceReadInput, error) {
	var in InvoiceReadInput
	err := s.pool.QueryRow(ctx, `
SELECT i.project_id::text, i.filename, f.sha256, f.byte_size, i.status, i.read_attempts, i.created_at
FROM invoices i JOIN files f ON f.org_id=i.org_id AND f.id=i.file_id
WHERE i.org_id=$1::uuid AND i.id=$2::uuid`, orgID, invoiceID).Scan(&in.ProjectID, &in.Filename, &in.SHA256, &in.Size, &in.Status, &in.Attempts, &in.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return in, ErrNotFound
	}
	if err != nil {
		return in, err
	}
	decisions, err := s.invoiceDecisions(ctx, s.pool, orgID, []string{invoiceID})
	if err != nil {
		return in, err
	}
	in.Keep = map[string]invoices.Decision{}
	for f, d := range decisions[invoiceID] {
		if d.DecidedBy == invoices.DecidedByUser || recheck && d.Band != invoices.BandGrey {
			in.Keep[f] = d
		}
	}
	pc, _, err := s.invoiceContext(ctx, orgID, in.ProjectID)
	in.Context = pc
	return in, err
}

type querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// projectDecisions reads every invoice decision in a project in one scan.
func projectDecisions(ctx context.Context, q querier, orgID, projectID string) (map[string]map[string]invoices.Decision, error) {
	rows, err := q.Query(ctx, `
SELECT invoice_id::text, field, COALESCE(value,''), band, decided_by, COALESCE(question_version,''), confidence
FROM invoice_decisions WHERE org_id=$1::uuid AND project_id=$2::uuid`, orgID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]map[string]invoices.Decision{}
	for rows.Next() {
		var id string
		var d invoices.Decision
		if err := rows.Scan(&id, &d.Field, &d.Value, &d.Band, &d.DecidedBy, &d.QuestionVersion, &d.Confidence); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = map[string]invoices.Decision{}
		}
		out[id][d.Field] = d
	}
	return out, rows.Err()
}

func (s *Store) invoiceDecisions(ctx context.Context, q querier, orgID string, ids []string) (map[string]map[string]invoices.Decision, error) {
	rows, err := q.Query(ctx, `
SELECT invoice_id::text, field, COALESCE(value,''), band, decided_by, COALESCE(question_version,''), confidence
FROM invoice_decisions WHERE org_id=$1::uuid AND invoice_id = ANY($2::uuid[])`, orgID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]map[string]invoices.Decision{}
	for rows.Next() {
		var id string
		var d invoices.Decision
		if err := rows.Scan(&id, &d.Field, &d.Value, &d.Band, &d.DecidedBy, &d.QuestionVersion, &d.Confidence); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = map[string]invoices.Decision{}
		}
		out[id][d.Field] = d
	}
	return out, rows.Err()
}

// invoiceContext is the project's live packages and posting cost lines, the
// lines' budgets and the plan's tax basis, read from the current draft.
func (s *Store) invoiceContext(ctx context.Context, orgID, projectID string) (invoices.Context, invoicePlan, error) {
	pc := invoices.Context{History: map[string][]string{}}
	plan := invoicePlan{packageTitles: map[string]string{}, packageABN: map[string]string{}}
	// One read-only snapshot. The draft is read directly: the ledger needs
	// lines, budgets and each line's package, not the plan's suggestions.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return pc, plan, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
SELECT id::text, title, kind, COALESCE(discipline_id,''), COALESCE(works_scope,''), COALESCE(supplier_abn,''), retired_at IS NOT NULL
FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND lifecycle_status <> 'cancelled'
ORDER BY title, id`, orgID, projectID)
	if err != nil {
		return pc, plan, err
	}
	var suggest []costs.SuggestPackage
	for rows.Next() {
		var p invoices.Package
		var scope string
		var retired bool
		if err := rows.Scan(&p.ID, &p.Title, &p.Kind, &p.Discipline, &scope, &p.ABN, &retired); err != nil {
			rows.Close()
			return pc, plan, err
		}
		plan.packageTitles[p.ID] = p.Title
		plan.packageABN[p.ID] = p.ABN
		suggest = append(suggest, costs.SuggestPackage{ID: p.ID, Title: p.Title, Kind: p.Kind, DisciplineID: p.Discipline, WorksScope: scope, Retired: retired})
		if !retired {
			pc.Packages = append(pc.Packages, p)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return pc, plan, err
	}
	cp, err := readCostPlan(ctx, tx, orgID, projectID, "")
	if err != nil {
		return pc, plan, err
	}
	plan.taxBasis = cp.TaxBasis
	sections := costs.CostSections(cp.Items)
	cat := s.workCatalog()
	holds, err := awardHolds(ctx, tx, orgID, projectID)
	if err != nil {
		return pc, plan, err
	}
	groups := map[string]bool{}
	for _, it := range cp.Items {
		if _, ok := it.Values["budget"]; ok && !it.Posting {
			groups[it.ID] = true
		}
	}
	for _, it := range cp.Items {
		if !it.Posting {
			// An awarded package's line keeps its budget for its children.
			if groups[it.ID] {
				var budget *int64
				if v := it.Values["budget"]; v.ValueState == "known" && v.Amount != nil {
					if c, ok, err := invoices.ParseCents(string(*v.Amount)); err == nil && ok {
						budget = &c
					}
				}
				plan.lines = append(plan.lines, invoices.BudgetLine{ID: it.ID, Label: it.Label, Section: sections[it.ID], Category: plan.packageTitles[it.PackageID], PackageID: it.PackageID, Budget: budget, Group: true})
			}
			continue
		}
		pkg, category := it.PackageID, ""
		if cat != nil {
			pkg, category = costs.ResolveCostPackage(cat, it, cp.Items, suggest)
		}
		if category == "" {
			category = plan.packageTitles[pkg]
		}
		line := invoices.Line{ID: it.ID, Label: it.Label, Section: sections[it.ID], PackageID: pkg, Category: category, Holds: holds[it.ID]}
		pc.Lines = append(pc.Lines, line)
		var budget *int64
		if v, ok := it.Values["budget"]; ok && v.ValueState == "known" && v.Amount != nil {
			if c, ok, err := invoices.ParseCents(string(*v.Amount)); err == nil && ok {
				budget = &c
			}
		}
		parent := ""
		if groups[it.ParentItemID] {
			parent = it.ParentItemID
		}
		plan.lines = append(plan.lines, invoices.BudgetLine{ID: it.ID, Label: it.Label, Section: line.Section, Category: category, PackageID: pkg, Budget: budget, Parent: parent})
	}
	hist, err := tx.Query(ctx, `
SELECT DISTINCT i.package_id::text, a.cost_item_id::text
FROM invoice_allocations a JOIN invoices i ON i.org_id=a.org_id AND i.project_id=a.project_id AND i.id=a.invoice_id
WHERE a.org_id=$1::uuid AND a.project_id=$2::uuid AND a.decided_by='user' AND a.cost_item_id IS NOT NULL
  AND i.package_id IS NOT NULL AND i.status NOT IN ('void','superseded')`, orgID, projectID)
	if err != nil {
		return pc, plan, err
	}
	for hist.Next() {
		var p, l string
		if err := hist.Scan(&p, &l); err != nil {
			hist.Close()
			return pc, plan, err
		}
		pc.History[p] = append(pc.History[p], l)
	}
	hist.Close()
	if err := hist.Err(); err != nil {
		return pc, plan, err
	}
	return pc, plan, tx.Commit(ctx)
}

type invoicePlan struct {
	taxBasis      string
	lines         []invoices.BudgetLine
	packageTitles map[string]string
	packageABN    map[string]string
}

// CommitInvoiceRead stores one read: decisions (never over a user's), then
// the invoice recomputed from them, in one transaction with its event. A
// read of an invoice no longer waiting is dropped, unless it is a re-check.
func (s *Store) CommitInvoiceRead(ctx context.Context, orgID, invoiceID string, out invoices.Outcome, recheck bool, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var projectID string
	if err := tx.QueryRow(ctx, `SELECT project_id::text FROM invoices WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID).Scan(&projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, err
	}
	if err := lockInvoices(ctx, tx, orgID, projectID); err != nil {
		return false, err
	}
	var status string
	var attempts int
	if err := tx.QueryRow(ctx, `SELECT status, read_attempts FROM invoices WHERE org_id=$1::uuid AND id=$2::uuid FOR UPDATE`, orgID, invoiceID).Scan(&status, &attempts); err != nil {
		return false, err
	}
	if !recheck && status != invoices.StatusReading || recheck && status == invoices.StatusReading {
		return false, nil
	}
	if out.NotRead != "" {
		if _, err := tx.Exec(ctx, `UPDATE invoices SET status='not_read', not_read_reason=$3, retry_after=NULL, updated_at=$4, version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID, out.NotRead, now); err != nil {
			return false, err
		}
		if err := s.applyInvoiceTx(ctx, tx, orgID, projectID, invoiceID); err != nil {
			return false, err
		}
		return true, s.finishInvoiceWrite(ctx, tx, orgID, projectID, invoiceID)
	}
	for _, d := range out.Decisions {
		if err := upsertMachineDecision(ctx, tx, orgID, projectID, invoiceID, d); err != nil {
			return false, err
		}
	}
	var retry any
	if out.Grey && attempts+1 < MaxInvoiceReadAttempts {
		retry = now.Add(30 * time.Second)
	}
	if _, err := tx.Exec(ctx, `
UPDATE invoices SET status = CASE WHEN status='reading' THEN 'filed' ELSE status END, invoice_count=$3, retry_after=$4,
 read_attempts = read_attempts + CASE WHEN $5 THEN 1 ELSE 0 END, updated_at=$6, version=version+1
WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID, out.InvoiceCount, retry, out.Asked, now); err != nil {
		return false, err
	}
	if err := s.applyInvoiceTx(ctx, tx, orgID, projectID, invoiceID); err != nil {
		return false, err
	}
	return true, s.finishInvoiceWrite(ctx, tx, orgID, projectID, invoiceID)
}

// upsertMachineDecision writes a rule or Jev decision. A user's decision is
// final: the conflict update does nothing when the row is the user's.
func upsertMachineDecision(ctx context.Context, tx pgx.Tx, orgID, projectID, invoiceID string, d invoices.Decision) error {
	if d.DecidedBy == invoices.DecidedByUser {
		return nil
	}
	var qv any
	if d.QuestionVersion != "" {
		qv = d.QuestionVersion
	}
	_, err := tx.Exec(ctx, `
INSERT INTO invoice_decisions (org_id, project_id, invoice_id, field, value, band, decided_by, question_version, confidence)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, NULLIF($5,''), $6, $7, $8, $9)
ON CONFLICT (org_id, invoice_id, field) DO UPDATE SET value=EXCLUDED.value, band=EXCLUDED.band, decided_by=EXCLUDED.decided_by,
 question_version=EXCLUDED.question_version, confidence=EXCLUDED.confidence, version=invoice_decisions.version+1, updated_at=now()
WHERE invoice_decisions.decided_by <> 'user'`, orgID, projectID, invoiceID, d.Field, d.Value, d.Band, d.DecidedBy, qv, d.Confidence)
	return err
}

// applyInvoiceTx recomputes the invoice's ledger columns from its decisions
// and keeps a single allocation in step with the line decision. A split
// stays as the user made it.
func (s *Store) applyInvoiceTx(ctx context.Context, tx pgx.Tx, orgID, projectID, invoiceID string) error {
	all, err := s.invoiceDecisions(ctx, tx, orgID, []string{invoiceID})
	if err != nil {
		return err
	}
	by := all[invoiceID]
	if by == nil {
		by = map[string]invoices.Decision{}
	}
	r := invoices.Resolve(by)
	if err := upsertMachineDecision(ctx, tx, orgID, projectID, invoiceID, by[invoices.FieldExGST]); err != nil {
		return err
	}
	var date, period, pkg, abn any
	if r.Date != "" {
		date = r.Date
	}
	if r.Period != "" {
		period = r.Period
	}
	if r.PackageID != "" {
		pkg = r.PackageID
	}
	if r.ABN != "" {
		abn = r.ABN
	}
	var kind, number any
	if r.Kind != "" {
		kind = r.Kind
	}
	if r.Number != "" {
		number = r.Number
	}
	if _, err := tx.Exec(ctx, `
UPDATE invoices SET kind=$3, number=$4, invoice_date=$5::date, period=COALESCE($6::date, period), package_id=$7::uuid, issuer_abn=$8,
 currency=$9, ex_gst_cents=$10, gst_cents=$11, total_cents=$12
WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID, kind, number, date, period, pkg, abn, r.Currency, r.Ex, r.GST, r.Total); err != nil {
		return err
	}
	var count int
	var explicit bool
	if err := tx.QueryRow(ctx, `SELECT count(*), COALESCE(bool_or(amount_cents IS NOT NULL), false) FROM invoice_allocations WHERE org_id=$1::uuid AND invoice_id=$2::uuid`, orgID, invoiceID).Scan(&count, &explicit); err != nil {
		return err
	}
	if count > 1 || explicit {
		return nil
	}
	var line any
	label := ""
	if r.LineID != "" {
		line = r.LineID
		_ = tx.QueryRow(ctx, `SELECT label FROM cost_item_revisions r JOIN cost_plans p ON p.org_id=r.org_id AND p.project_id=r.project_id AND p.draft_version_id=r.plan_version_id
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.cost_item_id=$3::uuid`, orgID, projectID, r.LineID).Scan(&label)
	}
	if count == 0 {
		_, err = tx.Exec(ctx, `INSERT INTO invoice_allocations (org_id, id, project_id, invoice_id, cost_item_id, line_label, decided_by, band) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,$8)`,
			orgID, newID(), projectID, invoiceID, line, label, r.LineBy, r.LineBand)
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE invoice_allocations SET cost_item_id=$3::uuid, line_label=CASE WHEN $3::uuid IS NULL THEN '' ELSE COALESCE(NULLIF($4,''), line_label) END, decided_by=$5, band=$6
WHERE org_id=$1::uuid AND invoice_id=$2::uuid`, orgID, invoiceID, line, label, r.LineBy, r.LineBand)
	return err
}

func (s *Store) finishInvoiceWrite(ctx context.Context, tx pgx.Tx, orgID, projectID, invoiceID string) error {
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM invoices WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID).Scan(&status); err != nil {
		return err
	}
	if err := invoiceEvent(ctx, tx, s, orgID, projectID, invoiceID, status); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// InvoiceRef names one invoice across orgs, for resuming and re-checks.
type InvoiceRef struct {
	OrgID, InvoiceID string
}

// PendingInvoiceReads are invoices stored but not read when the process
// last stopped.
func (s *Store) PendingInvoiceReads(ctx context.Context) ([]InvoiceRef, error) {
	return s.invoiceRefs(ctx, `SELECT org_id::text, id::text FROM invoices WHERE status='reading' ORDER BY created_at LIMIT 1000`)
}

// DueInvoiceRechecks are grey reads whose background re-check is due.
func (s *Store) DueInvoiceRechecks(ctx context.Context, now time.Time) ([]InvoiceRef, error) {
	return s.invoiceRefs(ctx, `SELECT org_id::text, id::text FROM invoices WHERE retry_after IS NOT NULL AND retry_after <= $1 AND status NOT IN ('void','superseded') ORDER BY retry_after LIMIT 50`, now)
}

func (s *Store) invoiceRefs(ctx context.Context, sql string, args ...any) ([]InvoiceRef, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InvoiceRef
	for rows.Next() {
		var r InvoiceRef
		if err := rows.Scan(&r.OrgID, &r.InvoiceID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClaimInvoiceRecheck clears a due re-check so only one worker takes it.
func (s *Store) ClaimInvoiceRecheck(ctx context.Context, ref InvoiceRef, now time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE invoices SET retry_after=NULL WHERE org_id=$1::uuid AND id=$2::uuid AND retry_after IS NOT NULL AND retry_after <= $3`, ref.OrgID, ref.InvoiceID, now)
	return tag.RowsAffected() == 1, err
}

// InvoiceFile is the stored bytes of one invoice in a project.
func (s *Store) InvoiceFile(ctx context.Context, orgID, projectID, invoiceID string) (File, string, error) {
	var f File
	var name string
	err := s.pool.QueryRow(ctx, `
SELECT f.id::text, f.project_id::text, f.sha256, f.byte_size, f.media_type, i.filename
FROM invoices i JOIN files f ON f.org_id=i.org_id AND f.id=i.file_id
WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.id=$3::uuid`, orgID, projectID, invoiceID).Scan(&f.ID, &f.ProjectID, &f.SHA256, &f.ByteSize, &f.MediaType, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return File{}, "", ErrNotFound
	}
	return f, name, err
}

// LedgerView is the Cost plan panel's data: every invoice, its rows and
// flags, the summary and the names a view needs, from one read.
type LedgerView struct {
	Invoices []invoices.Invoice
	Flags    map[string][]invoices.Flag
	Status   map[string]string
	Summary  invoices.Summary
	Packages []invoices.Package
	Lines    []invoices.BudgetLine
	Titles   map[string]string
	TaxBasis string
	Project  string
}

// InvoiceLedger reads the ledger and computes the summary and checks in Go.
func (s *Store) InvoiceLedger(ctx context.Context, orgID, projectID string, today string) (LedgerView, error) {
	var v LedgerView
	if err := s.pool.QueryRow(ctx, `SELECT name FROM projects WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, projectID).Scan(&v.Project); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return v, ErrNotFound
		}
		return v, err
	}
	pc, plan, err := s.invoiceContext(ctx, orgID, projectID)
	if err != nil {
		return v, err
	}
	v.Packages, v.Titles, v.TaxBasis = pc.Packages, plan.packageTitles, plan.taxBasis
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return v, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
SELECT id::text, filename, status, not_read_reason, COALESCE(kind,''), COALESCE(number,''), COALESCE(to_char(invoice_date,'YYYY-MM-DD'),''),
 to_char(period,'YYYY-MM-DD'), COALESCE(package_id::text,''), COALESCE(issuer_abn,''), currency, ex_gst_cents, gst_cents, total_cents,
 invoice_count, COALESCE(superseded_by::text,''), duplicate_ok, created_at
FROM invoices WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY created_at DESC, id DESC`, orgID, projectID)
	if err != nil {
		return v, err
	}
	index := map[string]int{}
	for rows.Next() {
		var inv invoices.Invoice
		if err := rows.Scan(&inv.ID, &inv.Filename, &inv.Status, &inv.NotReadReason, &inv.Kind, &inv.Number, &inv.Date, &inv.Period, &inv.PackageID, &inv.ABN, &inv.Currency,
			&inv.Ex, &inv.GST, &inv.Total, &inv.InvoiceCount, &inv.SupersededBy, &inv.DuplicateOK, &inv.CreatedAt); err != nil {
			rows.Close()
			return v, err
		}
		index[inv.ID] = len(v.Invoices)
		v.Invoices = append(v.Invoices, inv)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return v, err
	}
	arows, err := tx.Query(ctx, `
SELECT invoice_id::text, id::text, COALESCE(cost_item_id::text,''), amount_cents, line_label, decided_by, band
FROM invoice_allocations WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY invoice_id, ordinal`, orgID, projectID)
	if err != nil {
		return v, err
	}
	live := map[string]bool{}
	for _, l := range plan.lines {
		live[l.ID] = true
	}
	archived := map[string]string{}
	for arows.Next() {
		var invID string
		var a invoices.Allocation
		if err := arows.Scan(&invID, &a.ID, &a.LineID, &a.Amount, &a.Label, &a.DecidedBy, &a.Band); err != nil {
			arows.Close()
			return v, err
		}
		if a.LineID != "" && !live[a.LineID] {
			archived[a.LineID] = a.Label
		}
		if i, ok := index[invID]; ok {
			v.Invoices[i].Allocations = append(v.Invoices[i].Allocations, a)
		}
	}
	arows.Close()
	if err := arows.Err(); err != nil {
		return v, err
	}
	decisions, err := projectDecisions(ctx, tx, orgID, projectID)
	if err != nil {
		return v, err
	}
	for id, d := range decisions {
		v.Invoices[index[id]].Decisions = d
	}
	v.Lines = append(v.Lines, plan.lines...)
	for id, label := range archived {
		if label == "" {
			label = "Removed line"
		}
		v.Lines = append(v.Lines, invoices.BudgetLine{ID: id, Label: label, Archived: true})
	}
	v.Summary = invoices.Summarize(v.Lines, v.Invoices, plan.taxBasis)
	v.Flags = invoices.Check(v.Invoices, plan.packageABN, today)
	v.Status = map[string]string{}
	for _, inv := range v.Invoices {
		v.Status[inv.ID] = invoices.Status(inv, v.Flags[inv.ID])
	}
	return v, tx.Commit(ctx)
}

// InvoiceEdit is a user's change to one invoice: field values (an empty
// string clears a field to unknown) and at most one action.
type InvoiceEdit struct {
	Fields map[string]*string `json:"fields"`
	// Action is "", "void", "restore", "replaces" (this invoice supersedes
	// Target) or "keep_both" (a flagged duplicate stays).
	Action string `json:"action"`
	Target string `json:"target"`
}

// ErrInvoiceSplit means a line edit was sent for a split invoice; the split
// is edited as allocations.
var ErrInvoiceSplit = errors.New("invoice is split across lines")

// EditInvoice stores the user's values as final decisions and applies the
// action, then recomputes the invoice, all in one transaction.
func (s *Store) EditInvoice(ctx context.Context, orgID, projectID, invoiceID, userID string, e InvoiceEdit, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockInvoices(ctx, tx, orgID, projectID); err != nil {
		return err
	}
	var status, abn, pkg string
	err = tx.QueryRow(ctx, `SELECT status, COALESCE(issuer_abn,''), COALESCE(package_id::text,'') FROM invoices WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid FOR UPDATE`, orgID, projectID, invoiceID).Scan(&status, &abn, &pkg)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	// An edit while the invoice is still being read is safe: the read commits
	// after it and machine decisions never replace a user's.
	for field, raw := range e.Fields {
		value, err := s.cleanInvoiceValue(ctx, tx, orgID, projectID, field, raw)
		if err != nil {
			return err
		}
		if field == invoices.FieldLine {
			var n int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM invoice_allocations WHERE org_id=$1::uuid AND invoice_id=$2::uuid AND (amount_cents IS NOT NULL)`, orgID, invoiceID).Scan(&n); err != nil {
				return err
			}
			if n > 0 {
				return ErrInvoiceSplit
			}
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO invoice_decisions (org_id, project_id, invoice_id, field, value, band, decided_by)
VALUES ($1::uuid, $2::uuid, $3::uuid, $4, NULLIF($5,''), 'green', 'user')
ON CONFLICT (org_id, invoice_id, field) DO UPDATE SET
 proposed_value = CASE WHEN invoice_decisions.decided_by <> 'user' THEN invoice_decisions.value ELSE invoice_decisions.proposed_value END,
 proposed_by = CASE WHEN invoice_decisions.decided_by <> 'user' THEN invoice_decisions.decided_by ELSE invoice_decisions.proposed_by END,
 proposed_confidence = CASE WHEN invoice_decisions.decided_by <> 'user' THEN invoice_decisions.confidence ELSE invoice_decisions.proposed_confidence END,
 value=EXCLUDED.value, band='green', decided_by='user', question_version=NULL, confidence=NULL, version=invoice_decisions.version+1, updated_at=now()`,
			orgID, projectID, invoiceID, field, value); err != nil {
			return err
		}
		if field == invoices.FieldSupplier && value != "" {
			pkg = value
		}
		if field == invoices.FieldABN {
			abn = value
		}
	}
	// A user who confirms a supplier teaches the supplier rule its ABN.
	if _, ok := e.Fields[invoices.FieldSupplier]; ok && pkg != "" && abn != "" && invoices.ValidABN(abn) {
		if _, err := tx.Exec(ctx, `UPDATE packages SET supplier_abn=$4 WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid AND supplier_abn IS NULL`, orgID, projectID, pkg, abn); err != nil {
			return err
		}
	}
	if len(e.Fields) > 0 && status == invoices.StatusNotRead {
		if _, err := tx.Exec(ctx, `UPDATE invoices SET status='filed' WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID); err != nil {
			return err
		}
	}
	if err := s.invoiceAction(ctx, tx, orgID, projectID, invoiceID, status, e); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE invoices SET updated_at=$3, version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID, now); err != nil {
		return err
	}
	if err := s.applyInvoiceTx(ctx, tx, orgID, projectID, invoiceID); err != nil {
		return err
	}
	// A user value on the line field is the user's allocation too.
	if raw, ok := e.Fields[invoices.FieldLine]; ok && raw != nil {
		if _, err := tx.Exec(ctx, `UPDATE invoice_allocations SET decided_by='user', band='green' WHERE org_id=$1::uuid AND invoice_id=$2::uuid`, orgID, invoiceID); err != nil {
			return err
		}
	}
	return s.finishInvoiceWrite(ctx, tx, orgID, projectID, invoiceID)
}

func (s *Store) invoiceAction(ctx context.Context, tx pgx.Tx, orgID, projectID, invoiceID, status string, e InvoiceEdit) error {
	switch e.Action {
	case "":
		return nil
	case "void":
		if status == invoices.StatusVoid || status == invoices.StatusSuperseded {
			return invoices.ErrInvalid
		}
		_, err := tx.Exec(ctx, `UPDATE invoices SET status='void', retry_after=NULL WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID)
		return err
	case "restore":
		if status != invoices.StatusVoid {
			return invoices.ErrInvalid
		}
		_, err := tx.Exec(ctx, `UPDATE invoices SET status = CASE WHEN not_read_reason <> '' AND ex_gst_cents IS NULL AND NOT EXISTS (SELECT 1 FROM invoice_decisions d WHERE d.org_id=invoices.org_id AND d.invoice_id=invoices.id AND d.decided_by<>'rule') THEN 'not_read' ELSE 'filed' END WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID)
		return err
	case "keep_both":
		_, err := tx.Exec(ctx, `UPDATE invoices SET duplicate_ok=true WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID)
		return err
	case "replaces":
		if e.Target == "" || e.Target == invoiceID {
			return invoices.ErrInvalid
		}
		tag, err := tx.Exec(ctx, `UPDATE invoices SET status='superseded', superseded_by=$4::uuid, retry_after=NULL, version=version+1
WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3 AND status NOT IN ('void','superseded','reading')`, orgID, projectID, e.Target, invoiceID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return ErrNotFound
		}
		return invoiceEvent(ctx, tx, s, orgID, projectID, e.Target, invoices.StatusSuperseded)
	}
	return invoices.ErrInvalid
}

// cleanInvoiceValue validates a user value at the boundary and returns it in
// stored form. A package or line must belong to this project.
func (s *Store) cleanInvoiceValue(ctx context.Context, tx pgx.Tx, orgID, projectID, field string, raw *string) (string, error) {
	if raw == nil {
		return "", invoices.ErrInvalid
	}
	v := strings.TrimSpace(*raw)
	if len(v) > 200 {
		return "", invoices.ErrInvalid
	}
	if v == "" {
		switch field {
		case invoices.FieldKind, invoices.FieldPeriod, invoices.FieldCurrency:
			return "", invoices.ErrInvalid
		}
		return "", nil
	}
	switch field {
	case invoices.FieldKind:
		switch v {
		case "invoice", "credit_note", "quote", "statement", "remittance", "receipt", "payment_schedule", "other":
			return v, nil
		}
	case invoices.FieldNumber:
		if len(v) <= 100 {
			return v, nil
		}
	case invoices.FieldDate:
		if t, err := time.Parse("2006-01-02", v); err == nil && t.Year() >= 2000 && t.Year() <= 2099 {
			return v, nil
		}
	case invoices.FieldPeriod:
		if len(v) == 7 {
			v += "-01"
		}
		if t, err := time.Parse("2006-01-02", v); err == nil && t.Day() == 1 && t.Year() >= 2000 && t.Year() <= 2099 {
			return v, nil
		}
	case invoices.FieldABN:
		if d := invoices.NormalizeABN(v); d != "" {
			return d, nil
		}
	case invoices.FieldTotal, invoices.FieldGST, invoices.FieldExGST:
		if c, ok, err := invoices.ParseCents(v); err == nil && ok {
			return strconv.FormatInt(c, 10), nil
		}
	case invoices.FieldCurrency:
		v = strings.ToUpper(v)
		if len(v) == 3 && strings.Trim(v, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" {
			return v, nil
		}
	case invoices.FieldSupplier:
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM packages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3)`, orgID, projectID, v).Scan(&ok); err != nil {
			return "", err
		}
		if ok {
			return v, nil
		}
		return "", ErrNotFound
	case invoices.FieldLine:
		var ok bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cost_items WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text=$3)`, orgID, projectID, v).Scan(&ok); err != nil {
			return "", err
		}
		if ok {
			return v, nil
		}
		return "", ErrNotFound
	default:
		return "", fmt.Errorf("%w: unknown field %q", invoices.ErrInvalid, field)
	}
	return "", invoices.ErrInvalid
}

// ShareInput is one line of a user's split. Amount is ex GST, as typed.
type ShareInput struct {
	LineID string `json:"line_id"`
	Amount string `json:"amount"`
}

// SplitInvoice replaces the invoice's allocations with the user's. One share
// with no amount is the whole invoice; a split states every amount. A split
// that does not add up is stored and flagged, never refused.
func (s *Store) SplitInvoice(ctx context.Context, orgID, projectID, invoiceID, userID string, shares []ShareInput, now time.Time) error {
	if len(shares) == 0 || len(shares) > 50 {
		return invoices.ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockInvoices(ctx, tx, orgID, projectID); err != nil {
		return err
	}
	var status string
	err = tx.QueryRow(ctx, `SELECT status FROM invoices WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid FOR UPDATE`, orgID, projectID, invoiceID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status == invoices.StatusVoid || status == invoices.StatusSuperseded {
		return invoices.ErrInvalid
	}
	type share struct {
		line, label string
		amount      *int64
	}
	var clean []share
	for _, sh := range shares {
		var c share
		if sh.LineID != "" {
			v := sh.LineID
			line, err := s.cleanInvoiceValue(ctx, tx, orgID, projectID, invoices.FieldLine, &v)
			if err != nil {
				return err
			}
			c.line = line
			_ = tx.QueryRow(ctx, `SELECT label FROM cost_item_revisions r JOIN cost_plans p ON p.org_id=r.org_id AND p.project_id=r.project_id AND p.draft_version_id=r.plan_version_id
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.cost_item_id=$3::uuid`, orgID, projectID, line).Scan(&c.label)
		}
		amount, ok, err := invoices.ParseCents(sh.Amount)
		if err != nil {
			return err
		}
		if ok {
			c.amount = &amount
		} else if len(shares) > 1 {
			return invoices.ErrInvalid
		}
		clean = append(clean, c)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM invoice_allocations WHERE org_id=$1::uuid AND invoice_id=$2::uuid`, orgID, invoiceID); err != nil {
		return err
	}
	for i, c := range clean {
		var line any
		if c.line != "" {
			line = c.line
		}
		if _, err := tx.Exec(ctx, `INSERT INTO invoice_allocations (org_id, id, project_id, invoice_id, cost_item_id, amount_cents, line_label, decided_by, band, ordinal) VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::uuid,$6,$7,'user','green',$8)`,
			orgID, newID(), projectID, invoiceID, line, c.amount, c.label, i); err != nil {
			return err
		}
	}
	// One share is a line choice: record it as the user's line decision too.
	if len(clean) == 1 && clean[0].amount == nil {
		v := clean[0].line
		if _, err := tx.Exec(ctx, `
INSERT INTO invoice_decisions (org_id, project_id, invoice_id, field, value, band, decided_by) VALUES ($1::uuid,$2::uuid,$3::uuid,'line',NULLIF($4,''),'green','user')
ON CONFLICT (org_id, invoice_id, field) DO UPDATE SET value=EXCLUDED.value, band='green', decided_by='user', version=invoice_decisions.version+1, updated_at=now()`, orgID, projectID, invoiceID, v); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE invoices SET updated_at=$3, version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, orgID, invoiceID, now); err != nil {
		return err
	}
	return s.finishInvoiceWrite(ctx, tx, orgID, projectID, invoiceID)
}

// awardHolds names, for each live award line, the schedule items it holds.
func awardHolds(ctx context.Context, tx pgx.Tx, org, project string) (map[string][]string, error) {
	rows, err := tx.Query(ctx, `SELECT s.cost_item_id::text, s.label FROM cost_line_sources s JOIN awards a ON a.org_id=s.org_id AND a.id=s.award_id
WHERE s.org_id=$1::uuid AND s.project_id=$2::uuid AND a.undone_at IS NULL AND s.label <> '' ORDER BY s.ordinal, s.id`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var id, label string
		if err := rows.Scan(&id, &label); err != nil {
			return nil, err
		}
		out[id] = append(out[id], label)
	}
	return out, rows.Err()
}
