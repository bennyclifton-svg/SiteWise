package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"sitewise/internal/reports"
)

// RestoreTableFacts covers every tenant table, including future migrations.
// Only the count and SHA-256 leave this operator-only check; never row content.
type RestoreTableFacts struct {
	Table  string `json:"table"`
	OrgID  string `json:"org_id"`
	Rows   int64  `json:"rows"`
	SHA256 string `json:"sha256"`
}

func restoreTables(ctx context.Context, tx pgx.Tx) ([]RestoreTableFacts, error) {
	rows, e := tx.Query(ctx, `SELECT c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind='r' AND EXISTS (SELECT 1 FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attname='org_id' AND NOT a.attisdropped) ORDER BY c.relname`)
	if e != nil {
		return nil, e
	}
	var tables []string
	for rows.Next() {
		var name string
		if e = rows.Scan(&name); e != nil {
			rows.Close()
			return nil, e
		}
		tables = append(tables, name)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	out := []RestoreTableFacts{}
	for _, table := range tables {
		// JSONB text gives deterministic object-key order. UTC removes host timezone
		// differences, and C collation prevents host locale changing row order.
		rows, e = tx.Query(ctx, `SELECT org_id::text,to_jsonb(t)::text FROM `+pgx.Identifier{"public", table}.Sanitize()+` t ORDER BY org_id,to_jsonb(t)::text COLLATE "C"`)
		if e != nil {
			return nil, e
		}
		var current RestoreTableFacts
		hash := sha256.New()
		flush := func() { current.SHA256 = hex.EncodeToString(hash.Sum(nil)); out = append(out, current) }
		for rows.Next() {
			var org, raw string
			if e = rows.Scan(&org, &raw); e != nil {
				rows.Close()
				return nil, e
			}
			if current.OrgID != org {
				if current.OrgID != "" {
					flush()
				}
				current = RestoreTableFacts{Table: table, OrgID: org}
				hash.Reset()
			}
			current.Rows++
			hash.Write([]byte(raw))
			hash.Write([]byte{'\n'})
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return nil, e
		}
		if current.OrgID != "" {
			flush()
		} else {
			out = append(out, RestoreTableFacts{Table: table, SHA256: hex.EncodeToString(hash.Sum(nil))})
		}
	}
	return out, nil
}
func restoreIssued(ctx context.Context, tx pgx.Tx) (invalid, missing int64, err error) {
	rows, err := tx.Query(ctx, `SELECT v.snapshot,v.snapshot_sha256,v.report_id::text,v.id::text,EXISTS(SELECT 1 FROM files f WHERE f.org_id=v.org_id AND f.project_id=v.project_id AND f.sha256=v.export_file_sha256) FROM report_versions v WHERE v.status='issued' ORDER BY v.org_id,v.id`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var hash, report, version string
		var linked bool
		if err = rows.Scan(&raw, &hash, &report, &version, &linked); err != nil {
			return
		}
		var snapshot reports.IssueSnapshot
		e := json.Unmarshal(raw, &snapshot)
		_, actual, canonicalErr := reports.CanonicalSnapshot(snapshot)
		if e != nil || canonicalErr != nil || actual != hash || snapshot.ReportID != report || snapshot.VersionID != version {
			invalid++
		}
		if !linked {
			missing++
		}
	}
	err = rows.Err()
	return
}
