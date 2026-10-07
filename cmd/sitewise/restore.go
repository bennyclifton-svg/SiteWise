package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/store"
)

// restoreReport is what one restore-check run found. It is written on the
// source host and compared on the restored host; it holds counts, catalog
// state and content hashes, never document text or names.
type restoreReport struct {
	CheckedAt time.Time          `json:"checked_at"`
	Facts     store.RestoreFacts `json:"facts"`
	Blobs     blobCheck          `json:"blobs"`
}

// blobCheck is the content-hash check of every blob a files row references.
type blobCheck struct {
	Referenced int `json:"referenced"`
	Verified   int `json:"verified"`
	// Digest is SHA-256 over the sorted referenced hashes, so two hosts
	// agree on the set without listing it.
	Digest  string   `json:"digest"`
	Missing []string `json:"missing"`
	Corrupt []string `json:"corrupt"`
}

// runRestoreCheck verifies a restored database and blob directory. It needs
// only the database URL and file directory, so a rehearsal host does not
// carry the Jev key or session secret. It never migrates: a restore that is
// missing a migration must fail here, not be repaired silently.
func runRestoreCheck(args []string, getenv func(string) string, stderr, stdout io.Writer) int {
	fs := flag.NewFlagSet("restore-check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "write the report to this file")
	expect := fs.String("expect", "", "compare against a report written on the source host")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	dsn := strings.TrimSpace(getenv("SITEWISE_DATABASE_URL"))
	dir := strings.TrimSpace(getenv("SITEWISE_FILE_DIR"))
	var missing []string
	if dsn == "" {
		missing = append(missing, "SITEWISE_DATABASE_URL")
	}
	if dir == "" {
		missing = append(missing, "SITEWISE_FILE_DIR")
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "missing configuration: %s\n", strings.Join(missing, ", "))
		return 1
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		fmt.Fprintln(stderr, "SITEWISE_FILE_DIR is not a directory")
		return 1
	}

	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		fmt.Fprintln(stderr, "database: connection failed")
		return 1
	}
	defer st.Close()
	blobs, err := files.Open(dir, 1)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	rep := restoreReport{CheckedAt: time.Now().UTC()}
	if rep.Facts, err = st.RestoreFacts(ctx); err != nil {
		fmt.Fprintf(stderr, "database facts: %v\n", err)
		return 1
	}
	hashes, err := st.ListContentHashes(ctx)
	if err != nil {
		fmt.Fprintf(stderr, "content hashes: %v\n", err)
		return 1
	}
	rep.Blobs = verifyBlobs(blobs, hashes)

	problems := restoreProblems(rep)
	if *expect != "" {
		want, err := readReport(*expect)
		if err != nil {
			fmt.Fprintf(stderr, "expect: %v\n", err)
			return 1
		}
		problems = append(problems, compareRestore(want, rep)...)
	}
	if *out != "" {
		body, _ := json.MarshalIndent(rep, "", "  ")
		if err := os.WriteFile(*out, append(body, '\n'), 0o600); err != nil {
			fmt.Fprintf(stderr, "out: %v\n", err)
			return 1
		}
	}
	var docs int64
	for _, o := range rep.Facts.Orgs {
		docs += o.Documents
	}
	fmt.Fprintf(stdout, "orgs=%d documents=%d blobs=%d/%d foreign_keys=%d migrations=%d\n",
		len(rep.Facts.Orgs), docs, rep.Blobs.Verified, rep.Blobs.Referenced, rep.Facts.ForeignKeys, len(rep.Facts.Migrations))
	for _, p := range problems {
		fmt.Fprintln(stderr, "FAIL "+p)
	}
	if len(problems) > 0 {
		return 1
	}
	fmt.Fprintln(stdout, "restore check passed")
	return 0
}

// verifyBlobs reads every referenced blob in full; files.Store.Open refuses
// one whose bytes do not hash to its name.
func verifyBlobs(blobs *files.Store, hashes [][]byte) blobCheck {
	sorted := slices.Clone(hashes)
	slices.SortFunc(sorted, func(a, b []byte) int { return strings.Compare(string(a), string(b)) })
	c := blobCheck{Referenced: len(sorted), Missing: []string{}, Corrupt: []string{}}
	digest := sha256.New()
	for _, sum := range sorted {
		digest.Write(sum)
		name := hex.EncodeToString(sum)
		rc, err := blobs.Open(sum)
		switch {
		case errors.Is(err, files.ErrNotFound):
			c.Missing = append(c.Missing, name)
		case err != nil:
			c.Corrupt = append(c.Corrupt, name)
		default:
			rc.Close()
			c.Verified++
		}
	}
	c.Digest = hex.EncodeToString(digest.Sum(nil))
	return c
}

