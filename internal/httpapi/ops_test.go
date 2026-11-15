package httpapi_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"sitewise/internal/costs"
	psample "sitewise/internal/proposals/sample"
	tsample "sitewise/internal/tender/sample"
)

type opsReply struct {
	ChangeID string `json:"change_id"`
	Undo     struct {
		Kind string `json:"kind"`
	} `json:"undo"`
	Results []struct {
		Op      string `json:"op"`
		Summary string `json:"summary"`
	} `json:"results"`
	// A refusal.
	Index      int    `json:"op_index"`
	Field      string `json:"field"`
	Message    string `json:"message"`
	NeedsClick bool   `json:"needs_click"`
}

func op(name string, fields map[string]any) map[string]any {
	return map[string]any{"op": name, "fields": fields}
}

// ops sends a batch and returns the reply; want is the status expected.
func (m *member) ops(t *testing.T, project, area string, want int, calls ...map[string]any) opsReply {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"area": area, "ops": calls})
	var out opsReply
	if want != http.StatusOK && want != http.StatusUnprocessableEntity {
		m.call(t, http.MethodPost, "/api/projects/"+project+"/ops", body, want, nil)
		return out
	}
	m.call(t, http.MethodPost, "/api/projects/"+project+"/ops", body, want, &out)
	return out
}

// withoutVersions is a read with every version counter removed, so a read
// before a change and a read after its undo compare equal.
func withoutVersions(t *testing.T, m *member, path string) any {
	t.Helper()
	var v any
	m.getJSON(t, path, &v)
	var strip func(any) any
	strip = func(x any) any {
		switch n := x.(type) {
		case map[string]any:
			for _, k := range []string{"version", "revision", "provenance", "totals"} {
				delete(n, k)
			}
			for k, c := range n {
				n[k] = strip(c)
			}
		case []any:
			for i, c := range n {
				n[i] = strip(c)
			}
		}
		return x
	}
	return strip(v)
}

func (m *member) undo(t *testing.T, project, change string, want int) {
	t.Helper()
	m.call(t, http.MethodPost, "/api/projects/"+project+"/changes/"+change+"/undo", []byte(`{}`), want, nil)
}

func planOf(t *testing.T, m *member, project string) costs.Plan {
	t.Helper()
	var p costs.Plan
	m.getJSON(t, "/api/projects/"+project+"/cost-plan", &p)
	return p
}

func lineID(t *testing.T, p costs.Plan, label string) string {
	t.Helper()
	for _, i := range p.Items {
		if i.Label == label {
			return i.ID
		}
	}
	t.Fatalf("no line %q among %d", label, len(p.Items))
	return ""
}

