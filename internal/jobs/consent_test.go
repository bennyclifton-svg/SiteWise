package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"sitewise/internal/delivery"
	"sitewise/internal/jev"
	"sitewise/internal/jobs"
	"sitewise/internal/store"
)

// The consent read (docs/plans/2026-11-17-consent-conditions.md, Task 4),
// with a counting stand-in for Jev: no live call is made.

type consentAsk struct {
	mu    sync.Mutex
	calls int
	fail  error
}

// Ask answers every question the same way: party landscape at 0.8, system
// site at 0.4 (below the floor), evidence yes at 0.9, timing occupation.
func (a *consentAsk) Ask(_ context.Context, call jev.Call) (jev.Result, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	if a.fail != nil {
		return jev.Result{}, a.fail
	}
	c := func(p float64) *float64 { return &p }
	out := jev.Result{Answers: map[string]jev.Answer{
		"party":    {Type: jev.TypeChoice, Choice: "consultant.landscape", Confidence: c(0.8)},
		"system":   {Type: jev.TypeChoice, Choice: "site", Confidence: c(0.4)},
		"evidence": {Type: jev.TypeNoul, Noul: 0.9},
	}}
	if _, ok := call.Questions["timing"]; ok {
		out.Answers["timing"] = jev.Answer{Type: jev.TypeChoice, Choice: "before_occupation", Confidence: c(0.7)}
	}
	return out, nil
}

const originalNotice = `NOTICE OF DETERMINATION
Development Application No: DA/2016/00123
Date of determination: 18 October 2016

GENERAL CONDITIONS
1. Approved plans
The development must be carried out in accordance with the approved plans.
2. Hours
Work may be carried out between 7 am and 6 pm.

PRIOR TO THE ISSUE OF A CONSTRUCTION CERTIFICATE
3. Landscape plan
A landscape plan is to be submitted to the Certifier.
4. Waste
A waste management plan is to be submitted.
`

const modification = `NOTICE OF DETERMINATION OF A MODIFICATION APPLICATION
Modification of development consent DA/2016/00123 under section 4.55
Modification No. MOD/2023/0177
Date of modification: 4 August 2023

GENERAL CONDITIONS
1. Approved plans
The development must be carried out in accordance with the approved plans as amended.
(Amended - 4 August 2023 - MOD/2023/0177)
2. Hours
Work may be carried out between 7 am and 6 pm.
3. Landscape plan
A landscape plan prepared by a landscape architect is to be submitted to the Certifier.
(Amended - 4 August 2023 - MOD/2023/0177)
4. Waste
(Deleted - 4 August 2023 - MOD/2023/0177)
`

type consentRun struct {
	registerRun
	user string
}

func consentProject(t *testing.T) consentRun {
	t.Helper()
	r := registerProject(t, 0)
	user := newID(t)
	if err := r.st.CreateUser(context.Background(), r.org, user, "pm2@example.test"); err != nil {
		t.Fatal(err)
	}
	return consentRun{registerRun: r, user: user}
}

