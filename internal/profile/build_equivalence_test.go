package profile

import (
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/knowledge"
)

// Keep the previous complete two-pass algorithm as an independent oracle:
// the optimization must preserve every row, source, alternative and annotation.
func fullTwoPassBuild(in Input, cat *knowledge.Catalog) []Row {
	in.Facts = readFacts(in.Facts, in.Read)
	rows := Reconcile(in, cat)
	whole := wholePart(in.Parts)
	defaults := cat.ScopeDefaults(headerValue(rows, whole, "hdr.building_class"), headerValue(rows, whole, "hdr.subclass"), headerValue(rows, whole, "hdr.work_type"))
	removed := map[string]bool{}
	for _, u := range in.User {
		if leaf, ok := strings.CutPrefix(u.Key, scopePrefix); ok && u.Value != nil && *u.Value == scopeOut {
			removed[leaf] = true
		}
	}
	var suggested []string
	for _, leaf := range defaults {
		if !removed[leaf] {
			suggested = append(suggested, leaf)
		}
	}
	if len(suggested) > 0 {
		in.Suggested = suggested
		rows = Reconcile(in, cat)
	}
	rows = withScope(withExistingSystems(rows, whole), whole, suggested)
	Annotate(rows, cat)
	return rows
}

func TestScopeHeaderPassPreservesCompleteBuild(t *testing.T) {
	cat, err := knowledge.Load("../../knowledge")
	if err != nil {
		t.Fatal(err)
	}
	text := func(v string) *string { return &v }
	confidence := .9
	for _, work := range []string{"new", "extend", "refurb", "remediation", "advisory"} {
		for _, variant := range []string{"evidence", "user", "unknown", "cleared", "assumption", "conflict", "superseded", "part", "scope-out", "suggested"} {
			t.Run(work+"/"+variant, func(t *testing.T) {
				in := Input{
					Parts:      []Part{{ID: "whole", Kind: "whole"}, {ID: "part", Label: "Wing", Kind: "building"}},
					Thresholds: Thresholds{Amber: map[string]float64{"header": .6, "presence": .6, "determinant": .6, "assertion": .6}},
					User:       []UserValue{{PartID: "whole", Key: "hdr.work_type", Value: text(work)}},
					Planning:   []PlanningValue{{PartID: "whole", Key: "estimated_area", Value: text("120"), Origin: OriginAssumption}},
				}
				for _, pair := range [][2]string{{"hdr.subclass", "house"}, {"det.ncc_class", "1a"}, {"det.rise_in_storeys", "2"}, {"sys.fire-active.sprinklers.presence", "included"}, {"sys.fire-active.sprinklers.assertion", "requirement"}} {
					in.Facts = append(in.Facts, Fact{QuestionID: pair[0], Value: pair[1], DocumentID: "doc", PassageID: "passage", DecidedBy: "jev", Confidence: &confidence, Excerpt: "Full source text", FileSHA256: "hash", Page: 2})
				}
				switch variant {
				case "user", "unknown", "cleared", "assumption":
					u := UserValue{PartID: "whole", Key: "hdr.subclass", Value: text("apartment"), Origin: OriginUser}
					if variant == "unknown" || variant == "cleared" {
						u.State, u.Value = variant, nil
					}
					if variant == "assumption" {
						u.Origin = OriginAssumption
					}
					in.User = append(in.User, u)
				case "conflict":
					f := in.Facts[0]
					f.DocumentID, f.Value = "conflict", "warehouse"
					in.Facts = append(in.Facts, f)
				case "superseded":
					in.Facts[0].Superseded = true
				case "part":
					in.Facts[0].PartLabel = "Wing"
					in.User = append(in.User, UserValue{PartID: "part", Key: "hdr.work_type", Value: text("new")})
				case "scope-out":
					in.User = append(in.User, UserValue{PartID: "whole", Key: "scope.substructure.footings", Value: text("out")})
				case "suggested":
					in.Suggested = []string{"fire-active.sprinklers"}
				}
				want, got := fullTwoPassBuild(in, cat), Build(in, cat)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("optimized build differs from full evidence reconciliation\nwant=%+v\ngot=%+v", want, got)
				}
			})
		}
	}
}
