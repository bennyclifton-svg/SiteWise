package intake

import (
	"math"
	"strings"

	"sitewise/internal/identity"
)

func documentControlTitle(label identity.Run, text identity.Text) bool {
	if !strings.EqualFold(strings.TrimSpace(label.Text), "Title:") {
		return false
	}
	l := layoutSource(label.Source)
	if l.Height <= 0 {
		return false
	}
	for _, run := range text.Runs {
		name := strings.ToLower(strings.TrimSpace(run.Text))
		if name != "document control" && name != "document verification history" && name != "document information" {
			continue
		}
		s := layoutSource(run.Source)
		if s.Page == l.Page && s.Rotation == l.Rotation && math.Abs(s.X-l.X) < l.Height*2 && s.Y > l.Y && s.Y-l.Y < l.Height*8 {
			return true
		}
	}
	return false
}

// A short Rev below Date and Project No in a cover control column describes
// the current report. A standalone Rev table heading does not meet this test.
func coverRevisionCaption(run identity.Run, text identity.Text) bool {
	h := run.Source
	if !strings.EqualFold(strings.Trim(strings.TrimSpace(run.Text), ":."), "rev") || h.Page != 1 || h.Height <= 0 {
		return false
	}
	date, project := false, false
	for _, other := range text.Runs {
		s := other.Source
		if s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.X-h.X) > h.Height*.5 || s.Y <= h.Y || s.Y-h.Y > h.Height*8 {
			continue
		}
		date = date || (strings.EqualFold(strings.Trim(strings.TrimSpace(other.Text), ":."), "date") && s.Y-h.Y < h.Height*4)
		project = project || projectReferenceCaption.MatchString(strings.TrimSpace(other.Text))
	}
	return date && project
}

func ownRevisionCaption(v string) bool {
	issueCell := strings.EqualFold(strings.TrimSpace(v), "Issue:")
	v = strings.ToLower(strings.Trim(strings.TrimSpace(v), ":."))
	return v == "revision" || v == "rev issue" || issueCell
}

func besideRevisionCaption(label identity.Source, text identity.Text) bool {
	for _, r := range text.Runs {
		s := layoutSource(r.Source)
		if ownRevisionCaption(r.Text) && s.Page == label.Page && s.Rotation == label.Rotation && s.X > label.X && s.X-label.X < label.Height*20 && math.Abs(s.Y-label.Y) < label.Height*.5 {
			return true
		}
	}
	return false
}

// Number cells may be right-aligned within the column that ends at Revision.
// Do not extend across the neighbouring revision column or into another row.
func rightAlignedIdentity(field string, label, value identity.Source, text identity.Text) bool {
	gap := label.Y - (value.Y + value.Height)
	if gap < -label.Height*.5 || gap > label.Height*2 || value.X < label.X {
		return false
	}
	if field == FieldRevision {
		return value.X-label.X < label.Height*8
	}
	if field == FieldTitle {
		return value.X-label.X < label.Height*28
	}
	if field != FieldNumber {
		return false
	}
	for _, r := range text.Runs {
		s := layoutSource(r.Source)
		if ownRevisionCaption(r.Text) && s.Page == label.Page && s.Rotation == label.Rotation && math.Abs(s.Y-label.Y) < label.Height*.5 && s.X > label.X && s.X-label.X < label.Height*24 && value.X+value.Width < s.X {
			return true
		}
	}
	return false
}

// A wrapped Current / Revision heading belongs to a drawing register, not the
// current sheet's revision cell.
func registerRevisionCaption(label identity.Source, text identity.Text) bool {
	dateColumn, control := false, false
	for _, r := range text.Runs {
		s := layoutSource(r.Source)
		control = control || (strings.EqualFold(strings.TrimSpace(r.Text), "Document Control") && s.Page == label.Page && s.Rotation == label.Rotation && s.Y > label.Y)
		if strings.EqualFold(strings.TrimSpace(r.Text), "Date") && s.Page == label.Page && s.Rotation == label.Rotation && s.X > label.X && s.X-label.X < label.Height*20 && math.Abs(s.Y-label.Y) < label.Height*.5 {
			dateColumn = true
		}
	}

	for _, r := range text.Runs {
		name := strings.ToLower(strings.TrimSpace(r.Text))
		s := layoutSource(r.Source)
		if dateColumn && (name == "prepared by" || name == "checked by" || name == "approved by" || (control && (name == "prepared" || name == "reviewed"))) && s.Page == label.Page && s.Rotation == label.Rotation && s.X > label.X && s.X-label.X < label.Height*60 && math.Abs(s.Y-label.Y) < label.Height*.5 {
			return true
		}
	}

	for _, r := range text.Runs {
		s := layoutSource(r.Source)
		if strings.EqualFold(strings.TrimSpace(r.Text), "Current") && s.Page == label.Page && s.Rotation == label.Rotation && math.Abs(s.X-label.X) < label.Height && s.Y > label.Y && s.Y-label.Y < label.Height*1.5 {
			return true
		}
	}
	return false
}

