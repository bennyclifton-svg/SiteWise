package works

import (
	"sitewise/internal/knowledge"
	"testing"
)

func TestK1LayoutSmokeAndThermalReviews(t *testing.T) {
	cat := proposalCatalogue(t)
	for _, id := range []string{"cq.room-layout-smoke-strategy-review", "cq.room-layout-thermal-zoning-review"} {
		for _, mode := range []string{"yes", "no", "unknown", "absent", "excluded"} {
			t.Run(id+"/"+mode, func(t *testing.T) {
				item := Item{ID: "wall", PartID: "A", SystemID: "interiors.walls-linings", Action: "alter", Inclusion: "included", LayoutChange: mode}
				if mode == "absent" || mode == "excluded" {
					item.LayoutChange = "yes"
				}
				if mode == "excluded" {
					item.Inclusion = "excluded"
				}
				in := ProposalInput{Items: []Item{item}, Parts: map[string]ProposalPart{"A": {}}, Existing: func(string, string) knowledge.Truth {
					if mode == "absent" {
						return knowledge.False
					}
					return knowledge.True
				}}
				ps, err := EvaluateProposals(cat, in)
				if err != nil {
					t.Fatal(err)
				}
				count := 0
				for _, p := range ps {
					if p.RecordID != id {
						continue
					}
					count++
					if !p.Draft || p.Kind != "hold_point" || p.TargetSystemID != "" || len(p.Reason.Triggers) != 1 || p.Reason.Triggers[0].WorkItemID != "wall" {
						t.Fatal(p)
					}
				}
				want := 1
				if mode == "no" || mode == "absent" || mode == "excluded" {
					want = 0
				}
				if count != want {
					t.Fatalf("got%d want%d", count, want)
				}
			})
		}
	}
}
