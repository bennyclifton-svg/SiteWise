package knowledge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// The program library (SCHEMA.md "Program library (2026-10-18)") is the only
// place generation may take a stage pattern, an activity or a duration from.
// Draft rows load and validate so the checker and this loader agree, but
// nothing here hands a draft duration to generation.

// OtherStageID is the catalogue stage a user-made Program stage maps to until
// the user picks the stage it really is. It is never a library stage.
const OtherStageID = "other"

// GenericSkeletonID is the fallback skeleton with no applies_when.
const GenericSkeletonID = "sk.generic"

var (
	skeletonIDPattern = regexp.MustCompile(`^sk\.[a-z0-9-]+$`)
	fragmentIDPattern = regexp.MustCompile(`^fr\.[a-z0-9-]+$`)
	durationIDPattern = regexp.MustCompile(`^du\.[a-z0-9-]+$`)
	calendarIDPattern = regexp.MustCompile(`^cal\.[a-z0-9-]+$`)
	activityKeyRe     = regexp.MustCompile(`^[a-z0-9-]+$`)
	// Library amounts are days with at most two decimal places; time-cost
	// parameters keep up to four so a published K and B are not re-rounded.
	dayDecimal   = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,2})?$`)
	paramDecimal = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,4})?$`)
)

// ProgramLinkTypes are the four precedence types.
var ProgramLinkTypes = []string{"FS", "SS", "FF", "SF"}

// FragmentActions are the work-item actions a `work_action` predicate may
// name. retain brings no activities and investigate brings one Investigation
// activity from generation, so neither selects a fragment.
var FragmentActions = []string{"new", "replace", "upgrade", "alter", "repair", "remove"}

// DurationBases are the ways a duration row turns inputs into days.
var DurationBases = []string{"fixed", "per_unit", "floor_cycle", "time_cost", "statutory", "lead_time"}

// CalendarDayKinds is the closed set of non-working day kinds.
var CalendarDayKinds = []string{"public_holiday", "rdo", "shutdown"}

// CalendarStates are the states and territories a calendar may cover.
var CalendarStates = []string{"NSW", "VIC", "QLD", "SA", "WA", "TAS", "ACT", "NT"}

var weekdays = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

// ProgramLink is a precedence link. Lag is nil (none) or a duration id; a
// literal number would be a duration not in the library.
type ProgramLink struct {
	From string  `yaml:"from" json:"from"`
	To   string  `yaml:"to" json:"to"`
	Type string  `yaml:"type" json:"type"`
	Lag  *string `yaml:"lag" json:"lag"`
}

// Skeleton picks the stages a program has and how they link.
type Skeleton struct {
	ID          string              `yaml:"id" json:"id"`
	Label       string              `yaml:"label" json:"label"`
	AppliesWhen any                 `yaml:"applies_when" json:"applies_when,omitempty"`
	Stages      []string            `yaml:"stages" json:"stages"`
	Links       []ProgramLink       `yaml:"links" json:"links"`
	Status      string              `yaml:"status" json:"status"`
	Sources     []map[string]string `yaml:"sources" json:"sources"`
	Notes       string              `yaml:"notes" json:"-"`
}

// FragmentActivity is one activity a fragment brings. Duration is nil when
// unknown.
type FragmentActivity struct {
	Key      string   `yaml:"key" json:"key"`
	Label    string   `yaml:"label" json:"label"`
	Systems  []string `yaml:"systems" json:"systems"`
	Duration *string  `yaml:"duration" json:"duration"`
}

// FragmentAttach places a fragment in its stage: after and before are
// stage_start, stage_finish or another fragment id, each an FS link, no lag.
type FragmentAttach struct {
	After  string `yaml:"after" json:"after,omitempty"`
	Before string `yaml:"before" json:"before,omitempty"`
}

// Fragment is the set of activities a system's work items bring to a stage.
type Fragment struct {
	ID          string              `yaml:"id" json:"id"`
	Label       string              `yaml:"label" json:"label"`
	Stage       string              `yaml:"stage" json:"stage"`
	Systems     []string            `yaml:"systems" json:"systems"`
	AppliesWhen any                 `yaml:"applies_when" json:"applies_when,omitempty"`
	Activities  []FragmentActivity  `yaml:"activities" json:"activities"`
	Links       []ProgramLink       `yaml:"links" json:"links"`
	Attach      *FragmentAttach     `yaml:"attach" json:"attach,omitempty"`
	Status      string              `yaml:"status" json:"status"`
	Sources     []map[string]string `yaml:"sources" json:"sources"`
	Notes       string              `yaml:"notes" json:"-"`
}

// DurationDriver names the Profile value a scaled duration reads: exactly
// one of a planning key or a determinant.
type DurationDriver struct {
	PlanningKey string `yaml:"planning_key" json:"planning_key,omitempty"`
	Determinant string `yaml:"determinant" json:"determinant,omitempty"`
	Unit        string `yaml:"unit" json:"unit"`
}

