package proposals

import (
	"strings"
	"testing"
)

func TestBuilderSectionSummaryRequiresEvidence(t *testing.T) {
	for _, tc := range []struct{ text, relation, want string }{
		{"Kitchen included", "Scope wording", "Unclear"},
		{"Interiors, joinery and finishes: included", "Scope wording", "Included"},
		{"Interiors, joinery and finishes: not included", "Exclusion wording", "Excluded"},
		{"Kitchen allowance $20,000.00", "Allowance", "Unclear"},
		{"Kitchen allowance $22,000.00 including GST", "Allowance", "Allowance $20000.00"},
		{"Kitchen $20,000.00 excluding GST", "Quoted amount", "Component $20000.00"},
		{"Passage Sets PC $45 Inc gst x 4", "Quoted amount", "Unclear"},
		{"Tiles $55 inc GST per m2", "Allowance", "Unclear"},
	} {
		s := BuilderSection{Label: "Interiors, joinery and finishes", Details: []BuilderDetail{{Text: tc.text, Relationship: tc.relation}}}
		if got := builderSectionSummary(s); got != tc.want {
			t.Fatalf("%s: %s, want %s", tc.text, got, tc.want)
		}
	}
}

func TestBuilderDetailIsNonAdditive(t *testing.T) {
	r := ReviewBuilderQuote([]Passage{{Page: 3, Body: "Allowances\nKitchen $20,000.00\nOption air conditioning $9,000.00\nSTAGE 1 SHED $14,880.00\nTotal including GST $110,000.00"}})
	want := map[string]string{"Kitchen": "Allowance; inclusion unresolved", "Option": "Option; inclusion unresolved", "STAGE": "Stage / component amounts; do not add together", "Total": "Quoted total; not an extra"}
	for prefix, relationship := range want {
		found := false
		for _, section := range r.Sections {
			for _, detail := range section.Details {
				if strings.HasPrefix(detail.Text, prefix) {
					found = true
					if detail.Relationship != relationship || detail.SectionPage != 3 {
						t.Fatalf("%s: %+v", prefix, detail)
					}
				}
			}
		}
		if !found {
			t.Fatalf("missing %s", prefix)
		}
	}
	if r.ExGST == nil || *r.ExGST != "100000.00" {
		t.Fatal("detail changed total")
	}
}

func TestBuilderQuoteTotalsAndRelationships(t *testing.T) {
	for _, tc := range []struct{ name, text, want string }{
		{"inclusive", "Kitchen joinery $45,000.00\nTotal price including gst $228,000.00", "207272.73"},
		{"summary", "Quote Summary\nSub Total: $191,136.71\nGST: $19,113.68\nIncl. GST: $210,250.39", "191136.71"},
		{"margin", "EXC GST $182,417.00\nPLUS 10% BUILDER MARGIN $200,658.70", "200658.70"},
		{"no tax", "Total $200,000.00", ""},
		{"conflict", "Total including GST $110,000.00\nTotal including GST $220,000.00", ""},
		{"stage only", "STAGE 1 SHED $14,880.00\nSTAGE 5 FIT OFF $63,692.00", ""},
		{"bad margin", "EXC GST $182,417.00\nPLUS 10% BUILDER MARGIN $200,000.00", ""},
		{"bare subtotal", "Quote Summary\nSub Total: $100,000.00", ""},
		{"empty", "", ""},
		{"parentheses", "Total (excl. GST) $200,000.00", "200000.00"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := ReviewBuilderQuote([]Passage{{Body: tc.text, Page: 1}})
			got := ""
			if r.ExGST != nil {
				got = *r.ExGST
			}
			if got != tc.want {
				t.Fatalf("got %q want %q: %+v", got, tc.want, r)
			}
		})
	}
}
