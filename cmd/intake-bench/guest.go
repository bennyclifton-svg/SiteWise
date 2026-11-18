package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/store"
)

// guestLinks samples guest_open (the link exchange and the guest page read
// on a package link whose RFT transmittal holds 50 documents), team (share
// create, list and revoke) and procurement_board (the board with link
// state), as award.go samples procurement_invite
// (docs/plans/2026-11-17-guest-access.md, Task 10). Nothing here calls Jev.
func (a *apiClient) guestLinks(ctx context.Context, st *store.Store, dsn string, n int) error {
	project, err := a.createProject(ctx, "Guest link benchmark")
	if err != nil {
		return err
	}
	var row struct {
		ID string `json:"id"`
	}
	out, err := a.call(ctx, http.MethodPost, "/projects/"+project+"/procurement/rows", []byte(`{"discipline_id":"consultant.town_planner"}`), http.StatusCreated)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, &row); err != nil {
		return err
	}
	base := "/projects/" + project + "/procurement/packages/" + row.ID
	var inv struct {
		FirmID string `json:"firm_id"`
	}
	if out, err = a.call(ctx, http.MethodPost, base+"/invitations", []byte(`{"name":"Bench Guest Planning"}`), http.StatusCreated); err != nil {
		return err
	}
	if err := json.Unmarshal(out, &inv); err != nil {
		return err
	}
	var report struct {
		ID string `json:"id"`
	}
	if out, err = a.call(ctx, http.MethodPost, "/projects/"+project+"/reports", []byte(`{"kind":"rfp","package_id":"`+row.ID+`"}`), http.StatusCreated); err != nil {
		return err
	}
	if err := json.Unmarshal(out, &report); err != nil {
		return err
	}
	if _, err := a.call(ctx, http.MethodPost, "/reports/"+report.ID+"/draft", []byte(`{"use_last_completed":true}`), http.StatusOK); err != nil {
		return err
	}
	// Fifty filed documents and an issue whose transmittal holds them. The
	// RFP issue route's clause gates are not what this samples.
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var org string
	if err := conn.QueryRow(ctx, `SELECT org_id::text FROM projects WHERE id::text=$1`, project).Scan(&org); err != nil {
		return err
	}
	var ids []string
	for i := 0; i < 50; i++ {
		id, file := guestUUID(), guestUUID()
		sum := sha256.Sum256([]byte("guest bench " + id))
		if err := st.CreateFile(ctx, org, store.File{ID: file, ProjectID: project, SHA256: sum[:], ByteSize: 1, MediaType: "application/pdf"}); err != nil {
			return err
		}
		if err := st.CreateDocument(ctx, org, store.Document{ID: id, ProjectID: project, FileID: file, Filename: fmt.Sprintf("A-%03d Bench sheet.pdf", i), Status: "filed"}); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	sections, _ := json.Marshal([]map[string]any{{"id": "project", "title": "Project", "blocks": []map[string]any{{"id": "rfp:project:information", "label": "Transmittal", "text": "", "basis": map[string]any{"document_ids": ids}}}}})
	sum := sha256.Sum256([]byte("guest bench issue"))
	if _, err := conn.Exec(ctx, `UPDATE report_versions v SET status='issued', sections=$2::jsonb, snapshot='{}'::jsonb, snapshot_sha256=repeat('a',64), export_file_sha256=$3,
 issued_at=now(), issued_by=(SELECT u.id FROM users u WHERE u.org_id=v.org_id ORDER BY u.created_at LIMIT 1)
 FROM reports r WHERE r.org_id=v.org_id AND r.id=v.report_id AND r.id::text=$1 AND v.id=r.current_draft_version_id`, report.ID, string(sections), sum[:]); err != nil {
		return err
	}
	if _, err := conn.Exec(ctx, `UPDATE reports SET current_draft_version_id=NULL WHERE id::text=$1`, report.ID); err != nil {
		return err
	}
	shares := "/projects/" + project + "/shares"
	body := []byte(`{"kind":"package","package_id":"` + row.ID + `","firm_id":"` + inv.FirmID + `","shared_with":"bench@example.test"}`)
	for i := 0; i < n; i++ {
		out, err := a.timed(ctx, "team", http.MethodPost, shares, body, http.StatusCreated)
		if err != nil {
			return err
		}
		var created struct {
			Share struct {
				ID string `json:"id"`
			} `json:"share"`
			Link string `json:"link"`
		}
		if err := json.Unmarshal(out, &created); err != nil {
			return err
		}
		_, token, _ := strings.Cut(created.Link, "#t=")
		// Each sample is a different guest: its own browser and address, as
		// Caddy would forward it. The per-address exchange limit is 30.
		jar, _ := cookiejar.New(nil)
		g := &http.Client{Jar: jar, Timeout: 30 * time.Second}
		addr := fmt.Sprintf("203.0.113.%d", i%250+1)
		if err := a.guestTimed(ctx, g, addr, http.MethodPost, "/api/guest/session", []byte(`{"token":"`+token+`"}`), http.StatusNoContent); err != nil {
			return err
		}
		if err := a.guestTimed(ctx, g, addr, http.MethodGet, "/api/guest", nil, http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "procurement_board", http.MethodGet, "/projects/"+project+"/procurement/board", nil, http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "team", http.MethodGet, shares+"?package="+row.ID, nil, http.StatusOK); err != nil {
			return err
		}
		if _, err := a.timed(ctx, "team", http.MethodDelete, shares+"/"+created.Share.ID, nil, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

// guestTimed is one guest request, timed as guest_open.
func (a *apiClient) guestTimed(ctx context.Context, g *http.Client, addr, method, path string, body []byte, want int) error {
	req, err := http.NewRequestWithContext(ctx, method, a.origin+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Origin", a.origin)
	req.Header.Set("X-Forwarded-For", addr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	start := time.Now()
	resp, err := g.Do(req)
	if err != nil {
		return err
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	a.samples.Add("guest_open", time.Since(start))
	if resp.StatusCode != want {
		return fmt.Errorf("%s %s: status %d: %s", method, path, resp.StatusCode, bytes.TrimSpace(raw))
	}
	return nil
}

func guestUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
