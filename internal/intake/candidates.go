package intake

import (
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"sitewise/internal/identity"
)

const (
	FieldNumber   = "number"
	FieldRevision = "revision"
	FieldTitle    = "title"
	FieldDate     = "date"

	OriginFilename = "filename"
	OriginText     = "text"
)

// Provenance locates a candidate in the filename or in one identity run.
// Start and End are byte offsets into that filename or run text.
type Provenance struct {
	Origin   string
	Page     int
	Rotation int
	Sheet    string
	Cell     string
	Merge    string
	Table    int
	Row      int
	Col      int
	Start    int
	End      int
	Labeled  bool
	// OwnTitle marks an explicit Drawing/Sheet Title caption, not a generic
	// project title or a neighbouring title-like annotation.
	OwnTitle      bool
	BlockTitle    bool   // uncaptioned title cell within a complete sheet/version grid
	ControlTitle  bool   // explicit Title cell inside a document-control section
	OwnNumber     bool   // drawing number cell beside a current revision caption
	TitleBelow    bool   // inferred block below caption; inline values take priority
	OwnRevision   bool   // explicit Revision cell, rather than an issue-history row
	JoinedRuns    []int  `json:",omitempty"` // full literal runs joined in visual order
	IssueRevision string // explicit compact "REV B 06.11.2023" pairing
	IssueTable    bool   // pairing inferred from separate issue-table columns
	ExportName    bool   // filename ends in a CAD-generated LayoutN suffix
	Heading       bool
	Run           int
}

// Candidate is one possible field value. Display is the literal source text.
// Normalized is only for comparison.
type Candidate struct {
	Field      string
	Display    string
	Normalized string
	Provenance Provenance
}

// Harvest over-finds number, revision, title, and date candidates.
// It does not choose among them.
func Harvest(filename string, text identity.Text) []Candidate {
	out := harvestFilename(filename)
	// A filename's job reference is not its document number when the source
	// explicitly names that same value as the project/job reference.
	jobs := map[string]bool{}
	referenced := map[string]bool{}
	for _, run := range text.Runs {
		// Explicit derivation wording names a source instrument, not this
		// amended document. Do not blacklist all AS-prefixed drawing numbers.
		if strings.Contains(strings.ToLower(run.Text), "amended") {
			for _, m := range amendedStandard.FindAllStringSubmatch(run.Text, -1) {
				referenced[normalizeNumber(m[1]+m[2])] = true
				referenced[normalizeNumber(m[1]+m[2]+m[3])] = true
			}
		}
		for _, m := range projectReference.FindAllStringSubmatch(run.Text, -1) {
			jobs[normalizeNumber(m[1])] = true
		}
	}
	kept := out[:0]
	for _, c := range out {
		if c.Field != FieldNumber || !jobs[c.Normalized] {
			kept = append(kept, c)
		}
	}
	out = appendReportControl(appendScheduleOwnTitle(appendStackedIssueCell(appendMergedNumberTitle(appendCombinedDrawingCell(append(kept, harvestRuns(text)...), text), text), text), text), text)
	out = appendUncaptionedSheetCell(out, text)
	out = appendSheetVersionGrid(out, text)
	out = appendPagedDrawingBlock(out, text)
	// A separately positioned Project No cell is still a project reference,
	// including repetitions in page headers. Explicit drawing labels survive.
	for _, caption := range text.Runs {
		projectName := strings.EqualFold(strings.Trim(strings.TrimSpace(caption.Text), ":."), "Project Name")
		if !projectName && !projectReferenceCaption.MatchString(strings.TrimSpace(caption.Text)) {
			continue
		}
		h := caption.Source
		for _, value := range text.Runs {
			s := value.Source
			if h.Height <= 0 || s.Page != h.Page || s.Rotation != h.Rotation || s.X < h.X+h.Width || s.X-h.X >= h.Height*20 || math.Abs(s.Y-h.Y) >= h.Height*.5 {
				continue
			}
			v := strings.TrimSpace(value.Text)
			if projectName {
				// Only an explicit code-and-name prefix is a project code;
				// numbers elsewhere in a project name can be street addresses.
				parts := strings.Fields(v)
				if len(parts) < 3 || parts[1] != "-" {
					continue
				}
				v = parts[0]
			}
			if isNumber(v) {
				jobs[normalizeNumber(v)] = true
			}
		}
	}
	kept = out[:0]
	for _, c := range out {
		if c.Field == FieldNumber && referenced[c.Normalized] && !c.Provenance.OwnNumber {
			continue
		}
		if c.Field == FieldTitle && approvalStamp(c.Display) {
			continue
		}
		if c.Field != FieldNumber || !jobs[c.Normalized] || c.Provenance.Labeled {
			kept = append(kept, c)
		}
	}
	return kept
}

// titleWindow is how many runs either side of a standalone sheet-number cell
// are read as its title-block neighbours. Title blocks often list values
// apart from their captions, so a title has no label to follow.
const titleWindow = 2

const (
	// Dates stay ahead of numbers so 12/03/2024 is not a document number.
	dateRE = `\d{4}-\d{2}-\d{2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4}|\d{1,2}(?:st|nd|rd|th)?\s+(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*,?\s+\d{2,4}|(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\s+\d{1,2},\s+\d{4}`
	// A bare P-series token is a preliminary revision, not a document number.
	prelimRE = `P\d{1,3}`
	// Hyphenated sheet numbers, lettered job codes, 5-6 digit project numbers,
	// and compact sheets with at least three digits.
	numberRE = `[A-Z]{1,4}\d{3,6}\.\d{1,3}_[A-Z]{1,4}|[A-Z]{2,4}\d{3,6}/\d{2,4}|\d{5,6}\.\d{2}[A-Z]{2}|\d{2}-\d{4}-[A-Z]{1,3}-[A-Z]{1,3}\d{3}|\d{5,6}-\d{2}-[A-Z]{1,4}-[A-Z]\d{2}\.\d{2}|\d{3,6}-\d{1,3}\.\d{1,3}[A-Z]|[A-Z]{1,4}(?:-\d{1,2}){3}|[A-Z]{1,4}-(?:[A-Z]{1,4}|\d{1,2})-\d{1,2}\.\d{2,3}|[A-Z]{1,4}-\d{1,2}-\d{2,4}|[A-Z]{1,4}(?:-[A-Z]{1,4}){0,2}-\d{2,4}|[A-Z]{3,}\d{2,}|\d{5,6}|[A-Z]{1,3}\d{3,4}(?:-\d{2,4})?`
)

