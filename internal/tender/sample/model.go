// Package sample is a stand-in for the model in tests, the browser tests
// and the bench. It is to the tender pass what proposals/sample.Jev is to
// the old reader: it reads the request it is given by simple rules and
// answers in the model's own JSON, so everything after the call is real.
package sample

import (
	"context"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/llm"
	"sitewise/internal/tender"
)

// Model answers a tender pass without a network.
type Model struct {
	Delay time.Duration
}

var (
	pageHead   = regexp.MustCompile(`^Text of page (\d+)(?: \(read by OCR\))?:\n`)
	withAmount = regexp.MustCompile(`^(.*\S)\s+(-?\$\s?[\d,]+(?:\.\d\d)?)$`)
	onlyAmount = regexp.MustCompile(`^-?\$\s?[\d,]+(?:\.\d\d)?$`)
	rowsHead   = "The schedule rows to map each line to:\n"
)

// Ask lists a line for each line of page text that ends in a dollar amount.
// A label that starts "Total" is the total. "Option:" is an option outside
// the price, "Saving:" a saving, "Extra:" a provisional sum outside the
// price, and a label ending "(excluded)" an excluded line with no amount.
// A line's row is the schedule row whose label starts with the same word.
func (m Model) Ask(ctx context.Context, req llm.Request) (llm.Response, error) {
	began := time.Now()
	if m.Delay > 0 {
		select {
		case <-time.After(m.Delay):
		case <-ctx.Done():
			return llm.Response{}, ctx.Err()
		}
	}
	var rows []tender.Row
	out := tender.RawReading{Basis: "fixed", GST: "ex", Lines: []tender.RawLine{}, Exclusions: []tender.RawExclusion{}, Notes: []string{}}
	// The schedule comes last in the request and is needed first.
	for _, part := range req.Parts {
		if i := strings.Index(part.Text, rowsHead); i >= 0 {
			body := part.Text[i+len(rowsHead):]
			if end := strings.Index(body, "\n\n"); end >= 0 {
				body = body[:end]
			}
			_ = json.Unmarshal([]byte(body), &rows)
		}
	}
	for _, part := range req.Parts {
		text := part.Text
		head := pageHead.FindStringSubmatch(text)
		if head == nil {
			continue
		}
		page, _ := strconv.Atoi(head[1])
		prev := ""
		for _, raw := range strings.Split(text[len(head[0]):], "\n") {
			line := strings.TrimSpace(raw)
			switch {
			case strings.HasSuffix(line, "(excluded)"):
				label := strings.TrimSpace(strings.TrimSuffix(line, "(excluded)"))
				out.Lines = append(out.Lines, tender.RawLine{P: page, Label: label, G: "ex", St: "excluded", Kind: "base", Where: "sum", Row: rowFor(rows, label), Note: "excluded", Sure: true})
				prev = ""
			case onlyAmount.MatchString(line) && prev != "":
				out.Lines = append(out.Lines, lineOf(page, prev, line, rows))
				prev = ""
			case withAmount.MatchString(line):
				found := withAmount.FindStringSubmatch(line)
				out.Lines = append(out.Lines, lineOf(page, found[1], found[2], rows))
				prev = ""
			default:
				prev = line
			}
		}
	}
	body, _ := json.Marshal(out)
	return llm.Response{Text: string(body), StopReason: "end_turn", InTokens: 1000, OutTokens: int64(len(body) / 4), Latency: time.Since(began)}, nil
}

func lineOf(page int, label, amount string, rows []tender.Row) tender.RawLine {
	l := tender.RawLine{P: page, Label: label, Amt: amount, G: "ex", St: "priced", Kind: "base", Where: "sum", Sure: true}
	lower := strings.ToLower(label)
	switch {
	case strings.HasPrefix(lower, "total"):
		l.Kind, l.Where = "total", "inside"
		return l
	case strings.HasPrefix(lower, "option:"):
		l.Kind, l.Where = "option", "outside"
		return l
	case strings.HasPrefix(lower, "saving:"):
		l.Kind, l.Where = "saving", "outside"
		return l
	case strings.HasPrefix(lower, "extra:"):
		l.Kind, l.Where = "ps", "outside"
		l.Row = rowFor(rows, strings.TrimSpace(label[len("extra:"):]))
		return l
	}
	l.Row = rowFor(rows, label)
	return l
}

// rowFor is the row whose label starts with the line's first word.
func rowFor(rows []tender.Row, label string) string {
	words := strings.Fields(strings.ToLower(label))
	if len(words) == 0 {
		return ""
	}
	for _, r := range rows {
		if first := strings.Fields(strings.ToLower(r.Label)); len(first) > 0 && first[0] == words[0] {
			return r.Key
		}
	}
	return ""
}
