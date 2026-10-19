package programread

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/jev"
)

// replay answers with a recorded System One response through the real
// client, so the response is validated exactly as a live one would be
// (plan section 2, Jev rule 15; pattern internal/jev/testdata).
type replay struct {
	body []byte
	sent []byte
}

func (r *replay) RoundTrip(req *http.Request) (*http.Response, error) {
	r.sent, _ = io.ReadAll(req.Body)
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(r.body)), Request: req}, nil
}

func recorded(t *testing.T, name string) (*jev.Client, *replay) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "jev", name))
	if err != nil {
		t.Fatal(err)
	}
	rt := &replay{body: body}
	c, err := jev.New(jev.Options{BaseURL: "http://replay.invalid", APIKey: "test-key", Transport: rt, BreakerThreshold: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	return c, rt
}

type failAsker struct{ t *testing.T }

func (f failAsker) Ask(context.Context, jev.Call) (jev.Result, error) {
	f.t.Fatal("Jev must not be asked")
	return jev.Result{}, nil
}

type errAsker struct{ err error }

func (e errAsker) Ask(context.Context, jev.Call) (jev.Result, error) { return jev.Result{}, e.err }

func calibrated(t *testing.T) Thresholds {
	t.Helper()
	th, err := LoadThresholds(filepath.Join("testdata", "calibrated"))
	if err != nil {
		t.Fatal(err)
	}
	return th
}

func readingsFor(out Outcome, target string) map[string]Reading {
	m := map[string]Reading{}
	for _, r := range out.Readings {
		if r.TargetID == target {
			m[r.Field] = r
		}
	}
	return m
}

func TestRuleFastPathSkipsJev(t *testing.T) {
	targets := []Target{
		{ID: "t-site", Label: "1. Site Establishment", Kind: "stage", CatalogueStageID: "site", Tier: TierStage},
		{ID: "t-slab", Label: "Pour slab (Stage 2)", Kind: "activity", StageLabel: "Construction", Tier: TierPackage},
	}
	h := sampleHarvest()
	out, err := Read(context.Background(), failAsker{t}, Thresholds{QuestionVersion: QuestionVersion}, h, targets)
	if err != nil {
		t.Fatal(err)
	}
	if out.Asked {
		t.Fatal("asked Jev with every target settled by rule")
	}
	site := readingsFor(out, "t-site")
	if site[FieldStart].DecidedBy != ByRule || site[FieldStart].Band != BandGreen || site[FieldStart].Value != "1/03/2027" || site[FieldStart].Parsed["date"] != "2027-03-01" {
		t.Fatalf("start %+v", site[FieldStart])
	}
	// Evidence for a stage carries its catalogue stage (plan 7.7).
	if site[FieldStart].CatalogueStageID != "site" || readingsFor(out, "t-slab")[FieldStart].CatalogueStageID != "" {
		t.Fatalf("catalogue stage %+v", out.Readings)
	}
	if d := site[FieldDuration]; d.Value != "5 days" {
		t.Fatalf("duration %+v", d)
	}
}

func TestRuleSettlesARowOnlyOnce(t *testing.T) {
	h := sampleHarvest()
	h.Rows = append(h.Rows, Row{ID: "r6", Label: "Pour slab"})
	targets := []Target{{ID: "a", Label: "Pour slab", Kind: "activity", StageLabel: "Construction", Tier: TierPackage}}
	d := NewDraft(h, targets)
	if _, ok := d.Matches()["a"]; ok {
		t.Fatal("two rows with the same label settled by rule")
	}
	twoTargets := []Target{
		{ID: "a", Label: "Roof cladding", Kind: "activity", StageLabel: "Roof", Tier: TierPackage},
		{ID: "b", Label: "roof cladding.", Kind: "activity", StageLabel: "Re-roof", Tier: TierPackage},
	}
	if m := NewDraft(sampleHarvest(), twoTargets).Matches(); len(m) != 0 {
		t.Fatalf("one row settled two targets: %+v", m)
	}
}

func TestEmptyShortlistAsksNothing(t *testing.T) {
	targets := []Target{{ID: "t-zz", Label: "Qq", Kind: "hold_point", StageLabel: "Construction", Tier: TierPackage}}
	d := NewDraft(sampleHarvest(), targets)
	if _, ok := d.Call(); ok {
		t.Fatal("asked a target that no row resembles")
	}
	out := d.Finish()
	if len(out.NotMatched) != 1 || out.NotMatched[0] != "t-zz" || len(out.Readings) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestQuestionIsLiteralAndDateFree(t *testing.T) {
	c, rt := recorded(t, "program_row_pick.json")
	if _, err := Read(context.Background(), c, calibrated(t), sampleHarvest(), sampleTargets()); err != nil {
		t.Fatal(err)
	}
	sent := string(rt.sent)
	for _, banned := range []string{"2027", "/03/", "days", "wks"} {
		if strings.Contains(sent, banned) {
			t.Fatalf("request carries %q: %s", banned, sent)
		}
	}
	d := NewDraft(sampleHarvest(), sampleTargets())
	call, ok := d.Call()
	if !ok || call.Priority != jev.PriorityBackground || call.QuestionVersion != "program-row-1" || call.Deadline.Seconds() != 6 {
		t.Fatalf("%+v", call)
	}
	q := call.Questions["t-frame"]
	want := "Using `rows`, which row of this program is the same work as the activity \"Wall framing\" (stage: Construction)? Choose none when no row is that work. A heading or summary row that only groups other rows is not the same work. A row for a different trade or a different stage of the same trade is not the same work."
	if q.Instructions != want {
		t.Fatalf("instructions %q", q.Instructions)
	}
	crit := q.Criteria.(map[string]any)
	if crit["none"] != "No row in this program is the same work as this activity." || crit["r4"] != (rowCriterion{Row: "Frame walls and roof"}) {
		t.Fatalf("criteria %+v", crit)
	}
	// Rows settled by rule are not offered again.
	if _, ok := crit["r3"]; ok {
		t.Fatal("rule-settled row offered")
	}
	st := call.State.(state)
	for _, r := range st.Rows {
		if r.ID == "r3" {
			t.Fatal("state carries a row in no shortlist")
		}
	}
}

func TestCalibratedPickWritesStartFinishDuration(t *testing.T) {
	c, _ := recorded(t, "program_row_pick.json")
	out, err := Read(context.Background(), c, calibrated(t), sampleHarvest(), sampleTargets())
	if err != nil {
		t.Fatal(err)
	}
	frame := readingsFor(out, "t-frame")
	if len(frame) != 3 {
		t.Fatalf("frame readings %+v", out.Readings)
	}
	s := frame[FieldStart]
	if s.Band != BandGreen || s.DecidedBy != ByJev || s.QuestionVersion != QuestionVersion || s.Confidence == nil || s.Value != "22/03/2027" || s.Parsed["date"] != "2027-03-22" || s.Parsed["row"] != "r4" {
		t.Fatalf("start %+v", s)
	}
	if d := frame[FieldDuration]; d.Value != "4 wks" {
		t.Fatalf("duration %+v", d)
	}
	// none writes nothing.
	if len(readingsFor(out, "t-slab")) != 0 {
		t.Fatal("none wrote a reading")
	}
	if len(readingsFor(out, "t-pour")) != 3 || readingsFor(out, "t-pour")[FieldStart].DecidedBy != ByRule {
		t.Fatalf("rule readings %+v", out.Readings)
	}
}

func TestNoneWritesNothing(t *testing.T) {
	c, _ := recorded(t, "program_row_none.json")
	out, err := Read(context.Background(), c, calibrated(t), sampleHarvest(), sampleTargets())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range out.Readings {
		if r.DecidedBy == ByJev {
			t.Fatalf("none wrote %+v", r)
		}
	}
}

func TestUncalibratedAnswersStayBlank(t *testing.T) {
	c, _ := recorded(t, "program_row_pick.json")
	out, err := Read(context.Background(), c, Thresholds{QuestionVersion: QuestionVersion}, sampleHarvest(), sampleTargets())
	if err != nil {
		t.Fatal(err)
	}
	if !out.Asked {
		t.Fatal("not asked")
	}
	for _, r := range out.Readings {
		if r.DecidedBy != ByRule {
			t.Fatalf("an uncalibrated answer was applied: %+v", r)
		}
	}
	// A threshold for another version authorises nothing either.
	th := calibrated(t)
	th.QuestionVersion = "program-row-0"
	if _, apply := th.Band(Family, ptr(0.99), 4); apply {
		t.Fatal("threshold from another question version applied")
	}
	// A cut-off for another option count does not transfer.
	if _, apply := calibrated(t).Band(Family, ptr(0.99), 41); apply {
		t.Fatal("threshold transferred between option counts")
	}
}

func TestGreyOnTimeout(t *testing.T) {
	out, err := Read(context.Background(), errAsker{context.DeadlineExceeded}, calibrated(t), sampleHarvest(), sampleTargets())
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(out.Grey) != "[t-frame t-slab]" {
		t.Fatalf("grey %v", out.Grey)
	}
	for _, r := range out.Readings {
		if r.DecidedBy != ByRule {
			t.Fatalf("grey answer wrote %+v", r)
		}
	}
}

func TestShrinkKeepsNoneAndDropsFragmentsThenPackages(t *testing.T) {
	long := strings.Repeat("with the coordinated services rough-in to the level above ", 2)
	var rows []Row
	for i := 1; i <= 300; i++ {
		rows = append(rows, Row{ID: fmt.Sprintf("r%d", i), Label: fmt.Sprintf("Install partition walls zone %d %s", i, long)})
	}
	h := Harvest{Filename: "big.xlsx", Rows: rows}
	var targets []Target
	for i := 0; i < 20; i++ {
		targets = append(targets, Target{ID: fmt.Sprintf("s%d", i), Label: fmt.Sprintf("Partition walls stage %d", i), Kind: "stage", Tier: TierStage})
	}
	for i := 0; i < 100; i++ {
		targets = append(targets, Target{ID: fmt.Sprintf("p%d", i), Label: fmt.Sprintf("Install partition walls package %d", i), Kind: "activity", StageLabel: "Fit-out", Tier: TierPackage})
	}
	for i := 0; i < 400; i++ {
		targets = append(targets, Target{ID: fmt.Sprintf("f%d", i), Label: fmt.Sprintf("Partition walls fragment %d", i), Kind: "activity", StageLabel: "Fit-out", Tier: TierFragment})
	}
	d := NewDraft(h, targets)
	call, ok := d.Call()
	if !ok {
		t.Fatal("nothing asked")
	}
	if _, err := jev.CheckRequestSize(call); err != nil {
		t.Fatal(err)
	}
	for id, q := range call.Questions {
		crit := q.Criteria.(map[string]any)
		if _, ok := crit["none"]; !ok {
			t.Fatalf("%s lost none", id)
		}
		if len(crit) > shortlistSize+1 || len(crit) > jev.MaxChoiceOptions {
			t.Fatalf("%s has %d options", id, len(crit))
		}
	}
	asked := map[byte]int{}
	for id := range call.Questions {
		asked[id[0]]++
	}
	if asked['f'] != 0 || asked['s'] != 20 {
		t.Fatalf("asked %v: fragments must go before packages and stages", asked)
	}
	out := d.Finish()
	notMatched := map[byte]int{}
	for _, id := range out.NotMatched {
		notMatched[id[0]]++
	}
	if notMatched['f'] != 400 || notMatched['p']+asked['p'] != 100 {
		t.Fatalf("not matched %v", notMatched)
	}
}

func TestMoreThan255CandidatesAreShortlisted(t *testing.T) {
	var rows []Row
	for i := 1; i <= 300; i++ {
		rows = append(rows, Row{ID: fmt.Sprintf("r%d", i), Label: fmt.Sprintf("Concrete pour %d", i)})
	}
	d := NewDraft(Harvest{Filename: "pours.xlsx", Rows: rows}, []Target{{ID: "t", Label: "Concrete pour", Kind: "activity", StageLabel: "Structure", Tier: TierPackage}})
	call, ok := d.Call()
	if !ok {
		t.Fatal("not asked")
	}
	if n := len(call.Questions["t"].Criteria.(map[string]any)); n != shortlistSize+1 || d.Options("t") != n {
		t.Fatalf("%d options", n)
	}
	if n := len(call.State.(state).Rows); n != shortlistSize {
		t.Fatalf("state rows %d", n)
	}
}

func TestTwoTargetsOnOneRowAreNeverGreen(t *testing.T) {
	d := NewDraft(sampleHarvest(), []Target{
		{ID: "a", Label: "Wall framing", Kind: "activity", StageLabel: "Construction", Tier: TierPackage},
		{ID: "b", Label: "Roof framing", Kind: "activity", StageLabel: "Construction", Tier: TierPackage},
	})
	if _, ok := d.Call(); !ok {
		t.Fatal("not asked")
	}
	conf := 0.99
	d.Apply(jev.Result{Answers: map[string]jev.Answer{
		"a": {Type: jev.TypeChoice, Choice: "r4", Confidence: &conf},
		"b": {Type: jev.TypeChoice, Choice: "r4", Confidence: &conf},
	}}, nil, calibrated(t))
	for _, id := range []string{"a", "b"} {
		if m := d.Matches()[id]; m.Band != BandAmber {
			t.Fatalf("%s %+v", id, m)
		}
	}
}

func TestNormalise(t *testing.T) {
	for in, want := range map[string]string{
		"1.2 Frame Walls":          "frame walls",
		"Stage 3 - Roof, cladding": "roof cladding",
		"  POUR   slab (L2)  ":     "pour slab l2",
		"Level 2 slab":             "level 2 slab",
	} {
		if got := Normalise(in); got != want {
			t.Fatalf("%q → %q, want %q", in, got, want)
		}
	}
}

func ptr(v float64) *float64 { return &v }
