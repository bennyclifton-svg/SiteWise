package identity_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"sitewise/internal/identity"
)

func TestManifestHashes(t *testing.T) {
	var manifest struct {
		Fixtures []struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
			Bytes  int    `json:"bytes"`
		} `json:"fixtures"`
	}
	raw, err := os.ReadFile(filepath.Join(fixtureDir(t), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Fixtures) == 0 {
		t.Fatal("manifest has no fixtures")
	}
	for _, fx := range manifest.Fixtures {
		body := readFixture(t, fx.File)
		if len(body) != fx.Bytes {
			t.Fatalf("%s: bytes %d want %d", fx.File, len(body), fx.Bytes)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != fx.SHA256 {
			t.Fatalf("%s: hash changed", fx.File)
		}
	}
}

func TestRotatedTitleBlockProvenance(t *testing.T) {
	got := extractFixture(t, "rotated-title-block.pdf", "pdf", identity.Limits{})
	if !got.TextLayer {
		t.Fatal("text layer missing")
	}
	floor, ok := findRun(got, "GROUND FLOOR")
	if !ok || floor.Source.Page != 1 || floor.Source.Rotation != 0 {
		t.Fatalf("upright text: %+v", got.Runs)
	}
	// Text matrix 0 1 -1 0 is a quarter turn. PDFium's character angle for
	// that matrix is 270 degrees counterclockwise.
	number, ok := findRun(got, "A-101")
	if !ok || number.Source.Page != 1 || number.Source.Rotation != 270 {
		t.Fatalf("rotated title block: %+v", got.Runs)
	}
	if _, ok := findRun(got, "PAGE TWO SECRET"); ok {
		t.Fatal("identity extraction read past the first page")
	}
}

func TestSecondPageIncludedWhenLimitAllows(t *testing.T) {
	got := extractFixture(t, "rotated-title-block.pdf", "pdf", identity.Limits{MaxPages: 2})
	if _, ok := findRun(got, "PAGE TWO SECRET"); !ok {
		t.Fatalf("page limit was ignored: %+v", got.Runs)
	}
}

func TestScannedEmptyPDFHasNoTextLayer(t *testing.T) {
	got := extractFixture(t, "scanned-empty.pdf", "pdf", identity.Limits{})
	if got.TextLayer || len(got.Runs) != 0 {
		t.Fatalf("scanned page produced text: %+v", got)
	}
}

func TestMalformedZIP(t *testing.T) {
	body := readFixture(t, "malformed.docx")
	_, err := identity.Extract(context.Background(), "docx", bytes.NewReader(body), int64(len(body)), identity.Limits{})
	if !errors.Is(err, identity.ErrMalformed) {
		t.Fatalf("got %v", err)
	}
}

func TestLargeSharedStringsStayInIdentityRegion(t *testing.T) {
	got := extractFixture(t, "large-shared-strings.xlsx", "xlsx", identity.Limits{})
	project, ok := findRun(got, "Project Hale")
	if !ok || project.Source.Sheet != "Register" || project.Source.Cell != "A1" || project.Source.Cached {
		t.Fatalf("shared string: %+v", got.Runs)
	}
	if _, ok := findRun(got, "SHARED-STRING-NOT-IN-IDENTITY"); ok {
		t.Fatal("column outside the identity region was returned")
	}
	if stringsCount(got, "PAD-") > 0 {
		t.Fatal("unused shared strings were returned as identity text")
	}
	if len(got.Runs) > 8 {
		t.Fatalf("identity region too wide: %d runs", len(got.Runs))
	}
}

func TestMergedCellsAndCachedFormula(t *testing.T) {
	got := extractFixture(t, "merged-cells.xlsx", "xlsx", identity.Limits{})
	title, ok := findRun(got, "Hale House")
	if !ok || title.Source.Sheet != "Cover" || title.Source.Cell != "A1" || title.Source.Merge != "A1:C1" || title.Source.Cached {
		t.Fatalf("merge: %+v", got.Runs)
	}
	cached, ok := findRun(got, "99")
	if !ok || cached.Source.Cell != "A2" || !cached.Source.Cached {
		t.Fatalf("cached formula value: %+v", got.Runs)
	}
	for _, banned := range []string{"1+2", "3", "ROW-FAR", "macro-payload-do-not-execute"} {
		if _, ok := findRun(got, banned); ok {
			t.Fatalf("extracted %q from %+v", banned, got.Runs)
		}
	}
}

func TestDOCXTableAndHeading(t *testing.T) {
	got := extractFixture(t, "docx-table.docx", "docx", identity.Limits{})
	heading, ok := findRun(got, "Fire Services Specification")
	if !ok || !heading.Source.Heading || heading.Source.Table != 0 {
		t.Fatalf("heading: %+v", got.Runs)
	}
	body, ok := findRun(got, "Issued for tender")
	if !ok || body.Source.Heading || body.Source.Table != 0 {
		t.Fatalf("body: %+v", got.Runs)
	}
	drawing, ok := findRun(got, "Drawing")
	if !ok || drawing.Source.Table != 1 || drawing.Source.Row != 1 || drawing.Source.Col != 1 {
		t.Fatalf("table label: %+v", got.Runs)
	}
	number, ok := findRun(got, "A-101")
	if !ok || number.Source.Table != 1 || number.Source.Row != 1 || number.Source.Col != 2 {
		t.Fatalf("table value: %+v", got.Runs)
	}
	revision, ok := findRun(got, "C")
	if !ok || revision.Source.Table != 1 || revision.Source.Row != 2 || revision.Source.Col != 2 {
		t.Fatalf("second row: %+v", got.Runs)
	}
}

func TestDecompressedLimit(t *testing.T) {
	body := readFixture(t, "padded-document.docx")
	_, err := identity.Extract(context.Background(), "docx", bytes.NewReader(body), int64(len(body)), identity.Limits{MaxBytes: 1024})
	if !errors.Is(err, identity.ErrTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestZipSlipRejected(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("../secret.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("<w:document/>")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = identity.Extract(context.Background(), "docx", bytes.NewReader(buf.Bytes()), int64(buf.Len()), identity.Limits{})
	if !errors.Is(err, identity.ErrMalformed) {
		t.Fatalf("got %v", err)
	}
}

func TestCanceledBeforeRead(t *testing.T) {
	body := readFixture(t, "docx-table.docx")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := identity.Extract(ctx, "docx", bytes.NewReader(body), int64(len(body)), identity.Limits{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func extractFixture(t *testing.T, name, format string, limits identity.Limits) identity.Text {
	t.Helper()
	body := readFixture(t, name)
	got, err := identity.Extract(context.Background(), format, bytes.NewReader(body), int64(len(body)), limits)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != format {
		t.Fatalf("format %q", got.Format)
	}
	return got
}

func findRun(text identity.Text, want string) (identity.Run, bool) {
	for _, run := range text.Runs {
		if run.Text == want {
			return run, true
		}
	}
	return identity.Run{}, false
}

func stringsCount(text identity.Text, prefix string) int {
	n := 0
	for _, run := range text.Runs {
		if len(run.Text) >= len(prefix) && run.Text[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}

func readFixture(t testing.TB, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(fixtureDir(t), name))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func fixtureDir(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("fixture path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "identity")
}
