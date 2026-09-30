package intake_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

func TestThresholdFileDisablesAction(t *testing.T) {
	got, err := intake.LoadThresholds(intakeData(t))
	if err != nil {
		t.Fatal(err)
	}
	confidence := 0.99
	for _, field := range []string{intake.FieldKind, intake.FieldDiscipline, intake.FieldLifecycle, intake.FieldNumber, intake.FieldRevision, intake.FieldTitle, intake.FieldSupersedes} {
		band, apply := got.Band(field, &confidence, 2)
		if apply || band != intake.BandBlank {
			t.Fatalf("%s band %s apply %v", field, band, apply)
		}
	}
}

func TestPartialThresholdRejected(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{
		"question_version": "intake-1",
		"reconciliation": "partial cut-off",
		"questions": {"number": {"green": 0.9, "amber": null, "options": null}}
	}`)
	if err := os.WriteFile(filepath.Join(dir, "thresholds.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := intake.LoadThresholds(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestBandStaysWithQuestionShape(t *testing.T) {
	thresholds := intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Questions: map[string]intake.Threshold{
			intake.FieldNumber: cut(0.9, 0.6, 3),
		},
	}
	high := 0.95
	mid := 0.7
	low := 0.2
	if band, ok := thresholds.Band(intake.FieldNumber, &high, 3); !ok || band != intake.BandGreen {
		t.Fatalf("green %s %v", band, ok)
	}
	if band, ok := thresholds.Band(intake.FieldNumber, &mid, 3); !ok || band != intake.BandAmber {
		t.Fatalf("amber %s %v", band, ok)
	}
	if band, ok := thresholds.Band(intake.FieldNumber, &low, 3); ok || band != intake.BandBlank {
		t.Fatalf("blank %s %v", band, ok)
	}
	if _, ok := thresholds.Band(intake.FieldNumber, &high, 4); ok {
		t.Fatal("threshold transferred to another option count")
	}
	if _, ok := thresholds.Band(intake.FieldTitle, &high, 3); ok {
		t.Fatal("threshold transferred to another question")
	}
}

func TestFileAllRuleSendsNoJev(t *testing.T) {
	st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		t.Errorf("unexpected jev request: %s", body)
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 0)

	ctx := context.Background()
	org, project := fileOrg, testID(1)
	seedProject(t, st, org, project)
	docID := seedPending(t, st, org, project, "A-100 Floor Plan [P1].pdf", "a100")

	filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P1", "Floor Plan"))
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 0 {
		t.Fatalf("jev calls %d", hits.Load())
	}
	if filed.Status != store.StatusFiled || filed.Number != "A-100" || filed.Revision != "P1" {
		t.Fatalf("%+v", filed)
	}
	if filed.SupersedesID != "" {
		t.Fatalf("supersedes %s", filed.SupersedesID)
	}
	assertDecision(t, filed.Decisions, intake.FieldNumber, "A-100", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldRevision, "P1", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldTitle, "Floor Plan", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldKind, "drawing", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldDiscipline, "consultant.architect", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldLifecycle, "construction", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldDate, "", intake.BandBlank, intake.DecidedByRule)
	if jobStatus(t, st, org, docID, store.JobKindIntake) != store.JobStatusDone {
		t.Fatal("intake job")
	}
	if jobStatus(t, st, org, docID, store.JobKindFullText) != "queued" {
		t.Fatal("full text job")
	}
	if _, ok := findJob(t, st, org, docID, store.JobKindJevRetry); ok {
		t.Fatal("retry job")
	}
}

func TestFileAmbiguousSendsOneJev(t *testing.T) {
	var body []byte
	st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		writeRecorded(t, w, body, map[string]string{intake.FieldNumber: "A-100"}, map[string]float64{intake.FieldNumber: 0.97})
	}), intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Questions:       map[string]intake.Threshold{intake.FieldNumber: cut(0.9, 0.6, 3)},
	}, 0)

	ctx := context.Background()
	org, project := fileOrg, testID(1)
	seedProject(t, st, org, project)
	docID := seedPending(t, st, org, project, "A-100 Floor Plan [P1].pdf", "ambiguous")
	text := ruledText("A-100", "P1", "Floor Plan")
	text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})

	filed, err := svc.File(ctx, org, docID, text)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("jev calls %d body %s", hits.Load(), body)
	}
	keys := questionKeys(t, body)
	if len(keys) != 1 || keys[intake.FieldNumber] != 3 {
		t.Fatalf("questions %v body %s", keys, body)
	}
	number := assertDecision(t, filed.Decisions, intake.FieldNumber, "A-100", intake.BandGreen, intake.DecidedByJev)
	if number.QuestionVersion != intake.QuestionVersion || number.Confidence == nil {
		t.Fatalf("%+v", number)
	}
	if filed.Number != "A-100" {
		t.Fatalf("number %s", filed.Number)
	}
}

func TestFileDeadlineKeepsRuleValues(t *testing.T) {
	st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-r.Context().Done():
		case <-timer.C:
		}
	}), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 100*time.Millisecond)

	ctx := context.Background()
	org, project := fileOrg, testID(1)
	seedProject(t, st, org, project)
	docID := seedPending(t, st, org, project, "A-100 Floor Plan [P1].pdf", "deadline")
	text := ruledText("A-100", "P1", "Floor Plan")
	text.Runs = append(text.Runs, identity.Run{Text: "Rev A"})

	filed, err := svc.File(ctx, org, docID, text)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("jev calls %d", hits.Load())
	}
	assertDecision(t, filed.Decisions, intake.FieldNumber, "A-100", intake.BandGreen, intake.DecidedByRule)
	assertDecision(t, filed.Decisions, intake.FieldRevision, "", intake.BandGrey, intake.DecidedByJev)
	if filed.Number != "A-100" || filed.Revision != "" {
		t.Fatalf("identity %s %s", filed.Number, filed.Revision)
	}
	if jobStatus(t, st, org, docID, store.JobKindJevRetry) != "queued" {
		t.Fatal("retry")
	}
	if jobStatus(t, st, org, docID, store.JobKindFullText) != "queued" {
		t.Fatal("full text")
	}
}

func TestFileBlankDoesNotInvent(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		writeRecorded(t, w, body, map[string]string{intake.FieldNumber: "B-200"}, map[string]float64{intake.FieldNumber: 0.2})
	}), intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Questions:       map[string]intake.Threshold{intake.FieldNumber: cut(0.9, 0.6, 3)},
	}, 0)

	ctx := context.Background()
	org, project := fileOrg, testID(1)
	seedProject(t, st, org, project)
	docID := seedPending(t, st, org, project, "A-100 Floor Plan [P1].pdf", "blank")
	text := ruledText("A-100", "P1", "Floor Plan")
	text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})

	filed, err := svc.File(ctx, org, docID, text)
	if err != nil {
		t.Fatal(err)
	}
	number := assertDecision(t, filed.Decisions, intake.FieldNumber, "", intake.BandBlank, intake.DecidedByJev)
	if number.Confidence == nil || number.Value == "B-200" || number.Value == "A-100" {
		t.Fatalf("%+v", number)
	}
	doc, err := st.GetDocument(ctx, org, docID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Number != "" {
		t.Fatalf("stored number %q", doc.Number)
	}
}

func TestFileNoneIsNotANumber(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		writeRecorded(t, w, body, map[string]string{intake.FieldNumber: "none"}, map[string]float64{intake.FieldNumber: 0.99})
	}), intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Questions:       map[string]intake.Threshold{intake.FieldNumber: cut(0.9, 0.6, 3)},
	}, 0)

	ctx := context.Background()
	org, project := fileOrg, testID(1)
	seedProject(t, st, org, project)
	docID := seedPending(t, st, org, project, "A-100 Floor Plan [P1].pdf", "none")
	text := ruledText("A-100", "P1", "Floor Plan")
	text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})

	filed, err := svc.File(ctx, org, docID, text)
	if err != nil {
		t.Fatal(err)
	}
	assertDecision(t, filed.Decisions, intake.FieldNumber, "", intake.BandGreen, intake.DecidedByJev)
	if filed.Number != "" {
		t.Fatalf("number %q", filed.Number)
	}
}

func TestFileCorrectionBeatsLateJev(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		once.Do(func() { close(started) })
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		writeRecorded(t, w, body, map[string]string{intake.FieldNumber: "A-100"}, map[string]float64{intake.FieldNumber: 0.99})
	}), intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Questions:       map[string]intake.Threshold{intake.FieldNumber: cut(0.9, 0.6, 3)},
	}, 5*time.Second)

	ctx := context.Background()
	org, project := fileOrg, testID(1)
	seedProject(t, st, org, project)
	docID := seedPending(t, st, org, project, "A-100 Floor Plan [P1].pdf", "correct")
	text := ruledText("A-100", "P1", "Floor Plan")
	text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})

	go func() {
		defer close(release)
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Error("jev was not called")
			return
		}
		if err := svc.Correct(ctx, org, docID, intake.FieldNumber, "USER-1"); err != nil {
			t.Error(err)
		}
	}()

	filed, err := svc.File(ctx, org, docID, text)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("jev calls %d", hits.Load())
	}
	assertDecision(t, filed.Decisions, intake.FieldNumber, "USER-1", intake.BandGreen, intake.DecidedByUser)
	rows, err := st.DocumentDecisions(ctx, org, docID)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Field == intake.FieldNumber && row.Version != 1 {
			t.Fatalf("version %d value %s", row.Version, row.Value)
		}
	}
	if filed.Number != "USER-1" {
		t.Fatalf("number %s", filed.Number)
	}
}

func TestFileSupersessionStaysInSeries(t *testing.T) {
	ctx := context.Background()
	org := fileOrg
	otherOrg := fileOrgB

	t.Run("link", func(t *testing.T) {
		var body []byte
		st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			body, err = io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			parsed := decodeCall(t, body)
			choice := onlyPrior(t, parsed.Questions[intake.FieldSupersedes].Criteria)
			writeRecorded(t, w, body, map[string]string{intake.FieldSupersedes: choice}, map[string]float64{intake.FieldSupersedes: 0.99})
		}), supersedesThreshold(2, 0.95, 0.5), 0)

		project := testID(1)
		seedProject(t, st, org, project)
		seedProject(t, st, otherOrg, testID(2))
		prior := seedNumbered(t, st, org, project, "A-100", "P1", "prior")
		otherProject := testID(3)
		if err := st.CreateProject(ctx, org, otherProject, "Other"); err != nil {
			t.Fatal(err)
		}
		elsewhere := seedNumbered(t, st, org, otherProject, "A-100", "P1", "elsewhere")
		otherNumber := seedNumbered(t, st, org, project, "B-200", "P1", "other-number")
		foreign := seedNumbered(t, st, otherOrg, testID(2), "A-100", "P1", "foreign")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "newer")

		filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P2", "Floor Plan"))
		if err != nil {
			t.Fatal(err)
		}
		if hits.Load() != 1 {
			t.Fatalf("jev calls %d", hits.Load())
		}
		keys := questionKeys(t, body)
		if len(keys) != 1 || keys[intake.FieldSupersedes] != 2 {
			t.Fatalf("questions %v", keys)
		}
		criteria := decodeCall(t, body).Questions[intake.FieldSupersedes].Criteria
		if _, ok := criteria[prior]; !ok {
			t.Fatalf("missing prior in %v", criteria)
		}
		for _, id := range []string{docID, elsewhere, otherNumber, foreign} {
			if _, ok := criteria[id]; ok {
				t.Fatalf("option %s in %v", id, criteria)
			}
		}
		if filed.SupersedesID != prior {
			t.Fatalf("supersedes %s", filed.SupersedesID)
		}
		assertDecision(t, filed.Decisions, intake.FieldSupersedes, prior, intake.BandGreen, intake.DecidedByJev)
		priorDoc, err := st.GetDocument(ctx, org, prior)
		if err != nil {
			t.Fatal(err)
		}
		if priorDoc.Revision != "P1" || priorDoc.Number != "A-100" || priorDoc.SupersedesID != "" || priorDoc.Status != store.StatusFiled {
			t.Fatalf("prior changed %+v", priorDoc)
		}
		if _, err := st.GetDocument(ctx, org, foreign); err == nil {
			t.Fatal("foreign document visible")
		}
	})

	t.Run("incompatible series", func(t *testing.T) {
		st, svc, _ := openFiling(t, pickSupersedes(t, 0.99), supersedesThreshold(2, 0.95, 0.5), 0)
		project := testID(10)
		seedProject(t, st, org, project)
		prior := seedNumbered(t, st, org, project, "A-100", "A", "alpha")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "p2")
		filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P2", "Floor Plan"))
		if err != nil {
			t.Fatal(err)
		}
		if filed.SupersedesID != "" {
			t.Fatalf("linked %s", filed.SupersedesID)
		}
		supersedes := assertDecision(t, filed.Decisions, intake.FieldSupersedes, prior, intake.BandBlank, intake.DecidedByJev)
		if supersedes.Value != prior {
			t.Fatalf("%+v", supersedes)
		}
		bothRemain(t, st, org, docID, prior)
	})

	t.Run("not later", func(t *testing.T) {
		st, svc, _ := openFiling(t, pickSupersedes(t, 0.99), supersedesThreshold(2, 0.95, 0.5), 0)
		project := testID(11)
		seedProject(t, st, org, project)
		prior := seedNumbered(t, st, org, project, "A-100", "P3", "later")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "earlier")
		filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P2", "Floor Plan"))
		if err != nil {
			t.Fatal(err)
		}
		if filed.SupersedesID != "" {
			t.Fatalf("linked %s", filed.SupersedesID)
		}
		bothRemain(t, st, org, docID, prior)
	})

	t.Run("cycle", func(t *testing.T) {
		st, svc, _ := openFiling(t, pickSupersedes(t, 0.99), supersedesThreshold(2, 0.95, 0.5), 0)
		project := testID(12)
		seedProject(t, st, org, project)
		prior := seedNumbered(t, st, org, project, "A-100", "P1", "cycle-prior")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "cycle-new")
		if err := st.Supersede(ctx, org, prior, docID); err != nil {
			t.Fatal(err)
		}
		filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P2", "Floor Plan"))
		if err != nil {
			t.Fatal(err)
		}
		if filed.SupersedesID != "" {
			t.Fatalf("linked %s", filed.SupersedesID)
		}
		got, err := st.GetDocument(ctx, org, prior)
		if err != nil {
			t.Fatal(err)
		}
		if got.SupersedesID != docID {
			t.Fatalf("existing link %s", got.SupersedesID)
		}
	})

	t.Run("conflicting number", func(t *testing.T) {
		var priorA, priorB string
		st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			writeRecorded(t, w, body, map[string]string{
				intake.FieldNumber:     "A-100",
				intake.FieldSupersedes: priorB,
			}, map[string]float64{
				intake.FieldNumber:     0.99,
				intake.FieldSupersedes: 0.99,
			})
		}), intake.Thresholds{
			QuestionVersion: intake.QuestionVersion,
			Questions: map[string]intake.Threshold{
				intake.FieldNumber:     cut(0.9, 0.5, 3),
				intake.FieldSupersedes: cut(0.9, 0.5, 3),
			},
		}, 0)
		project := testID(13)
		seedProject(t, st, org, project)
		priorA = seedNumbered(t, st, org, project, "A-100", "P1", "series-a")
		priorB = seedNumbered(t, st, org, project, "B-200", "P1", "series-b")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "conflict")
		text := ruledText("A-100", "P2", "Floor Plan")
		text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})
		filed, err := svc.File(ctx, org, docID, text)
		if err != nil {
			t.Fatal(err)
		}
		if filed.SupersedesID != "" {
			t.Fatalf("linked %s", filed.SupersedesID)
		}
		if filed.Number != "A-100" {
			t.Fatalf("number %s", filed.Number)
		}
		bothRemain(t, st, org, docID, priorA)
		bothRemain(t, st, org, docID, priorB)
	})

	t.Run("selected number", func(t *testing.T) {
		var priorB string
		st, svc, hits := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
				return
			}
			writeRecorded(t, w, body, map[string]string{
				intake.FieldNumber:     "B-200",
				intake.FieldSupersedes: priorB,
			}, map[string]float64{
				intake.FieldNumber:     0.99,
				intake.FieldSupersedes: 0.99,
			})
		}), intake.Thresholds{
			QuestionVersion: intake.QuestionVersion,
			Questions: map[string]intake.Threshold{
				intake.FieldNumber:     cut(0.9, 0.5, 3),
				intake.FieldSupersedes: cut(0.9, 0.5, 3),
			},
		}, 0)
		project := testID(14)
		seedProject(t, st, org, project)
		_ = seedNumbered(t, st, org, project, "A-100", "P1", "keep-a")
		priorB = seedNumbered(t, st, org, project, "B-200", "P1", "keep-b")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "selected")
		text := ruledText("A-100", "P2", "Floor Plan")
		text.Runs = append(text.Runs, identity.Run{Text: "Drawing No B-200"})
		filed, err := svc.File(ctx, org, docID, text)
		if err != nil {
			t.Fatal(err)
		}
		if hits.Load() != 1 {
			t.Fatalf("jev calls %d", hits.Load())
		}
		if filed.SupersedesID != priorB || filed.Number != "B-200" {
			t.Fatalf("%+v", filed)
		}
	})

	t.Run("amber", func(t *testing.T) {
		st, svc, _ := openFiling(t, pickSupersedes(t, 0.7), supersedesThreshold(2, 0.95, 0.5), 0)
		project := testID(15)
		seedProject(t, st, org, project)
		prior := seedNumbered(t, st, org, project, "A-100", "P1", "amber")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "amber-new")
		filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P2", "Floor Plan"))
		if err != nil {
			t.Fatal(err)
		}
		if filed.SupersedesID != "" {
			t.Fatalf("linked %s", filed.SupersedesID)
		}
		assertDecision(t, filed.Decisions, intake.FieldSupersedes, prior, intake.BandAmber, intake.DecidedByJev)
		bothRemain(t, st, org, docID, prior)
	})

	t.Run("unknown threshold", func(t *testing.T) {
		thresholds, err := intake.LoadThresholds(intakeData(t))
		if err != nil {
			t.Fatal(err)
		}
		st, svc, hits := openFiling(t, pickSupersedes(t, 0.99), thresholds, 0)
		project := testID(16)
		seedProject(t, st, org, project)
		prior := seedNumbered(t, st, org, project, "A-100", "P1", "unknown")
		docID := seedPending(t, st, org, project, "A-100 Floor Plan [P2].pdf", "unknown-new")
		filed, err := svc.File(ctx, org, docID, ruledText("A-100", "P2", "Floor Plan"))
		if err != nil {
			t.Fatal(err)
		}
		if hits.Load() != 1 {
			t.Fatalf("jev calls %d", hits.Load())
		}
		if filed.SupersedesID != "" {
			t.Fatalf("linked %s", filed.SupersedesID)
		}
		assertDecision(t, filed.Decisions, intake.FieldSupersedes, "", intake.BandBlank, intake.DecidedByJev)
		assertDecision(t, filed.Decisions, intake.FieldNumber, "A-100", intake.BandGreen, intake.DecidedByRule)
		bothRemain(t, st, org, docID, prior)
	})
}

const (
	fileOrg  = "33333333-3333-4333-8333-333333333331"
	fileOrgB = "33333333-3333-4333-8333-333333333332"
)

func testID(n int) string {
	return fmt.Sprintf("33333333-3333-4333-8333-%012x", n)
}

func intakeData(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "data", "intake")
}

func cut(green, amber float64, options int) intake.Threshold {
	g, a, n := green, amber, options
	return intake.Threshold{Green: &g, Amber: &a, Options: &n}
}

func supersedesThreshold(options int, green, amber float64) intake.Thresholds {
	return intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Questions: map[string]intake.Threshold{
			intake.FieldSupersedes: cut(green, amber, options),
		},
	}
}

func openFiling(t *testing.T, handler http.HandlerFunc, thresholds intake.Thresholds, deadline time.Duration) (*store.Store, *intake.Service, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	client, err := jev.New(jev.Options{
		BaseURL:  srv.URL,
		APIKey:   "test-key",
		Deadline: deadline,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	cat, err := intake.LoadCatalog(intakeData(t))
	if err != nil {
		t.Fatal(err)
	}
	st := storeFrom(t)
	svc, err := intake.NewService(st, client, cat, thresholds)
	if err != nil {
		t.Fatal(err)
	}
	return st, svc, &hits
}

func ruledText(number, revision, title string) identity.Text {
	return identity.Text{Runs: []identity.Run{
		{Text: "Drawing No " + number},
		{Text: "Rev " + revision},
		{Text: "Title " + title},
		{Text: "Drawing"},
		{Text: "Architectural"},
		{Text: "Construction"},
	}}
}

func seedPending(t *testing.T, st *store.Store, org, project, filename, tag string) string {
	t.Helper()
	return seedDoc(t, st, org, project, filename, "", "", store.StatusPending, tag, true)
}

func seedNumbered(t *testing.T, st *store.Store, org, project, number, revision, tag string) string {
	t.Helper()
	return seedDoc(t, st, org, project, number+" "+revision+".pdf", number, revision, store.StatusFiled, tag, false)
}

func seedDoc(t *testing.T, st *store.Store, org, project, filename, number, revision, status, tag string, job bool) string {
	t.Helper()
	ctx := context.Background()
	fileID := seedID()
	docID := seedID()
	if err := st.CreateFile(ctx, org, store.File{
		ID:        fileID,
		ProjectID: project,
		SHA256:    hashOf(org + project + tag),
		ByteSize:  8,
		MediaType: "application/pdf",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateDocument(ctx, org, store.Document{
		ID:        docID,
		ProjectID: project,
		FileID:    fileID,
		Filename:  filename,
		Status:    status,
		Number:    number,
		Revision:  revision,
	}); err != nil {
		t.Fatal(err)
	}
	if job {
		if err := st.EnqueueJob(ctx, org, seedID(), docID, store.JobKindIntake); err != nil {
			t.Fatal(err)
		}
	}
	return docID
}

var seedSeq atomic.Uint64

func seedID() string {
	return fmt.Sprintf("44444444-4444-4444-8444-%012x", seedSeq.Add(1))
}

func hashOf(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

func assertDecision(t *testing.T, list []intake.Decision, field, value, band, by string) intake.Decision {
	t.Helper()
	for _, d := range list {
		if d.Field != field {
			continue
		}
		if d.Value != value || d.Band != band || d.DecidedBy != by {
			t.Fatalf("%s %+v", field, d)
		}
		return d
	}
	t.Fatalf("missing %s", field)
	return intake.Decision{}
}

func bothRemain(t *testing.T, st *store.Store, org, docID, priorID string) {
	t.Helper()
	ctx := context.Background()
	doc, err := st.GetDocument(ctx, org, docID)
	if err != nil {
		t.Fatal(err)
	}
	prior, err := st.GetDocument(ctx, org, priorID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.SupersedesID != "" || doc.Status != store.StatusFiled {
		t.Fatalf("new %+v", doc)
	}
	if prior.Status == "" {
		t.Fatalf("prior %+v", prior)
	}
}

func jobStatus(t *testing.T, st *store.Store, org, docID, kind string) string {
	t.Helper()
	job, ok := findJob(t, st, org, docID, kind)
	if !ok {
		t.Fatalf("missing %s", kind)
	}
	return job.Status
}

func findJob(t *testing.T, st *store.Store, org, docID, kind string) (store.Job, bool) {
	t.Helper()
	jobs, err := st.ListJobs(context.Background(), org)
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		if job.DocumentID == docID && job.Kind == kind {
			return job, true
		}
	}
	return store.Job{}, false
}

type recordedCall struct {
	Questions map[string]struct {
		Criteria map[string]string `json:"criteria"`
	} `json:"questions"`
}

func decodeCall(t *testing.T, body []byte) recordedCall {
	t.Helper()
	var call recordedCall
	if err := json.Unmarshal(body, &call); err != nil {
		t.Fatal(err)
	}
	return call
}

func questionKeys(t *testing.T, body []byte) map[string]int {
	t.Helper()
	call := decodeCall(t, body)
	out := make(map[string]int, len(call.Questions))
	for id, q := range call.Questions {
		out[id] = len(q.Criteria)
	}
	return out
}

func onlyPrior(t *testing.T, criteria map[string]string) string {
	t.Helper()
	var id string
	for key := range criteria {
		if key == "new" {
			continue
		}
		if id != "" {
			t.Errorf("priors %v", criteria)
			return ""
		}
		id = key
	}
	if id == "" {
		t.Error("no prior option")
	}
	return id
}

func pickSupersedes(t *testing.T, confidence float64) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		choice := onlyPrior(t, decodeCall(t, body).Questions[intake.FieldSupersedes].Criteria)
		writeRecorded(t, w, body, map[string]string{intake.FieldSupersedes: choice}, map[string]float64{intake.FieldSupersedes: confidence})
	}
}

func writeRecorded(t *testing.T, w http.ResponseWriter, request []byte, pick map[string]string, confidence map[string]float64) {
	t.Helper()
	var call recordedCall
	if err := json.Unmarshal(request, &call); err != nil {
		t.Error(err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	answers := make(map[string]any, len(call.Questions))
	for field, q := range call.Questions {
		choice := pick[field]
		if choice == "" {
			t.Errorf("no choice for %s", field)
			return
		}
		if _, ok := q.Criteria[choice]; !ok {
			t.Errorf("choice %s not in %v", choice, q.Criteria)
			return
		}
		probs := make(map[string]float64, len(q.Criteria))
		for id := range q.Criteria {
			probs[id] = 0
		}
		probs[choice] = 1
		conf := confidence[field]
		answers[field] = map[string]any{
			"type":          "choice",
			"choice":        choice,
			"probabilities": probs,
			"confidence":    conf,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"model":   config.PinnedJevModel,
		"answers": answers,
		"usage":   map[string]int{"input_tokens": 40, "output_tokens": 8},
	}); err != nil {
		t.Error(err)
	}
}
