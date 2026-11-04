package httpapi_test

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestProfileReadSurvivesSlowReader is F1 (testing/synthetic/FINDINGS.md):
// the Update button sends a small JSON body the handler never needed, with
// "Connection: close" (Python's urllib, as the harness does). The server
// closed the socket with that body unread in the kernel, which sends a TCP
// reset; Windows then drops the reply the client had not read yet, so a
// client reading slower than the server writes saw the ~200 KB profile cut
// off. Headers and body go in separate writes, as urllib sends them.
func TestProfileReadSurvivesSlowReader(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Slow reader")
	base, err := url.Parse(m.app.url)
	if err != nil {
		t.Fatal(err)
	}
	var cookies []string
	for _, c := range m.client.Jar.Cookies(base) {
		cookies = append(cookies, c.Name+"="+c.Value)
	}

	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", base.Host)
		if err != nil {
			t.Fatal(err)
		}
		head := "POST /api/projects/" + project + "/profile/read HTTP/1.1\r\n" +
			"Host: " + base.Host + "\r\n" +
			"Origin: " + m.app.url + "\r\n" +
			"Cookie: " + strings.Join(cookies, "; ") + "\r\n" +
			"Content-Type: application/json\r\n" +
			"Content-Length: 2\r\n" +
			"Connection: close\r\n\r\n"
		if _, err := io.WriteString(conn, head); err != nil {
			t.Fatal(err)
		}
		time.Sleep(50 * time.Millisecond)
		if _, err := io.WriteString(conn, "{}"); err != nil {
			t.Fatal(err)
		}
		// A busy client reads late; the server has finished by then.
		time.Sleep(1500 * time.Millisecond)
		resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			conn.Close()
			t.Fatalf("attempt %d: read response: %v", i, err)
		}
		raw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		conn.Close()
		if err != nil {
			t.Fatalf("attempt %d: status %d, body cut off after %d bytes: %v", i, resp.StatusCode, len(raw), err)
		}
		var got struct {
			ProjectID string `json:"project_id"`
			Queued    *int   `json:"queued"`
		}
		if resp.StatusCode != http.StatusOK || json.Unmarshal(raw, &got) != nil || got.ProjectID != project || got.Queued == nil {
			t.Fatalf("attempt %d: status %d, %d bytes", i, resp.StatusCode, len(raw))
		}
		if len(raw) < 64<<10 {
			t.Logf("profile body is only %d bytes; a small body may not show the reset", len(raw))
		}
	}
}