// Duration is one library duration. Amounts are decimal strings so the
// file, the checker and Go read the same number.
type Duration struct {
	ID       string          `yaml:"id" json:"id"`
	Version  int             `yaml:"version" json:"version"`
	Basis    string          `yaml:"basis" json:"basis"`
	Driver   *DurationDriver `yaml:"driver" json:"driver,omitempty"`
	UnitDays string          `yaml:"unit_days" json:"unit_days"`
	Low      string          `yaml:"low" json:"low,omitempty"`
	Typical  string          `yaml:"typical" json:"typical,omitempty"`
	High     string          `yaml:"high" json:"high,omitempty"`
	// time_cost only: T = K × C^B with C indexed to CostBaseDate.
	K            string `yaml:"k" json:"k,omitempty"`
	B            string `yaml:"b" json:"b,omitempty"`
	CostBaseDate string `yaml:"cost_base_date" json:"cost_base_date,omitempty"`
	IndexBasis   string `yaml:"index_basis" json:"index_basis,omitempty"`
	// statutory only. Verified stays false until checked against the
	// instrument itself.
	Instrument    string              `yaml:"instrument" json:"instrument,omitempty"`
	Clause        string              `yaml:"clause" json:"clause,omitempty"`
	Verified      *bool               `yaml:"verified" json:"verified,omitempty"`
	PrimarySource string              `yaml:"primary_source" json:"primary_source,omitempty"`
	AppliesWhen   any                 `yaml:"applies_when" json:"applies_when,omitempty"`
	Geography     string              `yaml:"geography" json:"geography"`
	Limitations   []string            `yaml:"limitations" json:"limitations"`
	Status        string              `yaml:"status" json:"status"`
	Sources       []map[string]string `yaml:"sources" json:"sources"`
	Notes         string              `yaml:"notes" json:"-"`
}

// CalendarDay is one non-working day (Date) or span (From to To).
type CalendarDay struct {
	Date  string `yaml:"date" json:"date,omitempty"`
	From  string `yaml:"from" json:"from,omitempty"`
	To    string `yaml:"to" json:"to,omitempty"`
	Kind  string `yaml:"kind" json:"kind"`
	Label string `yaml:"label" json:"label"`
}

// Calendar is the non-working days of one state in one year.
type Calendar struct {
	ID          string              `yaml:"id" json:"id"`
	State       string              `yaml:"state" json:"state"`
	Year        int                 `yaml:"year" json:"year"`
	WorkingWeek []string            `yaml:"working_week" json:"working_week"`
	Days        []CalendarDay       `yaml:"days" json:"days"`
	Verified    *bool               `yaml:"verified" json:"verified"`
	Status      string              `yaml:"status" json:"status"`
	Sources     []map[string]string `yaml:"sources" json:"sources"`
	Notes       string              `yaml:"notes" json:"-"`
}

type programData struct {
	skeletons []Skeleton
	fragments []Fragment
	durations map[string]Duration
	calendars []Calendar
	version   string
}

// Skeletons returns every skeleton in file order.
func (c *Catalog) Skeletons() []Skeleton { return append([]Skeleton(nil), c.program.skeletons...) }

// Fragments returns every fragment in file order. Generation still evaluates
// each applies_when; a fragment's draft status is shown, not hidden.
func (c *Catalog) Fragments() []Fragment { return append([]Fragment(nil), c.program.fragments...) }

// Calendars returns every calendar in file order.
func (c *Catalog) Calendars() []Calendar { return append([]Calendar(nil), c.program.calendars...) }

// LibraryVersion hashes the knowledge/program files parsed at Load, so a
// program records which library generated it. Empty when there is no library.
func (c *Catalog) LibraryVersion() string { return c.program.version }

// ReviewedDuration returns a duration row only when the owner has reviewed
// it. A draft row is loaded and validated but never applied (plan 4.3).
func (c *Catalog) ReviewedDuration(id string) (Duration, bool) {
	d, ok := c.program.durations[id]
	if !ok || d.Status != statusReviewed {
		return Duration{}, false
	}
	return d, true
}

// DurationApplies is true only for a reviewed row whose applies_when holds.
// Unknown applicability is not applicability: the duration stays unknown.
func (c *Catalog) DurationApplies(d Duration, env WorksEnv) bool {
	if d.Status != statusReviewed {
		return false
	}
	return d.AppliesWhen == nil || c.Holds(d.AppliesWhen, env) == True
}

// DurationInput is the driver value code read from the Profile or the Cost
// Plan: units for per_unit, storeys for floor_cycle, and for time_cost the
// construction cost already indexed to the row's cost_base_date. A single
// value sets all three; a range sets Low and High and leaves Typical nil.
type DurationInput struct {
	Low, Typical, High *big.Rat
}

// ExactInput is a single known driver value.
func ExactInput(v *big.Rat) DurationInput { return DurationInput{Low: v, Typical: v, High: v} }

// DayRange is a computed duration in whole days of the row's unit_days.
// TypicalKnown is false when the driver was a range with no typical value.
type DayRange struct {
	Low, Typical, High int64
	TypicalKnown       bool
}

