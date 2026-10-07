package costs

// Total retains the known subtotal while marking missing values. Null does not
// silently become zero; range-only lines retain their lower and upper bounds.
type Total struct {
	Amount        *Money         `json:"amount"`
	KnownSubtotal Money          `json:"known_subtotal"`
	Low           *Money         `json:"low"`
	High          *Money         `json:"high"`
	Lines         int            `json:"lines"`
	Unknown       int            `json:"unknown"`
	Complete      bool           `json:"complete"`
	Basis         map[string]int `json:"basis,omitempty"`
}
type Totals struct {
	By          string                      `json:"by"`
	Currency    string                      `json:"currency"`
	TaxBasis    string                      `json:"tax_basis"`
	Groups      map[string]map[string]Total `json:"groups"`
	Overall     map[string]Total            `json:"overall"`
	Works       map[string]Total            `json:"works"`
	Fees        map[string]Total            `json:"fees"`
	ProjectWide map[string]Total            `json:"project_wide"`
	Variance    *Money                      `json:"variance"`
}

func Forecast(item Item) (Value, string) {
	for _, m := range []string{"commitment", "estimate", "budget"} {
		v := item.Values[m]
		if v.ValueState == "known" && (v.Amount != nil || v.Low != nil) {
			return v, m
		}
	}
	return Value{ValueState: "unknown"}, "unknown"
}
func emptyTotals() map[string]Total {
	out := map[string]Total{}
	for _, m := range append(append([]string{}, Metrics...), "forecast") {
		zero := Money("0.00")
		out[m] = Total{Amount: &zero, KnownSubtotal: zero, Low: &zero, High: &zero, Complete: true, Basis: map[string]int{}}
	}
	return out
}
func addLine(out map[string]Total, item Item) {
	for m, t := range out {
		v := item.Values[m]
		basis := m
		if m == "forecast" {
			v, basis = Forecast(item)
			t.Basis[basis]++
		}
		t.Lines++
		if v.ValueState != "known" || (v.Amount == nil && v.Low == nil) {
			t.Unknown++
			t.Complete = false
			t.Amount = nil
			t.Low = nil
			t.High = nil
			out[m] = t
			continue
		}
		if v.Amount != nil {
			t.KnownSubtotal = Add(t.KnownSubtotal, *v.Amount)
		} else {
			t.Amount = nil
		}
		if t.Amount != nil {
			n := t.KnownSubtotal
			t.Amount = &n
		}
		lo, hi := v.Low, v.High
		if lo == nil {
			lo, hi = v.Amount, v.Amount
		}
		if t.Low != nil && lo != nil {
			n := Add(*t.Low, *lo)
			t.Low = &n
		}
		if t.High != nil && hi != nil {
			n := Add(*t.High, *hi)
			t.High = &n
		}
		out[m] = t
	}
}
func Summarize(plan Plan, by string) (Totals, error) {
	switch by {
	case "overall", "system", "part", "package":
	default:
		return Totals{}, ErrInvalid
	}
	out := Totals{By: by, Currency: plan.Currency, TaxBasis: plan.TaxBasis, Groups: map[string]map[string]Total{}, Overall: emptyTotals(), Works: emptyTotals(), Fees: emptyTotals(), ProjectWide: emptyTotals()}
	for _, item := range plan.Items {
		if !item.Posting || item.Excluded {
			continue
		}
		addLine(out.Overall, item)
		switch item.LineKind {
		case "works":
			addLine(out.Works, item)
		case "fee":
			addLine(out.Fees, item)
		case "project_wide":
			addLine(out.ProjectWide, item)
		}
		key := "overall"
		switch by {
		case "system":
			if item.LineKind != "works" {
				continue
			}
			key = item.SystemID
		case "part":
			if item.LineKind != "works" {
				continue
			}
			key = item.PartID
		case "package":
			key = item.PackageID
		}
		if key == "" {
			key = "unallocated"
		}
		if out.Groups[key] == nil {
			out.Groups[key] = emptyTotals()
		}
		addLine(out.Groups[key], item)
	}
	for _, bucket := range []map[string]Total{out.Overall, out.Works, out.Fees, out.ProjectWide} {
		for metric, total := range bucket {
			if total.Lines == 0 {
				total.Amount = nil
				total.Low = nil
				total.High = nil
				total.Complete = false
				bucket[metric] = total
			}
		}
	}
	if b, f := out.Overall["budget"].Amount, out.Overall["forecast"].Amount; b != nil && f != nil {
		v := Sub(*f, *b)
		out.Variance = &v
	}
	return out, nil
}

// Residual preserves every metric independently. An unpriced child produces an
// explicit unknown residual instead of inventing a zero allocation.
func Residual(parent Value, children []Value) (Value, error) {
	out := parent
	out.Version = 0
	out.Origin = "user"
	out.Meaning = "allowance"
	if parent.Amount == nil {
		return Value{ValueState: "unknown", Origin: "user", Meaning: "allowance", AsOf: parent.AsOf}, nil
	}
	sum := Money("0.00")
	for _, v := range children {
		if v.Amount == nil {
			return Value{ValueState: "unknown", Origin: "user", Meaning: "allowance", AsOf: parent.AsOf}, nil
		}
		sum = Add(sum, *v.Amount)
	}
	if Compare(sum, *parent.Amount) > 0 {
		return Value{}, ErrInvalid
	}
	n := Sub(*parent.Amount, sum)
	out.Amount = &n
	out.Low = nil
	out.High = nil
	return out, nil
}
