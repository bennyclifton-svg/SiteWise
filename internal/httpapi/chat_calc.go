package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"sitewise/internal/agent"
	"sitewise/internal/costs"
	"sitewise/internal/figures"
)

// calc is the chat's calculator (review change 2): Go does every sum, so
// the model never does. It takes only figures that are in the turn's book
// (the PM typed them or a tool returned them) and its result is SiteWise's
// figure, with a tag the answer can name.
func (b *chatBox) calc(ctx context.Context, st *agent.State, input json.RawMessage) agent.Output {
	var in struct {
		Op     string   `json:"op"`
		Values []string `json:"values"`
		Date   string   `json:"date"`
		Span   string   `json:"span"`
	}
	_ = json.Unmarshal(input, &in)
	nums := make([]*big.Rat, 0, len(in.Values))
	for _, v := range in.Values {
		a, ok := figures.Amount(strings.TrimSuffix(strings.TrimSpace(v), "%"))
		if !ok {
			return toolError("%q is not a number.", v)
		}
		if !st.Book.HasNumber(a) {
			return toolError("%s is not in the PM's message or in anything read this turn. calc only works on figures that were given.", v)
		}
		r, _ := new(big.Rat).SetString(a)
		nums = append(nums, r)
	}
	need := func(n int) *agent.Output {
		if len(nums) != n {
			out := toolError("%s takes %d values.", in.Op, n)
			return &out
		}
		return nil
	}
	money := func(r *big.Rat) string { p, _ := figures.Plain(r.FloatString(2)); return p }
	result := func(value, shown, label string) agent.Output {
		tag := b.nextTag("f")()
		return agent.Output{Text: fmt.Sprintf("%s = %s [%s]", label, value, tag), Facts: map[string]agent.Fact{tag: {Label: label, Value: shown, Source: "calculated by SiteWise"}}}
	}
	list := func(sep string) string {
		parts := make([]string, len(nums))
		for i, n := range nums {
			parts[i] = dollars(costs.Money(money(n)))
		}
		return strings.Join(parts, sep)
	}
	hundred := big.NewRat(100, 1)
	switch in.Op {
	case "sum":
		if len(nums) < 2 {
			return toolError("sum takes two or more values.")
		}
		total := new(big.Rat)
		for _, n := range nums {
			total.Add(total, n)
		}
		return result(money(total), dollars(costs.Money(money(total))), "Sum of "+list(", "))
	case "difference":
		if bad := need(2); bad != nil {
			return *bad
		}
		d := new(big.Rat).Sub(nums[0], nums[1])
		return result(money(d), dollars(costs.Money(money(d))), list(" less "))
	case "percentage", "ratio":
		if bad := need(2); bad != nil {
			return *bad
		}
		if nums[1].Sign() == 0 {
			return toolError("The second value is zero.")
		}
		q := new(big.Rat).Quo(nums[0], nums[1])
		if in.Op == "ratio" {
			return result(q.FloatString(3), q.FloatString(3), list(" over "))
		}
		pct := new(big.Rat).Mul(q, hundred).FloatString(1)
		return result(pct+"%", pct+"%", list(" as a share of "))
	case "percent_of":
		if bad := need(2); bad != nil {
			return *bad
		}
		v := new(big.Rat).Quo(new(big.Rat).Mul(nums[0], nums[1]), hundred)
		pct, _ := figures.Plain(nums[0].FloatString(4))
		return result(money(v), dollars(costs.Money(money(v))), pct+"% of "+dollars(costs.Money(money(nums[1]))))
	case "gst_ex", "gst_inc":
		if bad := need(1); bad != nil {
			return *bad
		}
		plan, err := b.d.Store.ReadCostPlan(ctx, b.who.OrgID, b.project, "")
		if err != nil || plan.TaxRate == nil {
			return toolError("The cost plan has no GST rate set, so this cannot be converted.")
		}
		rate, _ := new(big.Rat).SetString(string(*plan.TaxRate))
		// The plan stores the rate as a percentage (10) or a fraction (0.1).
		if rate.Cmp(big.NewRat(1, 1)) >= 0 {
			rate.Quo(rate, hundred)
		}
		factor := new(big.Rat).Add(big.NewRat(1, 1), rate)
		var v *big.Rat
		label := dollars(costs.Money(money(nums[0])))
		if in.Op == "gst_ex" {
			v = new(big.Rat).Quo(nums[0], factor)
			label += " including GST, with GST taken out"
		} else {
			v = new(big.Rat).Mul(nums[0], factor)
			label += " excluding GST, with GST added"
		}
		return result(money(v), dollars(costs.Money(money(v))), label)
	case "working_days", "shift_date":
		week := 5
		if prog, err := b.d.Store.LoadProgram(ctx, b.who.OrgID, b.project); err == nil {
			week = workingDays(prog.Program.WorkingWeek)
		}
		n, _, ok := figures.Duration(in.Span)
		days, ok2 := figures.WorkingDays(in.Span, week)
		if !ok || !ok2 || !(st.Book.HasNumber(fmt.Sprint(n)) || strings.Contains(strings.ToLower(in.Span), "fortnight")) {
			return toolError("Give span as the PM said it, for example \"2 weeks\".")
		}
		if in.Op == "working_days" {
			return result(fmt.Sprintf("%d working days", days), fmt.Sprintf("%d working days", days), fmt.Sprintf("%s on a %d-day week", in.Span, week))
		}
		from, err := time.Parse("2006-01-02", in.Date)
		if err != nil || !st.Book.HasDate(in.Date, st.Today) {
			return toolError("Give date as YYYY-MM-DD, a date from a read or from the PM.")
		}
		// A plain shift over the working week; public holidays are in the
		// Program's own calendar, which computes the real dates.
		for left := days; left > 0; {
			from = from.AddDate(0, 0, 1)
			wd := (int(from.Weekday()) + 6) % 7 // Monday is 0
			if wd < week {
				left--
			}
		}
		iso := from.Format("2006-01-02")
		return result(iso, iso, fmt.Sprintf("%s moved by %s on a %d-day week, public holidays not counted", in.Date, in.Span, week))
	}
	return toolError("calc does not know %q.", in.Op)
}
