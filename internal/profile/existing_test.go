package profile_test

import (
	"sitewise/internal/profile"
	"testing"
)

func TestExistingPresenceRoutingByWorkTypeAndPart(t *testing.T) {
	cat := repoCatalog(t)
	for _, projectType := range []string{"new", "extend", "refurb", "remediation", "advisory"} {
		for _, partType := range []string{"", "new", "extend", "refurb", "remediation", "advisory"} {
			for _, action := range []string{"", "not_stated", "repair"} {
				t.Run(projectType+"/"+partType+"/"+action, func(t *testing.T) {
					f := jevFact("sys.hydraulic.gas.presence", "included", "spec", .9)
					f.PartLabel = "Building B"
					in := input(f)
					in.User = []profile.UserValue{{PartID: whole, Key: "hdr.work_type", Value: str(projectType)}}
					if partType != "" {
						in.User = append(in.User, profile.UserValue{PartID: "p-b", Key: "hdr.work_type", Value: str(partType)})
					}
					if action != "" {
						a := jevFact("sys.hydraulic.gas.action", action, "spec", .9)
						a.PartLabel, a.DecidedBy = "Building B", "rule" // action-question calibration belongs to WP-22
						in.Facts = append(in.Facts, a)
					}
					rows := profile.Build(in, cat)
					kind := projectType
					if partType != "" {
						kind = partType
					}
					wantExisting := (kind == "refurb" || kind == "remediation" || kind == "advisory") && action != "repair"
					presence := row(t, rows, "p-b", "sys.hydraulic.gas.presence")
					if presence.Value != "included" || len(presence.Sources) != 1 {
						t.Fatal("factual presence was changed")
					}
					items := profile.ProposedWorks("project", rows, in.Parts, cat)
					if wantExisting {
						existing := row(t, rows, "p-b", "sys.hydraulic.gas.existing")
						if existing.Value != "present" || existing.Scope != "site" || existing.Sources[0].DocumentID != "spec" {
							t.Fatalf("site provenance %+v", existing)
						}
						if len(items) != 0 {
							t.Fatalf("presence became works: %+v", items)
						}
					} else {
						noRow(t, rows, "p-b", "sys.hydraulic.gas.existing")
						if len(items) != 1 || items[0].PartID != "p-b" {
							t.Fatalf("missing part work: %+v", items)
						}
					}
				})
			}
		}
	}
}

func TestExistingPresenceUserScopeAndSiteOverride(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "spec", .9))
	in.User = append(header("house", "refurb"), profile.UserValue{PartID: whole, Key: "scope.hydraulic.gas", Value: str("in")})
	rows := profile.Build(in, cat)
	items := profile.ProposedWorks("project", rows, in.Parts, cat)
	if len(items) != 1 || items[0].Action != "alter" {
		t.Fatalf("user scope lost: %+v", items)
	}
	in.User = append(in.User, profile.UserValue{PartID: whole, Key: "sys.hydraulic.gas.existing", Value: str("absent")})
	r := row(t, profile.Build(in, cat), whole, "sys.hydraulic.gas.existing")
	if r.Value != "absent" || r.Band != "user" {
		t.Fatalf("site user value overwritten: %+v", r)
	}
}

func TestUnresolvedPresenceDoesNotInventExistingSystem(t *testing.T) {
	cat := repoCatalog(t)
	for _, value := range []string{"included", "not_included"} {
		in := input(jevFact("sys.hydraulic.gas.presence", value, "spec", .2))
		in.User = header("house", "refurb")
		rows := profile.Build(in, cat)
		noRow(t, rows, whole, "sys.hydraulic.gas.existing")
		if len(profile.ProposedWorks("project", rows, in.Parts, cat)) != 0 {
			t.Fatal("unresolved reading became works")
		}
	}
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "spec", .9), jevFact("sys.hydraulic.gas.presence", "not_included", "report", .9))
	in.User = header("house", "refurb")
	rows := profile.Build(in, cat)
	noRow(t, rows, whole, "sys.hydraulic.gas.existing")
	if len(profile.ProposedWorks("project", rows, in.Parts, cat)) != 0 {
		t.Fatal("conflicting presence became work")
	}
}

func TestPartWorkTypeNeverReplacesProjectDefaultType(t *testing.T) {
	cat := repoCatalog(t)
	in := input()
	// Sorting puts Building B before Whole project. Header lookup must still
	// use the whole part, or the extension would activate project-wide defaults.
	in.User = append(header("house", "refurb"), profile.UserValue{PartID: "p-b", Key: "hdr.work_type", Value: str("new")})
	rows := profile.Build(in, cat)
	if len(profile.ProposedWorks("project", rows, in.Parts, cat)) != 0 {
		t.Fatal("part override contaminated project defaults")
	}
}
