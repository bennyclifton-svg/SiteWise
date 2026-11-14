package tender

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Margins a builder adds at the foot of a price, in per cent.
var marginRates = []float64{5, 7.5, 10, 12, 12.5, 15, 17.5, 20, 25}

// closeEnough: detail within this share of the tender sum is kept as the
// counted level even when it does not add up exactly (design section 7.1).
const closeEnough = 0.05

var (
	dotZero   = regexp.MustCompile(`(\d+)\.0\b`)
	wordToken = regexp.MustCompile(`[a-z0-9]+`)
	stopWords = map[string]bool{"the": true, "and": true, "of": true, "to": true, "for": true, "ex": true, "exc": true, "excl": true,
		"excluding": true, "exclusive": true, "inc": true, "incl": true, "including": true, "inclusive": true, "gst": true, "sub": true,
		"subtotal": true, "total": true, "cost": true, "costs": true, "price": true, "amount": true, "sum": true}
)

// normGroup reduces a heading or a label to the words that name it, so
// "2. PROJECT MANAGEMENT" and "2.0 Project Management" are the same group.
func normGroup(s string) string {
	s = dotZero.ReplaceAllString(strings.ToLower(s), "$1")
	var out []string
	for _, w := range wordToken.FindAllString(s, -1) {
		if !stopWords[w] {
			out = append(out, w)
		}
	}
	return strings.Join(out, " ")
}

func allowance(kind string) bool { return kind == "ps" || kind == "pc" }
func scope(kind string) bool     { return kind == "base" || allowance(kind) }

func (l *Line) flag(code, text string) { l.Flags = append(l.Flags, Flag{code, text}) }

// checkLines applies R1 to R3 and the row checks to what the model returned.
func checkLines(raw RawReading, pages []Page, rows []Row) []Line {
	figures := map[int]map[int64]bool{}
	ocr := map[int]bool{}
	for _, p := range pages {
		figures[p.Number] = pageFigures(p.Text)
		ocr[p.Number] = p.OCR
	}
	known := map[string]bool{}
	for _, r := range rows {
		known[r.Key] = true
	}
	lines := make([]Line, 0, len(raw.Lines))
	for i, r := range raw.Lines {
		l := Line{Ordinal: i, Page: r.P, Label: r.Label, GST: r.G, Kind: r.Kind, Placed: r.Where, Heading: r.Group, Row: r.Row, Note: r.Note, DecidedBy: "model"}
		switch r.St {
		case "priced":
			l.State = StatePriced
		case "included":
			l.State = StateIncluded
		case "excluded":
			l.State = StateExcluded
		default:
			l.State = StateNotPriced
			if l.Note == "" && r.St == "na" {
				l.Note = "N/A as printed"
			}
			if l.Note == "" && r.St == "tbc" {
				l.Note = "To be confirmed"
			}
		}
		if l.GST == "unknown" || l.GST == "" {
			l.GST = raw.GST
		}
		if l.GST == "" {
			l.GST = "unknown"
		}
		if l.State == StatePriced {
			c, ok, rate := parsePrinted(r.Amt)
			page, found := 0, false
			if ok {
				page, found = findFigure(figures, r.P, abs64(c))
			}
			switch {
			case !ok:
				l.flag(FlagAmountNotFound, "No amount could be read for this line. Type it from the document.")
			case !found:
				// R3: the model supplies no amount.
				l.flag(FlagAmountNotFound, "This amount is not in the text of the page. Type it from the document.")
			default:
				l.Page, l.Printed = page, r.Amt
				if l.GST == "inc" {
					c = exGST(c)
				}
				if l.Kind == "saving" && c > 0 {
					c = -c
				}
				l.Cents = &c
				if ocr[page] {
					l.flag(FlagFromScan, "Read from a scanned page. Check the figure.")
				}
				if rate && scope(l.Kind) {
					l.Kind, l.Placed = "rate", "outside"
				}
			}
		}
		if l.Row != "" && !known[l.Row] {
			l.Row = ""
			l.flag(FlagRowUnknown, "The row chosen for this line is not in the schedule.")
		} else if !r.Sure && scope(l.Kind) {
			l.flag(FlagRowUnsure, "Check the row this line is in.")
		}
		lines = append(lines, l)
	}
	return lines
}

// findFigure looks for an amount on the cited page, then on the pages just
// before and after it, nearest first.
func findFigure(figures map[int]map[int64]bool, page int, cents int64) (int, bool) {
	for _, p := range []int{page, page - 1, page + 1, page - 2, page + 2} {
		if figures[p][cents] {
			return p, true
		}
	}
	return 0, false
}

