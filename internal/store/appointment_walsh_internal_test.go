package store

import (
	"testing"

	"sitewise/internal/proposals"
	"sitewise/internal/rfp"
)

// Walsh engagement letter, 2026-10-27: each fee line maps to the RFP stage
// that names the same stage in other words, not to a partial guess.
func TestWalshFeeLinesMapToTheirStages(t *testing.T) {
	items := []rfp.ScheduleItem{
		{Key: "stage:concept", Kind: rfp.ItemStage, Label: "Concept"},
		{Key: "stage:design_development", Kind: rfp.ItemStage, Label: "Design development"},
		{Key: "stage:tender_traditional", Kind: rfp.ItemStage, Label: "Tender documents"},
		{Key: "stage:for_construction", Kind: rfp.ItemStage, Label: "For-construction documents"},
		{Key: "stage:construction_support", Kind: rfp.ItemStage, Label: "Design support during construction"},
		{Key: "stage:contract_admin", Kind: rfp.ItemStage, Label: "Contract administration"},
	}
	fees := []proposals.QuotedFee{
		{ID: "a3", Label: "Mobilisation & concept", Source: "| Mobilisation & concept | Engagement signed | $14,500 |"},
		{ID: "a5", Label: "Design development", Source: "| Design development | DD issue to builder tender | $22,000 |"},
		{ID: "a6", Label: "Construction documentation", Source: "| Construction documentation | IFC issue | $26,000 |"},
		{ID: "a8", Label: "Construction administration", Source: "| Construction administration | Monthly × est. 9 months | $3,500 |"},
	}
	got := labelMappings(items, fees, proposals.Offer{})
	want := map[string]string{"a3": "stage:concept", "a5": "stage:design_development", "a6": "stage:for_construction", "a8": "stage:contract_admin"}
	for fee, key := range want {
		if got[fee] != key {
			t.Errorf("%s mapped to %q, want %q (all: %v)", fee, got[fee], key, got)
		}
	}
}
