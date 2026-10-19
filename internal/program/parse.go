package program

// Typed formats for the grid (plan 9.5) and for program files (plan 10.2,
// harvest step 4). One parser serves both so a value typed in the grid and
// the same value read from a file mean the same thing. web/src/program/grid/
// parse.ts mirrors this file rule for rule; keep each rule a plain regular
// expression or a few lines of arithmetic so the mirror stays exact.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MaxDuration is the schema's limit on duration_days.
const MaxDuration = 3650

// Duration is a parsed duration in working days. A single value has
// Low == High; a range such as "4–6 wks" has Low < High. A range has no
// typical value: the caller decides what to do with it, never this parser.
type Duration struct {
	Low  int `json:"low"`
	High int `json:"high"`
}

// Milestone reports "0": a zero duration makes a milestone.
func (d Duration) Milestone() bool { return d.High == 0 }

var (
	// rangeSplit splits "4–6 wks", "4 - 6 weeks", "4 to 6 wks". A plain
	// hyphen only splits between digits and a later number, which is the
	// only place it can appear (negative durations are refused anyway).
	rangeSplit = regexp.MustCompile(`^\s*([0-9.]+\s*[a-z]*)\s*(?:–|—|-|to)\s*([0-9.]+\s*[a-z]*)\s*$`)
	// durationPart is a number and an optional unit: "5", "5d", "3 wks".
	durationPart = regexp.MustCompile(`^([0-9]+(?:\.[0-9]+)?)\s*([a-z]*)$`)
)

// durationUnit returns working days per unit; "" is days.
func durationUnit(u string, perWeek int) (int, bool) {
	switch u {
	case "", "d", "dy", "dys", "day", "days", "wd", "wds":
		return 1, true
	case "w", "wk", "wks", "week", "weeks":
		return perWeek, true
	}
	return 0, false
}

// ParseDuration reads "5", "5d", "3w", "0", "3 wks", "10 days" and ranges
// such as "4–6 wks". Weeks are working weeks: perWeek working days each.
func ParseDuration(s string, perWeek int) (Duration, error) {
	if perWeek <= 0 {
		perWeek = 5
	}
	t := strings.ToLower(strings.TrimSpace(s))
	if t == "" {
		return Duration{}, fmt.Errorf("Enter a duration, for example 5, 5d or 3w.")
	}
	if m := rangeSplit.FindStringSubmatch(t); m != nil {
		lowText, highText := strings.TrimSpace(m[1]), strings.TrimSpace(m[2])
		// "4–6 wks": the low end borrows the high end's unit.
		if hm := durationPart.FindStringSubmatch(highText); hm != nil {
			if lm := durationPart.FindStringSubmatch(lowText); lm != nil && lm[2] == "" {
				lowText = lm[1] + hm[2]
			}
		}
		low, err := durationValue(lowText, perWeek)
		if err != nil {
			return Duration{}, err
		}
		high, err := durationValue(highText, perWeek)
		if err != nil {
			return Duration{}, err
		}
		if low > high {
			return Duration{}, fmt.Errorf("A range runs from low to high, for example 4–6 wks.")
		}
		return Duration{Low: low, High: high}, nil
	}
	d, err := durationValue(t, perWeek)
	if err != nil {
		return Duration{}, err
	}
	return Duration{Low: d, High: d}, nil
}

func durationValue(t string, perWeek int) (int, error) {
	m := durationPart.FindStringSubmatch(strings.TrimSpace(t))
	if m == nil {
		return 0, fmt.Errorf("%q is not a duration. Use days or weeks, for example 5, 5d or 3w.", t)
	}
	per, ok := durationUnit(m[2], perWeek)
	if !ok {
		return 0, fmt.Errorf("%q is not a unit. Use d for working days or w for weeks.", m[2])
	}
	// Decimals are allowed only when they land on whole working days
	// ("1.5w" in a 4-day week is 6 days); integer arithmetic avoids any
	// floating-point rounding.
	whole, frac, _ := strings.Cut(m[1], ".")
	num, err := strconv.Atoi(whole + frac)
	if err != nil || len(whole+frac) > 6 {
		return 0, fmt.Errorf("Duration must be at most %d working days.", MaxDuration)
	}
	den := 1
	for range frac {
		den *= 10
	}
	if num*per%den != 0 {
		return 0, fmt.Errorf("%s is not a whole number of working days.", strings.TrimSpace(t))
	}
	days := num * per / den
	if days > MaxDuration {
		return 0, fmt.Errorf("Duration must be at most %d working days.", MaxDuration)
	}
	return days, nil
}