func TestOpsApplyBatchAndUndo(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Operations")
	planPath := "/api/projects/" + project + "/cost-plan"
	// A first line, so the plan exists and "before" is a real plan.
	m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.add_line", map[string]any{"label": "Escalation", "section": "contingency", "category": "escalation", "amount": "20000"}))
	before := withoutVersions(t, m, planPath)

	out := m.ops(t, project, "cost_plan", http.StatusOK,
		op("cost_plan.add_line", map[string]any{"label": "Joinery", "section": "construction", "system": "interiors", "amount": "$45k"}),
		op("cost_plan.add_line", map[string]any{"label": "DA fees", "section": "authority"}))
	if out.ChangeID == "" || out.Undo.Kind != "change" || len(out.Results) != 2 || !strings.Contains(out.Results[0].Summary, "$45,000") || !strings.Contains(out.Results[1].Summary, "amount unknown") {
		t.Fatalf("reply %+v", out)
	}
	p := planOf(t, m, project)
	joinery := p.Items[0]
	for _, i := range p.Items {
		if i.Label == "Joinery" {
			joinery = i
		}
	}
	if len(p.Items) != 3 || joinery.SystemID != "interiors" || joinery.Values["budget"].Amount == nil || costs.Compare(*joinery.Values["budget"].Amount, "45000") != 0 {
		t.Fatalf("plan after: %d lines, joinery %+v", len(p.Items), joinery)
	}
	m.undo(t, project, out.ChangeID, http.StatusOK)
	if after := withoutVersions(t, m, planPath); !reflect.DeepEqual(before, after) {
		t.Fatalf("undo did not restore the plan:\n%v\n%v", before, after)
	}
	m.undo(t, project, out.ChangeID, http.StatusConflict)

	// Refusals write nothing.
	var calls51 []map[string]any
	for i := 0; i < 51; i++ {
		calls51 = append(calls51, op("cost_plan.add_line", map[string]any{"label": fmt.Sprint("L", i), "section": "authority"}))
	}
	for name, calls := range map[string][]map[string]any{
		"unknown operation": {op("cost_plan.paint", nil)},
		"another area":      {op("profile.set_value", map[string]any{"key": "proj.zone", "value": "R3"})},
		"empty":             {},
		"too many":          calls51,
		"unknown field":     {op("cost_plan.add_line", map[string]any{"label": "X", "section": "authority", "colour": "red"})},
		"bad amount":        {op("cost_plan.add_line", map[string]any{"label": "X", "section": "authority", "amount": "about forty"})},
		"no system":         {op("cost_plan.add_line", map[string]any{"label": "X", "section": "construction"})},
	} {
		if r := m.ops(t, project, "cost_plan", http.StatusUnprocessableEntity, calls...); r.Message == "" || r.NeedsClick {
			t.Errorf("%s: reply %+v", name, r)
		}
	}
	// Several lines for the same place are one operation with labels.
	pair := m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.add_line", map[string]any{"labels": []string{"Stone", "Tapware"}, "section": "construction", "system": "interiors"}))
	if len(planOf(t, m, project).Items) != 3 || !strings.Contains(pair.Results[0].Summary, `"Stone", "Tapware"`) {
		t.Fatalf("labels: %+v", pair)
	}
	m.undo(t, project, pair.ChangeID, http.StatusOK)
	// A Click operation is not applied by this call: it waits for the PM.
	if r := m.ops(t, project, "cost_plan", http.StatusUnprocessableEntity, op("cost_plan.freeze", nil)); !r.NeedsClick {
		t.Errorf("freeze: %+v", r)
	}
	// The second operation is refused: the first is put back.
	r := m.ops(t, project, "cost_plan", http.StatusUnprocessableEntity,
		op("cost_plan.add_line", map[string]any{"label": "Stone", "section": "construction", "system": "interiors"}),
		op("cost_plan.set_amount", map[string]any{"line": "00000000", "amount": "5"}))
	if r.Index != 1 || r.Message == "" {
		t.Errorf("refused batch: %+v", r)
	}
	if after := withoutVersions(t, m, planPath); !reflect.DeepEqual(before, after) {
		t.Fatalf("a refused batch left something behind:\n%v\n%v", before, after)
	}
	// Another org cannot reach the project.
	other := a.member(t, newUUID(t))
	other.ops(t, project, "cost_plan", http.StatusNotFound, op("cost_plan.add_line", map[string]any{"label": "X", "section": "authority"}))
}

// Each Cost Plan operation shows on the plan and is put back by undo.
func TestCostOperationsApplyThenUndo(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Cost operations")
	planPath := "/api/projects/" + project + "/cost-plan"
	m.ops(t, project, "cost_plan", http.StatusOK,
		op("cost_plan.add_line", map[string]any{"label": "Landscaping", "section": "construction", "system": "site", "amount": "60000"}),
		op("cost_plan.add_line", map[string]any{"label": "Contingency", "section": "contingency", "amount": "90000"}),
		op("cost_plan.add_line", map[string]any{"label": "DA fees", "section": "authority", "amount": "8400"}))
	p := planOf(t, m, project)
	short := func(label string) string { return lineID(t, p, label)[:8] }
	find := func(label string) *costs.Item {
		for _, i := range planOf(t, m, project).Items {
			if i.Label == label {
				return &i
			}
		}
		return nil
	}
	cases := []struct {
		name  string
		call  map[string]any
		check func() bool
	}{
		{"edit_line", op("cost_plan.edit_line", map[string]any{"line": short("DA fees"), "label": "Council fees"}),
			func() bool { return find("Council fees") != nil && find("DA fees") == nil }},
		{"set_amount", op("cost_plan.set_amount", map[string]any{"line": short("Landscaping"), "amount": "72,500"}),
			func() bool { return costs.Compare(*find("Landscaping").Values["budget"].Amount, "72500") == 0 }},
		{"set_amount unknown", op("cost_plan.set_amount", map[string]any{"line": short("Landscaping"), "unknown": true}),
			func() bool { return find("Landscaping").Values["budget"].Amount == nil }},
		{"set contract", op("cost_plan.set_amount", map[string]any{"line": short("Landscaping"), "metric": "contract", "amount": "58000"}),
			func() bool { return find("Landscaping").Values["commitment"].Amount != nil }},
		{"set_lines", op("cost_plan.set_lines", map[string]any{"lines": []string{short("Landscaping")}, "excluded": true}),
			func() bool { return find("Landscaping").Excluded }},
		{"duplicate_lines", op("cost_plan.duplicate_lines", map[string]any{"lines": []string{short("Contingency")}}),
			func() bool { return find("Contingency (copy)") != nil }},
		{"delete_lines", op("cost_plan.delete_lines", map[string]any{"lines": []string{short("Contingency")}}),
			func() bool { return find("Contingency") == nil }},
	}
	for _, c := range cases {
		before := withoutVersions(t, m, planPath)
		out := m.ops(t, project, "cost_plan", http.StatusOK, c.call)
		if !c.check() {
			t.Fatalf("%s: the plan does not show the edit (%+v)", c.name, out)
		}
		m.undo(t, project, out.ChangeID, http.StatusOK)
		if after := withoutVersions(t, m, planPath); !reflect.DeepEqual(before, after) {
			t.Fatalf("%s: undo did not restore the plan:\n%v\n%v", c.name, before, after)
		}
	}
	// A delete that touches a contract amount, or empties the plan, waits
	// for a click and deletes nothing.
	m.ops(t, project, "cost_plan", http.StatusOK, op("cost_plan.set_amount", map[string]any{"line": short("Landscaping"), "metric": "contract", "amount": "58000"}))
	before := withoutVersions(t, m, planPath)
	if r := m.ops(t, project, "cost_plan", http.StatusUnprocessableEntity, op("cost_plan.delete_lines", map[string]any{"lines": []string{short("Landscaping")}})); !r.NeedsClick || !strings.Contains(r.Message, "contract") {
		t.Fatalf("delete of a contract line: %+v", r)
	}
	if r := m.ops(t, project, "cost_plan", http.StatusUnprocessableEntity, op("cost_plan.delete_lines", map[string]any{"lines": []string{short("Contingency"), short("DA fees"), short("Landscaping")}})); !r.NeedsClick {
		t.Fatalf("delete of every line: %+v", r)
	}
	if after := withoutVersions(t, m, planPath); !reflect.DeepEqual(before, after) {
		t.Fatal("a held delete changed the plan")
	}
}

