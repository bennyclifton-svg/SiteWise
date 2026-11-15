package agent

import (
	"strings"
	"testing"
)

func guardState() *State {
	st := &State{Passages: map[string]Passage{}, Facts: map[string]Fact{}}
	st.Passages["p1"] = Passage{ID: "p1", Document: "Arborist report", DocID: "d-1", Page: 3,
		Text: "Tree 4 (Eucalyptus saligna) is to be retained   and protected.\nA tree protection zone of 6.2 m applies."}
	st.Passages["p2"] = Passage{ID: "p2", Document: "Mechanical specification rev A", DocID: "d-2", Page: 12,
		Text: "The BMS shall be BACnet compatible.", Note: "superseded by rev B"}
	st.Facts["f1"] = Fact{Label: "Mechanical services, budget", Value: "$120,000", Source: "Cost Plan"}
	st.Facts["f2"] = Fact{Label: "120,000 less 45,000", Value: "$75,000", Source: "calc"}
	st.Seen.Add("What is left on mechanical? I think about 3 claims so far")
	st.Seen.Add("Mechanical services | budget 120000 | claimed 45000")
	return st
}

func marksOf(a Answer, kind string) []string {
	var out []string
	for _, m := range a.Marks {
		if m.Kind == kind {
			out = append(out, a.Text[m.Start:m.End])
		}
	}
	return out
}

func TestFigureTagsBecomeGoValues(t *testing.T) {
	a := Check("The budget is {f1} and {f2} is left. Unknown tag {f9}.", guardState())
	if !strings.Contains(a.Text, "The budget is $120,000 and $75,000 is left.") {
		t.Fatalf("text %q", a.Text)
	}
	facts := marksOf(a, "fact")
	if len(facts) != 2 || facts[0] != "$120,000" || a.Marks[0].Note != "Mechanical services, budget (Cost Plan)" {
		t.Fatalf("facts %v marks %+v", facts, a.Marks)
	}
	// A tag that names no fact is shown as needing a check, never as a value.
	if got := marksOf(a, "check"); len(got) != 1 || got[0] != "{f9}" || a.Unchecked != 1 {
		t.Fatalf("check %v unchecked %d", got, a.Unchecked)
	}
}

func TestTrueQuotationLinksAndAlteredOneIsMarked(t *testing.T) {
	st := guardState()
	a := Check(`The report says "Tree 4 (Eucalyptus saligna) is to be retained and protected." [p1]`, st)
	q := marksOf(a, "quote")
	if len(q) != 1 || a.Unchecked != 0 || len(a.Cited) != 1 || a.Cited[0] != "Arborist report" {
		t.Fatalf("quotes %v unchecked %d cited %v", q, a.Unchecked, a.Cited)
	}
	if m := a.Marks[0]; m.DocID != "d-1" || m.Page != 3 || !strings.Contains(m.Note, "Arborist report, page 3") {
		t.Fatalf("mark %+v", m)
	}
	if strings.Contains(a.Text, "[p1]") {
		t.Fatalf("the tag should be gone: %q", a.Text)
	}
	altered := Check(`The report says "Tree 4 is to be removed." [p1]`, st)
	if len(marksOf(altered, "quote")) != 0 || len(marksOf(altered, "check")) != 1 || altered.Unchecked != 1 || len(altered.Cited) != 0 {
		t.Fatalf("an altered quotation must be marked: %+v", altered)
	}
	unknown := Check(`It says "anything" [p7]`, st)
	if altered.Unchecked != 1 || unknown.Unchecked != 1 {
		t.Fatalf("a quotation from a passage nobody read must be marked: %+v", unknown)
	}
}

func TestQuotationUnderTheWrongTagLinksToWhereTheWordsAre(t *testing.T) {
	a := Check(`It says "The BMS shall be BACnet compatible." [p1]`, guardState())
	if a.Unchecked != 0 || len(a.Cited) != 1 || a.Cited[0] != "Mechanical specification rev A" || a.Marks[0].Page != 12 {
		t.Fatalf("the words are in p2, so the link must go there: %+v", a)
	}
}