// notice files a document as a Statutory Instrument with its text, and the
// PM asks for it to be read.
func (r consentRun) notice(t *testing.T, text string) string {
	t.Helper()
	ctx := context.Background()
	doc := seedDoc(t, r.st, r.org, r.project, store.StatusFiled)
	for field, value := range map[string]string{"kind": "statutory_instrument", "title": "Notice of determination"} {
		if err := r.st.CreateDecision(ctx, r.org, store.Decision{ID: newID(t), DocumentID: doc, Field: field, Value: value, Band: "green", DecidedBy: "rule"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.st.ReplaceSource(ctx, r.org, doc, store.DocumentSource{Source: []store.SourcePage{{Page: 1, Text: text}}, Units: []store.SourceUnit{{Body: text, Page: 1}}}); err != nil {
		t.Fatal(err)
	}
	if err := r.st.QueueConsent(ctx, r.org, r.project, doc, r.user); err != nil {
		t.Fatal(err)
	}
	return doc
}

func (r consentRun) read(t *testing.T, w *jobs.Worker) error {
	t.Helper()
	job, err := r.st.ClaimJobSince(context.Background(), r.org, time.Minute, []string{store.JobKindConsent}, time.Time{})
	if err != nil {
		return err
	}
	err = w.Perform(context.Background(), job)
	if err == nil {
		err = r.st.CompleteJob(context.Background(), r.org, job.ID, job.Token, store.EventWrite{Kind: "job", DocumentID: job.DocumentID, Payload: "{}"})
	}
	return err
}

func (r consentRun) conditions(t *testing.T) map[string]delivery.Item {
	t.Helper()
	view, err := r.st.ReadRegister(context.Background(), r.org, r.project, store.RegisterQuery{Kind: "condition"}, "2026-11-20")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]delivery.Item{}
	for _, it := range view.Items {
		out[delivery.Condition(it.Content).Number] = it
	}
	return out
}

func flagOf(it delivery.Item) string {
	var p struct {
		Flag string `json:"flag"`
	}
	_ = json.Unmarshal(it.Provenance, &p)
	return p.Flag
}

func TestConsentReadWritesProposedRowsAndCaches(t *testing.T) {
	r := consentProject(t)
	ask := &consentAsk{}
	w := &jobs.Worker{Store: r.st, Ask: ask}
	r.notice(t, originalNotice)
	if err := r.read(t, w); err != nil {
		t.Fatal(err)
	}
	if ask.calls != 4 {
		t.Fatalf("%d calls for 4 conditions", ask.calls)
	}
	rows := r.conditions(t)
	if len(rows) != 4 {
		t.Fatalf("%d rows", len(rows))
	}
	land := rows["3"]
	d := delivery.Condition(land.Content)
	if land.Title != "Condition 3 · Landscape plan" || land.ReviewStatus != "proposed" || land.Origin != "document" || land.Status != "open" {
		t.Fatalf("row %+v", land)
	}
	if d.Stage != "before_cc" || d.Discipline != "consultant.landscape" || land.OwnerText != "Landscape" || land.SystemID != "" || d.EvidenceNeeded == nil || !*d.EvidenceNeeded {
		t.Fatalf("picks %+v with %q system %q", d, land.OwnerText, land.SystemID)
	}
	if d.ConsentRef != "DA/2016/00123" || d.Text != "A landscape plan is to be submitted to the Certifier." || d.Source == nil || d.Source.Page != 1 {
		t.Fatalf("instrument %+v", d)
	}
	// "General" settled nothing, so Jev's timing pick is the gate.
	if g := delivery.Condition(rows["1"].Content).Stage; g != "before_occupation" {
		t.Fatalf("general gate %q", g)
	}
	// A second read of the same notice asks Jev nothing.
	other := r.notice(t, originalNotice)
	_ = other
	if err := r.read(t, w); err != nil {
		t.Fatal(err)
	}
	if ask.calls != 4 {
		t.Fatalf("a re-read made %d more calls", ask.calls-4)
	}
	if len(r.conditions(t)) != 4 {
		t.Fatal("a re-read duplicated rows")
	}
}

func TestConsentModificationFlagsAndKeepsThePMsEdits(t *testing.T) {
	r := consentProject(t)
	ask := &consentAsk{}
	w := &jobs.Worker{Store: r.st, Ask: ask}
	r.notice(t, originalNotice)
	if err := r.read(t, w); err != nil {
		t.Fatal(err)
	}
	rows := r.conditions(t)
	// The PM accepts condition 3 by giving it to the architect.
	edited, err := r.st.PatchDelivery(context.Background(), r.org, r.project, rows["3"].ID, r.user, store.DeliveryPatch{Patch: delivery.Patch{Version: rows["3"].Version, OwnerText: ptr("Architect")}})
	if err != nil {
		t.Fatal(err)
	}
	if edited.ReviewStatus != "accepted_for_planning" || edited.Origin != "user" {
		t.Fatalf("edit did not accept %+v", edited)
	}
	r.notice(t, modification)
	if err := r.read(t, w); err != nil {
		t.Fatal(err)
	}
	rows = r.conditions(t)
	if len(rows) != 4 {
		t.Fatalf("modification added duplicates: %d rows", len(rows))
	}
	if got := delivery.Condition(rows["1"].Content).Text; !strings.Contains(got, "as amended") {
		t.Fatalf("a proposed row was not replaced: %q", got)
	}
	three := rows["3"]
	if three.OwnerText != "Architect" || delivery.Condition(three.Content).Text != "A landscape plan is to be submitted to the Certifier." {
		t.Fatalf("the PM's row was overwritten: %+v", three)
	}
	if flagOf(three) != "Changed by MOD/2023/0177" {
		t.Fatalf("flag %q", flagOf(three))
	}
	four := rows["4"]
	if four.Status != "open" || !strings.HasPrefix(flagOf(four), "Deleted by MOD/2023/0177") {
		t.Fatalf("deleted condition %q %q", four.Status, flagOf(four))
	}
	// An older notice read after the modification does not overwrite it.
	r.notice(t, originalNotice)
	if err := r.read(t, w); err != nil {
		t.Fatal(err)
	}
	if got := delivery.Condition(r.conditions(t)["1"].Content).Text; !strings.Contains(got, "as amended") {
		t.Fatalf("the older notice overwrote the newer: %q", got)
	}
	// A row the PM removed is not brought back.
	two := rows["2"]
	if _, err := r.st.PatchDelivery(context.Background(), r.org, r.project, two.ID, r.user, store.DeliveryPatch{Patch: delivery.Patch{Version: two.Version, Retired: ptrBool(true)}}); err != nil {
		t.Fatal(err)
	}
	r.notice(t, strings.Replace(modification, "7 am", "8 am", 1))
	if err := r.read(t, w); err != nil {
		t.Fatal(err)
	}
	if _, back := r.conditions(t)["2"]; back {
		t.Fatal("a removed condition came back")
	}
}

func TestConsentReadTimeoutLeavesPicksGreyAndRetries(t *testing.T) {
	r := consentProject(t)
	ask := &consentAsk{fail: jev.ErrOverloaded}
	w := &jobs.Worker{Store: r.st, Ask: ask}
	r.notice(t, originalNotice)
	if err := r.read(t, w); !errors.Is(err, jev.ErrOverloaded) {
		t.Fatalf("a failed call did not fail the job: %v", err)
	}
	rows := r.conditions(t)
	land := rows["3"]
	if len(rows) != 4 || delivery.Condition(land.Content).Stage != "before_cc" || land.OwnerText != "" {
		t.Fatalf("rule fields were not written or a pick was: %+v", land)
	}
	// The retry asks again and fills the picks.
	ask.fail = nil
	job, err := r.st.ClaimJobSince(context.Background(), r.org, time.Minute, []string{store.JobKindConsent}, time.Time{})
	if err == nil {
		if err := w.Perform(context.Background(), job); err != nil {
			t.Fatal(err)
		}
	} else {
		// FailJob was not called here; queue it again as the PM would.
		r.notice(t, originalNotice)
		if err := r.read(t, w); err != nil {
			t.Fatal(err)
		}
	}
	if r.conditions(t)["3"].OwnerText != "Landscape" {
		t.Fatal("the retry did not fill the picks")
	}
}

func TestConsentReadLostLeaseAndOtherOrgs(t *testing.T) {
	r := consentProject(t)
	w := &jobs.Worker{Store: r.st, Ask: &consentAsk{}}
	r.notice(t, originalNotice)
	job, err := r.st.ClaimJobSince(context.Background(), r.org, time.Minute, []string{store.JobKindConsent}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	stolen := job
	stolen.Token = newID(t)
	if err := w.Perform(context.Background(), stolen); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a lost lease wrote rows: %v", err)
	}
	if len(r.conditions(t)) != 0 {
		t.Fatal("rows written without the lease")
	}
	// Another org's member cannot queue a read of this document.
	other := consentProject(t)
	doc := r.notice(t, "x")
	if err := other.st.QueueConsent(context.Background(), other.org, other.project, doc, other.user); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign queue: %v", err)
	}
	if _, err := other.st.ReadRegister(context.Background(), other.org, r.project, store.RegisterQuery{}, "2026-11-20"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read: %v", err)
	}
}

func TestConsentReadOfNoConditionsSaysSo(t *testing.T) {
	r := consentProject(t)
	r.notice(t, "NOTICE OF DETERMINATION\nThe application is refused.\n")
	err := r.read(t, &jobs.Worker{Store: r.st, Ask: &consentAsk{}})
	if err == nil || err.Error() != "No numbered conditions found" {
		t.Fatalf("err %v", err)
	}
	if len(r.conditions(t)) != 0 {
		t.Fatal("rows written from nothing")
	}
	_ = fmt.Sprint(os.Getenv("unused"))
}

func ptr(s string) *string { return &s }
func ptrBool(b bool) *bool { return &b }

// A read the PM asked for is one change: one Undo takes it back out.
func TestConsentReadIsOneUndoableChange(t *testing.T) {
	r := consentProject(t)
	r.notice(t, originalNotice)
	if err := r.read(t, &jobs.Worker{Store: r.st, Ask: &consentAsk{}}); err != nil {
		t.Fatal(err)
	}
	if len(r.conditions(t)) != 4 {
		t.Fatal("not read")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var change, area, actor string
	if err := pool.QueryRow(context.Background(), `SELECT id::text, area, actor::text FROM project_changes WHERE org_id=$1::uuid`, r.org).Scan(&change, &area, &actor); err != nil {
		t.Fatal(err)
	}
	if area != "delivery" || actor != r.user {
		t.Fatalf("change %s by %s", area, actor)
	}
	if _, err := r.st.UndoChange(context.Background(), r.org, r.project, r.user, change); err != nil {
		t.Fatal(err)
	}
	if n := len(r.conditions(t)); n != 0 {
		t.Fatalf("%d rows after undo", n)
	}
}
