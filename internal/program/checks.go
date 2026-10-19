package program

import (
	"fmt"
	"strconv"
	"time"
)

// Check codes (plan 6.6).
const (
	CheckConsentLapse      = "consent_lapse"
	CheckTargetCompletion  = "target_completion"
	CheckMSOConflict       = "mso_conflict"
	CheckCalendarGap       = "calendar_gap"
	CheckHoldPointUnlinked = "hold_point_unlinked"
	CheckSkeletonTie       = "skeleton_tie"
	CheckPackageRemoved    = "package_removed"
)

type Check struct {
	Code       string `json:"code"`
	ActivityID string `json:"activity_id,omitempty"`
	Message    string `json:"message"`
}

// CheckInput carries the facts generation and the store know and the engine
// does not. Empty fields switch their check off.
type CheckInput struct {
	Input  Input
	Output Output
	// ConstructionStageID is the stage whose start is compared with the
	// consent lapse date.
	ConstructionStageID string
	ConsentLapseDate    string // YYYY-MM-DD
	TargetCompletion    string // YYYY-MM-DD
	// CalendarYears are years the library has calendar rows for. Years that
	// appear in Input.Calendar.Days count as covered as well.
	CalendarYears []int
	SkeletonTie   bool
	// RemovedPackageRows are activities of packages removed in Procurement
	// that the user had edited, so they were kept for review.
	RemovedPackageRows []string
}

// longDate is the app's date style in messages ("12 March 2028"), the same
// layout the invoice export prints.
func longDate(iso string) string {
	d, ok := dayNumber(iso)
	if !ok {
		return iso
	}
	return time.Unix(int64(d)*86400, 0).UTC().Format("2 January 2006")
}

// Checks returns the program's warnings in a fixed order. Messages state the
// dates and never claim a breach.
func Checks(ci CheckInput) []Check {
	in, out := ci.Input, ci.Output
	cal := NewCalendar(in.Calendar)
	// While a loop stands the output has no dates, so the date checks find
	// nothing to say; the others still apply.
	var list []Check
	label := map[string]string{}
	for _, a := range in.Activities {
		label[a.ID] = a.Label
	}
	if c, ok := consentLapse(ci); ok {
		list = append(list, c)
	}
	if c, ok := targetCompletion(ci); ok {
		list = append(list, c)
	}
	for _, c := range out.Conflicts {
		list = append(list, Check{Code: CheckMSOConflict, ActivityID: c.ActivityID,
			Message: fmt.Sprintf("%s must start %s but %s holds it until %s.",
				label[c.ActivityID], dayText(cal, c.Index), label[c.HeldBy], dayText(cal, c.HeldIndex))})
	}
	for _, y := range calendarGaps(ci, cal) {
		list = append(list, Check{Code: CheckCalendarGap,
			Message: fmt.Sprintf("No holiday data for %d; weekends only.", y)})
	}
	list = append(list, holdPoints(in)...)
	if ci.SkeletonTie {
		list = append(list, Check{Code: CheckSkeletonTie, Message: "Two program templates fit; using the one with more conditions."})
	}
	for _, id := range ci.RemovedPackageRows {
		list = append(list, Check{Code: CheckPackageRemoved, ActivityID: id, Message: "Package removed in Procurement."})
	}
	return list
}

// dayText is a long date, or "Week 3, day 2" for a relative program.
func dayText(cal *Cal, i int) string {
	if cal.Relative() {
		return cal.DayLabel(i)
	}
	return longDate(cal.DateOf(i))
}

func consentLapse(ci CheckInput) (Check, bool) {
	t, ok := ci.Output.Activities.Get(ci.ConstructionStageID)
	if ci.ConsentLapseDate == "" || !ok || t.Start == "" || t.Start <= ci.ConsentLapseDate {
		return Check{}, false
	}
	return Check{Code: CheckConsentLapse, ActivityID: ci.ConstructionStageID,
		Message: fmt.Sprintf("Construction starts on %s, after consent lapses on %s.", longDate(t.Start), longDate(ci.ConsentLapseDate))}, true
}

// targetCompletion compares the typical finish. When some durations are
// unknown the known finish is a lower bound, so a finish already past the
// target is still a fact.
func targetCompletion(ci CheckInput) (Check, bool) {
	f := ci.Output.Finish.Typical
	if ci.TargetCompletion == "" || f == nil || *f <= ci.TargetCompletion {
		return Check{}, false
	}
	a, _ := dayNumber(*f)
	b, ok := dayNumber(ci.TargetCompletion)
	if !ok {
		return Check{}, false
	}
	return Check{Code: CheckTargetCompletion,
		Message: fmt.Sprintf("Finish on %s is %s after the target completion of %s.", longDate(*f), spanWords(a-b), longDate(ci.TargetCompletion))}, true
}

// spanWords states a gap in calendar days exactly: "4 days", "6 weeks",
// "6 weeks and 3 days".
func spanWords(days int) string {
	plural := func(n int, unit string) string {
		if n == 1 {
			return "1 " + unit
		}
		return strconv.Itoa(n) + " " + unit + "s"
	}
	w, d := days/7, days%7
	switch {
	case w == 0:
		return plural(d, "day")
	case d == 0:
		return plural(w, "week")
	}
	return plural(w, "week") + " and " + plural(d, "day")
}

// calendarGaps lists years the program spans, from its start to its latest
// finish (the high run when there is one), with no calendar rows.
func calendarGaps(ci CheckInput, cal *Cal) []int {
	f := ci.Output.Finish
	last := f.High
	if last == nil {
		last = f.Typical
	}
	if cal.Relative() || last == nil {
		return nil
	}
	covered := map[int]bool{}
	for _, y := range ci.CalendarYears {
		covered[y] = true
	}
	for _, d := range ci.Input.Calendar.Days {
		if y, err := strconv.Atoi(d.Day[:min(4, len(d.Day))]); err == nil {
			covered[y] = true
		}
	}
	from, _ := strconv.Atoi(ci.Input.Calendar.Start[:4])
	to, _ := strconv.Atoi((*last)[:4])
	var gaps []int
	for y := from; y <= to; y++ {
		if !covered[y] {
			gaps = append(gaps, y)
		}
	}
	return gaps
}

// holdPoints flags a hold point missing the link before it or after it.
// Containment in a stage is not a link: a hold point exists to stop one
// piece of work until another is released.
func holdPoints(in Input) []Check {
	hasPred, hasSucc := map[string]bool{}, map[string]bool{}
	for _, l := range in.Links {
		hasSucc[l.Pred] = true
		hasPred[l.Succ] = true
	}
	var ids []string
	for _, a := range in.Activities {
		if a.Kind == KindHoldPoint && (!hasPred[a.ID] || !hasSucc[a.ID]) {
			ids = append(ids, a.ID)
		}
	}
	var list []Check
	for _, id := range ids {
		msg := "Hold point has no successor."
		switch {
		case !hasPred[id] && !hasSucc[id]:
			msg = "Hold point has no predecessor or successor."
		case !hasPred[id]:
			msg = "Hold point has no predecessor."
		}
		list = append(list, Check{Code: CheckHoldPointUnlinked, ActivityID: id, Message: msg})
	}
	return list
}
