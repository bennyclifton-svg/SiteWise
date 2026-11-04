package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"sitewise/internal/invoices"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// Appointment rows are a view of source fees and schedule scope, not copied
// prices. Selection decisions survive re-reading without changing the source.
type AppointmentRow struct {
	ID               string   `json:"id"`
	Label            string   `json:"label"`
	Amount           *string  `json:"amount"`
	State            string   `json:"state"`
	Included         bool     `json:"included"`
	QuoteID          string   `json:"quote_id,omitempty"`
	ItemKey          string   `json:"item_key,omitempty"`
	Keys             []string `json:"keys"`
	SourceLabel      string   `json:"source_label,omitempty"`
	Source           string   `json:"source,omitempty"`
	Page             int      `json:"page,omitempty"`
	MappingSuggested bool     `json:"mapping_suggested,omitempty"`
	Alternative      bool     `json:"alternative,omitempty"`
	// Printed is the source figure when its ex-GST amount is unknown.
	Printed string `json:"printed,omitempty"`
}

// Appointment bases say how the included source fees relate to the letter.
const (
	BasisReconciled = "reconciled" // they add to the printed total
	BasisListed     = "listed"     // no printed total; the listed fees are summed
)

type AppointmentChoice struct {
	Included bool     `json:"included"`
	Keys     []string `json:"keys"`
	// Placed is true once the PM moved the row above or below the line.
	// Saving only a stage mapping leaves the row in its default place.
	// Nil on choices saved before the distinction existed.
	Placed *bool `json:"placed,omitempty"`
}

const appointmentPrefix = "appointment-row:"

