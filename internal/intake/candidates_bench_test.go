package intake_test

import (
	"testing"

	"sitewise/internal/intake"
	"sitewise/internal/latency"
)

func BenchmarkCandidateHarvest(b *testing.B) {
	name, text := denseIdentity()
	p50, p90 := benchSamples(b, func() { intake.Harvest(name, text) })
	if p50 > 5_000 || p90 > 10_000 {
		b.Fatalf("candidate harvesting p50 %dus p90 %dus exceeds 5000/10000", p50, p90)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		intake.Harvest(name, text)
	}
	b.ReportMetric(float64(p50)/1000, "p50_ms")
	b.ReportMetric(float64(p90)/1000, "p90_ms")
}

func BenchmarkRuleDecide(b *testing.B) {
	name, text := denseIdentity()
	candidates := intake.Harvest(name, text)
	p50, p90 := benchSamples(b, func() { intake.Decide(candidates) })
	if p50 > 1_000 || p90 > 1_000 {
		b.Fatalf("deterministic field rules p50 %dus p90 %dus exceeds 1000/1000", p50, p90)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		intake.Decide(candidates)
	}
	b.ReportMetric(float64(p50)/1000, "p50_ms")
	b.ReportMetric(float64(p90)/1000, "p90_ms")
}

func benchSamples(b *testing.B, fn func()) (int64, int64) {
	b.Helper()
	samples := make([]int64, 20)
	for i := range samples {
		samples[i] = timedCall(fn)
	}
	p50, err := latency.Percentile(samples, 0.5)
	if err != nil {
		b.Fatal(err)
	}
	p90, err := latency.Percentile(samples, 0.9)
	if err != nil {
		b.Fatal(err)
	}
	return p50, p90
}
