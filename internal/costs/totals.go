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
	Sections       map[string]map[string]Total `json:"sections"`
	SectionItems   map[string]string           `json:"section_items"`
	SectionFigures map[string]Derived          `json:"section_figures"`
	By             string                      `json:"by"`
	Currency       string                      `json:"currency"`
	TaxBasis       string                      `json:"tax_basis"`
	Groups         map[string]map[string]Total `json:"groups"`
	Overall        map[string]Total            `json:"overall"`
	Works          map[string]Total            `json:"works"`
	Fees           map[string]Total            `json:"fees"`
	ProjectWide    map[string]Total            `json:"project_wide"`
	Variance       *Money                      `json:"variance"`
	Items          map[string]Figures          `json:"items"`
	Buckets        map[string]Derived          `json:"buckets"`
}

// Derived figures are worked out here so the grid never sums money in the
// browser. Each is nil when an input is unknown or only a range.
type Derived struct {
	BudgetVariance *Money `json:"budget_variance"` // budget − forecast; positive is under budget
	Remaining      *Money `json:"remaining"`       // forecast − claimed to date
}
type Figures struct {
	Forecast Value  `json:"forecast"`
	Basis    string `json:"basis"`
	Derived
}

func diff(a, b *Money) *Money {
	if a == nil || b == nil {
		return nil
	}
	n := Sub(*a, *b)
	return &n
}
func exact(v Value) *Money {
	if v.ValueState != "known" {
		return nil
	}
	return v.Amount
}
func lineFigures(item Item) Figures {
	f, basis := Forecast(item)
	return Figures{Forecast: f, Basis: basis, Derived: Derived{
		BudgetVariance: diff(exact(item.Values["budget"]), exact(f)),
		Remaining:      diff(exact(f), exact(item.Values["claimed_to_date"])),
	}}
}

// Forecast keeps estimate in its order so a baseline frozen before estimate was
// retired keeps the forecast it was frozen with. Drafts no longer hold one.
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
	addMetrics(out, item, false)
}

