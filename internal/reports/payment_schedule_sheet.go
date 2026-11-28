package reports

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sitewise/internal/invoices"
)

// The payment schedule in the owner's format
// (docs/plans/2026-11-26-payment-schedules.md, slice 5): the sheet the
// screen shows, frozen as sections. Every figure is the sheet's own
// (invoices.BuildSheet), printed here and nowhere worked out again. The
// first layout (PaymentScheduleSections) stays for the schedules issued
// with it.

// ScheduleHead is what a schedule prints beside its figures: the parties,
// the claim it answers, and who approved it when. An empty value is left
// off the page; nothing here is filled in for the PM.
type ScheduleHead struct {
	Project, Address, Contract string
	Principal, PrincipalABN    string
	Contractor, ContractorABN  string
	// AssessorOrg is the principal's representative; AssessorNo its ACN or
	// ABN as typed.
	AssessorOrg, AssessorNo string
	Approver                string
	Number, Period          string
	Received, ReferenceDate string
	ContractDate            string
	IssuedOn, DueOn         string
	// Late: issued after its due date. The page says so.
	Late bool
	// Act is the Act's name, given only from a row the owner has verified.
	// No section number is printed.
	Act string
	// Note is the PM's own words. RetentionReason is why more retention is
	// held than the contractor claimed, when it is.
	Note, RetentionReason string
}

// Column widths in points, so every section's figures line up down the
// page. Each holds its head in the PDF's type (measured) and a figure of
// ten million dollars, and leaves the first column room for the longest
// section title, "Provisional sum adjustment".
var (
	scheduleLineWidths    = []float64{0, 76, 38, 84, 77, 77}
	schedulePaymentWidths = []float64{48, 50, 0, 78, 78, 68, 98}
)

var scheduleSectionTitles = map[string]string{
	invoices.SectionWorks:       "Contract summary: Works",
	invoices.SectionProvisional: "Provisional sum adjustment",
	invoices.SectionVariations:  "Variations",
	invoices.SectionUnplaced:    "Not placed on the contract",
}

var retentionBaseWords = map[string]string{
	invoices.RetentionBaseWorks:  "the works sub total",
	invoices.RetentionBaseLet:    "works and provisional sums",
	invoices.RetentionBaseToDate: "the contract sum to date",
}

// scheduleMoney prints cents as the sheet does: nil is unknown and nothing
// is a dash.
func scheduleMoney(p *int64) string {
	switch {
	case p == nil:
		return "unknown"
	case *p == 0:
		return "–"
	}
	return invoices.DisplayCents(*p)
}

