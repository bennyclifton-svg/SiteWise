package program

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Golden outputs are the Go engine's whole `computed` object for every shared
// fixture plus a set of seeded random programs. The fixture `expect` blocks
// state only the values worked by hand; the goldens hold every field of every
// row, so the TypeScript mirror (web/src/program/engine.test.ts) can compare
// its output wholesale. Regenerate after a deliberate engine change with
//
//	go test ./internal/program -run TestGolden -update
var updateGolden = flag.Bool("update", false, "rewrite testdata/engine/golden from the Go engine")

var goldenDir = filepath.Join("testdata", "engine", "golden")

// goldenCase is one golden file entry. Input is set for random cases only;
// a fixture's input is its own file.
type goldenCase struct {
	Name       string `json:"name"`
	Input      *Input `json:"input,omitempty"`
	Computed   Output `json:"computed"`
	CycleWords string `json:"cycle_words"`
}

func newGolden(name string, in Input, withInput bool) goldenCase {
	out := Compute(in)
	c := goldenCase{Name: name, Computed: out, CycleWords: CycleWords(in, out.Cycle)}
	if withInput {
		c.Input = &in
	}
	return c
}

// goldenRandom builds programs the hand-worked fixtures do not reach: every
// working week the property tests use, holidays and overrides (some on the
// same day), calendar lags of every type, ranges, unknown durations,
// start-no-earlier and mandatory starts, and some loops.
func goldenRandom() []goldenCase {
	r := rand.New(rand.NewSource(2027))
	var cases []goldenCase
	for n := 0; n < 60; n++ {
		in := randomProgram(r, 1+r.Intn(8), 6, r.Intn(40))
		start, dated := mustDay("2027-03-01"), in.Calendar.Start != ""
		for i := range in.Activities {
			a := &in.Activities[i]
			switch r.Intn(12) {
			case 0:
				a.Duration, a.Low, a.High = nil, nil, nil
			case 1:
				if dated {
					a.Constraint, a.ConstraintDate = ConstraintSNET, dayString(start+r.Intn(80))
				}
			case 2:
				if dated {
					a.Constraint, a.ConstraintDate = ConstraintMSO, dayString(start+r.Intn(80)-5)
				}
			}
		}
		if n%10 == 9 && len(in.Links) > 0 {
			l := in.Links[r.Intn(len(in.Links))]
			in.Links = append(in.Links, Link{ID: "back", Pred: l.Succ, Succ: l.Pred, Type: LinkSS})
		}
		cases = append(cases, newGolden(fmt.Sprintf("random-%02d", n), in, true))
	}
	return cases
}

func writeGolden(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// sameJSON compares two values as decoded JSON, so line endings and key
// order in the stored file never matter.
func sameJSON(t *testing.T, label string, got any, wantFile []byte, wantPath string) {
	t.Helper()
	var want any
	if err := json.Unmarshal(wantFile, &want); err != nil {
		t.Fatalf("%s: %v", wantPath, err)
	}
	if g := jsonValue(t, got); !reflect.DeepEqual(g, want) {
		t.Errorf("%s differs from %s; run go test ./internal/program -run TestGolden -update after a deliberate engine change", label, wantPath)
	}
}

func TestGolden(t *testing.T) {
	names := map[string]bool{}
	for _, path := range fixturePaths(t) {
		f := loadFixture(t, path)
		names[f.Name+".json"] = true
		gp := filepath.Join(goldenDir, f.Name+".json")
		c := newGolden(f.Name, f.Input, false)
		if *updateGolden {
			writeGolden(t, gp, c)
			continue
		}
		b, err := os.ReadFile(gp)
		if err != nil {
			t.Errorf("%s: %v (run with -update)", f.Name, err)
			continue
		}
		sameJSON(t, f.Name, c, b, gp)
	}

	rp := filepath.Join(goldenDir, "random.json")
	if *updateGolden {
		writeGolden(t, rp, goldenRandom())
		return
	}
	b, err := os.ReadFile(rp)
	if err != nil {
		t.Fatalf("%v (run with -update)", err)
	}
	// Verify against the stored inputs, so a change in math/rand's stream
	// cannot fail the check; only a change in the engine can.
	var stored []struct {
		Name  string `json:"name"`
		Input Input  `json:"input"`
	}
	if err := json.Unmarshal(b, &stored); err != nil {
		t.Fatal(err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	if len(stored) == 0 {
		t.Fatal("random goldens are empty")
	}
	for k, s := range stored {
		sameJSON(t, s.Name, newGolden(s.Name, s.Input, true), raw[k], rp)
	}

	// No stale golden for a fixture that no longer exists.
	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if n := e.Name(); n != "random.json" && strings.HasSuffix(n, ".json") && !names[n] {
			t.Errorf("golden %s has no fixture", n)
		}
	}
}
