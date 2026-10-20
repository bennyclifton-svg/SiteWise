package programread

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// sampleStages are one architect package's stages: each has its program
// package activity and the program stage it sits in.
func sampleStages() []Stage {
	return []Stage{
		{ID: "ps-concept", Label: "Concept design", TargetActivityID: "a-concept", CatalogueStageID: "concept_design"},
		{ID: "ps-dd", Label: "Design development", TargetActivityID: "a-dd", CatalogueStageID: "design_development"},
		{ID: "ps-da", Label: "DA documentation", TargetActivityID: "a-da", CatalogueStageID: "approvals"},
	}
}

// sheet is a returned response form: one run per cell of the R7 sheet.
func sheet(rows ...[]string) identity.Text {
	t := identity.Text{Format: "xlsx", TextLayer: true}
	for i, row := range rows {
		for j, cell := range row {
			if cell != "" {
				t.Runs = append(t.Runs, identity.Run{Text: cell, Source: identity.Source{Sheet: "R7 Program", Row: i + 1, Col: j + 1}})
			}
		}
	}
	return t
}

func r7Sheet(dd string) identity.Text {
	return sheet(
		[]string{"Program: your weeks for each stage."},
		nil,
		rfp.R7Columns,
		[]string{"Concept design", "3", "", "Survey by week 1"},
		[]string{"Design development", dd, "2", ""},
		[]string{"DA documentation", "6", "8", "Planning report"},
	)
}

const filler = "We will coordinate with the consultant team and attend meetings as required by the client to keep the design moving and resolve issues as they arise. "

// proseText is a letter-style proposal with no R7 table.
func proseText() identity.Text {
	lines := []string{
		"Fee proposal: architectural services for 12 Smith Street",
		"Concept design: we allow 3 weeks from receipt of survey.",
		filler + filler,
		"Design development will run 4–6 wks after concept sign-off; council review of the DA takes 8 weeks.",
		filler + filler,
		"DA documentation 30 working days.",
		filler + filler,
		"The whole commission is expected to take 26 weeks.",
	}
	t := identity.Text{Format: "pdf", TextLayer: true}
	for _, l := range lines {
		t.Runs = append(t.Runs, identity.Run{Text: l, Source: identity.Source{Page: 1}})
	}
	return t
}

func stageCalibrated(t *testing.T) Thresholds {
	t.Helper()
	th, err := LoadThresholds(filepath.Join("testdata", "stageweeks"))
	if err != nil {
		t.Fatal(err)
	}
	return th
}

func byStage(out StageOutcome) map[string]Reading {
	m := map[string]Reading{}
	for _, r := range out.Readings {
		m[r.Parsed["package_stage_id"].(string)] = r
	}
	return m
}