var (
	amendedStandard  = regexp.MustCompile(`(?i)\bamended\s+from\s+(AS(?:/NZS)?)\s*(\d{3,5})([-–—]\d{4})?\b`)
	projectReference = regexp.MustCompile(`(?i)\b(?:project|job)\s+(?:number|no\.?|ref\.?)\s*:?\s+([A-Z0-9][A-Z0-9./_-]*)`)
	monthYearTag     = regexp.MustCompile(`(?i)\[(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)\s*(?:\d{2}|\d{4})\]`)
	datePattern      = regexp.MustCompile(`(?i)\b(?:` + dateRE + `)\b`)
	numberPattern    = regexp.MustCompile(`(?i)\b(?:` + numberRE + `)\b`)
	prelimPattern    = regexp.MustCompile(`(?i)\b(?:` + prelimRE + `)\b`)
	revBracket       = regexp.MustCompile(`(?i)\[([A-Z0-9]{1,4}|\d{1,3}\.\d{1,3})\]`)
	revWord          = regexp.MustCompile(`(?i)\brev(?:ision)?\s+([A-Z0-9]{1,4})\b`)
	issueNumber      = regexp.MustCompile(`(?i)\bissue\s*(\d{1,3})\b`)
	// A hyphen joins the issue suffix to the sheet name; a space before (2)
	// is also the browser's duplicate-download suffix and cannot prove revision.
	revParen        = regexp.MustCompile(`(?i)-\(([A-Z]?\d{1,3})\)$`)
	cadLayoutSuffix = regexp.MustCompile(`(?i)(?:^|[-_ ])layout\d+$`)
)

type span struct{ start, end int }

func harvestFilename(name string) []Candidate {
	stem, base := fileStem(name)
	if stem == "" {
		return nil
	}
	var covered []span
	var out []Candidate
	// Underscores are filename separators, but regexp word boundaries treat
	// them as letters. Keep offsets into the original literal unchanged.
	searchStem := strings.ReplaceAll(stem, "_", " ")
	add := func(field, display string, start, end int, cover span) {
		c, ok := newCandidate(field, display, filenameProv(base+start, base+end))
		if !ok {
			return
		}
		out = append(out, c)
		covered = append(covered, cover)
	}
	for _, loc := range datePattern.FindAllStringIndex(stem, -1) {
		add(FieldDate, stem[loc[0]:loc[1]], loc[0], loc[1], span{loc[0], loc[1]})
	}
	// A bracketed month/year is an incomplete date, not a sheet number or
	// revision. Mask the whole tag without inventing a day for the date field.
	for _, loc := range monthYearTag.FindAllStringIndex(stem, -1) {
		covered = append(covered, span{loc[0], loc[1]})
	}
	for _, loc := range revBracket.FindAllStringSubmatchIndex(stem, -1) {
		if overlaps(covered, loc[0], loc[1]) {
			continue
		}
		add(FieldRevision, stem[loc[2]:loc[3]], loc[2], loc[3], span{loc[0], loc[1]})
	}
	for _, pattern := range []*regexp.Regexp{revWord, issueNumber} {
		for _, loc := range pattern.FindAllStringSubmatchIndex(stem, -1) {
			if overlaps(covered, loc[2], loc[3]) {
				continue
			}
			add(FieldRevision, stem[loc[2]:loc[3]], loc[2], loc[3], span{loc[0], loc[1]})
		}
	}
	for _, loc := range prelimPattern.FindAllStringIndex(stem, -1) {
		if overlaps(covered, loc[0], loc[1]) {
			continue
		}
		add(FieldRevision, stem[loc[0]:loc[1]], loc[0], loc[1], span{loc[0], loc[1]})
	}
	for _, loc := range revParen.FindAllStringSubmatchIndex(stem, -1) {
		add(FieldRevision, stem[loc[2]:loc[3]], loc[2], loc[3], span{loc[0], loc[1]})
	}
	for _, loc := range numberPattern.FindAllStringIndex(searchStem, -1) {
		if overlaps(covered, loc[0], loc[1]) {
			continue
		}
		display := stem[loc[0]:loc[1]]
		if fullMatch(datePattern, display) || fullMatch(prelimPattern, display) {
			continue
		}
		// S204-03 may be sheet S204 at revision 03. A stem that is only that
		// is the sheet-and-revision naming convention, so code splits it.
		// Inside a longer name both readings are offered and code does not
		// choose; Jev reads the fused literal too literally to be asked.
		m := suffixedSheet.FindStringSubmatchIndex(display)
		if m == nil || loc[0] != 0 || loc[1] != len(stem) {
			add(FieldNumber, display, loc[0], loc[1], span{loc[0], loc[1]})
		}
		if m != nil {
			add(FieldNumber, display[m[2]:m[3]], loc[0]+m[2], loc[0]+m[3], span{loc[0], loc[1]})
			add(FieldRevision, display[m[4]:m[5]], loc[0]+m[4], loc[0]+m[5], span{loc[0], loc[1]})
		}
	}
	if end := leadToken(stem); end > 0 && isShortSheet(stem[:end]) && !overlaps(covered, 0, end) {
		add(FieldNumber, stem[:end], 0, end, span{0, end})
	}
	if shortName(stem) {
		return out
	}
	if display, start, end, ok := titleFromGaps(stem, covered); ok {
		c, made := newCandidate(FieldTitle, display, filenameProv(base+start, base+end))
		if made {
			c.Provenance.ExportName = cadLayoutSuffix.MatchString(display)
			out = append(out, c)
		}
	}
	return out
}

func harvestRuns(text identity.Text) []Candidate {
	labelOnly := make([]string, len(text.Runs))
	out := make([]Candidate, 0, len(text.Runs)*4)
	for i, run := range text.Runs {
		var only string
		out, only = appendRun(out, run, i)
		labelOnly[i] = only
	}
	for i := 0; i < len(text.Runs)-1; i++ {
		if labelOnly[i] == "" {
			continue
		}
		if (labelOnly[i] == FieldNumber || labelOnly[i] == FieldTitle || labelOnly[i] == FieldRevision) && text.Runs[i].Source.Width > 0 {
			continue // PDF cells bind spatially below, not by content-stream order.
		}
		c, ok := candidateFromRun(text.Runs[i+1], i+1, labelOnly[i])
		if !ok {
			continue
		}
		c.Provenance.OwnTitle = labelOnly[i] == FieldTitle && ownTitleCaption(text.Runs[i].Text)
		out = mergeCandidate(out, c)
	}
	out = appendSpatialIdentity(out, text, labelOnly)
	out = appendIssueTableDates(out, text)
	out = appendExplicitReportReferences(out, text)
	out = appendReportHeadings(out, text)
	for i, c := range out {
		if c.Field != FieldTitle || c.Provenance.Run < 0 || !c.Provenance.Labeled {
			continue
		}
		line := strings.ToLower(strings.TrimSpace(text.Runs[c.Provenance.Run].Text))
		if strings.HasPrefix(line, "re:") || strings.HasPrefix(line, "subject:") {
			out[i] = joinTitleBelow(c, text, c.Provenance.Run)
		}
	}
	return appendNeighbourTitles(out, text)
}

