package intake

import (
	"math"
	"regexp"
	"sitewise/internal/identity"
	"strings"
)

var pagedDrawingNumber = regexp.MustCompile(`^[A-Z]{1,4}[0-9]{3,7}-[0-9]{2,3}$`)
var drawingPageFraction = regexp.MustCompile(`^[1-9][0-9]*/[1-9][0-9]*$`)

// Some manufacturers put Project Title above a Drg No / REV / Page row.
// All three columns must be present before Project Title means sheet identity.
func appendPagedDrawingBlock(out []Candidate, text identity.Text) []Candidate {
	for _, label := range text.Runs {
		if !strings.EqualFold(strings.Trim(strings.TrimSpace(label.Text), ":."), "Drg No") {
			continue
		}
		h := layoutSource(label.Source)
		if h.Height <= 0 {
			continue
		}
		rev, page := -1, -1
		for i, r := range text.Runs {
			s := layoutSource(r.Source)
			if s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.Y-h.Y) > h.Height*.5 || s.X <= h.X+h.Width || s.X-h.X > h.Height*24 {
				continue
			}
			n := strings.ToLower(strings.Trim(strings.TrimSpace(r.Text), ":."))
			if n == "rev" {
				rev = i
			}
			if n == "page" {
				page = i
			}
		}
		if rev < 0 || page < 0 || text.Runs[rev].Source.X >= text.Runs[page].Source.X {
			continue
		}
		rx, px := text.Runs[rev].Source.X, text.Runs[page].Source.X
		number, revision, fraction := -1, -1, -1
		for i, r := range text.Runs {
			s := layoutSource(r.Source)
			gap := h.Y - (s.Y + s.Height)
			if s.Page != h.Page || s.Rotation != h.Rotation || gap < -h.Height*.5 || gap > h.Height*2 {
				continue
			}
			v := strings.TrimSpace(r.Text)
			if s.X >= h.X && s.X+s.Width < rx && pagedDrawingNumber.MatchString(v) {
				number = i
			}
			if s.X >= rx && s.X+s.Width < px && accepts(FieldRevision, v) {
				revision = i
			}
			if s.X >= px && s.X-px < h.Height*8 && drawingPageFraction.MatchString(v) {
				fraction = i
			}
		}
		if number < 0 || revision < 0 || fraction < 0 {
			continue
		}
		for _, item := range []struct {
			field string
			index int
		}{{FieldNumber, number}, {FieldRevision, revision}} {
			r := text.Runs[item.index]
			v := strings.TrimSpace(r.Text)
			// Explicit labelled cells can contain manufacturer codes outside the
			// conservative free-text number regex.
			c := Candidate{Field: item.field, Display: v, Normalized: normalizeNumber(v), Provenance: textProv(r.Source, item.index, strings.Index(r.Text, v), strings.Index(r.Text, v)+len(v), true)}
			c.Provenance.OwnNumber = item.field == FieldNumber
			c.Provenance.OwnRevision = item.field == FieldRevision
			out = mergeCandidate(out, c)
		}
		for _, caption := range text.Runs {
			s := layoutSource(caption.Source)
			if !strings.EqualFold(strings.Trim(strings.TrimSpace(caption.Text), ":."), "Project Title") || s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.X-h.X) > h.Height || s.Y-h.Y < h.Height || s.Y-h.Y > h.Height*5 {
				continue
			}
			for i, r := range text.Runs {
				v := strings.TrimSpace(r.Text)
				a := layoutSource(r.Source)
				if a.Page != h.Page || a.Rotation != h.Rotation || a.X < s.X || a.X+a.Width > px+h.Height*8 || a.Y < h.Y+h.Height || a.Y >= s.Y || s.Y-(a.Y+a.Height) < -h.Height*.5 || !captionTitle(v) {
					continue
				}
				if c, ok := newCandidate(FieldTitle, v, textProv(r.Source, i, strings.Index(r.Text, v), strings.Index(r.Text, v)+len(v), true)); ok {
					c.Provenance.OwnTitle = true
					c.Provenance.BlockTitle = true
					out = mergeCandidate(out, c)
				}
			}
		}
	}
	return out
}