func TestProfileOperationApplyThenUndo(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Profile operations")
	path := "/api/projects/" + project + "/profile"
	valueOf := func(key string) string {
		var v struct {
			Details []struct {
				Key   string  `json:"key"`
				Value *string `json:"value"`
			} `json:"details"`
		}
		m.getJSON(t, path, &v)
		for _, d := range v.Details {
			if d.Key == key && d.Value != nil {
				return *d.Value
			}
		}
		return ""
	}
	if valueOf("proj.zone") != "" {
		t.Fatal("zone should start blank")
	}
	out := m.ops(t, project, "profile", http.StatusOK,
		op("profile.set_value", map[string]any{"key": "proj.zone", "value": "R3"}),
		op("profile.set_value", map[string]any{"key": "plan.site_area", "value": "612"}))
	if out.ChangeID == "" || !strings.Contains(out.Results[0].Summary, "Zone") || !strings.Contains(out.Results[1].Summary, "612") {
		t.Fatalf("reply %+v", out)
	}
	if valueOf("proj.zone") != "R3" {
		t.Fatalf("zone = %q", valueOf("proj.zone"))
	}
	m.undo(t, project, out.ChangeID, http.StatusOK)
	if valueOf("proj.zone") != "" {
		t.Fatalf("undo left zone = %q", valueOf("proj.zone"))
	}
	// A reset, then undo, restores the user's value.
	m.ops(t, project, "profile", http.StatusOK, op("profile.set_value", map[string]any{"key": "proj.zone", "value": "R4"}))
	reset := m.ops(t, project, "profile", http.StatusOK, op("profile.set_value", map[string]any{"key": "proj.zone", "reset": true}))
	if valueOf("proj.zone") != "" {
		t.Fatalf("after reset zone = %q", valueOf("proj.zone"))
	}
	m.undo(t, project, reset.ChangeID, http.StatusOK)
	if valueOf("proj.zone") != "R4" {
		t.Fatalf("after undoing the reset zone = %q", valueOf("proj.zone"))
	}
	for name, fields := range map[string]map[string]any{
		"unknown key":    {"key": "proj.widget_count", "value": "4"},
		"not a number":   {"key": "plan.site_area", "value": "big"},
		"two of a kind":  {"key": "proj.zone", "value": "R3", "reset": true},
		"nothing to set": {"key": "proj.zone"},
	} {
		if r := m.ops(t, project, "profile", http.StatusUnprocessableEntity, op("profile.set_value", fields)); r.Message == "" {
			t.Errorf("%s: %+v", name, r)
		}
	}
}

