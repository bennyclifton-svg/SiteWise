package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"sitewise/internal/invoices"
	"sitewise/internal/reports"
)

// Progress claims on the ledger (docs/plans/2026-11-17-variations-progress-claims.md,
// Tasks 6 to 9). A claim is an invoice of kind progress_claim; its rows are
// progress_claim_lines; its payment schedule is a report of its own kind.
// Figures and dates are worked out at read, in invoices.ClaimLedger and
// invoices.ScheduleDue, from what is stored here.

var (
	// ErrClaimIssued: the claim's payment schedule is issued, so its rows
	// are frozen.
	ErrClaimIssued = errors.New("this claim's payment schedule is issued; its rows can no longer change")
	// ErrInvalidClaim is input the PM must fix; its message says what.
	ErrInvalidClaim = errors.New("invalid claim")
)

func claimInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidClaim, fmt.Sprintf(format, args...))
}

// IssuedSchedule is one issued version of a claim's payment schedule.
type IssuedSchedule struct {
	ReportID  string    `json:"report_id"`
	VersionID string    `json:"version_id"`
	Number    int       `json:"number"`
	IssuedOn  string    `json:"issued_on"`
	IssuedAt  time.Time `json:"-"`
	SHA256    string    `json:"snapshot_sha256"`
}

// ClaimView is one claim on the Variations and claims tab.
type ClaimView struct {
	InvoiceID       string
	PackageID       string
	Number          string
	Served          string
	ServedConfirmed bool
	ReferenceDate   string
	ScheduleServed  string
	Rows            []invoices.ClaimRow
	Figures         invoices.ClaimFigures
	Schedule        invoices.DueDate
	Payment         invoices.DueDate
	Alert           string
	Late            bool
	Issued          []IssuedSchedule
	SettledBy       []string // tax invoices linked to it
}

func readClaimRows(ctx context.Context, q querier, org, project string) (map[string][]invoices.ClaimRow, error) {
	rows, err := q.Query(ctx, `
SELECT invoice_id::text, id::text, ordinal, kind, label, COALESCE(page,0), COALESCE(cost_item_id::text,''), COALESCE(variation_id::text,''), mapped_by, band,
 to_date_cents, previous_cents, this_claim_cents, assessed_to_date_cents, reason, version
FROM progress_claim_lines WHERE org_id=$1::uuid AND project_id=$2::uuid ORDER BY invoice_id, ordinal`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]invoices.ClaimRow{}
	for rows.Next() {
		var inv string
		var r invoices.ClaimRow
		if err := rows.Scan(&inv, &r.ID, &r.Ordinal, &r.Kind, &r.Label, &r.Page, &r.LineID, &r.VariationID, &r.MappedBy, &r.Band,
			&r.ToDate, &r.Previous, &r.ThisClaim, &r.Assessed, &r.Reason, &r.Version); err != nil {
			return nil, err
		}
		out[inv] = append(out[inv], r)
	}
	return out, rows.Err()
}

func readSchedules(ctx context.Context, q querier, org, project string) (map[string][]IssuedSchedule, error) {
	rows, err := q.Query(ctx, `
SELECT r.invoice_id::text, r.id::text, v.id::text, v.number, v.reporting_date::text, v.issued_at, v.snapshot_sha256
FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued'
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.kind='payment_schedule' ORDER BY r.invoice_id, v.number`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]IssuedSchedule{}
	for rows.Next() {
		var inv string
		var s IssuedSchedule
		if err := rows.Scan(&inv, &s.ReportID, &s.VersionID, &s.Number, &s.IssuedOn, &s.IssuedAt, &s.SHA256); err != nil {
			return nil, err
		}
		out[inv] = append(out[inv], s)
	}
	return out, rows.Err()
}

// projectState is the project's state or territory as the Profile applies
// it, as Program's calendar reads it; "" when not known.
func projectState(ctx context.Context, q querier, org, project string) (string, error) {
	return wholeProfileValue(ctx, q, org, project, "det.state")
}

