package identity_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"testing"

	"sitewise/internal/identity"
)

func TestCompleteExtractionRejectsRunTruncation(t *testing.T) {
	body := readFixture(t, "docx-table.docx")
	_, err := identity.Extract(context.Background(), "docx", bytes.NewReader(body), int64(len(body)), identity.Limits{RequireComplete: true, MaxRuns: 1})
	if !errors.Is(err, identity.ErrTooLarge) {
		t.Fatalf("partial document reported complete: %v", err)
	}
}

func TestCompleteExtractionRejectsUnsupportedNestedTable(t *testing.T) {
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	w, err := z.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:tbl><w:tr><w:tc><w:tbl><w:tr><w:tc><w:p><w:r><w:t>Hidden requirement</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:tc></w:tr></w:tbl></w:body></w:document>`))
	if err != nil {
		t.Fatal(err)
	}
	if err = z.Close(); err != nil {
		t.Fatal(err)
	}
	_, err = identity.Extract(context.Background(), "docx", bytes.NewReader(buf.Bytes()), int64(buf.Len()), identity.Limits{RequireComplete: true})
	if err == nil {
		t.Fatal("nested requirement silently omitted")
	}
}
