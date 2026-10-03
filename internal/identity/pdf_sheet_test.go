package identity_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/identity"
)

func TestPDFSheetBudget(t *testing.T) {
	body := readFixture(t, "rotated-title-block.pdf")
	if err := identity.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}
	samples := make([]int64, budgetSamples)
	for i := range samples {
		var out bytes.Buffer
		start := time.Now()
		_, err := identity.PDFSheet(context.Background(), bytes.NewReader(body), int64(len(body)), 1, 200, &out)
		samples[i] = time.Since(start).Microseconds()
		if err != nil {
			t.Fatal(err)
		}
	}
	p50, p90 := identityPercentiles(t, samples)
	t.Logf("sheet copy p50 %dus p90 %dus", p50, p90)
	if p50 > 80000 || p90 > 250000 {
		t.Fatalf("sheet copy exceeds 80/250ms: %d/%d us", p50, p90)
	}
}

func TestPDFSheetPreservesOnlyRequestedPage(t *testing.T) {
	body := readFixture(t, "rotated-title-block.pdf")
	for page := 1; page <= 2; page++ {
		var out bytes.Buffer
		count, err := identity.PDFSheet(context.Background(), bytes.NewReader(body), int64(len(body)), page, 200, &out)
		if err != nil || count != 2 {
			t.Fatalf("sheet %d: count %d, error %v", page, count, err)
		}
		text, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(out.Bytes()), int64(out.Len()), identity.Limits{MaxPages: 2})
		if err != nil {
			t.Fatal(err)
		}
		_, title := findRun(text, "GROUND FLOOR")
		_, secret := findRun(text, "PAGE TWO SECRET")
		if title != (page == 1) || secret != (page == 2) {
			t.Fatalf("sheet %d contains wrong text: %+v", page, text.Runs)
		}
		for _, run := range text.Runs {
			if run.Source.Page != 1 {
				t.Fatal("output contains a second page")
			}
		}
		if page == 1 {
			number, ok := findRun(text, "A-101")
			if !ok || number.Source.Rotation != 270 {
				t.Fatalf("rotation lost: %+v", number)
			}
		}
	}
}

func TestPDFSheetBoundsAndCancellation(t *testing.T) {
	body := readFixture(t, "rotated-title-block.pdf")
	for _, tc := range []struct {
		page, max int
		want      error
	}{
		{0, 200, identity.ErrMalformed}, {3, 200, identity.ErrMalformed}, {1, 1, identity.ErrTooLarge},
	} {
		var out bytes.Buffer
		_, err := identity.PDFSheet(context.Background(), bytes.NewReader(body), int64(len(body)), tc.page, tc.max, &out)
		if !errors.Is(err, tc.want) || out.Len() != 0 {
			t.Fatalf("%+v: %v, bytes %d", tc, err, out.Len())
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out bytes.Buffer
	_, err := identity.PDFSheet(ctx, bytes.NewReader(body), int64(len(body)), 1, 200, &out)
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = identity.PDFSheet(context.Background(), bytes.NewReader([]byte("broken")), 6, 1, 200, &out)
	if !errors.Is(err, identity.ErrMalformed) {
		t.Fatal(err)
	}
}
