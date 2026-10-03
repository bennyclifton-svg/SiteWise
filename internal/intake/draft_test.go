package intake_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

func draftCatalog(t *testing.T) intake.Catalog {
	t.Helper()
	cat, err := intake.LoadCatalog(intakeData(t))
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

func TestDisciplineCanExplicitlyAbstain(t *testing.T) {
	d := intake.NewDraft(draftCatalog(t), "agreement.pdf", identity.Text{}, nil, nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("missing classification questions")
	}
	raw, _ := json.Marshal(call.Questions[intake.FieldDiscipline].Criteria)
	var criteria map[string]json.RawMessage
	if err := json.Unmarshal(raw, &criteria); err != nil {
		t.Fatal(err)
	}
	if _, ok := criteria["none"]; !ok {
		t.Fatal("a document without an evidenced discipline is forced into the catalog")
	}
	confidence := .99
	result := jev.Result{Answers: map[string]jev.Answer{intake.FieldDiscipline: {Type: jev.TypeChoice, Choice: "none", Confidence: &confidence}}}
	choices := d.Choices(result)
	if len(choices) != 1 || choices[0].Value != "" || choices[0].Confidence == nil {
		t.Fatalf("abstention must be recorded as an empty discipline: %+v", choices)
	}
}

func TestTitleChoiceRetainsPrintedHeadingProvenance(t *testing.T) {
	candidates := []intake.Candidate{
		{Field: intake.FieldTitle, Display: "Renamed report", Normalized: "renamed report", Provenance: intake.Provenance{Origin: intake.OriginFilename, Heading: true}},
		{Field: intake.FieldTitle, Display: "Printed report title", Normalized: "printed report title", Provenance: intake.Provenance{Origin: intake.OriginText, Heading: true, Page: 1}},
	}
	d := intake.NewDraft(draftCatalog(t), "Renamed report.pdf", identity.Text{Format: "pdf", TextLayer: true}, candidates, nil)
	d.Plan(nil, "")
	call, ok := d.Call()
	if !ok {
		t.Fatal("ambiguous title was settled without Jev")
	}
	raw, err := json.Marshal(call.Questions[intake.FieldTitle].Criteria)
	if err != nil {
		t.Fatal(err)
	}
	var options map[string]map[string]string
	if err := json.Unmarshal(raw, &options); err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, option := range options {
		switch option["value"] {
		case "Renamed report":
			if strings.Contains(option["found"], "printed cover heading") {
				t.Fatal("filename promoted to printed heading")
			}
			seen++
		case "Printed report title":
			if !strings.Contains(option["found"], "printed cover heading") {
				t.Fatal("heading evidence omitted")
			}
			seen++
		}
	}
	if seen != 2 {
		t.Fatalf("title choices missing: %s", raw)
	}
}

// Two number candidates leave the number open, so the draft asks one question
// about it. Nothing in the draft touches a database.
func ambiguousNumber() (string, identity.Text) {
	text := ruledText("A-100", "P1", "Floor Plan")
	text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})
	return "A-100 Floor Plan [P1].pdf", text
}

func TestDraftAllRuleHasNoCall(t *testing.T) {
	name := "A-100 Floor Plan [P1].pdf"
	text := ruledText("A-100", "P1", "Floor Plan")
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan(nil, "self")
	if _, ok := d.Call(); ok {
		t.Fatal("a settled draft must not call Jev")
	}
	got := d.Decisions()
	assertDecision(t, got, intake.FieldNumber, "A-100", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, got, intake.FieldDate, "", intake.BandBlank, intake.DecidedByRule)
}

