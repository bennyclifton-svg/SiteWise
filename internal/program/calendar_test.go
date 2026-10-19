package program

import (
	"math/rand"
	"testing"
)

func cal(start string, week int, days ...CalendarDay) *Cal {
	return NewCalendar(Calendar{Start: start, WorkingWeek: week, Days: days})
}

func day(d, kind string) CalendarDay { return CalendarDay{Day: d, Kind: kind} }

func TestCalendarWeekend(t *testing.T) {
	c := cal("2027-03-05", 31) // a Friday
	if got := c.DateOf(0); got != "2027-03-05" {
		t.Errorf("index 0 = %s", got)
	}
	if got := c.DateOf(1); got != "2027-03-08" {
		t.Errorf("index 1 skips the weekend: %s", got)
	}
	if got := c.DateOf(-1); got != "2027-03-04" {
		t.Errorf("index -1 = %s", got)
	}
	for _, x := range []struct {
		date          string
		after, before int
	}{{"2027-03-06", 1, 0}, {"2027-03-07", 1, 0}, {"2027-03-08", 1, 1}, {"2027-03-01", -4, -4}} {
		a, _ := c.IndexOnOrAfter(x.date)
		b, _ := c.IndexOnOrBefore(x.date)
		if a != x.after || b != x.before {
			t.Errorf("%s: on or after %d, on or before %d; want %d, %d", x.date, a, b, x.after, x.before)
		}
	}
	// A Sunday start: index 0 is the first working day on or after it.
	if got := cal("2027-03-07", 31).DateOf(0); got != "2027-03-08" {
		t.Errorf("Sunday start: index 0 = %s", got)
	}
}

func TestCalendarHoliday(t *testing.T) {
	c := cal("2027-03-01", 31, day("2027-03-08", DayPublicHoliday))
	if got := c.DateOf(5); got != "2027-03-09" {
		t.Errorf("index 5 skips the Monday holiday: %s", got)
	}
	a, _ := c.IndexOnOrAfter("2027-03-08")
	b, _ := c.IndexOnOrBefore("2027-03-08")
	if a != 5 || b != 4 {
		t.Errorf("holiday: on or after %d, on or before %d", a, b)
	}
	// A holiday that falls on a weekend changes nothing.
	if got := cal("2027-03-01", 31, day("2027-03-06", DayPublicHoliday)).DateOf(5); got != "2027-03-08" {
		t.Errorf("Saturday holiday counted twice: %s", got)
	}
	// Index 4 (Fri 5 Mar) moved 3 calendar days is Mon 8, a holiday, so Tue 9.
	if got := c.ShiftCalendar(4, 3); got != 5 {
		t.Errorf("shift = %d", got)
	}
}

func TestCalendarShutdownAcrossYearEnd(t *testing.T) {
	var days []CalendarDay
	for _, d := range []string{"2027-12-24", "2027-12-27", "2027-12-28", "2027-12-29", "2027-12-30", "2027-12-31", "2028-01-03", "2028-01-04", "2028-01-05", "2028-01-06", "2028-01-07"} {
		days = append(days, day(d, DayShutdown))
	}
	c := cal("2027-12-20", 31, days...)
	want := []string{"2027-12-20", "2027-12-21", "2027-12-22", "2027-12-23", "2028-01-10", "2028-01-11"}
	for i, w := range want {
		if got := c.DateOf(i); got != w {
			t.Errorf("index %d = %s, want %s", i, got, w)
		}
	}
	if i, _ := c.IndexOnOrAfter("2027-12-25"); i != 4 {
		t.Errorf("first working day after Christmas is index %d", i)
	}
	if i, _ := c.IndexOnOrBefore("2028-01-09"); i != 3 {
		t.Errorf("last working day on or before 9 Jan is index %d", i)
	}
}

func TestCalendarWorkingOverride(t *testing.T) {
	c := cal("2027-03-01", 31, day("2027-03-06", DayWorkingOverride), day("2027-03-03", DayWorkingOverride))
	// The Saturday works; the override on a Wednesday that already works
	// changes nothing.
	want := []string{"2027-03-01", "2027-03-02", "2027-03-03", "2027-03-04", "2027-03-05", "2027-03-06", "2027-03-08"}
	for i, w := range want {
		if got := c.DateOf(i); got != w {
			t.Errorf("index %d = %s, want %s", i, got, w)
		}
	}
	// A six-day week from the bitmask (Mon … Sat = 63).
	if got := cal("2027-03-01", 63).DateOf(5); got != "2027-03-06" {
		t.Errorf("six-day week index 5 = %s", got)
	}
}

func TestCalendarRelative(t *testing.T) {
	c := cal("", 0, day("2027-03-02", DayPublicHoliday))
	if !c.Relative() || c.DateOf(3) != "" {
		t.Fatal("a program without a start date shows no dates")
	}
	if c.WeekLabel(4) != "Week 1" || c.WeekLabel(5) != "Week 2" || c.DayLabel(6) != "Week 2, day 2" {
		t.Errorf("labels: %s, %s, %s", c.WeekLabel(4), c.WeekLabel(5), c.DayLabel(6))
	}
	if s, f := c.Display(0, 5); s != "" || f != "" {
		t.Errorf("relative display %q %q", s, f)
	}
}

func TestDisplayHalfOpen(t *testing.T) {
	c := cal("2027-03-01", 31)
	if s, f := c.Display(0, 5); s != "2027-03-01" || f != "2027-03-05" {
		t.Errorf("5 days from Monday: %s to %s", s, f)
	}
	if s, f := c.Display(3, 1); s != "2027-03-04" || f != "2027-03-04" {
		t.Errorf("one day: %s to %s", s, f)
	}
	if s, f := c.Display(5, 0); s != "2027-03-08" || f != "" {
		t.Errorf("milestone shows its start only: %q %q", s, f)
	}
}