func TestFigureInsideACheckedQuotationIsNotMarked(t *testing.T) {
	a := Check(`"A tree protection zone of 6.2 m applies." [p1] So keep 6.5 m clear.`, guardState())
	if got := marksOf(a, "check"); len(got) != 1 || got[0] != "6.5" {
		t.Fatalf("only the figure outside the quotation is the model's own: %v", got)
	}
}

func TestFiguresOutsideTagsAreMarked(t *testing.T) {
	a := Check("Allow about 10% contingency, say $150,000, by 30 June 2027 under clause 12.3 and AS 4000. Two firms tendered.", guardState())
	got := strings.Join(marksOf(a, "check"), "|")
	for _, want := range []string{"10%", "$150,000", "30 June 2027", "12.3", "AS 4000", "Two"} {
		if !strings.Contains(got, want) {
			t.Errorf("%q not marked in %q", want, got)
		}
	}
	if a.Unchecked < 6 {
		t.Errorf("unchecked %d", a.Unchecked)
	}
}

func TestFigureSeenThisTurnIsNotTheModelsOwn(t *testing.T) {
	a := Check("You said 3 claims. The budget line reads 120000.", guardState())
	if a.Unchecked != 0 || len(marksOf(a, "seen")) != 2 {
		t.Fatalf("figures the PM typed or a tool returned are seen, not unchecked: %+v", a)
	}
	// A list number is not a figure.
	list := Check("Flags:\n1. Hazardous materials\n2. Occupied building", guardState())
	if len(list.Marks) != 0 {
		t.Fatalf("list numbers marked: %+v", list.Marks)
	}
}

func TestTypedFigureThatIsOneFactIsTiedToItsRow(t *testing.T) {
	st := guardState()
	st.Facts["f1"] = Fact{Label: "Mechanical services, budget", Value: "$120,000", Source: "Cost Plan", Plain: "120000"}
	a := Check("The budget is $120,000 and 45000 is claimed.", st)
	facts := marksOf(a, "fact")
	if len(facts) != 1 || facts[0] != "$120,000" || a.Unchecked != 0 || a.Marks[0].Note != "Mechanical services, budget (Cost Plan)" {
		t.Fatalf("marks %+v", a.Marks)
	}
	// Two rows with the same figure are not guessed between.
	st.Facts["f3"] = Fact{Label: "Another line, budget", Value: "$120,000", Source: "Cost Plan", Plain: "120000"}
	if b := Check("The budget is $120,000.", st); len(marksOf(b, "fact")) != 0 {
		t.Fatalf("an ambiguous figure was tied to a row: %+v", b.Marks)
	}
}

func TestSupersededRevisionIsNamed(t *testing.T) {
	a := Check(`"The BMS shall be BACnet compatible." [p2]`, guardState())
	if len(a.Marks) != 1 || !strings.Contains(a.Marks[0].Note, "Mechanical specification rev A, page 12") || !strings.Contains(a.Marks[0].Note, "superseded by rev B") {
		t.Fatalf("mark %+v", a.Marks)
	}
}

func TestVerdictWordsAreMarkedAndCoverageIsCarried(t *testing.T) {
	st := guardState()
	st.Coverage = "Searched 7 documents for tree, arborist; 1 has no text."
	a := Check("This project is low risk and the carpet complies with the specification.", st)
	if a.Verdicts != 2 || len(marksOf(a, "verdict")) != 2 || a.Coverage != st.Coverage {
		t.Fatalf("answer %+v", a)
	}
	// Declining to judge is not a judgment.
	if b := Check("I can't call this project low risk. SiteWise doesn't judge whether it complies with the specification.", st); b.Verdicts != 0 {
		t.Fatalf("a declined verdict was marked: %+v", b.Marks)
	}
}
