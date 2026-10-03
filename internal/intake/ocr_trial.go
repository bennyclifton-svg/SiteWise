package intake

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
	"strings"
)

// TrialOCR uses the existing one-fan-out filing policy, with OCR values capped
// at review. It neither changes thresholds nor guesses missing selections.
// https://docs.typesafe.ai/cookbooks/pre_parsed_value_extraction_cookbook
// https://docs.typesafe.ai/patterns/fan-out
func (s *Service) TrialOCR(ctx context.Context, orgID, documentID string, text identity.Text) ([]store.DecisionWrite, error) {
	doc, err := s.store.GetDocument(ctx, orgID, documentID)
	if err != nil {
		return nil, err
	}
	view, err := s.store.DocumentView(ctx, orgID, documentID)
	if err != nil {
		return nil, err
	}
	if view.Status != store.StatusNotFiled || view.Reason != ReasonNoText || len(text.Runs) == 0 || text.PageCount != 1 {
		return nil, errors.New("OCR trial requires a single-page textless, not-filed document")
	}
	rows, err := s.store.DocumentDecisions(ctx, orgID, documentID)
	if err != nil {
		return nil, err
	}
	users := map[string]Decision{}
	versions := map[string]int64{}
	for _, row := range rows {
		versions[row.Field] = row.Version
		if row.DecidedBy == DecidedByUser {
			users[row.Field] = decisionFromStored(row)
		}
	}
	d := NewDraft(s.catalog, doc.Filename, text, ocrTrialCandidates(text), users)
	// A trial never establishes supersession from unreviewed OCR.
	d.Plan(nil, documentID)
	if call, ok := d.Call(); ok {
		call.Priority = jev.PriorityBackground
		result, err := s.jev.Ask(ctx, call)
		if err != nil {
			return nil, err
		}
		d.Apply(result, nil, s.thresholds)
	}
	var writes []store.DecisionWrite
	for _, decision := range d.Decisions() {
		if decision.DecidedBy == DecidedByUser || decision.Field == FieldSupersedes {
			continue
		}
		band := decision.Band
		if band == BandGreen {
			band = BandAmber
		}
		writes = append(writes, store.DecisionWrite{Field: decision.Field, Value: decision.Value, Band: band,
			DecidedBy: decision.DecidedBy, QuestionVersion: store.OCRTrialVersion + "+" + QuestionVersion,
			Confidence: decision.Confidence, ExpectedVersion: versions[decision.Field]})
	}
	return writes, nil
}

var ocrDrawingNumber = regexp.MustCompile(`(?i)^DWG\.?\s*No\.\s*([A-Z]+\s*-\s*\d+)$`)

// Only used by the explicitly scoped A1 Bankstown-template trial. The helper
// splits the known title cell from its caption using TSV word geometry. These
// boundaries identify cells, never expected answers; the literal OCR survives.
func ocrTrialCandidates(text identity.Text) []Candidate {
	out := Harvest("", text) // A filename is not evidence that OCR read a title.
	var titles []Candidate
	for i, run := range text.Runs {
		s := run.Source
		if s.X >= 2115 && s.Y > 130 && s.Y < 155 && captionTitle(run.Text) {
			// OCR sometimes includes a horizontal cell border before the words.
			v := strings.TrimLeft(run.Text, "—~ ")
			if c, ok := newCandidate(FieldTitle, v, textProv(s, i, len(run.Text)-len(v), len(run.Text), true)); ok {
				c.Provenance.OwnTitle = true
				titles = append(titles, c)
			}
		}
		if m := ocrDrawingNumber.FindStringSubmatchIndex(run.Text); m != nil {
			v := run.Text[m[2]:m[3]]
			out = append(out, Candidate{Field: FieldNumber, Display: v, Normalized: normalizeNumber(v), Provenance: textProv(s, i, m[2], m[3], true)})
		}
	}
	if len(titles) > 0 {
		kept := out[:0]
		for _, c := range out {
			if c.Field != FieldTitle {
				kept = append(kept, c)
			}
		}
		out = append(kept, titles...)
	}
	for i := range out {
		c := &out[i]
		if c.Field != FieldDate || c.Provenance.Run < 1 {
			continue
		}
		run := text.Runs[c.Provenance.Run]
		prior := text.Runs[c.Provenance.Run-1]
		// Revision and date must be adjacent cells in the same issue-table row.
		if run.Source.Y < 1500 || prior.Source.X < 2050 || prior.Source.X > 2085 || math.Abs(prior.Source.Y+prior.Source.Height/2-run.Source.Y-run.Source.Height/2) > 4 {
			continue
		}
		v := strings.TrimSpace(prior.Text)
		if len(v) == 1 && lettersASCII(v) {
			c.Provenance.IssueRevision = strings.ToUpper(v)
			c.Provenance.IssueTable = true
		}
	}
	return out
}