// Put PDFium's right-angle glyph boxes into a common reading frame. Translation
// is irrelevant for relative cell distances; original provenance is unchanged.
func layoutSource(s identity.Source) identity.Source {
	x, y, w, h := s.X, s.Y, s.Width, s.Height
	switch s.Rotation {
	case 0:
	case 90:
		s.X, s.Y, s.Width, s.Height = -y-h, x, h, w
	case 180:
		s.X, s.Y = -x-w, -y-h
	case 270:
		s.X, s.Y, s.Width, s.Height = y, -x-w, h, w
	default:
		s.Width, s.Height = 0, 0
	}
	return s
}

func joinTitleBelow(c Candidate, text identity.Text, first int) Candidate {
	return joinTitleLines(c, text, first, false)
}

// Report covers may centre each title line independently; drawing cells retain
// their existing left-alignment requirement.
// Some report covers and response letters use body-sized capitals. Offer a
// centered block as a candidate only when several literal capital lines agree
// in alignment and size; this does not settle the title.
func compactCenteredHeading(text identity.Text, first int) (Candidate, bool) {
	run := text.Runs[first]
	v := strings.TrimSpace(run.Text)
	if v != strings.ToUpper(v) || run.Source.Height <= 0 {
		return Candidate{}, false
	}
	start := strings.Index(run.Text, v)
	prov := textProv(run.Source, first, start, start+len(v), false)
	prov.Heading = true
	c, ok := newCandidate(FieldTitle, v, prov)
	if !ok {
		return Candidate{}, false
	}
	anchor, previous := run.Source, run.Source
	indices := []int{first}
	for len(indices) < 4 {
		best, distance, tied := -1, math.Inf(1), false
		for i, next := range text.Runs {
			s, value := next.Source, strings.TrimSpace(next.Text)
			gap := previous.Y - s.Y - s.Height
			if s.Page != anchor.Page || s.Rotation != anchor.Rotation || math.Abs(s.Height-anchor.Height) > anchor.Height*.1 ||
				math.Abs(s.X+s.Width/2-anchor.X-anchor.Width/2) > anchor.Height*.5 || gap < -anchor.Height*.25 || gap > anchor.Height ||
				value != strings.ToUpper(value) || !captionTitle(value) {
				continue
			}
			if gap < distance {
				best, distance, tied = i, gap, false
			} else if gap == distance {
				tied = true
			}
		}
		if best < 0 || tied {
			break
		}
		indices = append(indices, best)
		c.Display += " " + strings.TrimSpace(text.Runs[best].Text)
		previous = text.Runs[best].Source
	}
	c.Provenance.JoinedRuns = indices
	c.Normalized = normalizeTitle(c.Display)
	return c, len(indices) > 1
}

func joinTitleLines(c Candidate, text identity.Text, first int, centered bool) Candidate {
	indices := []int{first}
	anchor := layoutSource(text.Runs[first].Source)
	previous := anchor
	for len(indices) < 4 {
		best, distance, tied := -1, math.Inf(1), false
		for i, r := range text.Runs {
			s := layoutSource(r.Source)
			gap := previous.Y - (s.Y + s.Height)
			v := strings.TrimSpace(r.Text)
			if s.Page != anchor.Page || s.Rotation != anchor.Rotation || s.Height <= 0 ||
				(math.Abs(s.X-anchor.X) > anchor.Height*.5 && (!centered || math.Abs(s.X+s.Width/2-anchor.X-anchor.Width/2) > anchor.Height*.5)) || math.Abs(s.Height-anchor.Height) > anchor.Height*.2 ||
				gap < -anchor.Height*.25 || gap > anchor.Height*.5 ||
				!(captionTitle(v) || (digitsOnly(v) && len(v) <= 2)) {
				continue
			}
			if gap < distance {
				best, distance, tied = i, gap, false
			} else if gap == distance {
				tied = true
			}
		}
		if best < 0 || tied {
			break
		}
		indices = append(indices, best)
		c.Display += " " + strings.TrimSpace(text.Runs[best].Text)
		previous = layoutSource(text.Runs[best].Source)
	}
	if len(indices) > 1 {
		c.Normalized = normalizeTitle(c.Display)
		c.Provenance.JoinedRuns = indices
	}
	return c
}

