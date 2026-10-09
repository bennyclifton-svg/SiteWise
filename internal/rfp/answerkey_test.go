package rfp

import (
	"sort"
	"strings"
	"testing"
)

// TestSelectionAnswerKeys checks what each discipline selects for each
// scenario (stages, positions, decisions), independently of wording.
func TestSelectionAnswerKeys(t *testing.T) {
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	for _, s := range loadScenarios(t) {
		disciplines := make([]string, 0, len(s.Expect))
		for id := range s.Expect {
			disciplines = append(disciplines, id)
		}
		sort.Strings(disciplines)
		for _, id := range disciplines {
			want := s.Expect[id]
			t.Run(s.Name+"/"+id, func(t *testing.T) {
				d, ok := cat.RFP(id)
				if !ok {
					t.Skipf("discipline %s not transcribed yet", id)
				}
				drivers := s.drivers(t, sk, d)
				r := Resolve(d, sk, drivers, s.settings(t, id, drivers))
				for _, f := range compare(r, want) {
					t.Error(f)
				}
				if t.Failed() {
					t.Logf("checks: %s", describeChecks(r))
				}
			})
		}
	}
}

func compare(r Resolved, want expectation) []string {
	fails := []string{}
	stage := func(keys []string, states ...string) {
		for _, k := range keys {
			got := r.Stage(k)
			if got.Key == "" {
				fails = append(fails, "unknown stage "+k)
				continue
			}
			ok := false
			for _, s := range states {
				ok = ok || got.State == s
			}
			if !ok {
				fails = append(fails, "stage "+k+": want "+strings.Join(states, "/")+", got "+got.State+" ("+got.Why+")")
			}
		}
	}
	stage(want.StagesInclude, Include)
	stage(want.StagesOption, Option)
	stage(want.StagesAbsent, Exclude)
	stage(want.StagesAsk, Ask)
	element := func(keys []string, state string) {
		for _, k := range keys {
			found := false
			for _, e := range r.Elements {
				if e.Key == k {
					found = true
					if e.State != state {
						fails = append(fails, "element "+k+": want "+state+", got "+e.State)
					}
				}
			}
			if !found {
				fails = append(fails, "unknown element "+k)
			}
		}
	}
	for _, k := range want.AllowPresent {
		if r.Allowance(k).Key == "" {
			fails = append(fails, "missing allowance "+k)
		}
	}
	for _, k := range want.AllowAbsent {
		if r.Allowance(k).Key != "" {
			fails = append(fails, "unexpected allowance "+k)
		}
	}
	element(want.ElementsInclude, Include)
	element(want.ElementsOption, Option)
	element(want.ElementsAbsent, Exclude)
	for k, v := range want.Positions {
		got := r.Position(k)
		switch {
		case v == "absent" && got.Key != "":
			fails = append(fails, "position "+k+": want absent, got "+got.Value)
		case v != "absent" && got.Value != v:
			fails = append(fails, "position "+k+": want "+v+", got "+orAbsent(got.Value)+" ("+got.Why+")")
		}
	}
	for k, v := range want.Party {
		if got := r.Position(k); !strings.Contains(got.Party, v) {
			fails = append(fails, "position "+k+": party want "+v+", got "+got.Party)
		}
	}
	blocking := map[string]bool{}
	all := map[string]bool{}
	for _, c := range r.Checks {
		all[c.ID] = true
		if c.Blocking {
			blocking[c.ID] = true
		}
	}
	for _, id := range want.Blocking {
		if !blocking[id] {
			fails = append(fails, "missing blocking check "+id)
		}
	}
	if want.BlockingOnly {
		for id := range blocking {
			if !contains(want.Blocking, id) {
				fails = append(fails, "unexpected blocking check "+id)
			}
		}
	}
	for _, id := range want.Checks {
		if !all[id] {
			fails = append(fails, "missing check "+id)
		}
	}
	for _, id := range want.NoChecks {
		if all[id] {
			fails = append(fails, "unexpected check "+id)
		}
	}
	if want.Scale != "" && r.Scale != want.Scale {
		fails = append(fails, "scale: want "+want.Scale+", got "+r.Scale+" ("+r.ScaleWhy+")")
	}
	roles := map[string]bool{}
	for _, row := range r.Role {
		roles[row.Key] = true
	}
	for _, k := range want.Role {
		if !roles[k] {
			fails = append(fails, "missing role line "+k)
		}
	}
	for _, k := range want.RoleAbsent {
		if roles[k] {
			fails = append(fails, "unexpected role line "+k)
		}
	}
	return fails
}

func orAbsent(v string) string {
	if v == "" {
		return "absent"
	}
	return v
}

func describeChecks(r Resolved) string {
	parts := []string{}
	for _, c := range r.Checks {
		mark := ""
		if c.Blocking {
			mark = "!"
		}
		parts = append(parts, mark+c.ID)
	}
	return strings.Join(parts, ", ")
}
