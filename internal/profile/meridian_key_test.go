package profile_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"

	"sitewise/internal/profile"
)

// keyEntry is one scored value in a profile answer key
// (data/eval/profile/README.md).
type keyEntry struct {
	Accept  []string `yaml:"accept"`
	BlankOK bool     `yaml:"blank_ok"`
}

type keySystem struct {
	Presence string `yaml:"presence"`
	Scope    string `yaml:"scope"`
}

type profileKey struct {
	Header       map[string]keyEntry  `yaml:"header"`
	Details      map[string]keyEntry  `yaml:"details"`
	Determinants map[string]keyEntry  `yaml:"determinants"`
	Signals      map[string]keyEntry  `yaml:"signals"`
	Systems      map[string]keySystem `yaml:"systems"`
}

// TestMeridianAnswerKey replays Meridian's stored Jev readings (a private
// fixture, exported from the dev database) through the code-only profile
// build with the repository thresholds, and scores the result against the
// draft answer key. No Jev call. It fails on any wrong filled value and on
// a fall in the share correct below the recorded floor.
func TestMeridianAnswerKey(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "data", "eval", "profile", "private", "meridian-facts.json"))
	if err != nil {
		t.Skip("private Meridian fixture absent:", err)
	}
	var fixture struct {
		Parts []profile.Part
		Facts []profile.Fact
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	var key profileKey
	body, err := os.ReadFile(filepath.Join(root, "data", "eval", "profile", "answer-keys", "meridian.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(body, &key); err != nil {
		t.Fatal(err)
	}
	th, err := profile.LoadThresholds(filepath.Join(root, "data", "profile", "thresholds.json"))
	if err != nil {
		t.Fatal(err)
	}
	rows := profile.Build(profile.Input{Parts: fixture.Parts, Facts: fixture.Facts, Thresholds: th}, repoCatalog(t))
	whole := ""
	for _, p := range fixture.Parts {
		if p.Kind == "whole" {
			whole = p.ID
		}
	}
	got := map[string]profile.Row{}
	for _, r := range rows {
		if r.PartID == whole {
			got[r.Key] = r
		}
	}
	// filled is the value a user sees as the profile's answer: red rows and
	// blanks show none.
	filled := func(k string) string {
		r := got[k]
		if r.Band == "red" || r.Band == "blank" {
			return ""
		}
		return r.Value
	}

	correct, total := 0, 0
	var misses, wrong []string
	score := func(k string, e keyEntry) {
		total++
		v := filled(k)
		switch {
		case v == "" && (e.BlankOK || len(e.Accept) == 0):
			correct++
		case v == "":
			misses = append(misses, k+" blank (band "+got[k].Band+")")
		case slices.Contains(e.Accept, v):
			correct++
		default:
			wrong = append(wrong, k+"="+v)
		}
	}
	for _, m := range []map[string]keyEntry{key.Header, key.Details, key.Determinants, key.Signals} {
		for k, e := range m {
			score(k, e)
		}
	}
	for id, s := range key.Systems {
		if s.Presence != "" {
			score("sys."+id+".presence", keyEntry{Accept: []string{s.Presence}})
		}
		if s.Scope != "" {
			score("scope."+id, keyEntry{Accept: []string{s.Scope}})
		}
	}
	sort.Strings(misses)
	sort.Strings(wrong)
	for _, m := range misses {
		t.Log("miss:", m)
	}
	t.Logf("Meridian: %d/%d correct without user action", correct, total)
	for _, w := range wrong {
		t.Error("wrong value:", w)
	}
	// Ratchet: raise this when a fix lands; never lower it to pass.
	const floor = 38 // 2026-10-24, profile-8 readings
	if correct < floor {
		t.Errorf("correct %d below floor %d", correct, floor)
	}
}
