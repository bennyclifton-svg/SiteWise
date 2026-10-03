package identity_test

import (
	"bytes"
	"context"
	"errors"
	"sitewise/internal/identity"
	"testing"
)

func TestDensePDFLimitKeepsTrailingSheetNumber(t *testing.T) {
	text := extractFixture(t, "identity-page.pdf", "pdf", identity.Limits{MaxRuns: 10})
	if len(text.Runs) > 10 {
		t.Fatalf("limit exceeded: %d", len(text.Runs))
	}
	if _, ok := findRun(text, "A-101"); !ok {
		t.Fatal("run limit discarded the trailing title block")
	}
	if _, ok := findRun(text, "Fire note 01 hydrant booster pump room access"); !ok {
		t.Fatal("lost first-page heading region")
	}
}

func TestLargePDFReadsOnlyIdentityRanges(t *testing.T) {
	body := readFixture(t, "identity-page.pdf")
	tail := body[bytes.LastIndex(body, []byte("startxref")):]
	padded := append(append(append([]byte{}, body...), bytes.Repeat([]byte(" "), 2<<20)...), tail...)
	got, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(padded), int64(len(padded)), identity.Limits{MaxBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := findRun(got, "A-101"); !ok {
		t.Fatal("range reader lost identity text")
	}
	_, err = identity.Extract(context.Background(), "pdf", bytes.NewReader(padded), int64(len(padded)), identity.Limits{MaxBytes: 32})
	if !errors.Is(err, identity.ErrTooLarge) {
		t.Fatalf("read budget not enforced: %v", err)
	}
}