func appointmentRows(items []rfp.ScheduleItem, col FirmColumn) ([]AppointmentRow, *int64, string) {
	if col.Quote == nil {
		return nil, nil, ""
	}
	rows := []AppointmentRow{}
	labels := map[string]string{}
	for _, it := range items {
		if it.Price() && it.Key != "comparison:appointment" {
			labels[it.Key] = it.Label
		}
	}
	// offerLine rows are above the line by default. An older saved choice
	// that kept one below only repeated the old default, unless it moved
	// the row above.
	chosen := func(row *AppointmentRow, offerLine bool) {
		if d, ok := col.Decisions[appointmentPrefix+row.ID]; ok && d.DecidedBy == proposals.ByUser {
			var c AppointmentChoice
			if json.Unmarshal([]byte(d.Value), &c) == nil {
				if c.Placed != nil && *c.Placed || c.Placed == nil && (c.Included || !offerLine) {
					row.Included = c.Included
				}
				if c.Keys != nil {
					row.Keys = c.Keys
					row.MappingSuggested = false
				}
			}
		}
	}
	var stated *int64
	if col.Submission.GSTBasis == "ex" {
		stated = col.Submission.StatedTotal
	}
	offer := proposals.ReadOffer(col.Quote.Fees, stated)
	confirmed := col.Cells["comparison:appointment"]
	var base *int64
	if confirmed.DecidedBy == proposals.ByUser {
		if confirmed.State == proposals.StatePriced {
			base = confirmed.Cents
		}
	} else if col.Submission.GSTBasis == "ex" {
		base = col.Submission.StatedTotal
	}
	// One printed total with no breakdown is the fee, once the letter's tax
	// basis is known.
	if base == nil && confirmed.DecidedBy != proposals.ByUser && offer.Single && (col.Submission.GSTBasis == "ex" || col.Submission.GSTBasis == "inc") {
		for _, f := range col.Quote.Fees {
			if f.ID == offer.TotalID && f.Amount != nil {
				if c, ok, err := invoices.ParseCents(*f.Amount); ok && err == nil {
					base = &c
				}
			}
		}
	}
	var stages []proposals.QuotedFee
	if base != nil {
		stages = awardStages(col.Quote, *base)
	}
	// The firm's own fee lines go above the line by default. A PM's
	// confirmed fee wins unless the lines add to it; a PM's choice on the
	// single professional-fee row keeps that row. Lines whose ex-GST amounts
	// are unknown cannot make a total, so the editable professional-fee row
	// stays for the PM to confirm, with the lines shown below it.
	var lines []proposals.OfferLine
	if sum, ok := offerSum(offer.Lines); ok && len(stages) == 0 && len(offer.Lines) > 0 {
		if _, kept := col.Decisions[appointmentPrefix+"fee"]; !kept && (confirmed.DecidedBy != proposals.ByUser || base != nil && sum == *base) {
			lines = offer.Lines
		}
	}
	labelKeys := labelMappings(items, col.Quote.Fees, offer)
	claimed := map[string]bool{}
	for _, key := range labelKeys {
		claimed[key] = true
	}
	used := map[string]bool{}
	covered := map[string]bool{}
	addFee := func(f proposals.QuotedFee, label string, included, offerLine bool) {
		row := AppointmentRow{ID: "quote:" + f.ID, Label: label, Amount: f.Amount, State: proposals.StatePriced, Included: included, QuoteID: f.ID, Keys: []string{}, SourceLabel: f.Label, Source: f.Source, Page: f.Page, Alternative: strings.Contains(f.Role, "Alternative") || f.Role == proposals.RoleTotal}
		if f.Amount == nil {
			row.Printed = f.Printed
		}
		// A stage named in the fee's own label maps first; then Jev's coverage
		// reading, full before partial. Equal amounts never establish scope.
		if key, ok := labelKeys[f.ID]; ok {
			row.Keys = append(row.Keys, key)
			row.MappingSuggested = true
		} else {
			for _, partial := range []bool{false, true} {
				for _, it := range items {
					if !it.Price() || it.Key == "comparison:appointment" || claimed[it.Key] {
						continue
					}
					cov := col.Quote.Coverage[it.Key]
					cell := col.Cells[it.Key]
					if cov.FeeID == f.ID && strings.HasPrefix(cov.Status, "Partial") == partial || !partial && cell.Label != "" && strings.EqualFold(strings.TrimSpace(cell.Label), strings.TrimSpace(f.Label)) {
						row.Keys = append(row.Keys, it.Key)
						row.MappingSuggested = true
					}
				}
				if len(row.Keys) > 0 {
					break
				}
			}
		}
		chosen(&row, offerLine)
		names := []string{}
		liveKeys := []string{}
		for _, key := range row.Keys {
			if label, ok := labels[key]; ok && !covered[key] {
				names = append(names, label)
				liveKeys = append(liveKeys, key)
				covered[key] = true
			}
		}
		row.Keys = liveKeys
		// The firm's own wording names the line (Walsh, 2026-10-27: the
		// mapped stage name hid "Construction documentation" behind
		// "Tender documents"); the stage stays the line's mapping.
		if len(names) == 1 && strings.TrimSpace(row.Label) == "" {
			row.Label = names[0]
		}
		if row.Amount == nil && row.State != proposals.StateIncluded {
			row.State = proposals.StateNotPriced
		}
		rows = append(rows, row)
		used[f.ID] = true
	}
	basis := ""
	switch {
	case len(stages) > 0:
		for _, f := range stages {
			addFee(f, f.Label, true, false)
		}
		basis = BasisReconciled
	case len(lines) > 0:
		for _, line := range lines {
			f := line.Fee
			f.Amount = line.Amount
			addFee(f, line.Label, true, true)
		}
		used[offer.TotalID] = true
		basis = BasisListed
		if offer.Reconciled {
			basis = BasisReconciled
		}
	default:
		// A non-reconciled breakdown stays inside the single appointment. It is
		// never added to the package total or silently split between RFP stages.
		row := AppointmentRow{ID: "fee", Label: "Professional fee", State: proposals.StateNotPriced, Included: true, ItemKey: "comparison:appointment", Keys: []string{}}
		if c := col.Cells["comparison:appointment"]; c.DecidedBy == proposals.ByUser && c.State != proposals.StatePriced {
			row.State = c.State
			row.Included = c.State != proposals.StateExcluded
		}
		if base != nil {
			text := invoices.FormatCents(*base)
			row.Amount = &text
			row.State = proposals.StatePriced
		}
		for _, f := range col.Quote.Fees {
			if f.Role == proposals.RoleTotal {
				row.Source = f.Source
				row.Page = f.Page
				for _, it := range items {
					cov := col.Quote.Coverage[it.Key]
					if labels[it.Key] != "" && cov.FeeID == f.ID && !strings.HasPrefix(cov.Status, "Partial") {
						row.Keys = append(row.Keys, it.Key)
						row.MappingSuggested = true
					}
				}
				break
			}
		}
		chosen(&row, false)
		live := []string{}
		for _, key := range row.Keys {
			if labels[key] != "" {
				covered[key] = true
				live = append(live, key)
			}
		}
		row.Keys = live
		rows = append(rows, row)
		if offer.Single && base != nil {
			basis = BasisReconciled
		}
	}
	for _, f := range col.Quote.Fees {
		if used[f.ID] || offer.Hidden[f.ID] != "" || !quoteID.MatchString(f.ID) || strings.HasPrefix(f.Role, "Repeated") || f.Role == proposals.RoleComponent && len(lines) == 0 || strings.HasPrefix(f.Suggestion, "Earlier") {
			continue
		}
		if f.Role == proposals.RoleTotal && f.Amount != nil && base != nil {
			c, ok, _ := invoices.ParseCents(*f.Amount)
			if ok && c == *base {
				continue
			}
		}
		addFee(f, proposals.CleanFeeLabel(f), false, false)
	}
	for _, it := range items {
		if !it.Price() || it.Key == "comparison:appointment" || covered[it.Key] {
			continue
		}
		c := col.Cells[it.Key]
		row := AppointmentRow{ID: "item:" + it.Key, Label: it.Label, ItemKey: it.Key, Keys: []string{it.Key}, State: c.State, Source: c.Passage, Page: c.Page}
		if row.State == "" {
			row.State = proposals.StateNotPriced
		}
		// Unmapped machine schedule prices may duplicate a source fee. Keep
		// them unresolved until the user maps or confirms them.
		if c.DecidedBy == proposals.ByUser {
			row.Amount = centsTextPointer(c.Cents)
			row.Included = it.Base && c.State != proposals.StateExcluded
		} else {
			row.State = proposals.StateNotPriced
		}
		if strings.HasPrefix(it.Key, "comparison:") {
			row.Included = it.Base
		}
		chosen(&row, false)
		rows = append(rows, row)
	}
	var total int64
	known, any := true, false
	for _, row := range rows {
		if !row.Included {
			continue
		}
		any = true
		if row.State == proposals.StateIncluded {
			continue
		}
		if row.Amount == nil {
			known = false
			continue
		}
		c, ok, err := invoices.ParseCents(*row.Amount)
		if err != nil || !ok || c < 0 || total > math.MaxInt64-c {
			known = false
			continue
		}
		total += c
	}
	if !known || !any {
		return rows, nil, basis
	}
	return rows, &total, basis
}

