package httpapi

import (
	"sitewise/internal/proposals"
	"sitewise/internal/store"
	"strings"
	"testing"
)

func TestBuilderExportSeparatesQuotedTotalAndDetails(t *testing.T) {
	review := proposals.ReviewBuilderQuote([]proposals.Passage{{Body: "Kitchen allowance $20,000.00\nTotal including GST $110,000.00", Page: 1}})
	sheet := priceSheet(store.PriceTable{Builder: true, Columns: []store.FirmColumn{{Builder: &review}}})
	quoted, detail := false, false
	for _, row := range sheet.Rows {
		if strings.HasPrefix(row[0].Text, "Quoted total ex GST") {
			quoted = row[1].Cents != nil && *row[1].Cents == 10000000
		}
		if strings.Contains(row[0].Text, "Interiors") {
			detail = strings.Contains(row[1].Text, "Allowance; inclusion unresolved") && row[1].Cents == nil
		}
		if row[0].Text == "Other items offered" {
			t.Fatal("builder detail exported as extras")
		}
	}
	if !quoted || !detail {
		t.Fatalf("missing quoted total or source detail: %+v", sheet)
	}
}
