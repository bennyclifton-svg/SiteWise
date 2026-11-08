package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/rfp"
	"sitewise/internal/store"
)

// groupRFP is a combined engagement with its RFP assembled.
type groupRFP struct {
	s       *store.Store
	group   procurement.Package
	members []procurement.Package
	report  store.Report
	draft   store.ReportDraft
}

var groupLabels = map[string]string{"consultant.structural": "Structural engineer", "consultant.civil": "Civil engineer", "consultant.hydraulic": "Hydraulic engineer", "consultant.electrical": "Electrical engineer", "consultant.mechanical": "Mechanical engineer", "consultant.fire_services": "Fire services engineer", "consultant.acoustic": "Acoustic consultant"}

// groupRFPFixture sets the profile facts the consultant RFPs read, makes a
// row for each discipline, combines them and assembles the group's RFP.
func groupRFPFixture(t *testing.T, s *store.Store, b store.ProfileBuild, part string, disciplines ...string) groupRFP {
	t.Helper()
	ctx := context.Background()
	for key, value := range map[string]string{"hdr.work_type": "extend", "hdr.cond.planning": "da", "hdr.cond.procurement_route": "traditional", "det.state": "NSW", "det.ncc_class": "7b",
		"proj.client": "Client", "proj.regulated_design": "no", "proj.flag.staged": "no", "proj.flag.multi_building": "no", "proj.budget_stated": "$5 million excluding GST"} {
		v := value
		scope := b.Catalog.KeyScope(key)
		if scope == "" {
			scope = "project"
		}
		if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, key, store.UserWrite{Value: &v, State: "set", Scope: scope}); err != nil {
			t.Fatal(key, err)
		}
	}
	site, err := s.ProjectSite(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	address := "1 Test St, Sydney NSW"
	if _, err := s.UpdateSite(ctx, orgA, site.ID, site.Version, nil, &address, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	g := groupRFP{s: s}
	ids := []string{}
	for _, d := range disciplines {
		row, ok := procurement.DisciplinePackage(d, groupLabels[d])
		if !ok {
			t.Fatal("unknown discipline", d)
		}
		p, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
		if err != nil {
			t.Fatal(err)
		}
		g.members = append(g.members, p)
		ids = append(ids, p.ID)
	}
	if g.group, err = s.CombinePackages(ctx, orgA, projectA, userA, ids, ""); err != nil {
		t.Fatal(err)
	}
	if _, g.report, _, err = s.ProcurementCreate(ctx, orgA, projectA, userA, g.group.ID, "", ""); err != nil {
		t.Fatalf("Create RFP on a group: %v", err)
	}
	if g.draft, err = s.RefreshReport(ctx, orgA, g.report.ID, userA, "test-build", true); err != nil {
		t.Fatalf("assemble the group's RFP: %v", err)
	}
	return g
}

func (g groupRFP) view(t *testing.T) store.ReportView {
	t.Helper()
	v, err := g.s.ReadReport(context.Background(), orgA, g.report.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	if v.RFP == nil || v.RFPGroup == nil {
		t.Fatalf("the group's view has shared and member controls: %+v", v.Stale)
	}
	return v
}

func (g groupRFP) member(t *testing.T, v store.ReportView, pkg string) store.RFPGroupMember {
	t.Helper()
	for _, m := range v.RFPGroup.Members {
		if m.PackageID == pkg {
			return m
		}
	}
	t.Fatalf("no member %s", pkg)
	return store.RFPGroupMember{}
}

func sectionIDs(d store.ReportDraft) []string {
	out := []string{}
	for _, s := range d.Sections {
		out = append(out, s.ID)
	}
	return out
}

func draftBlock(d store.ReportDraft, id string) *reports.Block {
	for i := range d.Sections {
		for j := range d.Sections[i].Blocks {
			if d.Sections[i].Blocks[j].ID == id {
				return &d.Sections[i].Blocks[j]
			}
		}
	}
	return nil
}

// partService is a service's position in a member's part of the group draft.
func partService(d store.ReportDraft, pkg, key string) string {
	b := draftBlock(d, rfp.PartSectionID(pkg)+":rfp:stages")
	if b == nil {
		return ""
	}
	var basis struct {
		Services []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"services"`
	}
	_ = json.Unmarshal(b.Basis, &basis)
	for _, s := range basis.Services {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

func TestGroupRFPAssemblesEveryMemberInOrder(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	g := groupRFPFixture(t, s, b, part, "consultant.structural", "consultant.civil")
	// Procurement order is by name: civil before structural.
	civil, structural := g.members[1], g.members[0]
	want := []string{"project", "scope", rfp.PartSectionID(civil.ID), rfp.PartSectionID(structural.ID), "fees", "transmittal", "terms", "returnables", "issue_dates", reports.ReviewSectionID}
	if strings.Join(sectionIDs(g.draft), ",") != strings.Join(want, ",") {
		t.Fatalf("sections %v, want %v", sectionIDs(g.draft), want)
	}
	if g.draft.TemplateID != "tpl.rfp-consultant" || g.draft.SourceRevisions.RFPParts != 2 {
		t.Fatalf("group draft %s with %d parts", g.draft.TemplateID, g.draft.SourceRevisions.RFPParts)
	}
	if text := draftBlock(g.draft, "rfp:group:engagement").Text; !strings.Contains(text, "engagement, "+g.group.Title+",") || !strings.Contains(text, "Civil engineering services and Structural engineering services") {
		t.Fatalf("page one names the engagement and its disciplines: %q", text)
	}
	// Each member has its own report and draft, and its part prints the
	// same stage rows as that draft, less its fee column and pricing rows.
	v := g.view(t)
	for _, m := range g.members {
		gm := g.member(t, v, m.ID)
		own, err := s.ReadReport(ctx, orgA, gm.ReportID, "test-build")
		if err != nil || own.Draft == nil || own.Report.PackageID != m.ID {
			t.Fatalf("member %s report: %v", m.Title, err)
		}
		alone := draftBlock(*own.Draft, "rfp:stages")
		inGroup := draftBlock(g.draft, rfp.PartSectionID(m.ID)+":rfp:stages")
		var basis struct {
			StageRows int `json:"stage_rows"`
		}
		_ = json.Unmarshal(alone.Basis, &basis)
		got, wantRows := []string{}, []string{}
		for _, row := range inGroup.Table.Rows {
			got = append(got, strings.Join(row, "|"))
		}
		for _, row := range alone.Table.Rows[:basis.StageRows] {
			wantRows = append(wantRows, strings.Join(row[:3], "|"))
		}
		if len(got) == 0 || !containsInOrder(wantRows, got) {
			t.Fatalf("%s part rows are not its own RFP's:\n%s\nvs\n%s", m.Title, strings.Join(got, "\n"), strings.Join(wantRows, "\n"))
		}
	}
	// Two clicks never make two group reports.
	if _, again, created, err := s.ProcurementCreate(ctx, orgA, projectA, userA, g.group.ID, "", ""); err != nil || created || again.ID != g.report.ID {
		t.Fatalf("second Create: %v created=%v", err, created)
	}
}

// containsInOrder: every row of got is in want, in want's order; a row
// moved to another member is the only kind missing.
func containsInOrder(want, got []string) bool {
	i := 0
	for _, row := range got {
		for i < len(want) && want[i] != row {
			i++
		}
		if i == len(want) {
			return false
		}
		i++
	}
	return true
}

func TestGroupRFPSettingsWriteToTheMemberAndTheGroup(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	g := groupRFPFixture(t, s, b, part, "consultant.structural", "consultant.civil")
	structural := g.members[0]
	v := g.view(t)
	m := g.member(t, v, structural.ID)
	key := ""
	for _, grp := range m.Controls.Groups {
		for _, c := range grp.Controls {
			if strings.HasPrefix(c.Key, "position:") && c.Value == "I" && key == "" {
				key = c.Key
			}
		}
	}
	if key == "" {
		t.Fatal("fixture: structural has an included service")
	}
	service := strings.TrimPrefix(key, "position:")
	if partService(g.draft, structural.ID, service) != "I" {
		t.Fatal("fixture: the part shows the service included")
	}
	// A member's choice is written to its own report; the group's draft is
	// recomposed in the same transaction.
	d, group, err := s.SetReportSettingInGroup(ctx, orgA, m.ReportID, userA, key, json.RawMessage(`"X"`), m.DraftVersion)
	if err != nil || group != g.report.ID {
		t.Fatalf("member setting: %v group=%q", err, group)
	}
	if d.ReportID != m.ReportID {
		t.Fatal("the member's own draft is returned")
	}
	v = g.view(t)
	if partService(*v.Draft, structural.ID, service) != "X" || v.Draft.Version <= g.draft.Version {
		t.Fatal("the group's draft shows the member's choice")
	}
	if c := rfpControl(g.member(t, v, structural.ID).Controls, key); !c.Set {
		t.Fatal("the choice is stored on the member's report")
	}
	// A stale member version is refused, like any setting.
	if _, _, err := s.SetReportSettingInGroup(ctx, orgA, m.ReportID, userA, key, json.RawMessage(`"I"`), m.DraftVersion); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale member version: %v", err)
	}
	// A discipline's choice is not the group's to store.
	if _, _, err := s.SetReportSettingInGroup(ctx, orgA, g.report.ID, userA, key, json.RawMessage(`"I"`), v.Draft.Version); !errors.Is(err, store.ErrInvalidSetting) {
		t.Fatalf("member key on the group: %v", err)
	}
	// The issue details are the engagement's, stored once on the group.
	d, group, err = s.SetReportSettingInGroup(ctx, orgA, g.report.ID, userA, "field:closes", json.RawMessage(`"2026-12-11"`), v.Draft.Version)
	if err != nil || group != g.report.ID || !strings.Contains(draftBlock(d, "rfp:header").Text, "2026-12-11") {
		t.Fatalf("shared setting: %v %+v", err, draftBlock(d, "rfp:header"))
	}
	own, err := s.ReadReport(ctx, orgA, m.ReportID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	if c := rfpControl(own.RFP, "field:closes"); c.Set {
		t.Fatal("a shared setting is not copied to members")
	}
	// Wrong org and wrong project are refused.
	if _, _, err := s.SetReportSettingInGroup(ctx, orgB, m.ReportID, userB, key, json.RawMessage(`"I"`), m.DraftVersion+1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other org's setting: %v", err)
	}
	if _, err := s.RefreshReport(ctx, orgB, g.report.ID, userB, "test-build", true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other org's refresh: %v", err)
	}
	if _, err := s.ReadReport(ctx, orgB, g.report.ID, "test-build"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("other org's read: %v", err)
	}
	if _, _, _, err := s.ProcurementCreate(ctx, orgA, projectB, userA, g.group.ID, "", ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another project's group: %v", err)
	}
}

func TestGroupRFPChecksAndResponseForm(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	g := groupRFPFixture(t, s, b, part, "consultant.structural", "consultant.civil", "consultant.acoustic")
	v := g.view(t)
	labelled := map[string]bool{}
	for _, c := range v.RFP.Checks {
		for _, title := range []string{"Structural engineering services: ", "Civil engineering services: ", "Acoustic consultant: "} {
			if strings.HasPrefix(c.Label, title) {
				labelled[title] = true
			}
		}
	}
	// The general consultant RFP raises no checks beyond the shared ones.
	if len(labelled) != 2 || labelled["Acoustic consultant: "] {
		t.Fatalf("every member's own checks are labelled: %v", labelled)
	}
	// A consultant with no discipline file of its own takes the general
	// consultant RFP (owner, 8 November 2026), named by its package.
	acoustic := g.member(t, v, g.members[2].ID)
	if acoustic.Controls == nil || !strings.Contains(draftBlock(*v.Draft, rfp.PartSectionID(g.members[2].ID)+":rfp:role").Text, "Your job is to give the project") {
		t.Fatalf("acoustic uses the general consultant RFP: %+v", acoustic)
	}
	form, err := s.ReadResponseForm(ctx, orgA, g.report.ID)
	if err != nil || len(form.Fees) != 3 || len(form.Sheets()) < 2 {
		t.Fatalf("response form: %v %d sections", err, len(form.Fees))
	}
}

func TestGroupRFPIssuesAsOneAndUncombineKeepsMembers(t *testing.T) {
	ctx := context.Background()
	base, _, part := workStore(t)
	cat := reviewedCatalog(t, "structural", "civil")
	b := profileBuildFor(cat)
	s := base.WithProfile(b)
	g := groupRFPFixture(t, s, b, part, "consultant.structural", "consultant.civil")
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	issue := func(version int64) error {
		_, err := s.IssueReport(ctx, orgA, g.report.ID, userA, "test-build", store.IssueOptions{Version: version, ReportingDate: "2026-11-07", AcceptStale: true, StaleReason: "fixture"}, blobs)
		return err
	}
	if err := issue(g.draft.Version); !errors.Is(err, store.ErrRFPOpenChecks) {
		t.Fatalf("open decisions block the engagement's issue: %v", err)
	}
	// Settle the shared issue details once, then each member's decisions.
	v := g.view(t)
	version := v.Draft.Version
	for key, value := range map[string]string{"field:ref": `"RFP-7"`, "field:issued": `"2026-11-07"`, "field:closes": `"2026-11-28"`, "field:agreement": `"AS 4122-2010 as amended"`} {
		d, _, err := s.SetReportSettingInGroup(ctx, orgA, g.report.ID, userA, key, json.RawMessage(value), version)
		if err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		version = d.Version
	}
	v = settleMembers(t, s, g)
	if len(v.RFP.Decisions) > 0 {
		t.Fatalf("decisions still open: %+v", v.RFP.Decisions)
	}
	issued, err := s.IssueReport(ctx, orgA, g.report.ID, userA, "test-build", store.IssueOptions{Version: v.Draft.Version, ReportingDate: "2026-11-07", AcceptStale: true, StaleReason: "fixture"}, blobs)
	if err != nil {
		t.Fatalf("one action issues the whole engagement: %v", err)
	}
	if issued.Snapshot.SourceRevisions.RFPParts != 2 || len(issued.ExportSHA256) == 0 {
		t.Fatal("the issue is one PDF of every part")
	}
	parts := 0
	for _, sec := range issued.Snapshot.Sections {
		if strings.HasPrefix(sec.ID, "part:") {
			parts++
		}
		if sec.Internal {
			t.Fatal("Before issue never reaches the issued document")
		}
	}
	if parts != 2 {
		t.Fatalf("issued %d parts", parts)
	}
	// Members' own RFPs were not issued, and still cannot be while grouped.
	before := map[string]store.ReportView{}
	for _, m := range v.RFPGroup.Members {
		own, err := s.ReadReport(ctx, orgA, m.ReportID, "test-build")
		if err != nil || len(own.Issues) != 0 || own.Draft == nil {
			t.Fatalf("member report after group issue: %v", err)
		}
		before[m.ReportID] = own
		if _, err := s.IssueReport(ctx, orgA, m.ReportID, userA, "test-build", store.IssueOptions{Version: own.Draft.Version, ReportingDate: "2026-11-07", AcceptStale: true, StaleReason: "fixture"}, blobs); !errors.Is(err, store.ErrReportPackageGrouped) {
			t.Fatalf("member issue while grouped: %v", err)
		}
	}
	// Uncombining keeps every member report, draft and setting as it was.
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, g.group.ID); err != nil {
		t.Fatal(err)
	}
	for id, was := range before {
		now, err := s.ReadReport(ctx, orgA, id, "test-build")
		if err != nil || now.Draft == nil || now.Draft.ID != was.Draft.ID || now.Draft.Version != was.Draft.Version || countSet(now.RFP) != countSet(was.RFP) {
			t.Fatalf("member report changed on uncombine: %v", err)
		}
	}
	gv, err := s.ReadReport(ctx, orgA, g.report.ID, "test-build")
	if err != nil || len(gv.Issues) != 1 {
		t.Fatalf("the issued engagement stays in history: %v", err)
	}
}

func countSet(c *rfp.Controls) int {
	n := 0
	if c == nil {
		return -1
	}
	for _, g := range c.Groups {
		for _, ctl := range g.Controls {
			if ctl.Set {
				n++
			}
		}
	}
	return n
}

// settleMembers answers each member's open decisions with the value its
// control offers first, through the member's own report.
func settleMembers(t *testing.T, s *store.Store, g groupRFP) store.ReportView {
	t.Helper()
	ctx := context.Background()
	for round := 0; round < 20; round++ {
		v := g.view(t)
		changed := false
		for _, m := range v.RFPGroup.Members {
			if m.Controls == nil {
				continue
			}
			for _, d := range m.Controls.Decisions {
				key, value := d.Control, ""
				c := rfpControl(m.Controls, key)
				switch {
				case strings.HasPrefix(key, "allowance:"):
					value = `"2"`
				case strings.HasPrefix(key, "position:") || strings.HasPrefix(key, "stage:") || strings.HasPrefix(key, "element:"):
					value = `"exclude"`
					if strings.HasPrefix(key, "position:") {
						value = `"X"`
					}
				case strings.HasPrefix(key, "responsible:"):
					value = `"client"`
				case key == "boundary:range" && len(m.Controls.Boundary.Stages) > 0:
					raw, _ := json.Marshal(map[string]string{"start": m.Controls.Boundary.Stages[0].Value, "end": m.Controls.Boundary.Stages[len(m.Controls.Boundary.Stages)-1].Value})
					value = string(raw)
				case len(c.Options) > 0:
					raw, _ := json.Marshal(c.Options[0].Value)
					value = string(raw)
				default:
					continue
				}
				if _, _, err := s.SetReportSettingInGroup(ctx, orgA, m.ReportID, userA, key, json.RawMessage(value), m.DraftVersion); err != nil {
					t.Fatalf("%s %s=%s: %v", m.Title, key, value, err)
				}
				changed = true
				break
			}
			if changed {
				break
			}
		}
		if !changed {
			return v
		}
	}
	return g.view(t)
}

func TestGroupRFPMemberLeavesAndIssuedHistoryStays(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	// Electrical's own RFP is issued before it is combined: a stand-in for an
	// issue, since its draft rules cannot be issued.
	elec, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, mustRow(t, "consultant.electrical", "Electrical engineer"))
	if err != nil {
		t.Fatal(err)
	}
	_, own, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, elec.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, own.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	m, err := s.ReadReport(ctx, orgA, own.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	key := ""
	for _, grp := range m.RFP.Groups {
		for _, c := range grp.Controls {
			if strings.HasPrefix(c.Key, "position:") && c.Value == "I" && key == "" {
				key = c.Key
			}
		}
	}
	d = setSetting(t, s, own, d, key, `"X"`)
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE report_versions SET status='issued',issued_at=now(),issued_by=$3,snapshot='{}'::jsonb,snapshot_sha256=repeat('a',64),export_file_sha256=decode(repeat('ab',32),'hex') WHERE org_id=$1 AND id=$2`, orgA, d.ID, userA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE reports SET current_draft_version_id=NULL WHERE org_id=$1 AND id=$2`, orgA, own.ID); err != nil {
		t.Fatal(err)
	}
	// The group adopts the existing rows: electrical plus two new ones.
	g := groupRFPFixtureWith(t, s, b, part, []string{elec.ID}, "consultant.hydraulic", "consultant.mechanical")
	electrical := elec.ID
	if partService(g.draft, electrical, strings.TrimPrefix(key, "position:")) != "X" {
		t.Fatal("a member's issued choices carry into the group's RFP")
	}
	after, err := s.ReadReport(ctx, orgA, own.ID, "test-build")
	if err != nil || len(after.Issues) != 1 || after.Draft == nil {
		t.Fatalf("its issued history stays and a new draft holds its choices: %v", err)
	}
	// A member removed while the draft is open leaves at the next build,
	// and the editor says so meanwhile.
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, g.members[len(g.members)-1].ID); err != nil {
		t.Fatal(err)
	}
	v := g.view(t)
	if len(v.RFPGroup.Left) != 1 || v.RFPGroup.Left[0] != g.members[len(g.members)-1].Title {
		t.Fatalf("left: %v", v.RFPGroup.Left)
	}
	next, err := s.RefreshReport(ctx, orgA, g.report.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range sectionIDs(next) {
		if id == rfp.PartSectionID(g.members[len(g.members)-1].ID) {
			t.Fatal("the next build drops the part of a member that left")
		}
	}
	if next.SourceRevisions.RFPParts != 2 {
		t.Fatalf("parts after leaving: %d", next.SourceRevisions.RFPParts)
	}
}

// groupRFPFixtureWith is groupRFPFixture that also combines existing rows.
func groupRFPFixtureWith(t *testing.T, s *store.Store, b store.ProfileBuild, part string, existing []string, disciplines ...string) groupRFP {
	t.Helper()
	ctx := context.Background()
	g := groupRFP{s: s}
	ids := append([]string{}, existing...)
	for _, id := range existing {
		g.members = append(g.members, packageByID(t, s, id))
	}
	for _, d := range disciplines {
		p, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, mustRow(t, d, groupLabels[d]))
		if err != nil {
			t.Fatal(err)
		}
		g.members = append(g.members, p)
		ids = append(ids, p.ID)
	}
	var err error
	if g.group, err = s.CombinePackages(ctx, orgA, projectA, userA, ids, ""); err != nil {
		t.Fatal(err)
	}
	if _, g.report, _, err = s.ProcurementCreate(ctx, orgA, projectA, userA, g.group.ID, "", ""); err != nil {
		t.Fatal(err)
	}
	if g.draft, err = s.RefreshReport(ctx, orgA, g.report.ID, userA, "test-build", true); err != nil {
		t.Fatal(err)
	}
	return g
}