// ComputeDuration turns a reviewed row into whole days, rounding each bound
// up. ok is false for a draft row, a missing or negative input, or a number
// too large for days; the caller then leaves the duration unknown.
func ComputeDuration(d Duration, in DurationInput) (DayRange, bool) {
	if d.Status != statusReviewed {
		return DayRange{}, false
	}
	switch d.Basis {
	case "fixed", "statutory", "lead_time":
		low, ok1 := ceilDecimal(d.Low, big.NewRat(1, 1))
		typ, ok2 := ceilDecimal(d.Typical, big.NewRat(1, 1))
		high, ok3 := ceilDecimal(d.High, big.NewRat(1, 1))
		return DayRange{Low: low, Typical: typ, High: high, TypicalKnown: true}, ok1 && ok2 && ok3
	case "per_unit", "floor_cycle":
		// math/big keeps a rate × quantity exact, so a product that is a
		// whole number of days is never pushed up a day by float error.
		if !rangeOK(in) {
			return DayRange{}, false
		}
		low, ok1 := ceilDecimal(d.Low, in.Low)
		high, ok2 := ceilDecimal(d.High, in.High)
		out := DayRange{Low: low, High: high}
		if in.Typical != nil {
			typ, ok := ceilDecimal(d.Typical, in.Typical)
			if !ok {
				return DayRange{}, false
			}
			out.Typical, out.TypicalKnown = typ, true
		}
		return out, ok1 && ok2
	case "time_cost":
		if !rangeOK(in) || in.Low.Sign() <= 0 {
			return DayRange{}, false
		}
		low, ok1 := timeCost(d, in.Low)
		high, ok2 := timeCost(d, in.High)
		out := DayRange{Low: low, High: high}
		if in.Typical != nil {
			typ, ok := timeCost(d, in.Typical)
			if !ok {
				return DayRange{}, false
			}
			out.Typical, out.TypicalKnown = typ, true
		}
		return out, ok1 && ok2
	}
	return DayRange{}, false
}

func rangeOK(in DurationInput) bool {
	if in.Low == nil || in.High == nil || in.Low.Sign() < 0 || in.Low.Cmp(in.High) > 0 {
		return false
	}
	if in.Typical != nil && (in.Typical.Cmp(in.Low) < 0 || in.Typical.Cmp(in.High) > 0) {
		return false
	}
	return true
}

// ceilDecimal returns ceil(rate × qty) as whole days.
func ceilDecimal(rate string, qty *big.Rat) (int64, bool) {
	r, ok := new(big.Rat).SetString(rate)
	if !ok || qty == nil || r.Sign() < 0 || qty.Sign() < 0 {
		return 0, false
	}
	return ceilRat(new(big.Rat).Mul(r, qty))
}

func ceilRat(v *big.Rat) (int64, bool) {
	q, m := new(big.Int).QuoRem(v.Num(), v.Denom(), new(big.Int))
	if m.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, false
	}
	return q.Int64(), true
}

// timeCost is Bromilow's T = K × C^B. A fractional power has no exact
// rational form, so this is the one place float64 is used; the result is
// rounded up to whole days at the end like every other bound.
func timeCost(d Duration, cost *big.Rat) (int64, bool) {
	k, ok1 := new(big.Rat).SetString(d.K)
	b, ok2 := new(big.Rat).SetString(d.B)
	if !ok1 || !ok2 {
		return 0, false
	}
	kf, _ := k.Float64()
	bf, _ := b.Float64()
	cf, _ := cost.Float64()
	days := math.Ceil(kf * math.Pow(cf, bf))
	if math.IsNaN(days) || math.IsInf(days, 0) || days < 0 || days > math.MaxInt64/2 {
		return 0, false
	}
	return int64(days), true
}

// SkeletonChoice is the skeleton generation applies and the other matching
// skeletons with the same number of terms, which generation reports as a
// check message.
type SkeletonChoice struct {
	Skeleton Skeleton
	Terms    int
	Ties     []string
}

// MatchSkeleton picks the matching skeleton with the most predicate terms.
// A predicate must hold, not merely be unknown: an unknown determinant or
// condition never selects a skeleton. With no match it falls back to
// sk.generic; ok is false only when no library is loaded.
func (c *Catalog) MatchSkeleton(env WorksEnv) (SkeletonChoice, bool) {
	var best SkeletonChoice
	found := false
	var generic *Skeleton
	for i := range c.program.skeletons {
		s := c.program.skeletons[i]
		if s.ID == GenericSkeletonID {
			generic = &c.program.skeletons[i]
			continue
		}
		if c.Holds(s.AppliesWhen, env) != True {
			continue
		}
		terms := PredicateTerms(s.AppliesWhen)
		switch {
		case !found || terms > best.Terms:
			best = SkeletonChoice{Skeleton: s, Terms: terms}
			found = true
		case terms == best.Terms:
			best.Ties = append(best.Ties, s.ID)
		}
	}
	if found {
		return best, true
	}
	if generic == nil {
		return SkeletonChoice{}, false
	}
	return SkeletonChoice{Skeleton: *generic}, true
}

// PredicateTerms counts the leaf conditions in a predicate (a determinant or
// condition comparison, a system test, a works or work_action clause).
func PredicateTerms(p any) int {
	if m, ok := asMap(p); ok {
		if _, det := m["det"]; det {
			return 1
		}
		if _, cond := m["cond"]; cond {
			return 1
		}
		n := 0
		for key, val := range m {
			switch key {
			case "all", "any":
				for _, sub := range asList(val) {
					n += PredicateTerms(sub)
				}
			case "not":
				n += PredicateTerms(val)
			default:
				n++
			}
		}
		return n
	}
	return 0
}

