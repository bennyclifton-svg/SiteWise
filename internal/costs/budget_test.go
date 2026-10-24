package costs

import (
	"math/big"
	"testing"
)

func TestParseStatedBudget(t *testing.T) {
	cases := []struct {
		text, amount, gst string
		ok                bool
	}{
		{"$1,850,000", "1850000.00", "not_stated", true},
		{"$1.8 million excluding GST", "1800000.00", "ex", true},
		{"$1,250,000 (incl GST)", "1136363.64", "inc", true},
		{"$5 million excluding GST", "5000000.00", "ex", true},
		{"approx. $2.5M plus GST", "2500000.00", "ex", true},
		{"$850k", "850000.00", "not_stated", true},
		{"$1.5–1.8 million", "", "not_stated", false},
		{"$1.5 to $1.8 million", "", "not_stated", false},
		{"$1,250,000 with a fee allowance of $90,000", "", "not_stated", false},
		{"to be confirmed", "", "not_stated", false},
		{"$0", "", "not_stated", false},
	}
	for _, c := range cases {
		got, ok := ParseStatedBudget(c.text)
		if ok != c.ok || ok && (string(got.Amount) != c.amount || got.GST != c.gst) {
			t.Errorf("%q: got %+v %v, want %s %s %v", c.text, got, ok, c.amount, c.gst, c.ok)
		}
	}
}

// The parts always add to the total, the residual cent lands on the
// heaviest line, and zero-weight lines get nothing.
func TestSpreadAddsToTheTotal(t *testing.T) {
	lines := []SpreadLine{{"a", big.NewRat(1, 3)}, {"b", big.NewRat(1, 3)}, {"c", big.NewRat(1, 3)}, {"z", new(big.Rat)}}
	got := Spread("100.00", lines)
	sum := Money("0.00")
	for _, m := range got {
		sum = Add(sum, m)
	}
	if sum != "100.00" || got["a"] != "33.34" || got["b"] != "33.33" || got["z"] != "0.00" {
		t.Fatalf("spread %v sums to %s", got, sum)
	}
	if Spread("100.00", []SpreadLine{{"z", new(big.Rat)}}) != nil {
		t.Fatal("no weight, no spread")
	}
	house := []SpreadLine{{"site", big.NewRat(5, 1)}, {"substructure", big.NewRat(11, 1)}, {"envelope", big.NewRat(24, 1)}, {"interiors", big.NewRat(24, 1)}}
	got = Spread("1850000.00", house)
	if got["envelope"] != "693750.00" || got["site"] != "144531.25" {
		t.Fatalf("house spread %v", got)
	}
}
