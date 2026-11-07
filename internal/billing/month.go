package billing

import (
	"time"
	// The host may lack zoneinfo (Windows development, minimal images).
	_ "time/tzdata"
)

var sydney = mustLoad("Australia/Sydney")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// MonthStart is the first day of now's calendar month in Sydney, where the
// monthly document allowance resets.
func MonthStart(now time.Time) time.Time {
	t := now.In(sydney)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
