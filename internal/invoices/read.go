package invoices

import (
	"sort"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // the period is Sydney time on any host

	"sitewise/internal/jev"
)

// Fields of an invoice. Jev may be asked the ones in JevFields; ex GST,
// period and currency are code.
const (
	FieldKind     = "kind"
	FieldNumber   = "number"
	FieldDate     = "date"
	FieldABN      = "issuer_abn"
	FieldTotal    = "total"
	FieldGST      = "gst"
	FieldExGST    = "ex_gst"
	FieldSupplier = "supplier"
	FieldLine     = "line"
	FieldPeriod   = "period"
	FieldCurrency = "currency"

	BandGreen = "green"
	BandAmber = "amber"
	BandBlank = "blank"
	BandGrey  = "grey"

	DecidedByRule = "rule"
	DecidedByJev  = "jev"
	DecidedByUser = "user"

	choiceNone = "none"
)

// JevFields are the questions one fan-out may carry, in a fixed order.
var JevFields = []string{FieldKind, FieldNumber, FieldDate, FieldABN, FieldTotal, FieldGST, FieldSupplier, FieldLine}

// AllFields is every stored decision, in display order.
var AllFields = []string{FieldKind, FieldNumber, FieldSupplier, FieldABN, FieldDate, FieldPeriod, FieldLine, FieldExGST, FieldGST, FieldTotal, FieldCurrency}

// Kind options. Only invoice and credit note count in the ledger; every
// other kind stays out of totals until the user says otherwise.
var kindOptions = []struct{ ID, Describe string }{
	{"invoice", "A tax invoice or invoice asking to be paid, including a progress claim issued as a tax invoice."},
	{"credit_note", "A credit or adjustment note that reduces an amount already invoiced."},
	{"quote", "A quote, quotation or fee proposal for future work."},
	{"statement", "A statement of account listing several invoices and payments."},
	{"remittance", "A remittance advice confirming a payment that was made."},
	{"receipt", "A receipt for a payment already made, such as a levy or council fee receipt."},
	{"payment_schedule", "A payment schedule or claim assessment that is not itself an invoice."},
	{"other", "Any other document."},
}

// LedgerKind maps a kind decision to the ledger's kind column.
func LedgerKind(v string) string {
	switch v {
	case "":
		return ""
	case "invoice", "credit_note":
		return v
	default:
		return "not_invoice"
	}
}

// Decision is one field. Value is empty when nothing was applied.
type Decision struct {
	Field           string   `json:"field"`
	Value           string   `json:"value"`
	Band            string   `json:"band"`
	DecidedBy       string   `json:"decided_by"`
	QuestionVersion string   `json:"question_version,omitempty"`
	Confidence      *float64 `json:"confidence,omitempty"`
}

// Package is a procurement package an invoice may come from.
type Package struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	Discipline string `json:"discipline"`
	ABN        string `json:"abn"`
}

// Line is a live posting cost plan line an invoice may be charged to.
type Line struct {
	ID        string
	Label     string
	Section   string
	PackageID string
	Category  string
}

// Context is the project an invoice is read into. History maps a package to
// the distinct live lines earlier user-confirmed invoices from it went to.
type Context struct {
	Packages []Package
	Lines    []Line
	History  map[string][]string
}

type choice struct {
	ID       string
	Value    string
	Describe any
}

type question struct {
	Field        string
	Instructions any
	Options      []choice
}

// roleQuestion names the role the value plays and what it is not, so
// similar candidates are told apart.
// https://docs.typesafe.ai/primitives/advanced
type roleQuestion struct {
	Question string `json:"question"`
	NotFor   string `json:"not_for"`
}

var instructions = map[string]any{
	FieldKind:   "What kind of document is this? Choose by what the document is, not by the words in its payment terms.",
	FieldNumber: roleQuestion{"Which candidate is this invoice's own invoice number, as the supplier would quote it on a remittance?", "A purchase order, job, project, account, customer or reference number, an ABN, a phone number, or another invoice's number listed in a claim history."},
	FieldDate:   roleQuestion{"Which candidate is the date this invoice was issued?", "A due date, payment date, period end, the date of a referenced fee proposal or order, or a print date. Choose none when the issue date is not among the candidates."},
	FieldABN:    roleQuestion{"Which ABN belongs to the business that issued this invoice and is asking to be paid?", "The customer's or client's ABN, a trustee's ABN, or a payment processor's ABN."},
	FieldTotal:  roleQuestion{"Which candidate is the total amount payable for this invoice, including GST?", "A contract value, a fee or claim to date, a previously claimed amount, the subtotal before GST, the GST amount alone, or an amount from a claim history table."},
	FieldGST:    roleQuestion{"Which candidate is the GST charged on this invoice?", "The total, the subtotal, a contract value, or GST on an earlier claim. Choose none when the invoice prints no GST amount."},
	FieldSupplier: roleQuestion{"Which of the project's consultant and contractor packages is this invoice from, judged by the work it charges for and who issued it?",
		"A package for different work. Choose none when no package matches."},
	FieldLine: roleQuestion{"Which cost plan line does this invoice's work belong to?", "A line for different work or a different consultant or contractor. Choose none when no line fits."},
}

