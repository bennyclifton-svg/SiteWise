package invoices

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
)

// Reading a progress claim: rules first, then one Jev fan-out for what
// rules leave open, all in one request
// (https://docs.typesafe.ai/patterns/fan-out,
// https://docs.typesafe.ai/concepts/how-to-build-with-system-one). Jev picks
// among options Go harvested and Go copies the value
// (https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook,
// https://docs.typesafe.ai/primitives/choice). Each question names its role
// and what it is not (https://docs.typesafe.ai/primitives/advanced); at most
// 254 options and none (https://docs.typesafe.ai/api). The state is the
// claim's excerpt: a larger state lowers accuracy
// (https://docs.typesafe.ai/model-jaggedness/jev-1.13). Bands come only from
// calibrated claim-1 cut-offs (https://docs.typesafe.ai/confidence,
// https://docs.typesafe.ai/patterns/confidence-routing); with no claim
// answer key yet every pick stays blank, stored with its confidence for
// calibration (owner decision 7).

// ClaimQuestionVersion is the claim question map. The calibrated invoice-1
// questions (data/invoice/thresholds.json) are never banded with it.
const ClaimQuestionVersion = "claim-1"

// ClaimReading is a claim's table as read: the rows, the printed total of
// this claim, and whether its header was found.
type ClaimReading struct {
	Rows         []ClaimRow
	PrintedGross *int64
	TableFound   bool
}

// claimContract says whether this read is a progress claim, and on which
// contract. The PM's drop on a contract sets kind and supplier; otherwise
// the words-plus-ABN rule decides, before any call.
func (d *Draft) claimContract() (ClaimContract, bool) {
	kind := d.by[FieldKind]
	sup := d.by[FieldSupplier]
	if kind.DecidedBy == DecidedByUser {
		if kind.Value != KindProgressClaim {
			return ClaimContract{}, false
		}
		return d.ctx.Contracts[sup.Value], true
	}
	if !claimWords(d.ev.KindWords) || sup.DecidedBy != DecidedByRule || sup.Band != BandGreen {
		return ClaimContract{}, false
	}
	c, ok := d.ctx.Contracts[sup.Value]
	if !ok {
		return ClaimContract{}, false
	}
	d.by[FieldKind] = Decision{Field: FieldKind, Value: KindProgressClaim, Band: BandGreen, DecidedBy: DecidedByRule}
	return c, true
}

// rowQuestion asks which contract line or variation one row charges for.
func rowQuestion(r ClaimRow, c ClaimContract) (question, bool) {
	var opts []choice
	for _, l := range c.Lines {
		opts = append(opts, choice{ID: "l" + strconv.Itoa(len(opts)+1), Value: "line:" + l.ID, Describe: map[string]string{"contract_line": l.Label}})
	}
	for _, v := range c.Variations {
		if v.Status == VariationRejected || v.Status == VariationWithdrawn {
			continue
		}
		opts = append(opts, choice{ID: "v" + strconv.Itoa(len(opts)+1), Value: "variation:" + v.ID, Describe: map[string]string{"variation": v.Number + " " + v.Title}})
	}
	if len(opts) == 0 || len(opts)+1 > jev.MaxChoiceOptions {
		return question{}, false
	}
	opts = append(opts, choice{ID: choiceNone, Describe: noneDescribe})
	return question{Field: rowField(r.Ordinal), Options: opts, Instructions: roleQuestion{
		Question: fmt.Sprintf("Which line of the contract, or which variation, does this progress claim row charge for? The row reads %q, claimed to date %s, this claim %s.", r.Label, shortAmount(r.ToDate), shortAmount(r.ThisClaim)),
		NotFor:   "A different line or variation, the contract total, a subtotal, retention or GST. Choose none when no line or variation fits.",
	}}, true
}

func rowField(ordinal int) string { return "row." + strconv.Itoa(ordinal) }

// claimCall is the claim's one fan-out: the invoice fields rules left open
// (not kind, which the drop or the rule settled, and not the line, which
// each row carries) and one question per unplaced contract row.
func (d *Draft) claimCall(rows []ClaimRow, c ClaimContract) (jev.Call, map[string]question, bool) {
	delete(d.questions, FieldKind)
	delete(d.questions, FieldLine)
	asked := map[string]question{}
	qs := map[string]jev.Question{}
	add := func(id string, q question) {
		criteria := map[string]any{}
		for _, o := range q.Options {
			criteria[o.ID] = o.Describe
		}
		qs[id] = jev.Question{Type: jev.TypeChoice, Instructions: q.Instructions, Criteria: criteria}
		asked[id] = q
	}
	for _, f := range JevFields {
		if q, ok := d.questions[f]; ok {
			add(f, q)
		}
	}
	for _, r := range rows {
		if r.Kind != RowContract || r.LineID != "" || len(qs) >= MaxClaimRows+len(JevFields) {
			continue
		}
		if q, ok := rowQuestion(r, c); ok {
			add(q.Field, q)
		}
	}
	if len(qs) == 0 {
		return jev.Call{}, nil, false
	}
	return jev.Call{State: state{Filename: d.ev.Filename, Text: d.ev.Excerpt}, Questions: qs, QuestionVersion: ClaimQuestionVersion, Priority: jev.PriorityInteractive}, asked, true
}

