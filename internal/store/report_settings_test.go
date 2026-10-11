package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
	"sitewise/internal/rfp"
	"sitewise/internal/store"
)

// rfpProject sets the profile facts an architect RFP reads and returns a
// draft for an Architect package with a confirmed discipline.
func rfpProject(t *testing.T, s *store.Store, b store.ProfileBuild, part string) (store.Report, store.ReportDraft) {
	t.Helper()
	ctx := context.Background()
	// Project facts live in Profile (D1), including those an RFP requires.
	for key, value := range map[string]string{"hdr.work_type": "extend", "hdr.cond.planning": "da", "hdr.cond.procurement_route": "traditional", "det.state": "NSW", "det.ncc_class": "7b",
		"proj.client": "Client", "proj.regulated_design": "no", "proj.flag.staged": "no", "proj.flag.multi_building": "no"} {
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
	p, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", DisciplineID: "consultant.architect", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Civil", LifecycleStatus: "proposed"}); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, "rfp", p.ID)
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	return r, d
}

func blockText(d store.ReportDraft, id string) string {
	for _, section := range d.Sections {
		for _, b := range section.Blocks {
			if b.ID == id {
				return b.Text
			}
		}
	}
	return ""
}

func setSetting(t *testing.T, s *store.Store, r store.Report, d store.ReportDraft, key, value string) store.ReportDraft {
	t.Helper()
	next, err := s.SetReportSetting(context.Background(), orgA, r.ID, userA, key, json.RawMessage(value), d.Version)
	if err != nil {
		t.Fatalf("%s=%s: %v", key, value, err)
	}
	return next
}

func TestConsultantRFPDraftAndSettings(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	r, d := rfpProject(t, s, b, part)
	if d.TemplateID != "tpl.rfp-consultant" || len(d.SourceRevisions.RFP) == 0 {
		t.Fatalf("a confirmed architect package uses the consultant RFP with frozen drivers: %s", d.TemplateID)
	}
	// The generic RFP's brief, work, proposal and gap blocks never reach a
	// discipline RFP; the first section a consultant reads is the project.
	visible := []string{}
	for _, section := range d.Sections {
		if !section.Internal {
			visible = append(visible, section.Title)
		}
		for _, b := range section.Blocks {
			for _, prefix := range []string{"work:", "proposal:", "gap:", "brief:", "clause:"} {
				if strings.HasPrefix(b.ID, prefix) {
					t.Fatalf("generic block %s in a discipline RFP", b.ID)
				}
			}
		}
	}
	if len(visible) < 2 || visible[0] != "" || visible[1] != "1. Project" {
		t.Fatalf("first visible section %v", visible)
	}
	if !strings.Contains(blockText(d, "rfp:role"), "Civil (to be appointed)") {
		t.Fatalf("other consultants carry their real status: %q", blockText(d, "rfp:role"))
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil || view.RFP == nil || view.PageFit == nil || view.PageFit.Limit != 4 {
		t.Fatalf("controls and the four-page limit: %+v %v", view.PageFit, err)
	}
	if hasCheck(view.RFP.Decisions, "boundary", true) {
		t.Fatalf("default stage selections need no second confirmation: %+v", view.RFP.Decisions)
	}
	frozen, _ := json.Marshal(d.SourceRevisions)

	next := setSetting(t, s, r, d, "position:bim_coordination", `"X"`)
	if scopeValue(next, "bim_coordination") != "X" {
		t.Fatalf("setting re-resolves the draft: %q", blockText(next, "rfp:positions"))
	}
	after, _ := json.Marshal(next.SourceRevisions)
	if string(frozen) != string(after) {
		t.Fatal("a setting change must not reload the project")
	}
	if _, err := s.SetReportSetting(ctx, orgA, r.ID, userA, "position:bim_coordination", json.RawMessage(`"I"`), d.Version); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("stale version: %v", err)
	}
	if _, err := s.SetReportSetting(ctx, orgB, r.ID, userB, "position:bim_coordination", json.RawMessage(`"I"`), next.Version); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's report: %v", err)
	}
	for key, value := range map[string]string{"position:no_such_item": `"I"`, "position:bim_coordination": `"Y"`, "allowance:design_meetings": `"lots"`, "field:closes": `"next week"`, "weight:price": `140`, "driver:flag.flood": `"maybe"`, "boundary:range": `{"start":"for_construction","end":"concept"}`} {
		if _, err := s.SetReportSetting(ctx, orgA, r.ID, userA, key, json.RawMessage(value), next.Version); !errors.Is(err, store.ErrInvalidSetting) {
			t.Fatalf("%s=%s must be refused: %v", key, value, err)
		}
	}
	if _, err := s.SetReportSetting(ctx, orgA, r.ID, userA, "scale", nil, next.Version); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("deleting a setting that does not exist: %v", err)
	}

	// A setting survives refresh.
	refreshed, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	if scopeValue(refreshed, "bim_coordination") != "X" {
		t.Fatal("a setting survives refresh")
	}

	// A project fact is Profile's: a change there reaches the RFP on refresh.
	existing := "stated_false"
	if _, err := s.SetUserValue(ctx, orgA, projectA, part, userA, "det.existing_building", store.UserWrite{Value: &existing, State: "set", Scope: scopeOf(b.Catalog, "det.existing_building")}); err != nil {
		t.Fatal(err)
	}
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true); err != nil {
		t.Fatal(err)
	}
	view, err = s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	if c := rfpControl(view.RFP, "fact:flag.existing_building"); c.Value != "no" || c.Source != "profile" || c.Home != rfp.HomeDeterminant || c.Locked {
		t.Fatalf("existing building follows Profile and saves there: %+v", c)
	}

	// Deleting a setting restores the rule.
	draft := setSetting(t, s, r, *view.Draft, "position:consultant_selection", `"X"`)
	if scopeValue(draft, "consultant_selection") != "X" {
		t.Fatal("the override excludes the item")
	}
	restored, err := s.SetReportSetting(ctx, orgA, r.ID, userA, "position:consultant_selection", nil, draft.Version)
	if err != nil || scopeValue(restored, "consultant_selection") != "O" {
		t.Fatalf("delete restores the rule: %v", err)
	}
}

