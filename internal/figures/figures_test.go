package figures

import (
	"slices"
	"testing"
	"time"
)

func TestAmount(t *testing.T) {
	for in, want := range map[string]string{
		"45000": "45000", "$45,000": "45000", "$45k": "45000", "45K": "45000", "1.2m": "1200000", "$1.2k": "1200",
		"49,500.50": "49500.5", "45000.00": "45000", " 8,000 ": "8000", "2 million": "2000000", "-500": "-500",
	} {
		got, ok := Amount(in)
		if !ok || got != want {
			t.Errorf("Amount(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "about 45k", "forty", "45kk", "$", "1,2,3x"} {
		if got, ok := Amount(bad); ok {
			t.Errorf("Amount(%q) = %q, want refused", bad, got)
		}
	}
}

func TestNumbersInAMessage(t *testing.T) {
	got := Numbers("Add five activities, two weeks each. Joinery is $45k, stone 1.2m, area 612.5 m2, 10% and 49,500.")
	for _, want := range []string{"5", "2", "45000", "1200000", "612.5", "10", "49500"} {
		if !slices.Contains(got, want) {
			t.Errorf("missing %s in %v", want, got)
		}
	}
	// "612 m" is metres, not millions: a suffix counts only when attached.
	if slices.Contains(Numbers("the site is 612 m wide"), "612000000") {
		t.Error("a spaced m must not be read as million")
	}
}

func TestDuration(t *testing.T) {
	cases := map[string][2]any{"2 weeks": {2, "week"}, "two weeks": {2, "week"}, "10 days": {10, "day"}, "a fortnight": {2, "week"},
		"3 months": {3, "month"}, "1 week": {1, "week"}, "10d": {10, "day"}, "8w": {8, "week"}, "5": {5, "day"}}
	for in, want := range cases {
		n, unit, ok := Duration(in)
		if !ok || n != want[0] || unit != want[1] {
			t.Errorf("Duration(%q) = %d %s %v", in, n, unit, ok)
		}
	}
	if _, _, ok := Duration("a while"); ok {
		t.Error("a vague duration must be refused")
	}
	if d, ok := WorkingDays("2 weeks", 5); !ok || d != 10 {
		t.Errorf("two weeks on a five-day week = %d", d)
	}
	if d, ok := WorkingDays("2 weeks", 6); !ok || d != 12 {
		t.Errorf("two weeks on a six-day week = %d", d)
	}
	if d, ok := WorkingDays("1 month", 5); !ok || d != 22 {
		t.Errorf("a month on a five-day week = %d", d)
	}
}

func TestBook(t *testing.T) {
	today := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	var b Book
	b.Add("Joinery is $45,000. Start on 30 June. Handover 1/3/2028. Two weeks each.")
	b.Add("Construction | budget 1500000 | finish 2027-08-14")
	for _, n := range []string{"45000", "45000.00", "2", "1500000"} {
		if !b.HasNumber(n) {
			t.Errorf("book should hold %s", n)
		}
	}
	if b.HasNumber("54000") || b.HasNumber("150000") {
		t.Error("book holds a number nobody gave")
	}
	for iso, want := range map[string]bool{
		"2027-06-30": true,  // no year typed: the next 30 June
		"2028-06-30": false, // a year nobody gave
		"2028-03-01": true,  // day/month/year, Australian order
		"2027-08-14": true,  // an ISO date in a tool result
		"2027-07-01": false,
	} {
		if got := b.HasDate(iso, today); got != want {
			t.Errorf("HasDate(%s) = %v, want %v", iso, got, want)
		}
	}
}

func TestBookChecksEveryNumberInAValue(t *testing.T) {
	var b Book
	b.Add("Set the site area to 612 square metres and the zone to R3")
	if !b.HasNumbersOf("612") || !b.HasNumbersOf("R3") || !b.HasNumbersOf("General Residential") {
		t.Error("values the PM typed must pass")
	}
	if b.HasNumbersOf("621") || b.HasNumbersOf("R4") {
		t.Error("a value with a number nobody typed must fail")
	}
}
