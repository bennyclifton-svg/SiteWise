package profile_test

import (
	"strings"
	"testing"

	"sitewise/internal/profile"
)

func TestLabelQuestionsAddTriggeredDeterminantsOnly(t *testing.T) {
	cat := repoCatalog(t)
	text := "Thermally Broken windows/doors, with BAL 40 compliance"
	qs, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "commercial", Ordinal: 40}, profile.Harvest(text, cat), cat)
	if _, ok := qs["det.bal"]; !ok {
		t.Fatal("det.bal missing")
	}
	if _, ok := qs["det.bal.assertion"]; !ok {
		t.Fatal("assertion missing")
	}
	for id := range qs {
		if strings.HasPrefix(id, "det.") && !strings.HasPrefix(id, "det.bal") {
			t.Fatalf("untriggered %s", id)
		}
	}
	crit := qs["det.bal"].Criteria.(map[string]string)
	if _, ok := crit["not_stated"]; !ok || crit["BAL-40"] == "" {
		t.Fatalf("criteria %v", crit)
	}
}

func TestPreParsedOptionsAreCandidatesPlusNone(t *testing.T) {
	cat := repoCatalog(t)
	text := "The size of the fire compartments must remain under 2,000sqm so that Type C construction can be achieved."
	qs, cands := profile.LabelQuestions(profile.PassageInfo{Kind: "design_brief", Ordinal: 30}, profile.Harvest(text, cat), cat)
	crit := qs["det.floor_area"].Criteria.(map[string]string)
	if len(crit) != 2 || crit["none"] == "" || !strings.HasPrefix(crit["c1"], "2,000sqm (in: '") {
		t.Fatalf("criteria %v", crit)
	}
	if cands["det.floor_area"][0].Value != "2,000sqm" {
		t.Fatalf("state candidates %v", cands)
	}
}

func TestClassIsAskedPerMentionedOption(t *testing.T) {
	cat := repoCatalog(t)
	qs, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "report"}, profile.Harvest("The building is classified as Class 7b and Class 5.", cat), cat)
	for _, id := range []string{"det.ncc_class.7b", "det.ncc_class.5", "det.ncc_class.assertion"} {
		if _, ok := qs[id]; !ok {
			t.Fatalf("%s missing from %v", id, keys(qs))
		}
	}
	if _, ok := qs["det.ncc_class"]; ok {
		t.Fatal("a choice cannot return a set; ask per option")
	}
}

func TestHeaderQuestionsOnlyOnRoutedPassages(t *testing.T) {
	cat := repoCatalog(t)
	h := profile.Harvest("The extension works comprises a single level warehouse tenancy.", cat)
	routed, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "design_brief", Ordinal: 3}, h, cat)
	if _, ok := routed["hdr.subclass"]; !ok {
		t.Fatal("brief passage 3 must get header questions")
	}
	late, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "design_brief", Ordinal: 40, Section: "9.0 Fire Services"}, h, cat)
	drawing, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "drawing", Ordinal: 0}, h, cat)
	quote, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "commercial", Ordinal: 50, Section: "Project description"}, h, cat)
	if _, ok := late["hdr.subclass"]; ok {
		t.Fatal("late fire section must not get header questions")
	}
	if _, ok := drawing["hdr.subclass"]; ok {
		t.Fatal("drawings never get header questions")
	}
	if _, ok := quote["hdr.work_type"]; !ok {
		t.Fatal("a quote's description section is header-routed")
	}
}

func TestEvidenceQuestionsForLabelledLeavesOnly(t *testing.T) {
	cat := repoCatalog(t)
	qs := profile.EvidenceQuestions([]string{"hydraulic", "hydraulic.gas", "fire-passive.bushfire-construction"}, cat)
	if len(qs) != 2 || qs["sys.hydraulic.gas.presence"].Type != "choice" || qs["sys.hydraulic.gas.provider"].Type != "choice" {
		t.Fatalf("questions %v", keys(qs))
	}
}

func TestWordingMatchesContract(t *testing.T) {
	cat := repoCatalog(t)
	qs := profile.EvidenceQuestions([]string{"hydraulic.gas"}, cat)
	p := qs["sys.hydraulic.gas.presence"]
	if !strings.Contains(p.Instructions.(string), "any of its components") {
		t.Fatalf("instructions %q", p.Instructions)
	}
	crit := p.Criteria.(map[string]string)
	if !strings.Contains(crit["not_included"], "Broader parent categories") {
		t.Fatalf("criteria %q", crit["not_included"])
	}
	lq, _ := profile.LabelQuestions(profile.PassageInfo{}, profile.Harvest("Bush Fire Attack Level Low", cat), cat)
	a := lq["det.bal.assertion"].Criteria.(map[string]string)
	if a["allowance"] != "It treats the value as an assumption or a pricing allowance, for example 'we have allowed for' or 'assumed'." {
		t.Fatalf("assertion %q", a["allowance"])
	}
}

func keys[V any](m map[string]V) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}
