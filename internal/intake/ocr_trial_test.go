package intake_test

import (
	"context"
	"errors"
	"testing"

	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

type ocrAsker struct {
	calls    int
	priority jev.Priority
}

func (a *ocrAsker) Ask(_ context.Context, c jev.Call) (jev.Result, error) {
	a.calls++
	a.priority = c.Priority
	return jev.Result{}, nil
}

func TestOCRTrialReviewAndCorrectionProtection(t *testing.T) {
	st := storeFrom(t)
	ctx := context.Background()
	org, project := testID(84), testID(85)
	seedProject(t, st, org, project)
	id := seedPending(t, st, org, project, "sheet.pdf", "ocr-review")
	if err := st.MarkNotFiled(ctx, org, id, intake.ReasonNoText); err != nil {
		t.Fatal(err)
	}
	cat, err := intake.LoadCatalog(intakeData(t))
	if err != nil {
		t.Fatal(err)
	}
	ask := &ocrAsker{}
	svc, err := intake.NewService(st, ask, cat, intake.Thresholds{QuestionVersion: intake.QuestionVersion})
	if err != nil {
		t.Fatal(err)
	}
	text := ruledText("A-101", "B", "Ground floor")
	text.PageCount = 1
	text.Runs = text.Runs[:3]
	writes, err := svc.TrialOCR(ctx, org, id, text)
	if err != nil {
		t.Fatal(err)
	}
	if ask.calls != 1 || ask.priority != jev.PriorityBackground {
		t.Fatalf("calls=%d priority=%s", ask.calls, ask.priority)
	}
	for _, d := range writes {
		if d.Band == "green" || d.Field == "supersedes" {
			t.Fatalf("unsafe OCR write %+v", d)
		}
	}
	// A correction after the OCR/Jev work must beat the stale proposal.
	if err = svc.Correct(ctx, org, id, "title", "Owner's title"); err != nil {
		t.Fatal(err)
	}
	if _, err = st.CommitOCRTrial(ctx, testID(86), id, writes); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("cross-org: %v", err)
	}
	bad := append([]store.DecisionWrite(nil), writes...)
	bad[0].Band = "green"
	if _, err = st.CommitOCRTrial(ctx, org, id, bad); err == nil {
		t.Fatal("accepted green OCR")
	}
	view, err := st.DocumentView(ctx, org, id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != store.StatusNotFiled {
		t.Fatal("failed trial changed status")
	}
	out, err := st.CommitOCRTrial(ctx, org, id, writes)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != store.StatusFiled || out.Number != "A-101" || out.Revision != "B" {
		t.Fatalf("outcome %+v", out)
	}
	title, number := false, false
	for _, d := range out.Decisions {
		if d.Field == "title" {
			title = d.Value == "Owner's title" && d.DecidedBy == "user"
		}
		if d.Field == "number" {
			number = d.Band == "amber" && d.Value == "A-101"
		}
	}
	if !title || !number {
		t.Fatalf("decisions %+v", out.Decisions)
	}
	view, err = st.DocumentView(ctx, org, id)
	if err != nil || view.Reason != "" {
		t.Fatalf("stale not-filed reason: %+v %v", view, err)
	}
	if _, err = st.CommitOCRTrial(ctx, org, id, writes); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("repeated commit: %v", err)
	}
}
