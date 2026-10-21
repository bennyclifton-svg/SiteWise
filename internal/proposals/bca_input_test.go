package proposals

import "testing"

func TestCommercialLimitsAreNotOfferedFees(t *testing.T) {
	ev := Harvest("Fee.pdf", text(page(1,
		[]string{"Fee proposal"},
		[]string{"Assessment", "$4,250.00"},
		[]string{"Policy Value:", "$2,000,000.00"},
		[]string{"A penalty of $110,000 and/or two years imprisonment."},
		[]string{"If no invoice has been issued by the Company, $10,000."},
	)))
	if len(ev.Amounts) != 1 || ev.Amounts[0].Cents != 425000 {
		t.Fatalf("non-fee limits offered as prices: %+v", ev.Amounts)
	}
}
