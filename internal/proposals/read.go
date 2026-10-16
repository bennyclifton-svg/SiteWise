package proposals

import (
	"context"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/invoices"
	"sitewise/internal/jev"
	"sitewise/internal/rfp"
)

// Bands and deciders, as in filing.
const (
	BandGreen = "green"
	BandAmber = "amber"
	BandBlank = "blank"
	BandGrey  = "grey"

	ByRule = "rule"
	ByJev  = "jev"
	ByUser = "user"
)

// Fields. A schedule item's mapping is "price:<key>" (a fee or a rate); a
// non-price criterion's passage is its own key, "criterion:<returnable>".
const (
	FieldKind        = "kind"
	FieldPackage     = "package"
	FieldFirmName    = "firm_name"
	FieldFirmABN     = "firm_abn"
	FieldFirmEmail   = "firm_email"
	FieldFirmPhone   = "firm_phone"
	FieldFirmAddress = "firm_address"
	FieldFirmContact = "firm_contact"
	FieldStatedTotal = "stated_total"
	FieldGSTBasis    = "gst_basis"
	PricePrefix      = "price:"
)

// FirmFields are the firm's details in display order.
var FirmFields = []string{FieldFirmName, FieldFirmABN, FieldFirmContact, FieldFirmEmail, FieldFirmPhone, FieldFirmAddress}

// Kinds of submission. Not a proposal leaves the package's firm columns.
const (
	KindFeeProposal = "fee_proposal"
	KindTender      = "tender"
	KindNone        = "not_proposal"
)

// Price states.
const (
	StatePriced    = "priced"
	StateNotPriced = "not_priced"
	StateIncluded  = "included"
	StateExcluded  = "excluded"
)

// Flags code raises on a submission; each has a plain reason on screen.
const (
	FlagRatesOnly    = "rates_only"
	FlagOptionsAB    = "options_ab"
	FlagGSTNotStated = "gst_not_stated"
	FlagGSTInc       = "gst_inc_normalised"
	FlagTotalGap     = "total_gap"
	FlagNoText       = "no_text"
	FlagNoSchedule   = "no_schedule"
	FlagResponseForm = "response_form"
)

const (
	choiceNone     = "none"
	choiceNotPrice = "not_priced"
	choiceIncluded = "included"
)

// Decision is one field's judgement. Value is empty when nothing applied.
type Decision struct {
	Field           string   `json:"field"`
	Value           string   `json:"value"`
	Band            string   `json:"band"`
	DecidedBy       string   `json:"decided_by"`
	QuestionVersion string   `json:"question_version,omitempty"`
	Confidence      *float64 `json:"confidence,omitempty"`
}

// Package is one of the project's live packages a proposal may answer.
type Package struct {
	ID         string
	Title      string
	Discipline string
}

// Firm is a directory firm. The caller looks it up by ABN or name key.
type Firm struct {
	ID      string
	Name    string
	ABN     string
	Address string
	Contact string
	Email   string
	Phone   string
}

// Context is the project a proposal is read into.
type Context struct {
	Packages []Package
	// Discipline is the document's filed discipline, or "".
	Discipline string
	// PackageID is set when the user dropped the file on a package row.
	PackageID string
	// Schedules holds each candidate package's live schedule.
	Schedules map[string][]rfp.ScheduleItem
	// OwnDomains are the org's own email domains: never the firm's.
	OwnDomains []string
	// SiteAddress is the project's address: never the firm's.
	SiteAddress string
	// ConstructionBudget is the cost plan's construction budget in cents,
	// the basis for a percentage fee; nil is unknown.
	ConstructionBudget *int64
	// Lookup finds a directory firm by any of these ABNs or name keys.
	Lookup func(abns, nameKeys []string) *Firm
}

// Asker is the one System One call a proposal read may make.
type Asker interface {
	Ask(context.Context, jev.Call) (jev.Result, error)
}

type choice struct {
	ID       string
	Value    string
	Describe any
	// Short replaces Describe when the full request is over the size budget.
	Short any
}

type question struct {
	Field        string
	Family       string
	Instructions any
	Options      []choice
}

// roleQuestion names the role the value plays and what it is not.
// https://docs.typesafe.ai/primitives/advanced
type roleQuestion struct {
	Question string `json:"question"`
	NotFor   string `json:"not_for"`
}

// Draft is one proposal reading in progress.
type Draft struct {
	ev        Evidence
	ctx       Context
	by        map[string]Decision
	questions map[string]question
	firm      *Firm
	items     []rfp.ScheduleItem
	flags     map[string]bool
}

// NewDraft settles what rules can. users holds the user's own decisions,
// which are never replaced.
func NewDraft(ev Evidence, pc Context, users map[string]Decision) *Draft {
	d := &Draft{ev: ev, ctx: pc, by: map[string]Decision{}, questions: map[string]question{}, flags: map[string]bool{}}
	for f, u := range users {
		d.by[f] = u
	}
	d.rule(FieldKind, kindRule(ev, pc.Discipline))
	d.settlePackage()
	d.settleFirm()
	d.settleGST()
	d.settleTotal()
	if pkg := d.by[FieldPackage].Value; pkg != "" {
		d.items = pc.Schedules[pkg]
		if len(d.items) == 0 {
			d.flags[FlagNoSchedule] = true
		}
		d.settlePrices()
	}
	if ev.OptionsAB {
		d.flags[FlagOptionsAB] = true
	}
	d.ask()
	return d
}

