package rfp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/reports"
	"sitewise/internal/reports/render"
)

// engagementFor resolves each discipline from one answer-key scenario, as
// the store resolves each member from its own settings and frozen drivers.
func engagementFor(t *testing.T, scenarioName, title string, ids ...string) Engagement {
	t.Helper()
	cat := loadCatalog(t)
	sk, _ := cat.RFPSkeleton()
	var sc *scenario
	for _, s := range loadScenarios(t) {
		if s.Name == scenarioName {
			sc = &s
			break
		}
	}
	if sc == nil {
		t.Fatalf("no scenario %s", scenarioName)
	}
	e := Engagement{Title: title}
	for _, id := range ids {
		d, ok := cat.RFP(id)
		if !ok {
			t.Fatalf("no discipline %s", id)
		}
		drivers := sc.drivers(t, sk, d)
		settings := sc.settings(t, id, drivers)
		e.Parts = append(e.Parts, GroupPart{PackageID: "pkg-" + id, PackageTitle: d.Title, DisciplineID: d.DisciplineIDs[0], Templated: true, Discipline: d, Resolved: Resolve(d, sk, drivers, settings), Settings: settings})
	}
	return e
}

func groupSections(t *testing.T, e Engagement) []reports.Section {
	t.Helper()
	out, err := GroupSections(e, consultantTemplate(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reports.Cite(out); err != nil {
		t.Fatalf("every block needs a unique anchor and an object basis: %v", err)
	}
	return out
}

func sectionByID(secs []reports.Section, id string) *reports.Section {
	for i := range secs {
		if secs[i].ID == id {
			return &secs[i]
		}
	}
	return nil
}

func blockByID(s *reports.Section, id string) *reports.Block {
	if s == nil {
		return nil
	}
	for i := range s.Blocks {
		if s.Blocks[i].ID == id {
			return &s.Blocks[i]
		}
	}
	return nil
}

func TestGroupDocumentOrderSharedPartsOnceThenEachMember(t *testing.T) {
	e := engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil")
	secs := groupSections(t, e)
	ids := []string{}
	for _, s := range secs {
		ids = append(ids, s.ID)
	}
	want := []string{"project", "scope", "part:pkg-structural", "part:pkg-civil", "fees", "transmittal", "terms", "returnables", "issue_dates", reports.ReviewSectionID}
	if strings.Join(ids, ",") != strings.Join(want, ",") {
		t.Fatalf("order %v, want %v", ids, want)
	}
	project := sectionByID(secs, "project")
	if project.Blocks[0].ID != "rfp:group:engagement" || !strings.Contains(project.Blocks[0].Text, "Structural and civil engineering") ||
		!strings.Contains(project.Blocks[0].Text, "Structural engineering services and Civil engineering services") || !strings.Contains(project.Blocks[0].Text, "subconsultant") {
		t.Fatalf("page one names the engagement and its disciplines: %q", project.Blocks[0].Text)
	}
	// Page one, the transmittal, terms, returnables and dates print once.
	count := map[string]int{}
	for _, s := range secs {
		for _, b := range s.Blocks {
			for _, id := range []string{"rfp:project:intro", "rfp:project:grid", "rfp:project:information", "rfp:terms", "rfp:returnables", "rfp:header"} {
				if b.ID == id {
					count[id]++
				}
			}
		}
	}
	for _, id := range []string{"rfp:project:intro", "rfp:project:grid", "rfp:project:information", "rfp:terms", "rfp:returnables", "rfp:header"} {
		if count[id] != 1 {
			t.Fatalf("%s printed %d times", id, count[id])
		}
	}
	// Each part carries its discipline's role and allowances.
	for _, p := range e.Parts {
		part := sectionByID(secs, PartSectionID(p.PackageID))
		if blockByID(part, part.ID+":rfp:role") == nil || blockByID(part, part.ID+":rfp:allowances") == nil || blockByID(part, part.ID+":rfp:stages") == nil {
			t.Fatalf("%s part misses its role, stage table or allowances", p.PackageID)
		}
	}
}

// Each member's part prints the same stages and lines as its own RFP,
// without the fee column and pricing rows, apart from items moved to
// another member.
func TestGroupPartMatchesStandaloneStageTable(t *testing.T) {
	e := engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil")
	secs := groupSections(t, e)
	moved, _ := e.within()
	for i, p := range e.Parts {
		alone, err := Sections(p.Resolved, consultantTemplate(t))
		if err != nil {
			t.Fatal(err)
		}
		standalone := blockByID(sectionByID(alone, "role"), "rfp:stages")
		var basis struct {
			StageRows int `json:"stage_rows"`
		}
		_ = json.Unmarshal(standalone.Basis, &basis)
		want := []string{}
		for _, row := range standalone.Table.Rows[:basis.StageRows] {
			skip := false
			for key := range moved[i] {
				for _, s := range p.Resolved.Scope {
					if s.Key == key && strings.HasPrefix(row[1], s.Label) {
						skip = true
					}
				}
			}
			if !skip {
				want = append(want, strings.Join(row[:3], "|"))
			}
		}
		part := blockByID(sectionByID(secs, PartSectionID(p.PackageID)), PartSectionID(p.PackageID)+":rfp:stages")
		got := []string{}
		for _, row := range part.Table.Rows {
			got = append(got, strings.Join(row, "|"))
		}
		if strings.Join(got, "\n") != strings.Join(want, "\n") {
			t.Fatalf("%s part table differs from its own RFP:\n%s\nwant\n%s", p.PackageID, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		if len(part.Table.Columns) != 3 || strings.Contains(part.Text, "Total fee") {
			t.Fatalf("a part prices nothing itself: %v", part.Table.Columns)
		}
	}
}

func TestGroupWithinEngagementItemsShownOnce(t *testing.T) {
	label := "Structural design of retaining walls"
	occurrences := func(secs []reports.Section, section string) int {
		n := 0
		for _, s := range secs {
			if section != "" && s.ID != section {
				continue
			}
			for _, b := range s.Blocks {
				n += strings.Count(b.Text, label)
				if b.Table != nil {
					for _, row := range b.Table.Rows {
						for _, cell := range row {
							n += strings.Count(cell, label)
						}
					}
				}
			}
		}
		return n
	}
	alone := engagementFor(t, "arthur-st", "Civil", "civil")
	if v := alone.Parts[0].Resolved.Position("retaining_structural").Value; v != "X" {
		t.Fatalf("fixture: civil excludes retaining wall design (got %q)", v)
	}
	e := engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil")
	secs := groupSections(t, e)
	if n := occurrences(secs, "part:pkg-civil"); n != 0 {
		t.Fatalf("civil still prints the structural engineer's item %d times", n)
	}
	within := blockByID(sectionByID(secs, "part:pkg-structural"), "part:pkg-structural:rfp:within")
	if within == nil || strings.Count(within.Text, label) != 1 || !strings.Contains(within.Text, "within this engagement") || !strings.Contains(within.Text, "Civil engineering services") {
		t.Fatalf("the item shows once under the structural engineer: %+v", within)
	}
	if n := occurrences(secs, ""); n != 1 {
		t.Fatalf("item printed %d times in the document", n)
	}
	// Without the structural engineer in the group, civil's wording stands.
	other := engagementFor(t, "arthur-st", "Civil and traffic", "civil", "traffic")
	secs = groupSections(t, other)
	if n := occurrences(secs, "part:pkg-civil"); n != 1 {
		t.Fatalf("civil wording changed when structural is not in the group: %d", n)
	}
	for _, s := range secs {
		for _, b := range s.Blocks {
			if strings.HasSuffix(b.ID, ":rfp:within") && strings.Contains(b.Text, label) {
				t.Fatalf("retaining walls moved with no structural engineer in the group: %s", b.ID)
			}
		}
	}
}

func TestGroupMemberForNamesOneMember(t *testing.T) {
	e := engagementFor(t, "electrical-office-design", "Services", "mechanical-services", "electrical-services", "hydraulic-services", "fire-services")
	for party, want := range map[string]int{
		"fire services consultant": 3, "fire-services designer": 3, "hydraulic engineer": 2, "electrical designer": 1,
		"fire-services or electrical designer": -1, "acoustic consultant / architect": -1, "fire engineer": -1, "client": -1, "": -1,
	} {
		if got := e.memberFor(party); got != want {
			t.Errorf("%q names member %d, want %d", party, got, want)
		}
	}
	la := engagementFor(t, "landscape-mixed-podium", "Design", "architecture", "landscape-architecture")
	if got := la.memberFor("landscape architect"); got != 1 {
		t.Errorf("the longest discipline word wins: %d", got)
	}
}

func TestGroupFeeScheduleSumsEachDisciplineAndTotal(t *testing.T) {
	e := engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil")
	secs := groupSections(t, e)
	fees := blockByID(sectionByID(secs, "fees"), "rfp:fees")
	if fees == nil || !strings.Contains(fees.Text, "Included in") || !strings.Contains(fees.Text, "combined") {
		t.Fatalf("fee schedule allows a combined stage: %+v", fees)
	}
	rows := fees.Table.Rows
	schedule := GroupSchedule(e)
	at := 0
	for i, f := range schedule {
		if rows[at][0] != e.Parts[i].Title() {
			t.Fatalf("row %d %q opens %s", at, rows[at][0], e.Parts[i].Title())
		}
		at++
		stages := 0
		for _, it := range Schedule(e.Parts[i].Resolved) {
			if it.Kind == ItemStage {
				stages++
				found := false
				for _, row := range rows {
					found = found || row[0] == it.Label
				}
				if !found {
					t.Fatalf("included stage %s missing from the fee schedule", it.Label)
				}
			}
		}
		if stages == 0 {
			t.Fatalf("%s has no priced stage", f.Title)
		}
		for at < len(rows) && rows[at][0] != "Sum for "+f.Title {
			at++
		}
		if at == len(rows) {
			t.Fatalf("no sum for %s", f.Title)
		}
		at++
		for at < len(rows) && strings.HasPrefix(rows[at][0], "Option: ") {
			at++
		}
	}
	if rows[len(rows)-1][0] != "Engagement total (the sums above)" || at != len(rows)-1 {
		t.Fatalf("the engagement total closes the schedule: %v", rows[len(rows)-1])
	}
}

func TestGroupChecksMergedAndLabelled(t *testing.T) {
	e := engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil")
	checks := GroupChecks(e)
	labelled := 0
	for _, c := range checks {
		switch {
		case strings.HasPrefix(c.ID, "pkg-structural:"):
			labelled++
			if !strings.HasPrefix(c.Label, "Structural engineering services: ") || c.Control != "" {
				t.Fatalf("member check %+v", c)
			}
		case strings.HasPrefix(c.ID, "pkg-civil:"):
			labelled++
			if !strings.HasPrefix(c.Label, "Civil engineering services: ") {
				t.Fatalf("member check %+v", c)
			}
		}
	}
	total := len(e.Parts[0].Resolved.Checks) + len(e.Parts[1].Resolved.Checks)
	if labelled == 0 || len(checks) > total {
		t.Fatalf("checks: %d labelled of %d (members raise %d)", labelled, len(checks), total)
	}
	// Every member's own checks are present, once each.
	for _, p := range e.Parts {
		for _, c := range p.Resolved.Checks {
			found := false
			for _, g := range checks {
				found = found || g.ID == c.ID || g.ID == p.PackageID+":"+c.ID
			}
			if !found {
				t.Fatalf("%s check %s dropped", p.PackageID, c.ID)
			}
		}
	}
	secs := groupSections(t, e)
	review := sectionByID(secs, reports.ReviewSectionID)
	reviews := 0
	for _, b := range review.Blocks {
		if b.ID == "part:pkg-civil:rfp:check:discipline_review" && strings.Contains(b.Label, "Civil engineering services: Rules await owner review") {
			reviews++
		}
	}
	if reviews != 1 {
		t.Fatal("each member's draft-rules notice is labelled with its discipline")
	}
	if len(GroupBlocking(e)) == 0 {
		t.Fatal("open decisions block the engagement")
	}
}

func TestGroupPartWithoutRFPRules(t *testing.T) {
	e := engagementFor(t, "arthur-st", "Structural and acoustic", "structural")
	e.Parts = append(e.Parts, GroupPart{PackageID: "pkg-acoustic", PackageTitle: "Acoustic consultant", DisciplineID: "consultant.acoustic"})
	secs := groupSections(t, e)
	part := sectionByID(secs, "part:pkg-acoustic")
	if part == nil || !strings.Contains(part.Blocks[0].Text, "Scope to be described") {
		t.Fatalf("the part says its scope is to be described: %+v", part)
	}
	blocks := false
	for _, c := range GroupBlocking(e) {
		blocks = blocks || c.ID == "pkg-acoustic:scope_described"
	}
	if !blocks {
		t.Fatal("a blocking check asks for the scope")
	}
	if fees := blockByID(sectionByID(secs, "fees"), "rfp:fees"); !strings.Contains(blockTableText(fees), "Sum for Acoustic consultant") {
		t.Fatal("the discipline still has a sum in the fee schedule")
	}
}

func blockTableText(b *reports.Block) string {
	if b == nil || b.Table == nil {
		return ""
	}
	out := []string{}
	for _, row := range b.Table.Rows {
		out = append(out, strings.Join(row, "|"))
	}
	return strings.Join(out, "\n")
}

func TestSharedSettingsComeFromTheGroup(t *testing.T) {
	sk := skeleton(t)
	member := Settings{"field:closes": {Value: json.RawMessage(`"2026-01-01"`)}, "stage:schematic": {Value: json.RawMessage(`"exclude"`)}, "transmittal": {Value: json.RawMessage(`[]`)}}
	group := Settings{"field:closes": {Value: json.RawMessage(`"2026-12-01"`)}, "stage:schematic": {Value: json.RawMessage(`"include"`)}}
	got := MergeSettings(sk, member, group)
	if string(got["field:closes"].Value) != `"2026-12-01"` || string(got["stage:schematic"].Value) != `"exclude"` {
		t.Fatalf("merged %v", got)
	}
	if _, ok := got["transmittal"]; ok {
		t.Fatal("a member's own transmittal is not the engagement's")
	}
	if SharedSetting(sk, "position:x") || !SharedSetting(sk, "field:issued") || !SharedSetting(sk, "transmittal") {
		t.Fatal("shared keys are the issue details, terms and the transmittal")
	}
}

// TestGoldenGroupRFPs snapshots two combined engagements: structural with
// civil, and four services disciplines.
func TestGoldenGroupRFPs(t *testing.T) {
	for _, g := range []struct {
		name, scenario, title string
		ids                   []string
	}{
		{"structural-civil", "arthur-st", "Structural and civil engineering", []string{"structural", "civil"}},
		{"services-four", "electrical-office-design", "Building services engineering", []string{"mechanical-services", "electrical-services", "hydraulic-services", "fire-services"}},
	} {
		t.Run(g.name, func(t *testing.T) {
			got := Markdown(groupSections(t, engagementFor(t, g.scenario, g.title, g.ids...)))
			path := filepath.Join("testdata", "groups", g.name+".md")
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(path)
			if err != nil || string(want) != got {
				t.Fatalf("snapshot mismatch for %s; run with -update and review it", path)
			}
		})
	}
}

// Both golden engagements fit their page allowance when issued.
func TestGroupRFPsFitTheirPageLimit(t *testing.T) {
	for _, e := range []Engagement{
		engagementFor(t, "arthur-st", "Structural and civil engineering", "structural", "civil"),
		engagementFor(t, "electrical-office-design", "Building services engineering", "mechanical-services", "electrical-services", "hydraulic-services", "fire-services"),
	} {
		secs, _, err := reports.Cite(groupSections(t, e))
		if err != nil {
			t.Fatal(err)
		}
		issued, refs, err := reports.IssueContent(secs, "rfp", false)
		if err != nil {
			t.Fatal(err)
		}
		snap := reports.IssueSnapshot{SchemaVersion: 1, ReportID: "r", VersionID: "v", Kind: "rfp", Title: "RFP", ReportingDate: "2026-10-12", SourceRevisions: reports.SourceState{TemplateID: "tpl.rfp-consultant", RFPParts: len(e.Parts)}, Sections: issued, References: refs, RendererVersion: render.Version}
		r, err := render.PDF(snap)
		t.Logf("%s: %d pages", e.Title, r.Pages)
		if err != nil {
			t.Fatalf("%s: %d pages: %v", e.Title, r.Pages, err)
		}
	}
}
