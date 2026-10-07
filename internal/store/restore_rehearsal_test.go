package store_test

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sitewise/internal/delivery"
	"sitewise/internal/files"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"testing"
	"time"
)

// Opt-in operator rehearsal: creates its own uniquely named database and never
// restores into or drops an existing database. The ordinary suite skips it.
func TestNextWaveDumpRestoreRehearsal(t *testing.T) {
	bin := os.Getenv("SITEWISE_RESTORE_PG_BIN")
	if bin == "" {
		t.Skip("set SITEWISE_RESTORE_PG_BIN for local dump/restore rehearsal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	s, b, part := workStore(t)
	pkg, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Restore engineering", LifecycleStatus: "planned"})
	if e != nil {
		t.Fatal(e)
	}
	w, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "hydraulic.gas", Action: "repair", Title: "Restore gas work"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SplitWorkItem(ctx, orgA, projectA, w.ID, userA, works.SplitRequest{Version: w.Version, Children: []works.SplitChild{{Title: "Branch"}, {Title: "Valve"}}}); e != nil {
		t.Fatal(e)
	}
	first, e := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "activity", Title: "Survey", PackageID: pkg.ID}})
	if e != nil {
		t.Fatal(e)
	}
	second, e := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "milestone", Title: "Release"}})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.WriteDeliveryDependency(ctx, orgA, projectA, userA, store.DeliveryDependencyInput{Dependency: delivery.Dependency{PredecessorID: first.ID, SuccessorID: second.ID, LagDays: 3}, Version: second.Version}, false); e != nil {
		t.Fatal(e)
	}
	plan, e := s.CreateCostItem(ctx, orgA, projectA, userA, ci("Restore allowance"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(plan)); e != nil {
		t.Fatal(e)
	}
	if e = s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); e != nil {
		t.Fatal(e)
	}
	report, e := s.CreateReport(ctx, orgA, projectA, userA, "pmp", "")
	if e != nil {
		t.Fatal(e)
	}
	draft, e := s.RefreshReport(ctx, orgA, report.ID, userA, "restore-test", true)
	if e != nil {
		t.Fatal(e)
	}
	pool := rawPool(t)
	if _, e = pool.Exec(ctx, `UPDATE report_versions SET sections='[{"id":"brief","title":"Brief","blocks":[{"id":"project","text":"Restore fixture","label":"U","basis":{}}]}]'::jsonb WHERE org_id=$1::uuid AND id=$2::uuid`, orgA, draft.ID); e != nil {
		t.Fatal(e)
	}
	blobRoot := t.TempDir()
	blobs, e := files.Open(blobRoot, 20<<20)
	if e != nil {
		t.Fatal(e)
	}
	issued, e := s.IssueReport(ctx, orgA, report.ID, userA, "restore-test", store.IssueOptions{Version: draft.Version, ReportingDate: "2026-10-07", AcceptStale: true, StaleReason: "Synthetic restore fixture"}, blobs)
	if e != nil {
		t.Fatal(e)
	}
	before, e := s.RestoreFacts(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if before.InvalidIssuedSnapshots != 0 || before.MissingIssuedBlobLinks != 0 {
		t.Fatal("source issue invalid")
	}
	source, e := url.Parse(testDSN(t))
	if e != nil {
		t.Fatal(e)
	}
	adminURL := *source
	adminURL.Path = "/postgres"
	admin, e := pgx.Connect(ctx, adminURL.String())
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(context.Background())
	name := fmt.Sprintf("sitewise_restore_%d", time.Now().UnixNano())
	ident := pgx.Identifier{name}.Sanitize()
	if _, e = admin.Exec(ctx, "CREATE DATABASE "+ident); e != nil {
		t.Fatal(e)
	}
	// Only this successfully created name is removed; never a caller-supplied DB.
	defer func() {
		if _, e := admin.Exec(context.Background(), "DROP DATABASE "+ident+" WITH (FORCE)"); e != nil {
			t.Errorf("cleanup owned rehearsal database: %v", e)
		}
	}()
	target := *source
	target.Path = "/" + name
	dump := filepath.Join(t.TempDir(), "next-wave.dump")
	command := func(tool string, args ...string) {
		t.Helper()
		exe := filepath.Join(bin, tool)
		if _, e := os.Stat(exe + ".exe"); e == nil {
			exe += ".exe"
		}
		cmd := exec.CommandContext(ctx, exe, args...)
		if out, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("%s: %v %s", tool, e, out)
		}
	}
	command("pg_dump", "--format=custom", "--file", dump, "--dbname", source.String())
	command("pg_restore", "--exit-on-error", "--no-owner", "--dbname", target.String(), dump)
	restored, e := store.Open(ctx, target.String())
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	after, e := restored.RestoreFacts(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("restored row counts, content digests, constraints or issue facts differ")
	}
	saved, e := restored.ReadIssuedReport(ctx, orgA, report.ID, issued.ID)
	if e != nil {
		t.Fatal(e)
	}
	// Copy issued bytes to a distinct restore blob directory and rehash on open.
	restoredRoot := t.TempDir()
	restoredBlobs, e := files.Open(restoredRoot, 20<<20)
	if e != nil {
		t.Fatal(e)
	}
	src, e := blobs.Open(saved.ExportSHA256)
	if e != nil {
		t.Fatal(e)
	}
	defer src.Close()
	copy, e := restoredBlobs.Put(ctx, src)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(copy.SHA256, saved.ExportSHA256) {
		t.Fatal("restored blob hash changed")
	}
	verified, e := restoredBlobs.Open(saved.ExportSHA256)
	if e != nil {
		t.Fatal(e)
	}
	verified.Close()
	// A hash existing elsewhere in the tenant must not satisfy this report's link.
	const otherProject = "e967a804-1b12-4a39-9d6e-cc9e5f661998"
	if e = restored.CreateProject(ctx, orgA, otherProject, "Wrong project blob test"); e != nil {
		t.Fatal(e)
	}
	targetConn, e := pgx.Connect(ctx, target.String())
	if e != nil {
		t.Fatal(e)
	}
	defer targetConn.Close(context.Background())
	if _, e = targetConn.Exec(ctx, `UPDATE files SET project_id=$3::uuid WHERE org_id=$1::uuid AND sha256=$2`, orgA, saved.ExportSHA256, otherProject); e != nil {
		t.Fatal(e)
	}
	invalid, e := restored.RestoreFacts(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if invalid.MissingIssuedBlobLinks != 1 {
		t.Fatal("wrong-project export link accepted", invalid.MissingIssuedBlobLinks)
	}
	t.Logf("dump/restore identical: %d tenant table/org digests, %d validated foreign keys; issued snapshot and blob link preserved", len(after.Tables), after.ForeignKeys)
}