func (d *Draft) rule(field, value string) {
	if _, done := d.by[field]; done || value == "" {
		return
	}
	d.by[field] = Decision{Field: field, Value: value, Band: BandGreen, DecidedBy: ByRule}
}

// kindRule settles the kind when the document calls itself a proposal or a
// quote and the filed discipline says which: a consultant's quotation is a
// fee proposal, a trade's is a tender. "Fee" or "tender" alone is not
// enough: a tender evaluation report says both.
func kindRule(ev Evidence, discipline string) string {
	has := map[string]bool{}
	for _, w := range ev.KindWords {
		has[w] = true
	}
	strong := ev.Form != nil || has["fee proposal"] || has["fee submission"] || has["fee estimate"] || has["fee quote"] || has["fee schedule"] || has["quotation"] || has["quote"] || has["tender submission"]
	if !strong || (len(ev.Amounts) == 0 && ev.Form == nil) {
		return ""
	}
	switch {
	case strings.HasPrefix(discipline, "consultant."):
		return KindFeeProposal
	case strings.HasPrefix(discipline, "trade.") || strings.HasPrefix(discipline, "supplier."):
		return KindTender
	}
	return ""
}

func (d *Draft) settlePackage() {
	if d.ctx.PackageID != "" {
		if _, done := d.by[FieldPackage]; !done {
			d.by[FieldPackage] = Decision{Field: FieldPackage, Value: d.ctx.PackageID, Band: BandGreen, DecidedBy: ByUser}
		}
		return
	}
	var match []string
	for _, p := range d.ctx.Packages {
		if d.ctx.Discipline != "" && p.Discipline == d.ctx.Discipline {
			match = append(match, p.ID)
		}
	}
	if len(match) == 1 {
		d.rule(FieldPackage, match[0])
	}
}

// settleFirm matches the directory by ABN or name. A match fills every firm
// field from the directory, so no firm question is asked.
func (d *Draft) settleFirm() {
	abns := d.ev.ABNs
	if u := d.by[FieldFirmABN]; u.DecidedBy == ByUser && u.Value != "" {
		abns = []string{u.Value}
	}
	var keys []string
	for _, n := range d.ev.Names {
		if k := NameKey(n); k != "" {
			keys = append(keys, k)
		}
	}
	if u := d.by[FieldFirmName]; u.DecidedBy == ByUser && u.Value != "" {
		keys = []string{NameKey(u.Value)}
	}
	if d.ctx.Lookup != nil {
		if f := d.ctx.Lookup(abns, keys); f != nil {
			d.firm = f
			for field, v := range map[string]string{FieldFirmName: f.Name, FieldFirmABN: f.ABN, FieldFirmAddress: f.Address, FieldFirmContact: f.Contact, FieldFirmEmail: f.Email, FieldFirmPhone: f.Phone} {
				d.rule(field, v)
			}
			return
		}
	}
	if len(d.ev.ABNs) == 1 {
		d.rule(FieldFirmABN, d.ev.ABNs[0])
	}
	d.rule(FieldFirmName, d.corroboratedName())
	if f := d.ev.FormFirm; f != nil {
		for field, label := range map[string]string{FieldFirmName: "Firm", FieldFirmContact: "Contact", FieldFirmEmail: "Email", FieldFirmPhone: "Phone", FieldFirmAddress: "Address"} {
			d.rule(field, strings.TrimSpace(f[label]))
		}
		if abn := invoices.NormalizeABN(f["ABN"]); abn != "" && invoices.ValidABN(abn) {
			d.rule(FieldFirmABN, abn)
		}
	}
	if e := d.firmEmails(); len(e) == 1 {
		d.rule(FieldFirmEmail, e[0])
	}
	if p := d.ev.Phones; len(p) == 1 {
		d.rule(FieldFirmPhone, p[0])
	}
	if a := d.firmAddresses(); len(a) == 1 {
		d.rule(FieldFirmAddress, a[0])
	}
	if c := d.ev.Contacts; len(c) == 1 {
		d.rule(FieldFirmContact, c[0])
	}
}

