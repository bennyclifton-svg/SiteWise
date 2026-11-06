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
		// Since the action floor was calibrated (2026-10-24), an applied
		// action is scope on its own (Walsh review, 2026-10-27); below its
		// floor it still is not.
		{"applied action alone is scope", "replace", conf(.8), false, "replace"},
		{"action below floor alone is not scope", "replace", conf(.99), false, ""},
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

// Knock-down D2 (F18): on new work, a passage about the old building or an
// inspection of the new work does not change the new system's action, and
// a removal reading alone is not scope. Other actions still apply, and on
// an extension an investigation still does.
func TestOldBuildingActionsDoNotChangeNewWork(t *testing.T) {
	cat := repoCatalog(t)
	for _, tc := range []struct {
		name, work, action string
		presence           bool
		want               string
	}{
		{"investigate on new work", "new", "investigate", true, "new"},
		{"remove on new work", "new", "remove", true, "new"},
		{"remove alone on new work is not scope", "new", "remove", false, ""},
		{"retain on new work still applies", "new", "retain", true, "retain"},
		{"investigate on an extension still applies", "extend", "investigate", true, "investigate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := input(jevFact("sys.hydraulic.gas.action", tc.action, "action-doc", .9))
			in.Thresholds.Amber["action"] = .8
			in.User = []profile.UserValue{{PartID: whole, Key: "hdr.work_type", Value: str(tc.work)}}
			if tc.presence {
				in.Facts = append(in.Facts, jevFact("sys.hydraulic.gas.presence", "included", "presence-doc", .9))
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
		})
	}
}

// Knock-down D5 (F30): a header condition the taxonomy limits to other
// building categories is not kept from a reading; the user's value is.
func TestConditionOutsideItsCategoryIsDropped(t *testing.T) {
	cat := repoCatalog(t)
	key := "hdr.cond.hazardous_process"
	for _, tc := range []struct {
		category string
		user     bool
		want     bool
	}{
		{"residential", false, false},
		{"industrial", false, true},
		{"", false, true},
		{"residential", true, true},
	} {
		in := input(jevFact(key, "dangerous_goods", "hazmat", .95))
		if tc.category != "" {
			in.User = append(in.User, profile.UserValue{PartID: whole, Key: "hdr.building_class", Value: str(tc.category)})
		}
		if tc.user {
			in.User = append(in.User, profile.UserValue{PartID: whole, Key: key, Value: str("dangerous_goods")})
		}
		found := false
		for _, r := range profile.Build(in, cat) {
			if r.Key == key && r.Value == "dangerous_goods" {
				found = true
			}
		}
		if found != tc.want {
			t.Errorf("category %q user %v: kept %v, want %v", tc.category, tc.user, found, tc.want)
		}
	}
}
