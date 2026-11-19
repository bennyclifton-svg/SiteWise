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

	"sitewise/internal/files"
	"sitewise/internal/invoices"
	"sitewise/internal/reports"
	"sitewise/internal/reports/render"
)

// PaymentScheduleTemplate names the schedule's fixed layout; it is Go's,
// not a knowledge template.
const PaymentScheduleTemplate = "tpl.payment-schedule"

// IssuePaymentSchedule freezes a claim's payment schedule in one
// transaction: the claim's report (made on its first issue), an issued
// version with its snapshot, the PDF and its file row (as IssueReport
// writes them). A later issue is a new version linked to the one before.
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
	in, title, err := s.scheduleInput(ctx, tx, org, project, actor, v, *claim, pkg, today)
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
	sections := reports.PaymentScheduleSections(in)
	snap := reports.IssueSnapshot{SchemaVersion: 1, ReportID: reportID, VersionID: versionID, Kind: reports.KindPaymentSchedule, Title: title, ReportingDate: today,
		SourceRevisions: reports.SourceState{TemplateID: PaymentScheduleTemplate, Domains: map[string]int64{}}, Sections: sections, References: []reports.Reference{}, RendererVersion: render.Version}
	raw, hash, err := reports.CanonicalSnapshot(snap)
	if err != nil {
		return out, err
	}
	// A claim of eighty rows may not fit one page; its rows are never cut.
	pdf, err := render.PDFUnlimited(snap)
	if err != nil {
		return out, err
	}
	blob, err := blobs.Put(ctx, bytes.NewReader(pdf.PDF))
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
		org, versionID, reportID, project, number, today, previous, string(sourceRaw), PaymentScheduleTemplate, string(sectionRaw), raw, hash, blob.SHA256, actor).Scan(&issuedAt); err != nil {
		return out, err
	}
	out = IssuedSchedule{ReportID: reportID, VersionID: versionID, Number: number, IssuedOn: today, IssuedAt: issuedAt, SHA256: hash}
	return out, s.finishClaimsWrite(ctx, tx, org, project)
}

// scheduleInput is what the PDF prints, every amount from the claim's
// figures in Go.
func (s *Store) scheduleInput(ctx context.Context, tx pgx.Tx, org, project, actor string, v LedgerView, c ClaimView, pkg, today string) (reports.ScheduleInput, string, error) {
	in := reports.ScheduleInput{Project: v.Project, ClaimNumber: c.Number, Served: c.Served, ReferenceDate: c.ReferenceDate, IssuedOn: today,
		DueOn: c.Schedule.Date, Late: c.Schedule.Date != "" && today > c.Schedule.Date}
	for _, ct := range v.Contracts {
		if ct.PackageID == pkg {
			in.Firm, in.Contract = ct.Firm, ct.Title
		}
	}
	for _, inv := range v.Invoices {
		if inv.ID == c.InvoiceID {
			in.ClaimDate = inv.Date
		}
	}
	client, err := wholeProfileValue(ctx, tx, org, project, "proj.client")
	if err != nil {
		return in, "", err
	}
	in.Client = client
	if err := tx.QueryRow(ctx, `SELECT email FROM users WHERE org_id=$1::uuid AND id=$2::uuid`, org, actor).Scan(&in.IssuedBy); err != nil {
		return in, "", err
	}
	state, err := projectState(ctx, tx, org, project)
	if err != nil {
		return in, "", err
	}
	if cat := s.workCatalog(); cat != nil && state != "" {
		if a, ok := cat.SOPAct(state); ok {
			in.Act = a.Act
			// A section is printed only from a verified row.
			if a.Verified && a.Schedule != nil {
				in.Section = a.Schedule.Section
			}
		}
	}
	labels := map[string]string{}
	for _, ct := range v.Contracts {
		for _, l := range ct.Lines {
			labels[l.ID] = l.Label
		}
	}
	for _, va := range v.Variations {
		labels[va.ID] = va.Number + " " + va.Title
	}
	money := func(p *int64) string {
		if p == nil {
			return "unknown"
		}
		return "$" + invoices.DisplayCents(*p)
	}
	for i, r := range c.Rows {
		label := r.Label
		if l := labels[r.LineID]; l != "" && !strings.EqualFold(l, label) {
			label = l + " (" + r.Label + ")"
		}
		if l := labels[r.VariationID]; l != "" {
			label = l
		}
		row := reports.ScheduleRow{Label: label, Claimed: money(c.Figures.Rows[i].Claimed), Scheduled: money(c.Figures.Rows[i].Scheduled)}
		if c.Figures.Rows[i].BelowClaimed {
			row.Reason = r.Reason
		}
		in.Rows = append(in.Rows, row)
	}
	f := c.Figures
	in.Gross, in.Retention, in.Net, in.GST, in.Payable = money(f.Gross), money(f.Retention), money(f.Net), money(f.GST), money(f.Payable)
	title := "Payment schedule"
	if c.Number != "" {
		title = fmt.Sprintf("Payment schedule for claim %s", c.Number)
	}
	return in, title, nil
}