// corroboratedName is the one printed company name that the filename's
// bracketed name or the firm's email domain also names: "Fee Traffic
// (Varga).pdf" and "Varga Traffic Planning Pty Ltd". Two such names, or
// none, leave the firm to Jev.
func (d *Draft) corroboratedName() string {
	var hints []string
	for _, h := range d.ev.Hints {
		if k := NameKey(h); len(k) >= 3 {
			hints = append(hints, k)
		}
	}
	own := map[string]bool{}
	for _, o := range d.ctx.OwnDomains {
		label, _, _ := strings.Cut(o, ".")
		own[NameKey(label)] = true
	}
	best, bestKey := "", ""
	for _, n := range d.ev.Names[:min(d.ev.Printed, len(d.ev.Names))] {
		k := NameKey(n)
		if len(k) < 3 || len(n) > 60 || !companyRe.MatchString(n) {
			continue
		}
		for _, h := range hints {
			if own[h] || !strings.Contains(k, h) {
				continue
			}
			switch {
			case bestKey == "":
				best, bestKey = n, k
			case strings.Contains(k, bestKey) || strings.Contains(bestKey, k):
				if len(n) > len(best) {
					best = n
				}
			default:
				return ""
			}
		}
	}
	return best
}

func (d *Draft) firmEmails() []string {
	var out []string
	for _, e := range d.ev.Emails {
		_, domain, _ := strings.Cut(e, "@")
		own := false
		for _, o := range d.ctx.OwnDomains {
			own = own || strings.EqualFold(domain, o)
		}
		if !own {
			out = append(out, e)
		}
	}
	return out
}

