package program

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Cal maps working-day indices to calendar days (plan 6.1). Index 0 is the
// first working day on or after the start date. Days are integers (days since
// 1970-01-01 internally, offsets from the start date in method arguments);
// time.Time is used only to parse and format, always in UTC, so no time zone
// can move a date.
//
// The whole calendar reduces to count(d): the number of working days in
// [0, d) for d ≥ 0, and minus the number in [d, 0) for d < 0. Every lookup is
// arithmetic on the weekly pattern plus one binary search over the sorted
// exceptions, so it costs O(log holidays) and never walks day by day.
type Cal struct {
	relative bool
	origin   int     // absolute day number of the start date (offset 0)
	perWeek  int     // working days in a normal week
	pre      [8]int  // pre[r]: pattern working days in offsets [0, r)
	works    [7]bool // works[r]: offset r (mod 7) is a pattern working day
	exDays   []int   // offsets of effective exceptions, ascending
	exSum    []int   // exSum[k]: sum of deltas of exDays[:k]
	p0       int     // sum of deltas of exceptions before offset 0
	// cache holds the dates of indices [cacheLo, cacheLo+cacheN) once warm is
	// called, 10 bytes each in one string, so every bar end is a substring:
	// one allocation for a whole read instead of one per date.
	cacheLo int
	cacheN  int
	cache   string
}

// relativeOrigin is a Monday. A relative program anchors its weekly pattern
// here but never shows a date from it.
var relativeOrigin = mustDay("2001-01-01")

func mustDay(s string) int {
	d, ok := dayNumber(s)
	if !ok {
		panic(s)
	}
	return d
}

// dayNumber parses a strict YYYY-MM-DD date.
func dayNumber(s string) (int, bool) {
	if len(s) != 10 {
		return 0, false
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return 0, false
	}
	return int(t.Unix() / 86400), true
}

// dayString formats a day number as YYYY-MM-DD with the civil-from-days
// algorithm (H. Hinnant), avoiding time.Format's allocations on the hot path.
func dayString(d int) string {
	var b [10]byte
	putDay(b[:], d)
	return string(b[:])
}

func putDay(b []byte, d int) { civil(d).put(b) }

type ymd struct{ y, m, d int }

func civil(n int) ymd {
	z := n + 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if mp >= 10 {
		m = mp - 9
	}
	if m <= 2 {
		y++
	}
	return ymd{y, m, day}
}

// next steps one calendar day; walking a span this way is far cheaper than
// converting each day from scratch.
func (c *ymd) next() {
	c.d++
	if c.d <= 28 {
		return
	}
	days := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[c.m]
	if c.m == 2 && c.y%4 == 0 && (c.y%100 != 0 || c.y%400 == 0) {
		days = 29
	}
	if c.d > days {
		c.d, c.m = 1, c.m+1
		if c.m > 12 {
			c.m, c.y = 1, c.y+1
		}
	}
}

func (c ymd) put(b []byte) {
	y, m, day := c.y, c.m, c.d
	b[0], b[1], b[2], b[3] = byte('0'+y/1000%10), byte('0'+y/100%10), byte('0'+y/10%10), byte('0'+y%10)
	b[4], b[7] = '-', '-'
	b[5], b[6] = byte('0'+m/10), byte('0'+m%10)
	b[8], b[9] = byte('0'+day/10), byte('0'+day%10)
}

