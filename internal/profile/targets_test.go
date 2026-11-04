package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

// Walsh brief, letter and builder email, 2026-10-27: targets and the
// builder's estimate are offered as candidates and read in Go.
func TestOwnerTargetsAndEstimateCandidates(t *testing.T) {
	cat := repoCatalog(t)
	for _, tc := range []struct {
		text, key string
		want      []string
	}{
		{"DA lodged by mid-2026; construction start early 2027", "proj.target_da_lodgement", []string{"mid-2026", "early 2027"}},
		{"DA lodged by mid-2026; construction start early 2027", "proj.target_construction_start", []string{"mid-2026", "early 2027"}},
		{"Target DA lodgement: **July 2026**, subject to heritage advisor input", "proj.target_da_lodgement", []string{"July 2026"}},
		{"Maximum **18 months** on site acceptable", "proj.max_construction_months", []string{"18 months"}},
		{"**ROM construction range:** $880,000 – $980,000 ex GST for scope as described", "proj.preliminary_estimate", []string{"$880,000 – $980,000"}},
		{"**Programme ROM:** 14–16 months on site if DA determined by October 2026.", "proj.preliminary_construction_months", []string{"14–16 months"}},
	} {
		h := profile.Harvest(tc.text, cat)
		got := map[string]bool{}
		for _, c := range h.Candidates[tc.key] {
			got[c.Value] = true
		}
		for _, w := range tc.want {
			if !got[w] {
				t.Errorf("%q: %s missing %q; got %v", tc.text, tc.key, w, got)
			}
		}
	}
}

func TestTargetPhrasesReadInGo(t *testing.T) {
	for phrase, want := range map[string]string{"July 2026": "2026-07-31", "mid-2026": "2026-08-31", "early 2027": "2027-04-30", "late 2026": "2026-12-31", "Q3 2026": "2026-09-30", "3 March 2026": "2026-03-03"} {
		if got, ok := profile.LatestDay(phrase); !ok || got != want {
			t.Errorf("%q: %q %v, want %q", phrase, got, ok, want)
		}
	}
	if _, ok := profile.LatestDay("soon"); ok {
		t.Fatal("read a date from no date")
	}
	if lo, hi, ok := profile.MonthRange("14–16 months"); !ok || lo != 14 || hi != 16 {
		t.Fatalf("range %d %d %v", lo, hi, ok)
	}
	if lo, hi, ok := profile.MonthRange("18 months"); !ok || lo != 18 || hi != 18 {
		t.Fatalf("single %d %d %v", lo, hi, ok)
	}
}
