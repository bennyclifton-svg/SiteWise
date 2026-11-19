package invoices

import (
	"fmt"
	"strings"
	"time"

	"sitewise/internal/program"
)

// Security of Payment dates (docs/plans/2026-11-17-variations-progress-claims.md,
// Task 8). Go works out when a claim's payment schedule is due and when
// payment is due: the earlier of the contract's own period and the Act's,
// counted after the day the claim was served. Business days skip weekends
// and the state's reviewed public holidays only, never a project's rostered
// days off or shutdowns, which would make a date late. Every fallback errs
// early: an unverified Act's excluded spans and any holiday not loaded are
// left out, so the true date is never earlier than the one shown. Dates are
// arithmetic on the Act rows and the PM's inputs, not legal advice, and the
// words say which inputs were used. Nothing is stored; it is worked out at
// read.

// SOPAct is one state's periods as the knowledge file records them.
type SOPAct struct {
	State, Act                      string
	ScheduleSection, PaymentSection string
	ScheduleDays, PaymentDays       int // 0 is not recorded
	Verified                        bool
	Source                          string // where the figures came from
	Spans                           [][2]string
}

// SOPInput is what a claim's dates are worked out from.
type SOPInput struct {
	State string
	Act   *SOPAct // nil: no row for the state
	// Holidays are the state's reviewed public holidays; Years the years
	// they cover.
	Holidays []string
	Years    map[int]bool
	// Served is the day the claim was served; ServedConfirmed is the PM's
	// word, else it is the drop day by rule.
	Served          string
	ServedConfirmed bool
	ScheduleDays    *int
	ScheduleBasis   string // business or calendar
	PaymentDays     *int
	PaymentBasis    string
}

// DueDate is one worked-out date with how it was worked out.
type DueDate struct {
	Date    string   `json:"date"`
	Words   string   `json:"words"`
	Section string   `json:"section"`
	Source  string   `json:"source"` // act, contract, earlier or unknown
	ToCheck bool     `json:"to_check"`
	Notes   []string `json:"notes"`
}

// ScheduleDue is when the payment schedule must be served.
func ScheduleDue(in SOPInput) DueDate {
	var act *period
	if in.Act != nil && in.Act.ScheduleDays > 0 {
		act = &period{in.Act.ScheduleDays, in.Act.ScheduleSection}
	}
	return dueDate(in, "payment schedule", act, in.ScheduleDays, in.ScheduleBasis)
}

// PaymentDue is when payment of the scheduled amount is due.
func PaymentDue(in SOPInput) DueDate {
	var act *period
	if in.Act != nil && in.Act.PaymentDays > 0 {
		act = &period{in.Act.PaymentDays, in.Act.PaymentSection}
	}
	return dueDate(in, "payment", act, in.PaymentDays, in.PaymentBasis)
}

type period struct {
	days    int
	section string
}

// sopCalendar is weekends and the state's loaded public holidays; a
// verified Act's yearly spans too.
func sopCalendar(in SOPInput) program.Calendar {
	c := program.Calendar{WorkingWeek: program.DefaultWorkingWeek}
	for _, d := range in.Holidays {
		c.Days = append(c.Days, program.CalendarDay{Day: d, Kind: "public_holiday"})
	}
	if in.Act != nil && in.Act.Verified && in.Served != "" {
		if t, err := time.Parse("2006-01-02", in.Served); err == nil {
			for y := t.Year() - 1; y <= t.Year()+1; y++ {
				for _, s := range in.Act.Spans {
					from, e1 := time.Parse("2006-01-02", fmt.Sprintf("%d-%s", y, s[0]))
					to, e2 := time.Parse("2006-01-02", fmt.Sprintf("%d-%s", y, s[1]))
					for d := from; e1 == nil && e2 == nil && !d.After(to); d = d.AddDate(0, 0, 1) {
						c.Days = append(c.Days, program.CalendarDay{Day: d.Format("2006-01-02"), Kind: "public_holiday"})
					}
				}
			}
		}
	}
	return c
}

func after(cal program.Calendar, from string, days int, basis string) string {
	if basis == "calendar" {
		t, err := time.Parse("2006-01-02", from)
		if err != nil {
			return ""
		}
		return t.AddDate(0, 0, days).Format("2006-01-02")
	}
	d, ok := program.BusinessDayAfter(cal, from, days)
	if !ok {
		return ""
	}
	return d
}