func readClaim(ctx context.Context, ask Asker, th Thresholds, d *Draft, text identity.Text, c ClaimContract, uploaded time.Time, priority jev.Priority) (Outcome, error) {
	t := readClaimTable(text)
	rows := t.rows
	if t.found {
		mapClaimRows(rows, c)
		rows = withRetention(rows)
	} else {
		rows = ContractRows(c)
	}
	d.claimNumberRule(text)
	out := Outcome{}
	if call, asked, ok := d.claimCall(rows, c); ok {
		if priority != "" {
			call.Priority = priority
		}
		out.Asked = true
		res, err := ask.Ask(ctx, call)
		if err != nil && ctx.Err() != nil {
			return Outcome{}, ctx.Err()
		}
		out.Grey = applyClaim(d, rows, asked, res, err, th)
	}
	out.Reading = d.Finish(uploaded)
	out.Claim = &ClaimReading{Rows: rows, PrintedGross: t.gross, TableFound: t.found}
	return out, nil
}

// applyClaim bands Jev's picks with the claim cut-offs. A failed call marks
// what it asked grey and leaves every rule value.
func applyClaim(d *Draft, rows []ClaimRow, asked map[string]question, res jev.Result, err error, th Thresholds) (grey bool) {
	pick := func(id string, q question) (choice, *float64, string) {
		if err != nil {
			return choice{}, nil, BandGrey
		}
		a, ok := res.Answers[id]
		if !ok || a.Type != jev.TypeChoice {
			return choice{}, nil, BandBlank
		}
		for _, o := range q.Options {
			if o.ID == a.Choice {
				if band, apply := th.BandFor(ClaimQuestionVersion, id, a.Confidence, len(q.Options)); apply && o.ID != choiceNone {
					return o, a.Confidence, band
				}
				return choice{}, a.Confidence, BandBlank
			}
		}
		return choice{}, nil, BandBlank
	}
	for id, q := range asked {
		o, conf, band := pick(id, q)
		grey = grey || band == BandGrey
		if f := q.Field; f != "" && !isRowField(id) {
			if cur, done := d.by[f]; done && cur.DecidedBy != DecidedByJev {
				continue
			}
			d.by[f] = Decision{Field: f, Value: o.Value, Band: band, DecidedBy: DecidedByJev, QuestionVersion: ClaimQuestionVersion, Confidence: conf}
			continue
		}
		for i := range rows {
			if rowField(rows[i].Ordinal) != id {
				continue
			}
			rows[i].MappedBy, rows[i].Band = DecidedByJev, band
			switch v := o.Value; {
			case len(v) > 5 && v[:5] == "line:":
				rows[i].LineID = v[5:]
			case len(v) > 10 && v[:10] == "variation:":
				rows[i].Kind, rows[i].VariationID = RowVariation, v[10:]
			}
		}
	}
	return grey
}

func isRowField(id string) bool { return len(id) > 4 && id[:4] == "row." }

// LoadClaimThresholds reads <dir>/thresholds.json for claim-1. A missing
// file is no calibration: every claim pick stays blank until one is fitted.
func LoadClaimThresholds(dir string) (Thresholds, error) {
	body, err := os.ReadFile(filepath.Join(dir, "thresholds.json"))
	if os.IsNotExist(err) {
		return Thresholds{QuestionVersion: ClaimQuestionVersion}, nil
	}
	if err != nil {
		return Thresholds{}, err
	}
	var t Thresholds
	if err := json.Unmarshal(body, &t); err != nil {
		return Thresholds{}, err
	}
	if t.QuestionVersion != ClaimQuestionVersion {
		return Thresholds{}, fmt.Errorf("claim threshold version %q, want %q", t.QuestionVersion, ClaimQuestionVersion)
	}
	return t, nil
}

var claimNumberLabel = regexp.MustCompile(`(?i)^\s*(?:progress\s+|payment\s+)?claim\s*(?:no\.?|number|#)\s*:?\s*$`)

// claimNumberRule reads the claim's number from a cell labelled "Claim no"
// (the invoice rule looks for "Invoice no"), so Jev is not asked for it.
func (d *Draft) claimNumberRule(text identity.Text) {
	if _, done := d.by[FieldNumber]; done {
		return
	}
	for _, r := range claimGrid(text) {
		for i := 0; i+1 < len(r.cells); i++ {
			if claimNumberLabel.MatchString(r.cells[i].text) {
				if v := strings.TrimSpace(r.cells[i+1].text); v != "" && len(v) <= 30 {
					d.by[FieldNumber] = Decision{Field: FieldNumber, Value: v, Band: BandGreen, DecidedBy: DecidedByRule}
					delete(d.questions, FieldNumber)
					return
				}
			}
		}
	}
}
