package store

import (
	"context"
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
// filed message in that message's thread, oldest first. Mail of another org
// or project never joins, because every join carries both.
func rfiThreads(ctx context.Context, tx pgx.Tx, org, project string) (map[string][]delivery.ThreadMessage, error) {
	rows, err := tx.Query(ctx, `SELECT item, id, direction, sender, sender_name, to_addrs, number, received_at, send_status, body, documents, body_document, reading FROM (
 SELECT i.id::text AS item, m.id::text AS id, m.direction, m.sender, m.sender_name, m.to_addrs, m.number, m.received_at, m.send_status,
  CASE WHEN m.direction='in' THEN left(m.body, 1200) ELSE '' END AS body,
  (SELECT count(*) FROM mail_parts p WHERE p.org_id=m.org_id AND p.message_id=m.id
    AND CASE WHEN m.direction='out' THEN p.document_id IS NOT NULL ELSE p.n>0 END)::int AS documents,
  COALESCE((SELECT p.document_id::text FROM mail_parts p WHERE p.org_id=m.org_id AND p.message_id=m.id AND p.n=0 AND m.direction='in'), '') AS body_document,
  COALESCE((SELECT r.id::text FROM mail_parts p JOIN delivery_readings r ON r.org_id=p.org_id AND r.document_id=p.document_id
    WHERE p.org_id=m.org_id AND p.message_id=m.id AND p.n=0 AND m.direction='in' AND r.project_id=i.project_id AND r.item_id=i.id AND r.status='proposed' AND r.field='answers' LIMIT 1), '') AS reading,
  row_number() OVER (PARTITION BY i.id ORDER BY m.received_at, m.id) AS n
 FROM project_delivery_items i
 JOIN mail_messages sent ON sent.org_id=i.org_id AND sent.project_id=i.project_id AND sent.id::text=i.details->>'sent_message_id'
 JOIN mail_messages m ON m.org_id=sent.org_id AND m.project_id=sent.project_id AND m.thread_id=sent.thread_id AND m.status='filed'
 WHERE i.org_id=$1::uuid AND i.project_id=$2::uuid AND i.kind='rfi' AND i.retired_at IS NULL AND COALESCE(i.details->>'sent_message_id','')<>''
) t WHERE n <= $3 ORDER BY item, received_at, id`, org, project, threadLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]delivery.ThreadMessage{}
	for rows.Next() {
		var item, body, sender, name, to string
		var at time.Time
		var m delivery.ThreadMessage
		if err := rows.Scan(&item, &m.ID, &m.Direction, &sender, &name, &to, &m.Number, &at, &m.SendStatus, &body, &m.Documents, &m.BodyDocument, &m.Reading); err != nil {
			return nil, err
		}
		m.At = at.UTC().Format(time.RFC3339)
		if m.Direction == "out" {
			m.With = to
		} else {
			m.With = sender
			if name != "" {
				m.With = name
			}
			m.Excerpt = replyExcerpt(body)
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
