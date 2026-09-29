package auth_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/httpapi"
	"sitewise/internal/store"
)

func TestExpiredInvite(t *testing.T) {
	st, logs := openStore(t)
	ctx := context.Background()
	orgID := newID()
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), orgID) })
	if err := st.CreateOrg(ctx, orgID, "A"); err != nil {
		t.Fatal(err)
	}
	raw, err := auth.CreateInvite(ctx, st, discardMail{}, orgID, "a@example.com", "member", time.Now().Add(-time.Hour), logs)
	if err != nil {
		t.Fatal(err)
	}
	rec := postSession(t, handler(st, logs), raw, orgID)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Result().Cookies() != nil && len(rec.Result().Cookies()) > 0 {
		t.Fatal("expired invite set a cookie")
	}
	if strings.Contains(logs.String(), raw) {
		t.Fatal("log contains invite token")
	}
}

func TestReplayedInvite(t *testing.T) {
	st, logs := openStore(t)
	ctx := context.Background()
	orgID := newID()
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), orgID) })
	if err := st.CreateOrg(ctx, orgID, "A"); err != nil {
		t.Fatal(err)
	}
	raw, err := auth.CreateInvite(ctx, st, discardMail{}, orgID, "a@example.com", "member", time.Now().Add(time.Hour), logs)
	if err != nil {
		t.Fatal(err)
	}
	h := handler(st, logs)
	first := postSession(t, h, raw, "")
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status %d body %s", first.Code, first.Body.String())
	}
	second := postSession(t, h, raw, "")
	if second.Code != http.StatusUnauthorized {
		t.Fatalf("replay status %d", second.Code)
	}
	if strings.Contains(logs.String(), raw) {
		t.Fatal("log contains invite token")
	}
}

func TestWrongOrgInvite(t *testing.T) {
	st, logs := openStore(t)
	ctx := context.Background()
	orgA, orgB := newID(), newID()
	projectB := newID()
	t.Cleanup(func() {
		_ = st.DeleteOrg(context.Background(), orgA)
		_ = st.DeleteOrg(context.Background(), orgB)
	})
	if err := st.CreateOrg(ctx, orgA, "A"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateOrg(ctx, orgB, "B"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, orgB, projectB, "Theirs"); err != nil {
		t.Fatal(err)
	}
	raw, err := auth.CreateInvite(ctx, st, discardMail{}, orgA, "a@example.com", "owner", time.Now().Add(time.Hour), logs)
	if err != nil {
		t.Fatal(err)
	}
	h := handler(st, logs)
	rec := postSession(t, h, raw, orgB)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("consume %d %s", rec.Code, rec.Body.String())
	}
	cookie := sessionCookie(t, rec)
	if cookie.Value == raw || strings.Contains(cookie.Value, orgB) {
		t.Fatalf("cookie value leaks token or org: %s", cookie.Value)
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie httpOnly=%v sameSite=%v", cookie.HttpOnly, cookie.SameSite)
	}
	get := request(t, http.MethodGet, "/projects/"+projectB, cookie, nil)
	got := httptest.NewRecorder()
	h.ServeHTTP(got, get)
	if got.Code != http.StatusNotFound {
		t.Fatalf("cross-org read %d", got.Code)
	}
	body := map[string]string{"name": "Ours", "org_id": orgB}
	created := postJSON(t, h, "/projects", cookie, body)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	var resp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetProject(ctx, orgA, resp.ID); err != nil {
		t.Fatalf("project not in invite org: %v", err)
	}
	if _, err := st.GetProject(ctx, orgB, resp.ID); err == nil {
		t.Fatal("project was created in the forged org")
	}
}

func TestProjectAccessBeforeMembership(t *testing.T) {
	st, logs := openStore(t)
	ctx := context.Background()
	orgID, userID, sessionID, projectID := newID(), newID(), newID(), newID()
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), orgID) })
	if err := st.CreateOrg(ctx, orgID, "A"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(ctx, orgID, userID, "a@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateSession(ctx, orgID, sessionID, userID); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, orgID, projectID, "P"); err != nil {
		t.Fatal(err)
	}
	h := handler(st, logs)
	cookie := &http.Cookie{Name: auth.SessionCookie, Value: sessionID}
	got := httptest.NewRecorder()
	h.ServeHTTP(got, request(t, http.MethodGet, "/projects/"+projectID, cookie, nil))
	if got.Code != http.StatusForbidden {
		t.Fatalf("status %d", got.Code)
	}
}

func TestMutationRequiresOrigin(t *testing.T) {
	st, logs := openStore(t)
	h := handler(st, logs)
	req := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"X"}`))
	req.Header.Set("Content-Type", "application/json")
	got := httptest.NewRecorder()
	h.ServeHTTP(got, req)
	if got.Code != http.StatusForbidden {
		t.Fatalf("status %d", got.Code)
	}
}

func TestMailSinkReceivesInvite(t *testing.T) {
	st, logs := openStore(t)
	ctx := context.Background()
	orgID := newID()
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), orgID) })
	if err := st.CreateOrg(ctx, orgID, "A"); err != nil {
		t.Fatal(err)
	}
	sink := &auth.Sink{}
	raw, err := auth.CreateInvite(ctx, st, sink, orgID, "a@example.com", "member", time.Now().Add(time.Hour), logs)
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.Messages) != 1 || sink.Messages[0].To != "a@example.com" || !strings.Contains(sink.Messages[0].Body, raw) {
		t.Fatalf("%+v", sink.Messages)
	}
	if strings.Contains(logs.String(), raw) {
		t.Fatal("log contains invite token")
	}
}

func TestBootstrapAdmin(t *testing.T) {
	st, logs := openStore(t)
	ctx := context.Background()
	raw, orgID, err := auth.Bootstrap(ctx, st, &auth.Sink{}, "Pilot", "owner@example.com", logs)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), orgID) })
	if strings.Contains(logs.String(), raw) {
		t.Fatal("log contains invite token")
	}
	h := handler(st, logs)
	rec := postSession(t, h, raw, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("consume %d %s", rec.Code, rec.Body.String())
	}
	created := postJSON(t, h, "/projects", sessionCookie(t, rec), map[string]string{"name": "First"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
}

func handler(st *store.Store, logs *bytes.Buffer) http.Handler {
	return httpapi.Handler(httpapi.Deps{
		Store:        st,
		PublicOrigin: "http://127.0.0.1:8080",
		Log:          log.New(logs, "", 0),
		MaxBodyBytes: 1024,
	})
}

func postSession(t *testing.T, h http.Handler, token, orgID string) *httptest.ResponseRecorder {
	t.Helper()
	body := map[string]string{"token": token}
	if orgID != "" {
		body["org_id"] = orgID
	}
	return postJSON(t, h, "/session", nil, body)
}

func postJSON(t *testing.T, h http.Handler, path string, cookie *http.Cookie, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, request(t, http.MethodPost, path, cookie, &buf))
	return rec
}

func request(t *testing.T, method, path string, cookie *http.Cookie, body io.Reader) *http.Request {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:8080")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == auth.SessionCookie {
			return cookie
		}
	}
	t.Fatal("missing session cookie")
	return nil
}

func openStore(t *testing.T) (*store.Store, *bytes.Buffer) {
	t.Helper()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimPrefix(u.Path, "/") != "sitewise_test" {
		t.Fatalf("refusing database %q", u.Path)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return st, &bytes.Buffer{}
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type discardMail struct{}

func (discardMail) SendInvite(context.Context, string, string) error { return nil }