// PDF runs have geometry but no semantic heading flag. Over-find prominent
// report headings as literal choices; prominence alone never settles a title.
func appendReportHeadings(out []Candidate, text identity.Text) []Candidate {
	if text.Format != "pdf" {
		return out
	}
	out = appendUnitPlanHeadings(out, text)
	var heights []float64
	for _, run := range text.Runs {
		if run.Source.Page == 1 && run.Source.Rotation == 0 && run.Source.Height > 0 {
			heights = append(heights, run.Source.Height)
		}
	}
	if len(heights) < 4 {
		return out
	}
	sort.Float64s(heights)
	// Covers contain oversized project, address and control text as well as
	// the report heading. The lower quartile estimates the small body text
	// without letting a single tiny page number define prominence.
	bodyHeight := heights[(len(heights)-1)/4]
	for i, run := range text.Runs {
		s := strings.TrimSpace(run.Text)
		// PDFium boxes carry floating-point noise; hundredths of a point must
		// not decide whether an exactly 1.4x heading is harvested.
		if run.Source.Page != 1 || run.Source.Rotation != 0 || !captionTitle(s) {
			continue
		}
		words := strings.Fields(strings.ToLower(s))
		application := strings.EqualFold(s, "Development Application")
		// A standalone schedule cover names the schedule itself. Technical
		// sheet titles such as COLUMN SCHEDULE already use title-block evidence.
		heading := application || strings.HasPrefix(strings.ToUpper(s), "SCHEDULE OF ")
		legalHeading := false
		for _, word := range words {
			if word == "contract" || word == "agreement" || word == "deed" {
				legalHeading = true
			}
			switch word {
			case "report", "advice", "statement", "memorandum", "specification", "certificate", "recommendations", "review", "brief", "assessment", "response", "manual", "contract", "agreement", "deed", "requirements", "matrix":
				heading = true
			}
		}
		if !heading {
			continue
		}
		// Sparse manual covers can print the document title at body size.
		// Offer its literal capital heading without treating it as settled.
		manualHeading := len(text.Runs) <= 20 && s == strings.ToUpper(s) && strings.HasSuffix(s, " MANUAL")
		// Legal instruments often use body-size capitals for their cover
		// title. Offer that literal early heading; do not settle it by rule.
		legalHeading = legalHeading && i < 20 && len(words) >= 3 && s == strings.ToUpper(s)
		if run.Source.Height+.01 < 1.4*bodyHeight && !manualHeading && !legalHeading {
			if c, ok := compactCenteredHeading(text, i); ok {
				out = mergeCandidate(out, c)
			}
			continue
		}
		first := i
		for first > 0 && i-first < 3 {
			previous := text.Runs[first-1]
			p, h := previous.Source, run.Source
			gap := p.Y - (h.Y + h.Height)
			if p.Page == h.Page && p.Rotation == h.Rotation && (math.Abs(p.X-h.X) < h.Height*.5 || math.Abs(p.X+p.Width/2-h.X-h.Width/2) < h.Height*.5) && math.Abs(p.Height-h.Height) < h.Height*.2 && gap >= -h.Height*.25 && gap < h.Height*.5 && captionTitle(previous.Text) {
				first, run, s = first-1, previous, strings.TrimSpace(previous.Text)
			} else {
				break
			}
		}
		start := strings.Index(run.Text, s)
		prov := textProv(run.Source, first, start, start+len(s), false)
		prov.Heading = true
		if c, ok := newCandidate(FieldTitle, s, prov); ok {
			c = joinTitleLines(c, text, first, true)
			out = mergeCandidate(out, c)
		}
	}
	return out
}

// PDF content order is not visual order: the value may be emitted before its
// caption. Bind only a nearby number or title cell on the same row or directly
// below its explicit caption, with no competing equally near cell.
func appendSpatialIdentity(out []Candidate, text identity.Text, labels []string) []Candidate {
	for i, field := range labels {
		if field != FieldNumber && field != FieldTitle && field != FieldRevision {
			continue
		}
		// Short REV is also the heading of a history table. Only the full
		// Revision caption identifies a current-value cell for spatial binding.
		versionCaption := strings.EqualFold(strings.TrimSpace(text.Runs[i].Text), "Version:")
		if field == FieldRevision && !ownRevisionCaption(text.Runs[i].Text) && !coverRevisionCaption(text.Runs[i], text) && !versionCaption {
			continue
		}
		label := layoutSource(text.Runs[i].Source)
		if field == FieldTitle && drawingScheduleTitle(text.Runs[i], text) {
			continue
		}
		controlTitle := field == FieldTitle && documentControlTitle(text.Runs[i], text)
		if field == FieldRevision && registerRevisionCaption(label, text) {
			continue
		}
		if label.Width <= 0 || label.Height <= 0 {
			continue
		}
		best, score, tied := -1, 1e9, false
		for j, run := range text.Runs {
			s := layoutSource(run.Source)
			if i == j || s.Page != label.Page || s.Width <= 0 || s.Height <= 0 || s.Rotation != label.Rotation {
				continue
			}
			v := strings.TrimSpace(run.Text)
			if labels[j] != "" || (field == FieldNumber && !isNumber(v) && !isShortSheet(v)) || (field == FieldTitle && !captionTitle(v)) || (field == FieldRevision && !accepts(field, v)) {
				continue
			}
			dx, dy := s.X-label.X, label.Y-(s.Y+s.Height)
			d := 1e9
			if dx >= 0 && s.X >= label.X+label.Width-label.Height*.5 && s.X-(label.X+label.Width) < label.Height*12 && math.Abs(s.Y-label.Y) < label.Height {
				d = math.Max(0, s.X-label.X-label.Width)
				if controlTitle && math.Abs(s.Y-label.Y) < label.Height*.25 {
					d -= label.Height * 20
				}
				// A tightly adjacent inline value precedes the next table row.
				if field == FieldTitle && d < label.Height*2 {
					d -= label.Height * 12
				}
			}
			if math.Abs(dx) < label.Height*2 && dy >= -label.Height*.5 && dy < label.Height*6 {
				d = math.Min(d, math.Max(0, dy)+math.Abs(dx))
			}
			if rightAlignedIdentity(field, label, s, text) {
				d = math.Min(d, math.Max(0, dy)+math.Abs(dx))
			}
			if d < score {
				best, score, tied = j, d, false
			} else if d == score && d < 1e9 {
				tied = true
			}
		}
		if best >= 0 && !tied {
			run := text.Runs[best]
			v := strings.TrimSpace(run.Text)
			if field == FieldTitle && strings.EqualFold(strings.TrimSpace(text.Runs[i].Text), "Report Type:") {
				// The report-type cell can end in an explicit revision. Keep
				// the literal title substring rather than folding that issue into it.
				if suffix := reportTypeRevision.FindStringIndex(v); suffix != nil {
					v = strings.TrimSpace(v[:suffix[0]])
				}
			}
			start := strings.Index(run.Text, v)
			if c, ok := newCandidate(field, v, textProv(run.Source, best, start, start+len(v), true)); ok {
				s := layoutSource(run.Source)
				gap := s.X - (label.X + label.Width)
				below := label.Y - (s.Y + s.Height)
				inline := gap >= 0 && gap < label.Height*2 && math.Abs(s.Y-label.Y) < label.Height
				under := math.Abs(s.X-label.X) < label.Height && below >= -label.Height*.5 && below < label.Height*2
				under = under || rightAlignedIdentity(field, label, s, text)
				c.Provenance.OwnTitle = field == FieldTitle && spatialOwnTitle(text.Runs[i], text) && (inline || under)
				c.Provenance.ControlTitle = controlTitle && gap >= 0 && gap < label.Height*12 && math.Abs(s.Y-label.Y) < label.Height*.25
				c.Provenance.TitleBelow = c.Provenance.OwnTitle && under
				c.Provenance.BlockTitle = c.Provenance.OwnTitle && strings.EqualFold(strings.Trim(strings.TrimSpace(text.Runs[i].Text), ":"), "Title")
				c.Provenance.OwnRevision = field == FieldRevision && (ownRevisionCaption(text.Runs[i].Text) || coverRevisionCaption(text.Runs[i], text))
				c.Provenance.OwnNumber = field == FieldNumber && (inline || under) && besideRevisionCaption(label, text)
				if c.Provenance.OwnTitle {
					if under {
						c = joinTitleBelow(c, text, best)
						c = joinSheetSubtitle(c, text, best)
					} else {
						c = joinInlineTitle(c, text, label, best)
					}
				}
				out = mergeCandidate(out, c)
			}
		}
	}
	return out
}

