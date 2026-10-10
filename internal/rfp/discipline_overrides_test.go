package rfp

import (
	"sitewise/internal/knowledge"
	"strings"
	"testing"
)

func TestPositionFollowsAppointedStage(t *testing.T) {
	d := discipline(t, "structural")
	d.Positions = append(d.Positions, knowledge.RFPPosition{Key: "appointment_test", Label: "Construction attendance", Kind: "technical", Value: &knowledge.RFPCell{Value: "I", When: knowledge.RFPWhen{{"stage_included": {"construction_phase"}}}, Else: "O"}})
	for _, c := range []struct{ state, want string }{{Include, "I"}, {Option, "O"}, {Exclude, "O"}} {
		r := Resolve(d, skeleton(t), facts(t), set("stage:construction_phase", `"`+c.state+`"`))
		if p := r.Position("appointment_test"); p.Value != c.want {
			t.Fatalf("stage %s: got %s, want %s", c.state, p.Value, c.want)
		}
	}
}

func TestFireResponsibilitiesUseSeparateAppointments(t *testing.T) {
	d := discipline(t, "structural")
	d.Positions = []knowledge.RFPPosition{{Key: "wet", Label: "Sprinkler design", Kind: "technical", Value: &knowledge.RFPCell{Value: "X"}, Responsible: "fire-services designer"}}
	for _, c := range []struct {
		id, title string
		allocated bool
	}{
		{"consultant.fire_engineer", "Fire Services", false},
		{"consultant.fire_services", "Selected designer", true},
		{"trade.fire_services", "Fire Services Contractor", false},
		{"", "Fire engineering", false},
		{"", "Fire Services Contractor", false},
		{"", "Fire Services", false},
		{"", "Fire-services design", true},
	} {
		drivers := facts(t)
		drivers.Consultants = []Consultant{{DisciplineID: c.id, Title: c.title, Status: "appointed"}}
		r := Resolve(d, skeleton(t), drivers, nil)
		allocated := !strings.Contains(r.Position("wet").Party, "not yet engaged")
		if allocated != c.allocated {
			t.Fatalf("%s / %s: %+v", c.id, c.title, r.Position("wet"))
		}
	}
	d.Positions[0].Responsible = "fire-services or electrical designer"
	drivers := facts(t)
	drivers.Consultants = []Consultant{{DisciplineID: "consultant.fire_services", Title: "Fire Services", Status: "appointed"}}
	if p := Resolve(d, skeleton(t), drivers, nil).Position("wet"); !strings.Contains(p.Party, "not yet engaged") {
		t.Fatalf("unsettled dry-fire alternative silently assigned: %+v", p)
	}
}

func TestScopeScheduleSurvivesSeparateFromPositions(t *testing.T) {
	d := discipline(t, "structural")
	d.ScopeSchedule = []knowledge.RFPRow{
		{Key: "intro", Text: "Name design, test and acceptance owners."},
		{Key: "interface", Label: "Power / controls", Text: "Allocate both sides; absent evidence is unresolved."},
	}
	r := Resolve(d, skeleton(t), facts(t), nil)
	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, section := range sections {
		if section.ID != "positions" {
			continue
		}
		if len(section.Blocks) != 2 || section.Blocks[0].ID != "rfp:positions" {
			t.Fatalf("positions overwritten: %+v", section.Blocks)
		}
		b := section.Blocks[1]
		if b.ID != "rfp:scope_schedule" || b.Text != d.ScopeSchedule[0].Text || b.Table == nil || len(b.Table.Rows) != 1 || b.Table.Rows[0][1] != d.ScopeSchedule[1].Text {
			t.Fatalf("allocation schedule lost: %+v", b)
		}
		return
	}
	t.Fatal("positions section missing")
}

