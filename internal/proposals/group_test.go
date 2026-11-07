package proposals

import (
	"context"
	"testing"

	"sitewise/internal/rfp"
)

// Combined engagements, slice 3: a proposal for a group is read onto the
// group's schedule. Code places each amount Jev matched to a stage: under a
// discipline only when the bid names it, as a combined cell when the bid
// gives one amount for the stage, otherwise unplaced. Nothing is split.

func memberItems() []rfp.ScheduleItem {
	return []rfp.ScheduleItem{
		{Key: "stage:concept", Kind: rfp.ItemStage, Label: "Concept design", Base: true},
		{Key: "stage:documentation", Kind: rfp.ItemStage, Label: "Documentation", Base: true},
		{Key: "stage:construction", Kind: rfp.ItemStage, Label: "Construction phase", Base: true},
		{Key: "option:flood", Kind: rfp.ItemOption, Label: "Flood study", Base: false},
		{Key: rfp.ReimbursablesKey, Kind: rfp.ItemReimbursable, Label: "Reimbursables", Base: true},
		{Key: "rate:director", Kind: rfp.ItemRate, Label: "Director"},
		{Key: "criterion:r2", Kind: rfp.ItemCriterion, Label: "Experience"},
	}
}

func twoMembers() GroupLayout {
	return GroupLayout{Members: []GroupMember{
		{Key: "structural", Title: "Structural engineering", Words: []string{"structural"}},
		{Key: "civil", Title: "Civil engineering", Words: []string{"civil"}},
	}}
}

func TestGroupScheduleIsEachMemberThenCombinedStages(t *testing.T) {
	layout := twoMembers()
	items := GroupSchedule(layout, [][]rfp.ScheduleItem{memberItems(), memberItems()[:2]})
	keys := []string{}
	for _, it := range items {
		keys = append(keys, it.Key)
	}
	want := []string{
		"structural:stage:concept", "structural:stage:documentation", "structural:stage:construction", "structural:option:flood", "structural:reimbursables",
		"civil:stage:concept", "civil:stage:documentation",
		"combined:stage:concept", "combined:stage:documentation",
		"criterion:r2",
	}
	if len(keys) != len(want) {
		t.Fatalf("keys %v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys %v, want %v", keys, want)
		}
	}
	if items[0].Label != "Structural engineering: Concept design" || items[7].Label != "Combined: Concept design" || !items[7].Base {
		t.Fatalf("labels %+v %+v", items[0], items[7])
	}
	for i, it := range items {
		if it.Ordinal != i {
			t.Fatalf("ordinal %d at %d", it.Ordinal, i)
		}
	}
}

func groupContext(layout GroupLayout) Context {
	schedule := GroupSchedule(layout, [][]rfp.ScheduleItem{memberItems(), memberItems()})
	asked := []rfp.ScheduleItem{}
	for _, it := range schedule {
		if _, _, combined := SplitItemKey(it.Key); !combined {
			asked = append(asked, it)
		}
	}
	return Context{
		Packages:  []Package{{ID: "g", Title: "Structural and civil engineering", Disciplines: []string{"consultant.structural", "consultant.civil"}}},
		PackageID: "g",
		Schedules: map[string][]rfp.ScheduleItem{"g": asked},
		Groups:    map[string]GroupLayout{"g": layout.WithCombined(schedule)},
	}
}

func priceOf(r Reading, key string) (Price, bool) {
	for _, p := range r.Prices {
		if p.Key == key {
			return p, true
		}
	}
	return Price{}, false
}

