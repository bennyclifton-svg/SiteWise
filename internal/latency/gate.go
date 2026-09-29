package latency

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"
)

// PathBudget is a latency budget in integer microseconds.
type PathBudget struct {
	Name  string `json:"name"`
	P50US int64  `json:"p50_us"`
	P90US int64  `json:"p90_us"`
}

// Budgets is the committed set of path budgets and the sample floor.
type Budgets struct {
	MinSamples int          `json:"min_samples"`
	Paths      []PathBudget `json:"paths"`
}

// Percentile returns the nearest-rank value at p, where 0 < p <= 1.
// The rank index is ceil(p*n)-1 on a sorted copy. Empty input is an error.
func Percentile(samples []int64, p float64) (int64, error) {
	if len(samples) == 0 {
		return 0, errors.New("empty samples")
	}
	if p <= 0 || p > 1 {
		return 0, errors.New("percentile out of range")
	}
	// Copy before sorting so a caller can reuse its sample buffer.
	sorted := append([]int64(nil), samples...)
	slices.Sort(sorted)
	rank := int(math.Ceil(p*float64(len(sorted)))) - 1
	if rank < 0 || rank >= len(sorted) {
		return 0, errors.New("percentile rank out of range")
	}
	return sorted[rank], nil
}

// Gate checks every budgeted path. It returns a nonzero code when a path has
// fewer than MinSamples observations or when p50 or p90 exceeds its budget.
func Gate(budgets Budgets, samples map[string][]int64) (int, string) {
	if budgets.MinSamples < 1 {
		return 2, "min_samples must be positive"
	}
	var report strings.Builder
	code := 0
	for _, path := range budgets.Paths {
		observed := samples[path.Name]
		if len(observed) < budgets.MinSamples {
			fmt.Fprintf(&report, "%s: insufficient samples: got %d need %d\n", path.Name, len(observed), budgets.MinSamples)
			if code == 0 {
				code = 2
			}
			continue
		}
		p50, err := Percentile(observed, 0.5)
		if err != nil {
			fmt.Fprintf(&report, "%s: %v\n", path.Name, err)
			if code == 0 {
				code = 2
			}
			continue
		}
		p90, err := Percentile(observed, 0.9)
		if err != nil {
			fmt.Fprintf(&report, "%s: %v\n", path.Name, err)
			if code == 0 {
				code = 2
			}
			continue
		}
		if p50 > path.P50US {
			fmt.Fprintf(&report, "%s: p50 %dus exceeds %dus\n", path.Name, p50, path.P50US)
			code = 1
		}
		if p90 > path.P90US {
			fmt.Fprintf(&report, "%s: p90 %dus exceeds %dus\n", path.Name, p90, path.P90US)
			code = 1
		}
	}
	return code, report.String()
}

// LoadBudgets reads a budget file. Durations in the file are microseconds.
func LoadBudgets(path string) (Budgets, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Budgets{}, err
	}
	var budgets Budgets
	if err := json.Unmarshal(body, &budgets); err != nil {
		return Budgets{}, err
	}
	if budgets.MinSamples < 1 {
		return Budgets{}, errors.New("min_samples must be positive")
	}
	if len(budgets.Paths) == 0 {
		return Budgets{}, errors.New("no paths in budgets")
	}
	return budgets, nil
}

// LoadSamples reads path name to microsecond observations.
func LoadSamples(path string) (map[string][]int64, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var samples map[string][]int64
	if err := json.Unmarshal(body, &samples); err != nil {
		return nil, err
	}
	if samples == nil {
		return nil, errors.New("samples file is empty")
	}
	return samples, nil
}

// RunGate is the benchmark command. A missing sample file is a failed build,
// not a skipped check.
func RunGate(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("gate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	budgetsPath := fs.String("budgets", "bench/budgets.json", "budget file")
	samplesPath := fs.String("samples", "bench/samples.json", "sample file of microsecond observations")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	budgets, err := LoadBudgets(*budgetsPath)
	if err != nil {
		fmt.Fprintf(stderr, "budgets: %v\n", err)
		return 2
	}
	samples, err := LoadSamples(*samplesPath)
	if err != nil {
		fmt.Fprintf(stderr, "samples: %v\n", err)
		return 2
	}
	code, report := Gate(budgets, samples)
	if report != "" {
		fmt.Fprint(stderr, report)
	}
	return code
}
