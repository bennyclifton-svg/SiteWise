package program

import (
	"reflect"
	"testing"
)

func TestParseDuration(t *testing.T) {
	ok := []struct {
		in        string
		perWeek   int
		low, high int
	}{
		{"5", 5, 5, 5},
		{"5d", 5, 5, 5},
		{" 5 D ", 5, 5, 5},
		{"10 days", 5, 10, 10},
		{"1 day", 5, 1, 1},
		{"3w", 5, 15, 15},
		{"3w", 6, 18, 18}, // weeks are working weeks of the program's calendar
		{"3 wks", 5, 15, 15},
		{"2 weeks", 4, 8, 8},
		{"1.5w", 4, 6, 6},
		{"0", 5, 0, 0},
		{"4–6 wks", 5, 20, 30},
		{"4-6 weeks", 5, 20, 30},
		{"4 to 6 wks", 5, 20, 30},
		{"2w-15d", 5, 10, 15},
		{"3650", 5, 3650, 3650},
	}
	for _, x := range ok {
		d, err := ParseDuration(x.in, x.perWeek)
		if err != nil || d.Low != x.low || d.High != x.high {
			t.Errorf("%q: %+v, %v; want %d–%d", x.in, d, err, x.low, x.high)
		}
	}
	if d, _ := ParseDuration("0", 5); !d.Milestone() {
		t.Error("0 makes a milestone")
	}
	for _, bad := range []string{"", "abc", "-3", "5x", "1.5d", "1.5", "6–4 wks", "3651", "2 months", "5 edays", "4–"} {
		if d, err := ParseDuration(bad, 5); err == nil {
			t.Errorf("%q accepted as %+v", bad, d)
		}
	}
}

func TestParsePredecessors(t *testing.T) {
	ok := map[string][]PredRef{
		"12":      {{Row: 12, Type: LinkFS, LagUnit: LagWorking}},
		"12FS":    {{Row: 12, Type: LinkFS, LagUnit: LagWorking}},
		"12SS+2d": {{Row: 12, Type: LinkSS, Lag: 2, LagUnit: LagWorking}},
		"8FF-1d":  {{Row: 8, Type: LinkFF, Lag: -1, LagUnit: LagWorking}},
		"4SF":     {{Row: 4, Type: LinkSF, LagUnit: LagWorking}},
		"7FS+3cd": {{Row: 7, Type: LinkFS, Lag: 3, LagUnit: LagCalendar}},
		"7fs+3ed": {{Row: 7, Type: LinkFS, Lag: 3, LagUnit: LagCalendar}},
		"5+1w":    {{Row: 5, Type: LinkFS, Lag: 5, LagUnit: LagWorking}},
		"3ss + 2": {{Row: 3, Type: LinkSS, Lag: 2, LagUnit: LagWorking}},
		"12, 8FF-1d; 4SF": {
			{Row: 12, Type: LinkFS, LagUnit: LagWorking},
			{Row: 8, Type: LinkFF, Lag: -1, LagUnit: LagWorking},
			{Row: 4, Type: LinkSF, LagUnit: LagWorking},
		},
		"":    nil,
		" , ": nil,
	}
	for in, want := range ok {
		got, err := ParsePredecessors(in, 5)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, bad := range []string{"0", "12XX", "12FS+", "a", "12,12FS", "1FS+400d", "1FS+80w", "12FS+2x", "-3"} {
		if got, err := ParsePredecessors(bad, 5); err == nil {
			t.Errorf("%q accepted as %+v", bad, got)
		}
	}
}

// What export writes, import must read back as the same link.
func TestFormatPredecessorRoundTrip(t *testing.T) {
	for _, r := range []PredRef{
		{Row: 12, Type: LinkFS, LagUnit: LagWorking},
		{Row: 12, Type: LinkSS, Lag: 2, LagUnit: LagWorking},
		{Row: 8, Type: LinkFF, Lag: -1, LagUnit: LagWorking},
		{Row: 7, Type: LinkFS, Lag: 3, LagUnit: LagCalendar},
		{Row: 4, Type: LinkSF, LagUnit: LagWorking},
	} {
		got, err := ParsePredecessors(FormatPredecessor(r), 5)
		if err != nil || len(got) != 1 || got[0] != r {
			t.Errorf("%+v → %q → %+v, %v", r, FormatPredecessor(r), got, err)
		}
	}
}

func TestParseDate(t *testing.T) {
	ok := []struct {
		in    string
		order DayOrder
		want  string
	}{
		{"1/3/2027", DayMonth, "2027-03-01"},
		{"1/3/2027", MonthDay, "2027-01-03"},
		{"1/3/27", DayMonth, "2027-03-01"},
		{"01.03.2027", DayMonth, "2027-03-01"},
		{"12 Mar 2027", DayMonth, "2027-03-12"},
		{"12 March 2027", MonthDay, "2027-03-12"}, // order applies to numeric dates only
		{"12-Mar-27", DayMonth, "2027-03-12"},
		{"2027-03-12", MonthDay, "2027-03-12"},
		{"Mon 1/03/27", DayMonth, "2027-03-01"},
		{"Monday, 1 March 2027", DayMonth, "2027-03-01"},
		{"1/03/2027 8:00 AM", DayMonth, "2027-03-01"},
		{"29/2/2028", DayMonth, "2028-02-29"},
	}
	for _, x := range ok {
		if got, err := ParseDate(x.in, x.order); err != nil || got != x.want {
			t.Errorf("%q: %q, %v; want %s", x.in, got, err, x.want)
		}
	}
	for _, bad := range []string{"", "tomorrow", "31/4/2027", "29/2/2027", "12 Foo 2027", "13/13/2027", "2027-02-30", "1/3"} {
		if got, err := ParseDate(bad, DayMonth); err == nil {
			t.Errorf("%q accepted as %s", bad, got)
		}
	}
}

func TestDecideDayOrder(t *testing.T) {
	cases := []struct {
		values  []string
		order   DayOrder
		assumed bool
	}{
		{[]string{"13/2/2027", "1/3/2027"}, DayMonth, false},
		{[]string{"2/13/2027", "3/1/2027"}, MonthDay, false},
		{[]string{"1/3/2027", "5/5/2027"}, DayMonth, true},
		{[]string{"5/5/2027", "2027-01-02", "12 Mar 2027", "TBA"}, DayMonth, false},
		{[]string{"13/2/2027", "2/13/2027"}, DayMonth, false}, // a first part over 12 decides first
		{nil, DayMonth, false},
	}
	for _, c := range cases {
		order, assumed := DecideDayOrder(c.values)
		if order != c.order || assumed != c.assumed {
			t.Errorf("%v: %v, %v; want %v, %v", c.values, order, assumed, c.order, c.assumed)
		}
	}
}
