package proposals

import (
	"sitewise/internal/rfp"
	"strings"
	"testing"
)

func TestFacingColumnsKeepTheirFeeHeadings(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1,
		[]string{"2.0 SCHEMATIC DESIGN", "", "3.0 TENDER/DESIGN DEVELOPMENT", ""},
		[]string{"Stage Fee", "$25,000", "", ""},
		[]string{"", "", "Stage Fee", "$20,000"},
	)))
	a, _ := find(ev, "$25,000")
	b, _ := find(ev, "$20,000")
	if !strings.Contains(a.Under, "SCHEMATIC") || strings.Contains(a.Under, "TENDER") || !strings.Contains(b.Under, "TENDER") {
		t.Fatalf("mixed facing stages: %q / %q", a.Under, b.Under)
	}
}

func TestFacingInsuranceDoesNotHideStaffRates(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1,
		[]string{"Role/Title", "Rate", "PUBLIC LIABILITY INSURANCE"},
		[]string{"Director", "$330.00", "Insurer: Example"},
		[]string{"Associate", "$215.00", "Amount: $20,000,000"},
	)))
	for _, a := range ev.Amounts {
		if !a.PerHour || a.Cents > 33000 {
			t.Fatalf("insurance or fee instead of rate: %+v", a)
		}
	}
	if len(ev.Amounts) != 2 {
		t.Fatalf("lost rates: %+v", ev.Amounts)
	}
}

func TestGSTHeaderSurvivesProseInFacingColumn(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1,
		[]string{"This unrelated scope sentence is longer than forty characters", "Fee", "GST amount", "Fee (inc GST)"},
		[]string{"Stage 1", "$25,500", "$2,550", "$28,050"},
	)))
	a, _ := find(ev, "$2,550")
	if !a.GSTPart {
		t.Fatalf("GST offered as a fee: %+v", a)
	}
}

func TestIncompleteScanCannotDeclareMissingFeeConfirmed(t *testing.T) {
	txt := text(page(1, []string{"Hourly rates"}, []string{"Director", "$350 per hour"}))
	txt.OCR, txt.Incomplete = true, true
	items := []rfp.ScheduleItem{{Key: "stage:schematic", Kind: rfp.ItemStage, Base: true}}
	d := NewDraft(Harvest("fee.pdf", txt), Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": items}}, nil)
	d.by[PricePrefix+items[0].Key] = Decision{Value: choiceNotPrice, Band: BandGreen, DecidedBy: ByJev}
	got := d.Finish()
	if got.Prices[0].State != "" || got.Prices[0].Band != BandBlank {
		t.Fatalf("unread fee confirmed: %+v", got.Prices[0])
	}
}

func TestHarvestSpaceGroupedFeesAndGSTColumns(t *testing.T) {
	ev := Harvest("proposal.pdf", text(page(1,
		[]string{"Stage", "Fee", "GST amount", "Fee (inc GST)"},
		[]string{"Stage 1", "$ 25 500.00", "$ 2 550.00", "$ 28 050.00"},
		[]string{"Stage 2", "$ 20 000.00", "$ 2 000.00", "$ 22 000.00"},
		[]string{"Total fees", "$ 45 500.00", "$ 4 550.00", "$ 50 050.00"},
	)))
	for _, tc := range []struct {
		display        string
		cents          int64
		gst, inclusive bool
	}{
		{"$ 25 500.00", 2550000, false, false}, {"$ 2 550.00", 255000, true, false}, {"$ 28 050.00", 2805000, false, true},
		{"$ 20 000.00", 2000000, false, false}, {"$ 45 500.00", 4550000, false, false},
	} {
		a, ok := find(ev, tc.display)
		if !ok || a.Cents != tc.cents || a.GSTPart != tc.gst || a.IncTwin != tc.inclusive {
			t.Errorf("%s: %+v, found %v", tc.display, a, ok)
		}
	}
}

func TestUnpricedBaseHasNoCompleteTotal(t *testing.T) {
	items := []rfp.ScheduleItem{{Key: "stage:design", Kind: rfp.ItemStage, Base: true}}
	got := Totals(items, map[string]Cell{"stage:design": {State: StateNotPriced}}, nil, nil)
	if got.BaseTotal != nil || got.NotPriced != 1 {
		t.Fatalf("missing fees became a zero quote: %+v", got)
	}
}

func TestSharedQuotedFeeIsCountedOnce(t *testing.T) {
	ev := Harvest("fee.pdf", text(page(1, []string{"Schematic design and planning approval", "$22,000"})))
	items := []rfp.ScheduleItem{{Key: "stage:schematic", Label: "Schematic design", Kind: rfp.ItemStage, Base: true}, {Key: "stage:planning", Label: "Planning approval", Kind: rfp.ItemStage, Base: true}}
	d := NewDraft(ev, Context{PackageID: "p", Schedules: map[string][]rfp.ScheduleItem{"p": items}}, nil)
	for _, it := range items {
		d.by[PricePrefix+it.Key] = Decision{Field: PricePrefix + it.Key, Value: "a1", Band: BandGreen, DecidedBy: ByJev}
	}
	got := d.Finish()
	var total int64
	for _, p := range got.Prices {
		if p.Cents != nil {
			total += *p.Cents
		}
	}
	if total != 2200000 {
		t.Fatalf("bundled fee counted more than once: %d", total)
	}
	if got.Prices[1].State != "" || got.Prices[1].Basis == "" || got.Prices[1].Band != BandBlank {
		t.Fatalf("shared fee needs an explicit review link: %+v", got.Prices)
	}
}
