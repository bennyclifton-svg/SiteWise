package proposals

import (
	"sort"
	"strings"
	"testing"

	"sitewise/internal/rfp"
)

// With SkipPrices the call holds no fee-row question and no stated-total
// question; without it the call is exactly as before. Figures are invented.
func TestSkipPricesLeavesOutFeeAndTotalQuestions(t *testing.T) {
	items := []rfp.ScheduleItem{
		{Key: "stage:concept", Kind: rfp.ItemStage, Label: "Concept design", Base: true},
		{Key: "option:model", Kind: rfp.ItemOption, Label: "Physical model"},
		{Key: "rate:director", Kind: rfp.ItemRate, Label: "Director"},
	}
	ev := Harvest("Fee.pdf", text(page(1,
		[]string{"Invented Planning Pty Ltd"}, []string{"FEE PROPOSAL"},
		[]string{"Concept design", "$4,250.00"}, []string{"Physical model", "$900.00"},
		[]string{"Total", "$5,150.00"}, []string{"Director", "$350.00 per hour"},
	)))
	pc := Context{PackageID: "p1", Packages: []Package{{ID: "p1", Title: "Town Planner"}}, Schedules: map[string][]rfp.ScheduleItem{"p1": items}}
	plain := NewDraft(ev, pc, nil).Questions()
	pc.SkipPrices = true
	skip := NewDraft(ev, pc, nil).Questions()
	has := func(list []string, field string) bool {
		for _, f := range list {
			if f == field {
				return true
			}
		}
		return false
	}
	if !has(plain, "price:stage:concept") {
		t.Fatalf("the plain call does not ask the fee row: %v", plain)
	}
	var dropped []string
	for _, f := range plain {
		if !has(skip, f) {
			dropped = append(dropped, f)
		}
	}
	for _, f := range skip {
		if !has(plain, f) {
			t.Fatalf("SkipPrices added a question: %s", f)
		}
	}
	sort.Strings(dropped)
	for _, f := range dropped {
		if f != FieldStatedTotal && !strings.HasPrefix(f, "price:stage:") && !strings.HasPrefix(f, "price:option:") {
			t.Fatalf("SkipPrices dropped %s, which the tender pass does not read", f)
		}
	}
	if has(skip, "price:stage:concept") || has(skip, "price:option:model") || has(skip, FieldStatedTotal) {
		t.Fatalf("a price question is still asked: %v", skip)
	}
	if has(plain, "price:rate:director") != has(skip, "price:rate:director") {
		t.Fatalf("the rate question changed: %v vs %v", plain, skip)
	}
}
