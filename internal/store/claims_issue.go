package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/export"
	"sitewise/internal/files"
	"sitewise/internal/invoices"
	"sitewise/internal/reports"
)

// PaymentScheduleTemplate names the schedule's first fixed layout (four
// columns and a reasons paragraph); schedules issued before 27 November
// 2026 carry it. PaymentScheduleSheetTemplate is the owner's format: the
// sheet the screen shows. Both are Go's, not knowledge templates.
const (
	PaymentScheduleTemplate      = "tpl.payment-schedule"
	PaymentScheduleSheetTemplate = "tpl.payment-schedule-sheet"
)

// IssuePaymentSchedule freezes a claim's payment schedule in one
// transaction: the claim's report (made on its first issue), an issued
// version with its snapshot, the PDF and its file row (as IssueReport
// writes them). The snapshot is the sheet's own sections
// (reports.ScheduleSheetSections) and the PDF is those sections in the
// house table style (internal/export), so screen, record and page agree. A later issue is a new version linked to the one before.
// Issue is refused while a row is unassessed, or below claimed with no
// reason. It is allowed late, and the PDF says so. The claim then counts in
// Claimed to Date at its scheduled amounts (worked out at read).
func (s *Store) IssuePaymentSchedule(ctx context.Context, org, project, actor, invoiceID, today string, blobs *files.Store) (IssuedSchedule, error) {
	var out IssuedSchedule
	if blobs == nil {
		return out, errors.New("no file store")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return out, err
	}
	if err := lockInvoices(ctx, tx, org, project); err != nil {
		return out, err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return out, err
	}
	pkg, err := claimPackage(ctx, tx, org, project, invoiceID)
	if err != nil {
		return out, err
	}
	v, err := s.invoiceLedgerTx(ctx, tx, org, project, today)
	if err != nil {
		return out, err
	}
	var claim *ClaimView
	for i := range v.Claims {
		if v.Claims[i].InvoiceID == invoiceID {
			claim = &v.Claims[i]
		}
	}
	if claim == nil {
		return out, claimInvalid("this claim is void or replaced")
	}
	f := claim.Figures
	switch {
	case len(claim.Rows) == 0:
		return out, claimInvalid("the claim has no rows to schedule")
	case f.Unassessed > 0:
		return out, claimInvalid("assess every row first (%d not assessed)", f.Unassessed)
	case f.MissingReasons > 0:
		return out, claimInvalid("give a reason for each row scheduled below what was claimed (%d without one)", f.MissingReasons)
	case f.Payable == nil:
		return out, claimInvalid("the schedule's amounts are not known yet")
	}
	sheet, ok := ClaimSheet(v, invoiceID)
	if !ok {
		return out, claimInvalid("this claim is not on a live contract")
	}
	if sheet.Retention.Open {
		return out, claimInvalid("say what retention this claim holds or returns first; no change this claim is one press")
	}
	head, title, err := s.scheduleHead(ctx, tx, org, project, actor, v, *claim, today)
	if err != nil {
		return out, err
	}
	var reportID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM reports WHERE org_id=$1::uuid AND invoice_id=$2::uuid AND kind='payment_schedule'`, org, invoiceID).Scan(&reportID)
	if errors.Is(err, pgx.ErrNoRows) {
		reportID = newID()
		if _, err := tx.Exec(ctx, `INSERT INTO reports (org_id, id, project_id, kind, package_id, invoice_id, title) VALUES ($1::uuid,$2::uuid,$3::uuid,'payment_schedule',$4::uuid,$5::uuid,$6)`,
			org, reportID, project, pkg, invoiceID, title); err != nil {
			return out, err
		}
	} else if err != nil {
		return out, err
	}
	var number int
	var previous *string
	if err := tx.QueryRow(ctx, `SELECT COALESCE(max(number),0)+1, (SELECT id::text FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid AND status='issued' ORDER BY number DESC LIMIT 1)
FROM report_versions WHERE org_id=$1::uuid AND report_id=$2::uuid`, org, reportID).Scan(&number, &previous); err != nil {
		return out, err
	}
	versionID := newID()
	sections := reports.ScheduleSheetSections(sheet, head)
	snap := reports.IssueSnapshot{SchemaVersion: 1, ReportID: reportID, VersionID: versionID, Kind: reports.KindPaymentSchedule, Title: title, ReportingDate: today,
		SourceRevisions: reports.SourceState{TemplateID: PaymentScheduleSheetTemplate, Domains: map[string]int64{}}, Sections: sections, References: []reports.Reference{}, RendererVersion: export.Version}
	raw, hash, err := reports.CanonicalSnapshot(snap)
	if err != nil {
		return out, err
	}
	// A breakdown of eighty lines runs on to the next page; no row is cut.
	subtitle := strings.Join(nonEmpty(head.Project, head.Contract, head.Contractor), " · ")
	pdf, err := export.PDF(export.FromSections(title, subtitle, today, sections))
	if err != nil {
		return out, err
	}
	blob, err := blobs.Put(ctx, bytes.NewReader(pdf))
	if err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO files(org_id,id,project_id,sha256,byte_size,media_type) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5,'application/pdf') ON CONFLICT(org_id,project_id,sha256) DO NOTHING`, org, newID(), project, blob.SHA256, blob.Size); err != nil {
		return out, err
	}
	sectionRaw, _ := json.Marshal(sections)
	sourceRaw, _ := json.Marshal(snap.SourceRevisions)
	var issuedAt time.Time
	if err := tx.QueryRow(ctx, `INSERT INTO report_versions (org_id, id, report_id, project_id, number, status, reporting_date, previous_issue_id, source_revisions, template_id, template_version,
 sections, snapshot, snapshot_sha256, export_file_sha256, issued_at, issued_by)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,'issued',$6::date,$7::uuid,$8::jsonb,$9,1,$10::jsonb,$11::jsonb,$12,$13,clock_timestamp(),$14::uuid) RETURNING issued_at`,
		org, versionID, reportID, project, number, today, previous, string(sourceRaw), PaymentScheduleSheetTemplate, string(sectionRaw), raw, hash, blob.SHA256, actor).Scan(&issuedAt); err != nil {
		return out, err
	}
	out = IssuedSchedule{ReportID: reportID, VersionID: versionID, Number: number, IssuedOn: today, IssuedAt: issuedAt, SHA256: hash}
	return out, s.finishClaimsWrite(ctx, tx, org, project)
}

