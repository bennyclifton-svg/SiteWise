package intake

import (
	"math"
	"regexp"
	"strings"

	"sitewise/internal/identity"
)

var unitHeading = regexp.MustCompile(`(?i)^UNITS?(?:\s+[A-Z]?\d+[A-Z]?)?$`)
var unitList = regexp.MustCompile(`(?i)^[A-Z]?\d+[A-Z]?(?:,\s*[A-Z]?\d+[A-Z]?)*,?$`)
var bedroomSubtitle = regexp.MustCompile(`(?i)^\d+\s+BED(?:ROOMS?)?(?:\s*\+\s*STUDY)?$`)

// Marketing plans can have a prominent unit heading instead of a technical
// title block. Preserve its literal wrapped lines as a candidate, never an own
// drawing number or a settled title. The bedroom subtitle bounds this group.
func appendUnitPlanHeadings(out []Candidate, text identity.Text) []Candidate {
	for i, r := range text.Runs {
		h := r.Source
		if h.Page != 1 || h.Rotation != 0 || h.Height <= 0 || !unitHeading.MatchString(strings.TrimSpace(r.Text)) {
			continue
		}
		indices := []int{i}
		parts := []string{strings.TrimSpace(r.Text)}
		previous := h
		for len(indices) < 5 {
			best, distance, tied := -1, math.Inf(1), false
			for j, next := range text.Runs {
				s := next.Source
				gap := previous.Y - (s.Y + s.Height)
				value := strings.TrimSpace(next.Text)
				if s.Page != h.Page || s.Rotation != h.Rotation || s.Height < h.Height*.65 || s.Height > h.Height*1.1 ||
					s.Y >= previous.Y || gap < -h.Height*.4 || gap > h.Height ||
					math.Abs(s.X+s.Width/2-h.X-h.Width/2) > h.Height*.5 ||
					!(unitList.MatchString(value) || bedroomSubtitle.MatchString(value)) {
					continue
				}
				if gap < distance {
					best, distance, tied = j, gap, false
				} else if gap == distance {
					tied = true
				}
			}
			if best < 0 || tied {
				break
			}
			indices = append(indices, best)
			value := strings.TrimSpace(text.Runs[best].Text)
			parts = append(parts, value)
			if bedroomSubtitle.MatchString(value) {
				prov := textProv(h, i, 0, len(r.Text), false)
				prov.Heading, prov.JoinedRuns = true, indices
				if c, ok := newCandidate(FieldTitle, strings.Join(parts, " "), prov); ok {
					out = mergeCandidate(out, c)
				}
				break
			}
			previous = text.Runs[best].Source
		}
	}
	return out
}