type unit struct {
	cents int64
	group string
}

// agg is a printed subtotal or total and the group it adds up.
type agg struct {
	i       int    // index into the lines
	target  string // normalised group it adds; "" for the whole document
	doc     bool   // a total of the whole document (a tender sum candidate)
	matched bool   // the lines it adds were found and add up to it
}

// Reconcile turns the model's answer into a checked reading (rules R1 to
// R8). stated is the total the existing rules found, ex GST, or nil.
func Reconcile(raw RawReading, pages []Page, rows []Row, stated *int64) Reading {
	r := Reading{Firm: raw.Firm, Basis: raw.Basis, GST: raw.GST, Exclusions: raw.Exclusions, Notes: raw.Notes}
	r.Lines = checkLines(raw, pages, rows)
	L := r.Lines
	leaf := func(i int) bool {
		l := &L[i]
		return l.Cents != nil && l.State == StatePriced && l.Placed == "sum" && scope(l.Kind)
	}
	isAgg := func(i int) bool {
		return L[i].Cents != nil && (L[i].Kind == "subtotal" || L[i].Kind == "total")
	}
	group := func(i int) string { return normGroup(L[i].Heading) }

	// Which groups have priced detail.
	leaves := map[string]int{}
	for i := range L {
		if leaf(i) {
			leaves[group(i)]++
		}
	}
	// R4: walk the lines in order, checking each printed subtotal against the
	// detail before it, and catching a subtotal the model called scope.
	var open []unit
	var aggs []*agg
	collapse := func(idx []int, cents int64, g string) {
		keep := open[:0:0]
		drop := map[int]bool{}
		for _, k := range idx {
			drop[k] = true
		}
		for k, u := range open {
			if !drop[k] {
				keep = append(keep, u)
			}
		}
		open = append(keep, unit{cents, g})
	}
	// trailing finds the shortest run of at least min units at the end of
	// open, all in group g (any group when g is ""), that adds to cents.
	trailing := func(g string, cents int64, min int) []int {
		var sum int64
		var idx []int
		for k := len(open) - 1; k >= 0; k-- {
			if g != "" && open[k].group != g {
				break
			}
			sum += open[k].cents
			idx = append(idx, k)
			if len(idx) >= min && near(sum, cents) {
				return idx
			}
		}
		return nil
	}
	for i := range L {
		c := L[i].Cents
		switch {
		case leaf(i):
			g := group(i)
			looksLikeTotal := strings.Contains(strings.ToLower(L[i].Label), "total")
			if idx := trailing(g, *c, 2); *c != 0 && idx != nil && (len(idx) >= 3 || looksLikeTotal) {
				L[i].Kind, L[i].Placed = "subtotal", "inside"
				leaves[g]--
				collapse(idx, *c, g)
				aggs = append(aggs, &agg{i: i, target: g, matched: true})
				continue
			}
			open = append(open, unit{*c, g})
		case isAgg(i):
			a := &agg{i: i}
			h, lb := group(i), normGroup(L[i].Label)
			switch {
			case h != "" && leaves[h] > 0:
				a.target = h
			case lb != "" && leaves[lb] > 0:
				a.target = lb // a summary line for a group of detail elsewhere
			case lb == "" || (L[i].Kind == "total" && h == ""):
				a.doc = true
			case h != "":
				a.target = h // a part priced only as a total
			default:
				a.target = "label:" + lb
			}
			if a.doc {
				if idx := trailing("", *c, 1); idx != nil {
					a.matched = true
					collapse(idx, *c, "")
				}
			} else if idx := trailing(a.target, *c, 1); idx != nil {
				a.matched = true
				collapse(idx, *c, a.target)
			} else {
				// The detail may sit elsewhere in the document.
				var idx []int
				var sum int64
				for k, u := range open {
					if u.group == a.target {
						idx, sum = append(idx, k), sum+u.cents
					}
				}
				if len(idx) > 0 && near(sum, *c) {
					a.matched = true
					collapse(idx, *c, a.target)
				}
			}
			aggs = append(aggs, a)
		}
	}

	// The top printed figure of each group.
	top := map[string]*agg{}
	for _, a := range aggs {
		if a.doc {
			continue
		}
		// The largest printed figure of the group; on a tie the total wins.
		if t := top[a.target]; t == nil || *L[a.i].Cents > *L[t.i].Cents || (*L[a.i].Cents == *L[t.i].Cents && L[a.i].Kind == "total") {
			top[a.target] = a
		}
	}
	sum := func(set []int) int64 {
		var s int64
		for _, i := range set {
			s += *L[i].Cents
		}
		return s
	}
	// Priced scope lines of any placement, by group: a group that has some is
	// not "priced only as a subtotal", even when they are all breakdowns.
	scopeLines := map[string]int{}
	for i := range L {
		if L[i].Cents != nil && L[i].State == StatePriced && scope(L[i].Kind) {
			scopeLines[group(i)]++
		}
	}
	var detail, level1, level3 []int
	for i := range L {
		if leaf(i) {
			detail = append(detail, i)
			level1 = append(level1, i)
			if top[group(i)] == nil {
				level3 = append(level3, i)
			}
		}
	}
	for g, a := range top {
		// A part priced only as a subtotal. A named subtotal with no heading
		// (a column total of a grid, say) is one only when nothing else is priced.
		part := scopeLines[g] == 0 && (!strings.HasPrefix(g, "label:") || len(detail) == 0)
		switch {
		case leaves[g] > 0:
			level3 = append(level3, a.i)
		case part:
			level1 = append(level1, a.i)
			level3 = append(level3, a.i)
		}
	}
	sort.Ints(level1)
	sort.Ints(level3)

	// R7: the tender sum.
	d := sum(level1)
	tender, how, flagged := tenderSum(L, aggs, d, sum(level3), stated)
	r.TenderSum, r.SumHow = tender, how
	if flagged != "" {
		r.Flags = append(r.Flags, Flag{FlagCheckTenderSum, flagged})
	}
	if tender == nil {
		for _, i := range level1 {
			L[i].Counted = true
		}
		finish(&r)
		return r
	}

	// R6: which level is counted.
	chosen, level := level1, 1
	fits := func(s int64) bool { return near(s, *tender) || marginOf(s, *tender) != 0 }
	// R8's single fixes come before a margin is assumed: a line that equals
	// the gap exactly is stronger evidence than a round percentage.
	repaired := false
	if !near(d, *tender) {
		if repaired = repair(L, level1, *tender); repaired {
			d = extraCounted(L, level1)
			for _, i := range level1 {
				if !uncounted(L, i) {
					d += *L[i].Cents
				}
			}
		}
	}
	switch {
	case fits(d) || repaired:
	case float64(abs64(*tender-d)) <= closeEnough*float64(abs64(*tender)):
	case len(top) > 0 && fits(sum(level3)):
		chosen, level = level3, 3
	case len(detail) > 0 && len(detail) != len(level1) && fits(sum(detail)):
		chosen, level = detail, 2
	}
	for _, i := range chosen {
		if !uncounted(L, i) {
			L[i].Counted = true
		}
	}
	if level == 3 {
		r.Flags = append(r.Flags, Flag{FlagSummaryCounted, "The printed subtotals add up and the detail does not, so the subtotals are counted and the detail is shown beneath them."})
		// A counted subtotal with no row takes the row most of its detail is in.
		for g, a := range top {
			if L[a.i].Row == "" {
				L[a.i].Row = majorityRow(L, g, group, leaf)
			}
		}
	}
	if level == 1 {
		var differs []string
		for g, a := range top {
			if a.matched || leaves[g] == 0 {
				continue
			}
			var s int64
			for _, i := range detail {
				if group(i) == g {
					s += *L[i].Cents
				}
			}
			if diff := *L[a.i].Cents - s; !near(diff, 0) {
				words := "are " + dollars(diff) + " short of"
				if diff < 0 {
					words = "add to " + dollars(-diff) + " more than"
				}
				differs = append(differs, fmt.Sprintf("%s: its lines %s its printed subtotal", L[a.i].Heading, words))
			}
		}
		sort.Strings(differs)
		for _, text := range differs {
			r.Flags = append(r.Flags, Flag{FlagGroupDiffers, text})
		}
	}
	finish(&r)
	return r
}

