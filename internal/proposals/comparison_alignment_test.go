package proposals

import (
	"sitewise/internal/rfp"
	"testing"
)

func TestExcludedBaseFeeCannotBeLowestComplete(t *testing.T) {
	items := []rfp.ScheduleItem{{Key: "assessment", Kind: rfp.ItemStage, Base: true}}
	c := Totals(items, map[string]Cell{"assessment": {State: StateExcluded}}, nil, nil)
	if c.BaseTotal != nil || c.NotPriced != 1 {
		t.Fatalf("excluded base scope was complete: %+v", c)
	}
}

func TestNoBaseRowsIsNotAZeroFee(t *testing.T) {
	c := Totals([]rfp.ScheduleItem{{Key: "optional", Kind: rfp.ItemOption}}, nil, nil, nil)
	if c.BaseTotal != nil {
		t.Fatal("no base scope was shown as a zero complete fee")
	}
}