func TestProgramOperationPassesThrough(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Program operations")
	base := "/api/projects/" + project + "/program"
	m.call(t, http.MethodPost, base+"/prepare", []byte(`{"surface":"program"}`), http.StatusOK, nil)
	var view programView
	m.getJSON(t, base, &view)
	stage := ""
	for _, act := range view.Activities {
		if act.Kind == "stage" && stage == "" {
			stage = act.ID
		}
	}
	if stage == "" {
		t.Fatal("the prepared program has no stage")
	}
	before := len(view.Activities)
	var calls []map[string]any
	for i := 1; i <= 5; i++ {
		calls = append(calls, op("program.edit", map[string]any{"op": "add", "parent_id": stage[:8], "kind": "activity", "label": fmt.Sprint("DA drawings ", i), "duration": "two weeks"}))
	}
	out := m.ops(t, project, "program", http.StatusOK, calls...)
	if out.Undo.Kind != "program" || len(out.Results) != 5 || !strings.Contains(out.Results[0].Summary, "10 working days") {
		t.Fatalf("reply %+v", out)
	}
	m.getJSON(t, base, &view)
	added := 0
	for _, act := range view.Activities {
		if strings.HasPrefix(act.Label, "DA drawings ") && act.DurationDays != nil && *act.DurationDays == 10 && string(act.ParentID) == stage {
			added++
		}
	}
	if added != 5 {
		t.Fatalf("%d of five rows of ten days under the stage", added)
	}
	// Program's own undo removes all five in one step.
	body, _ := json.Marshal(map[string]any{"version": view.Program.Version})
	m.call(t, http.MethodPost, base+"/undo", body, http.StatusOK, nil)
	m.getJSON(t, base, &view)
	if len(view.Activities) != before {
		t.Fatalf("after undo %d rows, want %d", len(view.Activities), before)
	}
	// Several rows alike are one edit with labels.
	many := m.ops(t, project, "program", http.StatusOK, op("program.edit", map[string]any{"op": "add", "parent_id": stage, "kind": "activity", "labels": []string{"Basement", "Podium", "Tower"}, "duration": "8 weeks"}))
	if len(many.Results) != 3 || !strings.Contains(many.Results[2].Summary, `"Tower"`) || !strings.Contains(many.Results[2].Summary, "40 working days") {
		t.Fatalf("labels: %+v", many)
	}
	// A refused batch names the edit, in Program's own words.
	r := m.ops(t, project, "program", http.StatusUnprocessableEntity,
		op("program.edit", map[string]any{"op": "add", "parent_id": stage, "kind": "activity", "label": "Fine", "duration": "1 week"}),
		op("program.edit", map[string]any{"op": "add", "parent_id": stage, "kind": "activity", "label": "", "duration": "1 week"}))
	if r.Index != 1 || r.Message == "" {
		t.Fatalf("refusal %+v", r)
	}
	if r := m.ops(t, project, "program", http.StatusUnprocessableEntity, op("program.edit", map[string]any{"op": "add", "parent_id": stage, "label": "X", "duration": "a while"})); r.Field != "duration" {
		t.Fatalf("vague duration: %+v", r)
	}
	// Releasing a hold point is the PM's to click.
	if r := m.ops(t, project, "program", http.StatusUnprocessableEntity, op("program.release_hold_point", map[string]any{"id": stage})); !r.NeedsClick {
		t.Fatalf("release: %+v", r)
	}
}

func TestTenderOperationPassesThrough(t *testing.T) {
	_, m, project, base, labels := tenderSetup(t, withTender(tsample.Model{}, tinyRender))
	m.modelSwitch(t, true, http.StatusOK)
	sub := m.dropTender(t, base, "Invented Planning Pty Ltd", labels, psample.Line{Label: "Zebra crossing study", Cents: 50000})
	table := m.waitTender(t, base, sub, "read")
	pkg := base[strings.LastIndex(base, "/")+1:]
	out := m.ops(t, project, "procurement", http.StatusOK,
		op("procurement.tender", map[string]any{"package": pkg[:8], "op": "add_adjustment", "label": "Rock excavation excluded"}))
	if out.ChangeID == "" || out.Undo.Kind != "change" {
		t.Fatalf("reply %+v", out)
	}
	var after tenderTableBody
	m.getJSON(t, base+"/prices", &after)
	if len(after.Tender.Adjustments) != len(table.Tender.Adjustments)+1 {
		t.Fatalf("adjustments %d, want one more than %d", len(after.Tender.Adjustments), len(table.Tender.Adjustments))
	}
	m.undo(t, project, out.ChangeID, http.StatusOK)
	m.getJSON(t, base+"/prices", &after)
	if len(after.Tender.Adjustments) != len(table.Tender.Adjustments) {
		t.Fatal("undo left the adjustment")
	}
	// Checking a column is the PM's to click.
	yes := true
	if r := m.ops(t, project, "procurement", http.StatusUnprocessableEntity, op("procurement.tender", map[string]any{"package": pkg, "op": "check_column", "submission": sub, "checked": yes})); !r.NeedsClick {
		t.Fatalf("check: %+v", r)
	}
}
