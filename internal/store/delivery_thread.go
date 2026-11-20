package store

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/delivery"
)

// The RFI sheet's history (register panel, 20 November 2026): each RFI that
// went out through Send carries the filed messages of its thread, read in
// one query for the whole register, so the screen never asks per row.

// threadLimit bounds the messages kept per RFI; a thread longer than this
// shows its first ones, the send and the early replies.
const threadLimit = 50

// excerptRunes bounds a reply's quoted lines.
const excerptRunes = 280

// rfiThreads reads every RFI's thread: the message it went out on and every
// filed message in that message's thread, oldest first. The RFIs and the
// messages they went out on come from the rows already read, so the query
// starts from those ids; mail of another org or project never joins,
// because every join carries both. A reply's possible answer is matched in
// Go from the readings already read.
func rfiThreads(ctx context.Context, tx pgx.Tx, org, project string, items []delivery.Item, readings map[string][]delivery.Reading) (map[string][]delivery.ThreadMessage, error) {
	out := map[string][]delivery.ThreadMessage{}
	var ids, sent []string
	for _, it := range items {
		if it.Kind != "rfi" {
			continue
		}
		if m := delivery.RFI(it.Content).SentMessageID; validUUID(m) {
			ids, sent = append(ids, it.ID), append(sent, m)
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := tx.Query(ctx, threadSQL, org, project, ids, sent, threadLimit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item, body, sender, name, to string
		var docs, attachments int
		var at time.Time
		var m delivery.ThreadMessage
		if err := rows.Scan(&item, &m.ID, &m.Direction, &sender, &name, &to, &m.Number, &at, &m.SendStatus, &body, &docs, &attachments, &m.BodyDocument); err != nil {
			return nil, err
		}
		m.At = at.UTC().Format(time.RFC3339)
		if m.Direction == "out" {
			m.With, m.Documents, m.BodyDocument = to, docs, ""
		} else {
			m.With, m.Documents = sender, attachments
			if name != "" {
				m.With = name
			}
			m.Excerpt = replyExcerpt(body)
			for _, r := range readings[item] {
				if r.DocumentID == m.BodyDocument && m.BodyDocument != "" && r.Field != "evidence" {
					m.Reading = r.ID
					break
				}
			}
		}
		out[item] = append(out[item], m)
	}
	return out, rows.Err()
}

// replyExcerpt is a reply's first lines as plain text: up to three lines
// that are not blank, stopping where the quoted earlier message starts.
func replyExcerpt(body string) string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ">") || (strings.HasPrefix(line, "On ") && strings.HasSuffix(line, "wrote:")) || line == "--" {
			break
		}
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) == 3 {
			break
		}
	}
	return truncateRunes(strings.Join(lines, "\n"), excerptRunes)
}

// threadSQL starts from the RFIs' sent messages; each message's parts are
// counted by a lateral lookup on mail_parts' key, so the plan does not
// depend on table statistics.
const threadSQL = `WITH s(item, id) AS (SELECT * FROM unnest($3::text[], $4::uuid[])),
t AS (
 SELECT s.item, m.id, m.direction, m.sender, m.sender_name, m.to_addrs, m.number, m.received_at, m.send_status,
  CASE WHEN m.direction='in' THEN left(m.body, 1200) ELSE '' END AS body,
  row_number() OVER (PARTITION BY s.item ORDER BY m.received_at, m.id) AS n
 FROM s JOIN mail_messages sent ON sent.org_id=$1::uuid AND sent.project_id=$2::uuid AND sent.id=s.id
 JOIN mail_messages m ON m.org_id=sent.org_id AND m.project_id=sent.project_id AND m.thread_id=sent.thread_id AND m.status='filed')
SELECT t.item, t.id::text, t.direction, t.sender, t.sender_name, t.to_addrs, t.number, t.received_at, t.send_status, t.body,
 p.docs::int, p.atts::int, COALESCE(p.body_document,'')
FROM t CROSS JOIN LATERAL (
 SELECT count(*) FILTER (WHERE x.document_id IS NOT NULL) AS docs, count(*) FILTER (WHERE x.n>0) AS atts,
  max(x.document_id::text) FILTER (WHERE x.n=0) AS body_document
 FROM mail_parts x WHERE x.org_id=$1::uuid AND x.message_id=t.id) p
WHERE t.n <= $5 ORDER BY t.item, t.received_at, t.id`

// formerMember is how a row names a member no longer in the organisation.
const formerMember = "a former team member"

// rowRaisers names the team member who added each row (its provenance
// actor), read for the whole register in one query within the org. A member
// who has left, or whose account is gone, is "a former team member".
func rowRaisers(ctx context.Context, tx pgx.Tx, org string, items []delivery.Item) (map[string]string, error) {
	actors := map[string]string{}
	var ids []string
	for _, it := range items {
		var p struct {
			Actor string `json:"actor"`
		}
		if json.Unmarshal(it.Provenance, &p) != nil || !validUUID(p.Actor) {
			continue
		}
		if _, seen := actors[p.Actor]; !seen {
			ids = append(ids, p.Actor)
		}
		actors[p.Actor] = formerMember
	}
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := tx.Query(ctx, `SELECT u.id::text, u.email FROM users u
 JOIN memberships m ON m.org_id=u.org_id AND m.user_id=u.id
 WHERE u.org_id=$1::uuid AND u.id::text = ANY($2::text[])`, org, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, email string
		if err := rows.Scan(&id, &email); err != nil {
			return nil, err
		}
		actors[id] = email
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, it := range items {
		var p struct {
			Actor string `json:"actor"`
		}
		if json.Unmarshal(it.Provenance, &p) == nil && actors[p.Actor] != "" {
			out[it.ID] = actors[p.Actor]
		}
	}
	return out, nil
}