// Pair literal values by the Issue/Date table columns and row geometry. This
// never ranks dates or assumes that the last row is the current revision.
func appendIssueTableDates(out []Candidate, text identity.Text) []Candidate {
	out = appendCompactIssueDates(out, text)
	out = appendBottomMergedIssueDates(out, text)
	out = appendDrawnCheckDateRows(out, text)
	for _, heading := range text.Runs {
		name := strings.ToLower(strings.Trim(strings.TrimSpace(heading.Text), ":."))
		numberedRevision := name == "revision no"
		if numberedRevision {
			name = "revision"
		}
		bottomHeader := name == "rev by"
		distribution := name == "revision issued to"
		if name != "issue" && name != "rev" && name != "revision" && name != "revision description" && name != "rev description" && !bottomHeader && !distribution {
			continue
		}
		h := layoutSource(heading.Source)
		if h.Height <= 0 {
			continue
		}
		// Some CAD revision tables grow upwards from separate REV / DESCRIPTION /
		// BY / DATE captions. The description column distinguishes this from the
		// current Revision cell below the drawing title.
		separateDescription := false
		for _, caption := range text.Runs {
			s := layoutSource(caption.Source)
			if name == "rev" && strings.EqualFold(strings.Trim(strings.TrimSpace(caption.Text), ":"), "DESCRIPTION") && s.Page == h.Page && s.Rotation == h.Rotation && math.Abs(s.Y-h.Y) < h.Height*.5 && s.X > h.X+h.Width && s.X-h.X < h.Height*20 {
				separateDescription = true
			}
		}
		status, author := false, false
		if name == "issue" && h.Page > 1 {
			for _, caption := range text.Runs {
				s := layoutSource(caption.Source)
				if s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.Y-h.Y) > h.Height*.5 {
					continue
				}
				status = status || (strings.EqualFold(strings.TrimSpace(caption.Text), "Status") && s.X < h.X && h.X-s.X < h.Height*10)
				author = author || (strings.EqualFold(strings.TrimSpace(caption.Text), "Prepared By") && s.X > h.X && s.X-h.X < h.Height*10)
			}
		}
		approvalTable := status && author
		for _, dateHeading := range text.Runs {
			dateLabel := strings.ToLower(strings.Trim(strings.TrimSpace(dateHeading.Text), ":."))
			formattedDate := dateLabel == "date (dd.mm.yy)" || dateLabel == "date (dd.mm.yyyy)" || dateLabel == "date (dd/mm/yy)" || dateLabel == "date (dd/mm/yyyy)"
			if dateLabel != "date" && dateLabel != "initial date" && dateLabel != "date approved by" && !formattedDate {
				continue
			}
			dh := layoutSource(dateHeading.Source)
			maxWidth := h.Height * 20
			if bottomHeader || separateDescription || name == "rev description" || approvalTable {
				maxWidth = h.Height * 40
			}
			if dh.Page != h.Page || dh.Rotation != h.Rotation || math.Abs(dh.Y-h.Y) > h.Height*.5 || dh.X <= h.X+h.Width || dh.X-h.X > maxWidth {
				continue
			}
			boundary := (h.X + h.Width/2 + dh.X + dh.Width/2) / 2
			for revisionIndex, revision := range text.Runs {
				rs := layoutSource(revision.Source)
				rv := strings.TrimSpace(revision.Text)
				if distribution && math.Abs(rs.X-h.X) > h.Height*2 {
					continue // The merged caption also spans the recipient column.
				}
				if approvalTable && math.Abs(rs.X+rs.Width/2-h.X-h.Width/2) > h.Height*2 {
					continue // Author/checker initials are not issue values.
				}
				if name == "rev description" {
					if tokens := strings.Fields(rv); len(tokens) > 1 {
						rv = tokens[0]
					}
				}
				normal, ok := revisionNormalized(rv)
				onDataSide := rs.Y < h.Y
				if bottomHeader {
					onDataSide = rs.Y > h.Y
				}
				if separateDescription && rs.Y > h.Y {
					onDataSide = true
				}
				if !ok || rs.Page != h.Page || rs.Rotation != h.Rotation || rs.Height <= 0 ||
					!onDataSide || math.Abs(h.Y-rs.Y) > h.Height*20 || rs.X < h.X-h.Height*2 || rs.X+rs.Width/2 >= boundary {
					continue
				}
				for i, date := range text.Runs {
					ds := layoutSource(date.Source)
					matches := datePattern.FindAllStringIndex(date.Text, -1)
					if len(matches) != 1 {
						continue
					}
					loc := matches[0]
					v := date.Text[loc[0]:loc[1]]
					// Some PDF producers merge the adjacent initials/date cells.
					prefix := strings.TrimSpace(date.Text[:loc[0]])
					if prefix != "" && (!lettersASCII(prefix) || len(prefix) > 3) {
						continue
					}
					columnDistance := math.Abs((ds.X + ds.Width/2) - (dh.X + dh.Width/2))
					// A format hint widens the caption, not the left-aligned date cell.
					if dateLabel == "date approved by" || formattedDate || numberedRevision {
						columnDistance = math.Abs(ds.X - dh.X)
					}
					if !isDate(v) || ds.Page != h.Page || ds.Rotation != h.Rotation || ds.Height <= 0 ||
						math.Abs(ds.Y-rs.Y) > h.Height*.4 || ds.X+ds.Width/2 <= boundary ||
						columnDistance > h.Height*2 {
						continue
					}
					c, ok := newCandidate(FieldDate, v, textProv(date.Source, i, strings.Index(date.Text, v), strings.Index(date.Text, v)+len(v), true))
					if ok {
						c.Provenance.IssueRevision = normal
						c.Provenance.IssueTable = true
						out = append(out, c)
						if approvalTable || distribution || registerRevisionCaption(h, text) {
							start := strings.Index(revision.Text, rv)
							if candidate, ok := newCandidate(FieldRevision, rv, textProv(revision.Source, revisionIndex, start, start+len(rv), true)); ok {
								candidate.Provenance.IssueTable = true
								out = mergeCandidate(out, candidate)
							}
						}
					}
				}
			}
		}
	}
	return out
}