// restoreProblems are failures on their own, without a source report.
func restoreProblems(r restoreReport) []string {
	var p []string
	if r.Facts.InvalidIssuedSnapshots > 0 {
		p = append(p, fmt.Sprintf("%d invalid issued snapshots", r.Facts.InvalidIssuedSnapshots))
	}
	if r.Facts.MissingIssuedBlobLinks > 0 {
		p = append(p, fmt.Sprintf("%d issued exports missing their tenant file record", r.Facts.MissingIssuedBlobLinks))
	}
	if r.Facts.Unvalidated > 0 {
		p = append(p, fmt.Sprintf("%d unvalidated foreign keys", r.Facts.Unvalidated))
	}
	for _, c := range r.Facts.CrossOrg {
		if c.Rows > 0 {
			p = append(p, fmt.Sprintf("%s: %d rows reference another org or nothing", c.Reference, c.Rows))
		}
	}
	for _, h := range r.Blobs.Missing {
		p = append(p, "blob missing "+h)
	}
	for _, h := range r.Blobs.Corrupt {
		p = append(p, "blob hash mismatch "+h)
	}
	return p
}

// compareRestore lists every way restored differs from the source report.
func compareRestore(want, got restoreReport) []string {
	var d []string
	if len(want.Facts.Tables) == 0 || len(got.Facts.Tables) == 0 {
		d = append(d, "tenant table digests missing; regenerate source and restored facts with this version")
	} else {
		have := map[string]store.RestoreTableFacts{}
		for _, t := range got.Facts.Tables {
			have[t.Table+"/"+t.OrgID] = t
		}
		for _, t := range want.Facts.Tables {
			key := t.Table + "/" + t.OrgID
			g, ok := have[key]
			if !ok || g.Rows != t.Rows || g.SHA256 != t.SHA256 {
				d = append(d, "tenant table content differs: "+key)
			}
			delete(have, key)
		}
		for key := range have {
			d = append(d, "tenant table absent from source: "+key)
		}
	}
	have := map[string]store.OrgCounts{}
	for _, o := range got.Facts.Orgs {
		have[o.OrgID] = o
	}
	for _, w := range want.Facts.Orgs {
		g, ok := have[w.OrgID]
		if !ok {
			d = append(d, "org "+w.OrgID+" missing")
			continue
		}
		delete(have, w.OrgID)
		d = append(d, countDiffs(w, g)...)
	}
	for id := range have {
		d = append(d, "org "+id+" not in source")
	}
	if want.Facts.ForeignKeys != got.Facts.ForeignKeys {
		d = append(d, fmt.Sprintf("foreign keys %d, source %d", got.Facts.ForeignKeys, want.Facts.ForeignKeys))
	}
	if !slices.Equal(want.Facts.Migrations, got.Facts.Migrations) {
		d = append(d, fmt.Sprintf("migrations %v, source %v", got.Facts.Migrations, want.Facts.Migrations))
	}
	if want.Blobs.Referenced != got.Blobs.Referenced || want.Blobs.Digest != got.Blobs.Digest {
		d = append(d, fmt.Sprintf("blob digest %s (%d), source %s (%d)", got.Blobs.Digest, got.Blobs.Referenced, want.Blobs.Digest, want.Blobs.Referenced))
	}
	return d
}

func countDiffs(w, g store.OrgCounts) []string {
	pairs := []struct {
		name      string
		want, got int64
	}{
		{"users", w.Users, g.Users},
		{"memberships", w.Memberships, g.Memberships},
		{"projects", w.Projects, g.Projects},
		{"files", w.Files, g.Files},
		{"documents", w.Documents, g.Documents},
		{"decisions", w.Decisions, g.Decisions},
		{"passages", w.Passages, g.Passages},
		{"events", w.Events, g.Events},
	}
	var d []string
	for _, p := range pairs {
		if p.want != p.got {
			d = append(d, fmt.Sprintf("org %s %s %d, source %d", w.OrgID, p.name, p.got, p.want))
		}
	}
	return d
}

func readReport(path string) (restoreReport, error) {
	var r restoreReport
	body, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	return r, json.Unmarshal(body, &r)
}