// wholeProfileValue is one whole-project Profile value as it applies, or
// "" when it does not.
func wholeProfileValue(ctx context.Context, q querier, org, project, key string) (string, error) {
	var value, band, origin, meaning, review, state string
	err := q.QueryRow(ctx, `SELECT COALESCE(r.value,''), r.band, r.origin, r.meaning, r.review_status, r.value_state
FROM profile_rows r JOIN project_parts p ON p.org_id=r.org_id AND p.site_id=r.site_id AND p.id=r.part_id
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND p.kind='whole' AND r.key=$3 LIMIT 1`, org, project, key).Scan(&value, &band, &origin, &meaning, &review, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if !appliedProgramRow(band, value, state, review, origin, meaning) {
		return "", nil
	}
	return value, nil
}

// sopInputs are the state's Act row and its reviewed public holidays.
func (s *Store) sopInputs(state string) (*invoices.SOPAct, []string, map[int]bool) {
	cat := s.workCatalog()
	if cat == nil || state == "" {
		return nil, nil, map[int]bool{}
	}
	var act *invoices.SOPAct
	if a, ok := cat.SOPAct(state); ok {
		act = &invoices.SOPAct{State: a.State, Act: a.Act, Verified: a.Verified, Source: a.SecondarySource}
		if a.PrimarySource != "" && a.Verified {
			act.Source = a.PrimarySource
		}
		if a.Schedule != nil {
			act.ScheduleDays, act.ScheduleSection = a.Schedule.BusinessDays, a.Schedule.Section
		}
		if a.Payment != nil {
			act.PaymentDays, act.PaymentSection = a.Payment.BusinessDays, a.Payment.Section
		}
		for _, sp := range a.Excludes.Spans {
			act.Spans = append(act.Spans, [2]string{sp.From, sp.To})
		}
	}
	days, years := libraryCalendar(cat, state)
	var holidays []string
	for _, d := range days {
		if d.Kind == "public_holiday" {
			holidays = append(holidays, d.Day)
		}
	}
	covered := map[int]bool{}
	for _, y := range years {
		covered[y] = true
	}
	return act, holidays, covered
}

// claimLedger works out every claim's figures and dates on the ledger's
// snapshot, links tax invoices to the claims they settle, and returns the
// tab's claims. It runs before the summary, so Claimed to Date counts
// certified claims.
func (s *Store) claimLedger(ctx context.Context, tx pgx.Tx, org, project, today string, v *LedgerView, settles map[string]string) error {
	rowsBy, err := readClaimRows(ctx, tx, org, project)
	if err != nil {
		return err
	}
	issuedBy, err := readSchedules(ctx, tx, org, project)
	if err != nil {
		return err
	}
	hasClaim := false
	for _, inv := range v.Invoices {
		hasClaim = hasClaim || inv.Kind == invoices.KindProgressClaim
	}
	for i := range v.Invoices {
		if id := settles[v.Invoices[i].ID]; id != "" {
			v.Invoices[i].SettlesClaim, v.Invoices[i].SettlesBy = id, invoices.DecidedByUser
		}
	}
	if !hasClaim {
		return nil
	}
	state, err := projectState(ctx, tx, org, project)
	if err != nil {
		return err
	}
	act, holidays, years := s.sopInputs(state)
	contracts := map[string]Contract{}
	for _, c := range v.Contracts {
		contracts[c.PackageID] = c
	}
	variationLine, variationStatus := map[string]string{}, map[string]string{}
	for _, va := range v.Variations {
		variationLine[va.ID], variationStatus[va.ID] = va.LineID, va.Status
	}
	lineToDate := map[string]*int64{}
	for _, l := range v.Lines {
		if l.Contract != nil {
			n := *l.Contract + l.Approved
			lineToDate[l.ID] = &n
		}
	}
	var inputs []invoices.ClaimInput
	views := map[string]*ClaimView{}
	var order []string
	for _, inv := range v.Invoices {
		if inv.Kind != invoices.KindProgressClaim || inv.Status == invoices.StatusVoid || inv.Status == invoices.StatusSuperseded {
			continue
		}
		d := inv.Decisions
		cv := &ClaimView{InvoiceID: inv.ID, PackageID: inv.PackageID, Number: inv.Number, Rows: rowsBy[inv.ID], Issued: issuedBy[inv.ID],
			ReferenceDate: d[invoices.FieldReferenceDate].Value, ScheduleServed: d[invoices.FieldScheduleServedOn].Value}
		if served := d[invoices.FieldServedOn]; served.Value != "" {
			cv.Served, cv.ServedConfirmed = served.Value, served.DecidedBy == invoices.DecidedByUser
		} else {
			cv.Served = inv.CreatedAt.In(invoices.Sydney).Format("2006-01-02")
		}
		c := contracts[inv.PackageID]
		in := invoices.SOPInput{State: state, Act: act, Holidays: holidays, Years: years, Served: cv.Served, ServedConfirmed: cv.ServedConfirmed,
			ScheduleDays: c.ScheduleDays, ScheduleBasis: c.ScheduleBasis, PaymentDays: c.PaymentDays, PaymentBasis: c.PaymentBasis}
		cv.Schedule, cv.Payment = invoices.ScheduleDue(in), invoices.PaymentDue(in)
		cv.Alert = invoices.ScheduleAlert(in, cv.Schedule, cv.ScheduleServed != "", today)
		// Issued in time but its service not yet recorded: a date to enter,
		// not a schedule missed. Before plan 1's Send the PM types it.
		if len(cv.Issued) > 0 && cv.Alert != "" && (cv.Schedule.Date == "" || cv.Issued[0].IssuedOn <= cv.Schedule.Date) {
			cv.Alert = invoices.AlertCheck
		}
		cv.Late = cv.Alert == invoices.AlertOverdue
		ci := invoices.ClaimInput{ID: inv.ID, PackageID: inv.PackageID, Created: inv.CreatedAt, Rows: cv.Rows, Late: cv.Late,
			ReferenceDate: cv.ReferenceDate, GST: inv.GST, Total: inv.Total}
		if n := len(cv.Issued); n > 0 {
			ci.Issued, ci.IssuedAt = true, cv.Issued[0].IssuedAt
		}
		inputs = append(inputs, ci)
		views[inv.ID] = cv
		order = append(order, inv.ID)
	}
	figures := invoices.ClaimLedger(inputs, variationLine, variationStatus, lineToDate)
	settled := map[string]bool{}
	for i := range v.Invoices {
		if f, ok := figures[v.Invoices[i].ID]; ok {
			f := f
			v.Invoices[i].Claim = &f
			views[v.Invoices[i].ID].Figures = f
		}
		if id := v.Invoices[i].SettlesClaim; id != "" {
			settled[id] = true
		}
	}
	// The link rule: an invoice from the claim's package whose total is the
	// schedule's payable to the cent stands for that claim.
	for i := range v.Invoices {
		inv := &v.Invoices[i]
		if inv.Kind != "invoice" || inv.SettlesClaim != "" || inv.Total == nil || inv.Status == invoices.StatusVoid || inv.Status == invoices.StatusSuperseded {
			continue
		}
		for _, id := range order {
			cv := views[id]
			if cv.PackageID == inv.PackageID && len(cv.Issued) > 0 && !settled[id] && cv.Figures.Payable != nil && *cv.Figures.Payable == *inv.Total {
				inv.SettlesClaim, inv.SettlesBy = id, invoices.DecidedByRule
				settled[id] = true
				break
			}
		}
	}
	for _, id := range order {
		v.Claims = append(v.Claims, *views[id])
	}
	for _, inv := range v.Invoices {
		if inv.SettlesClaim != "" {
			if cv := views[inv.SettlesClaim]; cv != nil {
				for i := range v.Claims {
					if v.Claims[i].InvoiceID == inv.SettlesClaim {
						v.Claims[i].SettledBy = append(v.Claims[i].SettledBy, inv.ID)
					}
				}
			}
		}
	}
	return nil
}

// claimInvoiceFlags asks about an unlinked invoice from a package that has
// claims: it may be the tax invoice for one.
func claimInvoiceFlags(v *LedgerView) {
	withClaims := map[string]bool{}
	for _, c := range v.Claims {
		withClaims[c.PackageID] = true
	}
	for _, inv := range v.Invoices {
		if inv.Kind == "invoice" && inv.SettlesClaim == "" && withClaims[inv.PackageID] && inv.Status != invoices.StatusVoid && inv.Status != invoices.StatusSuperseded {
			v.Flags[inv.ID] = append(v.Flags[inv.ID], invoices.Flag{Code: "claim_tax_invoice", Reason: "Is this the tax invoice for a progress claim? Link it so it is not counted twice"})
		}
	}
}

// ClaimAssess is the PM's assessment of a claim's rows, or Accept claimed.
type ClaimAssess struct {
	Rows []struct {
		ID        string  `json:"id"`
		Assessed  *string `json:"assessed_to_date"`
		Reason    *string `json:"reason"`
		Line      *string `json:"line"`
		Variation *string `json:"variation"`
		Retention bool    `json:"retention"`
	} `json:"rows"`
	AcceptClaimed bool `json:"accept_claimed"`
}

// AssessClaim saves the PM's assessed to date, reasons and mappings, or
// takes every row's claimed to date. A claim whose schedule is issued is
// refused, by the database's trigger as well as here.
func (s *Store) AssessClaim(ctx context.Context, org, project, actor, invoiceID string, in ClaimAssess) error {
	if len(in.Rows) > invoices.MaxClaimRows+20 || in.AcceptClaimed && len(in.Rows) > 0 {
		return claimInvalid("send rows or accept_claimed, not both")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	pkg, err := claimPackage(ctx, tx, org, project, invoiceID)
	if err != nil {
		return err
	}
	rows, err := readClaimRows(ctx, tx, org, project)
	if err != nil {
		return err
	}
	mine := map[string]invoices.ClaimRow{}
	for _, r := range rows[invoiceID] {
		mine[r.ID] = r
	}
	contracts, err := readContracts(ctx, tx, org, project)
	if err != nil {
		return err
	}
	vs, err := readVariations(ctx, tx, org, project)
	if err != nil {
		return err
	}
	lines, vars := map[string]bool{}, map[string]bool{}
	for _, c := range contracts {
		if c.PackageID == pkg {
			for _, l := range c.Lines {
				lines[l.ID] = true
			}
		}
	}
	for _, va := range vs {
		if va.PackageID == pkg {
			vars[va.ID] = true
		}
	}
	if in.AcceptClaimed {
		for _, r := range rows[invoiceID] {
			if r.ToDate == nil {
				continue
			}
			a := *r.ToDate
			if r.Kind == invoices.RowRetention && a < 0 {
				a = -a
			}
			if _, err := tx.Exec(ctx, `UPDATE progress_claim_lines SET assessed_to_date_cents=$3, version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid AND assessed_to_date_cents IS DISTINCT FROM $3`, org, r.ID, a); err != nil {
				return claimWriteError(err)
			}
		}
		return s.finishClaimsWrite(ctx, tx, org, project)
	}
	for _, w := range in.Rows {
		r, ok := mine[w.ID]
		if !ok {
			return ErrNotFound
		}
		if w.Assessed != nil {
			c, ok, err := invoices.ParseCents(*w.Assessed)
			if err != nil {
				return claimInvalid("the assessed amount is not an amount")
			}
			if ok {
				r.Assessed = &c
			} else {
				r.Assessed = nil
			}
		}
		if w.Reason != nil {
			reason := strings.TrimSpace(*w.Reason)
			if len(reason) > 1000 {
				return claimInvalid("a reason is at most 1,000 characters")
			}
			r.Reason = reason
		}
		switch {
		case w.Retention:
			r.Kind, r.LineID, r.VariationID, r.MappedBy, r.Band = invoices.RowRetention, "", "", invoices.DecidedByUser, invoices.BandGreen
		case w.Line != nil && *w.Line != "":
			if !lines[*w.Line] {
				return claimInvalid("that line is not on this claim's contract")
			}
			r.Kind, r.LineID, r.VariationID, r.MappedBy, r.Band = invoices.RowContract, *w.Line, "", invoices.DecidedByUser, invoices.BandGreen
		case w.Variation != nil && *w.Variation != "":
			if !vars[*w.Variation] {
				return claimInvalid("that variation is not on this claim's contract")
			}
			r.Kind, r.LineID, r.VariationID, r.MappedBy, r.Band = invoices.RowVariation, "", *w.Variation, invoices.DecidedByUser, invoices.BandGreen
		case w.Line != nil:
			r.Kind, r.LineID, r.VariationID, r.MappedBy, r.Band = invoices.RowContract, "", "", invoices.DecidedByUser, invoices.BandGreen
		}
		if _, err := tx.Exec(ctx, `UPDATE progress_claim_lines SET kind=$3, cost_item_id=NULLIF($4,'')::uuid, variation_id=NULLIF($5,'')::uuid, mapped_by=$6, band=$7,
 assessed_to_date_cents=$8, reason=$9, version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ID, r.Kind, r.LineID, r.VariationID, r.MappedBy, r.Band, r.Assessed, r.Reason); err != nil {
			return claimWriteError(err)
		}
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
}

// claimPackage is a claim's package, or ErrNotFound when the invoice is not
// a progress claim in this project.
func claimPackage(ctx context.Context, tx pgx.Tx, org, project, invoiceID string) (string, error) {
	var kind, pkg string
	err := tx.QueryRow(ctx, `SELECT COALESCE(kind,''), COALESCE(package_id::text,'') FROM invoices WHERE org_id=$1::uuid AND project_id=$2::uuid AND id=$3::uuid FOR UPDATE`, org, project, invoiceID).Scan(&kind, &pkg)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	if kind != invoices.KindProgressClaim {
		return "", claimInvalid("this invoice is not a progress claim")
	}
	return pkg, nil
}

func claimWriteError(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23514" && strings.Contains(pg.Message, "frozen") {
		return ErrClaimIssued
	}
	return err
}

// writeClaimRows stores a read claim's rows once: a later re-check or
// re-read never replaces rows the PM may have assessed. An unread claim
// gets its contract's lines with amounts unknown.
func writeClaimRows(ctx context.Context, tx pgx.Tx, org, project, invoiceID string, out invoices.Outcome, contracts map[string]invoices.ClaimContract) error {
	var kind, pkg string
	var n int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(i.kind,''), COALESCE(i.package_id::text,''), (SELECT count(*) FROM progress_claim_lines l WHERE l.org_id=i.org_id AND l.invoice_id=i.id)
FROM invoices i WHERE i.org_id=$1::uuid AND i.id=$2::uuid`, org, invoiceID).Scan(&kind, &pkg, &n); err != nil {
		return err
	}
	if kind != invoices.KindProgressClaim || n > 0 {
		return nil
	}
	var rows []invoices.ClaimRow
	switch {
	case out.Claim != nil:
		rows = out.Claim.Rows
	default:
		rows = invoices.ContractRows(contracts[pkg])
	}
	for i, r := range rows {
		var page any
		if r.Page > 0 {
			page = r.Page
		}
		if _, err := tx.Exec(ctx, `INSERT INTO progress_claim_lines (org_id, id, project_id, invoice_id, ordinal, kind, label, page, cost_item_id, variation_id, mapped_by, band, to_date_cents, previous_cents, this_claim_cents)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11,$12,$13,$14,$15)`,
			org, newID(), project, invoiceID, i, r.Kind, r.Label, page, r.LineID, r.VariationID, r.MappedBy, r.Band, r.ToDate, r.Previous, r.ThisClaim); err != nil {
			return err
		}
	}
	return nil
}

// claimContexts are the live works contracts a claim may be read against.
func claimContexts(ctx context.Context, q querier, org, project string) (map[string]invoices.ClaimContract, error) {
	contracts, err := readContracts(ctx, q, org, project)
	if err != nil {
		return nil, err
	}
	vs, err := readVariations(ctx, q, org, project)
	if err != nil {
		return nil, err
	}
	out := map[string]invoices.ClaimContract{}
	for _, c := range contracts {
		if !c.Claims() {
			continue
		}
		cc := invoices.ClaimContract{PackageID: c.PackageID}
		for _, l := range c.Lines {
			cc.Lines = append(cc.Lines, invoices.Line{ID: l.ID, Label: l.Label, PackageID: c.PackageID})
		}
		for _, va := range vs {
			if va.AwardID == c.AwardID {
				cc.Variations = append(cc.Variations, invoices.ClaimVariation{ID: va.ID, Number: va.Number, Title: va.Title, Status: va.Status})
			}
		}
		out[c.PackageID] = cc
	}
	return out, nil
}

// ContractPeriods are the contract's own Security of Payment periods.
type ContractPeriods struct {
	ScheduleDays  *int   `json:"schedule_days"`
	ScheduleBasis string `json:"schedule_basis"`
	PaymentDays   *int   `json:"payment_days"`
	PaymentBasis  string `json:"payment_basis"`
}

// SetContractPeriods types the contract's periods on its live award. A nil
// days is not entered. Awards carry no capture trigger, so from chat this
// waits for the PM's click.
func (s *Store) SetContractPeriods(ctx context.Context, org, project, actor, pkg string, in ContractPeriods) error {
	for _, b := range []*string{&in.ScheduleBasis, &in.PaymentBasis} {
		if *b == "" {
			*b = "business"
		}
		if *b != "business" && *b != "calendar" {
			return claimInvalid("days are business or calendar days")
		}
	}
	if in.ScheduleDays != nil && (*in.ScheduleDays < 0 || *in.ScheduleDays > 60) || in.PaymentDays != nil && (*in.PaymentDays < 0 || *in.PaymentDays > 120) {
		return claimInvalid("the schedule period is 0 to 60 days and the payment period 0 to 120")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockProject(ctx, tx, org, project); err != nil {
		return err
	}
	if err := packageActor(ctx, tx, org, actor); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE awards SET schedule_days=$4, schedule_basis=$5, payment_days=$6, payment_basis=$7
WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 AND undone_at IS NULL`, org, project, pkg, in.ScheduleDays, in.ScheduleBasis, in.PaymentDays, in.PaymentBasis)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
}

// AttentionClaims are the ledger's claims as the attention rules read them:
// one definition for the left-nav mark, chat and the project report.
func AttentionClaims(v LedgerView) []reports.AttentionClaim {
	var out []reports.AttentionClaim
	for _, c := range v.Claims {
		out = append(out, reports.AttentionClaim{ID: c.InvoiceID, Number: c.Number, Alert: c.Alert, Due: c.Schedule.Date})
	}
	return out
}