// uncounted reports whether a repair took this line out of the count.
func uncounted(L []Line, i int) bool {
	for _, f := range L[i].Flags {
		if f.Code == FlagBreakdown || f.Code == FlagRepeat {
			return true
		}
	}
	return false
}

// extraCounted is the sum of lines a repair brought into the count that
// were not in the level.
func extraCounted(L []Line, level []int) int64 {
	in := map[int]bool{}
	for _, i := range level {
		in[i] = true
	}
	var s int64
	for i := range L {
		if L[i].Counted && !in[i] && L[i].Cents != nil {
			s += *L[i].Cents
		}
	}
	return s
}

// marginOf returns the round margin, in per cent, that takes s to total, or 0.
func marginOf(s, total int64) float64 {
	if s <= 0 || total <= s {
		return 0
	}
	for _, rate := range marginRates {
		if near(int64(float64(s)*(1+rate/100)+0.5), total) {
			return rate
		}
	}
	return 0
}

// tenderSum applies R7. It returns the sum, how it was chosen, and words
// for a flag when it needs a look.
func tenderSum(L []Line, aggs []*agg, d, parts int64, stated *int64) (*int64, string, string) {
	type cand struct {
		cents int64
		ownEx bool // the line itself says ex GST
	}
	var totals []cand
	add := func(c int64, ownEx bool) {
		for k := range totals {
			if near(totals[k].cents, c) {
				totals[k].ownEx = totals[k].ownEx || ownEx
				return
			}
		}
		totals = append(totals, cand{c, ownEx})
	}
	// Lines printed ex GST first, so an inc GST copy of the same total folds into it.
	anyDoc := false
	for pass := 0; pass < 2; pass++ {
		for _, a := range aggs {
			l := &L[a.i]
			if ex := l.GST == "ex"; (a.doc || l.Kind == "total") && ex == (pass == 0) {
				add(*l.Cents, ex)
				anyDoc = anyDoc || a.doc
			}
		}
	}
	pick := func(c int64, how string) (*int64, string, string) { return &c, how, "" }
	var all int64
	for _, t := range totals {
		all += t.cents
	}
	cands := append([]cand{}, totals...)
	if len(totals) > 1 && !anyDoc {
		cands = append(cands, cand{all, true})
	}
	ambiguous := ""
	if d > 0 {
		for _, t := range cands {
			if rate := marginOf(d, t.cents); rate != 0 {
				if rate == 10 && !t.ownEx {
					ambiguous = "A printed total is 10% above the lines. Check whether that is GST or margin."
					continue
				}
				return pick(t.cents, "the lines plus "+trimRate(rate)+"% margin printed at the foot")
			}
		}
		for k, t := range cands {
			if near(t.cents, d) {
				how := "the printed total, which the lines add up to"
				if k >= len(totals) {
					how = "no overall total is printed; the parts' totals add up to the lines"
				}
				c := t.cents
				return &c, how, ambiguous
			}
		}
	}
	switch {
	case len(totals) == 1:
		return pick(totals[0].cents, "one printed total")
	case len(totals) > 1:
		for _, t := range totals {
			if near(all-t.cents, t.cents) {
				return pick(t.cents, "the printed total of the parts")
			}
		}
		// Every printed total belongs to a part and none is the whole: add
		// each part's printed figure, and the lines of any part without one.
		if !anyDoc {
			return pick(parts, "no overall total is printed; this is the sum of the parts")
		}
		largest := totals[0].cents
		for _, t := range totals {
			if t.cents > largest {
				largest = t.cents
			}
		}
		c := largest
		return &c, "the largest of several printed totals", "Several totals are printed and none matches the lines. Check the tender sum."
	}
	if stated != nil {
		c, how, _ := pick(*stated, "the total found on the document")
		return c, how, ""
	}
	if d > 0 {
		return &d, "no total is printed; this is the sum of the lines", ""
	}
	return nil, "", ""
}

