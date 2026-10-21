package intake_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
	"testing"
	"time"
)

type visualAsker struct{ calls int }

func (a *visualAsker) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	a.calls++
	options := call.Questions["title"].Criteria.(map[string]any)
	for key, value := range options {
		if c, ok := value.(intake.VisualCandidate); ok && c.Value == "LITERAL REVIEW TITLE" {
			return jev.Result{Answers: map[string]jev.Answer{"title": {Type: jev.TypeChoice, Choice: key}}}, nil
		}
	}
	return jev.Result{}, errors.New("candidate missing")
}
func TestVisualReviewIsDurableScopedAndNeverChangesCorrections(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "offline", 503) }), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, time.Millisecond*200)
	ctx := context.Background()
	org, project := runOrg, testID(192)
	seedProject(t, st, org, project)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	doc := uploadFixture(t, up, project, "identity-page.pdf")
	runner := intake.NewRunner(blobs, st, svc)
	if err := runner.Run(ctx, org, doc.DocumentID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Correct(ctx, org, doc.DocumentID, "title", "Owner correction"); err != nil {
		t.Fatal(err)
	}
	before, _ := st.DocumentView(ctx, org, doc.DocumentID)
	if err := st.QueueVisualIdentity(ctx, testID(193), doc.DocumentID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("wrong org queued", err)
	}
	for i := 0; i < 2; i++ {
		if err := st.QueueVisualIdentity(ctx, org, doc.DocumentID); err != nil {
			t.Fatal(err)
		}
	}
	ask := &visualAsker{}
	v := &intake.VisualReader{Store: st, Blobs: blobs, Ask: ask, Render: func(context.Context, string, bool, bool) (identity.VisualEvidence, error) {
		return identity.VisualEvidence{Pages: 1, Images: []identity.VisualImage{{Label: "Page", Page: 1, Data: "fixture"}}, Lines: []identity.VisualLine{{Text: "LITERAL REVIEW TITLE", Page: 1}}}, nil
	}}
	w := jobs.Worker{Store: st, Visual: v.Run, Kinds: []string{store.JobKindVisualIdentity}}
	if err := w.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	review, err := st.VisualIdentity(ctx, org, doc.DocumentID)
	if err != nil || review.Status != "ready" {
		t.Fatalf("review %+v %v", review, err)
	}
	var result intake.VisualResult
	if err := json.Unmarshal(review.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result.Suggestions["title"].Value != "LITERAL REVIEW TITLE" {
		t.Fatal(result)
	}
	after, _ := st.DocumentView(ctx, org, doc.DocumentID)
	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Fatal("visual job changed filing")
	}
	if err := st.QueueVisualIdentity(ctx, org, doc.DocumentID); err != nil {
		t.Fatal(err)
	}
	if err := w.Once(ctx, org); !errors.Is(err, store.ErrIdle) {
		t.Fatal("cached job was rerun", err)
	}
	if ask.calls != 1 {
		t.Fatal("more than one fan-out")
	}
	wrong, _ := st.VisualIdentity(ctx, testID(193), doc.DocumentID)
	if wrong.Status != "not_requested" {
		t.Fatal("cross-org evidence leaked")
	}
}

func TestVisualTitleAppearsInRegisterWithoutChangingSavedField(t *testing.T) {
	st, svc, _ := openFiling(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "offline", 503) }), intake.Thresholds{QuestionVersion: intake.QuestionVersion}, 200*time.Millisecond)
	ctx := context.Background()
	org, project := runOrg, testID(194)
	seedProject(t, st, org, project)
	blobs := openBlobs(t, t.TempDir(), 1<<20)
	up := intake.NewUploader(blobs, st)
	doc := uploadFixture(t, up, project, "identity-page.pdf")
	if _, err := st.CommitFiling(ctx, org, doc.DocumentID, store.CommitFiling{Decisions: []store.DecisionWrite{{Field: "title", Value: "", Band: "blank", DecidedBy: "jev"}}}); err != nil {
		t.Fatal(err)
	}
	pending, err := st.DocumentView(ctx, org, doc.DocumentID)
	if err != nil || pending.VisualStatus != "queued" {
		t.Fatal("pending image status missing", pending, err)
	}
	ask := &visualAsker{}
	reader := &intake.VisualReader{Store: st, Blobs: blobs, Ask: ask, Render: func(context.Context, string, bool, bool) (identity.VisualEvidence, error) {
		return identity.VisualEvidence{Pages: 1, Lines: []identity.VisualLine{{Text: "LITERAL REVIEW TITLE", Page: 1}}}, nil
	}}
	worker := jobs.Worker{Store: st, Visual: reader.Run, Kinds: []string{store.JobKindVisualIdentity}}
	if err := worker.Once(ctx, org); err != nil {
		t.Fatal(err)
	}
	views, err := st.ProjectDocumentViews(ctx, org, project)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 1 || views[0].VisualTitle != "LITERAL REVIEW TITLE" || views[0].VisualStatus != "ready" {
		t.Fatalf("register proposal missing: %+v", views)
	}
	for _, f := range views[0].Fields {
		if f.Field == "title" && f.Value != "" {
			t.Fatal("proposal accepted without user action")
		}
	}
	if err := svc.Correct(ctx, org, doc.DocumentID, "title", ""); err != nil {
		t.Fatal(err)
	}
	view, err := st.DocumentView(ctx, org, doc.DocumentID)
	if err != nil || view.VisualTitle != "" {
		t.Fatal("user blank overwritten", view, err)
	}
	wrong, err := st.ProjectDocumentViews(ctx, testID(195), project)
	if err != nil || len(wrong) != 0 {
		t.Fatal("wrong-org register leak", wrong, err)
	}
}