func TestWellCompletedR7IsSettledEntirelyByRule(t *testing.T) {
	in := StageWeeksInput{Filename: "Smith Architects response.xlsx", Text: r7Sheet("5"), PerWeek: 5, Stages: sampleStages()}
	d := NewStageDraft(in)
	if _, ok := d.Call(); ok {
		t.Fatalf("a well-completed R7 returnable asked Jev: %v", d.Asked())
	}
	out, err := ReadStageWeeks(context.Background(), failAsker{t}, Thresholds{QuestionVersion: QuestionVersion}, in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Asked || len(out.Readings) != 3 {
		t.Fatalf("%+v", out)
	}
	dd := byStage(out)["ps-dd"]
	if dd.DecidedBy != ByRule || dd.Band != BandGreen || dd.Value != "5" || dd.Parsed["stated"] != "5 weeks" || dd.Parsed["low"] != 25 || dd.Parsed["high"] != 25 ||
		dd.TargetID != "a-dd" || dd.CatalogueStageID != "design_development" || dd.Field != FieldDuration || dd.QuestionVersion != "" {
		t.Fatalf("dd %+v", dd)
	}
	// Weeks are the program's working weeks, converted in code.
	in.PerWeek = 4
	if r := byStage(NewStageDraft(in).Finish())["ps-concept"]; r.Parsed["low"] != 12 {
		t.Fatalf("4-day week %+v", r)
	}
}

func TestR7RuleNeedsTheExactTableAndAPlainNumber(t *testing.T) {
	// "4-6" is not a plain number: that row is not settled by rule.
	out := NewStageDraft(StageWeeksInput{Text: r7Sheet("4-6"), PerWeek: 5, Stages: sampleStages()}).Finish()
	if _, ok := byStage(out)["ps-dd"]; ok || len(out.Readings) != 2 {
		t.Fatalf("%+v", out.Readings)
	}
	// A table whose header differs is not our R7 table.
	other := sheet([]string{"Stage", "Weeks", "Authority weeks", "Inputs you need, and by when"}, []string{"Concept design", "3"})
	if out := NewStageDraft(StageWeeksInput{Text: other, PerWeek: 5, Stages: sampleStages()}).Finish(); len(out.Readings) != 0 {
		t.Fatalf("%+v", out.Readings)
	}
	// A PDF has no cells: the rule never reads it.
	pdf := r7Sheet("5")
	pdf.Format = "pdf"
	if out := NewStageDraft(StageWeeksInput{Text: pdf, PerWeek: 5, Stages: sampleStages()}).Finish(); len(out.Readings) != 0 {
		t.Fatalf("%+v", out.Readings)
	}
	// A stage listed twice is not settled by rule.
	twice := sheet(rfp.R7Columns, []string{"Concept design", "3"}, []string{"Concept design", "4"})
	if out := NewStageDraft(StageWeeksInput{Text: twice, PerWeek: 5, Stages: sampleStages()}).Finish(); len(out.Readings) != 0 {
		t.Fatalf("%+v", out.Readings)
	}
}

func TestStageQuestionIsLiteralWithVerbatimCandidates(t *testing.T) {
	c, rt := recorded(t, "program_stage_weeks_pick.json")
	in := StageWeeksInput{Filename: "Smith Architects proposal.pdf", Text: proseText(), PerWeek: 5, Stages: sampleStages()}
	if _, err := ReadStageWeeks(context.Background(), c, stageCalibrated(t), in); err != nil {
		t.Fatal(err)
	}
	var sent struct {
		State map[string]any `json:"state"`
	}
	if err := json.Unmarshal(rt.sent, &sent); err != nil {
		t.Fatal(err)
	}
	if len(sent.State) != 1 || sent.State["excerpt"] == nil {
		t.Fatalf("state %v", sent.State)
	}
	// Only text near the stage labels: not the last line, which is 300
	// characters past the nearest label.
	if ex := sent.State["excerpt"].(string); strings.Contains(ex, "26 weeks") || !strings.Contains(ex, "4–6 wks") {
		t.Fatalf("excerpt %q", ex)
	}
	d := NewStageDraft(in)
	call, ok := d.Call()
	if !ok || call.Priority != jev.PriorityBackground || call.QuestionVersion != "proposal-program-1" || call.Deadline.Seconds() != 6 {
		t.Fatalf("%+v", call)
	}
	q := call.Questions["ps-dd"]
	want := "Using `excerpt`, which candidate does the proposal state as the firm's own duration for the stage \"Design development\"? A duration stated for authority review, for a different stage, or for the whole commission is not it. Choose none when the proposal states no duration for this stage."
	if q.Type != jev.TypeChoice || q.Instructions != want {
		t.Fatalf("question %+v", q)
	}
	crit := q.Criteria.(map[string]any)
	if len(crit) != 3 || crit["c1"] != (valueCriterion{Value: "4–6 wks"}) || crit["c2"] != (valueCriterion{Value: "8 weeks"}) || crit["none"] != "The proposal states no duration of its own for this stage." {
		t.Fatalf("criteria %+v", crit)
	}
	if fmt.Sprint(d.Candidates("ps-concept")) != "[3 weeks]" || fmt.Sprint(d.Candidates("ps-da")) != "[30 working days]" {
		t.Fatalf("candidates %v %v", d.Candidates("ps-concept"), d.Candidates("ps-da"))
	}
}

func TestCalibratedPickConvertsInCodeAndARangeIsLowHigh(t *testing.T) {
	c, _ := recorded(t, "program_stage_weeks_pick.json")
	out, err := ReadStageWeeks(context.Background(), c, stageCalibrated(t), StageWeeksInput{Text: proseText(), PerWeek: 5, Stages: sampleStages()})
	if err != nil {
		t.Fatal(err)
	}
	got := byStage(out)
	concept := got["ps-concept"]
	if concept.Value != "3 weeks" || concept.Parsed["low"] != 15 || concept.Parsed["high"] != 15 || concept.DecidedBy != ByJev || concept.Band != BandGreen ||
		concept.QuestionVersion != StageWeeksQuestionVersion || concept.Confidence == nil {
		t.Fatalf("concept %+v", concept)
	}
	dd := got["ps-dd"]
	if dd.Value != "4–6 wks" || dd.Parsed["low"] != 20 || dd.Parsed["high"] != 30 || dd.Band != BandAmber {
		t.Fatalf("range %+v", dd)
	}
	// none writes nothing.
	if _, ok := got["ps-da"]; ok || len(out.Readings) != 2 {
		t.Fatalf("none wrote %+v", out.Readings)
	}
}

func TestStageNoneWritesNothing(t *testing.T) {
	c, _ := recorded(t, "program_stage_weeks_none.json")
	out, err := ReadStageWeeks(context.Background(), c, stageCalibrated(t), StageWeeksInput{Text: proseText(), PerWeek: 5, Stages: sampleStages()})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Asked || len(out.Readings) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestStageAnswersStayBlankUncalibrated(t *testing.T) {
	c, _ := recorded(t, "program_stage_weeks_pick.json")
	// The shipped file shape: program_row's version, nothing for this family.
	out, err := ReadStageWeeks(context.Background(), c, Thresholds{QuestionVersion: QuestionVersion}, StageWeeksInput{Text: proseText(), PerWeek: 5, Stages: sampleStages()})
	if err != nil {
		t.Fatal(err)
	}
	if !out.Asked || len(out.Readings) != 0 {
		t.Fatalf("an uncalibrated answer was applied: %+v", out.Readings)
	}
	// program_row cut-offs never authorise this family.
	if _, apply := calibrated(t).Band(StageWeeksFamily, ptr(0.99), 2); apply {
		t.Fatal("program_row thresholds applied to program_stage_weeks")
	}
	// Cut-offs fitted on another version of this map authorise nothing.
	th := stageCalibrated(t)
	th.Versions[StageWeeksFamily] = "proposal-program-0"
	if _, apply := th.Band(StageWeeksFamily, ptr(0.99), 2); apply {
		t.Fatal("another question version applied")
	}
	// A cut-off for another option count does not transfer.
	if _, apply := stageCalibrated(t).Band(StageWeeksFamily, ptr(0.99), 9); apply {
		t.Fatal("threshold transferred between option counts")
	}
}

func TestStageGreyOnTimeout(t *testing.T) {
	out, err := ReadStageWeeks(context.Background(), errAsker{context.DeadlineExceeded}, stageCalibrated(t), StageWeeksInput{Text: proseText(), PerWeek: 5, Stages: sampleStages()})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(out.Grey) != "[ps-concept ps-da ps-dd]" || len(out.Readings) != 0 {
		t.Fatalf("%+v", out)
	}
}

func TestHarvestIsWithin200CharactersAndAtMost20(t *testing.T) {
	near := "Design development " + strings.Repeat("x", 150) + " 7 weeks"
	far := "Design development " + strings.Repeat("y", 250) + " 9 weeks"
	in := StageWeeksInput{PerWeek: 5, Stages: []Stage{{ID: "s", Label: "Design development"}}, Text: identity.Text{Format: "pdf", Runs: []identity.Run{
		{Text: near, Source: identity.Source{Page: 1}}, {Text: strings.Repeat("z", 300), Source: identity.Source{Page: 1}}, {Text: far, Source: identity.Source{Page: 1}},
	}}}
	if got := fmt.Sprint(NewStageDraft(in).Candidates("s")); got != "[7 weeks]" {
		t.Fatalf("candidates %s", got)
	}
	var many []string
	for i := 1; i <= 25; i++ {
		many = append(many, strconv.Itoa(i)+"d")
	}
	in.Text.Runs = []identity.Run{{Text: "Design development " + strings.Join(many, " "), Source: identity.Source{Page: 1}}}
	d := NewStageDraft(in)
	if c := d.Candidates("s"); len(c) != 20 || c[0] != "1d" || c[19] != "20d" {
		t.Fatalf("candidates %v", c)
	}
	call, ok := d.Call()
	if !ok || len(call.Questions["s"].Criteria.(map[string]any)) != 21 || d.Options("s") != 21 {
		t.Fatal("20 candidates and none")
	}
	// A stage the proposal never names gets no question.
	in.Stages = []Stage{{ID: "q", Label: "Tender documentation"}}
	if _, ok := NewStageDraft(in).Call(); ok {
		t.Fatal("asked a stage with no candidates")
	}
}

func TestStageCallFitsAndKeepsNone(t *testing.T) {
	var stages []Stage
	var runs []identity.Run
	for i := 0; i < 40; i++ {
		label := fmt.Sprintf("Stage %d %s", i, strings.Repeat("w", 150))
		stages = append(stages, Stage{ID: fmt.Sprintf("s%d", i), Label: label})
		for k := 1; k <= 25; k++ {
			runs = append(runs, identity.Run{Text: fmt.Sprintf("%s %d weeks", label, k), Source: identity.Source{Page: 1}})
		}
	}
	in := StageWeeksInput{PerWeek: 5, Stages: stages, Text: identity.Text{Format: "pdf", Runs: runs}}
	call, ok := NewStageDraft(in).Call()
	if !ok || len(call.Questions) != 40 {
		t.Fatalf("asked %d", len(call.Questions))
	}
	if _, err := jev.CheckRequestSize(call); err != nil {
		t.Fatal(err)
	}
	for id, q := range call.Questions {
		crit := q.Criteria.(map[string]any)
		if _, ok := crit["none"]; !ok || len(crit) > maxStageCandidates+1 {
			t.Fatalf("%s has %d options", id, len(crit))
		}
	}
	if n := len(call.State.(stageState).Excerpt); n > 12000 {
		t.Fatalf("excerpt %d bytes", n)
	}
}

func TestParseStated(t *testing.T) {
	for in, want := range map[string][2]int{"6 weeks": {30, 30}, "30 working days": {30, 30}, "4–6 wks": {20, 30}, "6-week": {30, 30}, "10 business days": {10, 10}, "2 to 3 weeks": {10, 15}} {
		d, ok := ParseStated(in, 5)
		if !ok || d.Low != want[0] || d.High != want[1] {
			t.Fatalf("%q → %+v %v", in, d, ok)
		}
	}
	if _, ok := ParseStated("1.3 weeks", 5); ok {
		t.Fatal("a part day converted")
	}
}

// proposalMapSHA256 is the hash of the committed proposal-1 question map
// for proposalFixture. Reading B never changes that map (plan 13.3 Q3):
// a change there is a new proposals.QuestionVersion and a new calibration
// of every proposal question.
const proposalMapSHA256 = "4438efcf684bb2e657a3cf1a50ee0c190f03889603749f06036957ec1e91d53f"

func TestProposalOneQuestionMapIsUnchanged(t *testing.T) {
	if proposals.QuestionVersion != "proposal-1" {
		t.Fatalf("proposal question version %q", proposals.QuestionVersion)
	}
	ev := proposals.Harvest("Smith Architects fee proposal.pdf", proseText())
	pc := proposals.Context{Packages: []proposals.Package{{ID: "pkg", Title: "Architect", Discipline: "consultant.architect"}}, PackageID: "pkg",
		Schedules: map[string][]rfp.ScheduleItem{"pkg": {{Key: "stage:concept_design", Kind: rfp.ItemStage, Label: "Concept design", Base: true}, {Key: "stage:design_development", Kind: rfp.ItemStage, Label: "Design development", Base: true}}}}
	call, ok := proposals.NewDraft(ev, pc, nil).Call()
	if !ok {
		t.Fatal("no proposal call")
	}
	for id := range call.Questions {
		if strings.HasPrefix(id, "program") || strings.Contains(id, "weeks") {
			t.Fatalf("proposal-1 asks %s", id)
		}
	}
	raw, err := json.Marshal(call.Questions)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != proposalMapSHA256 {
		t.Fatalf("proposal-1 question map changed: %s", got)
	}
}