func basisWord(basis string) string {
	if basis == "calendar" {
		return "calendar days"
	}
	return "business days"
}

func dueDate(in SOPInput, what string, act *period, contractDays *int, basis string) DueDate {
	switch {
	case in.State == "":
		return DueDate{Source: "unknown", ToCheck: true, Words: "Due date unknown: set the project's state on the Profile."}
	case in.Served == "":
		return DueDate{Source: "unknown", ToCheck: true, Words: "Due date unknown: enter the date the claim was served."}
	case act == nil && contractDays == nil:
		return DueDate{Source: "unknown", ToCheck: true, Words: fmt.Sprintf("Due date unknown: no %s period is recorded for %s; enter the contract's period.", what, in.State)}
	}
	cal := sopCalendar(in)
	out := DueDate{}
	var actDate, contractDate string
	if act != nil {
		actDate = after(cal, in.Served, act.days, "business")
	}
	if contractDays != nil {
		contractDate = after(cal, in.Served, *contractDays, basis)
	}
	actWords := ""
	if act != nil {
		actWords = fmt.Sprintf("the %s period of %d business days after service (%s)", in.State, act.days, act.section)
	}
	switch {
	case act != nil && contractDays != nil:
		out.Date, out.Source, out.Section = actDate, "act", act.section
		out.Words = fmt.Sprintf("The earlier of the contract's %d %s and %s.", *contractDays, basisWord(basis), actWords)
		if contractDate <= actDate {
			out.Date, out.Source = contractDate, "contract"
		} else {
			out.Notes = append(out.Notes, "The contract's period is longer than the Act's; the contract cannot extend it.")
		}
	case act != nil:
		out.Date, out.Source, out.Section = actDate, "act", act.section
		out.Words = fmt.Sprintf("No contract period entered: %s.", actWords)
	default:
		out.Date, out.Source = contractDate, "contract"
		out.Words = fmt.Sprintf("The contract's %d %s after service. No %s period is recorded for %s, so the Act may set an earlier date.", *contractDays, basisWord(basis), what, in.State)
		out.ToCheck = true
	}
	if act != nil && in.Act != nil && !in.Act.Verified {
		out.ToCheck = true
		src := in.Act.Source
		if src == "" {
			src = "a source other than the Act"
		}
		out.Notes = append(out.Notes, fmt.Sprintf("The %s period is not verified against the Act (%s); to check.", in.State, src))
	}
	if !in.ServedConfirmed {
		out.ToCheck = true
		out.Notes = append(out.Notes, "Counted from the day the claim was dropped; confirm the day it was served.")
	}
	if missing := missingYears(in, out.Date); len(missing) > 0 {
		out.ToCheck = true
		out.Notes = append(out.Notes, fmt.Sprintf("Public holidays for %s %s are not loaded, so none is skipped: serve by this date to be safe.", in.State, strings.Join(missing, " and ")))
	}
	return out
}

func missingYears(in SOPInput, due string) []string {
	from, e1 := time.Parse("2006-01-02", in.Served)
	to, e2 := time.Parse("2006-01-02", due)
	if e1 != nil || e2 != nil {
		return nil
	}
	var out []string
	for y := from.Year(); y <= to.Year(); y++ {
		if !in.Years[y] {
			out = append(out, fmt.Sprint(y))
		}
	}
	return out
}

// Claim alerts for the one banner, the left-nav mark and Attention.
const (
	AlertOverdue = "overdue" // past due, no schedule served (red)
	AlertSoon    = "soon"    // due within two business days (amber)
	AlertCheck   = "check"   // a date to check, or unknown (amber)
)

// ScheduleAlert says how urgent an unserved schedule is on today.
func ScheduleAlert(in SOPInput, due DueDate, served bool, today string) string {
	switch {
	case served:
		return ""
	case due.Date == "":
		return AlertCheck
	case today > due.Date:
		return AlertOverdue
	}
	if soon, ok := program.BusinessDayAfter(sopCalendar(in), today, 2); ok && soon >= due.Date {
		return AlertSoon
	}
	if due.ToCheck {
		return AlertCheck
	}
	return ""
}
