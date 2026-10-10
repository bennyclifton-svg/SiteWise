package rfp

import (
	"strings"
	"testing"

	"sitewise/internal/reports"
)

// Page one is the project, shared by every RFP; the appointment starts in
// section 2 (docs/plans/2026-10-10-rfp-page-one-design.md).

func gridCell(r Resolved, column, label string) (GridCell, bool) {
	for _, c := range r.Grid {
		if c.Title != column {
			continue
		}
		for _, cell := range c.Cells {
			if cell.Label == label {
				return cell, true
			}
		}
	}
	return GridCell{}, false
}

func TestGridMirrorsProfileColumns(t *testing.T) {
	sk := skeleton(t)
	r := Resolve(discipline(t, "architecture"), sk, facts(t, "work_type=new", "pathway=da", "procurement=traditional", "site_area=31720", "class=2"), Settings{})
	titles := []string{}
	for _, c := range r.Grid {
		titles = append(titles, c.Title)
	}
	if strings.Join(titles, ",") != "The building,Site and planning,Delivery" {
		t.Fatalf("columns = %v", titles)
	}
	if c, ok := gridCell(r, "Site and planning", "Site area"); !ok || c.Text != "31,720 m²" || c.Control != "fact:site_area" {
		t.Fatalf("site area = %+v %v", c, ok)
	}
	if c, _ := gridCell(r, "The building", "Work type"); c.Text != "New build" || c.Control != "fact:work_type" {
		t.Fatalf("work type = %+v", c)
	}
	if c, ok := gridCell(r, "The building", "Storeys"); !ok || !c.Empty {
		t.Fatal("an unknown, non-essential row is kept for the screen but marked empty")
	}
	if c, ok := gridCell(r, "Delivery", "Client"); !ok || c.Text != "___" {
		t.Fatalf("an essential row prints a blank: %+v %v", c, ok)
	}
	if c, _ := gridCell(r, "Site and planning", "Site address"); c.Text != "1 Test St" {
		t.Fatalf("address = %+v", c)
	}
}

func TestDisciplineReplacesAGridCell(t *testing.T) {
	r := Resolve(discipline(t, "access"), skeleton(t), facts(t, "work_type=new", "procurement=traditional", "class=2"), Settings{})
	if c, _ := gridCell(r, "Delivery", "Procurement route"); c.Text != "Traditional lump sum; no automatic later appointment or novation." {
		t.Fatalf("procurement = %q", c.Text)
	}
	for _, row := range r.Appointment {
		if row.Key == "procurement" {
			t.Fatal("a replaced grid cell is not repeated in section 2")
		}
	}
}

func TestConditionsPrintEveryAnswer(t *testing.T) {
	sk := skeleton(t)
	r := Resolve(discipline(t, "architecture"), sk, facts(t, "work_type=new", "class=2", "flag.rail=yes"), Settings{})
	answers := map[string]Condition{}
	for _, c := range r.Conditions {
		answers[c.Key] = c
	}
	if c := answers["country"]; c.Answer != "Not known" || c.Control != "fact:flag.country" {
		t.Fatalf("country = %+v", c)
	}
	if answers["rail"].Answer != "Yes" {
		t.Fatal("a condition known to apply prints even when the discipline does not test it")
	}
	if answers["existing_building"].Answer != "No" {
		t.Fatalf("existing building = %+v", answers["existing_building"])
	}
	for _, f := range sk.Flags {
		if _, ok := answers[f.Key]; ok && f.ScopeDecision {
			t.Fatalf("%s is an appointment choice, not a site condition", f.Key)
		}
	}
}

func TestDisciplineRowsMoveToTheAppointment(t *testing.T) {
	drivers := facts(t, "work_type=new", "class=2")
	drivers.Text["design_vehicle"] = "19 m semi-trailer"
	r := Resolve(discipline(t, "civil"), skeleton(t), drivers, set("field:design_vehicle", `"19 m semi-trailer"`))
	keys := []string{}
	for _, row := range r.Project {
		keys = append(keys, row.Key)
	}
	if strings.Join(keys, ",") != "intro" {
		t.Fatalf("shared rows = %v", keys)
	}
	found := false
	for _, row := range r.Appointment {
		found = found || (row.Label == "Design vehicle" && row.Text == "19 m semi-trailer")
	}
	if !found {
		t.Fatalf("appointment rows = %+v", r.Appointment)
	}
}

func TestIntroNamesTheWholeProject(t *testing.T) {
	r := Resolve(discipline(t, "architecture"), skeleton(t), facts(t, "work_type=new", "class=7b", "class=5"), set("parts:affected", `["p7b"]`))
	if !strings.Contains(r.Project[0].Text, "Part 7b (Class 7b) and Part 5 (Class 5)") {
		t.Fatalf("intro = %q", r.Project[0].Text)
	}
	if len(r.Appointment) == 0 || r.Appointment[0].Text != "This appointment covers: Part 7b." {
		t.Fatalf("appointment = %+v", r.Appointment)
	}
}

