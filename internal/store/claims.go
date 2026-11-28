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
	// ScheduleServedConfirmed: the PM typed or confirmed the day served. A
	// day set by rule when the schedule's message was sent is not.
	ScheduleServedConfirmed bool
	// Send is the message the issued schedule last went out on; nil when it
	// has not been sent from SiteWise.
	Send *ClaimSend
	// TaxInvoice: the claim is its own tax invoice. AwaitingInvoice: it is
	// not, and no invoice is linked to it yet.
	TaxInvoice, AwaitingInvoice bool
	// Note is the PM's own words on the schedule.
	Note      string
	Rows      []invoices.ClaimRow
	Figures   invoices.ClaimFigures
	Schedule  invoices.DueDate
	Payment   invoices.DueDate
	Alert     string
	Late      bool
	Issued    []IssuedSchedule
	SettledBy []string // tax invoices linked to it
}

// ClaimSend is a sent schedule's message: queued, sent or failed, and the
// day it went (Sydney).
type ClaimSend struct {
	MessageID, Status, SentOn string
}

func readClaimRows(ctx context.Context, q querier, org, project string) (map[string][]invoices.ClaimRow, error) {
	rows, err := q.Query(ctx, `
SELECT invoice_id::text, id::text, ordinal, kind, label, COALESCE(page,0), COALESCE(cost_item_id::text,''), COALESCE(variation_id::text,''), mapped_by, band,
 to_date_cents, previous_cents, this_claim_cents, assessed_to_date_cents, reason, version, COALESCE(breakdown_line_id::text,'')
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
			&r.ToDate, &r.Previous, &r.ThisClaim, &r.Assessed, &r.Reason, &r.Version, &r.BreakdownID); err != nil {
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
	// A row that names only its cost line sits on that line's whole
	// breakdown line.
	for _, inv := range v.Invoices {
		if c, ok := contracts[inv.PackageID]; ok {
			c.placeRows(rowsBy[inv.ID])
		}
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
	sentOn := map[string]string{} // message id -> claim id
	for _, inv := range v.Invoices {
		if inv.Kind != invoices.KindProgressClaim || inv.Status == invoices.StatusVoid || inv.Status == invoices.StatusSuperseded {
			continue
		}
		d := inv.Decisions
		cv := &ClaimView{InvoiceID: inv.ID, PackageID: inv.PackageID, Number: inv.Number, Rows: rowsBy[inv.ID], Issued: issuedBy[inv.ID],
			ReferenceDate: d[invoices.FieldReferenceDate].Value, ScheduleServed: d[invoices.FieldScheduleServedOn].Value, Note: d[invoices.FieldScheduleNote].Value}
		if ss := d[invoices.FieldScheduleServedOn]; ss.Value != "" {
			cv.ScheduleServedConfirmed = ss.DecidedBy == invoices.DecidedByUser
		}
		cv.TaxInvoice = d[invoices.FieldTaxInvoice].Value == "yes" || d[invoices.FieldTaxInvoice].Value == "" && contracts[inv.PackageID].Services()
		if m := d[invoices.FieldScheduleMessage].Value; m != "" {
			sentOn[m] = inv.ID
		}
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
		// Past due warns whatever the date came from; the claimed amount
		// counts only once a verified row's own date has passed.
		cv.Late = cv.Alert == invoices.AlertOverdue && invoices.ScheduleLapsed(in, cv.ScheduleServed != "", today)
		ci := invoices.ClaimInput{ID: inv.ID, PackageID: inv.PackageID, Created: inv.CreatedAt, Rows: cv.Rows, Late: cv.Late,
			ReferenceDate: cv.ReferenceDate, GST: inv.GST, Total: inv.Total}
		if n := len(cv.Issued); n > 0 {
			ci.Issued, ci.IssuedAt = true, cv.Issued[0].IssuedAt
		}
		inputs = append(inputs, ci)
		views[inv.ID] = cv
		order = append(order, inv.ID)
	}
	// Only a claim whose schedule was sent from here has a message to read.
	if len(sentOn) > 0 {
		ids := make([]string, 0, len(sentOn))
		for id := range sentOn {
			ids = append(ids, id)
		}
		rows, err := tx.Query(ctx, `SELECT id::text, send_status, COALESCE(to_char(sent_at AT TIME ZONE 'Australia/Sydney','YYYY-MM-DD'),'')
FROM mail_messages WHERE org_id=$1::uuid AND project_id=$2::uuid AND id::text = ANY($3)`, org, project, ids)
		if err != nil {
			return err
		}
		for rows.Next() {
			var m ClaimSend
			if err := rows.Scan(&m.MessageID, &m.Status, &m.SentOn); err != nil {
				rows.Close()
				return err
			}
			if cv := views[sentOn[m.MessageID]]; cv != nil {
				cv.Send = &m
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
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
	linkTaxInvoices(v, views, order, settled)
	for _, id := range order {
		cv := views[id]
		cv.AwaitingInvoice = !cv.TaxInvoice && !settled[id]
		v.Claims = append(v.Claims, *cv)
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

// linkTaxInvoices is the link rule, worked out at read (slice 6): an
// invoice from a claim's package stands for the one claim, still without
// an invoice, whose figures it matches: the schedule's payable, the
// schedule's net (GST rounded the other way), or the claim's own printed
// total (an invoice sent before the schedule, or for the amount claimed).
// Two claims that could take it is no link. Then, on a contract with a
// claim still waiting for its invoice, an invoice no rule placed is held
// out of the totals until the PM places it.
func linkTaxInvoices(v *LedgerView, views map[string]*ClaimView, order []string, settled map[string]bool) {
	printed := map[string]*int64{}
	for _, inv := range v.Invoices {
		if inv.Kind == invoices.KindProgressClaim {
			printed[inv.ID] = inv.Total
		}
	}
	same := func(a, b *int64) bool { return a != nil && b != nil && *a == *b }
	open := func(inv invoices.Invoice) bool {
		return inv.Kind == "invoice" && inv.SettlesClaim == "" && !inv.NotClaim && inv.PackageID != "" &&
			inv.Status != invoices.StatusVoid && inv.Status != invoices.StatusSuperseded
	}
	for i := range v.Invoices {
		inv := &v.Invoices[i]
		if !open(*inv) {
			continue
		}
		var match, note string
		n := 0
		for _, id := range order {
			cv := views[id]
			if cv.PackageID != inv.PackageID || settled[id] || cv.TaxInvoice {
				continue
			}
			issued := len(cv.Issued) > 0
			how := ""
			switch {
			case issued && same(inv.Total, cv.Figures.Payable), issued && same(inv.Ex, cv.Figures.Net):
			case same(inv.Total, printed[id]):
				how = invoices.LinkWaiting
				if issued {
					how = invoices.LinkClaimedAmount
				}
			default:
				continue
			}
			n++
			match, note = id, how
		}
		if n == 1 {
			inv.SettlesClaim, inv.SettlesBy, inv.LinkNote = match, invoices.DecidedByRule, note
			settled[match] = true
		}
	}
	waiting := map[string]bool{}
	for _, id := range order {
		if cv := views[id]; !settled[id] && !cv.TaxInvoice {
			waiting[cv.PackageID] = true
		}
	}
	for i := range v.Invoices {
		if inv := &v.Invoices[i]; open(*inv) && waiting[inv.PackageID] {
			inv.Held = true
		}
	}
}

// claimInvoiceFlags says what the link rule could not settle: an invoice
// held until it is placed, and one linked on the amount claimed.
func claimInvoiceFlags(v *LedgerView) {
	for _, inv := range v.Invoices {
		switch {
		case inv.Held:
			v.Flags[inv.ID] = append(v.Flags[inv.ID], invoices.Flag{Code: "claim_invoice_unplaced", Reason: "Not counted until it is placed: is it the tax invoice for a progress claim, or not for a claim?"})
		case inv.LinkNote == invoices.LinkClaimedAmount:
			v.Flags[inv.ID] = append(v.Flags[inv.ID], invoices.Flag{Code: "claim_invoice_claimed_amount", Reason: "Invoiced the claimed amount; the schedule certified a different amount"})
		}
	}
}

// RetentionMove is what a claim holds or returns of retention, as a percent
// of the contract's base or as dollars. Go adds it to what was held before
// and refuses a hold past the limit or a return of more than is held. Zero
// is no change this claim.
type RetentionMove struct {
	HoldPercent   *string `json:"hold_percent"`
	ReturnPercent *string `json:"return_percent"`
	Hold          *string `json:"hold"`
	Return        *string `json:"return"`
}

// ClaimAssess is the PM's assessment of a claim's rows, or Accept claimed.
// A row with no id names a line, a variation or retention the claim left
// out, and is added. Percent is what is complete to date as a percent of
// the line's sum; Go works out the dollars.
type ClaimAssess struct {
	Rows []struct {
		ID        string  `json:"id"`
		Assessed  *string `json:"assessed_to_date"`
		Percent   *string `json:"percent"`
		Reason    *string `json:"reason"`
		Line      *string `json:"line"`
		Variation *string `json:"variation"`
		Retention bool    `json:"retention"`
	} `json:"rows"`
	AcceptClaimed bool `json:"accept_claimed"`
	// Retention sets what this claim holds or returns.
	Retention *RetentionMove `json:"retention"`
}

// AssessClaim saves the PM's assessed to date, reasons and mappings, or
// takes every row's claimed to date. A claim whose schedule is issued is
// refused, by the database's trigger as well as here.
func (s *Store) AssessClaim(ctx context.Context, org, project, actor, invoiceID string, in ClaimAssess) error {
	if len(in.Rows) > invoices.MaxClaimRows+20 || in.AcceptClaimed && (len(in.Rows) > 0 || in.Retention != nil) {
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
	var issued bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM reports r JOIN report_versions v ON v.org_id=r.org_id AND v.report_id=r.id AND v.status='issued'
WHERE r.org_id=$1::uuid AND r.kind='payment_schedule' AND r.invoice_id=$2::uuid)`, org, invoiceID).Scan(&issued); err != nil {
		return err
	}
	if issued {
		return ErrClaimIssued
	}
	rows, err := readClaimRows(ctx, tx, org, project)
	if err != nil {
		return err
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
	stored, whole := map[string]BreakdownLine{}, map[string]string{}
	sums, labels := map[string]*int64{}, map[string]string{} // by row key: what a percent is a percent of
	for _, c := range contracts {
		if c.PackageID == pkg {
			c.placeRows(rows[invoiceID])
			whole = c.wholeLines()
			for _, l := range c.Lines {
				lines[l.ID] = true
			}
			for _, b := range c.Breakdown {
				key := "l:" + b.ID
				if b.Stored {
					stored[b.ID], key = b, invoices.BreakdownKey(b.ID)
				}
				sums[key], labels[key] = b.Amount, b.Label
			}
		}
	}
	mine := map[string]invoices.ClaimRow{}
	for _, r := range rows[invoiceID] {
		mine[r.ID] = r
	}
	// put places a row on a line named by its breakdown line or its cost line.
	put := func(r *invoices.ClaimRow, id string) bool {
		if b, ok := stored[id]; ok && !b.Retired {
			r.LineID, r.BreakdownID = b.CostLineID, b.ID
			return true
		}
		if lines[id] {
			r.LineID, r.BreakdownID = id, whole[id]
			return true
		}
		return false
	}
	for _, va := range vs {
		if va.PackageID == pkg {
			vars[va.ID] = true
			labels["v:"+va.ID] = strings.TrimSpace(va.Number + " " + va.Title)
			if va.Status == invoices.VariationApproved {
				sums["v:"+va.ID] = va.Approved
			}
		}
	}
	// on is what a row sits on, as the ledger keys it.
	on := func(r invoices.ClaimRow) string {
		switch {
		case r.Kind == invoices.RowRetention:
			return "retention"
		case r.VariationID != "":
			return "v:" + r.VariationID
		case r.BreakdownID != "":
			return invoices.BreakdownKey(r.BreakdownID)
		case r.LineID != "":
			return "l:" + r.LineID
		}
		return ""
	}
	next, taken := 0, map[string]bool{}
	for _, r := range rows[invoiceID] {
		if r.Ordinal >= next {
			next = r.Ordinal + 1
		}
		taken[on(r)] = true
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
	if in.Retention != nil {
		held, err := s.retentionHeld(ctx, tx, org, project, invoiceID, *in.Retention)
		if err != nil {
			return err
		}
		// The move is the claim's retention row: its held to date.
		row := in.Rows[:0:0]
		row = append(row, in.Rows...)
		amount := invoices.FormatCents(held)
		in.Rows = append(row, struct {
			ID        string  `json:"id"`
			Assessed  *string `json:"assessed_to_date"`
			Percent   *string `json:"percent"`
			Reason    *string `json:"reason"`
			Line      *string `json:"line"`
			Variation *string `json:"variation"`
			Retention bool    `json:"retention"`
		}{Assessed: &amount, Retention: true})
		for _, r := range rows[invoiceID] {
			if r.Kind == invoices.RowRetention {
				in.Rows[len(in.Rows)-1].ID = r.ID
			}
		}
	}
	for _, w := range in.Rows {
		r, ok := mine[w.ID]
		add := w.ID == ""
		switch {
		case add:
			// The claim left this line out, so it claims nothing on it this
			// time; the row is added by its first assessment.
			nothing := int64(0)
			r = invoices.ClaimRow{MappedBy: invoices.DecidedByUser, Band: invoices.BandGreen, ThisClaim: &nothing}
			switch {
			case w.Retention:
				r.Kind, r.Label = invoices.RowRetention, "Retention"
			case w.Variation != nil && vars[*w.Variation]:
				r.Kind, r.VariationID = invoices.RowVariation, *w.Variation
			case w.Line != nil && put(&r, *w.Line):
				r.Kind = invoices.RowContract
			default:
				return claimInvalid("that line is not on this claim's contract")
			}
			if taken[on(r)] {
				return claimInvalid("that line already has a row on this claim")
			}
			taken[on(r)] = true
			if r.Label == "" {
				r.Label = labels[on(r)]
			}
		case !ok:
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
		case add:
		case w.Retention:
			r.Kind, r.LineID, r.BreakdownID, r.VariationID, r.MappedBy, r.Band = invoices.RowRetention, "", "", "", invoices.DecidedByUser, invoices.BandGreen
		case w.Line != nil && *w.Line != "":
			if !put(&r, *w.Line) {
				return claimInvalid("that line is not on this claim's contract")
			}
			r.Kind, r.VariationID, r.MappedBy, r.Band = invoices.RowContract, "", invoices.DecidedByUser, invoices.BandGreen
		case w.Variation != nil && *w.Variation != "":
			if !vars[*w.Variation] {
				return claimInvalid("that variation is not on this claim's contract")
			}
			r.Kind, r.LineID, r.BreakdownID, r.VariationID, r.MappedBy, r.Band = invoices.RowVariation, "", "", *w.Variation, invoices.DecidedByUser, invoices.BandGreen
		case w.Line != nil:
			r.Kind, r.LineID, r.BreakdownID, r.VariationID, r.MappedBy, r.Band = invoices.RowContract, "", "", "", invoices.DecidedByUser, invoices.BandGreen
		}
		if w.Percent != nil {
			hundredths, ok, err := invoices.ParseCents(*w.Percent)
			if err != nil || !ok || hundredths < 0 || hundredths > 100000 {
				return claimInvalid("the percent is a number from 0 to 1,000")
			}
			sum := sums[on(r)]
			if sum == nil || *sum <= 0 {
				return claimInvalid("a percent needs the line's sum; type the amount instead")
			}
			a := (*sum*hundredths + 5000) / 10000
			r.Assessed = &a
		}
		if add {
			if _, err := tx.Exec(ctx, `INSERT INTO progress_claim_lines (org_id, id, project_id, invoice_id, ordinal, kind, label, cost_item_id, variation_id, mapped_by, band, this_claim_cents, assessed_to_date_cents, reason, breakdown_line_id)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid,$10,$11,$12,$13,$14,NULLIF($15,'')::uuid)`,
				org, newID(), project, invoiceID, next, r.Kind, r.Label, r.LineID, r.VariationID, r.MappedBy, r.Band, r.ThisClaim, r.Assessed, r.Reason, r.BreakdownID); err != nil {
				return claimWriteError(err)
			}
			next++
			continue
		}
		if _, err := tx.Exec(ctx, `UPDATE progress_claim_lines SET kind=$3, cost_item_id=NULLIF($4,'')::uuid, variation_id=NULLIF($5,'')::uuid, mapped_by=$6, band=$7,
 assessed_to_date_cents=$8, reason=$9, breakdown_line_id=NULLIF($10,'')::uuid, version=version+1 WHERE org_id=$1::uuid AND id=$2::uuid`, org, r.ID, r.Kind, r.LineID, r.VariationID, r.MappedBy, r.Band, r.Assessed, r.Reason, r.BreakdownID); err != nil {
			return claimWriteError(err)
		}
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
}

// retentionHeld works out a claim's retention held to date from what it
// holds or returns this time. A percent is of the contract's base, so it
// needs the contract's retention terms.
func (s *Store) retentionHeld(ctx context.Context, tx pgx.Tx, org, project, invoiceID string, m RetentionMove) (int64, error) {
	given := 0
	for _, p := range []*string{m.HoldPercent, m.ReturnPercent, m.Hold, m.Return} {
		if p != nil {
			given++
		}
	}
	if given != 1 {
		return 0, claimInvalid("say one of: hold or return, as a percent or as dollars")
	}
	v, err := s.invoiceLedgerTx(ctx, tx, org, project, time.Now().In(invoices.Sydney).Format("2006-01-02"))
	if err != nil {
		return 0, err
	}
	sheet, ok := ClaimSheet(v, invoiceID)
	if !ok {
		return 0, claimInvalid("this claim is not on a live contract")
	}
	r := sheet.Retention
	var cents int64
	percent := m.HoldPercent
	if percent == nil {
		percent = m.ReturnPercent
	}
	dollars := m.Hold
	if dollars == nil {
		dollars = m.Return
	}
	switch {
	case percent != nil:
		hundredths, ok, err := invoices.ParseCents(*percent)
		if err != nil || !ok || hundredths < 0 || hundredths > 10000 {
			return 0, claimInvalid("the retention percent is a number from 0 to 100")
		}
		if r.Base == nil {
			return 0, claimInvalid("a percent needs the contract's retention terms (its rate and what it is a percent of); type dollars instead")
		}
		cents = (*r.Base*hundredths + 5000) / 10000
	default:
		c, ok, err := invoices.ParseCents(*dollars)
		if err != nil || !ok || c < 0 {
			return 0, claimInvalid("the retention amount is not an amount")
		}
		cents = c
	}
	if m.ReturnPercent != nil || m.Return != nil {
		if cents > r.HeldBefore {
			return 0, claimInvalid("only $%s is held, so $%s cannot be returned", invoices.DisplayCents(r.HeldBefore), invoices.DisplayCents(cents))
		}
		return r.HeldBefore - cents, nil
	}
	if r.Limit != nil && r.HeldBefore+cents > *r.Limit {
		return 0, claimInvalid("the limit is $%s and $%s is already held, so $%s more cannot be held", invoices.DisplayCents(*r.Limit), invoices.DisplayCents(r.HeldBefore), invoices.DisplayCents(cents))
	}
	return r.HeldBefore + cents, nil
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
		// A row the reader placed on a breakdown line is stored on that
		// line and on the cost line it rolls up to.
		breakdown := ""
		if cost, ok := contracts[pkg].CostLine[r.LineID]; ok {
			breakdown, r.LineID = r.LineID, cost
		}
		if _, err := tx.Exec(ctx, `INSERT INTO progress_claim_lines (org_id, id, project_id, invoice_id, ordinal, kind, label, page, cost_item_id, variation_id, mapped_by, band, to_date_cents, previous_cents, this_claim_cents, breakdown_line_id)
VALUES ($1::uuid,$2::uuid,$3::uuid,$4::uuid,$5,$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11,$12,$13,$14,$15,NULLIF($16,'')::uuid)`,
			org, newID(), project, invoiceID, i, r.Kind, r.Label, page, r.LineID, r.VariationID, r.MappedBy, r.Band, r.ToDate, r.Previous, r.ThisClaim, breakdown); err != nil {
			return err
		}
	}
	return nil
}

// claimContexts are the live contracts a claim may be read against. A
// services contract takes a claim only when the PM drops it there; the
// words rule never makes a consultant's invoice a claim (decision 7).
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
		// A claim's rows are placed on the breakdown's lines, in the firm's
		// own words where the breakdown came from its tender.
		cc := invoices.ClaimContract{PackageID: c.PackageID, CostLine: map[string]string{}, Services: c.Services()}
		for _, b := range c.Breakdown {
			if b.Retired {
				continue
			}
			cc.Lines = append(cc.Lines, invoices.Line{ID: b.ID, Label: b.Label, PackageID: c.PackageID})
			if b.Stored {
				cc.CostLine[b.ID] = b.CostLineID
			}
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

// ContractTerms are what a payment schedule prints of the contract, typed
// once: its own date, and its retention as a percent of a base. A nil field
// is left as it is; an empty one clears it.
type ContractTerms struct {
	ContractDate     *string `json:"contract_date"`
	RetentionPercent *string `json:"retention_percent"`
	RetentionBase    *string `json:"retention_base"`
	// ActApplies is "yes" or "no" to name the Act on this contract's
	// schedules, "" for the default by kind.
	ActApplies *string `json:"act_applies"`
}

// SetContractTerms types a contract's date and retention terms on its live
// award. Awards carry no capture trigger, so from chat this waits for the
// PM's click, as the periods do.
func (s *Store) SetContractTerms(ctx context.Context, org, project, actor, pkg string, in ContractTerms) error {
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
	var date, base *string
	var bp *int32
	var act *bool
	err = tx.QueryRow(ctx, `SELECT contract_date::text, retention_bp, retention_base, act_applies FROM awards
WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 AND undone_at IS NULL FOR UPDATE`, org, project, pkg).Scan(&date, &bp, &base, &act)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if in.ContractDate != nil {
		date = nil
		if v := strings.TrimSpace(*in.ContractDate); v != "" {
			if t, err := time.Parse("2006-01-02", v); err != nil || t.Year() < 2000 || t.Year() > 2099 {
				return claimInvalid("the contract date uses YYYY-MM-DD")
			}
			date = &v
		}
	}
	if in.RetentionPercent != nil {
		bp = nil
		if v := strings.TrimSpace(*in.RetentionPercent); v != "" {
			hundredths, ok, err := invoices.ParseCents(v)
			if err != nil || !ok || hundredths < 0 || hundredths > 10000 {
				return claimInvalid("the retention rate is a percent from 0 to 100")
			}
			n := int32(hundredths)
			bp = &n
		}
	}
	if in.RetentionBase != nil {
		base = nil
		if v := strings.TrimSpace(*in.RetentionBase); v != "" {
			switch v {
			case invoices.RetentionBaseWorks, invoices.RetentionBaseLet, invoices.RetentionBaseToDate:
				base = &v
			default:
				return claimInvalid("retention is a percent of works, let (works plus provisional sums) or to_date (the contract sum to date)")
			}
		}
	}
	if in.ActApplies != nil {
		switch strings.TrimSpace(*in.ActApplies) {
		case "":
			act = nil
		case "yes":
			yes := true
			act = &yes
		case "no":
			no := false
			act = &no
		default:
			return claimInvalid("act_applies is yes, no, or empty for the default")
		}
	}
	// A rate with no base said is of the works sub total, the "original
	// sum" of the owner's own schedule. No rate, no base.
	if bp != nil && base == nil {
		works := invoices.RetentionBaseWorks
		base = &works
	}
	if bp == nil {
		base = nil
	}
	if _, err := tx.Exec(ctx, `UPDATE awards SET contract_date=$4::text::date, retention_bp=$5, retention_base=$6, act_applies=$7
WHERE org_id=$1::uuid AND project_id=$2::uuid AND package_id::text=$3 AND undone_at IS NULL`, org, project, pkg, date, bp, base, act); err != nil {
		return err
	}
	return s.finishClaimsWrite(ctx, tx, org, project)
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