// PredRef is one entry of a predecessor cell: a row number as shown in the
// grid (or in a file), a link type, and a lag.
type PredRef struct {
	Row     int    `json:"row"`
	Type    string `json:"type"`
	Lag     int    `json:"lag"`
	LagUnit string `json:"lag_unit"`
}

// predPart is "12", "12FS", "12SS+2d", "8FF-1d", "4SF", "7FS+3cd", "12+2d".
// Lag units: d (working days, the default), w (working weeks), cd or ed
// (calendar days; ed is the "elapsed days" spelling program files use).
var predPart = regexp.MustCompile(`^([0-9]{1,6})\s*(fs|ss|ff|sf)?\s*(?:([+-])\s*([0-9]{1,4})\s*(d|days?|w|wks?|weeks?|cd|ed|edays?)?)?$`)

// ParsePredecessors reads a comma- or semicolon-separated predecessor cell.
// An empty cell is no predecessors.
func ParsePredecessors(s string, perWeek int) ([]PredRef, error) {
	if perWeek <= 0 {
		perWeek = 5
	}
	var refs []PredRef
	seen := map[int]bool{}
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' }) {
		t := strings.ToLower(strings.TrimSpace(part))
		if t == "" {
			continue
		}
		m := predPart.FindStringSubmatch(t)
		if m == nil {
			return nil, fmt.Errorf("%q is not a predecessor. Use a row number with an optional type and lag, for example 12, 12SS+2d or 7FS+3cd.", strings.TrimSpace(part))
		}
		row, err := strconv.Atoi(m[1])
		if err != nil || row < 1 {
			return nil, fmt.Errorf("Row numbers start at 1.")
		}
		if seen[row] {
			return nil, fmt.Errorf("Row %d is listed twice.", row)
		}
		seen[row] = true
		ref := PredRef{Row: row, Type: LinkFS, LagUnit: LagWorking}
		if m[2] != "" {
			ref.Type = strings.ToUpper(m[2])
		}
		if m[3] != "" {
			lag := atoi(m[4])
			switch m[5] {
			case "w", "wk", "wks", "week", "weeks":
				lag *= perWeek
			case "cd", "ed", "eday", "edays":
				ref.LagUnit = LagCalendar
			}
			if m[3] == "-" {
				lag = -lag
			}
			ref.Lag = lag
		}
		if ref.Lag < -365 || ref.Lag > 365 {
			return nil, fmt.Errorf("Lag must be between −365 and 365 days.")
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// FormatPredecessor writes a PredRef back in the grid's typed format, so an
// export re-imports to the same links: "12", "12SS+2d", "7FS+3cd".
func FormatPredecessor(r PredRef) string {
	s := strconv.Itoa(r.Row)
	if r.Type != LinkFS || r.Lag != 0 {
		s += r.Type
	}
	if r.Lag != 0 {
		unit := "d"
		if r.LagUnit == LagCalendar {
			unit = "cd"
		}
		s += fmt.Sprintf("%+d%s", r.Lag, unit)
	}
	return s
}

// DayOrder is how a column writes numeric dates.
type DayOrder int

const (
	DayMonth DayOrder = iota // d/m/yyyy, Australian
	MonthDay                 // m/d/yyyy
)

var (
	// numericDate is d/m/yyyy or d/m/yy with /, . or - between parts.
	numericDate = regexp.MustCompile(`^([0-9]{1,2})[/.-]([0-9]{1,2})[/.-]([0-9]{2}|[0-9]{4})$`)
	isoDate     = regexp.MustCompile(`^([0-9]{4})-([0-9]{1,2})-([0-9]{1,2})$`)
	// namedDate is "12 Mar 2027", "12 March 2027", "12-Mar-27".
	namedDate = regexp.MustCompile(`^([0-9]{1,2})[ -]([a-z]+)\.?,?[ -]([0-9]{2}|[0-9]{4})$`)
	// Program files often prefix a weekday ("Mon 12/03/27") or suffix a
	// time ("12/03/2027 8:00 AM"); neither changes the date.
	weekdayPrefix = regexp.MustCompile(`^(mon|tue|tues|wed|thu|thur|thurs|fri|sat|sun)[a-z]*\.?,?\s+`)
	timeSuffix    = regexp.MustCompile(`\s+[0-9]{1,2}:[0-9]{2}.*$`)
)

var months = map[string]int{
	"jan": 1, "january": 1, "feb": 2, "february": 2, "mar": 3, "march": 3,
	"apr": 4, "april": 4, "may": 5, "jun": 6, "june": 6, "jul": 7, "july": 7,
	"aug": 8, "august": 8, "sep": 9, "sept": 9, "september": 9, "oct": 10,
	"october": 10, "nov": 11, "november": 11, "dec": 12, "december": 12,
}

func cleanDate(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	t = weekdayPrefix.ReplaceAllString(t, "")
	return timeSuffix.ReplaceAllString(t, "")
}

// DecideDayOrder picks one order for a whole column (plan 10.2 step 3): any
// first part over 12 means day/month; otherwise any second part over 12 means
// month/day; otherwise day/month is assumed, and assumed is true when some
// value could be read either way, so the caller adds the "Date order
// assumed" check naming the column.
func DecideDayOrder(values []string) (DayOrder, bool) {
	firstOver, secondOver, ambiguous := false, false, false
	for _, v := range values {
		m := numericDate.FindStringSubmatch(cleanDate(v))
		if m == nil {
			continue
		}
		a, _ := strconv.Atoi(m[1])
		b, _ := strconv.Atoi(m[2])
		switch {
		case a > 12:
			firstOver = true
		case b > 12:
			secondOver = true
		case a != b:
			ambiguous = true
		}
	}
	switch {
	case firstOver:
		return DayMonth, false
	case secondOver:
		return MonthDay, false
	}
	return DayMonth, ambiguous
}

// ParseDate reads d/m/yyyy, d/m/yy, d Mon yyyy and yyyy-mm-dd and returns
// YYYY-MM-DD. order applies to the numeric form only. Two-digit years are
// 2000–2099: a construction program never refers to the 1900s.
func ParseDate(s string, order DayOrder) (string, error) {
	t := cleanDate(s)
	var y, mo, d int
	if m := isoDate.FindStringSubmatch(t); m != nil {
		y, mo, d = atoi(m[1]), atoi(m[2]), atoi(m[3])
	} else if m := numericDate.FindStringSubmatch(t); m != nil {
		d, mo, y = atoi(m[1]), atoi(m[2]), year(m[3])
		if order == MonthDay {
			d, mo = mo, d
		}
	} else if m := namedDate.FindStringSubmatch(t); m != nil {
		n, ok := months[m[2]]
		if !ok {
			return "", fmt.Errorf("%q is not a month.", m[2])
		}
		d, mo, y = atoi(m[1]), n, year(m[3])
	} else {
		return "", fmt.Errorf("%q is not a date. Use d/m/yyyy, d Mon yyyy or yyyy-mm-dd.", strings.TrimSpace(s))
	}
	// time.Date normalises 31 April to 1 May; a date that moved was not real.
	tm := time.Date(y, time.Month(mo), d, 0, 0, 0, 0, time.UTC)
	if tm.Year() != y || int(tm.Month()) != mo || tm.Day() != d {
		return "", fmt.Errorf("%q is not a real date.", strings.TrimSpace(s))
	}
	return tm.Format("2006-01-02"), nil
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func year(s string) int {
	if len(s) == 2 {
		return 2000 + atoi(s)
	}
	return atoi(s)
}