// SamePredicate reports whether two predicates are the same once list order
// inside all, any and any_of is ignored. Two skeletons with the same
// predicate could never be told apart, so the loader refuses them.
func SamePredicate(a, b any) bool {
	return canonicalPredicate(a) == canonicalPredicate(b)
}

func canonicalPredicate(p any) string {
	raw, _ := json.Marshal(canonicalValue(p))
	return string(raw)
}

func canonicalValue(p any) any {
	if m, ok := asMap(p); ok {
		out := make(map[string]any, len(m))
		for k, v := range m {
			out[k] = canonicalValue(v)
		}
		return out
	}
	if list, ok := p.([]any); ok {
		parts := make([]string, len(list))
		for i, v := range list {
			raw, _ := json.Marshal(canonicalValue(v))
			parts[i] = string(raw)
		}
		sort.Strings(parts)
		out := make([]any, len(parts))
		for i, s := range parts {
			out[i] = json.RawMessage(s)
		}
		return out
	}
	return p
}

// loadProgram reads knowledge/program. The directory is optional so fixtures
// without a library still load; a file that is present must be valid.
func (c *Catalog) loadProgram(root string) error {
	dir := filepath.Join(root, "program")
	c.program = programData{durations: map[string]Duration{}}
	if err := c.loadDurations(filepath.Join(dir, "durations.yaml")); err != nil {
		return err
	}
	if err := c.loadFragments(filepath.Join(dir, "fragments.yaml")); err != nil {
		return err
	}
	if err := c.loadSkeletons(filepath.Join(dir, "skeletons.yaml")); err != nil {
		return err
	}
	if err := c.loadCalendars(filepath.Join(dir, "calendars.yaml")); err != nil {
		return err
	}
	c.program.version = c.hashLoadedUnder("program/")
	return nil
}

// programFile holds the common fields of every program library file.
type programFile struct {
	Version int                 `yaml:"version"`
	Status  string              `yaml:"status"`
	Sources []map[string]string `yaml:"sources"`
}

// decodeProgram decodes strictly: an unknown or misspelt key in a library
// row (applies_whne) would otherwise load as a row that always applies.
// Keys in mustQuote must be quoted strings, so a date or decimal is read
// the same way by YAML 1.1 (the checker) and YAML 1.2 (this loader).
func (c *Catalog) decodeProgram(path string, dest any, mustQuote ...string) (bool, error) {
	if ok, err := exists(path); !ok {
		return false, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if err := c.remember(path, body); err != nil {
		return false, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	quote := map[string]bool{}
	for _, k := range mustQuote {
		quote[k] = true
	}
	if err := requireQuoted(&doc, quote); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(body))
	dec.KnownFields(true)
	if err := dec.Decode(dest); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	return true, nil
}

func requireQuoted(n *yaml.Node, keys map[string]bool) error {
	if n.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			if keys[k.Value] && v.Kind == yaml.ScalarNode && v.Tag != "!!str" && v.Tag != "!!null" {
				return fmt.Errorf("%s must be a quoted string, got %q", k.Value, v.Value)
			}
		}
	}
	for _, child := range n.Content {
		if err := requireQuoted(child, keys); err != nil {
			return err
		}
	}
	return nil
}

func checkProgramFile(path string, f programFile) error {
	if f.Version != 1 {
		return fmt.Errorf("%s: version %d", path, f.Version)
	}
	if !programStatus(f.Status) || len(f.Sources) == 0 {
		return fmt.Errorf("%s: needs status and sources", path)
	}
	return checkProgramSources(f.Sources)
}

func programStatus(s string) bool { return s == "draft" || s == statusReviewed }

func checkProgramSources(sources []map[string]string) error {
	if len(sources) == 0 {
		return fmt.Errorf("sources are required")
	}
	for _, source := range sources {
		valid := false
		for _, key := range []string{"design", "clerk_file", "document", "dataset", "seed"} {
			if strings.TrimSpace(source[key]) == "" {
				continue
			}
			valid = true
			if (key == "document" || key == "seed" || key == "design") && strings.TrimSpace(source["anchor"]) == "" {
				return fmt.Errorf("%s source needs an anchor", key)
			}
		}
		if !valid {
			return fmt.Errorf("unrecognised source")
		}
	}
	return nil
}

func (c *Catalog) loadDurations(path string) error {
	var file struct {
		programFile `yaml:",inline"`
		Durations   []Duration `yaml:"durations"`
	}
	ok, err := c.decodeProgram(path, &file, "low", "typical", "high", "k", "b", "cost_base_date")
	if !ok || err != nil {
		return err
	}
	if err := checkProgramFile(path, file.programFile); err != nil {
		return err
	}
	for _, d := range file.Durations {
		if err := c.validateDuration(d); err != nil {
			return fmt.Errorf("%s: %s: %w", path, d.ID, err)
		}
		if _, dup := c.program.durations[d.ID]; dup {
			return fmt.Errorf("%s: duplicate duration %s", path, d.ID)
		}
		c.program.durations[d.ID] = d
	}
	return nil
}