// A small regulated-design title cell can have two lines beside one caption.
// Join only aligned, same-size lines spanning that caption, not the next row.
func joinInlineTitle(c Candidate, text identity.Text, label identity.Source, best int) Candidate {
	anchor := layoutSource(text.Runs[best].Source)
	indices := []int{}
	for i, r := range text.Runs {
		s := layoutSource(r.Source)
		if s.Page == anchor.Page && s.Rotation == anchor.Rotation &&
			math.Abs(s.X-anchor.X) < anchor.Height*.5 &&
			math.Abs(s.Height-anchor.Height) < anchor.Height*.25 &&
			math.Abs((s.Y+s.Height/2)-(label.Y+label.Height/2)) < label.Height*.8 && captionTitle(r.Text) {
			indices = append(indices, i)
		}
	}
	if len(indices) != 2 {
		return c
	}
	sort.Slice(indices, func(i, j int) bool {
		return layoutSource(text.Runs[indices[i]].Source).Y > layoutSource(text.Runs[indices[j]].Source).Y
	})
	top, bottom := text.Runs[indices[0]], text.Runs[indices[1]]
	if layoutSource(top.Source).Y-layoutSource(bottom.Source).Y < anchor.Height*.5 {
		return c // overlapping duplicate text layers are not title continuations
	}
	c.Display = strings.TrimSpace(top.Text) + " " + strings.TrimSpace(bottom.Text)
	c.Normalized = normalizeTitle(c.Display)
	c.Provenance.JoinedRuns = indices
	return c
}

// appendNeighbourTitles offers title-like cells near a cell that holds only
// a sheet number. They are unlabeled candidates; Jev chooses among them.
func appendNeighbourTitles(out []Candidate, text identity.Text) []Candidate {
	seen := map[int]bool{}
	for i, run := range text.Runs {
		v := strings.TrimSpace(run.Text)
		if i > 0 && strings.EqualFold(strings.TrimSpace(text.Runs[i-1].Text), "REPORT NUMBER") {
			continue // A report cover's adjacent address/date cells are not sheet titles.
		}
		if v == "" || strings.ContainsAny(v, " \t") || !(isNumber(v) || isShortSheet(v)) {
			continue
		}
		for j := i - titleWindow; j <= i+titleWindow; j++ {
			if j < 0 || j >= len(text.Runs) || j == i || seen[j] {
				continue
			}
			display := strings.TrimSpace(text.Runs[j].Text)
			anchor, near := layoutSource(run.Source), layoutSource(text.Runs[j].Source)
			// PDF content order can place a header beside a footer. Only visual
			// neighbours are useful title evidence when geometry is available.
			if anchor.Height > 0 && near.Height > 0 && (anchor.Page != near.Page || anchor.Rotation != near.Rotation || math.Abs(anchor.Y-near.Y) > math.Max(anchor.Height, near.Height)*8) {
				continue
			}
			tokens := words(display, nil)
			if len(tokens) > 0 {
				if _, _, labelled := labelAt(display, tokens, 0); labelled {
					continue
				}
			}
			if nonTitleIdentityValue(text.Runs[j], text) || !cellTitle(display) {
				continue
			}
			start := strings.Index(text.Runs[j].Text, display)
			if c, ok := newCandidate(FieldTitle, display, textProv(text.Runs[j].Source, j, start, start+len(display), false)); ok {
				out = append(out, c)
				seen[j] = true
			}
		}
	}
	return out
}

// cellTitle is a short multi-word cell that reads as a title, not a note.
func cellTitle(s string) bool {
	words := strings.Fields(s)
	return len(words) >= 2 && len(words) <= 12 && len(s) <= 120 && looksLikeTitle(s)
}

func ownTitleCaption(s string) bool {
	s = strings.ToLower(strings.TrimSpace(strings.TrimRight(s, ":.")))
	return s == "drawing title" || s == "drawing name" || s == "sheet title"
}

func captionTitle(s string) bool {
	// NOTES is a caption elsewhere, but a valid value after Drawing Title.
	return len(s) <= 120 && len(strings.Fields(s)) <= 12 && (looksLikeTitle(s) || strings.EqualFold(s, "notes"))
}

type word struct{ start, end int }

