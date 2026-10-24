package intake

import (
	"regexp"
	"strings"
	"time"

	"sitewise/internal/identity"
)

// Markdown has no spatial title block. Read its leading document metadata,
// not every number, date or revision mentioned in the body or attachments.
func markdownIdentity(text identity.Text) []Candidate {
	var out []Candidate
	add := func(field, value string, run int) {
		value = strings.TrimSpace(value)
		start := strings.Index(text.Runs[run].Text, value)
		if start < 0 || value == "" {
			return
		}
		if c, ok := newCandidate(field, value, textProv(text.Runs[run].Source, run, start, start+len(value), true)); ok {
			out = append(out, c)
		}
	}
	if markdownEmail(text) {
		var latest time.Time
		var date string
		dateRun := 0
		inHeaders := false
		titleSeen := false
		for i, run := range text.Runs {
			key, value, ok := markdownLabel(run.Text)
			if ok && key == "from" {
				inHeaders = markdownEmail(identity.Text{Format: text.Format, Runs: text.Runs[i:]})
				continue
			}
			if !inHeaders {
				continue
			}
			if !ok || (key != "to" && key != "cc" && key != "bcc" && key != "subject" && key != "date" && key != "sent") {
				inHeaders = false
				continue
			}
			if key == "subject" && !titleSeen {
				add(FieldTitle, value, i)
				titleSeen = true
			}
			if key == "date" || key == "sent" {
				literal := datePattern.FindString(value)
				if parsed, ok := markdownDate(literal); ok && parsed.After(latest) {
					latest, date, dateRun = parsed, literal, i
				}
			}
		}
		// A thread's filing date is its latest explicit message-header calendar
		// date. Code orders dates; body and attachment dates never participate.
		if date != "" && !text.Incomplete {
			add(FieldDate, date, dateRun)
		}
		return out
	}
	for i, run := range text.Runs {
		line := strings.TrimSpace(run.Text)
		if strings.HasPrefix(line, "##") || line == "---" || strings.HasPrefix(line, "```") || strings.HasPrefix(line, "~~~") || strings.HasPrefix(strings.ToLower(line), "dear ") {
			break
		}
		if strings.HasPrefix(line, "# ") {
			add(FieldTitle, strings.TrimSpace(line[2:]), i)
			continue
		}
		key, value, ok := markdownLabel(line)
		if !ok {
			if _, valid := markdownDate(line); valid {
				add(FieldDate, line, i)
			}
			continue
		}
		switch key {
		case "title", "document title":
			add(FieldTitle, value, i)
		case "ref", "reference", "report ref", "report no", "report number", "document number", "document no", "drawing number", "drawing no",
			// A proposal or quote names itself by its own reference.
			"proposal reference", "proposal ref", "proposal no", "proposal number", "quote no", "quote number", "quote ref", "quote reference", "quotation no", "quotation number", "quotation reference":
			if markdownReference.MatchString(value) {
				add(FieldNumber, value, i)
			}
		case "revision", "rev", "version":
			parts := strings.Fields(value)
			if len(parts) > 0 {
				if _, valid := revisionNormalized(parts[0]); valid {
					add(FieldRevision, parts[0], i)
				}
			}
		case "date", "issue date", "report date":
			if _, valid := markdownDate(value); valid {
				add(FieldDate, value, i)
			}
		}
	}
	return out
}

var markdownReference = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9./_-]{1,79}$`)

// Strip only paired emphasis around the caption/value, preserving the exact
// source value and its byte offsets for provenance.
func markdownLabel(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	key, value, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	key = strings.ToLower(strings.Trim(strings.TrimSpace(key), "*_"))
	value = strings.TrimSpace(value)
	value = strings.TrimSpace(strings.Trim(value, "*_"))
	return strings.TrimSuffix(key, "."), value, true
}

func markdownEmail(text identity.Text) bool {
	if text.Format != "md" && text.Format != "markdown" {
		return false
	}
	if len(text.Runs) < 3 {
		return false
	}
	key, _, ok := markdownLabel(text.Runs[0].Text)
	if !ok || key != "from" {
		return false
	}
	to, subject := false, false
	for _, run := range text.Runs[1:min(8, len(text.Runs))] {
		key, _, ok := markdownLabel(run.Text)
		if !ok || (key != "to" && key != "cc" && key != "bcc" && key != "subject" && key != "date" && key != "sent") {
			break
		}
		to = to || key == "to"
		subject = subject || key == "subject"
	}
	return to && subject
}

func markdownDate(value string) (time.Time, bool) {
	for _, layout := range []string{"2 January 2006", "2 Jan 2006", "2006-01-02", "2/1/2006", "2.1.2006", "2-1-2006"} {
		if date, err := time.Parse(layout, value); err == nil {
			return date, true
		}
	}
	return time.Time{}, false
}
