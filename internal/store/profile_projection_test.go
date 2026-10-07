package store

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"sitewise/internal/profile"
)

func TestProfileProjectionDeltaPreservesEvidenceAndIsolation(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL required")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Database != "sitewise_test" {
		t.Fatal("dedicated test database required", err)
	}
	s, err := Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `CREATE TEMP TABLE profile_rows (LIKE public.profile_rows INCLUDING ALL) ON COMMIT DROP;
CREATE TEMP TABLE projection_changes (operation text,key text) ON COMMIT DROP;
CREATE FUNCTION pg_temp.record_profile_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO projection_changes VALUES (TG_OP,COALESCE(NEW.key,OLD.key)); RETURN NULL; END $$;
CREATE TRIGGER profile_change AFTER INSERT OR UPDATE OR DELETE ON profile_rows FOR EACH ROW EXECUTE FUNCTION pg_temp.record_profile_change()`); err != nil {
		t.Fatal(err)
	}
	const org = "a1100000-0000-4000-8000-000000000001"
	const project = "a1100000-0000-4000-8000-000000000002"
	const site = "a1100000-0000-4000-8000-000000000003"
	const part = "a1100000-0000-4000-8000-000000000004"
	const other = "a1100000-0000-4000-8000-000000000005"
	rows := []profile.Row{{PartID: part, Key: "one", Value: "yes", Band: "green", Sources: []profile.Source{{DocumentID: "source", Revision: "A", FileSHA256: "first", Page: 1}}}, {PartID: part, Key: "two", Band: "blank"}}
	write := func(org string, rows []profile.Row) {
		t.Helper()
		if err := writeProfileProjection(ctx, tx, org, project, site, rows); err != nil {
			t.Fatal(err)
		}
	}
	write(org, rows)
	write(other, rows)
	if _, err = tx.Exec(ctx, `TRUNCATE projection_changes`); err != nil {
		t.Fatal(err)
	}
	write(org, rows)
	var count int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM projection_changes`).Scan(&count); err != nil || count != 0 {
		t.Fatal("unchanged rows rewritten", count, err)
	}
	rows[0].Sources[0].Revision = "B"
	rows[0].Sources[0].FileSHA256 = "second"
	rows[0].Origin = "user"
	rows[0].ReviewStatus = "accepted_for_planning"
	rows[0].UserVersion = 2
	rows = append(rows[:1], profile.Row{PartID: part, Key: "three", Value: "unknown", ValueState: "unknown", Band: "user", Note: "Explicit uncertainty"})
	write(org, rows)
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM projection_changes`).Scan(&count); err != nil || count != 3 {
		t.Fatal("expected exactly insert/update/delete", count, err)
	}
	var revision, origin string
	var version int64
	if err = tx.QueryRow(ctx, `SELECT sources->0->>'revision',origin,user_version FROM profile_rows WHERE org_id=$1 AND key='one'`, org).Scan(&revision, &origin, &version); err != nil || revision != "B" || origin != "user" || version != 2 {
		t.Fatal("provenance not saved", revision, origin, version, err)
	}
	if err = tx.QueryRow(ctx, `SELECT sources->0->>'revision' FROM profile_rows WHERE org_id=$1 AND key='one'`, other).Scan(&revision); err != nil || revision != "A" {
		t.Fatal("other org changed", revision, err)
	}
	if _, err = tx.Exec(ctx, `SAVEPOINT failed_projection`); err != nil {
		t.Fatal(err)
	}
	invalid := append([]profile.Row(nil), rows...)
	invalid[0].Band = "invalid"
	if err = writeProfileProjection(ctx, tx, org, project, site, invalid); err == nil || !strings.Contains(err.Error(), "check") {
		t.Fatal("database checks bypassed", err)
	}
	if _, err = tx.Exec(ctx, `ROLLBACK TO SAVEPOINT failed_projection`); err != nil {
		t.Fatal(err)
	}
	write(org, rows)
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM projection_changes`).Scan(&count); err != nil || count != 3 {
		t.Fatal("failed projection advanced hash", count, err)
	}
	// Migration compatibility: an old row with an empty hash is refreshed once.
	if _, err = tx.Exec(ctx, `UPDATE profile_rows SET projection_hash='' WHERE org_id=$1 AND key='one'; TRUNCATE projection_changes`, pgx.QueryExecModeSimpleProtocol, org); err != nil {
		t.Fatal(err)
	}
	write(org, rows)
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM projection_changes`).Scan(&count); err != nil || count != 1 {
		t.Fatal("legacy row not refreshed", count, err)
	}
}
