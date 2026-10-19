// Package program schedules a construction program: a pure engine that turns
// stored inputs (activities, links, calendar) into dates, float and the
// critical path on every read. Nothing here touches the database or Jev;
// derived values are computed, never stored (plan section 2, product rule 10).
//
// Input JSON uses the shared fixture names (internal/program/testdata/engine),
// so the Go tests, the TypeScript mirror and the store all build the same
// shape. Output JSON is the `computed` object of the API response (plan 8.2).
package program

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// Activity kinds. A stage has no parent; everything else sits under a stage.
const (
	KindStage     = "stage"
	KindActivity  = "activity"
	KindMilestone = "milestone"
	KindHoldPoint = "hold_point"
)

// Constraint kinds. asap is the default; snet holds a start no earlier than a
// date (what a drag later sets); mso fixes the start and reports a conflict
// when links would hold it later.
const (
	ConstraintASAP = "asap"
	ConstraintSNET = "snet"
	ConstraintMSO  = "mso"
)

// Link types and lag units (plan 6.2).
const (
	LinkFS      = "FS"
	LinkSS      = "SS"
	LinkFF      = "FF"
	LinkSF      = "SF"
	LagWorking  = "working"
	LagCalendar = "calendar"
)

// Calendar day kinds. Every kind except working_override takes a day out of
// the working week; working_override puts a normally idle day back in.
const (
	DayPublicHoliday   = "public_holiday"
	DayRDO             = "rdo"
	DayShutdown        = "shutdown"
	DayWorkingOverride = "working_override"
)

// DefaultWorkingWeek is Monday to Friday in the Mon=1 … Sun=64 bitmask.
const DefaultWorkingWeek = 31

type Input struct {
	Calendar   Calendar   `json:"calendar"`
	Activities []Activity `json:"activities"`
	Links      []Link     `json:"links"`
}

type Calendar struct {
	// Start is YYYY-MM-DD, or "" for a relative program ("Week 1, day 1").
	Start string `json:"start"`
	// WorkingWeek is the bitmask Mon=1, Tue=2 … Sun=64; 0 means the default.
	WorkingWeek int           `json:"working_week"`
	Days        []CalendarDay `json:"days"`
}

type CalendarDay struct {
	Day   string `json:"day"`
	Kind  string `json:"kind"`
	Label string `json:"label,omitempty"`
}

type Activity struct {
	ID       string `json:"id"`
	ParentID string `json:"parent,omitempty"`
	Ordinal  int    `json:"ordinal"`
	Kind     string `json:"kind,omitempty"` // "" is read as activity
	Label    string `json:"label,omitempty"`
	// Duration is in working days; nil is unknown and stays unknown. A stage
	// with children ignores it (its span comes from the children); a
	// milestone is always 0.
	Duration *int `json:"duration"`
	// Low and High feed the optimistic and pessimistic runs. When absent the
	// typical duration is used in all three runs, which is how user, document
	// and profile durations keep one value (plan 6.5).
	Low            *int   `json:"low,omitempty"`
	High           *int   `json:"high,omitempty"`
	Constraint     string `json:"constraint,omitempty"` // "" is asap
	ConstraintDate string `json:"constraint_date,omitempty"`
}

type Link struct {
	ID      string `json:"id"`
	Pred    string `json:"pred"`
	Succ    string `json:"succ"`
	Type    string `json:"type"` // "" is read as FS
	Lag     int    `json:"lag"`
	LagUnit string `json:"lag_unit,omitempty"` // "" is working
}

// Output is the `computed` object of GET /program (plan 8.2).
type Output struct {
	Status Status `json:"status"`
	Finish Finish `json:"finish"`
	// Activities has one entry per input activity, in input order, and
	// marshals as an object keyed by id. Hidden stage start and finish nodes
	// are never returned.
	Activities   RowTimes `json:"activities"`
	DrivingLinks []string `json:"driving_links"`
	// Cycle is one loop of activity ids, closed (first id repeated last), or
	// nil. When set nothing else is computed.
	Cycle     []string   `json:"cycle"`
	Conflicts []Conflict `json:"conflicts"`
}