func (c *Catalog) validateDuration(d Duration) error {
	if !durationIDPattern.MatchString(d.ID) {
		return fmt.Errorf("id must match %s", durationIDPattern)
	}
	if d.Version < 1 || !inList(DurationBases, d.Basis) || (d.UnitDays != "working" && d.UnitDays != "calendar") {
		return fmt.Errorf("needs version, a known basis and unit_days working or calendar")
	}
	if !programStatus(d.Status) || strings.TrimSpace(d.Geography) == "" || d.Limitations == nil {
		return fmt.Errorf("needs status, geography and limitations")
	}
	if err := checkProgramSources(d.Sources); err != nil {
		return err
	}
	scaled := d.Basis == "per_unit" || d.Basis == "floor_cycle"
	if scaled != (d.Driver != nil) {
		return fmt.Errorf("driver is required for per_unit and floor_cycle and omitted otherwise")
	}
	if d.Driver != nil {
		if (d.Driver.PlanningKey == "") == (d.Driver.Determinant == "") || strings.TrimSpace(d.Driver.Unit) == "" {
			return fmt.Errorf("driver needs exactly one of planning_key or determinant, and a unit")
		}
		if d.Driver.PlanningKey != "" {
			if _, ok := c.PlanningKey(d.Driver.PlanningKey); !ok {
				return fmt.Errorf("unknown planning key %s", d.Driver.PlanningKey)
			}
		}
		if d.Driver.Determinant != "" {
			if _, ok := c.Determinant(d.Driver.Determinant); !ok {
				return fmt.Errorf("unknown determinant %s", d.Driver.Determinant)
			}
		}
	}
	timeCostFields := d.K != "" || d.B != "" || d.CostBaseDate != "" || d.IndexBasis != ""
	if d.Basis == "time_cost" {
		if d.Low != "" || d.Typical != "" || d.High != "" {
			return fmt.Errorf("time_cost rows carry k and b, not low, typical and high")
		}
		if !paramDecimal.MatchString(d.K) || !paramDecimal.MatchString(d.B) || strings.TrimSpace(d.IndexBasis) == "" {
			return fmt.Errorf("time_cost needs decimal-string k and b and an index_basis")
		}
		if _, err := time.Parse("2006-01-02", d.CostBaseDate); err != nil {
			return fmt.Errorf("cost_base_date must be YYYY-MM-DD")
		}
	} else {
		if timeCostFields {
			return fmt.Errorf("k, b, cost_base_date and index_basis belong to time_cost rows")
		}
		if err := checkDayRange(d.Low, d.Typical, d.High); err != nil {
			return err
		}
	}
	if d.Basis == "statutory" {
		if strings.TrimSpace(d.Instrument) == "" || strings.TrimSpace(d.Clause) == "" || d.Verified == nil {
			return fmt.Errorf("statutory rows carry instrument, clause and verified")
		}
		if *d.Verified && strings.TrimSpace(d.PrimarySource) == "" {
			return fmt.Errorf("a verified statutory row needs primary_source")
		}
	}
	if d.AppliesWhen != nil {
		if err := c.validateProgramPredicate(d.AppliesWhen, 0); err != nil {
			return err
		}
	}
	return nil
}

// checkDayRange: non-negative decimal strings, at most 2 dp, low ≤ typical ≤ high.
func checkDayRange(low, typical, high string) error {
	vals := make([]*big.Rat, 3)
	for i, s := range []string{low, typical, high} {
		if !dayDecimal.MatchString(s) {
			return fmt.Errorf("low, typical and high must be non-negative decimal strings with at most 2 decimal places")
		}
		vals[i], _ = new(big.Rat).SetString(s)
	}
	if vals[0].Cmp(vals[1]) > 0 || vals[1].Cmp(vals[2]) > 0 {
		return fmt.Errorf("needs low ≤ typical ≤ high")
	}
	return nil
}

func (c *Catalog) loadFragments(path string) error {
	var file struct {
		programFile `yaml:",inline"`
		Fragments   []Fragment `yaml:"fragments"`
	}
	ok, err := c.decodeProgram(path, &file)
	if !ok || err != nil {
		return err
	}
	if err := checkProgramFile(path, file.programFile); err != nil {
		return err
	}
	ids := map[string]bool{}
	for _, f := range file.Fragments {
		if !fragmentIDPattern.MatchString(f.ID) || ids[f.ID] {
			return fmt.Errorf("%s: invalid or duplicate fragment id %q", path, f.ID)
		}
		ids[f.ID] = true
	}
	for _, f := range file.Fragments {
		if err := c.validateFragment(f, ids); err != nil {
			return fmt.Errorf("%s: %s: %w", path, f.ID, err)
		}
	}
	if fragmentAttachCycle(file.Fragments) {
		return fmt.Errorf("%s: fragment attach links form a cycle", path)
	}
	c.program.fragments = file.Fragments
	return nil
}