func scheduleBlock(id string, table *Table, layout map[string]any) Block {
	b := Block{ID: "schedule:" + id, Table: table, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated"}
	if layout != nil {
		b.Basis, _ = json.Marshal(layout)
	}
	return b
}

func scheduleText(id, text string) Block {
	return Block{ID: "schedule:" + id, Text: text, Origin: "calculation", ReviewStatus: "accepted_for_planning", Meaning: "stated"}
}

// ScheduleSheetSections are the schedule's sections in the order the
// owner's own prints them: details, the contract summary, the provisional
// sum adjustment, variations, the total and retention, the note, the
// payment list and the approval. A section with no lines is left out. A
// section whose table names it in its first head (as the screen's does)
// has no title of its own, so the page carries no band over it.
func ScheduleSheetSections(sheet invoices.ScheduleSheet, h ScheduleHead) []Section {
	out := []Section{{ID: "details", Essential: true, Blocks: scheduleDetails(sheet, h)}}
	for _, sec := range sheet.Sections {
		if len(sec.Lines) == 0 {
			continue
		}
		out = append(out, Section{ID: sec.Kind, Essential: true, Blocks: []Block{scheduleLines(sec)}})
	}
	out = append(out, Section{ID: "totals", Essential: true, Blocks: []Block{scheduleTotals(sheet, h)}})
	note := "Scheduled for less than claimed. The reason is printed under each line."
	if sheet.PaidInFull {
		note = "Progress claim to be paid in full."
	}
	if own := strings.TrimSpace(h.Note); own != "" {
		note += "\n" + own
	}
	out = append(out, Section{ID: "note", Title: "Note", Essential: true, Blocks: []Block{scheduleText("note", note)}})
	out = append(out, Section{ID: "payments", Title: "Payment list", Essential: true, Blocks: []Block{schedulePayments(sheet)}})
	approval := &Table{Columns: []string{"Approved by", "Signature", "Date"}, Display: "document", Rows: [][]string{{h.Approver, "", LongDate(h.IssuedOn)}}}
	return append(out, Section{ID: "approval", Essential: true,
		Blocks: []Block{scheduleBlock("approval", approval, map[string]any{"layout": "schedule", "left": 3, "tall": []int{0}})}})
}

// scheduleDetails is the header: the parties on the left, the claim on the
// right, then the line that says what the page is.
func scheduleDetails(sheet invoices.ScheduleSheet, h ScheduleHead) []Block {
	with := func(name, word, number string) string {
		if name != "" && number != "" {
			return fmt.Sprintf("%s (%s %s)", name, word, number)
		}
		return name
	}
	// Eleven digits is an ABN; anything else typed is printed as an ACN.
	word := "ACN"
	if digits(h.AssessorNo) == 11 {
		word = "ABN"
	}
	var left, right []string
	add := func(col *[]string, label, value string) {
		if value != "" {
			*col = append(*col, label+": "+value)
		}
	}
	add(&left, "Project", h.Project)
	add(&left, "Address", h.Address)
	add(&left, "Contract", h.Contract)
	add(&left, "Principal", with(h.Principal, "ABN", h.PrincipalABN))
	add(&left, "Contractor", with(h.Contractor, "ABN", h.ContractorABN))
	add(&left, "Principal's rep", with(h.AssessorOrg, word, h.AssessorNo))
	add(&right, "Claim no.", h.Number)
	add(&right, "Claim received", LongDate(h.Received))
	add(&right, "Period", h.Period)
	add(&right, "Reference date", LongDate(h.ReferenceDate))
	invoice := "Awaiting invoice"
	for _, p := range sheet.Payments {
		if p.Current && p.InvoiceNo != "" {
			invoice = p.InvoiceNo
			if p.InvoiceDate != "" {
				invoice += ", " + LongDate(p.InvoiceDate)
			}
		}
	}
	add(&right, "Invoice no.", invoice)
	add(&right, "Contract date", LongDate(h.ContractDate))
	add(&right, "Issued", LongDate(h.IssuedOn))
	table := &Table{Columns: []string{"Parties", "Claim"}, Display: "document"}
	for i := 0; i < max(len(left), len(right)); i++ {
		row := []string{"", ""}
		if i < len(left) {
			row[0] = left[i]
		}
		if i < len(right) {
			row[1] = right[i]
		}
		table.Rows = append(table.Rows, row)
	}
	act := "This is a payment schedule."
	if h.Act != "" {
		act = "This is a payment schedule made under the " + h.Act + "."
	}
	// A late issue is red on screen; the sheet does not say so (owner
	// decision 9).
	return []Block{scheduleBlock("details", table, map[string]any{"layout": "label_value"}), scheduleText("act", act)}
}

func digits(s string) int {
	n := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n++
		}
	}
	return n
}

// scheduleLines is one section's table: its lines, each reason under its
// line, and the sub total.
func scheduleLines(sec invoices.SheetSection) Block {
	t := &Table{Columns: []string{scheduleSectionTitles[sec.Kind], "Total let ($)", "(%)", "Completed ($)", "Less prior ($)", "This claim ($)"}, Display: "document"}
	notes := []int{}
	for _, l := range sec.Lines {
		label := l.Label
		if l.Final {
			label += " (final)"
		}
		if l.Note != "" {
			label += "\n" + l.Note
		}
		let := scheduleMoney(l.TotalLet)
		if l.NotApproved {
			let = "not approved"
		}
		prior := l.Prior
		t.Rows = append(t.Rows, []string{label, let, l.Percent, scheduleMoney(l.Completed), scheduleMoney(&prior), scheduleMoney(l.ThisClaim)})
		if l.BelowClaimed && l.Reason != "" {
			notes = append(notes, len(t.Rows))
			t.Rows = append(t.Rows, []string{"Reason: " + l.Reason, "", "", "", "", ""})
		}
	}
	prior := sec.Prior
	bold := []int{len(t.Rows)}
	t.Rows = append(t.Rows, []string{"Sub total ex GST ($)", scheduleMoney(sec.TotalLet), "", scheduleMoney(sec.Completed), scheduleMoney(&prior), scheduleMoney(sec.ThisClaim)})
	return scheduleBlock(sec.Kind, t, map[string]any{"layout": "schedule", "left": 1, "merge": true, "widths": scheduleLineWidths, "bold": bold, "notes": notes})
}

