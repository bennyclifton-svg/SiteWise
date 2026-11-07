package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sitewise/internal/auth"
	"sitewise/internal/httpapi"
	"sitewise/internal/store"
)

const origin = "http://127.0.0.1:8080"

type resetBox chan [2]string

func (b resetBox) SendReset(_ context.Context, to, link string) error {
	b <- [2]string{to, link}
	return nil
}

func accountHandler(st *store.Store, logs *bytes.Buffer, mail auth.ResetMailer) http.Handler {
	return httpapi.Handler(httpapi.Deps{
		Store:        st,
		PublicOrigin: origin,
		Log:          log.New(logs, "", 0),
		MaxBodyBytes: 1024,
		ResetMail:    mail,
	})
}

// uniqueEmail keeps runs apart in the shared test database and removes the
// account's org afterwards.
func uniqueEmail(t *testing.T, st *store.Store) string {
	t.Helper()
	email := "acct-" + newID() + "@example.com"
	t.Cleanup(func() {
		if a, err := st.AccountByEmail(context.Background(), email); err == nil {
			_ = st.DeleteOrg(context.Background(), a.OrgID)
		}
	})
	return email
}

func creds(email, password string) map[string]any {
	return map[string]any{"email": email, "password": password, "accept_terms": true}
}

func getWith(t *testing.T, h http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, request(t, http.MethodGet, path, cookie, nil))
	return rec
}

func TestSignUpOpensSessionAndOwnOrg(t *testing.T) {
	st, logs := openStore(t)
	h := accountHandler(st, logs, nil)
	email := uniqueEmail(t, st)

	rec := postJSON(t, h, "/signup", nil, creds(email, "correct horse"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("signup %d %s", rec.Code, rec.Body)
	}
	cookie := sessionCookie(t, rec)
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("session cookie must be HttpOnly and SameSite=Lax")
	}
	if got := getWith(t, h, "/session", cookie); got.Code != http.StatusNoContent {
		t.Fatalf("session check %d", got.Code)
	}
	if got := postJSON(t, h, "/projects", cookie, map[string]string{"name": "Mine"}); got.Code != http.StatusCreated {
		t.Fatalf("create project %d %s", got.Code, got.Body)
	}
	if strings.Contains(logs.String(), "correct horse") {
		t.Fatal("log contains the password")
	}

	// The same email in another case is the same account.
	dup := postJSON(t, h, "/signup", nil, creds(strings.ToUpper(email), "another pass"))
	if dup.Code != http.StatusConflict {
		t.Fatalf("duplicate signup %d", dup.Code)
	}
}

func TestAccountsSeeOnlyTheirOwnProjects(t *testing.T) {
	st, logs := openStore(t)
	h := accountHandler(st, logs, nil)
	a := sessionCookie(t, postJSON(t, h, "/signup", nil, creds(uniqueEmail(t, st), "password-a")))
	b := sessionCookie(t, postJSON(t, h, "/signup", nil, creds(uniqueEmail(t, st), "password-b")))
	created := postJSON(t, h, "/projects", a, map[string]string{"name": "A only"})
	var p struct{ ID string }
	_ = json.NewDecoder(created.Body).Decode(&p)

	if got := getWith(t, h, "/projects/"+p.ID, b); got.Code != http.StatusNotFound {
		t.Fatalf("other account read project: %d", got.Code)
	}
	list := getWith(t, h, "/projects", b)
	if strings.Contains(list.Body.String(), p.ID) {
		t.Fatal("other account lists the project")
	}
}

func TestSignInChecksPassword(t *testing.T) {
	st, logs := openStore(t)
	h := accountHandler(st, logs, nil)
	email := uniqueEmail(t, st)
	first := sessionCookie(t, postJSON(t, h, "/signup", nil, creds(email, "right password")))

	wrong := postJSON(t, h, "/signin", nil, creds(email, "wrong password"))
	unknown := postJSON(t, h, "/signin", nil, creds("nobody-"+newID()+"@example.com", "wrong password"))
	if wrong.Code != http.StatusUnauthorized || unknown.Code != http.StatusUnauthorized {
		t.Fatalf("wrong %d unknown %d", wrong.Code, unknown.Code)
	}
	if wrong.Body.String() != unknown.Body.String() {
		t.Fatal("wrong password and unknown email must read the same")
	}
	if len(wrong.Result().Cookies()) > 0 {
		t.Fatal("failed sign-in set a cookie")
	}

	ok := postJSON(t, h, "/signin", nil, creds(" "+strings.ToUpper(email)+" ", "right password"))
	if ok.Code != http.StatusNoContent {
		t.Fatalf("signin %d %s", ok.Code, ok.Body)
	}
	second := sessionCookie(t, ok)
	if second.Value == first.Value {
		t.Fatal("sign-in must open a new session")
	}
	if got := getWith(t, h, "/session", second); got.Code != http.StatusNoContent {
		t.Fatalf("session check %d", got.Code)
	}
}