// brute lists the offsets of working days in [-400, 1200) by walking day by
// day, the obvious way the fast arithmetic must agree with.
func brute(c *Cal, mask int, kinds map[int]string) []int {
	var out []int
	for d := -400; d < 1200; d++ {
		works := weekdayBit(c.origin+d)&mask != 0
		switch k, ok := kinds[d]; {
		case !ok:
		case k == DayWorkingOverride:
			works = true
		default:
			works = false
		}
		if works {
			out = append(out, d)
		}
	}
	return out
}

func randomCalendar(r *rand.Rand) (*Cal, int, map[int]string) {
	mask := 1 + r.Intn(127)
	start := mustDay("2027-03-01") + r.Intn(14)
	kinds := map[int]string{}
	var days []CalendarDay
	for k := 0; k < r.Intn(60); k++ {
		off := r.Intn(1400) - 300
		kind := []string{DayPublicHoliday, DayRDO, DayShutdown, DayWorkingOverride}[r.Intn(4)]
		if _, dup := kinds[off]; dup {
			continue
		}
		kinds[off] = kind
		days = append(days, CalendarDay{Day: dayString(start + off), Kind: kind})
	}
	return NewCalendar(Calendar{Start: dayString(start), WorkingWeek: mask, Days: days}), mask, kinds
}

func TestCalendarMatchesDayByDayWalk(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	for n := 0; n < 200; n++ {
		c, mask, kinds := randomCalendar(r)
		work := brute(c, mask, kinds)
		zero := 0
		for zero < len(work) && work[zero] < 0 {
			zero++
		}
		for k, d := range work {
			i := k - zero
			if i < -100 || i > 600 {
				continue
			}
			if got := c.dayOf(i); got != d {
				t.Fatalf("case %d: index %d is day %d, walk says %d", n, i, got, d)
			}
			date := dayString(c.origin + d)
			a, _ := c.IndexOnOrAfter(date)
			b, _ := c.IndexOnOrBefore(date)
			if a != i || b != i {
				t.Fatalf("case %d: %s round trip gives %d and %d, want %d", n, date, a, b, i)
			}
		}
	}
}

// The four boundary helpers behind calendar-day lag are checked against
// their definitions by search: "on or after" is the least index meeting the
// bound, "on or before" the greatest. Being exact inverses is what keeps the
// backward pass from inventing or hiding float.
func TestCalendarLagBoundaries(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	for n := 0; n < 100; n++ {
		c, _, _ := randomCalendar(r)
		for k := 0; k < 50; k++ {
			m := r.Intn(500) - 50
			d := []int{0, 1, 3, 10}[r.Intn(4)]
			least, greatest := func(ok func(int) bool) int {
				for i := -200; ; i++ {
					if ok(i) {
						return i
					}
				}
			}, func(ok func(int) bool) int {
				for i := 800; ; i-- {
					if ok(i) {
						return i
					}
				}
			}
			cases := []struct {
				name      string
				got, want int
			}{
				{"startOnOrAfter", c.startOnOrAfter(m), least(func(s int) bool { return c.startDay(s) >= m })},
				{"startOnOrBefore", c.startOnOrBefore(m), greatest(func(s int) bool { return c.startDay(s) <= m })},
				{"finishOnOrAfter", c.finishOnOrAfter(m, d), least(func(f int) bool { return c.finishDay(f, d) >= m })},
				{"finishOnOrBefore", c.finishOnOrBefore(m, d), greatest(func(f int) bool { return c.finishDay(f, d) <= m })},
			}
			for _, x := range cases {
				if x.got != x.want {
					t.Fatalf("case %d: %s(m=%d, d=%d) = %d, want %d", n, x.name, m, d, x.got, x.want)
				}
			}
		}
	}
}

// Calendar-day lag in both passes, on the engine itself: a Friday finish
// plus 3 calendar days starts Tuesday when Monday is a holiday, and the
// backward pass gives the predecessor exactly the float the forward pass
// would allow.
func TestCalendarLagBothPasses(t *testing.T) {
	in := Input{
		Calendar: Calendar{Start: "2027-03-01", WorkingWeek: 31, Days: []CalendarDay{day("2027-03-08", DayPublicHoliday)}},
		Activities: []Activity{
			{ID: "a", Duration: ptr(5)},
			{ID: "b", Duration: ptr(2)},
			{ID: "c", Duration: ptr(12)},
		},
		Links: []Link{{ID: "ab", Pred: "a", Succ: "b", Type: LinkFS, Lag: 3, LagUnit: LagCalendar}},
	}
	out := Compute(in)
	m := out.Activities.Map()
	if b := m["b"]; b.ES != 5 || b.Start != "2027-03-09" {
		t.Fatalf("b starts %d (%s), want 5 (Tue 9 Mar)", b.ES, b.Start)
	}
	// c makes PF 12, so b has float; the largest slip of a that keeps b's
	// start within its late start is a's float, found here by trying.
	a := m["a"]
	slip := 0
	for s := 1; s < 10; s++ {
		trial := in
		trial.Activities = append([]Activity(nil), in.Activities...)
		trial.Activities[0].Constraint, trial.Activities[0].ConstraintDate = ConstraintSNET, NewCalendar(in.Calendar).DateOf(s)
		if Compute(trial).Finish.TypicalIndex == nil || *Compute(trial).Finish.TypicalIndex > 12 {
			break
		}
		slip = s
	}
	if a.TF != slip {
		t.Errorf("a's float %d, but it can slip %d days without moving the finish", a.TF, slip)
	}
}