func appendRun(out []Candidate, run identity.Run, index int) ([]Candidate, string) {
	text := run.Text
	// Some rotated PDFs merge stacked Job No / Dwg No captions. The bottom
	// caption binds the value below the combined box, only in a tall layout.
	box := layoutSource(run.Source)
	if strings.EqualFold(strings.TrimSpace(text), "Job No: Dwg No:") && box.Height > box.Width && box.Width > 0 {
		return out, FieldNumber
	}
	if strings.EqualFold(strings.TrimSpace(text), "Project Number/Drawing Number") {
		return out, FieldNumber
	}
	if strings.EqualFold(strings.TrimSpace(text), "REV BY") {
		return out, ""
	}
	// CAD plot footers describe the export operation, not the drawing issue.
	upper := strings.ToUpper(text)
	if strings.Contains(upper, "ALL RIGHTS RESERVED") {
		return out, "" // Copyright template IDs are not document identity fields.
	}
	if strings.Contains(upper, "ISO A") && strings.Contains(upper, " MM)") && strings.Contains(upper, "COPYRIGHT") && (strings.Contains(upper, " AM,") || strings.Contains(upper, " PM,")) {
		return out, ""
	}
	var storage [24]word
	tokens := words(text, storage[:0])
	labelOnly := ""
	emittedTitle := false
	for i := 0; i < len(tokens); {
		// Professional registration is the author's identity, not this file's.
		if eq(text[tokens[i].start:tokens[i].end], "rpeq") && i+1 < len(tokens) {
			i += 2
			continue
		}
		// Explicit project/job references identify the commission, not this
		// document. Skip only that value so later identity fields still parse.
		if i+2 < len(tokens) {
			a, b := text[tokens[i].start:tokens[i].end], text[tokens[i+1].start:tokens[i+1].end]
			if (eq(a, "project") || eq(a, "job")) && (eq(b, "no") || eq(b, "number") || eq(b, "ref")) {
				i += 3
				continue
			}
		}
		if field, n, ok := labelAt(text, tokens, i); ok {
			next := i + n
			if field == FieldRevision && next+1 < len(tokens) {
				start, end := tokens[next].start, tokens[next+1].end
				if reportRevision.MatchString(text[start:end]) || stagedReportRevision.MatchString(text[start:end]) {
					c, _ := newCandidate(field, text[start:end], textProv(run.Source, index, start, end, true))
					c.Provenance.OwnRevision = strings.EqualFold(text[tokens[i].start:tokens[i].end], "Revision:")
					out = append(out, c)
					i = next + 2
					continue
				}
			}
			if field == FieldTitle {
				if display, start, end, took, titled := takeTitle(text, tokens, next); titled {
					if c, made := newCandidate(FieldTitle, display, textProv(run.Source, index, start, end, true)); made {
						c.Provenance.OwnTitle = n == 2 && (eq(text[tokens[i].start:tokens[i].end], "drawing") || eq(text[tokens[i].start:tokens[i].end], "sheet"))
						out = append(out, c)
						emittedTitle = true
					}
					i = next + took
					continue
				}
			} else if next < len(tokens) && accepts(field, text[tokens[next].start:tokens[next].end]) {
				tok := tokens[next]
				if c, made := newCandidate(field, text[tok.start:tok.end], textProv(run.Source, index, tok.start, tok.end, true)); made {
					c.Provenance.OwnRevision = field == FieldRevision && strings.EqualFold(text[tokens[i].start:tokens[i].end], "Revision:")
					if field == FieldRevision && run.Source.Page == 1 && i == 0 && next == len(tokens)-1 && eq(text[tokens[i].start:tokens[i].end], "version") {
						c.Provenance.OwnRevision = true
					}
					out = append(out, c)
				}
				i = next + 1
				continue
			}
			if i == 0 && next == len(tokens) {
				labelOnly = field
			}
			i = next
			continue
		}
		tok := tokens[i]
		display := text[tok.start:tok.end]
		standalone := len(tokens) == 1
		if standalone && isShortSheet(display) {
			if c, ok := newCandidate(FieldNumber, display, textProv(run.Source, index, tok.start, tok.end, false)); ok {
				out = append(out, c)
			}
			i++
			continue
		}
		// A P-number inside a note (a pit, a pump) is not a revision unless
		// the note is about issue or revision.
		if isPrelim(display) && !standalone && !revisionContext(text) {
			i++
			continue
		}
		if kind := kindOf(display); kind != "" {
			if c, ok := newCandidate(kind, display, textProv(run.Source, index, tok.start, tok.end, false)); ok {
				if kind == FieldDate && len(tokens) == 3 && i == 2 && (eq(text[tokens[0].start:tokens[0].end], "rev") || eq(text[tokens[0].start:tokens[0].end], "revision")) {
					c.Provenance.IssueRevision, _ = revisionNormalized(text[tokens[1].start:tokens[1].end])
				}
				out = append(out, c)
			}
			i++
			continue
		}
		if i+2 < len(tokens) && ((isDay(display) && isMonth(text[tokens[i+1].start:tokens[i+1].end]) && isYear(text[tokens[i+2].start:tokens[i+2].end])) || (isMonth(display) && monthFirstDate(text[tok.start:tokens[i+2].end]))) {
			start, end := tok.start, tokens[i+2].end
			if c, ok := newCandidate(FieldDate, text[start:end], textProv(run.Source, index, start, end, false)); ok {
				out = append(out, c)
			}
			i += 3
			continue
		}
		i++
	}
	if run.Source.Heading && !emittedTitle {
		display := strings.TrimSpace(text)
		if looksLikeTitle(display) {
			start := strings.Index(text, display)
			if start >= 0 {
				if c, ok := newCandidate(FieldTitle, display, textProv(run.Source, index, start, start+len(display), false)); ok {
					out = append(out, c)
				}
			}
		}
	}
	return out, labelOnly
}

func words(s string, buf []word) []word {
	buf = buf[:0]
	i := 0
	for i < len(s) {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		j := i
		for j < len(s) && s[j] != ' ' && s[j] != '\t' {
			j++
		}
		buf = append(buf, word{i, j})
		i = j
	}
	return buf
}

func candidateFromRun(run identity.Run, index int, field string) (Candidate, bool) {
	display := strings.TrimSpace(run.Text)
	start := strings.Index(run.Text, display)
	if display == "" || start < 0 {
		return Candidate{}, false
	}
	end := start + len(display)
	switch field {
	case FieldNumber:
		if !isNumber(display) {
			return Candidate{}, false
		}
	case FieldRevision:
		if _, ok := revisionNormalized(display); !ok {
			return Candidate{}, false
		}
	case FieldDate:
		if !isDate(display) {
			return Candidate{}, false
		}
	case FieldTitle:
		if !looksLikeTitle(display) {
			return Candidate{}, false
		}
	default:
		return Candidate{}, false
	}
	return newCandidate(field, display, textProv(run.Source, index, start, end, true))
}

func mergeCandidate(out []Candidate, c Candidate) []Candidate {
	for i := range out {
		if out[i].Provenance.Origin != OriginText || out[i].Provenance.Run != c.Provenance.Run {
			continue
		}
		if out[i].Provenance.Start != c.Provenance.Start || out[i].Provenance.End != c.Provenance.End {
			continue
		}
		out[i].Field = c.Field
		out[i].Display = c.Display
		out[i].Normalized = c.Normalized
		out[i].Provenance.Labeled = true
		out[i].Provenance.OwnTitle = out[i].Provenance.OwnTitle || c.Provenance.OwnTitle
		out[i].Provenance.BlockTitle = out[i].Provenance.BlockTitle || c.Provenance.BlockTitle
		out[i].Provenance.ControlTitle = out[i].Provenance.ControlTitle || c.Provenance.ControlTitle
		out[i].Provenance.Heading = out[i].Provenance.Heading || c.Provenance.Heading
		out[i].Provenance.IssueTable = out[i].Provenance.IssueTable || c.Provenance.IssueTable
		out[i].Provenance.OwnNumber = out[i].Provenance.OwnNumber || c.Provenance.OwnNumber
		out[i].Provenance.TitleBelow = c.Provenance.TitleBelow
		out[i].Provenance.OwnRevision = out[i].Provenance.OwnRevision || c.Provenance.OwnRevision
		if len(c.Provenance.JoinedRuns) > 0 {
			out[i].Provenance.JoinedRuns = c.Provenance.JoinedRuns
		}
		return out
	}
	return append(out, c)
}

func newCandidate(field, display string, prov Provenance) (Candidate, bool) {
	normalized, ok := normalize(field, display)
	if !ok {
		return Candidate{}, false
	}
	prov.Labeled = prov.Labeled || prov.Origin == OriginFilename
	return Candidate{Field: field, Display: display, Normalized: normalized, Provenance: prov}, true
}