func rfpControl(c *rfp.Controls, key string) rfp.Control {
	for _, g := range c.Groups {
		for _, ctl := range g.Controls {
			if ctl.Key == key {
				return ctl
			}
		}
	}
	return rfp.Control{}
}

func scopeOf(cat *knowledge.Catalog, key string) string {
	if s := cat.KeyScope(key); s != "" {
		return s
	}
	return "project"
}

func hasCheck(checks []rfp.Check, id string, blocking bool) bool {
	for _, c := range checks {
		if c.ID == id && c.Blocking == blocking {
			return true
		}
	}
	return false
}

// finishRFP clears every blocking check an architect RFP for rfpProject has.
func finishRFP(t *testing.T, s *store.Store, r store.Report, d store.ReportDraft) store.ReportDraft {
	t.Helper()
	for key, value := range map[string]string{
		"boundary:range":                 `{"start":"brief_masterplan","end":"for_construction"}`,
		"field:ref":                      `"RFP-1"`,
		"field:issued":                   `"2026-10-12"`,
		"field:closes":                   `"2026-10-30"`,
		"field:contact":                  `"pm@example.com"`,
		"field:questions_by":             `"2026-10-23"`,
		"field:agreement":                `"AS 4122-2010 as amended"`,
		"allowance:client_presentations": `"2"`,
		"allowance:planning_cap":         `"20"`,
		"allowance:ca_inspections":       `"6"`,
		"parts:affected":                 "",
	} {
		if value == "" {
			continue
		}
		d = setSetting(t, s, r, d, key, value)
	}
	return d
}

