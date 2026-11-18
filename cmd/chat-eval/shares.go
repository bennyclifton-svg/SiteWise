package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Guest links in the key's fixtures (docs/plans/2026-11-17-guest-access.md,
// "From chat"). A case that needs "shares" gets the fixture's issued RFPs,
// the links its invited firms hold (opened or not) and its documents links.

// linkFix is an invited firm's link to its package's issued RFP or RFT.
type linkFix struct {
	Firm, To string
	Opened   bool
}

// docLinkFix is a documents link to one fixture document.
type docLinkFix struct {
	Document, To string
}

// issueReport marks a package's RFP or RFT draft issued, as the issue route
// leaves it. The RFP issue route's clause gates hold until the owner marks
// a discipline reviewed; the key needs an issue to share, not those gates.
// The issued PDF is never downloaded here, so its hash names no file.
func (l *live) issueReport(ctx context.Context, report string) error {
	if l.dsn == "" {
		return errors.New("issuing a fixture report needs the database address")
	}
	conn, err := pgx.Connect(ctx, l.dsn)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	sum := sha256.Sum256([]byte("chat-eval issued request " + report))
	tag, err := conn.Exec(ctx, `UPDATE report_versions v SET status='issued', snapshot='{}'::jsonb, snapshot_sha256=repeat('a',64), export_file_sha256=$2,
 issued_at=now(), issued_by=(SELECT u.id FROM users u WHERE u.org_id=v.org_id ORDER BY u.created_at LIMIT 1)
 FROM reports r WHERE r.org_id=v.org_id AND r.id=v.report_id AND r.id::text=$1 AND v.id=r.current_draft_version_id`, report, sum[:])
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("report %s has no draft to issue", report)
	}
	_, err = conn.Exec(ctx, `UPDATE reports SET current_draft_version_id=NULL WHERE id::text=$1`, report)
	return err
}

// seedShares issues the packages marked Issued and makes the fixture's links
// through the PM's own share route; an opened link is exchanged once, as a
// guest's click does.
func (s *session) seedShares(ctx context.Context, f fixture) error {
	base := "/api/projects/" + s.project
	var board struct {
		Packages map[string][]struct {
			FirmID   string `json:"firm_id"`
			FirmName string `json:"firm_name"`
		} `json:"packages"`
	}
	if err := s.must(http.MethodGet, base+"/procurement/board", nil, &board); err != nil {
		return err
	}
	pkgs, err := s.l.st.ReadPackages(ctx, s.org, s.project)
	if err != nil {
		return err
	}
	for _, p := range f.Packages {
		if !p.Issued {
			continue
		}
		id := ""
		for _, k := range pkgs {
			if k.DisciplineID == p.Discipline && k.RetiredAt == nil {
				id = k.ID
			}
		}
		kind := "rfp"
		if strings.HasPrefix(p.Discipline, "trade.") {
			kind = "rft"
		}
		var report struct {
			ID string `json:"id"`
		}
		if err := s.must(http.MethodPost, base+"/reports", map[string]string{"kind": kind, "package_id": id}, &report); err != nil {
			return err
		}
		if err := s.must(http.MethodPost, "/api/reports/"+report.ID+"/draft", map[string]bool{"use_last_completed": true}, nil); err != nil {
			return err
		}
		if err := s.l.issueReport(ctx, report.ID); err != nil {
			return err
		}
		for _, link := range p.Links {
			firm := ""
			for _, c := range board.Packages[id] {
				if c.FirmName == link.Firm {
					firm = c.FirmID
				}
			}
			if err := s.share(map[string]any{"kind": "package", "package_id": id, "firm_id": firm, "shared_with": link.To}, link.Opened); err != nil {
				return err
			}
		}
	}
	if len(f.DocumentLinks) == 0 {
		return nil
	}
	docs, err := s.l.st.ProjectDocumentViews(ctx, s.org, s.project)
	if err != nil {
		return err
	}
	for _, link := range f.DocumentLinks {
		id := ""
		for _, d := range docs {
			for _, fd := range d.Fields {
				if fd.Field == "title" && fd.Value == link.Document {
					id = d.ID
				}
			}
		}
		if err := s.share(map[string]any{"kind": "documents", "document_ids": []string{id}, "shared_with": link.To}, false); err != nil {
			return err
		}
	}
	return nil
}

// share makes one link as the PM; opened exchanges it once as a guest.
func (s *session) share(body map[string]any, opened bool) error {
	var out struct {
		Link string `json:"link"`
	}
	if err := s.must(http.MethodPost, "/api/projects/"+s.project+"/shares", body, &out); err != nil {
		return err
	}
	if !opened {
		return nil
	}
	_, token, _ := strings.Cut(out.Link, "#t=")
	raw, _ := json.Marshal(map[string]string{"token": token})
	req, err := http.NewRequest(http.MethodPost, s.l.url+"/api/guest/session", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Origin", s.l.url)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("guest exchange = %d", resp.StatusCode)
	}
	return nil
}