// scheduleTotals is the total, what retention is held, and what is payable.
// A deduction prints with a minus and a return with a plus, as the screen
// shows them.
func scheduleTotals(sheet invoices.ScheduleSheet, h ScheduleHead) Block {
	t := &Table{Columns: []string{"Total and retention", "Total let ($)", "(%)", "Completed ($)", "Less prior ($)", "This claim ($)"}, Display: "document"}
	bold, notes := []int{}, []int{}
	row := func(strong bool, cells ...string) {
		if strong {
			bold = append(bold, len(t.Rows))
		}
		t.Rows = append(t.Rows, cells)
	}
	prior := sheet.Prior
	row(true, "Total (ex GST)", scheduleMoney(sheet.TotalLet), "", scheduleMoney(sheet.Completed), scheduleMoney(&prior), scheduleMoney(sheet.ThisClaim))
	r := sheet.Retention
	if r.RateBP != nil {
		limit := "unknown"
		if r.Limit != nil {
			limit = "−" + invoices.DisplayCents(*r.Limit)
		}
		row(false, fmt.Sprintf("Retention limit: %s%% of %s", basisPoints(*r.RateBP), retentionBaseWords[r.BaseKind]), limit, "", "", "", "")
	}
	if r.HeldBefore != 0 {
		before := r.HeldBefore
		row(false, "Retention held before this claim", "", "", "", scheduleMoney(&before), "")
	}
	switch {
	case r.Held == nil:
		row(false, "Retention this claim", "", "", "", "", "unknown")
	case *r.Held == 0:
		row(false, "No retention change this claim", "", "", "", "", "–")
	case *r.Held < 0:
		row(false, "Retention returned this claim", "", r.Percent, "", "", "+"+invoices.DisplayCents(-*r.Held))
	default:
		row(false, "Retention held this claim", "", r.Percent, "", "", "−"+invoices.DisplayCents(*r.Held))
	}
	if why := strings.TrimSpace(h.RetentionReason); why != "" {
		notes = append(notes, len(t.Rows))
		row(false, "Reason: "+why, "", "", "", "", "")
	}
	if r.HeldToDate != nil && (*r.HeldToDate != 0 || r.HeldBefore != 0) {
		row(false, "Retention held to date", "", "", scheduleMoney(r.HeldToDate), "", "")
	}
	row(true, "Total claim inc retention (ex GST)", "", "", "", "", scheduleMoney(sheet.Net))
	row(false, "GST", "", "", "", "", scheduleMoney(sheet.GST))
	row(true, "Amount payable (inc GST)", "", "", "", "", scheduleMoney(sheet.Payable))
	return scheduleBlock("totals", t, map[string]any{"layout": "schedule", "left": 1, "merge": true, "widths": scheduleLineWidths, "bold": bold, "notes": notes})
}

// basisPoints prints a rate as a percent: 500 is "5", 250 is "2.5".
func basisPoints(bp int) string {
	s := fmt.Sprintf("%d.%02d", bp/100, bp%100)
	return strings.TrimSuffix(strings.TrimRight(s, "0"), ".")
}

// schedulePayments is the payment list: every counted claim on the contract
// up to this one, and their total.
func schedulePayments(sheet invoices.ScheduleSheet) Block {
	t := &Table{Columns: []string{"Period", "No.", "Inv no.", "Invoice date", "Value ($)", "GST ($)", "Value inc GST ($)"}, Display: "document"}
	for _, p := range sheet.Payments {
		invoice := p.InvoiceNo
		if invoice == "" {
			invoice = "Awaiting invoice"
		}
		if p.Unserved {
			invoice += " (no schedule served)"
		}
		t.Rows = append(t.Rows, []string{p.Period, p.Number, invoice, shortDate(p.InvoiceDate), scheduleMoney(p.Ex), scheduleMoney(p.GST), scheduleMoney(p.Inc)})
	}
	bold := []int{len(t.Rows)}
	t.Rows = append(t.Rows, []string{"Total", "", "", "", scheduleMoney(sheet.Paid.Ex), scheduleMoney(sheet.Paid.GST), scheduleMoney(sheet.Paid.Inc)})
	return scheduleBlock("payments", t, map[string]any{"layout": "schedule", "left": 4, "widths": schedulePaymentWidths, "bold": bold})
}

// shortDate is "30 Sep 2026": a date in a narrow column.
func shortDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("2 Jan 2006")
}