// Normalize is the comparison form intake uses for a harvested value. ok is
// false when the value is not a valid number, revision, title or date.
func Normalize(field, display string) (string, bool) {
	return normalize(field, display)
}

func normalize(field, display string) (string, bool) {
	switch field {
	case FieldNumber:
		n := normalizeNumber(display)
		return n, n != ""
	case FieldRevision:
		return revisionNormalized(display)
	case FieldDate:
		n := strings.TrimSpace(display)
		return n, n != ""
	case FieldTitle:
		n := normalizeTitle(display)
		return n, n != ""
	default:
		return "", false
	}
}

func normalizeNumber(s string) string {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' || c == ' ' || c == '\t' || c == '_' || c >= 0x80 {
			return normalizeNumberSlow(s)
		}
	}
	return s
}

func normalizeNumberSlow(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t':
			continue
		case '\u2013', '\u2014', '\u2212', '_':
			b.WriteByte('-')
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

func normalizeTitle(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '_', '-', '\u2013', '\u2014':
			b.WriteByte(' ')
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func looksLikeTitle(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if isCaption(s) {
		return false
	}
	if isDate(s) || isNumber(s) {
		return false
	}
	if _, ok := parseRevision(s); ok && !strings.ContainsAny(s, " \t") {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func labelAt(text string, tokens []word, i int) (field string, n int, ok bool) {
	a := text[tokens[i].start:tokens[i].end]
	if !wordLabel(a) {
		return "", 0, false
	}
	switch {
	case eq(a, "issue") && len(tokens) == 1 && strings.HasSuffix(strings.TrimSpace(text), ":"):
		return FieldRevision, 1, true
	case eq(a, "date"):
		if i+1 < len(tokens) && eq(text[tokens[i+1].start:tokens[i+1].end], "prepared") {
			return FieldDate, 2, true
		}
		return FieldDate, 1, true
	case (i == 0 && (strings.EqualFold(a, "Subject:") || strings.EqualFold(a, "Re:") || strings.EqualFold(a, "Report:"))):
		return FieldTitle, 1, true
	case eq(a, "title"):
		// Land-registry Title System describes tenure, not document identity.
		if i+1 < len(tokens) && eq(text[tokens[i+1].start:tokens[i+1].end], "system") {
			return "", 0, false
		}
		if i > 0 && eq(text[tokens[i-1].start:tokens[i-1].end], "project") {
			return "", 0, false
		}
		return FieldTitle, 1, true
	case eq(a, "version"):
		if i+1 < len(tokens) && eq(text[tokens[i+1].start:tokens[i+1].end], "no") {
			return FieldRevision, 2, true
		}
		return FieldRevision, 1, true
	case eq(a, "rev") || eq(a, "revision"):
		if i+1 < len(tokens) && eq(text[tokens[i+1].start:tokens[i+1].end], "issue") {
			return FieldRevision, 2, true
		}
		return FieldRevision, 1, true
	}
	if i+1 >= len(tokens) {
		return "", 0, false
	}
	b := text[tokens[i+1].start:tokens[i+1].end]
	switch {
	case eq(a, "report") && eq(b, "type"):
		return FieldTitle, 2, true
	case eq(a, "drawing") && (eq(b, "title") || eq(b, "name")):
		return FieldTitle, 2, true
	case eq(a, "drawing") && (eq(b, "no") || eq(b, "number")):
		return FieldNumber, 2, true
	case (eq(a, "drg") || eq(a, "dwg")) && eq(b, "no"):
		return FieldNumber, 2, true
	case (eq(a, "document") || eq(a, "report")) && (eq(b, "no") || eq(b, "number")):
		return FieldNumber, 2, true
	case eq(a, "doc") && eq(b, "ref"):
		return FieldNumber, 2, true
	case eq(a, "sheet") && eq(b, "title"):
		return FieldTitle, 2, true
	case eq(a, "sheet") && (eq(b, "no") || eq(b, "number")):
		return FieldNumber, 2, true
	case eq(a, "version") && eq(b, "no"):
		return FieldRevision, 2, true
	case eq(a, "issue") && eq(b, "date"):
		return FieldDate, 2, true
	default:
		return "", 0, false
	}
}

func takeTitle(text string, tokens []word, from int) (display string, start, end, took int, ok bool) {
	if from >= len(tokens) {
		return "", 0, 0, 0, false
	}
	last := from
	for last < len(tokens) {
		if _, _, isLabel := labelAt(text, tokens, last); isLabel {
			break
		}
		last++
	}
	if last == from {
		return "", 0, 0, 0, false
	}
	rawStart := tokens[from].start
	rawEnd := tokens[last-1].end
	raw := text[rawStart:rawEnd]
	display = strings.Trim(raw, " \t-_:.")
	if !looksLikeTitle(display) {
		return "", 0, 0, 0, false
	}
	rel := strings.Index(raw, display)
	return display, rawStart + rel, rawStart + rel + len(display), last - from, true
}

func accepts(field, display string) bool {
	switch field {
	case FieldNumber:
		return isNumber(display)
	case FieldDate:
		return isDate(display)
	case FieldRevision:
		_, ok := revisionNormalized(display)
		return ok
	default:
		return false
	}
}

func kindOf(s string) string {
	switch {
	case isDate(s):
		return FieldDate
	case isPrelim(s):
		return FieldRevision
	case isNumber(s):
		return FieldNumber
	default:
		return ""
	}
}

func isDate(s string) bool {
	if monthFirstDate(s) {
		return true
	}
	if parts := strings.Fields(s); len(parts) == 3 && isDay(parts[0]) && isMonth(parts[1]) && isYear(parts[2]) {
		return true
	}
	if len(s) == 10 && s[4] == '-' && s[7] == '-' && digitsOnly(s[:4]) && digitsOnly(s[5:7]) && digitsOnly(s[8:]) {
		return calendar(atoiOr(s[8:]), atoiOr(s[5:7]))
	}
	sep := byte(0)
	parts := 0
	run := 0
	first := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '/' || c == '.' {
			if sep == 0 {
				sep = c
			} else if c != sep {
				return false
			}
			if parts == 0 {
				first = run
			}
			if run < 1 || run > 4 {
				return false
			}
			parts++
			run = 0
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
		run++
	}
	if parts != 2 || first < 1 || first > 2 || run < 2 || run > 4 {
		return false
	}
	// Day first, as Australian documents write it.
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == rune(sep) })
	return calendar(atoiOr(fields[0]), atoiOr(fields[1]))
}

func monthFirstDate(s string) bool {
	if len(s) < 11 || !isLetter(s[0]) {
		return false
	}
	parts := strings.Fields(s)
	if len(parts) != 3 || !isMonth(parts[0]) || !strings.HasSuffix(parts[1], ",") || len(parts[2]) != 4 {
		return false
	}
	for _, layout := range []string{"January 2, 2006", "Jan 2, 2006"} {
		if _, err := time.Parse(layout, strings.Join(parts, " ")); err == nil {
			return true
		}
	}
	return false
}

func calendar(day, month int) bool {
	return day >= 1 && day <= 31 && month >= 1 && month <= 12
}

func atoiOr(s string) int {
	n, ok := atoi(s)
	if !ok {
		return -1
	}
	return n
}

// PDF runs can merge adjacent reference captions into one text string. This
// grammar requires at least two complete captions and no intervening values.
var joinedNumberCaptions = regexp.MustCompile(`(?i)^(?:C\.?A\.?P\.?|FILE|DRAWING|SHEET|PROJECT|JOB)\s+NO\.?(?:\s+(?:C\.?A\.?P\.?|FILE|DRAWING|SHEET|PROJECT|JOB)\s+NO\.?)+$`)

// isCaption is title-block caption text: a label, not a value.
func isCaption(s string) bool {
	s = strings.TrimSpace(s)
	if joinedNumberCaptions.MatchString(s) {
		return true
	}
	if strings.HasSuffix(s, ":") {
		return true
	}
	return captions[strings.Join(strings.Fields(strings.ToLower(strings.Trim(s, ".:"))), " ")]
}

var captions = map[string]bool{
	"revision issue date": true,
	"rev date":            true, "reg no": true, "full name": true,
	"rev": true, "revision": true, "date": true, "drawing": true, "no": true, "number": true,
	"sheet": true, "title": true, "project": true, "drawn": true, "drawn by": true,
	"checked": true, "checked by": true, "designed": true, "designed by": true,
	"approved": true, "approved by": true, "architect": true, "client": true,
	"scale": true, "job": true, "job no": true, "project no": true, "project title": true,
	"drawing no": true, "drawing number": true, "drawing title": true, "drawing name": true, "sheet title": true,
	"sheet no": true, "status": true, "issue": true, "amendment": true, "amendments": true,
	"report number": true, "date issued": true, "draft": true,
	"north": true, "notes": true, "consultant": true, "engineer": true, "description": true,
}

// revisionContext reports whether a run talks about issue or revision.
func revisionContext(s string) bool {
	l := strings.ToLower(s)
	for _, w := range []string{"issue", "rev", "amend", "status", "prelim"} {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// isShortSheet is a one- or two-letter sheet id with two digits, such as
// E01. It is only a number candidate where it stands alone.
func isShortSheet(s string) bool {
	if len(s) < 3 || len(s) > 4 || isPrelim(s) {
		return false
	}
	letters, digits, ok := trailingDigits(s)
	return ok && len(letters) <= 2 && len(digits) == 2 && strings.ToUpper(letters) == letters
}

// leadToken is the end of the first filename token, before a space, hyphen
// or underscore.
func leadToken(stem string) int {
	for i := 0; i < len(stem); i++ {
		switch stem[i] {
		case ' ', '-', '_':
			return i
		}
	}
	return len(stem)
}

// shortName is a Windows 8.3 alias such as E01-EL~1: truncated, so its words
// are not a title.
func shortName(stem string) bool {
	i := strings.IndexByte(stem, '~')
	return i >= 0 && i+1 < len(stem) && stem[i+1] >= '0' && stem[i+1] <= '9'
}

var suffixedSheet = regexp.MustCompile(`^([A-Za-z]{1,3}\d{3,4})-(\d{2})$`)

func isNumber(s string) bool {
	if s == "" || isDate(s) || isPrelim(s) {
		return false
	}
	if (strings.ContainsAny(s, "./") || (len(s) >= 2 && digitsOnly(s[:2]) && strings.Count(s, "-") == 3)) && fullMatch(numberPattern, s) {
		return true
	}
	if digitsOnly(s) {
		return len(s) == 5 || len(s) == 6
	}
	if !strings.Contains(s, "-") {
		letters, digits, ok := trailingDigits(s)
		if !ok {
			return false
		}
		if len(letters) >= 3 && len(digits) >= 2 {
			return true
		}
		return len(letters) >= 1 && len(letters) <= 3 && len(digits) >= 3 && len(digits) <= 4
	}
	return hyphenatedNumber(s)
}

func isPrelim(s string) bool {
	if len(s) < 2 || len(s) > 4 || (s[0] != 'P' && s[0] != 'p') {
		return false
	}
	return digitsOnly(s[1:])
}

func trailingDigits(s string) (letters, digits string, ok bool) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	if i == 0 || i == len(s) || !lettersASCII(s[:i]) {
		return "", "", false
	}
	return s[:i], s[i:], true
}

func hyphenatedNumber(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) == 4 && lettersASCII(parts[0]) && len(parts[0]) <= 4 {
		ok := true
		for _, p := range parts[1:] {
			ok = ok && len(p) >= 1 && len(p) <= 2 && digitsOnly(p)
		}
		if ok {
			return true
		}
	}
	if len(parts) == 3 && lettersASCII(parts[0]) && len(parts[0]) <= 4 && len(parts[1]) >= 1 && len(parts[1]) <= 2 && digitsOnly(parts[1]) && len(parts[2]) >= 2 && len(parts[2]) <= 4 && digitsOnly(parts[2]) {
		return true
	}
	hyphens := strings.Count(s, "-")
	if hyphens < 1 || hyphens > 3 {
		return false
	}
	last := strings.LastIndexByte(s, '-')
	tail := s[last+1:]
	if len(tail) < 2 || len(tail) > 4 || !digitsOnly(tail) {
		return false
	}
	head := s[:last]
	if hyphens == 1 && compactSheet(head) {
		return true
	}
	seg := 0
	for i := 0; i <= len(head); i++ {
		if i == len(head) || head[i] == '-' {
			if seg < 1 || seg > 4 || !lettersASCII(head[i-seg:i]) {
				return false
			}
			seg = 0
			continue
		}
		if !isLetter(head[i]) {
			return false
		}
		seg++
	}
	return true
}

func compactSheet(s string) bool {
	i := 0
	for i < len(s) && isLetter(s[i]) {
		i++
	}
	digits := len(s) - i
	return i >= 1 && i <= 3 && digits >= 3 && digits <= 4 && digitsOnly(s[i:])
}

func lettersASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isLetter(s[i]) {
			return false
		}
	}
	return true
}