var noneDescribe = map[string]string{"what": "None of these."}

// Draft is one invoice reading in progress: rules first, then at most one
// Jev fan-out for what is still open, then code arithmetic.
type Draft struct {
	ev        Evidence
	ctx       Context
	by        map[string]Decision
	questions map[string]question
}

// NewDraft settles what rules can. users holds the user's own decisions,
// which are never replaced.
func NewDraft(ev Evidence, ctx Context, users map[string]Decision) *Draft {
	d := &Draft{ev: ev, ctx: ctx, by: map[string]Decision{}, questions: map[string]question{}}
	for f, u := range users {
		d.by[f] = u
	}
	set := func(field, value string) {
		if _, user := d.by[field]; user || value == "" {
			return
		}
		d.by[field] = Decision{Field: field, Value: value, Band: BandGreen, DecidedBy: DecidedByRule}
	}
	set(FieldKind, kindRule(ev.KindWords))
	set(FieldNumber, displayNumber(ev, uniqueLabeled(ev.Numbers)))
	set(FieldDate, uniqueLabeled(ev.Dates))
	// No ABN rule: the only ABN printed can be the customer's, so the
	// issuer's ABN is always Jev's pick (same fan-out, no extra call).
	if e, g, t, ok := amountRule(ev.Amounts); ok {
		set(FieldTotal, strconv.FormatInt(t, 10))
		set(FieldGST, strconv.FormatInt(g, 10))
		_ = e
	}
	d.settleSupplier()
	d.settleLine()
	for _, f := range JevFields {
		if _, done := d.by[f]; done {
			continue
		}
		if q, ok := d.question(f); ok {
			d.questions[f] = q
		}
	}
	return d
}

func (d *Draft) settleSupplier() {
	if _, done := d.by[FieldSupplier]; done {
		return
	}
	abn := d.by[FieldABN]
	if abn.Value == "" || abn.Band != BandGreen && abn.DecidedBy != DecidedByUser {
		return
	}
	var match []string
	for _, p := range d.ctx.Packages {
		if p.ABN != "" && p.ABN == abn.Value {
			match = append(match, p.ID)
		}
	}
	if len(match) == 1 {
		d.by[FieldSupplier] = Decision{Field: FieldSupplier, Value: match[0], Band: BandGreen, DecidedBy: DecidedByRule}
	}
}

// settleLine uses the supplier's history, then its only live line. A Jev
// supplier carries its band to the derived line; a weak supplier derives
// nothing.
func (d *Draft) settleLine() {
	if cur, done := d.by[FieldLine]; done && (cur.DecidedBy == DecidedByUser || cur.DecidedBy == DecidedByRule) {
		return
	}
	sup := d.by[FieldSupplier]
	if sup.Value == "" || (sup.Band != BandGreen && sup.Band != BandAmber) {
		return
	}
	live := map[string]bool{}
	var own []string
	for _, l := range d.ctx.Lines {
		live[l.ID] = true
		if l.PackageID == sup.Value {
			own = append(own, l.ID)
		}
	}
	var hist []string
	for _, id := range d.ctx.History[sup.Value] {
		if live[id] {
			hist = append(hist, id)
		}
	}
	pick := ""
	switch {
	case len(hist) == 1:
		pick = hist[0]
	case len(hist) == 0 && len(own) == 1:
		pick = own[0]
	}
	if pick == "" {
		return
	}
	d.by[FieldLine] = Decision{Field: FieldLine, Value: pick, Band: sup.Band, DecidedBy: DecidedByRule}
}

