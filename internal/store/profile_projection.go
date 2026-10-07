package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/profile"
)

// Every saved column participates in the hash. The org/project are the query
// boundary; the part/key identify a row. No field is inferred from a prior row.
type profileProjectionRow struct {
	SiteID       string                `json:"site_id"`
	PartID       string                `json:"part_id"`
	Key          string                `json:"key"`
	Value        string                `json:"value"`
	Band         string                `json:"band"`
	Assertion    string                `json:"assertion"`
	Note         string                `json:"note"`
	Tenders      string                `json:"tenders"`
	Sources      []profile.Source      `json:"sources"`
	Alternatives []profile.Alternative `json:"alternatives"`
	Derived      *profile.Derived      `json:"derived"`
	Scope        string                `json:"scope"`
	Origin       string                `json:"origin"`
	ReviewStatus string                `json:"review_status"`
	Meaning      string                `json:"meaning"`
	ValueState   string                `json:"value_state"`
	UserVersion  int64                 `json:"user_version"`
	Hash         string                `json:"projection_hash,omitempty"`
}

func profileProjection(site string, r profile.Row) (profileProjectionRow, error) {
	p := profileProjectionRow{SiteID: site, PartID: r.PartID, Key: r.Key, Value: r.Value, Band: r.Band, Assertion: r.Assertion,
		Note: cutRunes(r.Note, 120), Tenders: r.Tenders, Sources: nonNil(r.Sources), Alternatives: nonNilAlts(r.Alternatives), Derived: r.Derived,
		Scope: orDefault(r.Scope, "project"), Origin: orDefault(r.Origin, "document"), ReviewStatus: orDefault(r.ReviewStatus, "proposed"),
		Meaning: orDefault(r.Meaning, "stated"), ValueState: orDefault(r.ValueState, "set"), UserVersion: r.UserVersion}
	raw, err := json.Marshal(p)
	if err != nil {
		return p, err
	}
	hash := sha256.Sum256(raw)
	p.Hash = hex.EncodeToString(hash[:])
	return p, nil
}

// The caller holds the project lock and commits this projection with its build,
// works and proposals. Removed keys are deleted; changed rows retain all checks.
func writeProfileProjection(ctx context.Context, tx pgx.Tx, org, project, site string, rows []profile.Row) error {
	type key struct{ part, name string }
	old := map[key]string{}
	stored, err := tx.Query(ctx, `SELECT part_id::text,key,projection_hash FROM profile_rows WHERE org_id=$1::uuid AND project_id=$2::uuid`, org, project)
	if err != nil {
		return err
	}
	for stored.Next() {
		var k key
		var hash string
		if err := stored.Scan(&k.part, &k.name, &hash); err != nil {
			stored.Close()
			return err
		}
		old[k] = hash
	}
	stored.Close()
	if err := stored.Err(); err != nil {
		return err
	}
	changed := []profileProjectionRow{}
	seen := map[key]bool{}
	for _, row := range rows {
		k := key{row.PartID, row.Key}
		if seen[k] {
			return fmt.Errorf("duplicate profile row %s", row.Key)
		}
		seen[k] = true
		p, err := profileProjection(site, row)
		if err != nil {
			return err
		}
		if old[k] != p.Hash {
			changed = append(changed, p)
		}
		delete(old, k)
	}
	if len(old) > 0 {
		parts, keys := make([]string, 0, len(old)), make([]string, 0, len(old))
		for k := range old {
			parts, keys = append(parts, k.part), append(keys, k.name)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM profile_rows r USING unnest($3::uuid[],$4::text[]) AS removed(part_id,key)
WHERE r.org_id=$1::uuid AND r.project_id=$2::uuid AND r.part_id=removed.part_id AND r.key=removed.key`, org, project, parts, keys); err != nil {
			return err
		}
	}
	if len(changed) == 0 {
		return nil
	}
	raw, err := json.Marshal(changed)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO profile_rows (org_id,project_id,site_id,part_id,key,value,band,assertion,note,tenders,sources,alternatives,derived,scope,origin,review_status,meaning,value_state,user_version,projection_hash)
SELECT $1::uuid,$2::uuid,p.site_id,p.part_id,p.key,p.value,p.band,p.assertion,p.note,p.tenders,p.sources,p.alternatives,p.derived,p.scope,p.origin,p.review_status,p.meaning,p.value_state,p.user_version,p.projection_hash
FROM jsonb_to_recordset($3::jsonb) AS p(site_id uuid,part_id uuid,key text,value text,band text,assertion text,note text,tenders text,sources jsonb,alternatives jsonb,derived jsonb,scope text,origin text,review_status text,meaning text,value_state text,user_version bigint,projection_hash text)
ON CONFLICT (org_id,project_id,part_id,key) DO UPDATE SET
site_id=EXCLUDED.site_id,value=EXCLUDED.value,band=EXCLUDED.band,assertion=EXCLUDED.assertion,note=EXCLUDED.note,tenders=EXCLUDED.tenders,sources=EXCLUDED.sources,alternatives=EXCLUDED.alternatives,derived=EXCLUDED.derived,scope=EXCLUDED.scope,origin=EXCLUDED.origin,review_status=EXCLUDED.review_status,meaning=EXCLUDED.meaning,value_state=EXCLUDED.value_state,user_version=EXCLUDED.user_version,projection_hash=EXCLUDED.projection_hash`, org, project, raw)
	return err
}
