package intake

import (
	"math"
	"regexp"
	"sitewise/internal/identity"
	"strings"
)

var ocrNumberPrefix = regexp.MustCompile(`(?i)^\s*(?:dw[ge]\.?\s*no\.?|drawing\s+(?:no\.?|number)|sheet\s+(?:no\.?|number))\s*:?\s*([a-z]{1,4}-)\s*(\d{2,4})?\s*$`)
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
		j, end := i, m[3]
		suffix := ""
		if m[4] >= 0 {
			// OCR may keep the hyphen gap inside one run instead of
			// splitting it. Both forms carry the same labelled identity.
			suffix, end = r.Text[m[4]:m[5]], m[5]
		} else {
			h := layoutSource(r.Source)
			matches := []int{}
			for k, v := range text.Runs {
				if k != i && ocrNumberSuffix.MatchString(strings.TrimSpace(v.Text)) && ocrAdjacent(h, layoutSource(v.Source), 2) {
					matches = append(matches, k)
				}
			}
			if len(matches) != 1 {
				continue
			}
			j = matches[0]
			suffix = strings.TrimSpace(text.Runs[j].Text)
		}
		// OCR can read the caption's G as E. Accept that caption only when
		// the literal identifier independently agrees with the filename.
		// Never repair a character in the identifier itself.
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Text)), "dwe") {
			numbers := map[string]bool{}
			for _, c := range out {
				if c.Field == FieldNumber && c.Provenance.Origin == OriginFilename {
					numbers[c.Normalized] = true
				}
			}
			if len(numbers) != 1 || !numbers[normalizeNumber(r.Text[m[2]:m[3]]+suffix)] {
				continue
			}
		}
		prov := textProv(r.Source, i, m[2], end, true)
		if j != i {
			prov.JoinedRuns = []int{i, j}
		}
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
	return trimOCRFilenameRevision(out, text)
}

// A filename may append the issue letter to the title. Remove it only when
// a labelled printed title and one explicit revision on its page prove the
// exact split. This does not repair OCR characters or choose a different title.
func trimOCRFilenameRevision(out []Candidate, text identity.Text) []Candidate {
	for i, file := range out {
		if file.Field != FieldTitle || file.Provenance.Origin != OriginFilename {
			continue
		}
		for _, title := range out {
			if title.Field != FieldTitle || title.Provenance.Origin != OriginText || !title.Provenance.Labeled {
				continue
			}
			revisions := map[string]bool{}
			for _, run := range text.Runs {
				if run.Source.Page != title.Provenance.Page {
					continue
				}
				if m := ocrInlineRevision.FindStringSubmatch(run.Text); m != nil {
					revisions[strings.ToLower(m[1])] = true
				}
			}
			if len(revisions) != 1 {
				continue
			}
			for revision := range revisions {
				suffix := " " + revision
				if file.Normalized != title.Normalized+suffix {
					continue
				}
				display := strings.TrimSpace(file.Display[:len(file.Display)-len(revision)])
				prov := file.Provenance
				prov.End = prov.Start + len(display)
				if c, ok := newCandidate(FieldTitle, display, prov); ok {
					out[i] = c
				}
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
	// PDF-point conversion can put an exact boundary a few ulps outside it.
	return gap >= 0 && gap <= a.Height*gapHeights+1e-6 && math.Abs((a.Y+a.Height/2)-(b.Y+b.Height/2)) < math.Min(a.Height, b.Height)*.5
}
