package latency

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestPercentileNearestRank(t *testing.T) {
	samples := []int64{50, 10, 40, 20, 30}
	original := append([]int64(nil), samples...)

	cases := []struct {
		p    float64
		want int64
	}{
		{p: 0.5, want: 30}, // ceil(2.5)-1 = 2
		{p: 0.9, want: 50}, // ceil(4.5)-1 = 4
		{p: 1, want: 50},   // ceil(5)-1 = 4
	}
	for _, tc := range cases {
		got, err := Percentile(samples, tc.p)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("p=%v got %d want %d", tc.p, got, tc.want)
		}
	}
	for i := range samples {
		if samples[i] != original[i] {
			t.Fatal("percentile sorted the caller slice")
		}
	}
}

func TestPercentileEmpty(t *testing.T) {
	if _, err := Percentile(nil, 0.9); err == nil {
		t.Fatal("expected error")
	}
	if _, err := Percentile([]int64{}, 0.5); err == nil {
		t.Fatal("expected error")
	}
}

func TestGateGoodMedianFailingP90(t *testing.T) {
	budgets := Budgets{
		MinSamples: 20,
		Paths: []PathBudget{{
			Name:  "whole_intake",
			P50US: 1_000,
			P90US: 2_000,
		}},
	}
	// 17 fast samples and 3 slow ones. p50 index is 9 (fast); p90 index is 17 (slow).
	samples := map[string][]int64{
		"whole_intake": append(repeat(100, 17), 50_000, 50_000, 50_000),
	}
	code, report := Gate(budgets, samples)
	if code == 0 {
		t.Fatalf("expected nonzero exit, report: %s", report)
	}
	if !bytes.Contains([]byte(report), []byte("p90")) {
		t.Fatalf("report should name the failing percentile: %s", report)
	}
}

func TestGateInsufficientSamples(t *testing.T) {
	budgets := Budgets{
		MinSamples: 20,
		Paths: []PathBudget{{
			Name:  "whole_intake",
			P50US: 1_000_000,
			P90US: 2_000_000,
		}},
	}
	code, _ := Gate(budgets, map[string][]int64{
		"whole_intake": repeat(100, 19),
	})
	if code == 0 {
		t.Fatal("expected nonzero exit for insufficient samples")
	}
}

func TestGatePass(t *testing.T) {
	budgets := Budgets{
		MinSamples: 20,
		Paths: []PathBudget{{
			Name:  "whole_intake",
			P50US: 1_000,
			P90US: 2_000,
		}},
	}
	code, report := Gate(budgets, map[string][]int64{
		"whole_intake": repeat(100, 20),
	})
	if code != 0 {
		t.Fatalf("exit %d: %s", code, report)
	}
}

func TestCommittedBudgets(t *testing.T) {
	path := filepath.Join("..", "..", "bench", "budgets.json")
	budgets, err := LoadBudgets(path)
	if err != nil {
		t.Fatal(err)
	}
	if budgets.MinSamples < 20 {
		t.Fatalf("min_samples = %d", budgets.MinSamples)
	}
	want := map[string][2]int64{
		"identity_text_extraction":  {80_000, 250_000},
		"candidate_harvesting":      {5_000, 10_000},
		"deterministic_field_rules": {1_000, 1_000},
		"jev_admission_request":     {350_000, 800_000},
		"commit_sse_enqueue":        {5_000, 15_000},
		"whole_intake":              {1_000_000, 2_000_000},
		"project_document_list":     {50_000, 150_000},
		"field_correction":          {50_000, 150_000},
		"project_invite_auth":       {100_000, 250_000},
		"health_speed":              {25_000, 100_000},
		"sse_reconnect":             {100_000, 250_000},
		"project_profile_read":      {50_000, 150_000},
		"profile_edit":              {50_000, 150_000},
		"document_delete":           {500_000, 1_000_000},
	}
	if len(budgets.Paths) != len(want) {
		t.Fatalf("paths = %d", len(budgets.Paths))
	}
	for _, pathBudget := range budgets.Paths {
		limits, ok := want[pathBudget.Name]
		if !ok {
			t.Fatalf("unexpected path %s", pathBudget.Name)
		}
		if pathBudget.P50US != limits[0] || pathBudget.P90US != limits[1] {
			t.Fatalf("%s = %d/%d", pathBudget.Name, pathBudget.P50US, pathBudget.P90US)
		}
	}
}

func TestRunGateMissingSamples(t *testing.T) {
	var stderr bytes.Buffer
	code := RunGate([]string{"-budgets", filepath.Join("..", "..", "bench", "budgets.json"), "-samples", filepath.Join(t.TempDir(), "missing.json")}, &stderr)
	if code == 0 {
		t.Fatal("missing samples must not exit 0")
	}
}

func repeat(v int64, n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = v
	}
	return out
}