func (d *Draft) firmAddresses() []string {
	site := NameKey(d.ctx.SiteAddress)
	var out []string
	for _, a := range d.ev.Addresses {
		if site != "" && strings.Contains(NameKey(a), site[:min(len(site), 12)]) {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (d *Draft) settleGST() {
	switch {
	case d.ev.GSTEx > 0 && d.ev.GSTInc == 0:
		d.rule(FieldGSTBasis, "ex")
	case d.ev.GSTInc > 0 && d.ev.GSTEx == 0:
		d.rule(FieldGSTBasis, "inc")
	case d.ev.GSTEx == 0 && d.ev.GSTInc == 0:
		// Australian consultant fees are quoted ex GST unless stated.
		d.rule(FieldGSTBasis, "ex")
		d.flags[FlagGSTNotStated] = true
	}
}

func (d *Draft) settleTotal() {
	if v := d.ev.Form["base_total"]; v != "" {
		if c, ok, err := invoices.ParseCents(v); ok && err == nil {
			d.rule(FieldStatedTotal, strconv.FormatInt(c, 10))
			return
		}
	}
	var totals []Amount
	var largest int64
	for _, a := range d.ev.Amounts {
		if a.PerHour || a.Kind != "" || a.Unit || a.GSTPart || a.IncTwin {
			continue
		}
		if a.Total && a.GST != "inc" {
			totals = append(totals, a)
		}
		largest = max(largest, a.Cents)
	}
	// One total, and no fee on the page larger than it: a disbursements
	// total beside a larger certification fee is not the stated total.
	if distinctCents(totals) == 1 && totals[0].Cents >= largest {
		d.rule(FieldStatedTotal, strconv.FormatInt(totals[0].Cents, 10))
	}
}

// settlePrices reads a returned response form by key, and marks every fee
// row not priced when the proposal holds hourly rates only.
func (d *Draft) settlePrices() {
	if d.ev.Form != nil {
		d.flags[FlagResponseForm] = true
		for _, it := range d.items {
			if !it.Price() && it.Kind != rfp.ItemRate {
				continue
			}
			v, ok := d.ev.Form[it.Key]
			if !ok {
				continue
			}
			d.rule(PricePrefix+it.Key, formValue(v, it.Key))
		}
		return
	}
	fees := 0
	for _, a := range d.ev.Amounts {
		if !a.PerHour {
			fees++
		}
	}
	if fees == 0 && len(d.ev.Amounts) > 0 {
		d.flags[FlagRatesOnly] = true
		for _, it := range d.items {
			if it.Price() {
				d.rule(PricePrefix+it.Key, choiceNotPrice)
			}
		}
	}
}

var includedIn = regexp.MustCompile(`(?i)^\s*incl`)

// formValue reads one response-form cell: an amount, "Not priced" or
// "Included in <ref>".
func formValue(v, key string) string {
	v = strings.TrimSpace(v)
	switch {
	case v == "":
		return choiceNotPrice
	case strings.HasPrefix(strings.ToLower(v), "not"):
		return choiceNotPrice
	case includedIn.MatchString(v):
		return choiceIncluded
	}
	if c, ok, err := invoices.ParseCents(v); ok && err == nil {
		return "cents:" + strconv.FormatInt(c, 10)
	}
	return ""
}

func distinctCents(as []Amount) int {
	seen := map[int64]bool{}
	for _, a := range as {
		seen[a.Cents] = true
	}
	return len(seen)
}

var instructions = map[string]any{
	FieldKind: "What kind of document is this? A fee proposal or fee submission offers a consultant's services for a fee. A tender or quotation offers construction work or supply for a price. Anything else, including an invoice, a report, a contract or an RFP issued to firms, is not a proposal.",
	FieldPackage: roleQuestion{"Which of the project's packages does this proposal offer to deliver, judged by the services or work it prices?",
		"A package for different work. Choose none when no package matches."},
	FieldFirmName: roleQuestion{"Which candidate is the name of the firm that submitted this proposal and would be engaged?",
		"The client, the project manager, the project or site, a subconsultant, an insurer, a referee's company, or a heading."},
	FieldFirmABN:     roleQuestion{"Which ABN belongs to the firm that submitted this proposal?", "The client's ABN, an insurer's, a subconsultant's or a related company's."},
	FieldFirmEmail:   roleQuestion{"Which email address is the submitting firm's contact for this proposal?", "The client's or project manager's address, or a referee's."},
	FieldFirmPhone:   roleQuestion{"Which phone number is the submitting firm's?", "The client's, a referee's or an insurer's number."},
	FieldFirmAddress: roleQuestion{"Which address is the submitting firm's office address?", "The project site, the client's address or a past project's address."},
	FieldFirmContact: roleQuestion{"Which person signed or is the contact for this proposal at the submitting firm?", "A client representative, a referee or a past project contact."},
	FieldStatedTotal: roleQuestion{"Which amount is the firm's stated total for its base fee, excluding GST?",
		"An amount including GST, a total of optional services only, a subtotal of one stage, an hourly rate or insurance cover. Choose none when the proposal states no overall total."},
	FieldGSTBasis: "Are the fees in this proposal stated excluding GST or including GST? Answer from the wording printed with the fee schedule.",
}

var noneDescribe = map[string]string{"what": "None of these."}

// ask builds the questions still open, in one fan-out.
func (d *Draft) ask() {
	add := func(q question) {
		if _, done := d.by[q.Field]; done {
			return
		}
		if len(q.Options) < 2 || len(q.Options) > jev.MaxChoiceOptions {
			return
		}
		d.questions[q.Field] = q
	}
	if _, ok := d.by[FieldKind]; !ok {
		add(question{Field: FieldKind, Family: FieldKind, Instructions: instructions[FieldKind], Options: []choice{
			{ID: KindFeeProposal, Value: KindFeeProposal, Describe: "A fee proposal or fee submission for consultant services."},
			{ID: KindTender, Value: KindTender, Describe: "A tender, quotation or quote for construction work or supply."},
			{ID: KindNone, Value: KindNone, Describe: "Not a proposal or tender."},
		}})
	}
	if _, ok := d.by[FieldPackage]; !ok && len(d.ctx.Packages) > 0 {
		opts := []choice{}
		for _, p := range d.ctx.Packages {
			opts = append(opts, choice{ID: p.ID, Value: p.ID, Describe: map[string]string{"package": p.Title}})
		}
		add(question{Field: FieldPackage, Family: FieldPackage, Instructions: instructions[FieldPackage], Options: append(opts, choice{ID: choiceNone, Describe: noneDescribe})})
	}
	if d.firm == nil {
		d.firmQuestion(FieldFirmName, d.ev.Names, 40, add)
		d.firmQuestion(FieldFirmABN, d.ev.ABNs, 20, add)
		d.firmQuestion(FieldFirmEmail, d.firmEmails(), 20, add)
		d.firmQuestion(FieldFirmPhone, d.ev.Phones, 20, add)
		d.firmQuestion(FieldFirmAddress, d.firmAddresses(), 20, add)
		d.firmQuestion(FieldFirmContact, d.ev.Contacts, 20, add)
	}
	if _, ok := d.by[FieldGSTBasis]; !ok {
		add(question{Field: FieldGSTBasis, Family: FieldGSTBasis, Instructions: instructions[FieldGSTBasis], Options: []choice{{ID: "ex", Value: "ex", Describe: "Excluding GST."}, {ID: "inc", Value: "inc", Describe: "Including GST."}}})
	}
	fees, rates, totals := d.candidates()
	if _, ok := d.by[FieldStatedTotal]; !ok && len(totals) > 0 {
		add(question{Field: FieldStatedTotal, Family: FieldStatedTotal, Instructions: instructions[FieldStatedTotal], Options: append(amountOptions(totals), choice{ID: choiceNone, Describe: noneDescribe})})
	}
	for _, it := range d.items {
		field := PricePrefix + it.Key
		switch {
		case it.Price() && len(fees) > 0:
			opts := append(amountOptions(fees),
				choice{ID: choiceNotPrice, Value: choiceNotPrice, Describe: map[string]string{"what": "The proposal gives no fee for this item."}},
				choice{ID: choiceIncluded, Value: choiceIncluded, Describe: map[string]string{"what": "The proposal says this item is included in another fee."}})
			add(question{Field: field, Family: "price", Instructions: priceQuestion(it), Options: opts})
		case it.Kind == rfp.ItemRate && len(rates) > 0:
			opts := append(amountOptions(rates), choice{ID: choiceNotPrice, Value: choiceNotPrice, Describe: map[string]string{"what": "No rate is given for this role."}})
			add(question{Field: field, Family: "rate", Instructions: rateQuestion(it), Options: opts})
		case it.Kind == rfp.ItemCriterion:
			opts := []choice{}
			for _, s := range d.ev.Sections {
				if s.Heading == "" || len(opts) >= 40 {
					continue
				}
				begins := clip(strings.Join(strings.Fields(strings.TrimPrefix(s.Body, s.Heading)), " "), 60)
				opts = append(opts, choice{ID: "s" + strconv.Itoa(s.Ordinal), Value: strconv.Itoa(s.Ordinal), Describe: clip(s.Heading, 80) + " | " + begins, Short: clip(s.Heading, 60)})
			}
			add(question{Field: it.Key, Family: "criterion", Instructions: roleQuestion{
				Question: "Which section of the proposal answers this returnable schedule: " + it.Label + " (" + clip(it.Detail, 200) + ")?",
				NotFor:   "A section that only mentions it in passing, a table of contents or a cover page. Choose none when no section answers it."}, Options: append(opts, choice{ID: choiceNone, Describe: noneDescribe})})
		}
	}
}

func (d *Draft) firmQuestion(field string, values []string, max int, add func(question)) {
	if len(values) < 2 {
		return
	}
	opts := []choice{}
	for i, v := range values {
		if i >= max {
			break
		}
		opts = append(opts, choice{ID: "c" + strconv.Itoa(i+1), Value: v, Describe: map[string]string{"value": v}})
	}
	add(question{Field: field, Family: "firm", Instructions: instructions[field], Options: append(opts, choice{ID: choiceNone, Describe: noneDescribe})})
}

// candidates are the amounts offered for fee rows, rate rows and the total.
// A fee row never sees an hourly rate, and the other way round.
func (d *Draft) candidates() (fees, rates, totals []Amount) {
	for _, a := range d.ev.Amounts {
		if a.Unit || a.GSTPart || a.IncTwin {
			continue
		}
		switch {
		case a.PerHour:
			rates = append(rates, a)
		default:
			fees = append(fees, a)
			if a.Total && a.Kind == "" {
				totals = append(totals, a)
			}
		}
	}
	return limit(distinct(fees), 120), limit(distinct(rates), 80), limit(distinct(totals), 30)
}

// distinct keeps the first amount of each figure: any copy of a figure
// gives the same price, and a proposal that prints its rates twice should
// not crowd the request.
func distinct(as []Amount) []Amount {
	seen := map[int64]bool{}
	var out []Amount
	for _, a := range as {
		key := a.Cents
		if a.Kind == "percent" {
			key = -int64(a.PercentBP) - 1
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, a)
	}
	return out
}

func limit(as []Amount, n int) []Amount {
	if len(as) > n {
		return as[:n]
	}
	return as
}

// amountOptions name each amount by its id and figure; the words around it
// are in the state once, not repeated in every question.
func amountOptions(as []Amount) []choice {
	out := make([]choice, len(as))
	for i, a := range as {
		full := map[string]string{"amount": a.Display, "printed_with": clip(a.Context, 120)}
		if a.Label != "" && !strings.HasPrefix(a.Context, a.Label) {
			full["row_label"] = a.Label
		}
		if a.Column != "" {
			full["column"] = a.Column
		}
		if a.Under != "" && !strings.Contains(a.Context, a.Under) {
			full["under"] = clip(a.Under, 70)
		}
		if a.Before != "" && len(strings.Fields(a.Context)) <= 3 {
			full["line_above"] = clip(a.Before, 90)
		}
		if a.PerHour {
			full["rate"] = "per hour"
		}
		out[i] = choice{ID: a.ID, Value: a.ID, Describe: full, Short: a.Display + " | " + clip(a.Context, 50)}
	}
	return out
}

func priceQuestion(it rfp.ScheduleItem) roleQuestion {
	what := it.Label
	if it.Detail != "" {
		what += " (" + clip(it.Detail, 220) + ")"
	}
	return roleQuestion{
		Question: "Which of these amounts is the firm's fee for this item of the requested fee schedule: " + what + "?",
		NotFor:   "An hourly rate, a total or subtotal of several items, insurance cover, a project value, or a fee for a different item. Choose not priced when the proposal gives no fee for this item; choose included when it says the item is part of another fee.",
	}
}

func rateQuestion(it rfp.ScheduleItem) roleQuestion {
	role := it.Label
	if it.Detail != "" && it.Detail != "Hourly rate excl. GST" {
		role += " (" + it.Detail + ")"
	}
	return roleQuestion{
		Question: "Which of these hourly rates does the firm charge for this role, or for the closest equivalent role it lists: " + role + "?",
		NotFor:   "A lump-sum fee, a per-meeting, per-day or per-visit rate, a GST amount, or a rate for a clearly different seniority. Choose not priced when the firm lists no role at this level.",
	}
}

type state struct {
	Filename string `json:"filename"`
	Excerpt  string `json:"proposal_text,omitempty"`
}

// Call is the one fan-out for what is still open, or false when nothing is.
func (d *Draft) Call() (jev.Call, bool) {
	if len(d.questions) == 0 {
		return jev.Call{}, false
	}
	build := func(short bool) map[string]jev.Question {
		qs := make(map[string]jev.Question, len(d.questions))
		for f, q := range d.questions {
			criteria := make(map[string]any, len(q.Options))
			for _, o := range q.Options {
				criteria[o.ID] = o.Describe
				if short && o.Short != nil {
					criteria[o.ID] = o.Short
				}
			}
			qs[f] = jev.Question{Type: jev.TypeChoice, Instructions: q.Instructions, Criteria: criteria}
		}
		return qs
	}
	st := state{Filename: d.ev.Filename}
	call := jev.Call{QuestionVersion: QuestionVersion, Priority: jev.PriorityBackground, Deadline: 6 * time.Second}
	// A long proposal keeps every candidate and question: the excerpt gives
	// way first, then the words printed with each option.
	for _, step := range []struct {
		short   bool
		excerpt int
	}{{false, 9000}, {false, 4000}, {true, 4000}, {true, 1500}, {true, 0}} {
		call.Questions = build(step.short)
		st.Excerpt = clip(d.ev.Excerpt, step.excerpt)
		call.State = st
		if fits(call) {
			return call, true
		}
	}
	return call, true
}

// maxRequestBytes holds a proposal call well inside the provider's token
// limit. Fee schedules tokenise densely (figures, currency, short words), so
// the generic four-bytes-per-token preflight is not enough here: a 150 KB
// request was refused as max_tokens_exceeded on 16 October 2026.
const maxRequestBytes = 90_000

func fits(call jev.Call) bool {
	s, err := jev.CheckRequestSize(call)
	return err == nil && s.Bytes <= maxRequestBytes
}

// Apply records Jev's answers. A failed call leaves the open questions grey.
//
//	https://docs.typesafe.ai/confidence
func (d *Draft) Apply(res jev.Result, err error, th Thresholds) (grey bool) {
	for f, q := range d.questions {
		if cur, done := d.by[f]; done && cur.DecidedBy != ByJev {
			continue
		}
		dec := Decision{Field: f, Band: BandBlank, DecidedBy: ByJev, QuestionVersion: QuestionVersion}
		if err != nil {
			dec.Band = BandGrey
			grey = true
			d.by[f] = dec
			continue
		}
		if a, ok := res.Answers[f]; ok && a.Type == jev.TypeChoice {
			for _, o := range q.Options {
				if o.ID != a.Choice {
					continue
				}
				dec.Confidence = a.Confidence
				if band, apply := th.Band(q.Family, a.Confidence, len(q.Options)); apply && o.ID != choiceNone {
					dec.Band, dec.Value = band, o.Value
				}
			}
		}
		d.by[f] = dec
	}
	return grey
}

// Questions are the fields asked, for the eval.
func (d *Draft) Questions() []string {
	out := make([]string, 0, len(d.questions))
	for f := range d.questions {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// OptionValue is what an answer id means for a field, for the eval.
func (d *Draft) OptionValue(field, id string) (string, bool) {
	for _, o := range d.questions[field].Options {
		if o.ID == id {
			return o.Value, true
		}
	}
	return "", false
}

// Price is one cell of the price table: a schedule item, or an item the
// firm offered unasked (Key "").
type Price struct {
	Key       string `json:"item_key"`
	Label     string `json:"label"`
	Cents     *int64 `json:"-"`
	State     string `json:"state"`
	Basis     string `json:"basis,omitempty"`
	Page      int    `json:"page,omitempty"`
	Passage   string `json:"passage,omitempty"`
	Band      string `json:"band"`
	DecidedBy string `json:"decided_by"`
}

// Passage is one section of the proposal as stored.
type Passage struct {
	Ordinal int
	Heading string
	Page    int
	Body    string
}

// Response is a non-price criterion's passages.
type Response struct {
	Key       string
	Passages  []int
	Band      string
	DecidedBy string
}

// Reading is the code result of one proposal read.
type Reading struct {
	Kind        string
	PackageID   string
	Firm        *Firm // directory match, or nil
	FirmFields  map[string]string
	GSTBasis    string
	StatedTotal *int64
	Prices      []Price
	Passages    []Passage
	Responses   []Response
	Flags       []string
	Decisions   []Decision
}

var departureHeading = regexp.MustCompile(`(?i)exclusion|assumption|departure|qualification|not included|limitation|clarification`)

// Finish computes every figure from the decisions: ex GST, percentage fees,
// other items, the sum check and the flags.
func (d *Draft) Finish() Reading {
	r := Reading{Kind: d.by[FieldKind].Value, PackageID: d.by[FieldPackage].Value, Firm: d.firm, GSTBasis: d.by[FieldGSTBasis].Value, FirmFields: map[string]string{}}
	for _, f := range FirmFields {
		if v := d.by[f].Value; v != "" {
			r.FirmFields[f] = v
		}
	}
	if v := d.by[FieldStatedTotal].Value; v != "" {
		if c, ok := d.centsOf(v); ok {
			r.StatedTotal = &c
		}
	}
	byID := map[string]Amount{}
	for _, a := range d.ev.Amounts {
		byID[a.ID] = a
	}
	claimed := map[string]bool{}
	var base int64
	baseKnown := true
	for _, it := range d.items {
		if !it.Price() && it.Kind != rfp.ItemRate {
			continue
		}
		dec, ok := d.by[PricePrefix+it.Key]
		p := Price{Key: it.Key, Label: it.Label, State: StateNotPriced, Band: BandBlank, DecidedBy: ByJev}
		if ok {
			p.Band, p.DecidedBy = dec.Band, dec.DecidedBy
		}
		switch v := dec.Value; {
		case !ok || v == "":
			p.State = "" // unread: the cell shows the band, not a figure
		case v == choiceNotPrice:
			p.State = StateNotPriced
		case v == choiceIncluded:
			p.State = StateIncluded
		case strings.HasPrefix(v, "cents:"):
			c, _ := strconv.ParseInt(strings.TrimPrefix(v, "cents:"), 10, 64)
			p.State, p.Cents, p.Basis = StatePriced, &c, d.gstBasisWords(r.GSTBasis, "")
			if r.GSTBasis == "inc" {
				ex := exGST(c)
				p.Cents = &ex
			}
		default:
			a, found := byID[v]
			if !found {
				p.State = ""
				break
			}
			claimed[a.ID] = true
			p.Page, p.Passage = a.Page, a.Context
			c, basis, known := d.amountExGST(a, r.GSTBasis)
			p.State, p.Basis = StatePriced, basis
			if known {
				p.Cents = &c
			}
		}
		if it.Base && it.Price() {
			switch {
			case p.State == StatePriced && p.Cents != nil:
				base += *p.Cents
			case p.State == StatePriced, p.State == "":
				baseKnown = false
			}
		}
		r.Prices = append(r.Prices, p)
	}
	if r.GSTBasis == "inc" {
		r.Flags = append(r.Flags, FlagGSTInc)
	}
	// Amounts no item claimed, that are not totals, rates or unit prices,
	// are what the firm offered unasked: kept, out of totals until award.
	if d.ev.Form == nil && len(d.items) > 0 {
		seen := map[int64]bool{}
		for _, p := range r.Prices {
			if p.Cents != nil {
				seen[*p.Cents] = true
			}
		}
		if r.StatedTotal != nil {
			seen[*r.StatedTotal] = true
		}
		seen[base] = true
		for _, a := range d.ev.Amounts {
			if claimed[a.ID] || a.PerHour || a.Total || a.Kind != "" || a.Unit || a.GSTPart || a.IncTwin || a.Cents < 10000 || seen[a.Cents] || unitPrice(a.Context) {
				continue
			}
			c, basis, _ := d.amountExGST(a, r.GSTBasis)
			if seen[c] {
				continue
			}
			seen[c], seen[a.Cents] = true, true
			label := strings.TrimSpace(strings.Replace(a.Context, a.Display, "", 1))
			if len(label) < 4 {
				label = a.Under
			}
			r.Prices = append(r.Prices, Price{Label: clip(label, 200), Cents: &c, State: StatePriced, Basis: basis, Page: a.Page, Passage: a.Context, Band: BandAmber, DecidedBy: ByRule})
		}
	}
	if r.StatedTotal != nil && baseKnown && len(d.items) > 0 {
		total := *r.StatedTotal
		if r.GSTBasis == "inc" {
			total = exGST(total)
		}
		if gap := total - base; gap > 100 || gap < -100 {
			r.Flags = append(r.Flags, FlagTotalGap)
		}
	}
	for f := range d.flags {
		r.Flags = append(r.Flags, f)
	}
	sort.Strings(r.Flags)
	for _, s := range d.ev.Sections {
		if strings.TrimSpace(s.Body) == "" {
			continue
		}
		r.Passages = append(r.Passages, Passage{Ordinal: s.Ordinal, Heading: s.Heading, Page: s.Page, Body: clip(s.Body, 20000)})
	}
	for _, it := range d.items {
		if it.Kind != rfp.ItemCriterion {
			continue
		}
		resp := Response{Key: it.Key, Band: BandBlank, DecidedBy: ByJev}
		if dec, ok := d.by[it.Key]; ok {
			resp.Band, resp.DecidedBy = dec.Band, dec.DecidedBy
			if n, err := strconv.Atoi(dec.Value); err == nil {
				resp.Passages = []int{n}
			}
		}
		if it.Key == "criterion:r6" {
			// Departures, exclusions and assumptions are collected by heading.
			for _, s := range d.ev.Sections {
				if departureHeading.MatchString(s.Heading) && !contains(resp.Passages, s.Ordinal) {
					resp.Passages = append(resp.Passages, s.Ordinal)
					if resp.DecidedBy != ByJev || resp.Band == BandBlank {
						resp.Band, resp.DecidedBy = BandGreen, ByRule
					}
				}
			}
		}
		r.Responses = append(r.Responses, resp)
	}
	fields := make([]string, 0, len(d.by))
	for f := range d.by {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		r.Decisions = append(r.Decisions, d.by[f])
	}
	return r
}

func contains(list []int, n int) bool {
	for _, v := range list {
		if v == n {
			return true
		}
	}
	return false
}

var unitWords = regexp.MustCompile(`(?i)\bper\b|/\s?(?:meeting|visit|hour|hr|day|week|month|title|page|sheet|copy|unit|item|lot|m2|sqm)\b|\beach\b|\bea\b|\bprint\b`)

func unitPrice(context string) bool { return unitWords.MatchString(context) }

// amountExGST is an amount as ex GST cents, with the basis in words. A
// percentage fee is computed from the construction budget, or unknown.
func (d *Draft) amountExGST(a Amount, docBasis string) (int64, string, bool) {
	if a.Kind == "percent" {
		pct := strconv.Itoa(a.PercentBP/100) + "." + strconv.Itoa(a.PercentBP%100/10) + strconv.Itoa(a.PercentBP%10)
		pct = strings.TrimSuffix(strings.TrimSuffix(pct, "0"), ".0")
		if d.ctx.ConstructionBudget == nil {
			return 0, pct + "% of the construction cost; the budget is not set", false
		}
		c := (*d.ctx.ConstructionBudget*int64(a.PercentBP) + 5000) / 10000
		return c, pct + "% of the construction budget $" + invoices.DisplayCents(*d.ctx.ConstructionBudget), true
	}
	basis := a.GST
	if basis == "" {
		basis = docBasis
	}
	if basis == "inc" {
		return exGST(a.Cents), "quoted including GST; ÷ 1.1", true
	}
	return a.Cents, "", true
}

func (d *Draft) gstBasisWords(doc, _ string) string {
	if doc == "inc" {
		return "quoted including GST; ÷ 1.1"
	}
	return ""
}

// centsOf reads a stored total decision: an amount id or "cents:<n>" or plain cents.
func (d *Draft) centsOf(v string) (int64, bool) {
	for _, a := range d.ev.Amounts {
		if a.ID == v {
			return a.Cents, true
		}
	}
	c, err := strconv.ParseInt(strings.TrimPrefix(v, "cents:"), 10, 64)
	return c, err == nil
}

// exGST removes 10 % GST from an inclusive amount, to the cent.
func exGST(inc int64) int64 {
	if inc < 0 {
		return -exGST(-inc)
	}
	return (inc*10 + 5) / 11
}

// Outcome is one finished read. NotRead is set when there is no text.
type Outcome struct {
	Reading
	NotRead string
	Grey    bool
	Asked   bool
}

// Not-read reasons.
const (
	NotReadNoText      = "no_text_layer"
	NotReadUnreadable  = "unreadable_text"
	NotReadUnsupported = "unsupported_format"
)

// Read reads one proposal: rules, at most one Jev fan-out, then code.
func Read(ctx context.Context, ask Asker, th Thresholds, filename string, text identity.Text, pc Context, users map[string]Decision) (Outcome, error) {
	if !text.TextLayer {
		return Outcome{NotRead: NotReadNoText, Reading: Reading{Flags: []string{FlagNoText}}}, nil
	}
	ev := Harvest(filename, text)
	if !ev.Readable {
		return Outcome{NotRead: NotReadUnreadable, Reading: Reading{Flags: []string{FlagNoText}}}, nil
	}
	d := NewDraft(ev, pc, users)
	out := Outcome{}
	if call, ok := d.Call(); ok {
		out.Asked = true
		res, err := ask.Ask(ctx, call)
		if err != nil && ctx.Err() != nil {
			return Outcome{}, ctx.Err()
		}
		out.Grey = d.Apply(res, err, th)
	}
	out.Reading = d.Finish()
	return out, nil
}

var nameNoise = regexp.MustCompile(`(?i)\b(pty|ltd|limited|proprietary|the|group|australia|aust|nsw|vic|qld|consulting|consultants|services|and|&)\b`)

// NameKey is a firm name reduced for matching: lower-case letters and digits
// without the company suffixes, so "Mecone Pty Ltd" matches "MECONE".
func NameKey(name string) string {
	s := nameNoise.ReplaceAllString(strings.ToLower(name), " ")
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return clip(b.String(), 200)
}

// Raw is Jev's own answer to an asked field before any threshold: the
// option's value, its confidence, and whether it answered. For the eval.
func (d *Draft) Raw(res jev.Result, field string) (string, *float64, bool) {
	q, ok := d.questions[field]
	if !ok {
		return "", nil, false
	}
	a, ok := res.Answers[field]
	if !ok || a.Type != jev.TypeChoice {
		return "", nil, false
	}
	for _, o := range q.Options {
		if o.ID == a.Choice {
			if o.ID == choiceNone {
				return "", a.Confidence, true
			}
			return o.Value, a.Confidence, true
		}
	}
	return "", nil, false
}

// Family is the threshold family of an asked field, and its option count.
func (d *Draft) Family(field string) (string, int) {
	q := d.questions[field]
	return q.Family, len(q.Options)
}

// Amount is the harvested amount with this id.
func (d *Draft) Amount(id string) (Amount, bool) {
	for _, a := range d.ev.Amounts {
		if a.ID == id {
			return a, true
		}
	}
	return Amount{}, false
}

// Decided is a field's current decision.
func (d *Draft) Decided(field string) (Decision, bool) {
	dec, ok := d.by[field]
	return dec, ok
}
