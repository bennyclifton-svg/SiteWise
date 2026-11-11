package rfp

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

func termKeys(rows []Row) []string {
	keys := []string{}
	for _, r := range rows {
		keys = append(keys, r.Key)
	}
	return keys
}

func termRow(t *testing.T, rows []Row, key string) Row {
	t.Helper()
	for _, r := range rows {
		if r.Key == key {
			return r
		}
	}
	t.Fatalf("no term %s in %v", key, termKeys(rows))
	return Row{}
}

// Terms rank by importance (owner, 10 Nov 2026): the agreement, the terms
// that control fees and variations, insurance, payment and rates, then the
// tender's own terms (validity, addenda) last. PI and PL print in a term.
func TestTermsRankByImportance(t *testing.T) {
	r := Resolve(discipline(t, "architecture"), skeleton(t), facts(t, "work_type=new", "class=2"), Settings{})
	want := []string{"agreement", "authority", "baseline", "errors", "insurance", "payment", "rates", "validity", "questions"}
	if got := termKeys(r.Terms); !slices.Equal(got, want) {
		t.Fatalf("term order = %v, want %v", got, want)
	}
	ins := termRow(t, r.Terms, "insurance")
	if !strings.Contains(ins.Template, "{field:pi}") || !strings.Contains(ins.Template, "{field:pl}") {
		t.Fatalf("insurance template holds no PI and PL: %q", ins.Template)
	}
	if strings.Count(ins.Text, "$20 million") != 2 || strings.Contains(ins.Text, "{") {
		t.Fatalf("insurance text = %q", ins.Text)
	}
}

// A discipline's own terms are appointment terms: they print before the
// tender's closing terms, not after them.
func TestDisciplineTermsPrintBeforeClosingTerms(t *testing.T) {
	r := Resolve(discipline(t, "electrical-services"), skeleton(t), facts(t, "work_type=new", "class=5"), Settings{})
	keys := termKeys(r.Terms)
	at := func(k string) int { return slices.Index(keys, k) }
	if at("rights") < 0 || at("risk") < 0 || !(at("rates") < at("rights") && at("risk") < at("validity") && at("validity") < at("questions")) {
		t.Fatalf("electrical term order = %v", keys)
	}
}

// Every required term field sits inside a term, so the terms table edits it
// where it reads, for every consultant RFP and trade RFT.
func TestEveryRequiredTermFieldPrintsInATerm(t *testing.T) {
	cat := loadCatalog(t)
	for _, id := range cat.RFPDisciplines() {
		d, _ := cat.RFP(id)
		sk, ok := cat.RFPSkeletonFor(d)
		if !ok {
			t.Fatalf("%s: no skeleton", id)
		}
		r := Resolve(d, sk, facts(t, "work_type=new", "class=2"), Settings{})
		for _, f := range sk.Fields {
			if f.Group != "terms" || !f.Required {
				continue
			}
			found := false
			for _, row := range r.Terms {
				found = found || strings.Contains(row.Template, "{field:"+f.Key+"}")
			}
			if !found {
				t.Errorf("%s: %s prints in no term", id, f.Key)
			}
		}
	}
}

// A term is reworded, deleted or added in the table, as a scope line is.
// A reworded term keeps the fields it held.
func TestTermsRewordDeleteAndAdd(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	drivers := facts(t, "work_type=new", "class=2")
	settings := set(
		"wording:terms.validity", `"Your proposal stays open for {field:validity_days} days from closing."`,
		"term:questions", `"D"`,
		"custom:terms", `[{"id":"u1","text":"No mobilisation payment.","value":"I"}]`,
	)
	for key, s := range settings {
		if err := ValidateSetting(d, sk, drivers, key, s.Value); err != nil {
			t.Fatalf("%s refused: %v", key, err)
		}
	}
	r := Resolve(d, sk, drivers, settings)
	v := termRow(t, r.Terms, "validity")
	if v.Text != "Your proposal stays open for 90 days from closing." || !v.Edited || v.Template != "Your proposal stays open for {field:validity_days} days from closing." {
		t.Fatalf("reworded validity = %+v", v)
	}
	if slices.Contains(termKeys(r.Terms), "questions") {
		t.Fatalf("deleted term still prints: %v", termKeys(r.Terms))
	}
	own := r.Terms[len(r.Terms)-1]
	if own.Key != "u1" || !own.Custom || own.Text != "No mobilisation payment." {
		t.Fatalf("own term = %+v", own)
	}
	text := sectionText(t, r, "terms")
	if !strings.Contains(text, "- No mobilisation payment.") || strings.Contains(text, "numbered addenda") {
		t.Fatalf("terms text = %q", text)
	}
}