func (c *Catalog) validateFragment(f Fragment, fragments map[string]bool) error {
	if strings.TrimSpace(f.Label) == "" || !programStatus(f.Status) {
		return fmt.Errorf("needs label and status")
	}
	if err := checkProgramSources(f.Sources); err != nil {
		return err
	}
	if err := c.checkLibraryStage(f.Stage); err != nil {
		return err
	}
	if err := c.checkSystemIDs(f.Systems); err != nil {
		return err
	}
	if f.AppliesWhen != nil {
		if err := c.validateProgramPredicate(f.AppliesWhen, 0); err != nil {
			return err
		}
	}
	if len(f.Activities) == 0 {
		return fmt.Errorf("needs activities")
	}
	keys := []string{}
	seen := map[string]bool{}
	for _, a := range f.Activities {
		if !activityKeyRe.MatchString(a.Key) || seen[a.Key] || strings.TrimSpace(a.Label) == "" {
			return fmt.Errorf("activity needs a unique key and a label: %q", a.Key)
		}
		seen[a.Key] = true
		keys = append(keys, a.Key)
		if len(a.Systems) == 0 {
			return fmt.Errorf("activity %s needs systems", a.Key)
		}
		if err := c.checkSystemIDs(a.Systems); err != nil {
			return err
		}
		if a.Duration != nil {
			if err := c.checkDurationRef(*a.Duration); err != nil {
				return err
			}
		}
	}
	if err := c.checkLinks(keys, f.Links); err != nil {
		return err
	}
	if f.Attach != nil {
		if f.Attach.After == "" && f.Attach.Before == "" {
			return fmt.Errorf("attach needs after or before")
		}
		for _, end := range []string{f.Attach.After, f.Attach.Before} {
			if end == "" || end == "stage_start" || end == "stage_finish" {
				continue
			}
			if !fragments[end] || end == f.ID {
				return fmt.Errorf("attach names unknown fragment %s", end)
			}
		}
	}
	return nil
}

func (c *Catalog) checkSystemIDs(ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("needs systems")
	}
	for _, id := range ids {
		if _, ok := c.systems[id]; !ok {
			return fmt.Errorf("unknown system %s", id)
		}
	}
	return nil
}

// checkLibraryStage refuses `other`: a library row belongs to a real stage.
func (c *Catalog) checkLibraryStage(id string) error {
	if id == OtherStageID {
		return fmt.Errorf("library rows never use the %s stage", OtherStageID)
	}
	if _, ok := c.DeliveryStage(id); !ok {
		return fmt.Errorf("unknown stage %s", id)
	}
	return nil
}

func (c *Catalog) checkDurationRef(id string) error {
	if _, ok := c.program.durations[id]; !ok {
		return fmt.Errorf("unknown duration %s", id)
	}
	return nil
}

func (c *Catalog) checkLinks(nodes []string, links []ProgramLink) error {
	in := map[string]bool{}
	for _, n := range nodes {
		in[n] = true
	}
	for _, l := range links {
		if !in[l.From] || !in[l.To] || l.From == l.To {
			return fmt.Errorf("link %s -> %s must join two different listed ids", l.From, l.To)
		}
		if !inList(ProgramLinkTypes, l.Type) {
			return fmt.Errorf("link type must be FS, SS, FF or SF")
		}
		if l.Lag != nil {
			if err := c.checkDurationRef(*l.Lag); err != nil {
				return fmt.Errorf("lag: %w", err)
			}
		}
	}
	if linkCycle(nodes, links) {
		return fmt.Errorf("links form a cycle")
	}
	return nil
}

// linkCycle runs Kahn's algorithm: any node left with an incoming edge after
// peeling sources sits on a cycle.
func linkCycle(nodes []string, links []ProgramLink) bool {
	indeg := map[string]int{}
	out := map[string][]string{}
	for _, n := range nodes {
		indeg[n] += 0
	}
	for _, l := range links {
		out[l.From] = append(out[l.From], l.To)
		indeg[l.To]++
	}
	queue := []string{}
	for n, d := range indeg {
		if d == 0 {
			queue = append(queue, n)
		}
	}
	done := 0
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		done++
		for _, m := range out[n] {
			indeg[m]--
			if indeg[m] == 0 {
				queue = append(queue, m)
			}
		}
	}
	return done != len(indeg)
}

// fragmentAttachCycle: "fr.a after fr.b" and "fr.b after fr.a" could never
// be placed, so attach references between fragments must be acyclic too.
func fragmentAttachCycle(fragments []Fragment) bool {
	var nodes []string
	var links []ProgramLink
	for _, f := range fragments {
		nodes = append(nodes, f.ID)
		if f.Attach == nil {
			continue
		}
		if strings.HasPrefix(f.Attach.After, "fr.") {
			links = append(links, ProgramLink{From: f.Attach.After, To: f.ID})
		}
		if strings.HasPrefix(f.Attach.Before, "fr.") {
			links = append(links, ProgramLink{From: f.ID, To: f.Attach.Before})
		}
	}
	return linkCycle(nodes, links)
}

func (c *Catalog) loadSkeletons(path string) error {
	var file struct {
		programFile `yaml:",inline"`
		Skeletons   []Skeleton `yaml:"skeletons"`
	}
	ok, err := c.decodeProgram(path, &file)
	if !ok || err != nil {
		return err
	}
	if err := checkProgramFile(path, file.programFile); err != nil {
		return err
	}
	ids := map[string]bool{}
	generic := 0
	for i, s := range file.Skeletons {
		if !skeletonIDPattern.MatchString(s.ID) || ids[s.ID] {
			return fmt.Errorf("%s: invalid or duplicate skeleton id %q", path, s.ID)
		}
		ids[s.ID] = true
		if err := c.validateSkeleton(s); err != nil {
			return fmt.Errorf("%s: %s: %w", path, s.ID, err)
		}
		if s.ID == GenericSkeletonID {
			generic++
			continue
		}
		for _, other := range file.Skeletons[:i] {
			if other.ID != GenericSkeletonID && SamePredicate(s.AppliesWhen, other.AppliesWhen) {
				return fmt.Errorf("%s: %s and %s have identical predicates", path, other.ID, s.ID)
			}
		}
	}
	if generic != 1 {
		return fmt.Errorf("%s: needs exactly one %s", path, GenericSkeletonID)
	}
	c.program.skeletons = file.Skeletons
	return nil
}