func TestGroupPlacementNamesDisciplinesCombinesStagesAndNeverSplits(t *testing.T) {
	tx := text(page(1,
		[]string{"Karrak Consulting Engineers Pty Ltd"},
		[]string{"FEE PROPOSAL"},
		[]string{"Structural concept design", "$5,000"},
		[]string{"Civil concept design", "$3,000"},
		[]string{"Documentation for both disciplines", "$8,000"},
		[]string{"Civil inspections", "$1,000"},
		[]string{"Reimbursables allowance", "$600"},
		[]string{"Printing allowance", "$400"},
		[]string{"Total (excl. GST)", "$18,000"},
	))
	ev := Harvest("fee.pdf", tx)
	id := func(display string) string {
		a, ok := find(ev, display)
		if !ok {
			t.Fatalf("no amount %s", display)
		}
		return a.ID
	}
	answers := map[string]string{
		"price:structural:stage:concept":       id("$5,000"),
		"price:civil:stage:concept":            id("$3,000"),
		"price:structural:stage:documentation": id("$8,000"),
		"price:civil:stage:documentation":      id("$8,000"),
		// A line naming civil is not offered under structural; an answer
		// placing it there anyway is not moved to civil by guesswork.
		"price:structural:stage:construction": id("$1,000"),
		"price:civil:stage:construction":      choiceNotPrice,
		// Two different unnamed amounts for one stage: neither is placed.
		"price:structural:reimbursables": id("$600"),
		"price:civil:reimbursables":      id("$400"),
		"gst_basis":                      "ex",
	}
	ask := &fakeJev{answers: answers, conf: 0.99}
	out, err := Read(context.Background(), ask, greenAt(0.9), "fee.pdf", tx, groupContext(twoMembers()), nil)
	if err != nil {
		t.Fatal(err)
	}
	r := out.Reading
	check := func(key, state string, cents int64) {
		t.Helper()
		p, ok := priceOf(r, key)
		if !ok || p.State != state || (cents >= 0 && (p.Cents == nil || *p.Cents != cents)) || (cents < 0 && p.Cents != nil) {
			t.Fatalf("%s: %+v (cents %v)", key, p, p.Cents)
		}
	}
	check("structural:stage:concept", StatePriced, 500000)
	check("civil:stage:concept", StatePriced, 300000)
	check("combined:stage:documentation", StatePriced, 800000)
	check("structural:stage:documentation", StateIncluded, -1)
	check("civil:stage:documentation", StateIncluded, -1)
	check("structural:stage:construction", "", -1)
	check("civil:stage:construction", StateNotPriced, -1)
	check("structural:reimbursables", "", -1)
	check("civil:reimbursables", "", -1)
	if _, ok := priceOf(r, "combined:stage:concept"); ok {
		t.Fatal("a stage priced by discipline made a combined cell")
	}
	if _, ok := priceOf(r, "combined:reimbursables"); ok {
		t.Fatal("two amounts for one stage were combined")
	}
	// Unplaced amounts stay visible, whole, for a cell correction.
	others := map[int64]bool{}
	for _, p := range r.Prices {
		if p.Key == "" && p.Cents != nil {
			others[*p.Cents] = true
		}
	}
	for _, c := range []int64{100000, 60000, 40000} {
		if !others[c] {
			t.Fatalf("unplaced %d missing from %+v", c, r.Prices)
		}
	}
	// Each figure is counted once: the combined cell, not its members.
	cells := map[string]Cell{}
	for _, p := range r.Prices {
		if p.Key != "" {
			cells[p.Key] = Cell{Key: p.Key, Cents: p.Cents, State: p.State}
		}
	}
	schedule := GroupSchedule(twoMembers(), [][]rfp.ScheduleItem{memberItems(), memberItems()})
	col := GroupTotals(schedule, cells, nil, nil)
	if col.KnownBase != 1600000 || col.BaseTotal != nil {
		t.Fatalf("totals %+v", col)
	}
}

func TestGroupPlacementFromAReturnedFormRecordsTheCombinedStage(t *testing.T) {
	layout := twoMembers()
	pc := groupContext(layout)
	ev := Evidence{Form: map[string]string{
		"structural:stage:concept": "5000", "civil:stage:concept": "Included in structural concept",
		"structural:stage:documentation": "4000", "civil:stage:documentation": "2500",
	}, Amounts: []Amount{{ID: "a1", Cents: 1}}}
	d := NewDraft(ev, pc, nil)
	r := d.Finish()
	for key, want := range map[string]string{"combined:stage:concept": StatePriced, "structural:stage:concept": StateIncluded, "civil:stage:concept": StateIncluded, "structural:stage:documentation": StatePriced, "civil:stage:documentation": StatePriced} {
		if p, ok := priceOf(r, key); !ok || p.State != want {
			t.Fatalf("%s %+v", key, p)
		}
	}
	if p, _ := priceOf(r, "combined:stage:concept"); p.Cents == nil || *p.Cents != 500000 {
		t.Fatalf("combined %+v", p)
	}
}

