package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/profile"
)

// The version forms part of the key: when explicitly using a stale profile,
// never attribute its old value to the author of a newer user input.
func reportUserAudit(ctx context.Context, tx pgx.Tx, org, project string) (map[string]json.RawMessage, error) {
	rows, err := tx.Query(ctx, `SELECT scope,part_id::text,key,version,jsonb_build_object('user_id',user_id,'updated_at',updated_at,'note',note,'provenance',provenance)
 FROM profile_user_values WHERE org_id=$1::uuid AND ((scope='project' AND project_id=$2::uuid) OR (scope='site' AND site_id=(SELECT site_id FROM projects WHERE org_id=$1::uuid AND id=$2::uuid)))`, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var scope, part, key string
		var version int64
		var raw []byte
		if err := rows.Scan(&scope, &part, &key, &version, &raw); err != nil {
			return nil, err
		}
		out[reportAuditKey(scope, part, key, version)] = raw
	}
	return out, rows.Err()
}

func reportAuditKey(scope, part, key string, version int64) string {
	return fmt.Sprintf("%s/%s/%s/%d", scope, part, key, version)
}

func reportRowBasis(row profile.Row, audit map[string]json.RawMessage) (json.RawMessage, error) {
	raw, err := json.Marshal(row)
	if err != nil {
		return nil, err
	}
	if row.UserVersion == 0 {
		return raw, nil
	}
	actor, ok := audit[reportAuditKey(row.Scope, row.PartID, row.Key, row.UserVersion)]
	if !ok {
		return raw, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	fields["audit"] = actor
	return json.Marshal(fields)
}
