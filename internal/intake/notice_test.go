package intake_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"sitewise/internal/correspondence"
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"sitewise/internal/jev"
	"sitewise/internal/mailin"
)

func emailDraft(t *testing.T, body string) *intake.Draft {
	t.Helper()
	m := mailin.Message{FromAddress: "eng@consult.example", To: "p-0@in.example.test", Subject: "Retaining wall", Date: time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC), BodyPlain: body}
	name, raw := mailin.BodyDocument(m, time.Now())
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(raw), int64(len(raw)), identity.Limits{})
	if err != nil {
		t.Fatal(err)
	}
	d := intake.NewDraft(draftCatalog(t), name, text, intake.Harvest(name, text), nil)
	d.Plan(nil, "self")
	return d
}

// The notice question joins the email body's one filing call; a PDF's call
// has none; the options go in their fixed order with "none" first.
func TestNoticeJoinsTheEmailFilingCall(t *testing.T) {
	d := emailDraft(t, "We give notice of a delay to the works.")
	call, ok := d.Call()
	if !ok {
		t.Fatal("no call for an email body")
	}
	q, ok := call.Questions[correspondence.NoticeField]
	if !ok || len(call.Questions) < 2 || !strings.HasSuffix(call.QuestionVersion, "+notice-1") {
		t.Fatalf("questions %v version %q", len(call.Questions), call.QuestionVersion)
	}
	raw, _ := json.Marshal(q.Criteria)
	order := []string{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	_, _ = dec.Token()
	for dec.More() {
		k, _ := dec.Token()
		order = append(order, k.(string))
		var skip json.RawMessage
		_ = dec.Decode(&skip)
	}
	want := "none,rfi,direction,variation_claim,eot_delay,payment_claim,dispute,other_notice"
	if strings.Join(order, ",") != want {
		t.Fatalf("option order %v", order)
	}
	pdf := identity.Text{Format: "pdf", Runs: []identity.Run{{Text: "From: a@b.example"}, {Text: "To: c@d.example"}, {Text: "Subject: x"}}}
	pd := intake.NewDraft(draftCatalog(t), "A-101.pdf", pdf, intake.Harvest("A-101.pdf", pdf), nil)
	pd.Plan(nil, "self")
	if c, ok := pd.Call(); ok {
		if _, asked := c.Questions[correspondence.NoticeField]; asked {
			t.Fatal("a PDF's call asked the notice question")
		}
	}
}

// The pick never reaches decisions; a failed call is grey; no cut-off is blank.
func TestNoticeNeverReachesDecisions(t *testing.T) {
	d := emailDraft(t, "Please treat this as a variation claim.")
	call, _ := d.Call()
	p := 0.97
	result := jev.Result{Answers: map[string]jev.Answer{}}
	for id := range call.Questions {
		result.Answers[id] = jev.Answer{Type: jev.TypeChoice, Choice: "none", Confidence: &p}
	}
	result.Answers[correspondence.NoticeField] = jev.Answer{Type: jev.TypeChoice, Choice: "variation_claim", Confidence: &p}
	d.Apply(result, nil, intake.Thresholds{})
	for _, dec := range d.Decisions() {
		if dec.Field == correspondence.NoticeField {
			t.Fatalf("notice became a decision: %+v", dec)
		}
	}
	if n := d.Notice(result, nil, correspondence.NoticeThresholds{}); n == nil || n.Pick != "variation_claim" || n.Band != "blank" {
		t.Fatalf("no cut-off: %+v", n)
	}
	if n := d.Notice(result, nil, correspondence.NoticeThresholds{Amber: 0.9}); n.Band != "amber" {
		t.Fatalf("past the cut-off: %+v", n)
	}
	if n := d.Notice(jev.Result{}, errors.New("timeout"), correspondence.NoticeThresholds{Amber: 0.9}); n.Band != "grey" || n.Pick != "" {
		t.Fatalf("timeout: %+v", n)
	}
	none := result
	none.Answers = map[string]jev.Answer{correspondence.NoticeField: {Type: jev.TypeChoice, Choice: "none", Confidence: &p}}
	if n := d.Notice(none, nil, correspondence.NoticeThresholds{Amber: 0.5}); n.Band == "amber" {
		t.Fatalf("none shown as a chip: %+v", n)
	}
}
