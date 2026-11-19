package invoices

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
)

// Asker is the one System One call an invoice read may make.
type Asker interface {
	Ask(context.Context, jev.Call) (jev.Result, error)
}

// Not-read reasons. The bytes are kept and the user can type the fields.
const (
	NotReadNoText      = "no_text_layer"
	NotReadUnreadable  = "unreadable_text"
	NotReadUnsupported = "unsupported_format"
	NotReadTooLarge    = "too_large"
)

// ExtractLimits reads up to six pages: an invoice's own figures are on its
// first pages, and a claim schedule or statutory declaration that follows
// must not crowd them out.
func ExtractLimits() identity.Limits {
	l := identity.DefaultLimits()
	l.MaxPages = 6
	l.ReadControlPage = false
	l.MaxRuns = 1500
	return l
}

// Outcome is one finished read. NotRead is set instead of fields when the
// file has no usable text. Grey means Jev did not answer; the caller
// re-asks the grey questions later at background priority.
type Outcome struct {
	Reading
	NotRead string
	Grey    bool
	Asked   bool
	// Claim is a progress claim's table, read as printed; nil for an
	// invoice.
	Claim *ClaimReading
}

// Read files one invoice's text: rules first, then at most one Jev fan-out
// for what is still open, then code arithmetic. Users' decisions stay.
//
// A progress claim (dropped on its contract, or "progress claim" printed
// with the supplier settled by rule to a live works contract) is read by
// ReadClaim instead, against claim, the claim-1 cut-offs.
func Read(ctx context.Context, ask Asker, th Thresholds, filename string, text identity.Text, pc Context, users map[string]Decision, uploaded time.Time, priority jev.Priority, claim ...Thresholds) (Outcome, error) {
	if !text.TextLayer {
		return Outcome{NotRead: NotReadNoText}, nil
	}
	ev := Harvest(filename, text)
	if !ev.Readable {
		return Outcome{NotRead: NotReadUnreadable}, nil
	}
	d := NewDraft(ev, pc, users)
	if c, ok := d.claimContract(); ok {
		cth := Thresholds{QuestionVersion: ClaimQuestionVersion}
		if len(claim) > 0 {
			cth = claim[0]
		}
		return readClaim(ctx, ask, cth, d, text, c, uploaded, priority)
	}
	out := Outcome{}
	if call, ok := d.Call(); ok {
		if priority != "" {
			call.Priority = priority
		}
		out.Asked = true
		res, err := ask.Ask(ctx, call)
		if err != nil && ctx.Err() != nil {
			return Outcome{}, ctx.Err()
		}
		out.Grey = d.Apply(res, err, th)
	}
	out.Reading = d.Finish(uploaded)
	return out, nil
}

// FormatOf is the text format an invoice file can be read as, or "".
func FormatOf(filename string) string {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".xlsx":
		return "xlsx"
	case ".md", ".markdown":
		return "md"
	}
	return ""
}