func TestScopeChoicesDoNotLeakIntoOtherRFPs(t *testing.T) {
	sk := skeleton(t)
	sk.Flags = append(sk.Flags, knowledge.RFPOption{Key: "test_package", Label: "Other consultant package", ScopeDecision: true})
	d := discipline(t, "structural")
	drivers := facts(t, "flag.test_package=yes", "flag.flood=yes")
	conditions := func(r Resolved) string {
		labels := []string{}
		for _, c := range r.Conditions {
			labels = append(labels, c.Label+": "+c.Answer)
		}
		return strings.Join(labels, "; ")
	}
	text := conditions(Resolve(d, sk, drivers, nil))
	if strings.Contains(text, "Other consultant package") || !strings.Contains(text, "Flood") {
		t.Fatalf("unrelated scope choice or missing shared condition: %s", text)
	}
	d.Role = append(d.Role, knowledge.RFPRow{Key: "test_package", Text: "Appointed package", When: knowledge.RFPWhen{{"flag": {"test_package"}}}})
	// Its own scope choice is part of the appointment (section 2), never a
	// site condition on the shared page one.
	r := Resolve(d, sk, drivers, nil)
	if text := conditions(r); strings.Contains(text, "Other consultant package") {
		t.Fatalf("scope choice printed as a site condition: %s", text)
	}
	if !strings.Contains(strings.Join(func() []string {
		out := []string{}
		for _, row := range r.Role {
			out = append(out, row.Text)
		}
		return out
	}(), " "), "Appointed package") {
		t.Fatal("own scope choice missing from the role")
	}
}

func TestDisciplineContractOverrides(t *testing.T) {
	sk := skeleton(t)
	d := discipline(t, "structural")
	d.Scale.Complex = nil
	d.ProjectRows = append(d.ProjectRows, knowledge.RFPRow{Key: "procurement", Label: "Procurement", Text: "{procurement_route}; separate statutory appointment.", Replace: true})
	d.Terms = append(d.Terms, knowledge.RFPRow{Key: "payment", Text: "Certification payment arrangement.", Replace: true, Required: true, When: knowledge.RFPWhen{{"flag": {"heritage"}}}})
	d.ReturnableNotes = append(d.ReturnableNotes, knowledge.RFPRow{Key: "r3", Text: "Identify mandatory actions and their fees.", Replace: true})
	d.Evaluation = []knowledge.RFPWeight{{Key: "independence", Label: "Independence", Default: 75}, {Key: "price", Label: "Fee", Default: 25}}
	d.EvaluationNote = "Assess the separate appointment."
	d.AllowancesNote = "Required statutory action remains required."
	for _, flag := range []string{"yes", "no"} {
		r := Resolve(d, sk, facts(t, "flag.heritage="+flag, "procurement=design_construct", "novation=yes"), nil)
		if r.Scale != "simple" {
			t.Fatal("an empty complex condition must not imply complexity")
		}
		// The replacement rewords the page-one grid cell, once.
		cell, _ := gridCell(r, "Delivery", "Procurement route")
		if strings.Contains(cell.Text, "this appointment is novated") || !strings.Contains(cell.Text, "separate statutory appointment") {
			t.Fatal(cell.Text)
		}
		for _, row := range append(r.Project, r.Appointment...) {
			if row.Key == "procurement" {
				t.Fatalf("procurement printed twice: %s", row.Text)
			}
		}
		payments := []string{}
		for _, row := range r.Terms {
			if row.Key == "payment" {
				payments = append(payments, row.Text)
			}
		}
		if len(payments) != 1 || (strings.Contains(payments[0], "Certification") != (flag == "yes")) {
			t.Fatalf("conditional payment override: %s %+v", flag, payments)
		}
		for _, row := range r.Returnables {
			if row.Key == "r3" && (!strings.HasPrefix(row.Text, "Identify mandatory actions and their fees.") || strings.Contains(row.Text, "written authority")) {
				t.Fatalf("contradictory returnable: %s", row.Text)
			}
		}
		if len(r.Weights) != 2 || r.Weights[0].Key != "independence" || r.Weights[0].Value != 75 {
			t.Fatalf("discipline weights: %+v", r.Weights)
		}
		if r.EvalNote != d.EvaluationNote || r.AllowNote != d.AllowancesNote {
			t.Fatal("discipline notes ignored")
		}
	}
	if err := ValidateSetting(d, sk, facts(t), "weight:independence", []byte(`60`)); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSetting(d, sk, facts(t), "weight:method", []byte(`20`)); err == nil {
		t.Fatal("removed shared weight remains editable")
	}
}