// scheduleHead is what the schedule prints beside the sheet's figures: the
// parties as the project's terms resolve them, the claim's own dates, and
// who approved it. The title names the claim as the sheet does.
func (s *Store) scheduleHead(ctx context.Context, tx pgx.Tx, org, project, actor string, v LedgerView, c ClaimView, today string) (reports.ScheduleHead, string, error) {
	h, err := s.scheduleHeader(ctx, tx, org, project, actor, v, c.InvoiceID)
	if err != nil {
		return reports.ScheduleHead{}, "", err
	}
	print := ClaimPrints(v)[c.InvoiceID]
	head := reports.ScheduleHead{Project: h.Project, Address: h.Address, Principal: h.Principal, PrincipalABN: h.PrincipalABN,
		Contractor: h.Contractor, ContractorABN: h.ContractorABN, AssessorOrg: h.AssessorOrg, AssessorNo: h.AssessorCompanyNo, Approver: h.Approver,
		Number: print.Number, Period: print.Period, Received: c.Served, ReferenceDate: c.ReferenceDate, ContractDate: h.ContractDate,
		IssuedOn: today, DueOn: c.Schedule.Date, Late: c.Schedule.Date != "" && today > c.Schedule.Date, Act: h.Act, Note: c.Note}
	for _, ct := range v.Contracts {
		if ct.PackageID == c.PackageID {
			head.Contract = ct.Title
		}
	}
	for i, r := range c.Rows {
		if r.Kind == invoices.RowRetention && i < len(c.Figures.Rows) && c.Figures.Rows[i].BelowClaimed {
			head.RetentionReason = r.Reason
		}
	}
	title := "Payment schedule"
	if print.Number != "" {
		title = fmt.Sprintf("Payment schedule %s", print.Number)
	}
	return head, title, nil
}

func nonEmpty(parts ...string) []string {
	out := []string{}
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// ClaimScheduleToSend is the latest issued version of a claim's payment
// schedule, for Send. A claim with none issued is ErrClaimNotIssued; a
// claim that is not this project's is ErrNotFound.
func (s *Store) ClaimScheduleToSend(ctx context.Context, org, project, claim string) (string, error) {
	if !validUUID(org) || !validUUID(project) || !validUUID(claim) {
		return "", ErrNotFound
	}
	var version *string
	err := s.pool.QueryRow(ctx, `SELECT (SELECT v.id::text FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued'
  WHERE r.org_id=i.org_id AND r.invoice_id=i.id AND r.kind='payment_schedule' ORDER BY v.number DESC LIMIT 1)
FROM invoices i WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.id=$3::uuid AND i.kind='progress_claim' AND i.status NOT IN ('void','superseded')`, org, project, claim).Scan(&version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if version == nil {
		return "", ErrClaimNotIssued
	}
	return *version, nil
}

// ErrClaimNotIssued: the claim's schedule has not been issued, so there is
// nothing to send.
var ErrClaimNotIssued = errors.New("the payment schedule is not issued")

// LinkClaimMessage records the message a claim's issued schedule went out
// on. When that message is sent, RecordSendAttempt sets the day served by
// rule, for the PM to confirm.
func (s *Store) LinkClaimMessage(ctx context.Context, org, project, claim, message string) error {
	tag, err := s.pool.Exec(ctx, `INSERT INTO invoice_decisions (org_id, project_id, invoice_id, field, value, band, decided_by)
SELECT m.org_id, m.project_id, $3::uuid, $5, m.id::text, 'green', 'rule' FROM mail_messages m
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.id=$4::uuid AND m.direction='out'
ON CONFLICT (org_id, invoice_id, field) DO UPDATE SET value=EXCLUDED.value, version=invoice_decisions.version+1, updated_at=now()`,
		org, project, claim, message, invoices.FieldScheduleMessage)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