func floorDiv(a, b int) int {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

// weekdayBit is the bitmask bit for an absolute day: Mon=1 … Sun=64.
// 1 January 1970 was a Thursday, so (d+3) mod 7 counts from Monday = 0.
func weekdayBit(d int) int {
	return 1 << ((d%7 + 7 + 3) % 7)
}

// NewCalendar builds a calendar. It is tolerant: a malformed start date makes
// the program relative and a malformed day is skipped, because Compute has no
// error path. Validate reports those problems at the boundary.
func NewCalendar(c Calendar) *Cal {
	mask := c.WorkingWeek
	if mask <= 0 || mask > 127 {
		mask = DefaultWorkingWeek
	}
	cal := &Cal{exSum: []int{0}}
	if d, ok := dayNumber(c.Start); ok {
		cal.origin = d
	} else {
		cal.relative = true
		cal.origin = relativeOrigin
	}
	for r := 0; r < 7; r++ {
		cal.works[r] = weekdayBit(cal.origin+r)&mask != 0
		cal.pre[r+1] = cal.pre[r]
		if cal.works[r] {
			cal.pre[r+1]++
			cal.perWeek++
		}
	}
	// A relative program has no dates, so holidays cannot apply to it.
	if !cal.relative {
		cal.setExceptions(c.Days)
	}
	return cal
}

// setExceptions keeps only days that change the pattern: a holiday on a
// working weekday (−1) or an override on an idle weekday (+1). A holiday on a
// Saturday changes nothing and must not be counted twice.
func (c *Cal) setExceptions(days []CalendarDay) {
	kinds := map[int]string{}
	for _, d := range days {
		n, ok := dayNumber(d.Day)
		if !ok || !validDayKind(d.Kind) {
			continue
		}
		kinds[n-c.origin] = d.Kind
	}
	delta := map[int]int{}
	for off, kind := range kinds {
		works := c.works[off-7*floorDiv(off, 7)]
		if kind == DayWorkingOverride && !works {
			delta[off] = 1
		} else if kind != DayWorkingOverride && works {
			delta[off] = -1
		}
	}
	c.exDays = make([]int, 0, len(delta))
	for off := range delta {
		c.exDays = append(c.exDays, off)
	}
	sort.Ints(c.exDays)
	c.exSum = make([]int, len(c.exDays)+1)
	for k, off := range c.exDays {
		c.exSum[k+1] = c.exSum[k] + delta[off]
	}
	c.p0 = c.exBefore(0)
}

func validDayKind(k string) bool {
	return k == DayPublicHoliday || k == DayRDO || k == DayShutdown || k == DayWorkingOverride
}

func (c *Cal) exBefore(off int) int {
	lo, hi := 0, len(c.exDays) // first k with exDays[k] >= off
	for lo < hi {
		m := int(uint(lo+hi) >> 1)
		if c.exDays[m] < off {
			lo = m + 1
		} else {
			hi = m
		}
	}
	return c.exSum[lo]
}

// count is the signed number of working days in [0, off).
func (c *Cal) count(off int) int {
	q := floorDiv(off, 7)
	return q*c.perWeek + c.pre[off-7*q] + c.exBefore(off) - c.p0
}

// dayOf is the offset of working day i: the smallest offset d with
// count(d+1) ≥ i+1. It starts from the pattern's guess and widens until it
// brackets the answer, so a long shutdown costs a few more probes, not a loop.
func (c *Cal) dayOf(i int) int {
	g := floorDiv(i, c.perWeek) * 7
	lo, hi := g-7, g+7
	for step := 7; c.count(lo+1) >= i+1; step *= 2 {
		lo -= step
	}
	for step := 7; c.count(hi+1) < i+1; step *= 2 {
		hi += step
	}
	for hi-lo > 1 { // count(lo+1) < i+1 ≤ count(hi+1)
		m := lo + (hi-lo)/2
		if c.count(m+1) >= i+1 {
			hi = m
		} else {
			lo = m
		}
	}
	return hi
}

// Boundary helpers for links in calendar days (plan 6.2). An activity starts
// at the beginning of its first day and finishes at the end of its last day,
// which is the beginning of the next calendar day. Using that instant, a
// Friday finish plus 3 calendar days is Tuesday (Saturday, Sunday and Monday
// pass), which is what "3 days to cure" means. A zero-length row finishes
// when it starts. Each "on or before" helper is the exact inverse of its "on
// or after" pair, so the backward pass never invents float.

func (c *Cal) startDay(s int) int { return c.dayOf(s) }

func (c *Cal) finishDay(f, d int) int {
	if d == 0 {
		return c.dayOf(f)
	}
	return c.dayOf(f-1) + 1
}

// startOnOrAfter: the first index whose start instant is at or after offset m.
func (c *Cal) startOnOrAfter(m int) int { return c.count(m) }

// startOnOrBefore: the last index whose start instant is at or before m.
func (c *Cal) startOnOrBefore(m int) int { return c.count(m+1) - 1 }

// finishOnOrAfter: the first exclusive finish, for duration d, whose finish
// instant is at or after m.
func (c *Cal) finishOnOrAfter(m, d int) int {
	if d == 0 {
		return c.count(m)
	}
	return c.count(m-1) + 1
}

// finishOnOrBefore: the last exclusive finish, for duration d, whose finish
// instant is at or before m.
func (c *Cal) finishOnOrBefore(m, d int) int {
	if d == 0 {
		return c.count(m+1) - 1
	}
	return c.count(m)
}

// Relative reports a program with no start date: no dates, weeks only.
func (c *Cal) Relative() bool { return c.relative }

// WorkingDaysPerWeek is what "3w" multiplies by.
func (c *Cal) WorkingDaysPerWeek() int { return c.perWeek }

// DateOf is the date of working day i, or "" for a relative program.
func (c *Cal) DateOf(i int) string {
	if c.relative {
		return ""
	}
	if k := i - c.cacheLo; k >= 0 && k < c.cacheN {
		return c.cache[10*k : 10*k+10]
	}
	return dayString(c.origin + c.dayOf(i))
}

// warm precomputes the dates of indices lo..hi by walking the calendar once,
// so formatting a whole program costs one pass over its span instead of a
// search per date. Spans far longer than the rows they serve are skipped:
// DateOf still answers, just one search at a time.
func (c *Cal) warm(lo, hi, rows int) {
	if c.relative || hi < lo || hi-lo > 8*rows+400 {
		return
	}
	n := hi - lo + 1
	buf := make([]byte, 10*n)
	k := 0 // next exception at or after d
	d := c.dayOf(lo)
	for k < len(c.exDays) && c.exDays[k] < d {
		k++
	}
	date := civil(c.origin + d)
	for i := 0; i < n; i++ {
		date.put(buf[10*i:])
		// Step to the next working day: the pattern, flipped by any
		// exception that falls on the day.
		for {
			d++
			date.next()
			for k < len(c.exDays) && c.exDays[k] < d {
				k++
			}
			works := c.works[d-7*floorDiv(d, 7)]
			if k < len(c.exDays) && c.exDays[k] == d {
				works = !works
			}
			if works {
				break
			}
		}
	}
	c.cache, c.cacheLo, c.cacheN = string(buf), lo, n
}

// IndexOnOrAfter is the index of the first working day on or after date.
func (c *Cal) IndexOnOrAfter(date string) (int, bool) {
	d, ok := dayNumber(date)
	if !ok || c.relative {
		return 0, false
	}
	return c.count(d - c.origin), true
}

// IndexOnOrBefore is the index of the last working day on or before date.
func (c *Cal) IndexOnOrBefore(date string) (int, bool) {
	d, ok := dayNumber(date)
	if !ok || c.relative {
		return 0, false
	}
	return c.count(d-c.origin+1) - 1, true
}

// ShiftCalendar moves a start index by calendar days and lands on the first
// working day on or after the result.
func (c *Cal) ShiftCalendar(i, days int) int {
	return c.startOnOrAfter(c.dayOf(i) + days)
}

// Display is the one conversion from a half-open interval to shown dates
// (plan 6.1): start = date(start); finish = date(start+duration−1) when the
// duration is above zero, and "" for a milestone, which shows its start only.
// Both are "" for a relative program.
func (c *Cal) Display(start, duration int) (string, string) {
	if c.relative {
		return "", ""
	}
	if duration <= 0 {
		return c.DateOf(start), ""
	}
	return c.DateOf(start), c.DateOf(start + duration - 1)
}

// WeekLabel names the week holding working day i, counted from the start
// date: "Week 1" for the first seven calendar days. A relative program shows
// this instead of dates.
func (c *Cal) WeekLabel(i int) string {
	return "Week " + strconv.Itoa(floorDiv(c.dayOf(i), 7)+1)
}

// DayLabel is WeekLabel plus the working day within that week, for a relative
// program's tooltips: "Week 2, day 3".
func (c *Cal) DayLabel(i int) string {
	off := c.dayOf(i)
	week := floorDiv(off, 7)
	n := c.count(off+1) - c.count(week*7)
	return fmt.Sprintf("Week %d, day %d", week+1, n)
}