// PDF text runs can merge the two adjacent Issue and Date column captions.
// Retain the original value runs; pair only tightly aligned rows below them.
func appendCompactIssueDates(out []Candidate, text identity.Text) []Candidate {
	for _, heading := range text.Runs {
		name := strings.TrimSpace(heading.Text)
		mergedRevision := strings.EqualFold(name, "Revision Issue Date")
		if !strings.EqualFold(name, "Issue Date") && !mergedRevision {
			continue
		}
		h := layoutSource(heading.Source)
		if h.Height <= 0 {
			continue
		}
		for _, revision := range text.Runs {
			rs := layoutSource(revision.Source)
			rev, ok := revisionNormalized(strings.TrimSpace(revision.Text))
			columnWidth := h.Height
			if mergedRevision {
				columnWidth *= 2
			}
			if !ok || rs.Page != h.Page || rs.Rotation != h.Rotation || math.Abs(rs.X-h.X) > columnWidth || rs.Y >= h.Y || h.Y-rs.Y > h.Height*20 {
				continue
			}
			for i, date := range text.Runs {
				ds := layoutSource(date.Source)
				v := strings.TrimSpace(date.Text)
				if !isDate(v) || ds.Page != h.Page || ds.Rotation != h.Rotation || math.Abs(ds.Y-rs.Y) > h.Height*.4 || ds.X <= rs.X+rs.Width || ds.X > h.X+h.Width+h.Height*2 {
					continue
				}
				if c, ok := newCandidate(FieldDate, v, textProv(date.Source, i, strings.Index(date.Text, v), strings.Index(date.Text, v)+len(v), true)); ok {
					c.Provenance.IssueRevision = rev
					c.Provenance.IssueTable = true
					out = append(out, c)
				}
			}
		}
	}
	return out
}