func trimRate(r float64) string {
	return strings.TrimSuffix(strings.TrimSuffix(strconv.FormatFloat(r, 'f', 1, 64), "0"), ".")
}

// repair applies R8's single fixes to the level and reports whether the
// level then adds up to the tender sum.
func repair(L []Line, level []int, tender int64) bool {
	var s int64
	in := map[int]bool{}
	for _, i := range level {
		s += *L[i].Cents
		in[i] = true
	}
	gap := tender - s
	if gap > 0 {
		// One line the model put outside the price makes it add up.
		var outside []int
		for i := range L {
			if !in[i] && L[i].Cents != nil && L[i].State == StatePriced && L[i].Placed == "outside" && scope(L[i].Kind) {
				outside = append(outside, i)
			}
		}
		for _, i := range outside {
			if near(*L[i].Cents, gap) {
				L[i].Counted = true
				L[i].flag(FlagCountedToMatch, "Counted so the lines add up to the tender sum. Check it belongs in the price.")
				return true
			}
		}
		var allow int64
		var allowances []int
		for _, i := range outside {
			if allowance(L[i].Kind) {
				allow += *L[i].Cents
				allowances = append(allowances, i)
			}
		}
		if len(allowances) > 1 && near(allow, gap) {
			for _, i := range allowances {
				L[i].Counted = true
				L[i].flag(FlagCountedToMatch, "Counted so the lines add up to the tender sum. Check it belongs in the price.")
			}
			return true
		}
		return false
	}
	// One counted line is a breakdown of another.
	pick := -1
	for _, i := range level {
		if near(*L[i].Cents, -gap) && (pick < 0 || allowance(L[i].Kind) || !allowance(L[pick].Kind)) {
			pick = i
		}
	}
	if pick >= 0 {
		L[pick].flag(FlagBreakdown, "Left out of the count: with it the lines add to more than the tender sum.")
		return true
	}
	// The same prices listed twice.
	seen := map[string]bool{}
	var dups []int
	var dupSum int64
	for _, i := range level {
		key := L[i].Row
		if key == "" {
			key = normGroup(L[i].Label)
		}
		key = strconv.FormatInt(*L[i].Cents, 10) + "|" + key
		if seen[key] {
			dups = append(dups, i)
			dupSum += *L[i].Cents
		}
		seen[key] = true
	}
	if len(dups) > 0 && near(dupSum, -gap) {
		for _, i := range dups {
			L[i].flag(FlagRepeat, "Left out of the count: the same price is listed earlier.")
		}
		return true
	}
	return false
}

