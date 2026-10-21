package proposals

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/rfp"
)

func TestQuotePreservesNumberedRowsAndExGSTAlternatives(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1,
		[]string{"Borehole 1", "$400 plus GST"},
		[]string{"Borehole 2", "$400 plus GST"},
		[]string{"Option A total", "$10,000 plus GST"},
		[]string{"Option B total", "$11,000 plus GST"},
		[]string{"Fee includes the reporting", "$800 plus GST"},
	)))
	q := NewDraft(ev, Context{}, nil).Finish().Quote
	if len(q.Fees) != 5 {
		t.Fatalf("distinct source fees lost: %+v", q.Fees)
	}
	for _, f := range q.Fees {
		if strings.Contains(f.Label, "includes the") && strings.Contains(f.Role, "Alternative") {
			t.Fatal("ordinary inclusion became alternative")
		}
	}
}

func TestCoverageQuestionsOnlyReferenceTransmittedFees(t *testing.T) {
	var rows [][]string
	for i := 0; i < 80; i++ {
		rows = append(rows, []string{fmt.Sprintf("Service %d", i), "2", "hour", "$200", "$400"})
	}
	rows = append(rows, []string{"Total fee excluding GST", "$32,000"})
	ev := Harvest("architecture.pdf", text(page(1, rows...)))
	d := NewDraft(ev, Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": {{Key: "stage:design", Kind: rfp.ItemStage, Label: "Design", Base: true}}}}, nil)
	call, ok := d.Call()
	if !ok || !fits(call) {
		t.Fatal("fanout exceeds bound")
	}
	raw, _ := json.Marshal(call)
	for _, fee := range coverageAmounts(ev) {
		if !strings.Contains(string(raw), `"id":"`+fee.ID+`"`) {
			t.Fatalf("source %s missing from state", fee.ID)
		}
	}
}

func TestQuoteKeepsBundleSeparateFromCoverageAcrossDisciplines(t *testing.T) {
	for _, tc := range []struct{ discipline, first, second string }{
		{"Geotechnical", "Field investigation", "Geotechnical report"},
		{"Architectural", "Concept design", "Construction documentation"},
		{"Traffic", "Traffic impact assessment", "Car parking review"},
		{"Town planning", "Planning advice", "Statement of environmental effects"},
	} {
		t.Run(tc.discipline, func(t *testing.T) {
			ev := Harvest("Fee ("+tc.discipline+").pdf", text(page(1,
				[]string{tc.discipline + " services: " + tc.first + " and " + tc.second},
				[]string{"Total lump sum fee excluding GST", "$21,395.00"},
				[]string{"Principal", "10", "hour", "$280.00", "$2,800.00"},
				[]string{"Review", "1", "sum", "$400.00", "$400.00"},
				[]string{"Meeting", "1", "sum", "$400.00", "$400.00"},
			)))
			items := []rfp.ScheduleItem{{Key: "stage:one", Kind: rfp.ItemStage, Label: tc.first, Base: true}, {Key: "stage:two", Kind: rfp.ItemStage, Label: tc.second, Base: true}}
			d := NewDraft(ev, Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": items}}, nil)
			a, ok := find(ev, "$21,395.00")
			if !ok {
				t.Fatal("missing bundle")
			}
			d.Apply(jev.Result{Answers: map[string]jev.Answer{
				"coverage:stage:one": {Type: jev.TypeChoice, Choice: a.ID},
				"coverage:stage:two": {Type: jev.TypeChoice, Choice: a.ID},
			}}, nil, Thresholds{})
			r := d.Finish()
			for _, item := range items {
				if r.Quote.Coverage[item.Key].FeeID != a.ID {
					t.Fatal("bundle coverage lost")
				}
			}
			for _, price := range r.Prices {
				if price.Key != "" && price.Cents != nil {
					t.Fatal("uncalibrated coverage created money")
				}
			}
			equal := 0
			for _, fee := range r.Quote.Fees {
				if fee.Amount != nil && *fee.Amount == "400.00" {
					equal++
				}
			}
			if equal != 2 {
				t.Fatalf("equal-valued distinct fees lost: %d", equal)
			}
		})
	}
}

func TestQuoteTaxRatesAndLiability(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1,
		[]string{"Geotechnical fee"},
		[]string{"Sub Total", "$47,000.00"},
		[]string{"+ 10% GST", "$4,700.00"},
		[]string{"Cost Estimate", "$51,700.00"},
		[]string{"Chromium testing rate only", "$110", "rate only"},
		[]string{"Vehicle", "$250 per visit"},
		[]string{"for the services concerned (to a maximum of $3,000,000); or"},
	)))
	d := NewDraft(ev, Context{}, nil)
	q := d.Finish().Quote
	if len(q.Fees) != 1 || q.Fees[0].Amount == nil || *q.Fees[0].Amount != "47000.00" {
		t.Fatalf("tax, rate or liability became a fee: %+v", q.Fees)
	}
}

func TestQuoteUnknownTaxAndTimeoutRemainUnknown(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1, []string{"Total fee", "$5,000"})))
	d := NewDraft(ev, Context{}, nil)
	d.Apply(jev.Result{}, errors.New("timeout"), Thresholds{})
	q := d.Finish().Quote
	if len(q.Fees) != 1 || q.Fees[0].Amount != nil {
		t.Fatalf("invented tax basis: %+v", q.Fees)
	}
}

func TestQuoteNameCandidatesRemoveSurroundingProse(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Please return to Example Geotechnics", "Example Geotechnics"},
		{"Example Partners Pty Ltd  Reviewed by", "Example Partners Pty Ltd"},
		{"Unless Group Example expressly agrees otherwise", ""},
	} {
		if got := cleanFirmCandidate(tc.in); got != tc.want {
			t.Errorf("%q: %q", tc.in, got)
		}
	}
}

func TestQuoteFanoutBoundedAndSuggestionsCannotAuthorizePrices(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1, []string{"Optional investigation including the base scope $15,500+GST"})))
	d := NewDraft(ev, Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": {{Key: "stage:one", Kind: rfp.ItemStage, Label: "Investigation", Base: true}}}}, nil)
	call, ok := d.Call()
	if !ok || !fits(call) {
		t.Fatal("missing or oversized fanout")
	}
	for k := range call.Questions {
		if strings.HasPrefix(k, "coverage:") {
			return
		}
	}
	t.Fatal("coverage not in shared fanout")
}