// CAD exports may merge the number and revision values into one text run.
// Bind only a complete two-token value spanning both explicitly named columns.
func appendCombinedDrawingCell(out []Candidate, text identity.Text) []Candidate {
	for _, caption := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(caption.Text), "Project Number/Drawing Number") {
			continue
		}
		label := layoutSource(caption.Source)
		if label.Height <= 0 {
			continue
		}
		for _, revision := range text.Runs {
			rs := layoutSource(revision.Source)
			if !ownRevisionCaption(revision.Text) || rs.Page != label.Page || rs.Rotation != label.Rotation || math.Abs(rs.Y-label.Y) > label.Height*.5 || rs.X <= label.X+label.Width || rs.X-label.X > label.Height*24 {
				continue
			}
			for i, run := range text.Runs {
				s := layoutSource(run.Source)
				tokens := strings.Fields(run.Text)
				gap := label.Y - (s.Y + s.Height)
				if len(tokens) != 2 || !isNumber(tokens[0]) || !accepts(FieldRevision, tokens[1]) || s.Page != label.Page || s.Rotation != label.Rotation || s.X < label.X || s.X-label.X > label.Height*2 || gap < -label.Height*.5 || gap > label.Height*2 || s.X+s.Width < rs.X || s.X+s.Width > rs.X+rs.Width+label.Height {
					continue
				}
				for j, field := range []string{FieldNumber, FieldRevision} {
					start := strings.Index(run.Text, tokens[j])
					if j == 1 {
						start = strings.LastIndex(run.Text, tokens[j])
					}
					c, ok := newCandidate(field, tokens[j], textProv(run.Source, i, start, start+len(tokens[j]), true))
					if ok {
						c.Provenance.OwnNumber = field == FieldNumber
						c.Provenance.OwnRevision = field == FieldRevision
						out = mergeCandidate(out, c)
					}
				}
			}
		}
	}
	return out
}

// Some revision tables put their merged REV/DATE caption beneath the rows.
// The first two literal tokens are paired; current revision is chosen elsewhere.
func appendBottomMergedIssueDates(out []Candidate, text identity.Text) []Candidate {
	for _, heading := range text.Runs {
		name := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(heading.Text), ".", ""))
		if name != "rev date" {
			continue
		}
		h := layoutSource(heading.Source)
		if h.Height <= 0 {
			continue
		}
		for i, run := range text.Runs {
			s := layoutSource(run.Source)
			tokens := strings.Fields(run.Text)
			if len(tokens) < 2 || s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.X-h.X) > h.Height || s.Y <= h.Y || s.Y-h.Y > h.Height*20 {
				continue
			}
			rev, ok := revisionNormalized(tokens[0])
			if !ok || !isDate(tokens[1]) {
				continue
			}
			start := strings.Index(run.Text, tokens[1])
			c, ok := newCandidate(FieldDate, tokens[1], textProv(run.Source, i, start, start+len(tokens[1]), true))
			if ok {
				c.Provenance.IssueRevision = rev
				c.Provenance.IssueTable = true
				out = append(out, c)
			}
		}
	}
	return out
}

// A bare Title caption is a sheet title only within the drawing identity row.
// Project Title and report Title captions cannot acquire this provenance.
func spatialOwnTitle(run identity.Run, text identity.Text) bool {
	if ownTitleCaption(run.Text) {
		return true
	}
	if !strings.EqualFold(strings.Trim(strings.TrimSpace(run.Text), ":"), "Title") {
		return false
	}
	h := layoutSource(run.Source)
	if h.Height <= 0 {
		return false
	}
	for _, r := range text.Runs {
		s := layoutSource(r.Source)
		// A TITLE cell immediately above the DRAWING NO / REVISION row is
		// equally explicit; CAD publishers need not use one combined caption.
		if (strings.EqualFold(strings.Trim(strings.TrimSpace(r.Text), ":."), "Drawing No") || strings.EqualFold(strings.Trim(strings.TrimSpace(r.Text), ":."), "Drawing Number")) && s.Page == h.Page && s.Rotation == h.Rotation && s.X >= h.X && s.X-h.X < h.Height*20 && h.Y-s.Y > 0 && h.Y-s.Y < h.Height*8 && besideRevisionCaption(s, text) {
			return true
		}
		if !strings.EqualFold(strings.TrimSpace(r.Text), "Project Number/Drawing Number") {
			continue
		}
		if s.Page == h.Page && s.Rotation == h.Rotation && s.X > h.X && s.X-h.X < h.Height*40 && math.Abs(s.Y-h.Y) < h.Height*.5 && besideRevisionCaption(s, text) {
			return true
		}
	}
	return false
}