func (c *Catalog) validateSkeleton(s Skeleton) error {
	if strings.TrimSpace(s.Label) == "" || !programStatus(s.Status) || len(s.Stages) == 0 {
		return fmt.Errorf("needs label, status and stages")
	}
	if err := checkProgramSources(s.Sources); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, id := range s.Stages {
		if seen[id] {
			return fmt.Errorf("stage %s listed twice", id)
		}
		seen[id] = true
		if err := c.checkLibraryStage(id); err != nil {
			return err
		}
	}
	if err := c.checkLinks(s.Stages, s.Links); err != nil {
		return err
	}
	if s.ID == GenericSkeletonID {
		if s.AppliesWhen != nil {
			return fmt.Errorf("the fallback has no applies_when")
		}
		return c.checkGeneric(s)
	}
	if s.AppliesWhen == nil {
		return fmt.Errorf("needs applies_when (only %s has none)", GenericSkeletonID)
	}
	return c.validateProgramPredicate(s.AppliesWhen, 0)
}

// GenericStages derives sk.generic's stages from the catalogue: each
// top-level stage, replaced by its sub-stages where it has more than one,
// without `other`. A single sub-stage (Construction's inspections fee line)
// is a fee item inside its stage, not a replacement for it: replacing it
// would leave a program with no Construction stage (plan 13.3, P1 note).
// The loader holds the file to this so the two cannot drift.
func (c *Catalog) GenericStages() []string {
	var out []string
	for _, top := range c.stages {
		if top.ParentID != "" || top.ID == OtherStageID {
			continue
		}
		var subs []string
		for _, s := range c.stages {
			if s.ParentID == top.ID {
				subs = append(subs, s.ID)
			}
		}
		if len(subs) < 2 {
			subs = []string{top.ID}
		}
		out = append(out, subs...)
	}
	return out
}

func (c *Catalog) checkGeneric(s Skeleton) error {
	want := c.GenericStages()
	if strings.Join(s.Stages, ",") != strings.Join(want, ",") {
		return fmt.Errorf("stages must be the catalogue order %v", want)
	}
	if len(s.Links) != len(want)-1 {
		return fmt.Errorf("links must chain the stages finish-to-start")
	}
	for i, l := range s.Links {
		if l.From != want[i] || l.To != want[i+1] || l.Type != "FS" || l.Lag != nil {
			return fmt.Errorf("links must chain the stages finish-to-start with no lag")
		}
	}
	return nil
}

func (c *Catalog) loadCalendars(path string) error {
	var file struct {
		programFile `yaml:",inline"`
		Calendars   []Calendar `yaml:"calendars"`
	}
	ok, err := c.decodeProgram(path, &file, "date", "from", "to", "kind")
	if !ok || err != nil {
		return err
	}
	if err := checkProgramFile(path, file.programFile); err != nil {
		return err
	}
	ids := map[string]bool{}
	cover := map[string]bool{}
	for _, cal := range file.Calendars {
		if !calendarIDPattern.MatchString(cal.ID) || ids[cal.ID] {
			return fmt.Errorf("%s: invalid or duplicate calendar id %q", path, cal.ID)
		}
		ids[cal.ID] = true
		if err := validateCalendar(cal); err != nil {
			return fmt.Errorf("%s: %s: %w", path, cal.ID, err)
		}
		key := fmt.Sprintf("%s/%d", cal.State, cal.Year)
		if cover[key] {
			return fmt.Errorf("%s: two calendars for %s", path, key)
		}
		cover[key] = true
	}
	c.program.calendars = file.Calendars
	return nil
}

func validateCalendar(cal Calendar) error {
	if !inList(CalendarStates, cal.State) || cal.Year < 2000 || cal.Year > 2100 || cal.Verified == nil || !programStatus(cal.Status) {
		return fmt.Errorf("needs a state, a year, verified and status")
	}
	if err := checkProgramSources(cal.Sources); err != nil {
		return err
	}
	if len(cal.WorkingWeek) == 0 {
		return fmt.Errorf("needs a working_week")
	}
	seen := map[string]bool{}
	for _, d := range cal.WorkingWeek {
		if !inList(weekdays, d) || seen[d] {
			return fmt.Errorf("working_week day %q", d)
		}
		seen[d] = true
	}
	inYear := func(s string) (time.Time, error) {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return t, fmt.Errorf("date %q must be YYYY-MM-DD", s)
		}
		if t.Year() != cal.Year {
			return t, fmt.Errorf("date %s is outside %d", s, cal.Year)
		}
		return t, nil
	}
	for _, day := range cal.Days {
		if !inList(CalendarDayKinds, day.Kind) || strings.TrimSpace(day.Label) == "" {
			return fmt.Errorf("day needs a kind in %v and a label", CalendarDayKinds)
		}
		single := day.Date != ""
		span := day.From != "" || day.To != ""
		if single == span {
			return fmt.Errorf("day needs either date or from and to")
		}
		if single {
			if _, err := inYear(day.Date); err != nil {
				return err
			}
			continue
		}
		from, err := inYear(day.From)
		if err != nil {
			return err
		}
		to, err := inYear(day.To)
		if err != nil {
			return err
		}
		if from.After(to) {
			return fmt.Errorf("from %s is after to %s", day.From, day.To)
		}
	}
	return nil
}

