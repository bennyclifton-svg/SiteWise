package reports

import (
	"sitewise/internal/delivery"
	"testing"
)

func TestDeliveryComparisonUsesSavedDatesAndProtectsWording(t *testing.T) {
	target, forecast, actual := "2026-10-08", "2026-10-10", "2026-10-07"
	d := delivery.Item{Kind: "milestone", TargetDate: &target, ForecastDate: &forecast}
	table := DeliveryComparison(d)
	if table.Rows[0][2] != "2 days later" {
		t.Fatal(table)
	}
	d.ActualDate = &actual
	if got := DeliveryComparison(d); got.Rows[0][1] != actual || got.Rows[0][2] != "1 day earlier" {
		t.Fatal(got)
	}
	d.TargetDate = nil
	if got := DeliveryComparison(d); got.Rows[0][0] != "Unknown" || got.Rows[0][2] != "Unknown" {
		t.Fatal(got)
	}
	b := Block{ID: "date", Text: "Original wording", Table: table}
	hash, err := ContentHash(b)
	if err != nil {
		t.Fatal(err)
	}
	sections, err := ApplyEdits([]Section{{Blocks: []Block{b}}}, []Edit{{TargetID: "date", Text: "Protected wording", BaseContentSHA256: hash}})
	if err != nil {
		t.Fatal(err)
	}
	got := sections[0].Blocks[0]
	if !got.Edited || got.Text != "Protected wording" || got.Table.Rows[0][2] != "2 days later" {
		t.Fatal(got)
	}
	b.Table = &Table{Columns: table.Columns, Rows: [][]string{{target, actual, "1 day earlier"}}}
	changed, _ := ContentHash(b)
	if changed == hash {
		t.Fatal("table change missing from protected-edit conflict hash")
	}
}
