package profile_test

import (
	"testing"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
)

// A facade is on every building; only a passage naming a designed facade
// system asks Jev how complex it is, inside the label fan-out already sent.
func TestFacadeTypeAskedOnlyWhereAFacadeSystemIsNamed(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	ask := func(text string) bool {
		qs, _ := profile.LabelQuestions(profile.PassageInfo{Kind: "specification", Ordinal: 50, Text: text}, profile.Harvest(text, cat), cat)
		q, ok := qs["det.facade_type"]
		crit, _ := q.Criteria.(map[string]string)
		if ok && (crit["complex"] == "" || crit["standard"] == "" || crit["not_stated"] == "") {
			t.Fatalf("criteria %+v", q.Criteria)
		}
		return ok
	}
	if !ask("The tower facade is a unitised curtain wall system, designed and tested by the facade contractor.") {
		t.Fatal("curtain wall passage not asked")
	}
	if ask("External walls are face brick with painted render to the parapet.") {
		t.Fatal("ordinary wall passage asked about facade complexity")
	}
}
