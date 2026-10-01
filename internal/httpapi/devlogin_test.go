package httpapi_test

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"sitewise/internal/auth"
	"sitewise/internal/httpapi"
)

func noFollow(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func TestDevLoginSignsInEveryVisit(t *testing.T) {
	a := newApp(t, func(o *httpapi.Options) { o.DevLogin = true })
	t.Cleanup(func() { _ = a.store.DeleteOrg(t.Context(), httpapi.DevOrgID) })
	for visit := 1; visit <= 2; visit++ {
		c := noFollow(t)
		resp, err := c.Get(a.url + "/dev/login")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
			t.Fatalf("visit %d: status %d location %q", visit, resp.StatusCode, resp.Header.Get("Location"))
		}
		var session *http.Cookie
		for _, ck := range resp.Cookies() {
			if ck.Name == auth.SessionCookie {
				session = ck
			}
		}
		if session == nil || !session.HttpOnly {
			t.Fatalf("visit %d: no HttpOnly session cookie", visit)
		}
		check, err := c.Get(a.url + "/api/session")
		if err != nil {
			t.Fatal(err)
		}
		check.Body.Close()
		if check.StatusCode != http.StatusNoContent && check.StatusCode != http.StatusOK {
			t.Fatalf("visit %d: session check %d", visit, check.StatusCode)
		}
	}
}

func TestDevLoginIsOffByDefault(t *testing.T) {
	a := newApp(t)
	resp, err := noFollow(t).Get(a.url + "/dev/login")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	for _, ck := range resp.Cookies() {
		if ck.Name == auth.SessionCookie {
			t.Fatal("dev login must not issue a session unless enabled")
		}
	}
	if !strings.Contains(string(body), "sitewise-test") {
		t.Fatalf("an unknown path serves the app shell: %d %s", resp.StatusCode, body)
	}
}
