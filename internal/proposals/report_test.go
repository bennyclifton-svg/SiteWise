package proposals

import (
	"strings"
	"testing"
)

func cents64(v int64) *int64 { return &v }

func TestDraftRecommendationStatesOnlyFacts(t *testing.T) {
	text := DraftRecommendation("Town planning", []FirmFacts{
		{Name: "Urbis", KnownBase: 1300000, NotPriced: 1},
		{Name: "Mecone", BaseTotal: cents64(1550000), KnownBase: 1550000, Lowest: true, Ratings: map[string]string{"Team": "strong"}},
		{Name: "Willowtree", BaseTotal: cents64(1100000), KnownBase: 1100000, NotPriced: 2, Ratings: map[string]string{"Experience": "concern"}},
	})
	for _, want := range []string{"Lowest complete base fee: Mecone at $15,500 ex GST.", "Willowtree: $11,000 ex GST (2 items not priced).", "Urbis: $13,000 ex GST known so far (1 item not priced).", "Rated a concern: Willowtree (Experience)."} {
		if !strings.Contains(text, want) {
			t.Fatalf("draft lacks %q:\n%s", want, text)
		}
	}
	if !strings.Contains(text, "[state the firm to appoint and why]") {
		t.Fatal("the draft must leave the choice to the PM")
	}
}

func TestCheckDraftRejectsPlantedFigureAndUncitedClaim(t *testing.T) {
	passages := map[string]string{"p1": "Our team of 4 planners has delivered 12 industrial DAs.", "p2": "Exclusions: council fees."}
	figures := []int64{1550000, 1100000}
	in := GeneratedDraft{
		Summaries: []GeneratedText{
			{Firm: "Mecone", Criterion: "Team", Text: "Four-person team; 12 industrial DAs delivered.", Cites: []string{"p1"}},
			{Firm: "Mecone", Criterion: "Experience", Text: "Delivered 40 projects like this.", Cites: []string{"p1"}},
			{Firm: "Urbis", Criterion: "Team", Text: "Strong local team.", Cites: nil},
			{Firm: "Urbis", Criterion: "Program", Text: "Six weeks.", Cites: []string{"p9"}},
		},
		Departures:     []GeneratedText{{Firm: "Mecone", Text: "Council fees excluded.", Cites: []string{"p2"}}},
		Recommendation: "Mecone's $15,500 is the lowest complete fee; Willowtree's $9,000 omits two stages.",
	}
	out, problems := CheckDraft(in, passages, figures)
	if len(out.Summaries) != 1 || out.Summaries[0].Criterion != "Team" || len(out.Departures) != 1 {
		t.Fatalf("kept %+v", out)
	}
	if out.Recommendation != "" {
		t.Fatalf("a planted $9,000 must drop the recommendation: %q", out.Recommendation)
	}
	reasons := []string{}
	for _, p := range problems {
		reasons = append(reasons, p.Criterion+": "+p.Reason)
	}
	joined := strings.Join(reasons, "\n")
	for _, want := range []string{"Experience: summary states 40", "Team: summary cites nothing", "Program: summary cites p9", "recommendation: states $9,000"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("problems lack %q:\n%s", want, joined)
		}
	}
	good := GeneratedDraft{Recommendation: "Mecone at $15,500 is the lowest complete fee."}
	if out, _ := CheckDraft(good, passages, figures); out.Recommendation == "" {
		t.Fatal("a recommendation stating a computed figure must stand")
	}
}

func TestTotalsAndLowest(t *testing.T) {
	items := planning
	full := map[string]Cell{"stage:advice": {State: StatePriced, Cents: cents64(300000)}, "stage:report": {State: StatePriced, Cents: cents64(900000)}, "reimbursables": {State: StateIncluded}, "option:panel": {State: StatePriced, Cents: cents64(50000)}}
	cheap := map[string]Cell{"stage:advice": {State: StatePriced, Cents: cents64(200000)}, "stage:report": {State: StateNotPriced}, "reimbursables": {State: StateIncluded}}
	unread := map[string]Cell{"stage:advice": {State: StatePriced, Cents: cents64(100000)}}
	cols := []Column{Totals(items, full, nil, cents64(1250000)), Totals(items, cheap, []Cell{{Cents: cents64(5000)}}, nil), Totals(items, unread, nil, nil)}
	MarkLowest(cols)
	if *cols[0].BaseTotal != 1200000 || cols[0].OptionsTotal != 50000 || cols[0].Gap == nil || *cols[0].Gap != 50000 || !cols[0].Lowest {
		t.Fatalf("full %+v", cols[0])
	}
	if cols[1].Lowest || cols[1].NotPriced != 1 || cols[1].BaseTotal != nil || cols[1].KnownBase != 200000 || cols[1].OthersTotal != 5000 {
		t.Fatalf("a cheaper total with an item not priced is not lowest: %+v", cols[1])
	}
	if cols[2].BaseTotal != nil || cols[2].Unread != 2 {
		t.Fatalf("unread cells leave the total unknown: %+v", cols[2])
	}
}