// A separately spaced SHEET nn subtitle belongs to the title cell. Do not
// increase the general line-joining distance for arbitrary report headings.
func joinSheetSubtitle(c Candidate, text identity.Text, first int) Candidate {
	if len(c.Provenance.JoinedRuns) > 0 {
		return c
	}
	h := layoutSource(text.Runs[first].Source)
	best := -1
	for i, r := range text.Runs {
		words := strings.Fields(r.Text)
		if len(words) != 2 || !strings.EqualFold(words[0], "SHEET") || !digitsOnly(words[1]) || len(words[1]) > 3 {
			continue
		}
		s := layoutSource(r.Source)
		gap := h.Y - (s.Y + s.Height)
		if s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.X-h.X) > h.Height*.5 || math.Abs(s.Height-h.Height) > h.Height*.2 || gap < 0 || gap > h.Height {
			continue
		}
		if best >= 0 {
			return c
		}
		best = i
	}
	if best >= 0 {
		c.Display += " " + strings.TrimSpace(text.Runs[best].Text)
		c.Normalized = normalizeTitle(c.Display)
		c.Provenance.JoinedRuns = []int{first, best}
	}
	return c
}

// A drawing schedule can provide this sheet's title when its literal number
// matches the independently bound current-number cell. Other rows stay references.
func appendScheduleOwnTitle(out []Candidate, text identity.Text) []Candidate {
	for _, c := range out {
		if c.Field == FieldTitle && c.Provenance.OwnTitle {
			return out // The sheet's own title precedes an abbreviated register entry.
		}
	}
	own := map[string]bool{}
	for _, c := range out {
		if c.Field == FieldNumber && c.Provenance.OwnNumber {
			own[c.Normalized] = true
		}
	}
	if len(own) != 1 {
		return out
	}
	for _, header := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(header.Text), "Drawing Number") {
			continue
		}
		h := layoutSource(header.Source)
		if h.Height <= 0 {
			continue
		}
		for _, description := range text.Runs {
			if !strings.EqualFold(strings.TrimSpace(description.Text), "Description") {
				continue
			}
			d := layoutSource(description.Source)
			if d.Page != h.Page || d.Rotation != h.Rotation || math.Abs(d.Y-h.Y) > h.Height*.5 || d.X < h.X+h.Width || d.X-h.X > h.Height*20 {
				continue
			}
			for _, number := range text.Runs {
				n := layoutSource(number.Source)
				if !own[normalizeNumber(strings.TrimSpace(number.Text))] || n.Page != h.Page || n.Rotation != h.Rotation || math.Abs(n.X-h.X) > h.Height || n.Y >= h.Y || h.Y-n.Y > h.Height*60 {
					continue
				}
				for i, title := range text.Runs {
					t := layoutSource(title.Source)
					v := strings.TrimSpace(title.Text)
					if t.Page != h.Page || t.Rotation != h.Rotation || math.Abs(t.X-d.X) > h.Height || math.Abs(t.Y-n.Y) > n.Height*.3 || !captionTitle(v) {
						continue
					}
					start := strings.Index(title.Text, v)
					c, ok := newCandidate(FieldTitle, v, textProv(title.Source, i, start, start+len(v), true))
					if ok {
						c.Provenance.OwnTitle = true
						out = mergeCandidate(out, c)
					}
				}
			}
		}
	}
	return out
}