func offerSum(lines []proposals.OfferLine) (int64, bool) {
	var sum int64
	for _, l := range lines {
		if l.Amount == nil {
			return 0, false
		}
		c, ok, err := invoices.ParseCents(*l.Amount)
		if err != nil || !ok || c < 0 || sum > math.MaxInt64-c {
			return 0, false
		}
		sum += c
	}
	return sum, true
}

// labelMappings proposes the stage a fee's own label names: every word of
// the stage name appears in the fee label ("Lodgement" in "Task 4 -
// Collation and Lodgement"). Each fee and each stage is used once, longer
// stage names first. It is a suggestion the PM can change.
func labelMappings(items []rfp.ScheduleItem, fees []proposals.QuotedFee, offer proposals.Offer) map[string]string {
	type pair struct {
		fee, key string
		score    int
		order    int
	}
	var pairs []pair
	for n, f := range fees {
		if offer.Hidden[f.ID] != "" || f.ID == offer.TotalID {
			continue
		}
		words := labelWords(proposals.CleanFeeLabel(f) + " " + f.Label)
		for _, it := range items {
			if !it.Price() || strings.HasPrefix(it.Key, "comparison:") {
				continue
			}
			need := labelWords(it.Label)
			all := len(need) > 0
			for w := range need {
				all = all && words[w]
			}
			if all {
				pairs = append(pairs, pair{f.ID, it.Key, len(need), n})
			}
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].score != pairs[j].score {
			return pairs[i].score > pairs[j].score
		}
		return pairs[i].order < pairs[j].order
	})
	out := map[string]string{}
	taken := map[string]bool{}
	for _, p := range pairs {
		if _, done := out[p.fee]; done || taken[p.key] {
			continue
		}
		out[p.fee], taken[p.key] = p.key, true
	}
	return out
}

var labelStop = map[string]bool{"and": true, "the": true, "for": true, "with": true, "incl": true, "including": true, "stage": true, "phase": true, "task": true, "services": true, "service": true, "fee": true, "fees": true}
var labelSplit = regexp.MustCompile(`[^a-z0-9]+`)
var labelHyphen = regexp.MustCompile(`([a-z])-([a-z])`)

// labelSame are stage names firms print differently for the same stage
// (Walsh letter, 2026-10-27: "Construction documentation" for the RFP's
// "For-construction documents", "Construction administration" for
// "Contract administration"). Each becomes one token before matching.
var labelSame = []struct {
	re    *regexp.Regexp
	token string
}{
	{regexp.MustCompile(`for[- ]construction documents?|construction documentation|construction documents?`), " constructiondocs "},
	{regexp.MustCompile(`(?:construction|contract) administration`), " contractadmin "},
}