func kindRule(words []string) string {
	has := map[string]bool{}
	for _, w := range words {
		has[w] = true
	}
	if !has["tax invoice"] {
		return ""
	}
	for _, w := range []string{"credit note", "adjustment note", "credit memo", "quote", "quotation", "fee proposal", "statement", "remittance", "receipt", "payment schedule"} {
		if has[w] {
			return ""
		}
	}
	return "invoice"
}

func uniqueLabeled(cs []Candidate) string {
	v := ""
	for _, c := range cs {
		if !c.Labeled {
			continue
		}
		if v != "" && v != c.Value {
			return ""
		}
		v = c.Value
	}
	return v
}

func totalish(labels []string) bool {
	for _, l := range labels {
		if (strings.Contains(l, "total") && !strings.Contains(l, "sub")) || strings.Contains(l, "amount due") || strings.Contains(l, "balance due") || strings.Contains(l, "payable") || strings.Contains(l, "total due") {
			if strings.Contains(l, "gst") && !strings.Contains(l, "inc") {
				continue
			}
			return true
		}
	}
	return false
}

func gstish(labels []string) bool {
	for _, l := range labels {
		if (strings.Contains(l, "gst") || strings.Contains(l, "tax")) && !strings.Contains(l, "inc") && !strings.Contains(l, "ex ") && !strings.Contains(l, "excl") && !strings.Contains(l, "ex.") && !strings.Contains(l, "invoice") {
			return true
		}
	}
	return false
}

// amountRule settles ex GST, GST and total only when the page holds exactly
// one consistent set: a GST-labelled amount, the ex-GST amount it is 10 % of,
// and their sum labelled as a total. A second consistent set anywhere on the
// page (a contract summary, an earlier claim) leaves the amounts to Jev.
func amountRule(amounts []Candidate) (ex, gst, total int64, ok bool) {
	type triple struct{ e, g, t int64 }
	present := map[int64]bool{}
	for _, c := range amounts {
		v, _ := strconv.ParseInt(c.Value, 10, 64)
		present[v] = true
	}
	loose := map[triple]bool{}
	var strong []triple
	for _, t := range amounts {
		tv, _ := strconv.ParseInt(t.Value, 10, 64)
		if tv <= 0 {
			continue
		}
		for _, g := range amounts {
			gv, _ := strconv.ParseInt(g.Value, 10, 64)
			if gv <= 0 || gv >= tv || !gstish(g.Labels) {
				continue
			}
			e := tv - gv
			if abs(GSTOf(e)-gv) > 1 || !present[e] {
				continue
			}
			tr := triple{e, gv, tv}
			if !loose[tr] && totalish(t.Labels) {
				strong = append(strong, tr)
			}
			loose[tr] = true
		}
	}
	if len(loose) != 1 || len(strong) != 1 {
		return 0, 0, 0, false
	}
	return strong[0].e, strong[0].g, strong[0].t, true
}

func (d *Draft) question(field string) (question, bool) {
	var opts []choice
	switch field {
	case FieldKind:
		for _, k := range kindOptions {
			opts = append(opts, choice{ID: k.ID, Value: k.ID, Describe: k.Describe})
		}
		return question{Field: field, Instructions: instructions[field], Options: opts}, true
	case FieldNumber:
		// The number is stored as printed, so the option carries the display.
		opts = candidateOptions(d.ev.Numbers, nil)
		for i := range opts {
			opts[i].Value = d.ev.Numbers[i].Display
		}
	case FieldDate:
		opts = candidateOptions(d.ev.Dates, nil)
	case FieldABN:
		opts = candidateOptions(d.ev.ABNs, nil)
	case FieldTotal:
		opts = candidateOptions(d.ev.Amounts, func(c Candidate) bool { return !strings.HasPrefix(c.Value, "-") && c.Value != "0" })
	case FieldGST:
		// A printed $0.00 GST is a GST-free invoice, so zero is a choice.
		opts = candidateOptions(d.ev.Amounts, func(c Candidate) bool { return !strings.HasPrefix(c.Value, "-") })
	case FieldSupplier:
		for _, p := range d.ctx.Packages {
			role := map[string]string{"services": "consultant", "works": "contractor", "supply": "supplier"}[p.Kind]
			opts = append(opts, choice{ID: "p" + strconv.Itoa(len(opts)+1), Value: p.ID, Describe: map[string]string{"package": p.Title, "role": role}})
		}
	case FieldLine:
		for _, l := range d.ctx.Lines {
			desc := map[string]string{"line": l.Label, "section": l.Section}
			if l.Category != "" {
				desc["category"] = l.Category
			}
			opts = append(opts, choice{ID: "l" + strconv.Itoa(len(opts)+1), Value: l.ID, Describe: desc})
		}
	}
	if len(opts) == 0 || len(opts)+1 > jev.MaxChoiceOptions {
		return question{}, false
	}
	opts = append(opts, choice{ID: choiceNone, Describe: noneDescribe})
	return question{Field: field, Instructions: instructions[field], Options: opts}, true
}