func TestConsultantRFPIssueGates(t *testing.T) {
	ctx := context.Background()
	s, b, part := workStore(t)
	r, d := rfpProject(t, s, b, part)
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	issue := func(d store.ReportDraft) error {
		_, err := s.IssueReport(ctx, orgA, r.ID, userA, "test-build", store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-12", AcceptStale: true, StaleReason: "fixture"}, blobs)
		return err
	}
	if err := issue(d); !errors.Is(err, store.ErrRFPOpenChecks) {
		t.Fatalf("open decisions block issue: %v", err)
	}
	d = finishRFP(t, s, r, d)
	view, err := s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.RFP.Decisions) != 0 {
		t.Fatalf("fixture still has decisions: %+v", view.RFP.Decisions)
	}
	if err := issue(*view.Draft); !errors.Is(err, store.ErrReportUnreviewed) {
		t.Fatalf("draft discipline rules block issue: %v", err)
	}
}

// reviewedCatalog copies knowledge/ and marks the architecture rules
// reviewed, as the owner would, without touching the real files.
func reviewedCatalog(t *testing.T) *knowledge.Catalog {
	t.Helper()
	dst := t.TempDir()
	src := filepath.Join("..", "..", "knowledge")
	err := filepath.WalkDir(src, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if e.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if filepath.ToSlash(rel) == "rfp/architecture.yaml" {
			body = []byte(strings.Replace(strings.ReplaceAll(string(body), "\r\n", "\n"), "\nstatus: draft\n", "\nstatus: reviewed\nreviewed_by: test owner\nreviewed_on: \"2026-10-10\"\n", 1))
		}
		return os.WriteFile(target, body, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := knowledge.Load(dst)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestConsultantRFPIssueFreezesAndCarriesForward(t *testing.T) {
	ctx := context.Background()
	base, _, part := workStore(t)
	cat := reviewedCatalog(t)
	b := store.ProfileBuild{Catalog: cat, KnowledgeVersion: cat.Version(), QuestionVersion: profile.QuestionVersion, Compute: func(s store.ProfileSnapshot) []profile.Row {
		return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: profile.Thresholds{Amber: map[string]float64{"presence": .6, "header": .6, "action": .8}}}, cat)
	}}
	s := base.WithProfile(b)
	r, d := rfpProject(t, s, b, part)
	d = finishRFP(t, s, r, d)
	d = setSetting(t, s, r, d, "position:bim_coordination", `"X"`)
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadReport(ctx, orgA, r.ID, "test-build")
	if err != nil {
		t.Fatal(err)
	}
	issued, err := s.IssueReport(ctx, orgA, r.ID, userA, "test-build", store.IssueOptions{Version: view.Draft.Version, ReportingDate: "2026-10-12", AcceptStale: true, StaleReason: "fixture"}, blobs)
	if err != nil {
		t.Fatalf("a reviewed discipline with no open decision issues: %v", err)
	}
	if len(issued.Snapshot.SourceRevisions.RFP) == 0 || issued.Snapshot.SourceRevisions.TemplateID != "tpl.rfp-consultant" {
		t.Fatal("the issue freezes the drivers and rules it was resolved from")
	}
	for _, section := range issued.Snapshot.Sections {
		if section.Internal {
			t.Fatal("Before issue never reaches the issued RFP")
		}
		for _, block := range section.Blocks {
			if block.Provisional || block.ID == "rfp:budget" {
				t.Fatalf("issued block %s", block.ID)
			}
		}
	}
	// An issued version's settings are immutable.
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `UPDATE report_settings SET value='"I"' WHERE org_id=$1::uuid AND report_version_id=$2::uuid`, orgA, issued.ID); err == nil {
		t.Fatal("issued settings changed")
	}
	next, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test-build", true)
	if err != nil {
		t.Fatal(err)
	}
	if next.ID == issued.ID || scopeValue(next, "bim_coordination") != "X" {
		t.Fatal("settings carry into the next draft after issue")
	}
}

// Scope selections live in the single table's metadata, independently of wording.
func scopeValue(d store.ReportDraft, key string) string {
	for _, section := range d.Sections {
		for _, block := range section.Blocks {
			if block.ID != "rfp:positions" {
				continue
			}
			var basis struct {
				Positions []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"positions"`
			}
			_ = json.Unmarshal(block.Basis, &basis)
			for _, p := range basis.Positions {
				if p.Key == key {
					return p.Value
				}
			}
		}
	}
	return ""
}