type Status struct {
	Known int    `json:"known"`
	Total int    `json:"total"`
	Words string `json:"words"`
}

// Finish is the project finish from the typical, low and high runs. Indices
// are the exclusive finish (PF); dates are the last working day, nil for a
// relative program or when nothing is known. Complete is false when some row
// is unknown: the known finish is then a lower bound, never the answer.
type Finish struct {
	Typical      *string `json:"typical"`
	Low          *string `json:"low"`
	High         *string `json:"high"`
	TypicalIndex *int    `json:"typical_index"`
	LowIndex     *int    `json:"low_index"`
	HighIndex    *int    `json:"high_index"`
	Complete     bool    `json:"complete"`
}

// Times are working-day indices on the half-open interval [es, ef). When
// Unknown is true the numbers are zero and meaningless, and the JSON carries
// null for every number and date: unknown stays unknown. Start and Finish
// are YYYY-MM-DD, "" (null in JSON) for a relative program; Finish is also
// "" for a zero-length row, which shows its start only.
type Times struct {
	ID       string
	ES       int
	EF       int
	LS       int
	LF       int
	TF       int
	FF       int
	Critical bool
	Start    string
	Finish   string
	Unknown  bool
}

// MarshalJSON writes {"es":…,"ef":…,"ls":…,"lf":…,"tf":…,"ff":…,
// "critical":…,"start":…,"finish":…,"unknown":…}. It is written by hand
// because a read of 2,000 rows serialises it 2,000 times.
func (t Times) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, 128)
	num := func(key string, v int) {
		b = append(b, '"')
		b = append(b, key...)
		b = append(b, '"', ':')
		if t.Unknown {
			b = append(b, "null"...)
		} else {
			b = strconv.AppendInt(b, int64(v), 10)
		}
		b = append(b, ',')
	}
	str := func(key, v string) {
		b = append(b, '"')
		b = append(b, key...)
		b = append(b, '"', ':')
		if t.Unknown || v == "" {
			b = append(b, "null"...)
		} else {
			b = strconv.AppendQuote(b, v)
		}
		b = append(b, ',')
	}
	b = append(b, '{')
	num("es", t.ES)
	num("ef", t.EF)
	num("ls", t.LS)
	num("lf", t.LF)
	num("tf", t.TF)
	num("ff", t.FF)
	b = append(b, `"critical":`...)
	b = strconv.AppendBool(b, t.Critical)
	b = append(b, ',')
	str("start", t.Start)
	str("finish", t.Finish)
	b = append(b, `"unknown":`...)
	b = strconv.AppendBool(b, t.Unknown)
	return append(b, '}'), nil
}

// Conflict is a mandatory start that links would hold later. HeldBy is the
// predecessor row (a stage when the hold comes through the stage).
type Conflict struct {
	ActivityID string `json:"activity_id"`
	Index      int    `json:"index"`
	HeldIndex  int    `json:"held_index"`
	HeldBy     string `json:"held_by"`
	LinkID     string `json:"link_id,omitempty"`
}

// RowTimes is aligned with Input.Activities: entry i is activity i. A slice
// instead of a map keeps Compute inside its 2 ms budget (no hashing of 2,000
// keys on every read); the JSON is still the object keyed by id of plan 8.2.
type RowTimes []Times

func (r RowTimes) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for k, t := range r {
		if k > 0 {
			b.WriteByte(',')
		}
		id, err := json.Marshal(t.ID)
		if err != nil {
			return nil, err
		}
		v, _ := t.MarshalJSON()
		b.Write(id)
		b.WriteByte(':')
		b.Write(v)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// Get finds one activity's times. It scans; for many lookups index by input
// position or build Map once.
func (r RowTimes) Get(id string) (Times, bool) {
	for _, t := range r {
		if t.ID == id {
			return t, true
		}
	}
	return Times{}, false
}

// Map keys the times by activity id.
func (r RowTimes) Map() map[string]Times {
	m := make(map[string]Times, len(r))
	for _, t := range r {
		m[t.ID] = t
	}
	return m
}
