// Package sample writes synthetic fee proposals as small text PDFs, and a
// stand-in for System One that reads them, for the browser suite and the
// latency bench. No real firm or fee is here; the answer key of real
// proposals stays private.
package sample

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sitewise/internal/invoices"
	"sitewise/internal/jev"
)

// Line is one priced row of a synthetic proposal: the label printed and its
// fee in cents ex GST. Cents < 0 prints "Not priced".
type Line struct {
	Label string
	Cents int64
}

// Proposal is one synthetic fee proposal.
type Proposal struct {
	Firm  string
	ABN   string // eleven digits, checksum-valid
	Email string
	Lines []Line
	Rates []Line // per hour
	Extra *Line  // an item offered unasked
}

// PDF lays the proposal out as consultants do: a letterhead, a fee table
// with the amounts in a right-hand column, a total and an hourly rates list.
func PDF(p Proposal) []byte {
	var c strings.Builder
	text := func(x, y, size int, s string) {
		fmt.Fprintf(&c, "BT\n/F1 %d Tf\n1 0 0 1 %d %d Tm\n(%s) Tj\nET\n", size, x, y, escape(s))
	}
	text(40, 800, 14, p.Firm)
	text(40, 786, 9, "ABN "+invoices.DisplayABN(p.ABN)+"   "+p.Email)
	text(40, 760, 16, "FEE PROPOSAL")
	text(40, 740, 10, "Our fee proposal for the services requested. Fees exclude GST.")
	text(40, 716, 11, "1. FEES")
	y := 698
	var total int64
	for _, l := range p.Lines {
		text(40, y, 10, l.Label)
		if l.Cents < 0 {
			text(430, y, 10, "Not priced")
		} else {
			text(430, y, 10, "$"+invoices.DisplayCents(l.Cents))
			total += l.Cents
		}
		y -= 16
	}
	if p.Extra != nil {
		text(40, y, 10, p.Extra.Label)
		text(430, y, 10, "$"+invoices.DisplayCents(p.Extra.Cents))
		y -= 16
	}
	text(40, y, 10, "Total (excl. GST)")
	text(430, y, 10, "$"+invoices.DisplayCents(total))
	y -= 30
	text(40, y, 11, "2. HOURLY RATES")
	y -= 18
	for _, r := range p.Rates {
		text(40, y, 10, r.Label)
		text(430, y, 10, "$"+invoices.DisplayCents(r.Cents)+" per hour")
		y -= 16
	}
	y -= 20
	text(40, y, 11, "3. EXCLUSIONS AND ASSUMPTIONS")
	text(40, y-16, 10, "Authority fees are excluded.")
	text(40, y-50, 10, "Yours sincerely")
	text(40, y-66, 10, "Alex Example")
	return writePDF(c.String())
}

func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, "(", `\(`, ")", `\)`).Replace(s)
}

func writePDF(content string) []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>",
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, body := range objects {
		offsets[i+1] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objects)+1)
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&b, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return []byte(b.String())
}

// Jev answers proposal questions the way a careful reader of these samples
// would: a fee row takes the amount printed beside its own label, a rate
// the amount beside its role; the firm is the first candidate. Delay is
// the provider latency it stands in for.
type Jev struct{ Delay time.Duration }

func (j Jev) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	if j.Delay > 0 {
		select {
		case <-time.After(j.Delay):
		case <-ctx.Done():
			return jev.Result{}, ctx.Err()
		}
	}
	res := jev.Result{Answers: map[string]jev.Answer{}, QuestionVersion: call.QuestionVersion}
	conf := 0.99
	for field, q := range call.Questions {
		choice := pick(field, q)
		if choice == "" {
			continue
		}
		res.Answers[field] = jev.Answer{Type: jev.TypeChoice, Choice: choice, Confidence: &conf}
	}
	return res, nil
}

func pick(field string, q jev.Question) string {
	raw, _ := json.Marshal(q.Criteria)
	var criteria map[string]json.RawMessage
	_ = json.Unmarshal(raw, &criteria)
	switch {
	case field == "kind":
		return "fee_proposal"
	case field == "gst_basis":
		return "ex"
	case strings.HasPrefix(field, "firm_"):
		if _, ok := criteria["c1"]; ok {
			return "c1"
		}
		return "none"
	case strings.HasPrefix(field, "criterion:"):
		return "none"
	case strings.HasPrefix(field, "price:"):
		ins, _ := json.Marshal(q.Instructions)
		var rq struct{ Question string }
		_ = json.Unmarshal(ins, &rq)
		label := rq.Question
		for _, lead := range []string{"fee schedule: ", "role it lists: "} {
			if i := strings.Index(label, lead); i >= 0 {
				label = label[i+len(lead):]
			}
		}
		label = strings.TrimSuffix(label, "?")
		if i := strings.Index(label, " ("); i >= 0 {
			label = label[:i]
		}
		label = strings.ToLower(strings.TrimSpace(label))
		for id, c := range criteria {
			var d struct {
				Printed string `json:"printed_with"`
			}
			if json.Unmarshal(c, &d) == nil && d.Printed != "" && strings.HasPrefix(strings.ToLower(d.Printed), label) {
				return id
			}
		}
		return "not_priced"
	}
	return ""
}
