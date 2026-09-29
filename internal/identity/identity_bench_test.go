package identity_test

import (
	"bytes"
	"context"
	"runtime"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/latency"
)

const (
	identityP50US = 80_000
	identityP90US = 250_000
	budgetSamples = 20
)

func TestIdentityPDFBudget(t *testing.T) {
	body := readFixture(t, "identity-page.pdf")
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	coldStart := time.Now()
	if err := identity.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}
	coldUS := time.Since(coldStart).Microseconds()
	runtime.GC()
	runtime.ReadMemStats(&after)
	t.Logf("warm_init_us=%d sys_delta=%d heap_alloc_delta=%d", coldUS, int64(after.Sys)-int64(before.Sys), int64(after.HeapAlloc)-int64(before.HeapAlloc))
	samples := warmSamples(t, "pdf", body, budgetSamples)
	assertIdentityBudget(t, samples)
}

func BenchmarkIdentityPDF(b *testing.B) {
	body := readFixture(b, "identity-page.pdf")
	samples := warmSamples(b, "pdf", body, budgetSamples)
	p50, p90 := identityPercentiles(b, samples)
	if p50 > identityP50US || p90 > identityP90US {
		b.Fatalf("identity extraction p50 %dus p90 %dus exceeds %d/%d", p50, p90, identityP50US, identityP90US)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(body), int64(len(body)), identity.Limits{}); err != nil {
			b.Fatal(err)
		}
	}
	// ResetTimer clears metrics reported before the timed loop.
	b.ReportMetric(float64(p50)/1000, "p50_ms")
	b.ReportMetric(float64(p90)/1000, "p90_ms")
}

func BenchmarkIdentityDOCX(b *testing.B) {
	body := readFixture(b, "docx-table.docx")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := identity.Extract(context.Background(), "docx", bytes.NewReader(body), int64(len(body)), identity.Limits{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIdentityXLSX(b *testing.B) {
	body := readFixture(b, "large-shared-strings.xlsx")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := identity.Extract(context.Background(), "xlsx", bytes.NewReader(body), int64(len(body)), identity.Limits{}); err != nil {
			b.Fatal(err)
		}
	}
}

func warmSamples(tb testing.TB, format string, body []byte, n int) []int64 {
	tb.Helper()
	// One untimed call loads the PDFium worker so the samples are warm.
	if _, err := identity.Extract(context.Background(), format, bytes.NewReader(body), int64(len(body)), identity.Limits{}); err != nil {
		tb.Fatal(err)
	}
	samples := make([]int64, n)
	for i := range samples {
		start := time.Now()
		got, err := identity.Extract(context.Background(), format, bytes.NewReader(body), int64(len(body)), identity.Limits{})
		samples[i] = time.Since(start).Microseconds()
		if err != nil {
			tb.Fatal(err)
		}
		if format == "pdf" && (!got.TextLayer || len(got.Runs) == 0) {
			tb.Fatal("budget fixture produced no identity text")
		}
	}
	return samples
}

func assertIdentityBudget(tb testing.TB, samples []int64) {
	tb.Helper()
	p50, p90 := identityPercentiles(tb, samples)
	tb.Logf("identity extraction warm p50 %dus p90 %dus over %d samples", p50, p90, len(samples))
	if p50 > identityP50US || p90 > identityP90US {
		tb.Fatalf("identity extraction p50 %dus p90 %dus exceeds %d/%d", p50, p90, identityP50US, identityP90US)
	}
}

func identityPercentiles(tb testing.TB, samples []int64) (int64, int64) {
	tb.Helper()
	p50, err := latency.Percentile(samples, 0.5)
	if err != nil {
		tb.Fatal(err)
	}
	p90, err := latency.Percentile(samples, 0.9)
	if err != nil {
		tb.Fatal(err)
	}
	return p50, p90
}