// Control tables separate captions from values. A Document Subject repeated
// verbatim on the cover corroborates the descriptive title; project references
// remain project references even when their numeric values occupy separate runs.
func appendReportControl(out []Candidate, text identity.Text) []Candidate {
	projects := map[string]bool{}
	for _, caption := range text.Runs {
		name := strings.ToLower(strings.TrimSpace(caption.Text))
		if name != "document subject" && name != "project number" && name != "job number" {
			continue
		}
		h := layoutSource(caption.Source)
		if h.Height <= 0 {
			continue
		}
		control := false
		for _, heading := range text.Runs {
			s := layoutSource(heading.Source)
			v := strings.ToLower(strings.TrimSpace(heading.Text))
			if (v == "document information" || v == "document control") && s.Page == h.Page && s.Rotation == h.Rotation && s.Y > h.Y && s.Y-h.Y < h.Height*30 {
				control = true
			}
		}
		if !control {
			continue
		}
		for i, r := range text.Runs {
			s := layoutSource(r.Source)
			v := strings.TrimSpace(r.Text)
			if s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.Y-h.Y) > h.Height*.3 || s.X < h.X+h.Width || s.X-h.X-h.Width > h.Height*16 {
				continue
			}
			if name != "document subject" {
				if isNumber(v) {
					projects[normalizeNumber(v)] = true
				}
				continue
			}
			if !captionTitle(v) {
				continue
			}
			for j, cover := range text.Runs {
				if cover.Source.Page != 1 || normalizeTitle(cover.Text) != normalizeTitle(v) {
					continue
				}
				start := strings.Index(r.Text, v)
				c, ok := newCandidate(FieldTitle, v, textProv(r.Source, i, start, start+len(v), true))
				if !ok {
					continue
				}
				c.Provenance.ControlTitle = true
				out = mergeCandidate(out, c)
				c.Provenance = textProv(cover.Source, j, 0, len(cover.Text), false)
				c.Provenance.Heading = true
				out = mergeCandidate(out, c)
			}
		}
	}
	kept := out[:0]
	for _, c := range out {
		if c.Field != FieldNumber || !projects[c.Normalized] || c.Provenance.OwnNumber {
			kept = append(kept, c)
		}
	}
	return kept
}

// Some exports merge the number and title values while preserving their two
// captions. Split only a complete leading number in a run spanning both cells.
func appendMergedNumberTitle(out []Candidate, text identity.Text) []Candidate {
	for _, label := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(label.Text), "Drawing No.") {
			continue
		}
		h := layoutSource(label.Source)
		if h.Height <= 0 {
			continue
		}
		for _, caption := range text.Runs {
			if !strings.EqualFold(strings.TrimSpace(caption.Text), "Drawing Name") {
				continue
			}
			cs := layoutSource(caption.Source)
			if cs.Page != h.Page || cs.Rotation != h.Rotation || math.Abs(cs.Y-h.Y) > h.Height*.5 || cs.X <= h.X+h.Width || cs.X-h.X > h.Height*20 {
				continue
			}
			for i, run := range text.Runs {
				rs := layoutSource(run.Source)
				gap := h.Y - (rs.Y + rs.Height)
				if rs.Page != h.Page || rs.Rotation != h.Rotation || math.Abs(rs.X-h.X) > h.Height || gap < 0 || gap > h.Height*2 || rs.X+rs.Width <= cs.X {
					continue
				}
				value := strings.TrimSpace(run.Text)
				split := strings.IndexByte(value, ' ')
				if split < 0 || !isNumber(value[:split]) || !captionTitle(strings.TrimSpace(value[split:])) {
					continue
				}
				for _, part := range []struct{ field, value string }{{FieldNumber, value[:split]}, {FieldTitle, strings.TrimSpace(value[split:])}} {
					start := strings.Index(run.Text, part.value)
					if c, ok := newCandidate(part.field, part.value, textProv(run.Source, i, start, start+len(part.value), true)); ok {
						c.Provenance.OwnNumber = part.field == FieldNumber
						c.Provenance.OwnTitle = part.field == FieldTitle
						out = mergeCandidate(out, c)
					}
				}
			}
		}
	}
	return out
}

// A bottom Issue / Revision Description / Drawn Check Date table can merge
// initials and date into one run. Pair by row, using only its final date token.
func appendDrawnCheckDateRows(out []Candidate, text identity.Text) []Candidate {
	for _, heading := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(heading.Text), "Drawn Check Date") {
			continue
		}
		h := layoutSource(heading.Source)
		if h.Height <= 0 {
			continue
		}
		for _, issue := range text.Runs {
			if !strings.EqualFold(strings.TrimSpace(issue.Text), "Issue") {
				continue
			}
			is := layoutSource(issue.Source)
			if is.Page != h.Page || is.Rotation != h.Rotation || math.Abs(is.Y-h.Y) > h.Height*.5 || is.X >= h.X || h.X-is.X > h.Height*24 {
				continue
			}
			for _, revision := range text.Runs {
				rs := layoutSource(revision.Source)
				rev, ok := revisionNormalized(strings.TrimSpace(revision.Text))
				if !ok || rs.Page != h.Page || rs.Rotation != h.Rotation || math.Abs(rs.X-is.X) > h.Height || rs.Y <= h.Y || rs.Y-h.Y > h.Height*20 {
					continue
				}
				for i, date := range text.Runs {
					ds := layoutSource(date.Source)
					if ds.Page != h.Page || ds.Rotation != h.Rotation || math.Abs(ds.Y-rs.Y) > h.Height*.4 || math.Abs(ds.X-h.X) > h.Height || math.Abs(ds.X+ds.Width-h.X-h.Width) > h.Height*2 {
						continue
					}
					parts := strings.Fields(date.Text)
					if len(parts) != 3 || !isDate(parts[2]) || !lettersASCII(strings.ReplaceAll(parts[0], "/", "")) || !lettersASCII(parts[1]) || len(parts[0]) > 12 || len(parts[1]) > 3 {
						continue
					}
					start := strings.LastIndex(date.Text, parts[2])
					if c, ok := newCandidate(FieldDate, parts[2], textProv(date.Source, i, start, start+len(parts[2]), true)); ok {
						c.Provenance.IssueTable = true
						c.Provenance.IssueRevision = rev
						out = append(out, c)
					}
				}
			}
		}
	}
	return out
}