func isLetter(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func isDay(s string) bool {
	for _, suffix := range []string{"st", "nd", "rd", "th"} {
		if strings.HasSuffix(strings.ToLower(s), suffix) {
			s = s[:len(s)-2]
			break
		}
	}
	return (len(s) == 1 || len(s) == 2) && digitsOnly(s) && atoiOr(s) >= 1 && atoiOr(s) <= 31
}

func isYear(s string) bool {
	return (len(s) == 2 || len(s) == 4) && digitsOnly(s)
}

func isMonth(s string) bool {
	for len(s) > 0 && (s[len(s)-1] == '.' || s[len(s)-1] == ',') {
		s = s[:len(s)-1]
	}
	switch len(s) {
	case 3:
		return eq(s, "jan") || eq(s, "feb") || eq(s, "mar") || eq(s, "apr") || eq(s, "may") || eq(s, "jun") || eq(s, "jul") || eq(s, "aug") || eq(s, "sep") || eq(s, "oct") || eq(s, "nov") || eq(s, "dec")
	case 4:
		return eq(s, "june") || eq(s, "july") || eq(s, "sept")
	case 5:
		return eq(s, "march") || eq(s, "april")
	case 6:
		return eq(s, "august")
	case 7:
		return eq(s, "january") || eq(s, "october")
	case 8:
		return eq(s, "february") || eq(s, "november") || eq(s, "december")
	case 9:
		return eq(s, "september")
	default:
		return false
	}
}

func wordLabel(s string) bool {
	if s == "" || !isLetter(s[0]) {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' || c == '-' || c == '/' {
			return false
		}
	}
	return true
}

