package rfp

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"reflect"
	"sitewise/internal/knowledge"
	"sort"
	"strings"
	"testing"
)

// This is the pre-rollout inventory, not a snapshot regenerated from current rules.
func TestConsultantScopeInventoryPreserved(t *testing.T) {
	raw, err := os.ReadFile("testdata/scope-alignment-inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var baseline map[string]knowledge.RFPDiscipline
	if err = yaml.Unmarshal(raw, &baseline); err != nil {
		t.Fatal(err)
	}
	if len(baseline) != 18 {
		t.Fatal("incomplete discipline inventory")
	}
	normalize := func(v any) string {
		b, _ := json.Marshal(v)
		s := string(b)
		for _, pair := range [][2]string{{"section 5", "section 2"}, {"Section 5", "Section 2"}, {"section 6", "section 5"}, {"sections 3–6", "sections 2–5"}, {"sections 3-6", "sections 2-5"}, {"WSC", "Water Services Coordinator"}} {
			s = strings.ReplaceAll(s, pair[0], pair[1])
		}
		return s
	}
	ids := []string{}
	for id := range baseline {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	ledger := []string{"# Consultant RFP coverage ledger", "", "Every entry below is checked against the pre-rollout inventory. Role wording remains in Scope of services; allocation notes sit beside their service. Stage deliverables and fee bases remain in the combined pricing schedule. Inputs retain their original setting keys and Profile homes. Inapplicable scope remains selectable in the editor.", ""}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			old, d, sk := baseline[id], discipline(t, id), skeleton(t)
			ledger = append(ledger, "## "+d.Title, "", "| Original item | Destination |", "| --- | --- |")
			for _, pair := range []struct {
				name          string
				before, after any
			}{{"role", old.Role, d.Role}, {"scope schedule", old.ScopeSchedule, d.ScopeSchedule}, {"stages", old.Stages, d.Stages}, {"elements", old.Elements, d.Elements}} {
				if normalize(pair.before) != normalize(pair.after) {
					t.Errorf("%s definitions changed or lost", pair.name)
				}
			}
			if len(old.Positions) != len(d.Positions) {
				t.Fatal("position count changed")
			}
			for i, p := range old.Positions {
				q := d.Positions[i]
				if q.Group == "" {
					t.Errorf("ungrouped scope %s", q.Key)
				}
				q.Group = p.Group
				if normalize(p) != normalize(q) {
					t.Errorf("scope %s content/rules changed", p.Key)
				}
				destination := "Scope of services / " + d.Positions[i].Group
				if id == "hydraulic-services" && hydraulicStageService(p.Key) {
					destination = "Linked stage in Scope by stage; saved position overrides retained"
				}
				ledger = append(ledger, fmt.Sprintf("| `position:%s` — %s | %s |", p.Key, strings.ReplaceAll(p.Label, "|", "/"), destination))
			}
			for _, row := range old.Role {
				ledger = append(ledger, "| Role `"+row.Key+"` | Scope of services introduction (`rfp:role`) |")
			}
			for _, row := range old.ScopeSchedule {
				ledger = append(ledger, "| Responsibility `"+row.Key+"` | Scope of services / `position:"+scopeNoteTarget(id, row.Key)+"` |")
			}
			for _, row := range old.Stages {
				ledger = append(ledger, "| `stage:"+row.Key+"` | Scope by stage and fee proposal |")
			}
			for _, row := range old.Elements {
				ledger = append(ledger, "| `element:"+row.Key+"` | Scope of services; stage-by-element fee rows |")
			}
			drivers := facts(t, "class=5", "work_type=new", "pathway=da", "procurement=traditional")
			r := Resolve(d, sk, drivers, nil)
			c := BuildControls(r, d, sk, nil)
			if len(r.Scope) != len(old.Positions) {
				t.Fatal("editor lost inactive scope")
			}
			all := map[string]Control{}
			for _, g := range c.Groups {
				for _, ctl := range g.Controls {
					all[ctl.Key] = ctl
				}
			}
			used := usedDrivers(old, sk)
			inputKeys := []string{"parts:affected", "scale"}
			for _, f := range sk.Flags {
				if f.ScopeDecision && used["flag."+f.Key] {
					inputKeys = append(inputKeys, "fact:flag."+f.Key)
				}
			}
			for _, f := range sk.Fields {
				if f.Group == "appointment" && usesField(old, f.Key) {
					prefix := "field:"
					if f.Profile != "" {
						prefix = "fact:"
					}
					inputKeys = append(inputKeys, prefix+f.Key)
				}
			}
			for _, key := range inputKeys {
				ctl, ok := all[key]
				if !ok {
					t.Errorf("old role input lost: %s", key)
				}
				if key != "parts:affected" && ctl.Target == "" {
					t.Errorf("no inline home: %s", key)
				}
				dest := ctl.Target
				if key == "parts:affected" {
					dest = "Project"
				}
				ledger = append(ledger, "| Input `"+key+"` — "+ctl.Label+" | `"+dest+"` |")
			}
			sections, err := Sections(r, consultantTemplate(t))
			if err != nil {
				t.Fatal(err)
			}
			text := Markdown(sections)
			for _, obsolete := range []string{"Your role", "This appointment buys", "Contact for questions", "Questions in writing by", "Confirm you can start by", "field:contact", "field:questions_by"} {
				if strings.Contains(text, obsolete) {
					t.Errorf("obsolete template wording remains: %s", obsolete)
				}
			}
			for _, row := range r.Role {
				if !strings.Contains(text, row.Text) {
					t.Errorf("resolved role lost: %s", row.Key)
				}
			}
			for _, row := range r.ScopeSchedule {
				if !strings.Contains(text, row.Text) {
					t.Errorf("resolved allocation lost: %s", row.Key)
				}
			}
			for _, s := range sections {
				if s.ID == "role" || s.ID == "evaluation" {
					t.Errorf("old section %s", s.ID)
				}
				for _, b := range s.Blocks {
					if b.ID == "rfp:returnables:r1" {
						t.Fatal("duplicate fee schedule")
					}
				}
			}
			if sections[0].ID != "issue_dates" {
				t.Fatal("dates not first")
			}
			_, issue := controlKeys(c, "issue")
			if !reflect.DeepEqual(issue, []string{"field:issued", "field:closes"}) {
				t.Errorf("issue fields %v", issue)
			}
			for _, p := range old.Positions {
				for _, value := range []string{"I", "O", "X", Ask} {
					key := "position:" + p.Key
					settings := set(key, fmt.Sprintf("%q", value))
					if err := ValidateSetting(d, sk, drivers, key, settings[key].Value); err != nil {
						t.Fatal(err)
					}
					rr := Resolve(d, sk, drivers, settings)
					if rr.Position(p.Key).Value != value {
						t.Errorf("%s override %s lost", key, value)
					}
					if value == Ask && !rr.HasCheck("scope:"+p.Key) {
						t.Errorf("unknown %s does not block issue", key)
					}
				}
			}
			ledger = append(ledger, "")
		})
	}
	if out := os.Getenv("SITEWISE_SCOPE_LEDGER"); out != "" && !t.Failed() {
		if err := os.WriteFile(out, []byte(strings.Join(ledger, "\n")), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCivilElementUnknownAndOptionalFeeRows(t *testing.T) {
	d, sk := discipline(t, "civil"), skeleton(t)
	drivers := facts(t, "class=5", "work_type=new", "pathway=da", "procurement=traditional")
	key := d.Elements[0].Key
	settings := set("element:"+key, `"ask"`)
	if err := ValidateSetting(d, sk, drivers, "element:"+key, settings["element:"+key].Value); err != nil {
		t.Fatal(err)
	}
	r := Resolve(d, sk, drivers, settings)
	if !r.HasCheck("scope-element:" + key) {
		t.Fatal("unknown element must block issue")
	}
	settings["element:"+key] = set("x", `"option"`)["x"]
	r = Resolve(d, sk, drivers, settings)
	sections, err := Sections(r, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, sec := range sections {
		for _, block := range sec.Blocks {
			if block.ID != "rfp:stages" {
				continue
			}
			for _, row := range block.Table.Rows {
				if strings.HasPrefix(row[0], d.Elements[0].Code+" —") {
					count++
					if row[0] != d.Elements[0].Code+" — Option" {
						t.Fatal("optional element leaked into base fee")
					}
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("element pricing disappeared")
	}
}
