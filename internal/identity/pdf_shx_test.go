package identity_test

import (
	"bytes"
	"context"
	"sitewise/internal/identity"
	"strings"
	"testing"
)

func TestSHXAnnotationsAndSheetCopy(t *testing.T) {
	body := readFixture(t, "shx-annotations.pdf")
	for _, copySheet := range []bool{false, true} {
		if copySheet {
			var out bytes.Buffer
			if _, err := identity.PDFSheet(context.Background(), bytes.NewReader(body), int64(len(body)), 1, 200, &out); err != nil {
				t.Fatal(err)
			}
			body = out.Bytes()
		}
		got, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(body), int64(len(body)), identity.DefaultLimits())
		if err != nil {
			t.Fatal(err)
		}
		if !got.TextLayer || len(got.Runs) != 2 {
			t.Fatalf("annotation text missing or ordinary comment admitted: %+v", got)
		}
		r := got.Runs[0]
		if r.Text != "M01" || r.Source.X != 100 || r.Source.Width != 30 || r.Source.Annotation != "AutoCAD SHX Text" {
			t.Fatalf("reversed SHX rectangle/provenance: %+v", r)
		}
	}
}

func TestPrintedStampAppearanceText(t *testing.T) {
	body := readFixture(t, "printed-stamp.pdf")
	got, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(body), int64(len(body)), identity.DefaultLimits())
	if err != nil {
		t.Fatal(err)
	}
	date := false
	for _, r := range got.Runs {
		if strings.Contains(r.Text, "NONPRINTED") {
			t.Fatal("annotation comment read instead of printed appearance")
		}
		if strings.Contains(r.Text, "10/05/23") {
			date = true
			if r.Source.Y < 600 || r.Source.Width <= 0 {
				t.Fatalf("stamp geometry lost: %+v", r)
			}
		}
	}
	if !date {
		t.Fatalf("printed date missing: %+v", got)
	}
}

func TestSHXDoesNotDisplaceNativeTextAtRunLimit(t *testing.T) {
	body := readFixture(t, "shx-annotations.pdf")
	var out bytes.Buffer
	if _, err := identity.PDFSheet(context.Background(), bytes.NewReader(body), int64(len(body)), 2, 200, &out); err != nil {
		t.Fatal(err)
	}
	got, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(out.Bytes()), int64(out.Len()), identity.Limits{MaxRuns: 1, MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Runs) != 1 || got.Runs[0].Text != "NATIVE TITLE" {
		t.Fatalf("native identity displaced: %+v", got)
	}
}
