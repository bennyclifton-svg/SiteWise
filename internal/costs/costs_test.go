package costs

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func money(s string) *Money { m := Money(s); return &m }
func val(s string) Value {
	return Value{Amount: money(s), ValueState: "known", Origin: "user", Meaning: "allowance"}
}
func TestRoundingAndTax(t *testing.T) {
	for in, want := range map[string]string{"0.005": "0.01", "-0.005": "-0.01", "1.2349": "1.23", "-1.235": "-1.24", "9999999999999999.99": "9999999999999999.99"} {
		r, e := Parse(in)
		if e != nil || string(Round(r)) != want {
			t.Fatal(in, e, Round(r))
		}
	}
	m, e := Multiply("3.333", "0.3350")
	if e != nil || m != "1.12" {
		t.Fatal(m, e)
	}
	net, tax, e := Tax("110.00", "inc_tax", "0.1000")
	if e != nil || net != "100.00" || tax != "10.00" {
		t.Fatal(net, tax, e)
	}
	if _, e = Normalize("99999999999999999"); e == nil {
		t.Fatal("overflow accepted")
	}
}
func TestMoneyJSONExact(t *testing.T) {
	var m Money
	for _, s := range []string{`"9007199254740993.01"`, `9007199254740993.01`} {
		if e := json.Unmarshal([]byte(s), &m); e != nil || m != "9007199254740993.01" {
			t.Fatal(m, e)
		}
	}
	for _, s := range []string{`1e8`, `"NaN"`, `"1/3"`} {
		if json.Unmarshal([]byte(s), &m) == nil {
			t.Fatal(s)
		}
	}
}
func TestForecastNullRangeAndExclusion(t *testing.T) {
	p := Plan{Items: []Item{{Content: Content{Posting: true, LineKind: "works"}, Values: map[string]Value{"budget": val("100.00"), "estimate": val("120.00"), "commitment": val("90.00")}}}}
	out, e := Summarize(p, "overall")
	if e != nil || *out.Overall["forecast"].Amount != "90.00" || *out.Variance != "-10.00" {
		t.Fatal(out, e)
	}
	p.Items = append(p.Items, Item{Content: Content{Posting: true, LineKind: "works"}, Values: map[string]Value{}})
	out, _ = Summarize(p, "overall")
	if out.Overall["forecast"].Amount != nil || out.Overall["forecast"].Complete || out.Overall["forecast"].KnownSubtotal != "90.00" {
		t.Fatal(out)
	}
	p.Items[1].Excluded = true
	out, _ = Summarize(p, "overall")
	if !out.Overall["forecast"].Complete {
		t.Fatal(out)
	}
	p.Items[0].Values["commitment"] = Value{ValueState: "known", Low: money("80.00"), High: money("95.00")}
	out, _ = Summarize(p, "overall")
	v := out.Overall["forecast"]
	if v.Amount != nil || *v.Low != "80.00" || *v.High != "95.00" {
		t.Fatal(v)
	}
}
func TestReconciliationRandomized(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	for trial := 0; trial < 100; trial++ {
		p := Plan{}
		for n := 0; n < 100; n++ {
			m := RoundFromCents(r.Int63n(100000) - 10000)
			p.Items = append(p.Items, Item{PartID: string(rune('a' + r.Intn(7))), Content: Content{SystemID: string(rune('a' + r.Intn(5))), Posting: true, LineKind: "works"}, Values: map[string]Value{"budget": val(string(m))}})
		}
		for _, by := range []string{"system", "part", "package"} {
			tot, e := Summarize(p, by)
			if e != nil {
				t.Fatal(e)
			}
			sum := Money("0.00")
			for _, g := range tot.Groups {
				sum = Add(sum, *g["budget"].Amount)
			}
			if sum != *tot.Works["budget"].Amount {
				t.Fatal(by, sum, tot)
			}
		}
	}
}
func RoundFromCents(n int64) Money { r, _ := Parse("0"); r.SetFrac64(n, 100); return Round(r) }
func TestResidualAndValidation(t *testing.T) {
	v, e := Residual(val("100.00"), []Value{val("30.00"), val("20.00")})
	if e != nil || *v.Amount != "50.00" {
		t.Fatal(v, e)
	}
	if _, e = Residual(val("1.00"), []Value{val("2.00")}); e == nil {
		t.Fatal("overspent")
	}
	v, e = Residual(val("100.00"), []Value{{ValueState: "unknown"}})
	if e != nil || v.Amount != nil || v.ValueState != "unknown" {
		t.Fatal(v, e)
	}
	v = val("0.00")
	if ValidateValue("claimed_to_date", &v) == nil {
		t.Fatal("missing date")
	}
	v = Value{ValueState: "unknown", Amount: money("0.00"), Origin: "user", Meaning: "stated"}
	if ValidateValue("budget", &v) == nil {
		t.Fatal("unknown zero")
	}
}
func TestMoneyCoreHasNoBinaryFloats(t *testing.T) {
	files, e := filepath.Glob("*.go")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		b, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if strings.Contains(string(b), "float64") || strings.Contains(string(b), "float32") {
			t.Fatal(file)
		}
	}
}
func TestEstimateIsNotWritable(t *testing.T) {
	v := val("10.00")
	if ValidateValue("estimate", &v) == nil {
		t.Fatal("estimate accepted")
	}
	if ValidateValue("budget", &v) != nil {
		t.Fatal("budget rejected")
	}
}
func TestGridFigures(t *testing.T) {
	claimed := val("40.00")
	date := "2026-10-01"
	claimed.AsOf = &date
	ranged := Value{Low: money("80.00"), High: money("120.00"), ValueState: "known", Origin: "user", Meaning: "allowance"}
	p := Plan{Items: []Item{
		{ID: "contract", Content: Content{Posting: true, LineKind: "works"}, Values: map[string]Value{"budget": val("100.00"), "commitment": val("90.00"), "claimed_to_date": claimed}},
		{ID: "budget-only", Content: Content{Posting: true, LineKind: "fee"}, Values: map[string]Value{"budget": val("100.00")}},
		{ID: "range", Content: Content{Posting: true, LineKind: "project_wide"}, Values: map[string]Value{"budget": ranged}},
		{ID: "excluded", Content: Content{Posting: true, LineKind: "works", Excluded: true}, Values: map[string]Value{"budget": val("500.00")}},
		{ID: "group", Content: Content{Posting: false, LineKind: "works"}},
	}}
	out, e := Summarize(p, "overall")
	if e != nil {
		t.Fatal(e)
	}
	is := func(m *Money, want string) bool { return (m == nil && want == "") || (m != nil && string(*m) == want) }
	f := out.Items["contract"]
	if f.Basis != "commitment" || !is(f.Forecast.Amount, "90.00") || !is(f.BudgetVariance, "10.00") || !is(f.Remaining, "50.00") {
		t.Fatal("contract", f)
	}
	f = out.Items["budget-only"]
	if f.Basis != "budget" || !is(f.BudgetVariance, "0.00") || !is(f.Remaining, "") {
		t.Fatal("budget only; an unclaimed line has unknown remaining", f)
	}
	if f = out.Items["range"]; !is(f.BudgetVariance, "") {
		t.Fatal("range gave a single variance", f)
	}
	if f, ok := out.Items["excluded"]; !ok || !is(f.BudgetVariance, "0.00") {
		t.Fatal("excluded line lost its own figures", f)
	}
	if _, ok := out.Items["group"]; ok {
		t.Fatal("group line has figures; its children hold them")
	}
	if b := out.Buckets["works"]; !is(b.BudgetVariance, "10.00") || !is(b.Remaining, "50.00") {
		t.Fatal("works bucket counted the excluded line", b)
	}
	if b := out.Buckets["overall"]; !is(b.BudgetVariance, "") {
		t.Fatal("overall variance with a range line must stay unknown", b)
	}
}