func TestAccountInputIsChecked(t *testing.T) {
	st, logs := openStore(t)
	h := accountHandler(st, logs, nil)
	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"no terms", map[string]any{"email": uniqueEmail(t, st), "password": "long enough"}, http.StatusBadRequest},
		{"no at", creds("not-an-email", "long enough"), http.StatusBadRequest},
		{"header injection", creds("a@example.com\r\nBcc: x@example.com", "long enough"), http.StatusBadRequest},
		{"short password", creds(uniqueEmail(t, st), "short"), http.StatusBadRequest},
		{"long password", creds(uniqueEmail(t, st), strings.Repeat("x", 73)), http.StatusBadRequest},
	}
	for _, c := range cases {
		if got := postJSON(t, h, "/signup", nil, c.body); got.Code != c.want {
			t.Errorf("%s: %d", c.name, got.Code)
		}
	}
	// No Origin header: refused before anything is read.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(`{"email":"x@example.com","password":"long enough"}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing origin %d", rec.Code)
	}
}

func TestPasswordReset(t *testing.T) {
	st, logs := openStore(t)
	box := make(resetBox, 4)
	h := accountHandler(st, logs, box)
	email := uniqueEmail(t, st)
	old := sessionCookie(t, postJSON(t, h, "/signup", nil, creds(email, "old password")))

	if got := postJSON(t, h, "/password/forgot", nil, map[string]string{"email": "nobody-" + newID() + "@example.com"}); got.Code != http.StatusNoContent {
		t.Fatalf("forgot unknown %d", got.Code)
	}
	if got := postJSON(t, h, "/password/forgot", nil, map[string]string{"email": email}); got.Code != http.StatusNoContent {
		t.Fatalf("forgot %d", got.Code)
	}
	var link string
	select {
	case m := <-box:
		if m[0] != email {
			t.Fatalf("mail went to %q", m[0])
		}
		link = m[1]
	case <-time.After(5 * time.Second):
		t.Fatal("no reset mail")
	}
	select {
	case m := <-box:
		t.Fatalf("unknown email was mailed: %v", m)
	case <-time.After(200 * time.Millisecond):
	}
	if !strings.HasPrefix(link, origin+"/#reset=") {
		t.Fatalf("link %q", link)
	}
	token := strings.TrimPrefix(link, origin+"/#reset=")
	if strings.Contains(logs.String(), token) {
		t.Fatal("log contains reset token")
	}

	rec := postJSON(t, h, "/password/reset", nil, map[string]string{"token": token, "password": "new password"})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("reset %d %s", rec.Code, rec.Body)
	}
	fresh := sessionCookie(t, rec)
	if got := getWith(t, h, "/session", fresh); got.Code != http.StatusNoContent {
		t.Fatalf("new session %d", got.Code)
	}
	if got := getWith(t, h, "/session", old); got.Code != http.StatusUnauthorized {
		t.Fatalf("old session survived reset: %d", got.Code)
	}
	if got := postJSON(t, h, "/signin", nil, creds(email, "old password")); got.Code != http.StatusUnauthorized {
		t.Fatalf("old password %d", got.Code)
	}
	if got := postJSON(t, h, "/signin", nil, creds(email, "new password")); got.Code != http.StatusNoContent {
		t.Fatalf("new password %d", got.Code)
	}
	if got := postJSON(t, h, "/password/reset", nil, map[string]string{"token": token, "password": "third password"}); got.Code != http.StatusUnauthorized {
		t.Fatalf("reused reset %d", got.Code)
	}
}

func TestSignInGuessesAreLimited(t *testing.T) {
	st, logs := openStore(t)
	h := accountHandler(st, logs, nil)
	email := uniqueEmail(t, st)
	postJSON(t, h, "/signup", nil, creds(email, "right password"))
	var last int
	for i := 0; i < 11; i++ {
		last = postJSON(t, h, "/signin", nil, creds(email, "guess "+newID())).Code
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("11th guess %d", last)
	}
	// The limit holds even for the right password until the window passes.
	if got := postJSON(t, h, "/signin", nil, creds(email, "right password")); got.Code != http.StatusTooManyRequests {
		t.Fatalf("after limit %d", got.Code)
	}
}
