package profile_test

import (
	"sitewise/internal/profile"
	"sitewise/internal/works"
	"strings"
	"testing"
)

func TestActionQuestionUsesCatalogueBoundaries(t *testing.T) {
	cat := repoCatalog(t)
	q := profile.EvidenceQuestions([]string{"hydraulic.gas"}, cat)["sys.hydraulic.gas.action"]
	criteria := q.Criteria.(map[string]string)
	if len(criteria) != 10 || !strings.Contains(q.Instructions.(string), "`text`") {
		t.Fatalf("action question %+v", q)
	}
	for _, a := range cat.Actions() {
		if !strings.Contains(criteria[a.ID], a.Describes) || !strings.Contains(criteria[a.ID], a.Excludes) {
			t.Fatalf("missing catalogue boundary: %s", a.ID)
		}
	}
	for id, wording := range cat.ActionAnswers() {
		if criteria[id] != wording {
			t.Fatalf("duplicated extra wording: %s", id)
		}
	}
}

func TestActionThresholdAndScopeAreIndependent(t *testing.T) {
	cat := repoCatalog(t)
	for _, tc := range []struct {
		name, action string
		floor        *float64
		presence     bool
		want         string
	}{
		{"own floor applied", "replace", conf(.8), true, "replace"},
		{"presence floor is not action floor", "replace", nil, true, "new"},
		{"action below own floor", "replace", conf(.99), true, "new"},
		{"silence defaults", "not_stated", conf(.8), true, "new"},
		{"action alone is not scope", "replace", conf(.8), false, ""},
		{"several needs mapping", "several", conf(.8), true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := input(jevFact("sys.hydraulic.gas.action", tc.action, "action-doc", .9))
			in.User = []profile.UserValue{{PartID: whole, Key: "hdr.work_type", Value: str("new")}}
			if tc.presence {
				in.Facts = append(in.Facts, jevFact("sys.hydraulic.gas.presence", "included", "presence-doc", .9))
			}
			if tc.floor != nil {
				in.Thresholds.Amber["action"] = *tc.floor
			}
			rows := profile.Build(in, cat)
			items := profile.ProposedWorks("project", rows, in.Parts, cat)
			if tc.want == "" {
				if len(items) != 0 {
					t.Fatalf("invented scope %+v", items)
				}
				return
			}
			if len(items) != 1 || items[0].Action != tc.want {
				t.Fatalf("actions %+v", items)
			}
			if tc.want == "replace" && !strings.Contains(string(items[0].Provenance.Sources), "action-doc") {
				t.Fatal("action provenance lost")
			}
		})
	}
}

func TestAcceptedWorkActionConflictIsVisible(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.hydraulic.gas.action", "replace", "spec", .9))
	in.Thresholds.Amber["action"] = .8
	rows := profile.ProjectWorkScope(profile.Build(in, cat), []works.Item{{PartID: whole, SystemID: "hydraulic.gas", Action: "repair", Inclusion: "included", UserTouched: true, ReviewStatus: profile.ReviewAccepted}})
	r := row(t, rows, whole, "scope.hydraulic.gas")
	if !strings.Contains(r.Note, "repair") || !strings.Contains(r.Note, "replace") || r.Band != "user" {
		t.Fatalf("conflict %+v", r)
	}
}