func eq(s, lower string) bool {
	for len(s) > 0 && (s[len(s)-1] == '.' || s[len(s)-1] == ':') {
		s = s[:len(s)-1]
	}
	if len(s) != len(lower) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		if c != lower[i] {
			return false
		}
	}
	return true
}

func filenameProv(start, end int) Provenance {
	return Provenance{Origin: OriginFilename, Start: start, End: end, Labeled: true, Run: -1}
}

func textProv(src identity.Source, index, start, end int, labeled bool) Provenance {
	return Provenance{
		Origin:   OriginText,
		Page:     src.Page,
		Rotation: src.Rotation,
		Sheet:    src.Sheet,
		Cell:     src.Cell,
		Merge:    src.Merge,
		Table:    src.Table,
		Row:      src.Row,
		Col:      src.Col,
		Start:    start,
		End:      end,
		Labeled:  labeled,
		Heading:  src.Heading,
		Run:      index,
	}
}

func fileStem(name string) (stem string, offset int) {
	base := filepath.Base(name)
	offset = strings.LastIndex(name, base)
	if offset < 0 {
		return "", 0
	}
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".pdf", ".docx", ".xlsx", ".xlsm":
		base = base[:len(base)-len(ext)]
	}
	return base, offset
}

func titleFromGaps(stem string, covered []span) (string, int, int, bool) {
	merged := mergeSpans(covered)
	type piece struct {
		text       string
		start, end int
	}
	var pieces []piece
	cursor := 0
	for _, s := range merged {
		if s.start > cursor {
			pieces = append(pieces, piece{stem[cursor:s.start], cursor, s.start})
		}
		if s.end > cursor {
			cursor = s.end
		}
	}
	if cursor < len(stem) {
		pieces = append(pieces, piece{stem[cursor:], cursor, len(stem)})
	}
	bestText := ""
	bestStart, bestEnd := 0, 0
	for _, p := range pieces {
		trimmed, rel := trimSep(p.text)
		if len(trimmed) < len(bestText) || !looksLikeTitle(trimmed) {
			continue
		}
		bestText = trimmed
		bestStart = p.start + rel
		bestEnd = bestStart + len(trimmed)
	}
	if bestText == "" {
		return "", 0, 0, false
	}
	return bestText, bestStart, bestEnd, true
}

func mergeSpans(covered []span) []span {
	if len(covered) == 0 {
		return nil
	}
	sorted := append([]span(nil), covered...)
	for i := 1; i < len(sorted); i++ {
		j := i
		for j > 0 && sorted[j-1].start > sorted[j].start {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
			j--
		}
	}
	merged := []span{sorted[0]}
	for _, s := range sorted[1:] {
		last := &merged[len(merged)-1]
		if s.start > last.end {
			merged = append(merged, s)
			continue
		}
		if s.end > last.end {
			last.end = s.end
		}
	}
	return merged
}

func trimSep(s string) (string, int) {
	start := 0
	end := len(s)
	for start < end && isSep(s[start]) {
		start++
	}
	for end > start && isSep(s[end-1]) {
		end--
	}
	return s[start:end], start
}

func isSep(b byte) bool {
	switch b {
	case ' ', '\t', '_', '-', ':', '.':
		return true
	default:
		return false
	}
}

func fullMatch(re *regexp.Regexp, s string) bool {
	loc := re.FindStringIndex(s)
	return loc != nil && loc[0] == 0 && loc[1] == len(s)
}

func overlaps(spans []span, start, end int) bool {
	for _, s := range spans {
		if start < s.end && end > s.start {
			return true
		}
	}
	return false
}

var reportTypeRevision = regexp.MustCompile(`(?i)\s+rev(?:ision)?\s*[a-z0-9]{1,4}$`)
var projectReferenceCaption = regexp.MustCompile(`(?i)^(?:project|job)\s+(?:number|no\.?|ref\.?)\s*:?$`)
var reportReferenceValue = regexp.MustCompile(`(?i)^(?:(?:report|our|document)\s+(?:reference|ref\.?)|reference|ref\.?)\s*:\s*([a-z0-9][a-z0-9 ./_-]{1,60})$`)
var reportReferenceCaption = regexp.MustCompile(`(?i)^(?:(?:report|our|document)\s+(?:reference|ref\.?)|reference|ref\.?)\s*:$`)
var reportReferenceCell = regexp.MustCompile(`(?i)^[a-z0-9][a-z0-9 ./_-]{1,60}$`)
var documentNameCode = regexp.MustCompile(`(?i)^document name\s*\|\s*([a-z0-9][a-z0-9./_-]{1,60})$`)

// An explicit report-reference label permits identifiers with embedded spaces
// and date-like suffixes. Unlabelled project codes do not receive this rule.
func appendExplicitReportReferences(out []Candidate, text identity.Text) []Candidate {
	for i, run := range text.Runs {
		loc := reportReferenceValue.FindStringSubmatchIndex(strings.TrimSpace(run.Text))
		if loc == nil {
			loc = documentNameCode.FindStringSubmatchIndex(strings.TrimSpace(run.Text))
		}
		value := ""
		if loc != nil {
			value = strings.TrimSpace(strings.TrimSpace(run.Text)[loc[2]:loc[3]])
		} else if reportReferenceCaption.MatchString(strings.TrimSpace(run.Text)) && run.Source.Height > 0 {
			// A separate reference cell must be the only nearby value to the
			// right of its caption. Do not infer identifiers from project labels.
			found := -1
			for j, candidate := range text.Runs {
				h, s := run.Source, candidate.Source
				if s.Page != h.Page || s.Rotation != h.Rotation || s.X < h.X+h.Width || s.X-h.X > h.Height*20 || math.Abs(s.Y-h.Y) > h.Height*.5 || !reportReferenceCell.MatchString(strings.TrimSpace(candidate.Text)) {
					continue
				}
				if found >= 0 {
					found = -1
					break
				}
				found = j
			}
			if found >= 0 {
				i, run = found, text.Runs[found]
				value = strings.TrimSpace(run.Text)
			}
		}
		if value == "" {
			continue
		}
		letter, digit := false, false
		for _, r := range value {
			letter = letter || unicode.IsLetter(r)
			digit = digit || unicode.IsDigit(r)
		}
		if !letter || !digit {
			continue
		}
		start := strings.Index(run.Text, value)
		if c, ok := newCandidate(FieldNumber, value, textProv(run.Source, i, start, start+len(value), true)); ok {
			out = mergeCandidate(out, c)
		}
	}
	return out
}
