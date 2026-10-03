package intake

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"sitewise/internal/identity"
)

var sheetCounter = regexp.MustCompile(`(?i)^([0-9]{1,3})\s+OF\s+([0-9]{1,3})$`)

func approvalStamp(s string) bool {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	return strings.Contains(s, "forms part of") && (strings.Contains(s, "this plan") || strings.Contains(s, "this document"))
}

// Match the literal two-column Sheet/Scale, Drawn By/Date, Version/Construction
// control grid. Sheet counters elsewhere (reports, schedules) are not identifiers.
func appendSheetVersionGrid(out []Candidate, text identity.Text) []Candidate {
	for _, head := range text.Runs {
		if strings.ToLower(strings.TrimSpace(head.Text)) != "sheet:" {
			continue
		}
		h := layoutSource(head.Source)
		if h.Height <= 0 {
			continue
		}
		scale, drawn, date, version, construction := -1, -1, -1, -1, -1
		for i, r := range text.Runs {
			s := layoutSource(r.Source)
			if s.Page != h.Page || s.Rotation != h.Rotation {
				continue
			}
			dx, dy := (s.X-h.X)/h.Height, (h.Y-s.Y)/h.Height
			name := strings.ToLower(strings.Trim(strings.TrimSpace(r.Text), ".:"))
			if math.Abs(dx) < .5 {
				if name == "drawn by" && dy > 1.5 && dy < 3 {
					drawn = i
				}
				if name == "version no" && dy > 3.5 && dy < 6 {
					version = i
				}
			}
			if dx > 4 && dx < 10 {
				if name == "scale" && math.Abs(dy) < .3 {
					scale = i
				}
				if name == "date" && dy > 1.5 && dy < 3 {
					date = i
				}
				if name == "construction no" && dy > 3.5 && dy < 6 {
					construction = i
				}
			}
		}
		if scale < 0 || drawn < 0 || date < 0 || version < 0 || construction < 0 {
			continue
		}
		right := layoutSource(text.Runs[scale].Source).X
		if math.Abs(layoutSource(text.Runs[date].Source).X-right) > h.Height*.5 || math.Abs(layoutSource(text.Runs[construction].Source).X-right) > h.Height*.5 {
			continue
		}
		under := func(label identity.Source, end float64, valid func(string) bool) int {
			found := -1
			for i, r := range text.Runs {
				s := layoutSource(r.Source)
				gap := label.Y - (s.Y + s.Height)
				if s.Page == label.Page && s.Rotation == label.Rotation && s.Height > 0 && s.X >= label.X && s.X+s.Width <= end && gap >= -label.Height*.2 && gap < label.Height && valid(strings.TrimSpace(r.Text)) {
					if found >= 0 {
						return -1
					}
					found = i
				}
			}
			return found
		}
		n := under(h, right, func(s string) bool {
			m := sheetCounter.FindStringSubmatch(s)
			if m == nil {
				return false
			}
			a, _ := strconv.Atoi(m[1])
			b, _ := strconv.Atoi(m[2])
			return a > 0 && a <= b
		})
		v := under(layoutSource(text.Runs[version].Source), right, func(s string) bool { _, ok := revisionNormalized(s); return ok })
		if n < 0 || v < 0 {
			continue
		}
		add := func(field, value string, index int) Candidate {
			r := text.Runs[index]
			start := strings.Index(r.Text, value)
			c, _ := newCandidate(field, value, textProv(r.Source, index, start, start+len(value), true))
			return c
		}
		number := add(FieldNumber, sheetCounter.FindStringSubmatch(strings.TrimSpace(text.Runs[n].Text))[1], n)
		number.Provenance.OwnNumber = true
		out = mergeCandidate(out, number)
		rev := add(FieldRevision, strings.TrimSpace(text.Runs[v].Text), v)
		rev.Provenance.OwnRevision = true
		out = mergeCandidate(out, rev)
		dt := under(layoutSource(text.Runs[date].Source), right+h.Height*10, isDate)
		if dt >= 0 {
			c := add(FieldDate, strings.TrimSpace(text.Runs[dt].Text), dt)
			c.Provenance.IssueRevision = rev.Normalized
			out = append(out, c)
		}
		// The uncaptioned title sits immediately above Job Address, bounded on the
		// right by Council. It is separate from the submission status and house type.
		for _, address := range text.Runs {
			if !strings.EqualFold(strings.TrimSpace(address.Text), "JOB ADDRESS:") {
				continue
			}
			a := layoutSource(address.Source)
			if a.Page != h.Page || a.Rotation != h.Rotation || a.X >= h.X || h.X-a.X > h.Height*30 || math.Abs(a.Y-layoutSource(text.Runs[drawn].Source).Y) > h.Height*.5 {
				continue
			}
			for _, council := range text.Runs {
				if !strings.EqualFold(strings.TrimSpace(council.Text), "COUNCIL:") {
					continue
				}
				b := layoutSource(council.Source)
				if b.Page != h.Page || b.Rotation != h.Rotation || b.X <= a.X || b.X >= h.X || math.Abs(b.Y-a.Y) > h.Height*.5 {
					continue
				}
				for i, r := range text.Runs {
					s := layoutSource(r.Source)
					value := strings.TrimSpace(r.Text)
					if s.Page != h.Page || s.Rotation != h.Rotation || s.X < a.X || s.X+s.Width > b.X || s.Y-a.Y < h.Height || s.Y-a.Y > h.Height*2.5 || s.Height < h.Height || !captionTitle(value) {
						continue
					}
					c := add(FieldTitle, value, i)
					c.Provenance.Labeled = false
					c.Provenance.BlockTitle = true
					out = mergeCandidate(out, c)
				}
			}
		}
	}
	return out
}