func candidateOptions(cs []Candidate, keep func(Candidate) bool) []choice {
	var out []choice
	for _, c := range cs {
		if keep != nil && !keep(c) {
			continue
		}
		found := strings.Join(c.Labels, "; ")
		if found == "" {
			found = "page text"
		}
		out = append(out, choice{ID: "c" + strconv.Itoa(len(out)+1), Value: c.Value, Describe: map[string]string{"value": c.Display, "printed_with": clip(found, 160)}})
		if len(out) == 63 {
			break
		}
	}
	return out
}

type state struct {
	Filename string `json:"filename"`
	Text     string `json:"invoice_text"`
}

// Call is the one fan-out for this invoice, or false when rules settled
// everything Jev could answer.
func (d *Draft) Call() (jev.Call, bool) {
	return d.call(false)
}

// CallAll asks every question that has options, settled or not. Only the
// accuracy eval uses it, to calibrate questions rules usually answer.
func (d *Draft) CallAll() (jev.Call, bool) {
	return d.call(true)
}

func (d *Draft) call(all bool) (jev.Call, bool) {
	qs := map[string]jev.Question{}
	for _, f := range JevFields {
		q, ok := d.questions[f]
		if !ok && all {
			q, ok = d.question(f)
			if ok {
				d.questions[f] = q
			}
		}
		if !ok {
			continue
		}
		criteria := map[string]any{}
		for _, o := range q.Options {
			criteria[o.ID] = o.Describe
		}
		qs[f] = jev.Question{Type: jev.TypeChoice, Instructions: q.Instructions, Criteria: criteria}
	}
	if len(qs) == 0 {
		return jev.Call{}, false
	}
	return jev.Call{
		State:           state{Filename: d.ev.Filename, Text: d.ev.Excerpt},
		Questions:       qs,
		QuestionVersion: QuestionVersion,
		Priority:        jev.PriorityInteractive,
	}, true
}

// Options is the option count Jev saw for a field, for calibration.
func (d *Draft) Options(field string) int {
	return len(d.questions[field].Options)
}

// Raw is what Jev picked for a field before any threshold, for calibration.
func (d *Draft) Raw(res jev.Result, field string) (value string, confidence *float64, ok bool) {
	q, asked := d.questions[field]
	a, answered := res.Answers[field]
	if !asked || !answered || a.Type != jev.TypeChoice {
		return "", nil, false
	}
	for _, o := range q.Options {
		if o.ID == a.Choice {
			return o.Value, a.Confidence, true
		}
	}
	return "", nil, false
}

// Apply bands Jev's answers. A failed call leaves rule values and marks the
// unanswered questions grey; grey is true so the caller re-checks later.
func (d *Draft) Apply(res jev.Result, err error, th Thresholds) (grey bool) {
	for f, q := range d.questions {
		if cur, done := d.by[f]; done && cur.DecidedBy != DecidedByJev {
			continue // rule or user settled it; the eval asked anyway
		}
		dec := Decision{Field: f, Band: BandBlank, DecidedBy: DecidedByJev, QuestionVersion: QuestionVersion}
		if err != nil {
			dec.Band = BandGrey
			grey = true
			d.by[f] = dec
			continue
		}
		a, ok := res.Answers[f]
		if ok && a.Type == jev.TypeChoice {
			for _, o := range q.Options {
				if o.ID != a.Choice {
					continue
				}
				dec.Confidence = a.Confidence
				if band, apply := th.Band(f, a.Confidence, len(q.Options)); apply && o.ID != choiceNone {
					dec.Band, dec.Value = band, o.Value
				}
			}
		}
		d.by[f] = dec
	}
	d.settleLine()
	return grey
}

// Reading is the code result of one invoice read.
type Reading struct {
	Kind         string
	Number       string
	Date         string
	ABN          string
	PackageID    string
	LineID       string
	LineBand     string
	LineBy       string
	Currency     string
	Period       string
	Total        *int64
	GST          *int64
	Ex           *int64
	InvoiceCount int
	Decisions    []Decision
}

