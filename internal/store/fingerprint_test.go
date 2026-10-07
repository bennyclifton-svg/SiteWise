package store

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"sitewise/internal/profile"
	"sitewise/internal/works"
)

func TestFingerprintPreservesCanonicalHash(t *testing.T) {
	snap := ProfileSnapshot{
		Facts:     []profile.Fact{{ID: "z", Excerpt: "Evidence <&>\n"}, {ID: "a", Value: "12.500"}},
		Documents: []json.RawMessage{json.RawMessage(`{ "read": "auto" }`)},
		WorkItems: []json.RawMessage{json.RawMessage(`{"action":"retain"}`)},
	}
	b := ProfileBuild{ReadKinds: []string{"specification"}, KnowledgeVersion: "k", QuestionVersion: "q", ThresholdsVersion: "t"}
	var canonical []any
	for _, component := range []any{snap.Parts, snap.Facts, snap.User, snap.Planning, snap.Documents, snap.WorkItems, b.ReadKinds} {
		rows, err := sortedFingerprintRows(component)
		if err != nil {
			t.Fatal(err)
		}
		canonical = append(canonical, rows)
	}
	canonical = append(canonical, snap.Site, b.ThresholdsVersion, b.QuestionVersion, b.KnowledgeVersion)
	raw, err := json.Marshal(canonical)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("%x", sha256.Sum256(raw))
	got, err := profileFingerprint(snap, b)
	if err != nil || got != want {
		t.Fatalf("canonical hash changed: %s want %s: %v", got, want, err)
	}
	snap.WorkItems = []json.RawMessage{json.RawMessage(`invalid`)}
	if _, err := profileFingerprint(snap, b); err == nil {
		t.Fatal("invalid work input accepted")
	}
}

func TestFingerprintInputClasses(t *testing.T) {
	base := func() ProfileSnapshot {
		value := "10"
		return ProfileSnapshot{
			Parts:     []profile.Part{{ID: "part", Label: "Building"}},
			Facts:     []profile.Fact{{ID: "fact", QuestionID: "det.height", Value: "10"}},
			User:      []profile.UserValue{{PartID: "part", Key: "height", Value: &value}},
			Planning:  []profile.PlanningValue{{PartID: "part", Key: "area", Value: &value}},
			Documents: []json.RawMessage{json.RawMessage(`{"id":"doc","read":"auto"}`)},
			WorkItems: []json.RawMessage{json.RawMessage(`{"id":"work","action":"retain"}`)},
		}
	}
	build := ProfileBuild{KnowledgeVersion: "k1", QuestionVersion: "q1", ThresholdsVersion: "t1", ReadKinds: []string{"specification"}}
	want, err := profileFingerprint(base(), build)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*ProfileSnapshot, *ProfileBuild){
		"fact identity": func(s *ProfileSnapshot, b *ProfileBuild) { s.Facts[0].ID = "new" },
		"evidence":      func(s *ProfileSnapshot, b *ProfileBuild) { s.Facts[0].Value = "11" },
		"user":          func(s *ProfileSnapshot, b *ProfileBuild) { s.User[0].Origin = "assumption" },
		"planning":      func(s *ProfileSnapshot, b *ProfileBuild) { s.Planning[0].Meaning = "allowance" },
		"reading": func(s *ProfileSnapshot, b *ProfileBuild) {
			s.Documents[0] = json.RawMessage(`{"id":"doc","read":"skip"}`)
		},
		"work set": func(s *ProfileSnapshot, b *ProfileBuild) {
			s.WorkItems[0] = json.RawMessage(`{"id":"work","action":"alter"}`)
		},
		"part":       func(s *ProfileSnapshot, b *ProfileBuild) { s.Parts[0].NCCClass = "2" },
		"site":       func(s *ProfileSnapshot, b *ProfileBuild) { s.Site.Address = "changed" },
		"knowledge":  func(s *ProfileSnapshot, b *ProfileBuild) { b.KnowledgeVersion = "k2" },
		"questions":  func(s *ProfileSnapshot, b *ProfileBuild) { b.QuestionVersion = "q2" },
		"thresholds": func(s *ProfileSnapshot, b *ProfileBuild) { b.ThresholdsVersion = "t2" },
		"policy":     func(s *ProfileSnapshot, b *ProfileBuild) { b.ReadKinds = []string{"drawing"} },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			s, b := base(), build
			change(&s, &b)
			got, err := profileFingerprint(s, b)
			if err != nil || got == want {
				t.Fatalf("input change lost: %v", err)
			}
		})
	}
	s := base()
	s.Facts = append(s.Facts, profile.Fact{ID: "other", Value: "20"})
	first, _ := profileFingerprint(s, build)
	s.Facts[0], s.Facts[1] = s.Facts[1], s.Facts[0]
	second, _ := profileFingerprint(s, build)
	if first != second {
		t.Fatal("ordering changed fingerprint")
	}
	same, _ := profileFingerprint(base(), build)
	if same != want {
		t.Fatal("unchanged inputs changed fingerprint")
	}
	unchanged := base()
	unchanged.Site.Version = 9
	unchanged.User[0].Version = 9
	unchanged.Planning[0].Version = 9
	if got, err := profileFingerprint(unchanged, build); err != nil || got != want {
		t.Fatal("save counters changed semantic fingerprint")
	}
}