// addMetrics adds a line's figures. A line under a budgeted group adds no
// budget: the group's budget stands for its children (an awarded package).
func addMetrics(out map[string]Total, item Item, noBudget bool) {
	for m, t := range out {
		if noBudget && m == "budget" {
			continue
		}
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
	out := Totals{By: by, Currency: plan.Currency, TaxBasis: plan.TaxBasis, Groups: map[string]map[string]Total{}, Overall: emptyTotals(), Works: emptyTotals(), Fees: emptyTotals(), ProjectWide: emptyTotals(), Items: map[string]Figures{}, Buckets: map[string]Derived{}}
	out.Sections = map[string]map[string]Total{}
	out.SectionFigures = map[string]Derived{}
	out.SectionItems = CostSections(plan.Items)
	for _, section := range []string{"authority", "consultants", "construction", "contingency"} {
		out.Sections[section] = emptyTotals()
	}
	groups := budgetGroups(plan.Items)
	for _, item := range plan.Items {
		if !item.Posting {
			if b, ok := item.Values["budget"]; ok && !item.Excluded {
				// A group that keeps its budget: the budget counts once, here.
				g := Item{Values: map[string]Value{"budget": b}}
				for _, bucket := range []map[string]Total{out.Overall, out.Sections[out.SectionItems[item.ID]], kindBucket(&out, item.LineKind)} {
					if bucket != nil {
						addOnly(bucket, "budget", g)
					}
				}
			}
			continue
		}
		// An excluded line still reads its own figures; it stays out of totals.
		out.Items[item.ID] = lineFigures(item)
		if item.Excluded {
			continue
		}
		under := groups[item.ID] != ""
		addMetrics(out.Overall, item, under)
		addMetrics(out.Sections[out.SectionItems[item.ID]], item, under)
		if b := kindBucket(&out, item.LineKind); b != nil {
			addMetrics(b, item, under)
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
			key = item.ResolvedPackageID
			if key == "" {
				key = item.PackageID
			}
		}
		if key == "" {
			key = "unallocated"
		}
		if out.Groups[key] == nil {
			out.Groups[key] = emptyTotals()
		}
		addMetrics(out.Groups[key], item, under)
	}
	groupFigures(plan.Items, groups, out.Items)
	for key, bucket := range out.Sections {
		for metric, total := range bucket {
			if total.Lines == 0 {
				total.Amount = nil
				total.Low = nil
				total.High = nil
				total.Complete = false
				bucket[metric] = total
			}
		}
		out.SectionFigures[key] = Derived{BudgetVariance: diff(bucket["budget"].Amount, bucket["forecast"].Amount), Remaining: diff(bucket["forecast"].Amount, bucket["claimed_to_date"].Amount)}
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
	for name, bucket := range map[string]map[string]Total{"overall": out.Overall, "works": out.Works, "fees": out.Fees, "project_wide": out.ProjectWide} {
		out.Buckets[name] = Derived{BudgetVariance: diff(bucket["budget"].Amount, bucket["forecast"].Amount), Remaining: diff(bucket["forecast"].Amount, bucket["claimed_to_date"].Amount)}
	}
	if b, f := out.Overall["budget"].Amount, out.Overall["forecast"].Amount; b != nil && f != nil {
		v := Sub(*f, *b)
		out.Variance = &v
	}
	return out, nil
}

func kindBucket(out *Totals, kind string) map[string]Total {
	switch kind {
	case "works":
		return out.Works
	case "fee":
		return out.Fees
	case "project_wide":
		return out.ProjectWide
	}
	return nil
}

// addOnly adds one metric of a line.
func addOnly(out map[string]Total, metric string, item Item) {
	one := map[string]Total{metric: out[metric]}
	addMetrics(one, item, false)
	out[metric] = one[metric]
}

// budgetGroups maps each line under a group that holds a budget to that
// group. An awarded package's line is such a group: it keeps the budget the
// PM set while its children hold the contract.
func budgetGroups(items []Item) map[string]string {
	byID := map[string]Item{}
	for _, i := range items {
		byID[i.ID] = i
	}
	out := map[string]string{}
	for _, i := range items {
		seen := map[string]bool{i.ID: true}
		for p := i.ParentItemID; p != "" && !seen[p]; {
			seen[p] = true
			parent, ok := byID[p]
			if !ok {
				break
			}
			if _, has := parent.Values["budget"]; has && !parent.Posting {
				out[i.ID] = parent.ID
				break
			}
			p = parent.ParentItemID
		}
	}
	return out
}

// groupFigures gives each budgeted group its own figures: the forecast is
// its children's, known only when every child's is; the variance is its
// budget less that forecast.
func groupFigures(items []Item, groups map[string]string, figs map[string]Figures) {
	sum := map[string]Money{}
	unknown := map[string]bool{}
	for _, i := range items {
		g := groups[i.ID]
		if g == "" || !i.Posting || i.Excluded {
			continue
		}
		f := figs[i.ID].Forecast
		if f.ValueState != "known" || f.Amount == nil {
			unknown[g] = true
			continue
		}
		cur, ok := sum[g]
		if !ok {
			cur = "0.00"
		}
		sum[g] = Add(cur, *f.Amount)
	}
	for _, i := range items {
		b, ok := i.Values["budget"]
		if i.Posting || !ok {
			continue
		}
		fig := Figures{Forecast: Value{ValueState: "unknown"}, Basis: "children"}
		if total, ok := sum[i.ID]; ok && !unknown[i.ID] {
			t := total
			fig.Forecast = Value{ValueState: "known", Amount: &t, Origin: "calculation", Meaning: "stated"}
			fig.BudgetVariance = diff(exact(b), &t)
		}
		figs[i.ID] = fig
	}
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

// CostSections keeps descendants in their root's display section, including legacy trees.
func CostSections(items []Item) map[string]string {
	byID := map[string]Item{}
	for _, i := range items {
		byID[i.ID] = i
	}
	result := map[string]string{}
	for _, i := range items {
		root := i
		seen := map[string]bool{i.ID: true}
		for root.ParentItemID != "" {
			parent, ok := byID[root.ParentItemID]
			if !ok || seen[parent.ID] {
				break
			}
			seen[parent.ID] = true
			root = parent
		}
		section := "contingency"
		switch root.LineKind {
		case "fee":
			section = "consultants"
		case "works":
			section = "construction"
		case "project_wide":
			if root.Category == "authority_fees" {
				section = "authority"
			}
		}
		result[i.ID] = section
	}
	return result
}