func TestAcceptedDrawingUsesDesignLifecycleWithoutPromotingConfidence(t *testing.T) {
	thresholds, err := intake.LoadThresholds(intakeData(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		confidence     float64
		userLifecycle  bool
		want, band, by string
	}{
		{"accepted drawing", 1, false, "design", intake.BandAmber, intake.DecidedByRule},
		{"withheld drawing", .1, false, "", intake.BandBlank, intake.DecidedByJev},
		{"user lifecycle", 1, true, "construction", intake.BandGreen, intake.DecidedByUser},
	} {
		t.Run(tc.name, func(t *testing.T) {
			users := map[string]intake.Decision{}
			if tc.userLifecycle {
				users[intake.FieldLifecycle] = intake.Decision{Field: intake.FieldLifecycle, Value: "construction", Band: intake.BandGreen, DecidedBy: intake.DecidedByUser}
			}
			text := identity.Text{Runs: []identity.Run{{Text: "Utility supply layout"}}}
			d := intake.NewDraft(draftCatalog(t), "utility.pdf", text, intake.Harvest("utility.pdf", text), users)
			d.Plan(nil, "self")
			d.Apply(jev.Result{Answers: map[string]jev.Answer{intake.FieldKind: {Type: jev.TypeChoice, Choice: "drawing", Confidence: &tc.confidence}}}, nil, thresholds)
			assertDecision(t, d.Decisions(), intake.FieldLifecycle, tc.want, tc.band, tc.by)
		})
	}
}

func TestDraftChoicesKeepTheRawAnswerBeforeThresholds(t *testing.T) {
	name, text := ambiguousNumber()
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok || len(call.Questions) != 1 {
		t.Fatalf("want one number question, got %+v", call.Questions)
	}
	conf := 0.99
	result := jev.Result{Answers: map[string]jev.Answer{
		intake.FieldNumber: {Type: jev.TypeChoice, Choice: "B-200", Confidence: &conf},
	}}
	choices := d.Choices(result)
	if len(choices) != 1 || choices[0].Value != "B-200" || choices[0].Options != 3 || *choices[0].Confidence != conf {
		t.Fatalf("%+v", choices)
	}
	// Unknown thresholds: the raw choice is evidence for calibration, but
	// nothing is applied.
	if grey := d.Apply(result, nil, intake.Thresholds{QuestionVersion: intake.QuestionVersion}); grey {
		t.Fatal("an answered call is not grey")
	}
	assertDecision(t, d.Decisions(), intake.FieldNumber, "", intake.BandBlank, intake.DecidedByJev)
}

func TestDraftErrorIsGrey(t *testing.T) {
	name, text := ambiguousNumber()
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan(nil, "self")
	if grey := d.Apply(jev.Result{}, errors.New("deadline"), intake.Thresholds{QuestionVersion: intake.QuestionVersion}); !grey {
		t.Fatal("a failed call is grey")
	}
	assertDecision(t, d.Decisions(), intake.FieldNumber, "", intake.BandGrey, intake.DecidedByJev)
	assertDecision(t, d.Decisions(), intake.FieldRevision, "P1", intake.BandGreen, intake.DecidedByRule)
}

func TestDraftKeepsUserDecisions(t *testing.T) {
	name, text := ambiguousNumber()
	users := map[string]intake.Decision{
		intake.FieldNumber: {Field: intake.FieldNumber, Value: "C-300", Band: intake.BandGreen, DecidedBy: intake.DecidedByUser},
	}
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), users)
	d.Plan(nil, "self")
	if _, ok := d.Call(); ok {
		t.Fatal("a user value is not asked again")
	}
	assertDecision(t, d.Decisions(), intake.FieldNumber, "C-300", intake.BandGreen, intake.DecidedByUser)
}

func TestDraftOffersSameNumberPriors(t *testing.T) {
	name := "A-100 Floor Plan [P2].pdf"
	text := ruledText("A-100", "P2", "Floor Plan")
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan([]store.NumberedDocument{
		{ID: "prior", Number: "A-100", Revision: "P1"},
		{ID: "other", Number: "Z-900", Revision: "P1"},
	}, "self")
	call, ok := d.Call()
	if !ok {
		t.Fatal("a same-number prior asks supersession")
	}
	q := call.Questions[intake.FieldSupersedes]
	raw, _ := json.Marshal(q.Criteria)
	var criteria map[string]json.RawMessage
	if err := json.Unmarshal(raw, &criteria); err != nil {
		t.Fatal(err)
	}
	if _, ok := criteria["prior"]; !ok || len(criteria) != 2 {
		t.Fatalf("%+v", q.Criteria)
	}
}

func TestServiceObservesEachPath(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 200*time.Millisecond)
	var mu sync.Mutex
	seen := map[string]int{}
	svc.Observe(func(path string, d time.Duration) {
		mu.Lock()
		seen[path]++
		mu.Unlock()
		if d < 0 {
			t.Errorf("%s negative duration", path)
		}
	})
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	runner := intake.NewRunner(blobs, st, svc)
	project := testID(60)
	seedProject(t, st, runOrg, project)
	filing := uploadFixture(t, up, project, "identity-page.pdf")
	if err := runner.Run(context.Background(), runOrg, filing.DocumentID); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{intake.PathIdentity, intake.PathHarvest, intake.PathRules, intake.PathCommit} {
		if seen[path] != 1 {
			t.Fatalf("%s observed %d times: %v", path, seen[path], seen)
		}
	}
}

func TestNumberQuestionStatesTheRoleAndWhereEachOptionWasFound(t *testing.T) {
	name, text := ambiguousNumber()
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan(nil, "self")
	call, ok := d.Call()
	if !ok || call.QuestionVersion != "intake-76" {
		t.Fatalf("version %q", call.QuestionVersion)
	}
	raw, err := json.Marshal(call.Questions[intake.FieldNumber])
	if err != nil {
		t.Fatal(err)
	}
	var q struct {
		Instructions map[string]string         `json:"instructions"`
		Criteria     map[string]map[string]any `json:"criteria"`
	}
	if err := json.Unmarshal(raw, &q); err != nil {
		t.Fatalf("instructions and options are structured: %v\n%s", err, raw)
	}
	if q.Instructions["question"] == "" || q.Instructions["not_for"] == "" {
		t.Fatalf("%s", raw)
	}
	a100 := q.Criteria["A-100"]
	if a100["value"] != "A-100" || !strings.Contains(a100["found"].(string), "filename") {
		t.Fatalf("an option says its literal and where code found it: %s", raw)
	}
	if q.Criteria["none"]["what"] == nil {
		t.Fatalf("none says what it means: %s", raw)
	}
}
