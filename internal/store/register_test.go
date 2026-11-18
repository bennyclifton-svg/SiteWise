package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"

	"sitewise/internal/delivery"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// The register (docs/plans/2026-11-17-rfis-review-comments.md, Task 2).

func rfiInput(title, raised string) store.DeliveryInput {
	details := `{}`
	if raised != "" {
		details = fmt.Sprintf(`{"question":"Which membrane?","raised_on":%q}`, raised)
	}
	return store.DeliveryInput{Content: delivery.Content{Kind: "rfi", Title: title, Details: json.RawMessage(details)}}
}

func TestRegisterNumbersConcurrentRFIs(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	var wg sync.WaitGroup
	got := make(chan int, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			item, err := s.CreateDelivery(ctx, orgA, projectA, userA, rfiInput(fmt.Sprintf("RFI %d", i), ""))
			if err != nil {
				t.Error(err)
				return
			}
			got <- *item.Number
		}(i)
	}
	wg.Wait()
	close(got)
	seen := map[int]bool{}
	for n := range got {
		seen[n] = true
	}
	for n := 1; n <= 20; n++ {
		if !seen[n] {
			t.Fatalf("numbers %v miss %d", seen, n)
		}
	}
	// A risk carries no number.
	risk, err := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "risk", Title: "Flooding"}})
	if err != nil || risk.Number != nil {
		t.Fatalf("risk %+v %v", risk.Number, err)
	}
}

func TestRegisterDueDatesAndIsolation(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	// A Friday raise is due five working days on: the next Friday.
	item, err := s.CreateDelivery(ctx, orgA, projectA, userA, rfiInput("Basement wall waterproofing", "2026-11-20"))
	if err != nil {
		t.Fatal(err)
	}
	if item.TargetDate == nil || *item.TargetDate != "2026-11-27" || item.Status != "draft" {
		t.Fatalf("due %v status %s", item.TargetDate, item.Status)
	}
	// Moving the raised date moves an untyped due date.
	moved := json.RawMessage(`{"question":"Which membrane?","raised_on":"2026-11-23"}`)
	item, err = s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version, Details: moved}})
	if err != nil || *item.TargetDate != "2026-11-30" {
		t.Fatalf("moved %v %v", item.TargetDate, err)
	}
	// A typed due date survives a later change to the raised date.
	typed := "2026-12-15"
	item, err = s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version, TargetDate: delivery.DateChange{Present: true, Value: &typed}}})
	if err != nil {
		t.Fatal(err)
	}
	again := json.RawMessage(`{"question":"Which membrane?","raised_on":"2026-11-25"}`)
	item, err = s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version, Details: again}})
	if err != nil || *item.TargetDate != typed {
		t.Fatalf("typed due lost %v %v", item.TargetDate, err)
	}
	// No raised date: Due stays blank.
	blank, err := s.CreateDelivery(ctx, orgA, projectA, userA, rfiInput("No date", ""))
	if err != nil || blank.TargetDate != nil {
		t.Fatalf("blank due %v %v", blank.TargetDate, err)
	}
	// Another org's document as an answer is not found; nothing is written.
	foreign := json.RawMessage(fmt.Sprintf(`{"answer_documents":[%q]}`, docB))
	if _, err := s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version, Details: foreign}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign document %v", err)
	}
	// Numbers belong to one kind.
	risk := "risk"
	if _, err := s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version, Kind: &risk}}); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("kind change %v", err)
	}
	// An unknown system, or a child system, is refused.
	for _, sys := range []string{"not.a.system", "mechanical.air-conditioning"} {
		if _, err := s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version, SystemID: &sys}}); !errors.Is(err, store.ErrInvalidDelivery) {
			t.Fatalf("system %s: %v", sys, err)
		}
	}
	// The system comes from the work item, at the top level.
	w, err := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if err != nil {
		t.Fatal(err)
	}
	in := rfiInput("Plant platform", "")
	in.WorkItemID = w.ID
	plant, err := s.CreateDelivery(ctx, orgA, projectA, userA, in)
	if err != nil || plant.SystemID != "mechanical" {
		t.Fatalf("system from work item %q %v", plant.SystemID, err)
	}
	// A notice cannot name a message until messages exist (plan 1).
	notice := store.DeliveryInput{Content: delivery.Content{Kind: "notice", Title: "Notice of delay", Details: json.RawMessage(`{"message_id":"0b5f1c1e-1111-4111-8111-111111111111"}`)}}
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, notice); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("notice message %v", err)
	}
	// Another org reads nothing.
	if _, err := s.ReadRegister(ctx, orgB, projectA, store.RegisterQuery{}, "2026-12-01"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign read %v", err)
	}
	// Read: late days for the open RFI past its date, the Kind list, the System filter.
	view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-12-17")
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Kinds) != 1 || view.Kinds[0].Kind != "rfi" || view.Kinds[0].Rows != 3 || !view.ResponseTypical || view.Calendar != "weekdays" {
		t.Fatalf("view %+v", view)
	}
	for _, it := range view.Items {
		if it.ID == item.ID && (it.LateDays == nil || *it.LateDays != 2) {
			t.Fatalf("late days %v", it.LateDays)
		}
	}
	mech, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{System: "mechanical"}, "2026-12-17")
	if err != nil || len(mech.Items) != 1 || mech.AlsoOpen == nil || mech.AlsoOpen.System != "mechanical" {
		t.Fatalf("system read %+v %v", mech, err)
	}
	if _, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{System: "nowhere"}, "2026-12-17"); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("unknown system read %v", err)
	}
}