func TestGroupTotalsCountACombinedStageOnceAndIgnoreBlankCombinedCells(t *testing.T) {
	schedule := GroupSchedule(twoMembers(), [][]rfp.ScheduleItem{memberItems()[:2], memberItems()[:2]})
	c := func(state string, cents int64) Cell {
		if state != StatePriced {
			return Cell{State: state}
		}
		return Cell{State: state, Cents: &cents}
	}
	// By discipline: every member cell priced, combined cells blank.
	byDiscipline := map[string]Cell{"structural:stage:concept": c(StatePriced, 100), "structural:stage:documentation": c(StatePriced, 200), "civil:stage:concept": c(StatePriced, 300), "civil:stage:documentation": c(StatePriced, 400)}
	if col := GroupTotals(schedule, byDiscipline, nil, nil); col.BaseTotal == nil || *col.BaseTotal != 1000 {
		t.Fatalf("by discipline %+v", col)
	}
	// One lump per stage: members blank, combined priced.
	combined := map[string]Cell{"combined:stage:concept": c(StatePriced, 500), "combined:stage:documentation": c(StatePriced, 700)}
	if col := GroupTotals(schedule, combined, nil, nil); col.BaseTotal == nil || *col.BaseTotal != 1200 {
		t.Fatalf("combined %+v", col)
	}
	// A member left blank with no combined cell is unknown, not zero.
	partial := map[string]Cell{"structural:stage:concept": c(StatePriced, 100), "structural:stage:documentation": c(StatePriced, 200)}
	if col := GroupTotals(schedule, partial, nil, nil); col.BaseTotal != nil || col.Unread != 2 || col.KnownBase != 300 {
		t.Fatalf("partial %+v", col)
	}
}

func TestPackageRuleRoutesAMemberDisciplineToItsGroup(t *testing.T) {
	pc := Context{Discipline: "consultant.civil", Packages: []Package{
		{ID: "arch", Title: "Architect", Discipline: "consultant.architect"},
		{ID: "g", Title: "Structural and civil engineering", Disciplines: []string{"consultant.structural", "consultant.civil"}},
	}}
	d := NewDraft(Evidence{}, pc, nil)
	if got := d.by[FieldPackage]; got.Value != "g" || got.DecidedBy != ByRule {
		t.Fatalf("package %+v", got)
	}
	// A dropped group says whose fee it is: a consultant's.
	pc = Context{PackageID: "g", Packages: pc.Packages}
	if got := kindDiscipline(pc); got != "consultant.structural" {
		t.Fatalf("kind discipline %q", got)
	}
}

func TestGroupPriceQuestionsOfferOnlyPlaceableAmounts(t *testing.T) {
	tx := text(page(1,
		[]string{"Karrak Consulting Engineers Pty Ltd"},
		[]string{"FEE PROPOSAL"},
		[]string{"Structural concept design", "$5,000"},
		[]string{"Civil concept design", "$3,000"},
		[]string{"Documentation", "$8,000"},
	))
	ev := Harvest("fee.pdf", tx)
	d := NewDraft(ev, groupContext(twoMembers()), nil)
	q, ok := d.questions["price:structural:stage:concept"]
	if !ok {
		t.Fatal("no price question")
	}
	offered := map[string]bool{}
	for _, o := range q.Options {
		if a, ok := d.Amount(o.ID); ok {
			offered[a.Display] = true
		}
	}
	if !offered["$5,000"] || !offered["$8,000"] || offered["$3,000"] {
		t.Fatalf("offered %v", offered)
	}
}
