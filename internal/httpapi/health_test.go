package httpapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"path/filepath"
	"testing"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/latency"
)

type healthBody struct {
	Status   string   `json:"status"`
	Reasons  []string `json:"reasons"`
	Database struct {
		OK bool `json:"ok"`
	} `json:"database"`
	Jev struct {
		Circuit string `json:"circuit"`
		Stale   bool   `json:"stale"`
	} `json:"jev"`
	Backlog []struct {
		Kind    string  `json:"kind"`
		Queued  int64   `json:"queued"`
		OldestS float64 `json:"oldest_s"`
	} `json:"backlog"`
}

type speedBody struct {
	Window int `json:"window"`
	Paths  []struct {
		Name         string  `json:"name"`
		N            int     `json:"n"`
		P50MS        float64 `json:"p50_ms"`
		P90MS        float64 `json:"p90_ms"`
		WithinBudget *bool   `json:"within_budget"`
	} `json:"paths"`
}

func TestPublicHealthIsMinimal(t *testing.T) {
	app := newApp(t)
	resp, err := http.Get(app.url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz = %d: %s", resp.StatusCode, raw)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	// Status only: no reasons, counts, ages, versions or provider detail.
	if len(body) != 1 {
		t.Fatalf("public health exposes %v", body)
	}
	if s := body["status"]; s != "ok" && s != "degraded" {
		t.Fatalf("status %v", s)
	}
}

func TestDetailedHealthAndSpeedNeedAnOwner(t *testing.T) {
	app := newApp(t)
	owner := app.member(t, "Org A")
	for _, path := range []string{"/api/health", "/api/speed"} {
		resp, err := http.Get(app.url + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("anonymous %s = %d", path, resp.StatusCode)
		}
	}
	plain := app.invited(t, owner.orgID, "member")
	for _, path := range []string{"/api/health", "/api/speed"} {
		if got := plain.status(t, http.MethodGet, path, nil); got != http.StatusForbidden {
			t.Fatalf("member %s = %d", path, got)
		}
	}
	var h healthBody
	owner.getJSON(t, "/api/health", &h)
	if !h.Database.OK || h.Jev.Circuit != "closed" || h.Status == "" {
		t.Fatalf("health %+v", h)
	}
	var s speedBody
	owner.getJSON(t, "/api/speed", &s)
	if s.Window == 0 || len(s.Paths) == 0 {
		t.Fatalf("speed %+v", s)
	}
}

// Detailed health reports only the caller's org's queue.
func TestHealthBacklogIsTheCallersOrg(t *testing.T) {
	app := newApp(t)
	a := app.member(t, "Org A")
	b := app.member(t, "Org B")
	project := b.createProject(t, "B work")
	stream := b.events(t, 0)
	doc := b.upload(t, project, "identity-page.pdf", http.StatusCreated)
	stream.next(t, "filing", doc.ID)

	var hb healthBody
	b.getJSON(t, "/api/health", &hb)
	if queued(hb, "full_text") == 0 {
		t.Fatalf("org B backlog %+v", hb.Backlog)
	}
	var ha healthBody
	a.getJSON(t, "/api/health", &ha)
	if len(ha.Backlog) != 0 {
		t.Fatalf("org A sees %+v", ha.Backlog)
	}
}

func TestSpeedIsRecordedNotAsked(t *testing.T) {
	app := newApp(t)
	owner := app.member(t, "Org A")
	project := owner.createProject(t, "Speed")
	stream := owner.events(t, 0)
	doc := owner.upload(t, project, "identity-page.pdf", http.StatusCreated)
	stream.next(t, "filing", doc.ID)
	owner.list(t, project)

	// whole_intake is recorded just after the filing event is written, so
	// the client can see the event first.
	var s speedBody
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		owner.getJSON(t, "/api/speed", &s)
		if recorded(s, "whole_intake") {
			break
		}
	}
	before := app.jevHits.Load()
	owner.getJSON(t, "/api/speed", &s)
	var h healthBody
	owner.getJSON(t, "/api/health", &h)
	if resp, err := http.Get(app.url + "/healthz"); err == nil {
		resp.Body.Close()
	}
	if after := app.jevHits.Load(); after != before {
		t.Fatalf("health and speed called Jev %d times", after-before)
	}
	if before == 0 {
		t.Fatal("the filing did not reach the fake provider")
	}
	for _, name := range []string{"whole_intake", "identity_text_extraction", "jev_admission_request", "project_document_list"} {
		if !recorded(s, name) {
			t.Fatalf("%s not recorded: %+v", name, s.Paths)
		}
	}
}

func recorded(s speedBody, name string) bool {
	for _, p := range s.Paths {
		if p.Name == name && p.N >= 1 {
			return true
		}
	}
	return false
}

func TestHealthSpeedBudget(t *testing.T) {
	app := newApp(t)
	owner := app.member(t, "Org A")
	budgets, err := latency.LoadBudgets(filepath.Join("..", "..", "bench", "budgets.json"))
	if err != nil {
		t.Fatal(err)
	}
	samples := map[string][]int64{}
	for i := 0; i < budgets.MinSamples; i++ {
		path := "/api/health"
		if i%2 == 1 {
			path = "/api/speed"
		}
		start := time.Now()
		owner.getJSON(t, path, &map[string]any{})
		samples["health_speed"] = append(samples["health_speed"], time.Since(start).Microseconds())
	}
	scoped := latency.Budgets{MinSamples: budgets.MinSamples}
	for _, p := range budgets.Paths {
		if p.Name == "health_speed" {
			scoped.Paths = append(scoped.Paths, p)
		}
	}
	if code, report := latency.Gate(scoped, samples); code != 0 {
		t.Fatal(report)
	}
}

func queued(h healthBody, kind string) int64 {
	for _, b := range h.Backlog {
		if b.Kind == kind {
			return b.Queued
		}
	}
	return 0
}

// invited signs in a new user of orgID with role.
func (a *app) invited(t *testing.T, orgID, role string) *member {
	t.Helper()
	raw, err := auth.CreateInvite(context.Background(), a.store, &auth.Sink{}, orgID, newUUID(t)+"@example.test", role, time.Now().Add(time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := &member{app: a, client: &http.Client{Jar: jar}, orgID: orgID}
	body, _ := json.Marshal(map[string]string{"token": raw})
	if got := m.status(t, http.MethodPost, "/api/session", body); got != http.StatusNoContent {
		t.Fatalf("sign in = %d", got)
	}
	return m
}
