package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"slices"
	"strings"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/store"
)

const canary = "CANARY_SECRET_VALUE"

func fullEnv() map[string]string {
	return map[string]string{
		"SITEWISE_DATABASE_URL":   "postgres://sitewise:" + canary + "@127.0.0.1:1/none",
		"SITEWISE_FILE_DIR":       "unused",
		"SITEWISE_SESSION_SECRET": canary,
		"SITEWISE_JEV_API_KEY":    canary,
		"SITEWISE_JEV_MODEL":      "jev-1.13.0",
	}
}

// A missing secret stops serve before it opens the database or a listener.
func TestServeRefusesToStartWithoutSecrets(t *testing.T) {
	for _, key := range []string{"SITEWISE_DATABASE_URL", "SITEWISE_FILE_DIR", "SITEWISE_SESSION_SECRET", "SITEWISE_JEV_API_KEY", "SITEWISE_JEV_MODEL"} {
		t.Run(key, func(t *testing.T) {
			env := fullEnv()
			delete(env, key)
			var stderr, stdout bytes.Buffer
			code := run([]string{"serve", "-addr", "127.0.0.1:0"}, func(k string) string { return env[k] }, &stderr, &stdout)
			if code == 0 {
				t.Fatal("serve started")
			}
			if !strings.Contains(stderr.String(), key) {
				t.Fatalf("stderr %q does not name %s", stderr.String(), key)
			}
			if strings.Contains(stderr.String()+stdout.String(), canary) {
				t.Fatal("output contains a secret value")
			}
			if strings.Contains(stdout.String(), "listening") {
				t.Fatal("serve opened a listener")
			}
		})
	}
	// Production also needs mail and its public origin.
	env := fullEnv()
	env["SITEWISE_ENV"] = "production"
	var stderr bytes.Buffer
	if code := run([]string{"serve"}, func(k string) string { return env[k] }, &stderr, &bytes.Buffer{}); code == 0 || !strings.Contains(stderr.String(), "SITEWISE_PUBLIC_ORIGIN") {
		t.Fatalf("production without origin: %d %q", code, stderr.String())
	}
}

func TestRestoreCheckNeedsOnlyDatabaseAndFiles(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"restore-check"}, func(string) string { return "" }, &stderr, &bytes.Buffer{})
	if code == 0 {
		t.Fatal("ran without configuration")
	}
	for _, key := range []string{"SITEWISE_DATABASE_URL", "SITEWISE_FILE_DIR"} {
		if !strings.Contains(stderr.String(), key) {
			t.Fatalf("stderr %q does not name %s", stderr.String(), key)
		}
	}
	if strings.Contains(stderr.String(), "JEV") {
		t.Fatalf("restore check asked for the Jev key: %q", stderr.String())
	}
}

func TestVerifyBlobsFindsMissingAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	blobs, err := files.Open(dir, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	good, err := blobs.Put(context.Background(), strings.NewReader("drawing A-100"))
	if err != nil {
		t.Fatal(err)
	}
	bad, err := blobs.Put(context.Background(), strings.NewReader("drawing A-101"))
	if err != nil {
		t.Fatal(err)
	}
	badPath, err := blobs.Path(bad.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(badPath, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badPath, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	gone := bytes.Repeat([]byte{0xcd}, 32)

	got := verifyBlobs(blobs, [][]byte{good.SHA256, bad.SHA256, gone})
	if got.Referenced != 3 || got.Verified != 1 {
		t.Fatalf("%+v", got)
	}
	if !slices.Equal(got.Missing, []string{hex.EncodeToString(gone)}) || !slices.Equal(got.Corrupt, []string{hex.EncodeToString(bad.SHA256)}) {
		t.Fatalf("missing %v corrupt %v", got.Missing, got.Corrupt)
	}
	if problems := restoreProblems(restoreReport{Blobs: got}); len(problems) != 2 {
		t.Fatalf("problems %v", problems)
	}
}

func TestRestoreProblemsAndComparison(t *testing.T) {
	src := restoreReport{
		Facts: store.RestoreFacts{
			Tables:      []store.RestoreTableFacts{{Table: "cost_values", OrgID: "a", Rows: 1, SHA256: "original"}},
			Orgs:        []store.OrgCounts{{OrgID: "a", Documents: 4, Files: 3, Events: 9}},
			ForeignKeys: 17,
			Migrations:  []string{"001_init.sql", "002_invite_role.sql"},
			CrossOrg:    []store.CrossOrg{{Reference: "events.document_id"}},
		},
		Blobs: blobCheck{Referenced: 3, Verified: 3, Digest: "abc"},
	}
	if p := restoreProblems(src); len(p) != 0 {
		t.Fatalf("clean report has problems %v", p)
	}
	if d := compareRestore(src, src); len(d) != 0 {
		t.Fatalf("identical reports differ %v", d)
	}

	restored := src
	restored.Facts.Orgs = []store.OrgCounts{{OrgID: "a", Documents: 3, Files: 3, Events: 9}}
	restored.Facts.ForeignKeys = 15
	restored.Facts.Unvalidated = 1
	restored.Facts.Migrations = []string{"001_init.sql"}
	restored.Facts.CrossOrg = []store.CrossOrg{{Reference: "events.document_id", Rows: 2}}
	restored.Blobs.Digest = "abd"

	problems := restoreProblems(restored)
	for _, want := range []string{"unvalidated", "events.document_id"} {
		if !slices.ContainsFunc(problems, func(s string) bool { return strings.Contains(s, want) }) {
			t.Fatalf("problems %v lack %q", problems, want)
		}
	}
	diffs := compareRestore(src, restored)
	for _, want := range []string{"documents", "foreign keys", "migrations", "blob digest"} {
		if !slices.ContainsFunc(diffs, func(s string) bool { return strings.Contains(s, want) }) {
			t.Fatalf("diffs %v lack %q", diffs, want)
		}
	}

	// A missing org on the restored side is a difference too.
	restored = src
	restored.Facts.Orgs = nil
	if d := compareRestore(src, restored); len(d) == 0 {
		t.Fatal("lost org not reported")
	}
}

func TestRestoreDetectsSameCountContentAndIssuedCorruption(t *testing.T) {
	source := restoreReport{Facts: store.RestoreFacts{Tables: []store.RestoreTableFacts{{Table: "cost_values", OrgID: "a", Rows: 1, SHA256: "before"}}}}
	changed := source
	changed.Facts.Tables = []store.RestoreTableFacts{{Table: "cost_values", OrgID: "a", Rows: 1, SHA256: "after"}}
	if len(compareRestore(source, changed)) == 0 {
		t.Fatal("same-count cost corruption ignored")
	}
	changed.Facts.InvalidIssuedSnapshots = 1
	changed.Facts.MissingIssuedBlobLinks = 1
	if p := restoreProblems(changed); len(p) != 2 {
		t.Fatal(p)
	}
	legacy := source
	legacy.Facts.Tables = nil
	if len(compareRestore(legacy, source)) == 0 {
		t.Fatal("old incomplete facts silently accepted")
	}
}