func majorityRow(L []Line, g string, group func(int) string, leaf func(int) bool) string {
	byRow := map[string]int64{}
	var all int64
	for i := range L {
		if leaf(i) && group(i) == g {
			byRow[L[i].Row] += *L[i].Cents
			all += *L[i].Cents
		}
	}
	for row, c := range byRow {
		if row != "" && c*2 > all {
			return row
		}
	}
	return ""
}

// finish sets the gap, the remainder and the column flags from the counted lines.
func finish(r *Reading) {
	r.Gap, r.Remainder, r.Why = 0, 0, ""
	var counted, onRows int64
	baseCounted, offered, gstUnknown := false, false, false
	for i := range r.Lines {
		l := &r.Lines[i]
		if l.Kind == "option" || l.Kind == "saving" {
			offered = true
		}
		if !l.Counted || l.Cents == nil {
			continue
		}
		counted += *l.Cents
		if l.Row != "" {
			onRows += *l.Cents
		}
		baseCounted = baseCounted || l.Kind == "base" || l.Kind == "subtotal" || l.Kind == "total"
		gstUnknown = gstUnknown || l.GST == "unknown"
	}
	if r.Basis == "estimate" || r.Basis == "cost_plus" || r.Basis == "rates_only" {
		r.Flags = append(r.Flags, Flag{FlagNotFixedPrice, "This is not a fixed price: " + map[string]string{"estimate": "the firm calls it an estimate", "cost_plus": "it is a cost-plus offer", "rates_only": "only rates are given"}[r.Basis] + "."})
	}
	if gstUnknown {
		r.Flags = append(r.Flags, Flag{FlagGSTUnknown, "The document does not say whether its prices include GST. They are treated as excluding GST."})
	}
	if offered {
		r.Flags = append(r.Flags, Flag{FlagOptionsOffered, "The firm offers options, alternatives or savings. They are not in the tender sum."})
	}
	if r.TenderSum == nil {
		return
	}
	if strings.HasPrefix(r.SumHow, "no total is printed") || strings.HasPrefix(r.SumHow, "no overall total") {
		r.Flags = append(r.Flags, Flag{FlagNoTotalPrinted, "The firm prints no single total. The tender sum is added up from what it prints."})
	}
	r.Gap = *r.TenderSum - counted
	r.Remainder = *r.TenderSum - onRows
	if near(r.Gap, 0) && strings.HasPrefix(r.SumHow, "the lines plus") {
		r.SumHow = "the printed total, which the lines add up to"
	}
	switch rate := marginOf(counted, *r.TenderSum); {
	case r.Gap == 0:
	case near(r.Gap, 0):
		r.Why = "rounding"
	case rate != 0:
		r.Why = "margin:" + trimRate(rate)
	case !baseCounted || counted*2 < *r.TenderSum:
		r.Why = "unbroken" // a lump sum the firm did not break down, or broke down only in part
	case r.Gap > 0:
		r.Why = "differs"
		r.Flags = append(r.Flags, Flag{FlagLinesShort, "The lines are " + dollars(r.Gap) + " short of the tender sum."})
	default:
		r.Why = "differs"
		r.Flags = append(r.Flags, Flag{FlagLinesOver, "The lines add to " + dollars(-r.Gap) + " more than the tender sum. Some may be counted twice."})
	}
}
