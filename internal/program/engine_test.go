package program

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fixture is one shared engine fixture (testdata/engine/README.md).
type fixture struct {
	Name string `json:"name"`
	Why  string `json:"why"`
	Input
	Expect map[string]json.RawMessage `json:"expect"`
}

func loadFixture(t testing.TB, path string) fixture {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f fixture
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return f
}

func fixturePaths(t testing.TB) []string {
	paths, err := filepath.Glob(filepath.Join("testdata", "engine", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no fixtures: %v", err)
	}
	return paths
}

// jsonValue round-trips v through JSON so expected and actual compare in the
// same types (numbers as float64, nil pointers as nil).
func jsonValue(t testing.TB, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func rawValue(t testing.TB, r json.RawMessage) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(r, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// checkFixture compares only the keys a fixture states; "why" is prose.
func checkFixture(t *testing.T, f fixture, out Output) {
	t.Helper()
	top := map[string]any{
		"finish":       out.Finish.TypicalIndex,
		"finish_low":   out.Finish.LowIndex,
		"finish_high":  out.Finish.HighIndex,
		"finish_dates": map[string]any{"typical": out.Finish.Typical, "low": out.Finish.Low, "high": out.Finish.High},
		"complete":     out.Finish.Complete,
		"status":       out.Status,
		"driving":      out.DrivingLinks,
		"cycle":        out.Cycle,
		"conflicts":    out.Conflicts,
	}
	for key, raw := range f.Expect {
		switch key {
		case "activities":
			var acts map[string]map[string]json.RawMessage
			if err := json.Unmarshal(raw, &acts); err != nil {
				t.Fatal(err)
			}
			if len(acts) == 0 && len(out.Activities) != 0 {
				t.Errorf("activities: want none computed, got %d", len(out.Activities))
			}
			for id, fields := range acts {
				got, ok := out.Activities.Map()[id]
				if !ok {
					t.Errorf("%s: missing from output", id)
					continue
				}
				gotMap := jsonValue(t, got).(map[string]any)
				for k, v := range fields {
					if k == "why" {
						continue
					}
					if want := rawValue(t, v); !reflect.DeepEqual(gotMap[k], want) {
						t.Errorf("%s.%s = %v, want %v", id, k, gotMap[k], want)
					}
				}
			}
		case "cycle_words":
			if got, want := CycleWords(f.Input, out.Cycle), rawValue(t, raw); got != want {
				t.Errorf("cycle_words = %q, want %q", got, want)
			}
		default:
			v, ok := top[key]
			if !ok {
				t.Fatalf("unknown expect key %q", key)
			}
			if got, want := jsonValue(t, v), rawValue(t, raw); !reflect.DeepEqual(got, want) {
				t.Errorf("%s = %v, want %v", key, got, want)
			}
		}
	}
}

func TestFixtures(t *testing.T) {
	for _, path := range fixturePaths(t) {
		f := loadFixture(t, path)
		t.Run(f.Name, func(t *testing.T) {
			if f.Name+".json" != filepath.Base(path) {
				t.Errorf("name %q does not match file %s", f.Name, filepath.Base(path))
			}
			if err := Validate(f.Input); err != nil {
				t.Fatalf("fixture input invalid: %v", err)
			}
			checkFixture(t, f, Compute(f.Input))
		})
	}
}

func fixtureByName(t *testing.T, name string) fixture {
	return loadFixture(t, filepath.Join("testdata", "engine", name+".json"))
}

func sameOutput(t *testing.T, got, want Output) {
	t.Helper()
	if g, w := jsonValue(t, got), jsonValue(t, want); !reflect.DeepEqual(g, w) {
		t.Errorf("outputs differ\n got  %v\n want %v", g, w)
	}
}

func TestFirstChildMoveKeepsDates(t *testing.T) {
	before, after := fixtureByName(t, "first-child-before"), fixtureByName(t, "first-child-after")
	in := before.Input
	in.Activities = append([]Activity(nil), in.Activities...)
	child := Activity{ID: "concept", ParentID: "design", Kind: KindActivity, Label: "Concept design"}
	for i, a := range in.Activities {
		if a.ID == "design" {
			var moved bool
			in.Activities[i], child, moved = FirstChildMove(a, child)
			if !moved {
				t.Fatal("design had 40 days to move")
			}
		}
	}
	if child.Duration == nil || *child.Duration != 40 {
		t.Fatalf("child took %v, want 40", child.Duration)
	}
	in.Activities = append(in.Activities, child)
	sameOutput(t, Compute(in), Compute(after.Input))

	// A stage with no duration has nothing to move.
	if _, _, moved := FirstChildMove(Activity{ID: "s", Kind: KindStage}, child); moved {
		t.Error("moved a duration that did not exist")
	}
}

func TestLastChildSpanKeepsDates(t *testing.T) {
	before, after := fixtureByName(t, "last-child-before"), fixtureByName(t, "last-child-after")
	span, ok := StageSpan(Compute(before.Input), "design")
	if !ok || span != 40 {
		t.Fatalf("span = %d, %v; want 40", span, ok)
	}
	var in Input
	in.Calendar = before.Calendar
	for _, a := range before.Activities {
		if a.ParentID == "design" {
			continue
		}
		if a.ID == "design" {
			a.Duration = &span
		}
		in.Activities = append(in.Activities, a)
	}
	for _, l := range before.Links {
		if l.ID != "cd" { // retired with its rows
			in.Links = append(in.Links, l)
		}
	}
	sameOutput(t, Compute(in), Compute(after.Input))

	// An unknown child leaves the span, and so the stage's duration, unknown.
	before.Activities[3].Duration = nil
	if _, ok := StageSpan(Compute(before.Input), "design"); ok {
		t.Error("span of a stage with an unknown child should be unknown")
	}
}

func TestHiddenStageNodesNeverReturned(t *testing.T) {
	f := fixtureByName(t, "stage-links")
	out := Compute(f.Input)
	if len(out.Activities) != len(f.Activities) {
		t.Fatalf("got %d rows for %d activities", len(out.Activities), len(f.Activities))
	}
	for id := range out.Activities.Map() {
		found := false
		for _, a := range f.Activities {
			found = found || a.ID == id
		}
		if !found {
			t.Errorf("unexpected row %q", id)
		}
	}
}

func TestValidateLink(t *testing.T) {
	f := fixtureByName(t, "stage-links")
	in := f.Input
	in.Activities = append(append([]Activity(nil), in.Activities...), Activity{ID: "G", Ordinal: 6, Kind: KindStage, Label: "Survey", Duration: ptr(1)})
	cases := []struct {
		name string
		link Link
		want string
	}{
		{"FF into a stage with children", Link{Pred: "G", Succ: "B", Type: LinkFF}, MsgLinkIntoStage},
		{"SF into a stage with children", Link{Pred: "G", Succ: "B", Type: LinkSF}, MsgLinkIntoStage},
		{"FS into a stage with children", Link{Pred: "G", Succ: "B", Type: LinkFS}, ""},
		{"FF out of a stage with children", Link{Pred: "B", Succ: "G", Type: LinkFF}, ""},
		{"child to its own stage", Link{Pred: "b1", Succ: "B", Type: LinkFS}, MsgLinkOwnStage},
		{"stage to its own child", Link{Pred: "B", Succ: "b2", Type: LinkSS}, MsgLinkOwnStage},
		{"self", Link{Pred: "C", Succ: "C"}, MsgLinkSelf},
		{"missing row", Link{Pred: "C", Succ: "nope"}, MsgLinkMissing},
		{"duplicate pair", Link{Pred: "A", Succ: "B", Type: LinkSS}, MsgLinkDuplicate},
		{"same id is an update, not a duplicate", Link{ID: "ab", Pred: "A", Succ: "B", Type: LinkSS, Lag: 1}, ""},
		{"bad type", Link{Pred: "G", Succ: "C", Type: "XX"}, MsgLinkType},
		{"lag too long", Link{Pred: "G", Succ: "C", Lag: 366}, MsgLinkLag},
		{"loop through a stage's hidden nodes", Link{Pred: "C", Succ: "A"}, "This link would make a loop: Design → Structure → Roof → Design"},
	}
	for _, c := range cases {
		err := ValidateLink(in, c.link)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	err := ValidateLink(fixtureByName(t, "leaf-stage").Input, Link{Pred: "construction", Succ: "design"})
	var le *LinkError
	if !errors.As(err, &le) || le.Message != "This link would make a loop: Design → Approval → Construction → Design" ||
		!reflect.DeepEqual(le.Cycle, []string{"design", "approval", "construction", "design"}) {
		t.Errorf("leaf-stage loop: %#v", err)
	}
}

func ptr(n int) *int { return &n }

// randomProgram builds an acyclic two-tier program: links only run from an
// earlier row to a later one in outline order, never between a stage and its
// own child, and never FF or SF into a stage with children.
func randomProgram(r *rand.Rand, stages, kidsMax, links int) Input {
	in := Input{Calendar: Calendar{WorkingWeek: []int{31, 63, 62, 15}[r.Intn(4)]}}
	if r.Intn(4) > 0 {
		in.Calendar.Start = "2027-03-01"
		for k := 0; k < 30; k++ {
			kind := []string{DayPublicHoliday, DayRDO, DayShutdown, DayWorkingOverride}[r.Intn(4)]
			in.Calendar.Days = append(in.Calendar.Days, CalendarDay{Day: dayString(mustDay("2027-03-01") + r.Intn(400) - 20), Kind: kind})
		}
	}
	var order []Activity
	hasKids := map[string]bool{}
	for s := 0; s < stages; s++ {
		st := Activity{ID: fmt.Sprintf("s%d", s), Ordinal: s, Kind: KindStage, Label: fmt.Sprintf("Stage %d", s), Duration: ptr(r.Intn(15))}
		order = append(order, st)
		for c := 0; c < r.Intn(kidsMax+1); c++ {
			a := Activity{ID: fmt.Sprintf("s%d-%d", s, c), ParentID: st.ID, Ordinal: c, Kind: KindActivity, Label: fmt.Sprintf("Activity %d.%d", s, c), Duration: ptr(r.Intn(12))}
			if r.Intn(8) == 0 {
				a.Kind, a.Duration = KindMilestone, ptr(0)
			}
			if r.Intn(4) == 0 {
				lo, hi := max(0, *a.Duration-r.Intn(3)), *a.Duration+r.Intn(4)
				a.Low, a.High = &lo, &hi
			}
			order = append(order, a)
			hasKids[st.ID] = true
		}
	}
	in.Activities = order
	seen := map[[2]int]bool{}
	for k := 0; k < links*3 && len(in.Links) < links; k++ {
		i := r.Intn(len(order))
		j := i + 1 + r.Intn(min(30, len(order)))
		if j >= len(order) || seen[[2]int{i, j}] || order[j].ParentID == order[i].ID {
			continue
		}
		typ := []string{LinkFS, LinkFS, LinkFS, LinkSS, LinkFF, LinkSF}[r.Intn(6)]
		if hasKids[order[j].ID] && (typ == LinkFF || typ == LinkSF) {
			typ = LinkFS
		}
		unit := LagWorking
		if r.Intn(5) == 0 {
			unit = LagCalendar
		}
		seen[[2]int{i, j}] = true
		in.Links = append(in.Links, Link{ID: fmt.Sprintf("l%d", len(in.Links)), Pred: order[i].ID, Succ: order[j].ID, Type: typ, Lag: r.Intn(11) - 4, LagUnit: unit})
	}
	return in
}

// With no constraints, every known row has ES ≤ LS and non-negative total
// and free float, and something is critical.
func TestPropertyFloatNonNegative(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for n := 0; n < 300; n++ {
		in := randomProgram(r, 1+r.Intn(8), 6, r.Intn(40))
		out := Compute(in)
		if out.Cycle != nil {
			t.Fatalf("case %d: acyclic program reported cycle %v", n, out.Cycle)
		}
		least := 1 << 30
		for id, tm := range out.Activities.Map() {
			if tm.Unknown {
				continue
			}
			if tm.ES > tm.LS || tm.TF < 0 || tm.FF < 0 || tm.ES < 0 {
				t.Fatalf("case %d: %s es %d ls %d tf %d ff %d\n%+v", n, id, tm.ES, tm.LS, tm.TF, tm.FF, in)
			}
			least = min(least, tm.TF)
		}
		if len(out.Activities) > 0 && least != 0 {
			t.Fatalf("case %d: no critical row (least float %d)", n, least)
		}
	}
}

// A reported cycle is a real loop: each step is a link or a stage-child tie,
// and it closes on itself.
func TestFindCycleIsReal(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for n := 0; n < 200; n++ {
		in := randomProgram(r, 2+r.Intn(6), 5, 10+r.Intn(30))
		if len(in.Links) == 0 {
			continue
		}
		// Close a loop: link a link's successor back to its predecessor.
		l := in.Links[r.Intn(len(in.Links))]
		in.Links = append(in.Links, Link{ID: "back", Pred: l.Succ, Succ: l.Pred, Type: LinkSS})
		cycle := FindCycle(in)
		if cycle == nil {
			t.Fatalf("case %d: no cycle found", n)
		}
		if len(cycle) < 3 || cycle[0] != cycle[len(cycle)-1] {
			t.Fatalf("case %d: not closed: %v", n, cycle)
		}
		parent := map[string]string{}
		for _, a := range in.Activities {
			parent[a.ID] = a.ParentID
		}
		edge := map[[2]string]bool{}
		for _, x := range in.Links {
			edge[[2]string{x.Pred, x.Succ}] = true
		}
		for k := 0; k+1 < len(cycle); k++ {
			u, v := cycle[k], cycle[k+1]
			if !edge[[2]string{u, v}] && parent[v] != u && parent[u] != v {
				t.Fatalf("case %d: %s → %s is not a link or a stage tie in %v", n, u, v, cycle)
			}
		}
		if out := Compute(in); !reflect.DeepEqual(out.Cycle, cycle) || len(out.Activities) != 0 {
			t.Fatalf("case %d: Compute disagrees with FindCycle", n)
		}
	}
}

func TestStatusWords(t *testing.T) {
	for _, c := range []struct {
		known, total int
		want         string
	}{{31, 44, "Indicative: 31 of 44 durations known"}, {0, 12, "Indicative: 0 of 12 durations known"}, {3, 3, "All 3 durations known"}, {0, 0, "No activities yet"}} {
		if got := StatusWords(c.known, c.total); got != c.want {
			t.Errorf("%d/%d: %q", c.known, c.total, got)
		}
	}
}

func TestValidateRejectsBadInput(t *testing.T) {
	bad := []Input{
		{Calendar: Calendar{Start: "1/3/2027"}},
		{Calendar: Calendar{WorkingWeek: 128}},
		{Calendar: Calendar{Days: []CalendarDay{{Day: "2027-02-30", Kind: DayRDO}}}},
		{Calendar: Calendar{Days: []CalendarDay{{Day: "2027-03-01", Kind: "picnic"}}}},
		{Activities: []Activity{{ID: "a", Duration: ptr(3651)}}},
		{Activities: []Activity{{ID: "a", Duration: ptr(5), Low: ptr(6), High: ptr(4)}}},
		{Activities: []Activity{{ID: "a", Duration: ptr(5), Low: ptr(4)}}},
		{Activities: []Activity{{ID: "a", Constraint: ConstraintMSO}}},
		{Activities: []Activity{{ID: "a", ConstraintDate: "2027-03-01"}}},
	}
	for i, in := range bad {
		if Validate(in) == nil {
			t.Errorf("case %d accepted", i)
		}
	}
}

// Compute tolerates input that Validate refuses: it never panics and ignores
// what it cannot read.
func TestComputeTolerantOfBadRows(t *testing.T) {
	in := Input{
		Calendar: Calendar{Start: "not a date", WorkingWeek: 999},
		Activities: []Activity{
			{ID: "a", Duration: ptr(2)},
			{ID: "a", Duration: ptr(9)}, // duplicate id: the first wins
			{ID: "b", ParentID: "ghost", Duration: ptr(3)},
			{ID: "c", ParentID: "b", Duration: ptr(1)}, // a third tier is scheduled as top level
		},
		Links: []Link{{ID: "x", Pred: "a", Succ: "zz"}, {ID: "y", Pred: "a", Succ: "b", Type: "QQ"}},
	}
	out := Compute(in)
	if out.Finish.TypicalIndex == nil || *out.Finish.TypicalIndex != 9 || out.Finish.Typical != nil {
		t.Errorf("finish = %v (date %v), want 9 relative", out.Finish.TypicalIndex, out.Finish.Typical)
	}
	if !strings.HasPrefix(out.Status.Words, "All") {
		t.Errorf("status %q", out.Status.Words)
	}
}

func loadPerf(b testing.TB) Input {
	return loadFixture(b, filepath.Join("testdata", "engine", "perf-2000.json")).Input
}

func BenchmarkCompute2000(b *testing.B) {
	in := loadPerf(b)
	if len(in.Activities) != 2000 || len(in.Links) != 4000 {
		b.Fatalf("perf fixture has %d activities and %d links", len(in.Activities), len(in.Links))
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Compute(in)
	}
}

// The plan's budget: Compute on 2,000 activities and 4,000 links under 2 ms.
func TestCompute2000Budget(t *testing.T) {
	if testing.Short() {
		t.Skip("timing test")
	}
	// Best of three, so another process on the machine does not fail the
	// build; a real regression is slow every time.
	best := int64(math.MaxInt64)
	for k := 0; k < 3; k++ {
		best = min(best, testing.Benchmark(BenchmarkCompute2000).NsPerOp())
	}
	if ms := float64(best) / 1e6; ms >= 2 {
		t.Errorf("Compute 2000 takes %.2f ms/op, budget 2 ms", ms)
	}
}
