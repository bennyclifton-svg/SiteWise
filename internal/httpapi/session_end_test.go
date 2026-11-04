package httpapi_test

import (
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"sitewise/bench"
	"sitewise/internal/auth"
	"sitewise/internal/latency"
)

func TestEndSessionRevokesOnlyCurrentSession(t *testing.T) {
	a := newApp(t)
	m := a.member(t, "Sign out")
	other := a.member(t, "Other organisation")
	u, _ := url.Parse(a.url)
	var stolen *http.Cookie
	for _, cookie := range m.client.Jar.Cookies(u) {
		if cookie.Name == auth.SessionCookie {
			stolen = cookie
		}
	}
	if stolen == nil {
		t.Fatal("missing authenticated cookie")
	}
	stream := m.do(t, http.MethodGet, "/api/events", nil)
	defer stream.Body.Close()
	streamEnded := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, stream.Body); close(streamEnded) }()
	for _, origin := range []string{"", "https://wrong.example"} {
		req, _ := http.NewRequest(http.MethodDelete, a.url+"/api/session", nil)
		req.Header.Set("Origin", origin)
		res, err := m.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Fatalf("cross-origin logout: %d", res.StatusCode)
		}
	}
	if got := m.status(t, http.MethodGet, "/api/session", nil); got != http.StatusNoContent {
		t.Fatalf("rejected logout changed session: %d", got)
	}
	res := m.do(t, http.MethodDelete, "/api/session", nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %d", res.StatusCode)
	}
	select {
	case <-streamEnded:
	case <-time.After(2 * time.Second):
		t.Fatal("sign-out left its event stream open")
	}
	if len(res.Cookies()) != 1 || res.Cookies()[0].MaxAge != -1 || !res.Cookies()[0].HttpOnly {
		t.Fatal("logout did not expire HttpOnly cookie")
	}
	// Clearing a cookie alone is insufficient: a captured cookie must fail too.
	req, _ := http.NewRequest(http.MethodGet, a.url+"/api/session", nil)
	req.AddCookie(stolen)
	replay, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	replay.Body.Close()
	if replay.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked cookie accepted: %d", replay.StatusCode)
	}
	if got := other.status(t, http.MethodGet, "/api/session", nil); got != http.StatusNoContent {
		t.Fatalf("other session revoked: %d", got)
	}
	if got := m.status(t, http.MethodDelete, "/api/session", nil); got != http.StatusUnauthorized {
		t.Fatalf("anonymous logout: %d", got)
	}
}

func TestEndSessionLatencyBudget(t *testing.T) {
	a := newApp(t)
	budgets, err := bench.Budgets()
	if err != nil {
		t.Fatal(err)
	}
	var selected []latency.PathBudget
	for _, path := range budgets.Paths {
		if path.Name == "project_invite_auth" {
			selected = append(selected, path)
		}
	}
	if len(selected) != 1 {
		t.Fatal("missing authentication budget")
	}
	var samples []int64
	for i := 0; i < budgets.MinSamples; i++ {
		m := a.member(t, "Sign-out timing")
		start := time.Now()
		status := m.status(t, http.MethodDelete, "/api/session", nil)
		samples = append(samples, time.Since(start).Microseconds())
		if status != http.StatusNoContent {
			t.Fatalf("logout: %d", status)
		}
	}
	code, report := latency.Gate(latency.Budgets{MinSamples: budgets.MinSamples, Paths: selected}, map[string][]int64{"project_invite_auth": samples}, false)
	t.Log(report)
	p50, _ := latency.Percentile(samples, 0.5)
	p90, _ := latency.Percentile(samples, 0.9)
	t.Logf("sign-out p50=%.3fms p90=%.3fms; budget %.0f/%.0fms (%d samples)", float64(p50)/1000, float64(p90)/1000, float64(selected[0].P50US)/1000, float64(selected[0].P90US)/1000, len(samples))
	if code != 0 {
		t.Fatal("sign-out exceeded authentication budget")
	}
}
