package intake_test

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestRequirementsCoverChallengesFilenameAtBodySize(t *testing.T) {
	// The PPR cover uses nearly the same type size for its title and details.
	text := identity.Text{Format: "pdf", TextLayer: true, Runs: []identity.Run{
		{Text: "Example site – Principal Project Requirements", Source: identity.Source{Page: 1, X: 72, Y: 38, Width: 244, Height: 9.74}},
		{Text: "(Rev E– 21.06.2021)", Source: identity.Source{Page: 1, X: 72, Y: 28, Width: 69, Height: 9.74}},
		{Text: "1", Source: identity.Source{Page: 1, X: 515, Y: 28, Width: 4, Height: 9.74}},
		{Text: "PRINCIPAL’S PROJECT REQUIREMENTS", Source: identity.Source{Page: 1, X: 71, Y: 659, Width: 180, Height: 12.16}},
		{Text: "Example building: Example Road", Source: identity.Source{Page: 1, X: 71, Y: 633, Width: 223, Height: 12.06}},
		{Text: "Principal: Example Pty Ltd", Source: identity.Source{Page: 1, X: 71, Y: 562, Width: 212, Height: 12.36}},
		{Text: "DRAFT Revision - E", Source: identity.Source{Page: 1, X: 71, Y: 497, Width: 87, Height: 12.16}},
		{Text: "21 June 2021", Source: identity.Source{Page: 1, X: 71, Y: 472, Width: 59, Height: 12.16}},
	}}
	const filename = "ANX Q PPR [E].pdf"
	candidates := intake.Harvest(filename, text)
	if _, ok := findCandidate(candidates, intake.FieldTitle, text.Runs[3].Text); !ok {
		t.Fatal("printed requirements heading missing from title candidates")
	}
	draft := intake.NewDraft(draftCatalog(t), filename, text, candidates, nil)
	draft.Plan(nil, "self")
	call, ok := draft.Call()
	if !ok {
		t.Fatal("missing filing fan-out")
	}
	question, ok := call.Questions[intake.FieldTitle]
	if !ok {
		t.Fatal("filename incorrectly settled before Jev can judge the printed title")
	}
	raw, err := json.Marshal(question.Criteria)
	if err != nil || !strings.Contains(string(raw), text.Runs[3].Text) {
		t.Fatalf("printed title missing from Jev choices: %s (%v)", raw, err)
	}
	for _, change := range []struct {
		name, title string
		page        int
	}{
		{"body prose", "Principal’s project requirements", 1},
		{"later page", "PRINCIPAL’S PROJECT REQUIREMENTS", 2},
	} {
		t.Run(change.name, func(t *testing.T) {
			text.Runs[3].Text = change.title
			text.Runs[3].Source.Page = change.page
			if _, ok := findCandidate(intake.Harvest(filename, text), intake.FieldTitle, change.title); ok {
				t.Fatal("ordinary body text offered as cover title")
			}
		})
	}
}