func TestRegisterResponseTime(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	if err := s.SetDeliverySettings(ctx, orgA, projectA, userA, 0); !errors.Is(err, store.ErrInvalidDelivery) {
		t.Fatalf("zero days %v", err)
	}
	if err := s.SetDeliverySettings(ctx, orgB, projectA, userB, 10); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign org %v", err)
	}
	if err := s.SetDeliverySettings(ctx, orgA, projectA, userA, 10); err != nil {
		t.Fatal(err)
	}
	item, err := s.CreateDelivery(ctx, orgA, projectA, userA, rfiInput("Ten days", "2026-11-20"))
	if err != nil || *item.TargetDate != "2026-12-04" {
		t.Fatalf("ten days %v %v", item.TargetDate, err)
	}
}

func addReading(t *testing.T, item, decidedBy, band string) string {
	t.Helper()
	id := newID(t)
	if _, err := rawPool(t).Exec(context.Background(), `INSERT INTO delivery_readings(org_id,id,project_id,item_id,document_id,field,decided_by,band,basis) VALUES($1,$2,$3,$4,$5,'answers',$6,$7,'cites RFI-001')`, orgA, id, projectA, item, docA, decidedBy, band); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRegisterAcceptDismissAndUndo(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	item, err := s.CreateDelivery(ctx, orgA, projectA, userA, rfiInput("Basement wall waterproofing", "2026-11-20"))
	if err != nil {
		t.Fatal(err)
	}
	rule := addReading(t, item.ID, "rule", "amber")
	view, err := s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-21")
	if err != nil || len(view.Items[0].Readings) != 1 || view.Kinds[0].ToCheck != 1 {
		t.Fatalf("reading not shown %+v %v", view.Items, err)
	}
	// A dismissal keeps the row's version.
	dismissed, err := s.PatchDelivery(ctx, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version}, DismissReadings: []string{rule}})
	if err != nil || dismissed.Version != item.Version {
		t.Fatalf("dismiss %d %v", dismissed.Version, err)
	}
	// Accepting is an ordinary edit, captured under one change with Undo.
	if _, err := rawPool(t).Exec(ctx, `UPDATE delivery_readings SET status='proposed' WHERE org_id=$1 AND id=$2`, orgA, rule); err != nil {
		t.Fatal(err)
	}
	change := store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "delivery"}
	marked := store.WithChange(ctx, change)
	accepted, err := s.PatchDelivery(marked, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: item.Version}, AcceptReading: rule})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != "answered" || accepted.ActualDate == nil || len(delivery.RFI(accepted.Content).AnswerDocuments) != 1 {
		t.Fatalf("accepted %+v", accepted)
	}
	view, _ = s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-21")
	if view.Items[0].With != "You" || len(view.Items[0].Readings) != 0 {
		t.Fatalf("after accept %+v", view.Items[0])
	}
	// Two more edits in the same change: one Undo puts all three back.
	title := "Basement wall membrane"
	closed := "closed"
	if accepted, err = s.PatchDelivery(marked, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: accepted.Version, Title: &title}}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.PatchDelivery(marked, orgA, projectA, item.ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: accepted.Version, Status: &closed}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err != nil {
		t.Fatal(err)
	}
	view, _ = s.ReadRegister(ctx, orgA, projectA, store.RegisterQuery{}, "2026-11-21")
	back := view.Items[0]
	if back.Status != "draft" || back.Title != "Basement wall waterproofing" || back.ActualDate != nil || len(back.Readings) != 1 {
		t.Fatalf("undo left %+v", back)
	}
	// Deleting the document deletes its readings.
	if _, err := rawPool(t).Exec(ctx, `DELETE FROM documents WHERE org_id=$1 AND id=$2`, orgA, docA); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := rawPool(t).QueryRow(ctx, `SELECT count(*) FROM delivery_readings WHERE org_id=$1`, orgA).Scan(&n); err != nil || n != 0 {
		t.Fatalf("readings left %d %v", n, err)
	}
}
