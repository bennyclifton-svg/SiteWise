package profile_test

import (
	"sitewise/internal/profile"
	"strings"
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

// A refurb brief that lists a system as the contractor's works is scope, not
// existing building, even when it names no action (Meridian fitout, 2026-10-24).
func TestContractorProvidedSystemIsWorksInRefurb(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.electrical.lighting.presence", "included", "brief", .85),
		jevFact("sys.electrical.lighting.provider", "contractor", "brief", .77))
	in.User = header("office", "refurb")
	rows := profile.Build(in, cat)
	noRow(t, rows, whole, "sys.electrical.lighting.existing")
	if r := row(t, rows, whole, "scope.electrical.lighting"); r.Value != "in" {
		t.Fatalf("contractor works left out of scope: %+v", r)
	}
	items := profile.ProposedWorks("project", rows, in.Parts, cat)
	if len(items) != 1 || items[0].Action != "alter" {
		t.Fatalf("want one alter work item: %+v", items)
	}
	// Another party supplying it says nothing about the works; D-27 still holds.
	in = input(jevFact("sys.electrical.lighting.presence", "included", "brief", .85),
		jevFact("sys.electrical.lighting.provider", "owner", "brief", .77))
	in.User = header("office", "refurb")
	rows = profile.Build(in, cat)
	if r := row(t, rows, whole, "sys.electrical.lighting.existing"); r.Value != "present" {
		t.Fatalf("owner-supplied presence should stay existing: %+v", r)
	}
}

// refurbWithActions is an office refurb whose documents include a system and
// read the given actions for it, with the real action floor (0.85).
func refurbWithActions(leaf string, actions ...profile.Fact) profile.Input {
	in := input(append([]profile.Fact{jevFact("sys."+leaf+".presence", "included", "brief", .9)}, actions...)...)
	in.Thresholds.Amber["action"] = .85
	in.User = header("office", "refurb")
	return in
}

// R16 (F43): on a refurb, an included system whose works action is unsettled
// is in scope. T03 read "new" for its interiors below the action floor; T06
// read upgrade against alter for its sprinklers. Neither is existing building.
func TestRefurbUnsettledWorksActionStaysInScope(t *testing.T) {
	cat := repoCatalog(t)
	cases := map[string]profile.Input{
		"below the floor": refurbWithActions("interiors.joinery", jevFact("sys.interiors.joinery.action", "new", "brief", .73)),
		"works actions conflict": refurbWithActions("fire-active.sprinklers",
			jevFact("sys.fire-active.sprinklers.action", "upgrade", "brief", .99),
			jevFact("sys.fire-active.sprinklers.action", "alter", "spec", 1)),
		"works against retain": refurbWithActions("fire-active.fire-water",
			jevFact("sys.fire-active.fire-water.action", "upgrade", "brief", .99),
			jevFact("sys.fire-active.fire-water.action", "retain", "spec", .97)),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			leaf := strings.TrimSuffix(strings.TrimPrefix(in.Facts[0].QuestionID, "sys."), ".presence")
			rows := profile.Build(in, cat)
			noRow(t, rows, whole, "sys."+leaf+".existing")
			if r := row(t, rows, whole, "scope."+leaf); r.Value != "in" || r.Band != "amber" {
				t.Fatalf("unsettled works left scope: %+v", r)
			}
			if a := row(t, rows, whole, "sys."+leaf+".action"); a.Value != "" {
				t.Fatalf("the action must stay unsettled for the user: %+v", a)
			}
			if items := profile.ProposedWorks("project", rows, in.Parts, cat); len(items) != 1 || items[0].SystemID != leaf {
				t.Fatalf("want one work item for %s: %+v", leaf, items)
			}
		})
	}
}

// No action read, or a conflict with no works action in it, keeps D-27: the
// system is existing building, not scope. (A settled action keeps its own,
// earlier behaviour and is not this rule's business.)
func TestRefurbSettledOrNonWorksActionStaysExisting(t *testing.T) {
	cat := repoCatalog(t)
	cases := map[string]profile.Input{
		"retain against investigate": refurbWithActions("fire-active.hydrants",
			jevFact("sys.fire-active.hydrants.action", "retain", "spec", .97),
			jevFact("sys.fire-active.hydrants.action", "investigate", "report", .91)),
		"no action read": refurbWithActions("fire-active.hydrants"),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			rows := profile.Build(in, cat)
			if r := row(t, rows, whole, "sys.fire-active.hydrants.existing"); r.Value != "present" {
				t.Fatalf("existing building lost: %+v", r)
			}
			if r, ok := scopeRow(rows, "fire-active.hydrants"); ok && r.Band != "suggested" {
				t.Fatalf("existing building put in scope from evidence: %+v", r)
			}
		})
	}
}