// Finish adds the code fields (period, currency, ex GST) and returns the
// reading. uploaded is the upload time; the period is its month in Sydney.
func (d *Draft) Finish(uploaded time.Time) Reading {
	if _, ok := d.by[FieldPeriod]; !ok {
		d.by[FieldPeriod] = Decision{Field: FieldPeriod, Value: PeriodOf(uploaded), Band: BandGreen, DecidedBy: DecidedByRule}
	}
	if _, ok := d.by[FieldCurrency]; !ok {
		if d.ev.Currency != "" {
			d.by[FieldCurrency] = Decision{Field: FieldCurrency, Value: d.ev.Currency, Band: BandAmber, DecidedBy: DecidedByRule}
		} else {
			d.by[FieldCurrency] = Decision{Field: FieldCurrency, Value: "AUD", Band: BandGreen, DecidedBy: DecidedByRule}
		}
	}
	r := Resolve(d.by)
	r.InvoiceCount = d.ev.InvoiceCount
	return r
}

// Resolve turns decisions into ledger values. Ex GST is total − GST unless
// the user typed it; a credit note's amounts are negative whatever sign was
// printed. It is the one place ledger columns are computed, after a read and
// after a user edit alike. by gains the derived ex GST decision.
func Resolve(by map[string]Decision) Reading {
	var r Reading
	total, gst := by[FieldTotal], by[FieldGST]
	if ex, ok := by[FieldExGST]; !ok || ex.DecidedBy != DecidedByUser {
		dec := Decision{Field: FieldExGST, Band: BandBlank, DecidedBy: DecidedByRule}
		if t, ok1 := cents(total.Value); ok1 {
			if g, ok2 := cents(gst.Value); ok2 {
				dec.Value = strconv.FormatInt(t-g, 10)
				dec.Band = weaker(total.Band, gst.Band)
				if total.DecidedBy == DecidedByUser && gst.DecidedBy == DecidedByUser {
					dec.Band = BandGreen
				}
			}
		}
		by[FieldExGST] = dec
	}
	r.Kind = LedgerKind(by[FieldKind].Value)
	r.Number = by[FieldNumber].Value
	r.Date = by[FieldDate].Value
	r.ABN = by[FieldABN].Value
	r.PackageID = by[FieldSupplier].Value
	r.LineID = by[FieldLine].Value
	r.LineBand, r.LineBy = by[FieldLine].Band, by[FieldLine].DecidedBy
	if r.LineBand == "" {
		r.LineBand, r.LineBy = BandBlank, DecidedByRule
	}
	r.Currency = by[FieldCurrency].Value
	if r.Currency == "" {
		r.Currency = "AUD"
	}
	r.Period = by[FieldPeriod].Value
	r.Total, r.GST, r.Ex = centsPtr(total.Value), centsPtr(gst.Value), centsPtr(by[FieldExGST].Value)
	if r.Kind == "credit_note" {
		for _, p := range []*int64{r.Total, r.GST, r.Ex} {
			if p != nil && *p > 0 {
				*p = -*p
			}
		}
	}
	fields := make([]string, 0, len(by))
	for f := range by {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		r.Decisions = append(r.Decisions, by[f])
	}
	return r
}

// displayNumber returns the number as printed, from its normalized value.
func displayNumber(ev Evidence, v string) string {
	if v == "" {
		return ""
	}
	for _, c := range ev.Numbers {
		if c.Value == v && c.Labeled {
			return c.Display
		}
	}
	for _, c := range ev.Numbers {
		if c.Value == v {
			return c.Display
		}
	}
	return v
}

var bandRank = map[string]int{BandGreen: 3, BandAmber: 2, BandGrey: 1, BandBlank: 0, "": 0}

func weaker(a, b string) string {
	if bandRank[a] < bandRank[b] {
		return a
	}
	return b
}

func cents(v string) (int64, bool) {
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 64)
	return n, err == nil
}

func centsPtr(v string) *int64 {
	n, ok := cents(v)
	if !ok {
		return nil
	}
	return &n
}

// Sydney is the projects' local time. Version 1 projects are Australian;
// the period an invoice is certified in is its upload month there.
var Sydney = mustZone("Australia/Sydney")

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.FixedZone("AEST", 10*3600)
	}
	return loc
}

// PeriodOf is the first day of t's month in Sydney, as YYYY-MM-01.
func PeriodOf(t time.Time) string {
	return t.In(Sydney).Format("2006-01") + "-01"
}