func TestWorkFingerprintIgnoresEditTimeButKeepsScopeAndActor(t *testing.T) {
	first, later := time.Unix(1, 0), time.Unix(2, 0)
	item := works.Item{ID: "work", Action: "repair", Version: 1, Provenance: works.Provenance{LastEditedBy: "owner", LastEditedAt: &first}}
	before := workFingerprint([]works.Item{item})
	item.Version = 2
	item.Provenance.LastEditedAt = &later
	if !reflect.DeepEqual(before, workFingerprint([]works.Item{item})) {
		t.Fatal("audit time changed semantic fingerprint")
	}
	if item.Provenance.LastEditedAt != &later {
		t.Fatal("normalization mutated caller")
	}
	item.Action = "replace"
	if reflect.DeepEqual(before, workFingerprint([]works.Item{item})) {
		t.Fatal("action change lost")
	}
	item.Action = "repair"
	item.Provenance.LastEditedBy = "other"
	if reflect.DeepEqual(before, workFingerprint([]works.Item{item})) {
		t.Fatal("authorship change lost")
	}
}

func TestFingerprintRowsPreservePreviousCanonicalBytes(t *testing.T) {
	for _, input := range []any{
		[]profile.Fact(nil), []string{}, []string{"z", "<quoted>\n\u2028", "a"},
		[]json.RawMessage{json.RawMessage(`{ "value" : 12.500, "text": "<evidence>" }`), json.RawMessage(`null`)},
		[]profile.Fact{{ID: "b", Excerpt: "Long evidence <&>\n"}, {ID: "a", Value: "12.5"}},
	} {
		// The previous algorithm is the compatibility oracle, including raw JSON
		// compaction, HTML escaping, null slices and lexicographic row ordering.
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil {
			t.Fatal(err)
		}
		want := make([]string, len(rows))
		for i, row := range rows {
			want[i] = string(row)
		}
		sort.Strings(want)
		got, err := sortedFingerprintRows(input)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("canonical bytes changed: %v %v %v", got, want, err)
		}
	}
	if _, err := sortedFingerprintRows([]json.RawMessage{json.RawMessage(`invalid`)}); err == nil {
		t.Fatal("invalid evidence accepted")
	}
}

func BenchmarkFingerprintRows(b *testing.B) {
	facts := make([]profile.Fact, 1153)
	for i := range facts {
		facts[i] = profile.Fact{ID: "fact", Excerpt: strings.Repeat("Evidence from the saved source. ", 40)}
	}
	b.Run("whole_slice_round_trip", func(b *testing.B) {
		for b.Loop() {
			raw, _ := json.Marshal(facts)
			var rows []json.RawMessage
			if err := json.Unmarshal(raw, &rows); err != nil {
				b.Fatal(err)
			}
			out := make([]string, len(rows))
			for i, row := range rows {
				out[i] = string(row)
			}
			sort.Strings(out)
		}
	})
	b.Run("direct_rows", func(b *testing.B) {
		for b.Loop() {
			if _, err := sortedFingerprintRows(facts); err != nil {
				b.Fatal(err)
			}
		}
	})
}
