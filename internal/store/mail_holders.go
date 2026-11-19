package store

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Who holds which revision (docs/plans/2026-11-17-correspondence-
// transmittals.md, Task 5). "Holds" is what SiteWise sent, never what a firm
// received some other way: the screen says "sent".

// DocumentSend is one transmittal that carried this document or an older
// revision of it.
type DocumentSend struct {
	MessageID  string    `json:"message_id"`
	Number     int       `json:"number"`
	SentAt     time.Time `json:"sent_at"`
	To         string    `json:"to"`
	Cc         string    `json:"cc"`
	DocumentID string    `json:"document_id,omitempty"`
	Revision   string    `json:"revision"`
	Status     string    `json:"send_status"`
	// This: the document itself went, not an older revision.
	This bool `json:"this"`
}

// Holder is someone sent an older revision and never sent this one.
type Holder struct {
	Address  string `json:"address"`
	Firm     string `json:"firm,omitempty"`
	FirmID   string `json:"firm_id,omitempty"`
	Revision string `json:"revision"`
	Number   int    `json:"number"`
}

// DocumentTransmittals is the document's sends and, for the newest
// revision, who holds an older one.
type DocumentTransmittals struct {
	Sends        []DocumentSend `json:"sends"`
	Newest       bool           `json:"newest"`
	HoldingOlder []Holder       `json:"holding_older"`
}

// TransmittalsOfDocument reads them in one transaction. The chain is the
// document and every revision it supersedes; a part whose document was
// deleted still counts by the number it went with.
func (s *Store) TransmittalsOfDocument(ctx context.Context, org, project, document string) (DocumentTransmittals, error) {
	out := DocumentTransmittals{Sends: []DocumentSend{}, HoldingOlder: []Holder{}}
	if !validUUID(org) || !validUUID(project) || !validUUID(document) {
		return out, ErrNotFound
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var number string
	err = tx.QueryRow(ctx, `SELECT COALESCE(NULLIF((SELECT value FROM decisions k WHERE k.org_id=d.org_id AND k.document_id=d.id AND k.field='number'),''),d.document_number,''),
 NOT EXISTS(SELECT 1 FROM supersessions s WHERE s.org_id=d.org_id AND s.prior_document_id=d.id)
FROM documents d WHERE d.org_id=$1::uuid AND d.project_id=$2::uuid AND d.id=$3::uuid`, org, project, document).Scan(&number, &out.Newest)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	rows, err := tx.Query(ctx, `WITH RECURSIVE chain(id) AS (
 SELECT $3::uuid
 UNION SELECT s.prior_document_id FROM supersessions s JOIN chain c ON s.org_id=$1::uuid AND s.document_id=c.id
)
SELECT m.id::text, m.number, m.received_at, m.to_addrs, m.cc_addrs, COALESCE(p.document_id::text,''), p.doc_revision, m.send_status,
 COALESCE(p.document_id=$3::uuid,false)
FROM mail_parts p JOIN mail_messages m ON m.org_id=p.org_id AND m.id=p.message_id
WHERE m.org_id=$1::uuid AND m.project_id=$2::uuid AND m.direction='out' AND m.number IS NOT NULL
  AND (p.document_id IN (SELECT id FROM chain) OR ($4<>'' AND p.document_id IS NULL AND upper(p.doc_number)=upper($4)))
ORDER BY m.number`, org, project, document, number)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var d DocumentSend
		if err := rows.Scan(&d.MessageID, &d.Number, &d.SentAt, &d.To, &d.Cc, &d.DocumentID, &d.Revision, &d.Status, &d.This); err != nil {
			rows.Close()
			return out, err
		}
		out.Sends = append(out.Sends, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	if !out.Newest {
		return out, nil
	}
	// Who was sent an older revision and never this one, latest send each.
	hasThis := map[string]bool{}
	older := map[string]Holder{}
	for _, d := range out.Sends {
		if d.Status == "failed" {
			continue
		}
		for _, a := range addressList(d.To + "," + d.Cc) {
			if d.This {
				hasThis[a] = true
				continue
			}
			if h, ok := older[a]; !ok || d.Number > h.Number {
				older[a] = Holder{Address: a, Revision: d.Revision, Number: d.Number}
			}
		}
	}
	// The team's own addresses (the PM is Cc'd on every send) hold nothing.
	team := map[string]bool{}
	urows, err := tx.Query(ctx, `SELECT lower(email) FROM users WHERE org_id=$1::uuid`, org)
	if err != nil {
		return out, err
	}
	for urows.Next() {
		var e string
		if err := urows.Scan(&e); err != nil {
			urows.Close()
			return out, err
		}
		team[e] = true
	}
	urows.Close()
	var addrs []string
	for a := range older {
		if !hasThis[a] && !team[a] {
			addrs = append(addrs, a)
		}
	}
	sort.Strings(addrs)
	if len(addrs) == 0 {
		return out, nil
	}
	firms := map[string][2]string{}
	frows, err := tx.Query(ctx, `SELECT lower(email), id::text, name FROM firms WHERE org_id=$1::uuid AND email<>'' AND lower(email) = ANY($2::text[])`, org, addrs)
	if err != nil {
		return out, err
	}
	for frows.Next() {
		var e, id, name string
		if err := frows.Scan(&e, &id, &name); err != nil {
			frows.Close()
			return out, err
		}
		firms[e] = [2]string{id, name}
	}
	frows.Close()
	for _, a := range addrs {
		h := older[a]
		if f, ok := firms[strings.ToLower(a)]; ok {
			h.FirmID, h.Firm = f[0], f[1]
		}
		out.HoldingOlder = append(out.HoldingOlder, h)
	}
	return out, frows.Err()
}