// The title column of an explicitly headed drawing schedule describes many
// sheets. It is not the current sheet's own title block.
func drawingScheduleTitle(label identity.Run, text identity.Text) bool {
	if !strings.EqualFold(strings.Trim(strings.TrimSpace(label.Text), ":"), "Drawing Title") {
		return false
	}
	h := layoutSource(label.Source)
	if h.Height <= 0 {
		return false
	}
	for _, r := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(r.Text), "DRAWING SCHEDULE") {
			continue
		}
		s := layoutSource(r.Source)
		if s.Page == h.Page && s.Rotation == h.Rotation && s.X <= h.X && h.X-s.X < h.Height*20 && s.Y > h.Y && s.Y-h.Y < h.Height*20 {
			return true
		}
	}
	return false
}

// A value aligned with an explicit Issue/Revision/Date caption cannot also
// become a title merely because a document reference precedes it in the PDF.
func nonTitleIdentityValue(value identity.Run, text identity.Text) bool {
	v := layoutSource(value.Source)
	if v.Height <= 0 {
		return false
	}
	for _, label := range text.Runs {
		name := strings.ToLower(strings.TrimSpace(label.Text))
		if name != "issue:" && name != "revision:" && name != "date:" {
			continue
		}
		s := layoutSource(label.Source)
		gap := v.X - (s.X + s.Width)
		if s.Height > 0 && s.Page == v.Page && s.Rotation == v.Rotation && gap >= 0 && gap < s.Height*12 && math.Abs(s.Y-v.Y) < s.Height*.25 {
			return true
		}
	}
	return false
}

// In a stacked title block, Issue can be a right-aligned value under Job No.
// Require that pair below Drawing No.; a history table's Issue is not eligible.
func appendStackedIssueCell(out []Candidate, text identity.Text) []Candidate {
	for _, issue := range text.Runs {
		if !strings.EqualFold(strings.TrimSpace(issue.Text), "Issue") {
			continue
		}
		h := layoutSource(issue.Source)
		if h.Height <= 0 {
			continue
		}
		job, drawing := false, false
		for _, label := range text.Runs {
			s := layoutSource(label.Source)
			if s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.X-h.X) > h.Height*.5 {
				continue
			}
			name := strings.ToLower(strings.Trim(strings.TrimSpace(label.Text), ".:"))
			dy := s.Y - h.Y
			job = job || (name == "job no" && dy > h.Height && dy < h.Height*2.5)
			drawing = drawing || (name == "drawing no" && dy > h.Height*3 && dy < h.Height*12)
		}
		if !job || !drawing {
			continue
		}
		for i, run := range text.Runs {
			s := layoutSource(run.Source)
			v := strings.TrimSpace(run.Text)
			if !accepts(FieldRevision, v) || s.Page != h.Page || s.Rotation != h.Rotation || math.Abs(s.Y-h.Y) > h.Height*.25 || s.X < h.X+h.Width || s.X-h.X > h.Height*20 {
				continue
			}
			start := strings.Index(run.Text, v)
			if c, ok := newCandidate(FieldRevision, v, textProv(run.Source, i, start, start+len(v), true)); ok {
				c.Provenance.OwnRevision = true
				out = mergeCandidate(out, c)
			}
		}
	}
	return out
}