func TestTermSettingsRefused(t *testing.T) {
	d, sk := discipline(t, "architecture"), skeleton(t)
	drivers := facts(t, "work_type=new", "class=2")
	for key, raw := range map[string]string{
		"wording:terms.nope":       `"A term"`,
		"term:nope":                `"D"`,
		"term:validity":            `"X"`,
		"wording:terms.validity":   `"Valid for {field:payment_days} days."`, // a field the term does not hold
		"wording:terms.authority":  `"Only {client} instructs."`,
		"wording:terms.errors":     `"Unbalanced { brace"`,
		"wording:terms.baseline":   `""`,
		"custom:terms":             `[{"id":"u1","text":"Valid for {field:validity_days} days.","value":"I"}]`,
		"custom:nope":              `[{"id":"u1","text":"A term","value":"I"}]`,
	} {
		if ValidateSetting(d, sk, drivers, key, json.RawMessage(raw)) == nil {
			t.Errorf("%s = %s accepted", key, raw)
		}
	}
}

// A term whose blank is optional and unset (a trade RFT's completion date)
// stays in the table to be filled, but does not print until it is.
func TestBlankOptionalTermIsHeld(t *testing.T) {
	cat := loadCatalog(t)
	d, _ := cat.RFP(knowledge.GeneralTradeRFP)
	sk, _ := cat.RFPSkeletonFor(d)
	drivers := facts(t, "work_type=new", "class=2")
	r := Resolve(d, sk, drivers, Settings{})
	if c := termRow(t, r.Terms, "completion"); !c.Held || c.Template != "Practical completion by {field:completion}." {
		t.Fatalf("blank completion term = %+v", c)
	}
	if strings.Contains(sectionText(t, r, "terms"), "Practical completion") {
		t.Fatal("a held term printed")
	}
	r = Resolve(d, sk, drivers, set("field:completion", `"2027-06-30"`))
	if c := termRow(t, r.Terms, "completion"); c.Held {
		t.Fatalf("filled completion term still held: %+v", c)
	}
	if !strings.Contains(sectionText(t, r, "terms"), "Practical completion by") {
		t.Fatal("a filled term did not print")
	}
}

// Terms print once in a combined engagement, so their edits are stored once.
func TestTermSettingsAreShared(t *testing.T) {
	sk := skeleton(t)
	for _, key := range []string{"term:questions", "wording:terms.validity", "custom:terms"} {
		if !SharedSetting(sk, key) {
			t.Errorf("%s is not shared", key)
		}
	}
	if SharedSetting(sk, "wording:brief_concept.site_inspection") || SharedSetting(sk, "custom:brief_concept") {
		t.Error("a scope line's wording is shared")
	}
}

// sectionText is a section's printed text, as the PDF numbers it.
func sectionText(t *testing.T, r Resolved, id string) string {
	t.Helper()
	tpl := consultantTemplate(t)
	if r.Kind == "works" {
		var ok bool
		if tpl, ok = loadCatalog(t).CurrentReportTemplate("tpl.rft-trade"); !ok {
			t.Fatal("tpl.rft-trade missing")
		}
	}
	sections, err := Sections(r, tpl)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sections {
		if s.ID == id && len(s.Blocks) > 0 {
			return s.Blocks[0].Text
		}
	}
	t.Fatalf("no %s section", id)
	return ""
}
