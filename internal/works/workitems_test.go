package works

import "testing"

func TestCoarseIdentityAndDefaults(t *testing.T) {
	id := CoarseID("project", "part", "structure")
	if id != CoarseID("project", "part", "structure") || id == CoarseID("project", "other", "structure") || id == CoarseID("other", "part", "structure") || id[14] != '5' {
		t.Fatal("unstable or non-v5 identity", id)
	}
	for _, c := range []struct{ part, project, want string }{{"", "new", "new"}, {"refurb", "extend", "alter"}, {"remediation", "new", "repair"}, {"advisory", "new", "investigate"}, {"", "", "investigate"}} {
		if got := DefaultAction(nil, c.part, c.project); got != c.want {
			t.Fatalf("%+v: %s", c, got)
		}
	}
}
