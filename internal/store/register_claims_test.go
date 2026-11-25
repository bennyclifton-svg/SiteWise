package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/store"
)

// Correspondence between parties, first slice
// (docs/plans/2026-11-23-correspondence-between-parties.md, section 18).

func claimInput(kind, title, details string) store.DeliveryInput {
	return store.DeliveryInput{Content: delivery.Content{Kind: kind, Title: title, Details: json.RawMessage(details)}}
}

func TestRegisterClaimsNumberPerKind(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	rfi, err := s.CreateDelivery(ctx, orgA, projectA, userA, rfiInput("Which membrane", ""))
	if err != nil {
		t.Fatal(err)
	}
	nod, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("delay_notice", "Rain, week 46", `{"received_on":"2026-11-16","cause":"Rain"}`))
	if err != nil {
		t.Fatal(err)
	}
	eot1, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Rain, week 46", `{"received_on":"2026-11-23","days_claimed":5,"notice_id":"`+nod.ID+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	eot2, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Late steel", `{"received_on":"2026-11-23"}`))
	if err != nil {
		t.Fatal(err)
	}
	letter, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("letter", "Hoarding licence", `{"received_on":"2026-11-20"}`))
	if err != nil {
		t.Fatal(err)
	}
	// Each kind numbers from 1 in its own run.
	for _, c := range []struct {
		it   delivery.Item
		want string
	}{{rfi, "RFI-001"}, {nod, "NOD-1"}, {eot1, "EOT-1"}, {eot2, "EOT-2"}, {letter, "LTR-1"}} {
		if c.it.Number == nil || delivery.Label(c.it.Kind, *c.it.Number) != c.want {
			t.Fatalf("%s numbered %v", c.want, c.it.Number)
		}
	}
	// A notice of delay owes no response; an EOT claim is due 28 calendar
	// days after receipt by SiteWise's default; a letter five working days.
	if nod.TargetDate != nil {
		t.Fatalf("notice due %v", *nod.TargetDate)
	}
	if eot1.TargetDate == nil || *eot1.TargetDate != "2026-12-21" {
		t.Fatalf("EOT due %v", eot1.TargetDate)
	}
	if letter.TargetDate == nil || *letter.TargetDate != "2026-11-27" {
		t.Fatalf("letter due %v", letter.TargetDate)
	}
	// A numbered kind keeps its kind.
	kind := "delay_notice"
	if _, err := s.PatchDelivery(ctx, orgA, projectA, eot2.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: eot2.Version, Kind: &kind}}); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("kind change %v", err)
	}
	// A notice id from another kind is refused.
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Bad link", `{"notice_id":"`+rfi.ID+`"}`)); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("bad notice link %v", err)
	}
	view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if err != nil {
		t.Fatal(err)
	}
	if view.EOTDays != 28 || !view.EOTTypical {
		t.Fatalf("EOT period %d %v", view.EOTDays, view.EOTTypical)
	}
	for _, it := range view.Items {
		if it.ID == eot1.ID && (it.DueBasis != "default" || it.With != "You") {
			t.Fatalf("EOT read %q with %q", it.DueBasis, it.With)
		}
	}
}

func TestRegisterEOTResultAndCompletion(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	eot, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Rain, week 46", `{"received_on":"2026-11-23","days_claimed":5,"raised_by":"Acme Builders"}`))
	if err != nil {
		t.Fatal(err)
	}
	assessed := "assessed"
	patch := func(details string, status *string) (delivery.Item, error) {
		return s.PatchDelivery(ctx, orgA, projectA, eot.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: eot.Version, Status: status, Details: json.RawMessage(details)}})
	}
	// Assessed needs a decided result; fewer days than claimed needs reasons.
	if _, err := patch(`{"received_on":"2026-11-23","days_claimed":5}`, &assessed); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("assessed without result %v", err)
	}
	if _, err := patch(`{"received_on":"2026-11-23","days_claimed":5,"result":{"outcome":"granted_in_part","days_granted":2}}`, &assessed); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("part without reasons %v", err)
	}
	if _, err := patch(`{"received_on":"2026-11-23","days_claimed":5,"result":{"outcome":"granted","days_granted":2}}`, &assessed); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("granted fewer %v", err)
	}
	// With no agreed date the days are still totalled and nothing moves.
	change := store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "delivery"}
	marked := store.WithChange(ctx, change)
	got, err := s.PatchDelivery(marked, orgA, projectA, eot.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: eot.Version, Status: &assessed,
		Details: json.RawMessage(`{"received_on":"2026-11-23","days_claimed":5,"raised_by":"Acme Builders","result":{"outcome":"granted_in_part","days_granted":2,"reasons":"Three days fell on a weekend shutdown."}}`)}})
	if err != nil {
		t.Fatal(err)
	}
	if r := delivery.EOT(got.Content).Result; r == nil || r.AssessedOn == nil {
		t.Fatalf("assessed on not filled %+v", r)
	}
	view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if err != nil {
		t.Fatal(err)
	}
	if c := view.Completion; c.DaysGranted != 2 || c.Claims != 1 || c.Agreed != nil || c.Adjusted != nil {
		t.Fatalf("completion without date %+v", c)
	}
	if view.Items[0].With != "Acme Builders" {
		t.Fatalf("assessed with %q", view.Items[0].With)
	}
	// The agreed date is a term; the date after EOTs is computed at read.
	agreed := "2027-03-01"
	if err := s.SetDeliveryTerms(ctx, orgA, projectA, userA, store.TermsPatch{AgreedCompletion: store.Optional[string]{Present: true, Value: &agreed}}); err != nil {
		t.Fatal(err)
	}
	view, _ = s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if c := view.Completion; c.Adjusted == nil || *c.Adjusted != "2027-03-03" {
		t.Fatalf("adjusted %+v", c)
	}
	// Withdrawing the claim takes its grant out.
	withdrawn := "withdrawn"
	if _, err := s.PatchDelivery(ctx, orgA, projectA, eot.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: got.Version, Status: &withdrawn}}); err != nil {
		t.Fatal(err)
	}
	view, _ = s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if c := view.Completion; c.DaysGranted != 0 || *c.Adjusted != agreed {
		t.Fatalf("after withdraw %+v", c)
	}
}

func TestRegisterTermsOptionalAndUndo(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	ten := 10
	if err := s.SetDeliveryTerms(ctx, orgA, projectA, userA, store.TermsPatch{EOTDays: store.Optional[int]{Present: true, Value: &ten}}); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if err != nil {
		t.Fatal(err)
	}
	// Setting the EOT period leaves the RFI period at its default.
	if view.EOTDays != 10 || view.EOTTypical || view.ResponseDays != 5 || !view.ResponseTypical {
		t.Fatalf("terms %+v", view)
	}
	eot, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Late steel", `{"received_on":"2026-11-23"}`))
	if err != nil || *eot.TargetDate != "2026-12-03" {
		t.Fatalf("ten days %v %v", eot.TargetDate, err)
	}
	bad := 0
	if err := s.SetDeliveryTerms(ctx, orgA, projectA, userA, store.TermsPatch{EOTDays: store.Optional[int]{Present: true, Value: &bad}}); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("zero days %v", err)
	}
	badDate := "1 March"
	if err := s.SetDeliveryTerms(ctx, orgA, projectA, userA, store.TermsPatch{AgreedCompletion: store.Optional[string]{Present: true, Value: &badDate}}); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("bad date %v", err)
	}
	if err := s.SetDeliveryTerms(ctx, orgB, projectA, userB, store.TermsPatch{EOTDays: store.Optional[int]{Present: true, Value: &ten}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org %v", err)
	}
	// Undo of a terms change puts the default back.
	change := store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "delivery"}
	agreed := "2027-03-01"
	if err := s.SetDeliveryTerms(store.WithChange(ctx, change), orgA, projectA, userA, store.TermsPatch{AgreedCompletion: store.Optional[string]{Present: true, Value: &agreed}, EOTDays: store.Optional[int]{Present: true}}); err != nil {
		t.Fatal(err)
	}
	view, _ = s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if view.EOTDays != 28 || view.Completion.Agreed == nil {
		t.Fatalf("cleared %+v", view.Completion)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err != nil {
		t.Fatal(err)
	}
	view, _ = s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
	if view.EOTDays != 10 || view.Completion.Agreed != nil {
		t.Fatalf("after undo %d %+v", view.EOTDays, view.Completion)
	}
}

func TestRegisterClaimLateness(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Rain", `{"received_on":"2026-10-01"}`)); err != nil {
		t.Fatal(err)
	}
	view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-01")
	if err != nil {
		t.Fatal(err)
	}
	// Due 29 October; 1 November is three calendar days late.
	if l := view.Items[0].LateDays; l == nil || *l != 3 {
		t.Fatalf("late %v", l)
	}
}

// TestRegisterClaimsBudget holds the register read and write budgets
// (bench/budgets.json delivery_read and delivery_write: p50 100 ms, p90
// 250 ms) on 1,000 rows, a quarter of them assessed EOT claims, so the
// completion total and the claim due dates are inside the timed read.
func TestRegisterClaimsBudget(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	for i := 0; i < 1000; i++ {
		in := rfiInput(fmt.Sprintf("Seeded RFI %d", i), "2026-11-02")
		if i%4 == 1 {
			in = claimInput("eot_claim", fmt.Sprintf("Seeded EOT %d", i), `{"received_on":"2026-11-02","days_claimed":5,"result":{"outcome":"granted_in_part","days_granted":2,"reasons":"Weekend."}}`)
			in.Status = "assessed"
		}
		if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, in); err != nil {
			t.Fatal(err)
		}
	}
	var reads, writes []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-24")
		if err != nil {
			t.Fatal(err)
		}
		reads = append(reads, time.Since(start))
		if view.Completion.DaysGranted != 500 {
			t.Fatalf("days granted %d", view.Completion.DaysGranted)
		}
		start = time.Now()
		if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", fmt.Sprintf("Timed EOT %d", i), `{"received_on":"2026-11-23","days_claimed":5}`)); err != nil {
			t.Fatal(err)
		}
		writes = append(writes, time.Since(start))
	}
	check := func(name string, d []time.Duration) {
		slices.Sort(d)
		p50, p90 := d[len(d)/2], d[len(d)*9/10]
		t.Logf("%s p50 %v p90 %v", name, p50, p90)
		if p50 > 100*time.Millisecond || p90 > 250*time.Millisecond {
			t.Fatalf("%s over budget: p50 %v p90 %v", name, p50, p90)
		}
	}
	check("delivery_read", reads)
	check("delivery_write", writes)
}

// A builder hosting the project lodges an EOT claim with the principal: the
// claim is the host's own, with the principal until the result comes back,
// and then with the host. Its due date is the principal's period from the
// day it was lodged.
func TestRegisterRaisedClaimDirection(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	in := claimInput("eot_claim", "Rain, week 47", `{"direction":"out","received_on":"2026-11-23","days_claimed":3}`)
	in.Content.OwnerText = "Principal"
	out, err := s.CreateDelivery(ctx, orgA, projectA, userA, in)
	if err != nil {
		t.Fatal(err)
	}
	if !delivery.Raised(out.Content) || delivery.With(out.Content) != "Principal" {
		t.Fatalf("raised claim with %q", delivery.With(out.Content))
	}
	if out.TargetDate == nil || *out.TargetDate != "2026-12-21" {
		t.Fatalf("raised claim due %v", out.TargetDate)
	}
	// With no responder named, a raised claim is with nobody named, never "You".
	bare, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("letter", "Site access", `{"direction":"out","received_on":"2026-11-23"}`))
	if err != nil {
		t.Fatal(err)
	}
	if w := delivery.With(bare.Content); w != "" {
		t.Fatalf("raised letter with %q", w)
	}
	status := "assessed"
	assessed, err := s.PatchDelivery(ctx, orgA, projectA, out.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: out.Version,
		Status: &status, Details: json.RawMessage(`{"direction":"out","received_on":"2026-11-23","days_claimed":3,"result":{"outcome":"granted","days_granted":3,"reasons":""}}`)}})
	if err != nil {
		t.Fatal(err)
	}
	if w := delivery.With(assessed.Content); w != "You" {
		t.Fatalf("assessed raised claim with %q", w)
	}
	// Only in and out are directions.
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, claimInput("eot_claim", "Bad", `{"direction":"sideways"}`)); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("bad direction %v", err)
	}
}
