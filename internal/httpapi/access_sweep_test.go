package httpapi_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"sitewise/internal/delivery"
	"sitewise/internal/httpapi"
	"sitewise/internal/procurement"
	"sitewise/internal/profile"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// WP-X1: valid requests must fail at the ownership boundary, not merely at
// parsing. Project and site references must belong to the addressed project.
func TestNextWaveRouteAccessSweep(t *testing.T) {
	ctx := context.Background()
	var build store.ProfileBuild
	a := newApp(t, withProfile(t), func(o *httpapi.Options) {
		build = store.ProfileBuild{Catalog: o.Knowledge, KnowledgeVersion: o.Knowledge.Version(), Compute: func(s store.ProfileSnapshot) []profile.Row {
			return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Planning: s.Planning, Thresholds: o.ProfileThresholds, Read: o.ProfileReading}, o.Knowledge)
		}}
	})
	m, foreign := a.member(t, newUUID(t)), a.member(t, newUUID(t))
	project := m.createProject(t, "Private sweep project")
	sibling := m.createProject(t, "Different site intervention")
	var site store.Site
	m.getJSON(t, "/api/projects/"+project+"/site", &site)
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	// V1 permits one project per site. Verify the guard rather than dropping
	// it to fabricate a shared-site fixture the product cannot currently hold.
	if _, err := conn.Exec(ctx, `UPDATE projects SET site_id=$3::uuid WHERE org_id=$1::uuid AND id=$2::uuid`, m.orgID, sibling, site.ID); err != nil {
		pgErr, ok := err.(*pgconn.PgError)
		if !ok || pgErr.Code != "23505" || pgErr.ConstraintName != "projects_one_per_site_v1" {
			t.Fatal(err)
		}
	} else {
		t.Fatal("V1 allowed a second project on the same site")
	}
	for _, id := range []string{project, sibling} {
		m.getJSON(t, "/api/projects/"+id+"/profile", new(profileBody))
	}
	var p profileBody
	m.getJSON(t, "/api/projects/"+project+"/profile", &p)
	part := p.Parts[0].ID
	base := "/api/projects/" + project
	jsonBody := func(v any) []byte {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	var work works.Item
	workBody := jsonBody(map[string]any{"title": "Private plant", "part_id": part, "system_id": "mechanical.air-conditioning", "action": "new"})
	m.call(t, "POST", base+"/works", workBody, 201, &work)
	var pkg procurement.Package
	pkgBody := []byte(`{"kind":"services","title":"Private engineering"}`)
	m.call(t, "POST", base+"/packages", pkgBody, 201, &pkg)
	pkgPath := base + "/packages/" + pkg.ID
	var scope store.ScopeItem
	scopeBody := jsonBody(map[string]any{"item_kind": "responsibility", "work_item_id": work.ID, "role": "design", "user_text": "Private scope", "inclusion": "included"})
	m.call(t, "POST", pkgPath+"/scope", scopeBody, 201, &scope)
	var item delivery.Item
	deliveryBody := jsonBody(map[string]any{"kind": "approval", "title": "Private approval", "package_id": pkg.ID, "work_item_id": work.ID})
	m.call(t, "POST", base+"/delivery", deliveryBody, 201, &item)
	evaluator, err := works.NewEvaluator(build.Catalog)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.store.WithProfile(build).RebuildProposals(ctx, m.orgID, project, evaluator); err != nil {
		t.Fatal(err)
	}
	proposals, err := a.store.ReadProposals(ctx, m.orgID, project)
	if err != nil || len(proposals) == 0 {
		t.Fatalf("proposals: %v", err)
	}
	proposal := proposals[0]
	proposalPath := base + "/proposals/" + url.PathEscape(proposal.Key)
	decisionBody := jsonBody(map[string]any{"inputs_fingerprint": proposal.InputsFingerprint})
	var decision store.ProposalDecisionView
	m.call(t, "POST", proposalPath+"/dismiss", decisionBody, 200, &decision)
	var report store.Report
	reportBody := jsonBody(map[string]any{"kind": "rfp", "package_id": pkg.ID})
	m.call(t, "POST", base+"/reports", reportBody, 201, &report)
	reportPath := "/api/reports/" + report.ID
	var draft store.ReportDraft
	m.call(t, "POST", reportPath+"/draft", []byte(`{"use_last_completed":true}`), 200, &draft)
	editPath := reportPath + "/edits/" + url.PathEscape("package:"+pkg.ID)
	m.call(t, "PUT", editPath, jsonBody(map[string]any{"version": draft.Version, "text": "Private protected wording"}), 200, &draft)
	putPlanning(t, m, project, "gross_floor_area", map[string]any{"value": 1200, "unit": "m2", "version": 0}, 200, nil)
	// These records are valid individually; only their attempted combination
	// below is invalid. No upload/background judgement is needed for isolation.
	makeDocument := func(actor *member, projectID string) string {
		t.Helper()
		id, fileID := newUUID(t), newUUID(t)
		sum := sha256.Sum256([]byte(id))
		if err := a.store.CreateFile(ctx, actor.orgID, store.File{ID: fileID, ProjectID: projectID, SHA256: sum[:], ByteSize: 1, MediaType: "application/pdf"}); err != nil {
			t.Fatal(err)
		}
		if err := a.store.CreateDocument(ctx, actor.orgID, store.Document{ID: id, ProjectID: projectID, FileID: fileID, Filename: "Private source.pdf", Status: "filed"}); err != nil {
			t.Fatal(err)
		}
		if err := a.store.ReplaceSource(ctx, actor.orgID, id, store.DocumentSource{Pages: 1, Source: []store.SourcePage{{Page: 1, Text: "Private source evidence"}}, Units: []store.SourceUnit{{Category: "paragraph", Body: "Private source evidence", Page: 1}}}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	doc := makeDocument(m, project)
	otherDoc := makeDocument(m, sibling)
	foreignProject := foreign.createProject(t, "Private foreign project")
	foreignDoc := makeDocument(foreign, foreignProject)
	var otherPkg procurement.Package
	m.call(t, "POST", base+"/packages", pkgBody, 201, &otherPkg)
	var siblingProfile profileBody
	m.getJSON(t, "/api/projects/"+sibling+"/profile", &siblingProfile)
	var siblingWork works.Item
	m.call(t, "POST", "/api/projects/"+sibling+"/works", jsonBody(map[string]any{"title": "Private sibling work", "part_id": siblingProfile.Parts[0].ID, "system_id": "mechanical.air-conditioning", "action": "new"}), 201, &siblingWork)
	// Capture authoritative rows, including versions, histories, projections
	// and events. Checking just the returned status would miss partial writes.
	snapshot := func() map[string]string {
		out := map[string]string{}
		for _, table := range []string{"projects", "sites", "project_parts", "profile_user_values", "profile_planning_values", "profile_builds", "profile_rows", "project_revisions", "work_items", "packages", "package_stages", "package_scope_items", "project_delivery_items", "proposals", "proposal_decisions", "reports", "report_versions", "report_edits", "report_references", "events", "documents", "files", "passages", "passage_sources", "document_sources", "profile_facts", "passage_calls", "passage_evidence", "passage_systems", "jobs"} {
			var raw []byte
			err := conn.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text),'[]'::jsonb) FROM `+pgx.Identifier{table}.Sanitize()+` r WHERE org_id=ANY($1::uuid[])`, []string{m.orgID, foreign.orgID}).Scan(&raw)
			if err != nil {
				t.Fatal(table, err)
			}
			out[table] = string(raw)
		}
		return out
	}
	before := snapshot()
	type route struct {
		method, path  string
		body          []byte
		foreignTarget bool
	}
	routes := []route{
		{"PUT", base + "/documents/profile-read", jsonBody(map[string]any{"document_ids": []string{doc}, "setting": "skip"}), true},
		{"POST", base + "/documents/delete", jsonBody(map[string]any{"document_ids": []string{doc}}), true},
		{"GET", base + "/works", nil, false}, {"POST", base + "/works", workBody, true}, {"PATCH", base + "/works/" + work.ID, []byte(`{"version":1,"title":"Intrusion"}`), true},
		{"GET", base + "/gaps", nil, false},
		{"GET", base + "/packages", nil, false}, {"POST", base + "/packages", pkgBody, false}, {"PATCH", pkgPath, []byte(`{"version":1,"title":"Intrusion"}`), true},
		{"POST", pkgPath + "/stages", []byte(`{"stage_id":"completion","label":"Intrusion","ordinal":8}`), true}, {"PATCH", pkgPath + "/stages/" + pkg.Stages[0].ID, []byte(`{"version":1,"label":"Intrusion"}`), true},
		{"GET", pkgPath + "/scope", nil, true}, {"POST", pkgPath + "/scope", scopeBody, true}, {"PATCH", pkgPath + "/scope/" + scope.ID, []byte(`{"version":1,"user_text":"Intrusion"}`), true},
		{"GET", base + "/delivery", nil, false}, {"POST", base + "/delivery", deliveryBody, true}, {"PATCH", base + "/delivery/" + item.ID, []byte(`{"version":1,"status":"approved"}`), true},
		{"GET", base + "/proposals", nil, false}, {"POST", proposalPath + "/accept", decisionBody, true}, {"POST", proposalPath + "/dismiss", decisionBody, true}, {"DELETE", proposalPath + "/decision", jsonBody(map[string]any{"version": decision.Version}), true},
		{"GET", base + "/reports", nil, false}, {"POST", base + "/reports", reportBody, true}, {"GET", reportPath, nil, false}, {"POST", reportPath + "/draft", []byte(`{"use_last_completed":true}`), false},
		{"PUT", editPath, jsonBody(map[string]any{"version": draft.Version, "text": "Intrusion"}), false}, {"DELETE", editPath, jsonBody(map[string]any{"version": draft.Version}), false},
		{"GET", base + "/site", nil, false}, {"PATCH", "/api/sites/" + site.ID, []byte(`{"version":1,"address":"Intrusion"}`), false},
		{"GET", base + "/profile", nil, false}, {"GET", base + "/profile/sources", nil, false},
		{"PUT", base + "/profile/hdr.subclass", []byte(`{"value":"house"}`), false}, {"PUT", base + "/profile/scope", []byte(`{"systems":{"mechanical.air-conditioning":"in"}}`), false}, {"POST", base + "/profile/read", nil, false},
		{"POST", base + "/parts", []byte(`{"label":"Intrusion","kind":"roof"}`), false}, {"PATCH", base + "/parts/" + part, []byte(`{"label":"Intrusion"}`), true},
		{"GET", base + "/planning", nil, false}, {"PUT", base + "/planning/gross_floor_area", jsonBody(map[string]any{"part_id": part, "value": 1300, "unit": "m2", "version": 1}), true}, {"DELETE", base + "/planning/gross_floor_area?version=1&part_id=" + part, nil, true},
	}
	denied := func(t *testing.T, actor *member, r route, path string, want int) {
		t.Helper()
		response := actor.do(t, r.method, path, r.body)
		defer response.Body.Close()
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != want {
			t.Fatalf("%s %s: %d: %s", r.method, path, response.StatusCode, raw)
		}
		for _, secret := range []string{project, work.ID, pkg.ID, report.ID, "Private", proposal.Label} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("response disclosed %q: %s", secret, raw)
			}
		}
	}
	for i, r := range routes {
		t.Run(fmt.Sprintf("%02d_%s", i, r.method), func(t *testing.T) {
			t.Run("signed_out", func(t *testing.T) { denied(t, &member{app: a, client: &http.Client{}}, r, r.path, 401) })
			if r.method != "GET" {
				t.Run("wrong_origin", func(t *testing.T) {
					req, err := http.NewRequest(r.method, a.url+r.path, bytes.NewReader(r.body))
					if err != nil {
						t.Fatal(err)
					}
					req.Header.Set("Origin", "https://untrusted.example")
					req.Header.Set("Content-Type", "application/json")
					response, err := m.client.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					defer response.Body.Close()
					if response.StatusCode != 403 {
						t.Fatalf("origin accepted: %d", response.StatusCode)
					}
				})
			}
			t.Run("foreign_org", func(t *testing.T) { denied(t, foreign, r, r.path, 404) })
			if r.foreignTarget {
				t.Run("other_project_and_site", func(t *testing.T) { denied(t, m, r, strings.Replace(r.path, base, "/api/projects/"+sibling, 1), 404) })
			}
		})
	}
	// Both orders matter: a permitted first member must not be changed before
	// the foreign second member is rejected, nor expose its state on a retry.
	for _, wrongDoc := range []string{otherDoc, foreignDoc, newUUID(t)} {
		for _, ids := range [][]string{{doc, wrongDoc}, {wrongDoc, doc}} {
			for _, op := range []struct{ method, suffix string }{{"PUT", "profile-read"}, {"POST", "delete"}} {
				body := map[string]any{"document_ids": ids}
				if op.method == "PUT" {
					body["setting"] = "skip"
				}
				t.Run("mixed_document_batch/"+op.suffix+"/"+ids[0], func(t *testing.T) {
					denied(t, m, route{method: op.method, body: jsonBody(body)}, base+"/documents/"+op.suffix, 404)
				})
			}
		}
	}
	otherPkgPath := base + "/packages/" + otherPkg.ID
	nested := []route{
		{"PATCH", otherPkgPath + "/stages/" + pkg.Stages[0].ID, []byte(`{"version":1,"label":"Intrusion"}`), false},
		{"PATCH", otherPkgPath + "/scope/" + scope.ID, []byte(`{"version":1,"user_text":"Intrusion"}`), false},
		{"POST", otherPkgPath + "/scope", jsonBody(map[string]any{"item_kind": "obligation", "user_text": "Intrusion", "inclusion": "included", "stage_id": pkg.Stages[0].ID}), false},
		{"POST", pkgPath + "/scope", jsonBody(map[string]any{"item_kind": "responsibility", "user_text": "Intrusion", "inclusion": "included", "work_item_id": siblingWork.ID, "role": "design"}), false},
		{"PATCH", pkgPath + "/scope/" + scope.ID, jsonBody(map[string]any{"version": scope.Version, "stage_id": otherPkg.Stages[0].ID}), false},
		{"PATCH", base + "/delivery/" + item.ID, jsonBody(map[string]any{"version": item.Version, "stage_id": otherPkg.Stages[0].ID}), false},
		{"POST", base + "/delivery", jsonBody(map[string]any{"kind": "approval", "title": "Intrusion", "package_id": pkg.ID, "work_item_id": siblingWork.ID}), false},
		{"POST", base + "/delivery", jsonBody(map[string]any{"kind": "approval", "title": "Intrusion", "package_id": pkg.ID, "stage_id": otherPkg.Stages[0].ID}), false},
		{"PUT", reportPath + "/edits/" + url.PathEscape("package:"+otherPkg.ID), jsonBody(map[string]any{"version": draft.Version, "text": "Intrusion"}), false},
		{"DELETE", reportPath + "/edits/" + url.PathEscape("package:"+otherPkg.ID), jsonBody(map[string]any{"version": draft.Version}), false},
	}
	for i, r := range nested {
		t.Run(fmt.Sprintf("nested_parent_%02d", i), func(t *testing.T) { denied(t, m, r, r.path, 404) })
	}
	var obligation *store.ProposalView
	for i := range proposals {
		if proposals[i].Kind == "obligation" {
			obligation = &proposals[i]
			break
		}
	}
	if obligation == nil {
		t.Fatal("fixture has no obligation proposal")
	}
	for name, choice := range map[string]map[string]any{
		"stage_from_other_package": {"package_id": pkg.ID, "stage_id": otherPkg.Stages[0].ID},
		"work_from_other_project":  {"package_id": pkg.ID, "work_item_id": siblingWork.ID, "role": "design"},
	} {
		choice["inputs_fingerprint"] = obligation.InputsFingerprint
		path := base + "/proposals/" + url.PathEscape(obligation.Key) + "/accept"
		t.Run("proposal_assignment/"+name, func(t *testing.T) { denied(t, m, route{method: "POST", body: jsonBody(choice)}, path, 404) })
	}
	if after := snapshot(); !reflect.DeepEqual(before, after) {
		for table, raw := range before {
			if raw != after[table] {
				t.Errorf("rejected requests changed %s", table)
			}
		}
	}
	if a.jevHits.Load() != 0 {
		t.Fatal("access sweep called Jev")
	}
}