func TestPageOnePrintsTheGridAndConditions(t *testing.T) {
	sk := skeleton(t)
	r := Resolve(discipline(t, "civil"), sk, facts(t, "work_type=new", "pathway=da", "class=2"), set("field:design_vehicle", `"19 m semi-trailer"`, "field:information", `"Survey and geotechnical report"`))
	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	ids := func(id string) ([]string, []reportsBlock) {
		for _, s := range sections {
			if s.ID == id {
				out, blocks := []string{}, []reportsBlock{}
				for _, b := range s.Blocks {
					out = append(out, b.ID)
					blocks = append(blocks, reportsBlock{b.ID, b.Text, b.Table})
				}
				return out, blocks
			}
		}
		return nil, nil
	}
	got, blocks := ids("project")
	if strings.Join(got, ",") != "rfp:header,rfp:project:intro,rfp:project:grid,rfp:project:conditions,rfp:project:information" {
		t.Fatalf("page one blocks = %v", got)
	}
	grid := blocks[2].table
	if strings.Join(grid.Columns, "|") != "The building|Site and planning|Delivery" || grid.Rows[0][0] != "Work type: New build" {
		t.Fatalf("grid = %+v", grid)
	}
	// Every answer prints, one line per answer.
	cond := strings.Split(blocks[3].text, "\n")
	if len(cond) != 2 || !strings.HasPrefix(cond[0], "Conditions that do not apply: Work to an existing building") || !strings.HasPrefix(cond[1], "Conditions not yet known: ") || !strings.Contains(cond[1], "Groundwater above the base of excavation") {
		t.Fatalf("conditions = %q", blocks[3].text)
	}
	if blocks[4].text != "Information provided: Survey and geotechnical report" {
		t.Fatalf("information = %q", blocks[4].text)
	}
	role, roleBlocks := ids("role")
	if strings.Join(role, ",") != "rfp:role" || !strings.HasSuffix(roleBlocks[0].text, "Design vehicle: 19 m semi-trailer") {
		t.Fatalf("role = %v %+v", role, roleBlocks)
	}
}

type reportsBlock struct {
	id    string
	text  string
	table *reports.Table
}

func controlKeys(c Controls, group string) (string, []string) {
	for _, g := range c.Groups {
		if g.ID == group {
			keys := []string{}
			for _, ctl := range g.Controls {
				keys = append(keys, ctl.Key)
			}
			return g.Section, keys
		}
	}
	return "", nil
}

func TestControlsFollowThePage(t *testing.T) {
	sk := skeleton(t)
	for _, id := range []string{"architecture", "civil"} {
		d := discipline(t, id)
		r := Resolve(d, sk, facts(t, "work_type=new", "class=7b", "class=5", "flag.rail=yes"), Settings{})
		c := BuildControls(r, d, sk, Settings{})
		section, project := controlKeys(c, "project")
		joined := strings.Join(project, ",")
		if section != "project" || strings.Contains(joined, "parts:affected") || strings.Contains(joined, "design_vehicle") || strings.Contains(joined, "field:agreement") {
			t.Fatalf("%s project controls = %v", id, project)
		}
		flags := []string{}
		for _, k := range project {
			if strings.HasPrefix(k, "fact:flag.") {
				flags = append(flags, k)
			}
		}
		want := []string{}
		for _, cond := range r.Conditions {
			want = append(want, cond.Control)
		}
		if strings.Join(flags, ",") != strings.Join(want, ",") {
			t.Fatalf("%s condition controls = %v, want %v", id, flags, want)
		}
		section, appointment := controlKeys(c, "appointment")
		if section != "role" || appointment[0] != "parts:affected" || strings.Contains(strings.Join(appointment, ","), "design_vehicle") != (id == "civil") {
			t.Fatalf("%s appointment controls = %s %v", id, section, appointment)
		}
		if section, terms := controlKeys(c, "terms"); section != "terms" || !strings.Contains(strings.Join(terms, ","), "field:agreement") || strings.Contains(strings.Join(terms, ","), "field:ref") {
			t.Fatalf("%s terms controls = %s %v", id, section, terms)
		}
		if section, issue := controlKeys(c, "issue"); section != "issue" || !strings.Contains(strings.Join(issue, ","), "field:ref") || strings.Contains(strings.Join(issue, ","), "field:pi") {
			t.Fatalf("%s issue controls = %s %v", id, section, issue)
		}
	}
}

func TestMissingTermsAreTheirOwnCheck(t *testing.T) {
	r := Resolve(discipline(t, "architecture"), skeleton(t), facts(t, "work_type=new", "class=2"), Settings{})
	byID := map[string]Check{}
	for _, c := range r.Checks {
		byID[c.ID] = c
	}
	if c := byID["terms"]; !c.Blocking || c.Control != "field:agreement" {
		t.Fatalf("terms check = %+v", c)
	}
	if strings.Contains(byID["fields"].Text, "Client agreement") {
		t.Fatalf("issue details check names a term: %+v", byID["fields"])
	}
}
