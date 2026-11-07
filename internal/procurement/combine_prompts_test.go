package procurement

import (
	"reflect"
	"testing"
	"time"
)

func promptRow(id, discipline, title string) Package {
	return Package{ID: id, Kind: "services", Role: RoleSingle, DisciplineID: discipline, Title: title, LifecycleStatus: "planned"}
}

func architectLead(d string) bool { return d == "consultant.architect" }

func TestCombinePromptServices(t *testing.T) {
	items := []Package{
		promptRow("se", ServicesFiling, "Services Engineer"),
		promptRow("mech", "consultant.mechanical", "Mechanical"),
		promptRow("elec", "consultant.electrical", "Electrical"),
		promptRow("str", "consultant.structural", "Structural"),
	}
	got := CombinePrompts(items, architectLead, nil, nil)
	if len(got) != 1 || got[0].Key != PromptServices || got[0].Reason != "services" {
		t.Fatalf("one services prompt: %+v", got)
	}
	if want := []string{"se", "mech", "elec"}; !reflect.DeepEqual(got[0].PackageIDs, want) {
		t.Fatalf("members ticked %v, want %v", got[0].PackageIDs, want)
	}
	if got[0].Text == "" {
		t.Fatal("a prompt says why")
	}
	// Alone, a Services Engineer row has nothing to combine with.
	if got := CombinePrompts(items[:1], architectLead, nil, nil); len(got) != 0 {
		t.Fatalf("no prompt for a lone services row: %+v", got)
	}
	// Separate services rows without a combined row are not prompted.
	if got := CombinePrompts(items[1:], architectLead, nil, nil); len(got) != 0 {
		t.Fatalf("no prompt without a combined row: %+v", got)
	}
	if got := CombinePrompts(items, architectLead, nil, map[string]bool{PromptServices: true}); len(got) != 0 {
		t.Fatalf("a dismissed prompt stays hidden: %+v", got)
	}
}

func TestCombinePromptEngagedBy(t *testing.T) {
	access := promptRow("acc", "consultant.access", "Access")
	access.EngagedBy = EngagedByLead
	bca := promptRow("bca", "consultant.bca", "BCA")
	bca.EngagedBy = EngagedByClient
	items := []Package{promptRow("arch", "consultant.architect", "Architect"), access, bca}
	got := CombinePrompts(items, architectLead, nil, nil)
	if len(got) != 1 || got[0].Key != PromptEngagedBy || got[0].Reason != "engaged_by" {
		t.Fatalf("one engaged-by prompt: %+v", got)
	}
	if want := []string{"arch", "acc"}; !reflect.DeepEqual(got[0].PackageIDs, want) {
		t.Fatalf("lead then subconsultants %v, want %v", got[0].PackageIDs, want)
	}
	// No lead row: nothing to combine the subconsultant with.
	if got := CombinePrompts(items[1:], architectLead, nil, nil); len(got) != 0 {
		t.Fatalf("no prompt without a lead: %+v", got)
	}
}

func TestCombinePromptsLeaveOutRowsThatCannotCombine(t *testing.T) {
	now := time.Now()
	retired := promptRow("hyd", "consultant.hydraulic", "Hydraulic")
	retired.RetiredAt = &now
	grouped := promptRow("fire", "consultant.fire_services", "Fire services")
	grouped.GroupID = "g"
	group := Package{ID: "g", Kind: "services", Role: RoleGroup, Title: "Fire and electrical"}
	items := []Package{promptRow("se", ServicesFiling, "Services Engineer"), retired, grouped, group, promptRow("mech", "consultant.mechanical", "Mechanical"), promptRow("elec", "consultant.electrical", "Electrical")}
	got := CombinePrompts(items, architectLead, map[string]bool{"elec": true}, nil)
	if len(got) != 1 || !reflect.DeepEqual(got[0].PackageIDs, []string{"se", "mech"}) {
		t.Fatalf("retired, grouped and awarded rows are not offered: %+v", got)
	}
	// Awarded Services Engineer: no prompt at all.
	if got := CombinePrompts(items, architectLead, map[string]bool{"se": true}, nil); len(got) != 0 {
		t.Fatalf("awarded combined row: %+v", got)
	}
	// A grouped Services Engineer row is already combined.
	se := promptRow("se", ServicesFiling, "Services Engineer")
	se.GroupID = "g"
	if got := CombinePrompts([]Package{se, promptRow("mech", "consultant.mechanical", "Mechanical")}, architectLead, nil, nil); len(got) != 0 {
		t.Fatalf("grouped combined row: %+v", got)
	}
}
