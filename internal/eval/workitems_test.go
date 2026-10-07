package eval

import (
	"sitewise/internal/profile"
	"sitewise/internal/works"
	"testing"
	"time"
)

func workScoreFixture() (WorkKey, WorkSnapshot) {
	key := WorkKey{SchemaVersion: 2, Project: "fixture", Reviewed: true, Documents: []string{"drawing"}, Parts: []string{"A", "B"}, Items: []ExpectedWork{
		{ID: "one", System: "fire-active.hydrants", Action: "upgrade", Part: "A", Document: "drawing", Page: 1},
		{ID: "two", System: "fire-active.sprinklers", Action: "replace", Part: "B", Document: "drawing", Page: 2},
	}}
	actual := WorkSnapshot{Project: "fixture", Parts: []profile.Part{{ID: "a", Label: "A"}, {ID: "b", Label: "B"}}, Items: []works.Item{
		{ID: "1", PartID: "a", SystemID: "fire-active.hydrants", Action: "upgrade", Inclusion: "included"},
		{ID: "2", PartID: "b", SystemID: "fire-active.sprinklers", Action: "replace", Inclusion: "included"},
	}}
	return key, actual
}

func TestWorkScoreRequiresExactSemanticsAndCountsDuplicates(t *testing.T) {
	for _, field := range []string{"system", "part", "action", "duplicate", "abstain", "correct"} {
		t.Run(field, func(t *testing.T) {
			key, actual := workScoreFixture()
			switch field {
			case "system":
				actual.Items[0].SystemID = "fire-active"
			case "part":
				actual.Items[0].PartID = "b"
			case "action":
				actual.Items[0].Action = "replace"
			case "duplicate":
				actual.Items = append(actual.Items, actual.Items[0])
				actual.Items[2].ID = "3"
			case "abstain":
				actual.Items = nil
			}
			score, err := ScoreWorkItems(key, actual)
			if err != nil {
				t.Fatal(err)
			}
			switch field {
			case "correct":
				if score.TP != 2 || score.FP != 0 || score.FN != 0 || !score.GatePassed {
					t.Fatal(score)
				}
			case "duplicate":
				if score.TP != 2 || score.FP != 1 || score.FN != 0 || score.GatePassed {
					t.Fatal(score)
				}
			case "abstain":
				if score.TP != 0 || score.FN != 2 || score.Precision != nil || score.GatePassed {
					t.Fatal(score)
				}
			default:
				if score.TP != 1 || score.FP != 1 || score.FN != 1 || score.GatePassed {
					t.Fatal(score)
				}
			}
		})
	}
}

func TestWorkScoreReviewAndPhysicalWorkBoundary(t *testing.T) {
	key, actual := workScoreFixture()
	key.Reviewed = false
	now := time.Now()
	for _, item := range []works.Item{{ID: "g", IsGroup: true}, {ID: "r", RetiredAt: &now}, {ID: "x", Inclusion: "excluded"}} {
		actual.Items = append(actual.Items, item)
	}
	score, err := ScoreWorkItems(key, actual)
	if err != nil || score.TP != 2 || score.FP != 0 || score.GateEligible || score.GatePassed {
		t.Fatalf("%+v %v", score, err)
	}
	key.Items = append(key.Items, key.Items[0])
	if _, err := ScoreWorkItems(key, actual); err == nil {
		t.Fatal("duplicate key passed")
	}
	key, actual = workScoreFixture()
	actual.Items[0].PartID = "missing"
	if _, err := ScoreWorkItems(key, actual); err == nil {
		t.Fatal("unknown part passed")
	}
	key, actual = workScoreFixture()
	actual.Project = "other"
	if _, err := ScoreWorkItems(key, actual); err == nil {
		t.Fatal("wrong project passed")
	}
}