func labelWords(s string) map[string]bool {
	s = strings.ToLower(s)
	for _, same := range labelSame {
		s = same.re.ReplaceAllString(s, same.token)
	}
	s = labelHyphen.ReplaceAllString(s, "$1$2")
	out := map[string]bool{}
	for _, w := range labelSplit.Split(s, -1) {
		if len(w) < 3 || labelStop[w] || strings.Trim(w, "0123456789") == "" {
			continue
		}
		if len(w) > 4 {
			w = strings.TrimSuffix(w, "s")
		}
		out[w] = true
	}
	return out
}

func centsTextPointer(c *int64) *string {
	if c == nil {
		return nil
	}
	s := invoices.FormatCents(*c)
	return &s
}

type AppointmentRowInput struct {
	Version int64  `json:"version"`
	ID      string `json:"id"`
	AppointmentChoice
}

func (s *Store) SetAppointmentRow(ctx context.Context, org, project, id string, in AppointmentRowInput) error {
	if len(in.ID) > 160 || len(in.Keys) > 30 {
		return ErrInvalidInput
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	sub, err := readSubmission(ctx, tx, org, id)
	if err != nil {
		return err
	}
	if sub.ProjectID != project {
		return ErrNotFound
	}
	if err = lockComparison(ctx, tx, org, project, sub.PackageID); err != nil {
		return err
	}
	sub, err = lockSubmission(ctx, tx, org, project, id)
	if err != nil {
		return err
	}
	if sub.Version != in.Version || sub.Status != SubmissionReceived && sub.Status != SubmissionNeedsReview {
		return ErrConflict
	}
	table, err := s.priceTable(ctx, tx, org, project, sub.PackageID)
	if err != nil {
		return err
	}
	var found *AppointmentRow
	var current []AppointmentRow
	for _, col := range table.Columns {
		if col.Submission.ID == id {
			current = col.AppointmentRows
		}
	}
	for _, row := range current {
		if row.ID == in.ID {
			copy := row
			found = &copy
		}
	}
	if found == nil {
		return ErrNotFound
	}
	// An arrow sends no keys or changes the place; a mapping save sends
	// keys with the row's current place. A place once chosen is kept; an
	// older choice counts as one where the projection still honours it.
	placed := in.Keys == nil || in.Included != found.Included
	for _, col := range table.Columns {
		if d, ok := col.Decisions[appointmentPrefix+in.ID]; col.Submission.ID == id && ok && d.DecidedBy == proposals.ByUser {
			var prev AppointmentChoice
			if json.Unmarshal([]byte(d.Value), &prev) == nil && (prev.Placed != nil && *prev.Placed || prev.Placed == nil && prev.Included == found.Included) {
				placed = true
			}
		}
	}
	in.Placed = &placed
	if in.Keys == nil {
		in.Keys = found.Keys
	}
	valid, seen := map[string]bool{}, map[string]bool{}
	for _, it := range table.Items {
		valid[it.Key] = it.Price() && !it.Retired && it.Key != "comparison:appointment"
	}
	for _, key := range in.Keys {
		if !valid[key] || seen[key] {
			return ErrInvalidInput
		}
		seen[key] = true
	}
	if found.QuoteID == "" && found.ID != "fee" && len(in.Keys) > 0 && (len(in.Keys) != 1 || in.Keys[0] != found.ItemKey) {
		return ErrInvalidInput
	}
	for _, row := range current {
		if row.ID == in.ID {
			continue
		}
		if row.QuoteID != "" || row.ID == "fee" {
			for _, key := range row.Keys {
				if seen[key] {
					return fmt.Errorf("%w: this stage is already covered by another fee; remove that mapping first", ErrConflict)
				}
			}
		}
		if in.Included && row.Included && row.Amount != nil && (found.Alternative || row.Alternative) {
			return fmt.Errorf("%w: set aside the current offer before selecting an alternative", ErrConflict)
		}
	}
	raw, err := json.Marshal(in.AppointmentChoice)
	if err != nil || len(raw) > 2000 {
		return ErrInvalidInput
	}
	if err = upsertSubmissionDecision(ctx, tx, org, project, id, proposals.Decision{Field: appointmentPrefix + in.ID, Value: string(raw)}, true); err != nil {
		return err
	}
	return s.finishSubmissionWrite(ctx, tx, org, project, id)
}
