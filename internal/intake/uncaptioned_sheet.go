package intake

import (
	"math"
	"regexp"
	"strings"

	"sitewise/internal/identity"
)

var shortDecimalSheet = regexp.MustCompile(`^[A-Z]{1,3}\s?[0-9]{2,3}\.[0-9]{1,2}$`)

// Some drawing blocks omit number/title captions. Offer literal values only
// inside the column bounded by Project Number, Print Date, Drawn By and Scale.
// These remain unlabelled candidates, not evidence of an explicit own-field caption.
func appendUncaptionedSheetCell(out []Candidate, text identity.Text) []Candidate {
	for _, anchor := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(anchor.Text), "Project Number") {
			continue
		}
		a := layoutSource(anchor.Source)
		if a.Height <= 0 {
			continue
		}
		printDate, drawn, scale := false, false, false
		for _, r := range text.Runs {
			s := layoutSource(r.Source)
			if s.Page != a.Page || s.Rotation != a.Rotation {
				continue
			}
			gap := (a.Y - s.Y) / a.Height
			aligned := math.Abs(s.X-a.X) < a.Height
			name := strings.ToLower(strings.TrimSpace(r.Text))
			printDate = printDate || name == "print date" && aligned && gap > 1 && gap < 4
			drawn = drawn || name == "drawn by" && aligned && gap > 4 && gap < 7
			scale = scale || name == "scale" && s.X-a.X > a.Height*10 && s.X-a.X < a.Height*20 && gap > 4 && gap < 7
		}
		if !printDate || !drawn || !scale {
			continue
		}
		for i, r := range text.Runs {
			s := layoutSource(r.Source)
			value := strings.TrimSpace(r.Text)
			if s.Page != a.Page || s.Rotation != a.Rotation || s.Height < a.Height*1.5 || s.Height > a.Height*3 || s.X-a.X < a.Height*12 || s.X-a.X > a.Height*26 || s.Y > a.Y || a.Y-s.Y > a.Height*3 || !shortDecimalSheet.MatchString(value) {
				continue
			}
			if c, ok := newCandidate(FieldNumber, value, textProv(r.Source, i, 0, len(r.Text), false)); ok {
				out = mergeCandidate(out, c)
			}
			for j, title := range text.Runs {
				t := layoutSource(title.Source)
				if t.Page != a.Page || t.Rotation != a.Rotation || math.Abs(t.X-a.X) > a.Height*2 || t.Y-a.Y < a.Height || t.Y-a.Y > a.Height*5 || t.Height < a.Height*1.5 || t.Height > a.Height*3 || math.Abs(t.X+t.Width-s.X-s.Width) > a.Height*3 || !captionTitle(strings.TrimSpace(title.Text)) {
					continue
				}
				if c, ok := newCandidate(FieldTitle, strings.TrimSpace(title.Text), textProv(title.Source, j, 0, len(title.Text), false)); ok {
					out = mergeCandidate(out, c)
				}
			}
		}
	}
	return out
}
