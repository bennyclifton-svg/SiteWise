package knowledge

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Security of Payment periods (knowledge/contracts/security_of_payment.yaml;
// docs/plans/2026-11-17-variations-progress-claims.md, Task 1). One row per
// state. A row is used whether or not it is verified; the dates worked out
// from an unverified row are shown "to check" with their section, and its
// excluded spans are left out so a date can only fall early.

// SOPAct is one state's Act as the file records it.
type SOPAct struct {
	ID       string
	State    string
	Act      string
	Excludes SOPExcludes
	// Schedule and Payment are nil when the period is not recorded.
	Schedule *SOPPeriod
	Payment  *SOPPeriod
	Verified bool
	// PrimarySource is the Act's URL once read; SecondarySource names any
	// guidance the figures came from instead.
	PrimarySource   string
	SecondarySource string
	CheckedOn       string
	Status          string
	Notes           string
}

// SOPExcludes are the days an Act's business days leave out.
type SOPExcludes struct {
	Weekdays       []string
	PublicHolidays bool
	// Spans are month-day ranges ("12-27" to "12-31") left out every year.
	Spans []SOPSpan
}

// SOPSpan is one yearly run of excluded days, inclusive.
type SOPSpan struct{ From, To string }

// SOPPeriod is one period in business days after the claim, with the
// section it comes from.
type SOPPeriod struct {
	Section      string
	BusinessDays int
	Rule         string
}

// SOPAct is the state's row, or false when the file has none.
func (c *Catalog) SOPAct(state string) (SOPAct, bool) {
	for _, a := range c.sopActs {
		if a.State == state {
			return a, true
		}
	}
	return SOPAct{}, false
}

// SOPActs are every row in file order.
func (c *Catalog) SOPActs() []SOPAct { return append([]SOPAct(nil), c.sopActs...) }

var (
	sopIDPattern    = regexp.MustCompile(`^sop\.[a-z]+$`)
	monthDayPattern = regexp.MustCompile(`^(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$`)
	sopWeekdays     = map[string]bool{"mon": true, "tue": true, "wed": true, "thu": true, "fri": true, "sat": true, "sun": true}
)

// loadContracts reads the file when it exists. It is not hashed into the
// catalogue's version: no profile, report or Program reads it, so a change
// to it must not mark their caches stale.
func (c *Catalog) loadContracts(root string) error {
	path := filepath.Join(root, "contracts", "security_of_payment.yaml")
	if ok, err := exists(path); !ok {
		return err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	acts, err := parseSOPActs(body)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	c.sopActs = acts
	return nil
}

type sopPeriodFile struct {
	Section      string `yaml:"section"`
	BusinessDays any    `yaml:"business_days"`
	Rule         string `yaml:"rule"`
	Who          string `yaml:"who"`
}

func parseSOPActs(body []byte) ([]SOPAct, error) {
	var file struct {
		Version int    `yaml:"version"`
		Status  string `yaml:"status"`
		Acts    []struct {
			ID       string `yaml:"id"`
			State    string `yaml:"state"`
			Act      string `yaml:"act"`
			Excludes struct {
				Weekdays       []string `yaml:"weekdays"`
				PublicHolidays bool     `yaml:"public_holidays"`
				Spans          []struct {
					From string `yaml:"from"`
					To   string `yaml:"to"`
				} `yaml:"spans"`
			} `yaml:"business_day_excludes"`
			Schedule        *sopPeriodFile  `yaml:"schedule"`
			Payment         *sopPeriodFile  `yaml:"payment"`
			OtherPayments   []sopPeriodFile `yaml:"other_payments"`
			Verified        *bool           `yaml:"verified"`
			PrimarySource   string          `yaml:"primary_source"`
			SecondarySource string          `yaml:"secondary_source"`
			CheckedOn       string          `yaml:"checked_on"`
			Status          string          `yaml:"status"`
			Notes           string          `yaml:"notes"`
		} `yaml:"acts"`
	}
	if err := strictYAML(body, &file); err != nil {
		return nil, err
	}
	if file.Version != 1 || (file.Status != "draft" && file.Status != "reviewed") {
		return nil, fmt.Errorf("needs version 1 and status draft or reviewed")
	}
	seen := map[string]bool{}
	var out []SOPAct
	for _, a := range file.Acts {
		where := a.ID
		if !sopIDPattern.MatchString(a.ID) || a.ID != "sop."+strings.ToLower(a.State) {
			return nil, fmt.Errorf("%s: the id is sop.<state>", where)
		}
		if !inList(CalendarStates, a.State) {
			return nil, fmt.Errorf("%s: unknown state %q", where, a.State)
		}
		if seen[a.State] {
			return nil, fmt.Errorf("%s: two rows for %s", where, a.State)
		}
		seen[a.State] = true
		if strings.TrimSpace(a.Act) == "" || a.Verified == nil || (a.Status != "draft" && a.Status != "reviewed") {
			return nil, fmt.Errorf("%s: needs act, verified and status", where)
		}
		if *a.Verified {
			if u, err := url.Parse(a.PrimarySource); err != nil || u.Scheme != "https" || u.Host == "" {
				return nil, fmt.Errorf("%s: verified needs the Act's URL as primary_source", where)
			}
			if _, err := time.Parse("2006-01-02", a.CheckedOn); err != nil {
				return nil, fmt.Errorf("%s: verified needs checked_on", where)
			}
		}
		if a.CheckedOn != "" {
			if _, err := time.Parse("2006-01-02", a.CheckedOn); err != nil {
				return nil, fmt.Errorf("%s: checked_on is YYYY-MM-DD", where)
			}
		}
		act := SOPAct{ID: a.ID, State: a.State, Act: a.Act, Verified: *a.Verified, PrimarySource: a.PrimarySource,
			SecondarySource: a.SecondarySource, CheckedOn: a.CheckedOn, Status: a.Status, Notes: a.Notes}
		act.Excludes.PublicHolidays = a.Excludes.PublicHolidays
		for _, d := range a.Excludes.Weekdays {
			if !sopWeekdays[d] {
				return nil, fmt.Errorf("%s: unknown weekday %q", where, d)
			}
			act.Excludes.Weekdays = append(act.Excludes.Weekdays, d)
		}
		for _, s := range a.Excludes.Spans {
			if !monthDayPattern.MatchString(s.From) || !monthDayPattern.MatchString(s.To) || s.To < s.From {
				return nil, fmt.Errorf("%s: a span is MM-DD to a later MM-DD", where)
			}
			act.Excludes.Spans = append(act.Excludes.Spans, SOPSpan{From: s.From, To: s.To})
		}
		var err error
		if act.Schedule, err = sopPeriod(where+" schedule", a.Schedule); err != nil {
			return nil, err
		}
		if act.Payment, err = sopPeriod(where+" payment", a.Payment); err != nil {
			return nil, err
		}
		for _, p := range a.OtherPayments {
			if _, err := sopPeriod(where+" other payment", &p); err != nil {
				return nil, err
			}
		}
		out = append(out, act)
	}
	return out, nil
}

// sopPeriod reads one period. The count must be a quoted whole number so
// no YAML reading of it is in doubt.
func sopPeriod(where string, p *sopPeriodFile) (*SOPPeriod, error) {
	if p == nil {
		return nil, nil
	}
	s, quoted := p.BusinessDays.(string)
	n, err := strconv.Atoi(s)
	if !quoted || err != nil || n < 1 || n > 60 {
		return nil, fmt.Errorf("%s: business_days is a quoted whole number from 1 to 60", where)
	}
	if strings.TrimSpace(p.Section) == "" {
		return nil, fmt.Errorf("%s: needs its section", where)
	}
	return &SOPPeriod{Section: p.Section, BusinessDays: n, Rule: p.Rule}, nil
}