// validateProgramPredicate accepts the documented operators plus the program
// library's two: `cond` (a Profile condition) and `work_action`.
func (c *Catalog) validateProgramPredicate(p any, depth int) error {
	if depth > 50 {
		return fmt.Errorf("predicate too deep")
	}
	m, ok := asMap(p)
	if !ok || len(m) == 0 {
		return fmt.Errorf("predicate must be a non-empty mapping")
	}
	if det, ok := m["det"]; ok {
		id := fmt.Sprint(det)
		if _, known := c.Determinant(id); !known {
			return fmt.Errorf("unknown determinant %s", id)
		}
		op, arg, err := oneOperator(m, "det", "any_of", "eq", "is", "gt", "gte", "lt", "lte")
		if err != nil {
			return err
		}
		if op == "is" {
			if _, ok := arg.(bool); !ok {
				return fmt.Errorf("is requires a boolean")
			}
		}
		// work_type is fed by code from the Profile's work types, so its
		// literals must be taxonomy work type ids or the term never holds.
		if id == "work_type" {
			var allowed []string
			for _, w := range c.profile.taxonomy.WorkTypes {
				allowed = append(allowed, w.ID)
			}
			return checkLiterals(op, arg, allowed, "work type")
		}
		return nil
	}
	if key, ok := m["cond"]; ok {
		var cond *Condition
		for i := range c.profile.taxonomy.Conditions {
			if c.profile.taxonomy.Conditions[i].Key == fmt.Sprint(key) {
				cond = &c.profile.taxonomy.Conditions[i]
			}
		}
		if cond == nil {
			return fmt.Errorf("unknown condition %v", key)
		}
		op, arg, err := oneOperator(m, "cond", "any_of", "eq")
		if err != nil {
			return err
		}
		var allowed []string
		for _, o := range cond.AllOptions() {
			allowed = append(allowed, o.ID)
		}
		return checkLiterals(op, arg, allowed, "option of "+cond.Key)
	}
	for op, v := range m {
		switch op {
		case "all", "any":
			list, ok := v.([]any)
			if !ok || len(list) == 0 {
				return fmt.Errorf("%s needs a non-empty list", op)
			}
			for _, child := range list {
				if err := c.validateProgramPredicate(child, depth+1); err != nil {
					return err
				}
			}
		case "not":
			if err := c.validateProgramPredicate(v, depth+1); err != nil {
				return err
			}
		case "system_present", "system_existing":
			if _, ok := c.System(fmt.Sprint(v)); !ok {
				return fmt.Errorf("unknown system %v", v)
			}
		case "works":
			if err := c.checkWorksRefs("program", "applies_when", map[string]any{"works": v}, nil); err != nil {
				return err
			}
		case "work_action":
			if err := c.checkWorkAction(v); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown predicate operator %s", op)
		}
	}
	return nil
}

func (c *Catalog) checkWorkAction(v any) error {
	w, ok := asMap(v)
	if !ok || len(w) != 2 {
		return fmt.Errorf("work_action needs exactly system and any_of")
	}
	sys, ok := w["system"].(string)
	if !ok {
		return fmt.Errorf("work_action needs one system id")
	}
	if _, known := c.System(sys); !known {
		return fmt.Errorf("work_action names unknown system %s", sys)
	}
	actions := asList(w["any_of"])
	if len(actions) == 0 {
		return fmt.Errorf("work_action needs a non-empty any_of")
	}
	for _, a := range actions {
		s := fmt.Sprint(a)
		if !inList(FragmentActions, s) || !c.isAction(s) {
			return fmt.Errorf("work_action action %s must be one of %v", s, FragmentActions)
		}
	}
	return nil
}

// oneOperator returns the single comparison beside the operand key.
func oneOperator(m map[string]any, operand string, allowed ...string) (string, any, error) {
	if len(m) != 2 {
		return "", nil, fmt.Errorf("%s needs exactly one operator", operand)
	}
	for op, v := range m {
		if op == operand {
			continue
		}
		if !inList(allowed, op) {
			return "", nil, fmt.Errorf("unknown predicate operator %s", op)
		}
		if v == nil {
			return "", nil, fmt.Errorf("empty comparison")
		}
		return op, v, nil
	}
	return "", nil, fmt.Errorf("%s needs an operator", operand)
}

func checkLiterals(op string, arg any, allowed []string, what string) error {
	values := []any{arg}
	if op == "any_of" {
		values = asList(arg)
		if len(values) == 0 {
			return fmt.Errorf("any_of needs values")
		}
	} else if op != "eq" {
		return fmt.Errorf("%s compares with any_of or eq", what)
	}
	for _, v := range values {
		if !inList(allowed, fmt.Sprint(v)) {
			return fmt.Errorf("%v is not a valid %s", v, what)
		}
	}
	return nil
}

func inList(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
