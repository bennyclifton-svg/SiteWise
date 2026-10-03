package intake

import (
	"math"
	"regexp"
	"sitewise/internal/identity"
	"strings"
)

var ocrNumberPrefix = regexp.MustCompile(`(?i)^\s*(?:dwg\.?\s*no\.?|drawing\s+(?:no\.?|number)|sheet\s+(?:no\.?|number))\s*:?\s*([a-z]{1,4}-)\s*$`)
var ocrNumberSuffix = regexp.MustCompile(`^\d{2,4}$`)
var ocrInlineRevision = regexp.MustCompile(`(?i)^\s*revision\s*:?\s+([a-z0-9]{1,4})\s*$`)

// OCR often splits a printed identifier at its hyphen and gives a small
// caption a different baseline from its value. Bind literal fragments using
// their boxes, never a project name, absolute position or inferred character.
func appendOCRIdentity(out []Candidate, text identity.Text) []Candidate {
	if !text.OCR {
		return out
	}
	for i, r := range text.Runs {
		m := ocrNumberPrefix.FindStringSubmatchIndex(r.Text)
		if m == nil {
			continue
		}
		h := layoutSource(r.Source)
		matches := []int{}
		for j, v := range text.Runs {
			s := layoutSource(v.Source)
			if j != i && ocrNumberSuffix.MatchString(strings.TrimSpace(v.Text)) && ocrAdjacent(h, s, 2) {
				matches = append(matches, j)
			}
		}
		if len(matches) != 1 {
			continue
		}
		j := matches[0]
		suffix := strings.TrimSpace(text.Runs[j].Text)
		prov := textProv(r.Source, i, m[2], m[3], true)
		prov.JoinedRuns = []int{i, j}
		// A nearby, explicit current Revision cell completes the identity block.
		for k, v := range text.Runs {
			rev := ocrInlineRevision.FindStringSubmatchIndex(v.Text)
			if rev == nil || !ocrAdjacent(layoutSource(text.Runs[j].Source), layoutSource(v.Source), 12) {
				continue
			}
			prov.OwnNumber = true
			p := textProv(v.Source, k, rev[2], rev[3], true)
			p.OwnRevision = true
			if c, ok := newCandidate(FieldRevision, v.Text[rev[2]:rev[3]], p); ok {
				out = mergeCandidate(out, c)
			}
		}
		if c, ok := newCandidate(FieldNumber, r.Text[m[2]:m[3]]+suffix, prov); ok {
			out = mergeCandidate(out, c)
		}
	}
	// Upgrade only the candidate already bound by the spatial harvester. A
	// competing equally near value remains unresolved by that harvester.
	for i, c := range out {
		if c.Field != FieldTitle || !c.Provenance.Labeled || c.Provenance.Origin != OriginText || c.Provenance.Run < 0 {
			continue
		}
		s := layoutSource(text.Runs[c.Provenance.Run].Source)
		for _, r := range text.Runs {
			if ownTitleCaption(r.Text) && ocrAdjacent(layoutSource(r.Source), s, 3) {
				out[i].Provenance.OwnTitle = true
			}
		}
	}
	return out
}

func ocrAdjacent(a, b identity.Source, gapHeights float64) bool {
	if a.Height <= 0 || b.Height <= 0 || a.Width <= 0 || b.Width <= 0 || a.Page != b.Page || a.Rotation != b.Rotation {
		return false
	}
	gap := b.X - a.X - a.Width
	return gap >= 0 && gap < a.Height*gapHeights && math.Abs((a.Y+a.Height/2)-(b.Y+b.Height/2)) < math.Min(a.Height, b.Height)*.5
}
