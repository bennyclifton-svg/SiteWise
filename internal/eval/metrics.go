// Package eval measures intake accuracy against answer keys and fits the
// green/amber cut-offs that internal/intake applies.
//
// Cut-offs are fitted per question and option shape on the calibration split
// only, and stay unknown when the data cannot support them. No probability
// from the TypeSafe documentation is used as a default.
//
//	https://docs.typesafe.ai/confidence
//	https://docs.typesafe.ai/patterns/confidence-routing
package eval

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

const (
	SplitCalibration = "calibration"
	SplitHeldout     = "heldout"
)

// Outcome is one scored field on one case. Value is empty when nothing was
// applied; Correct is only true for an applied value matching Expected.
type Outcome struct {
	Case      string
	Field     string
	Expected  string
	Value     string
	Band      string
	DecidedBy string
	Correct   bool
}

// FieldMetrics summarises outcomes for one field. Accuracy is over applied
// values; Coverage is applied over scored. The interval is Wilson's.
type FieldMetrics struct {
	Scored         int     `json:"scored"`
	Applied        int     `json:"applied"`
	Correct        int     `json:"correct"`
	Amber          int     `json:"amber"`
	Grey           int     `json:"grey"`
	FalseConfident int     `json:"false_confident"`
	Coverage       float64 `json:"coverage"`
	Accuracy       float64 `json:"accuracy"`
	AccuracyLow    float64 `json:"accuracy_low"`
	AccuracyHigh   float64 `json:"accuracy_high"`
	AmberRate      float64 `json:"amber_rate"`
}

// Summarize groups outcomes by field. A green value that is wrong is a false
// confident action whether a rule or Jev applied it.
func Summarize(outcomes []Outcome, z float64) map[string]FieldMetrics {
	out := map[string]FieldMetrics{}
	for _, o := range outcomes {
		m := out[o.Field]
		m.Scored++
		if o.Band == "grey" {
			m.Grey++
		}
		if o.Value != "" {
			m.Applied++
			if o.Band == "amber" {
				m.Amber++
			}
			if o.Correct {
				m.Correct++
			} else if o.Band == "green" {
				m.FalseConfident++
			}
		}
		out[o.Field] = m
	}
	for field, m := range out {
		m.Coverage = ratio(m.Applied, m.Scored)
		m.AmberRate = ratio(m.Amber, m.Scored)
		if m.Applied > 0 {
			m.Accuracy = ratio(m.Correct, m.Applied)
			m.AccuracyLow, m.AccuracyHigh = Wilson(m.Correct, m.Applied, z)
		}
		out[field] = m
	}
	return out
}

func ratio(k, n int) float64 {
	if n == 0 {
		return 0
	}
	return float64(k) / float64(n)
}

// Wilson is the score interval for k successes in n trials. An empty sample
// is [0, 1]: it supports no claim.
func Wilson(k, n int, z float64) (lo, hi float64) {
	if n <= 0 {
		return 0, 1
	}
	p := float64(k) / float64(n)
	nn := float64(n)
	z2 := z * z
	denom := 1 + z2/nn
	center := p + z2/(2*nn)
	margin := z * math.Sqrt(p*(1-p)/nn+z2/(4*nn*nn))
	lo = math.Max(0, (center-margin)/denom)
	hi = math.Min(1, (center+margin)/denom)
	return lo, hi
}

// Shape is an inclusive option-count range. Choice confidence is only
// compared within one shape.
type Shape struct {
	Min int `json:"min_options"`
	Max int `json:"max_options"`
}

// ShapeOf buckets an option count by powers of two. Identity questions vary
// in option count per document; exact counts would leave every bucket too
// small to fit. Fewer than two options is not a question.
func ShapeOf(options int) Shape {
	if options < 2 || options > 255 {
		return Shape{}
	}
	if options == 2 {
		return Shape{2, 2}
	}
	hi := 4
	for hi < options {
		hi *= 2
	}
	if hi > 255 {
		return Shape{129, 255}
	}
	return Shape{hi/2 + 1, hi}
}

// Answer is one raw Jev choice on a scored case, before any threshold.
type Answer struct {
	Case       string
	Field      string
	Options    int
	Confidence float64
	Correct    bool
}

// FitPolicy is the owner's accuracy requirement for one question. The lower
// bounds are on Wilson intervals at Z. MaxErrors < 0 means no hard cap.
type FitPolicy struct {
	Z              float64 `json:"z"`
	MinSamples     int     `json:"min_samples"`
	MinBandSamples int     `json:"min_band_samples"`
	GreenLower     float64 `json:"green_lower"`
	AmberLower     float64 `json:"amber_lower"`
	MaxErrors      int     `json:"max_errors"`
}

// FitResult is one fitted cut-off. Nil cut-offs are unknown and Reason says why.
type FitResult struct {
	Field  string   `json:"field"`
	Shape  Shape    `json:"shape"`
	N      int      `json:"n"`
	Green  *float64 `json:"green"`
	Amber  *float64 `json:"amber"`
	Reason string   `json:"reason,omitempty"`
}

// Fit returns one result per field and shape seen in answers. Green is the
// lowest observed confidence whose at-or-above set meets the green bound;
// amber is the lowest whose band below green meets the amber bound, or green
// itself when no band qualifies. Without green, amber is independently fitted
// over the at-or-above set. A field with no policy stays unknown.
func Fit(answers []Answer, policies map[string]FitPolicy) []FitResult {
	type key struct {
		field string
		shape Shape
	}
	groups := map[key][]Answer{}
	for _, a := range answers {
		s := ShapeOf(a.Options)
		if s == (Shape{}) || math.IsNaN(a.Confidence) {
			continue
		}
		k := key{a.Field, s}
		groups[k] = append(groups[k], a)
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].field != keys[j].field {
			return keys[i].field < keys[j].field
		}
		return keys[i].shape.Min < keys[j].shape.Min
	})
	out := make([]FitResult, 0, len(keys))
	for _, k := range keys {
		group := groups[k]
		res := FitResult{Field: k.field, Shape: k.shape, N: len(group)}
		policy, ok := policies[k.field]
		switch {
		case !ok:
			res.Reason = "no fit policy for this question"
		case len(group) < policy.MinSamples:
			res.Reason = fmt.Sprintf("%d calibration answers, need %d", len(group), policy.MinSamples)
		default:
			res.Green, res.Amber, res.Reason = fitGroup(group, policy)
		}
		out = append(out, res)
	}
	return out
}

func fitGroup(group []Answer, p FitPolicy) (green, amber *float64, reason string) {
	cuts := distinctDescending(group)
	var g float64
	found := false
	for _, c := range cuts {
		k, n := tally(group, c, math.Inf(1))
		if n < p.MinBandSamples {
			continue
		}
		if p.MaxErrors >= 0 && n-k > p.MaxErrors {
			continue
		}
		if lo, _ := Wilson(k, n, p.Z); lo >= p.GreenLower {
			g = c
			found = true
		}
	}
	upper := math.Inf(1)
	if found {
		green = &g
		upper = g
		amber = &g
	}
	for _, c := range cuts {
		if c >= upper {
			continue
		}
		k, n := tally(group, c, upper)
		if n < p.MinBandSamples {
			continue
		}
		if p.MaxErrors >= 0 && n-k > p.MaxErrors {
			continue
		}
		if lo, _ := Wilson(k, n, p.Z); lo >= p.AmberLower {
			cut := c
			amber = &cut
		}
	}
	if green == nil && amber == nil {
		return nil, nil, "no cut-off meets either accuracy bound"
	}
	if green == nil {
		return nil, amber, "review only; no cut-off meets the green bound"
	}
	return green, amber, ""
}

// tally counts correct and total answers with from <= confidence < below.
func tally(group []Answer, from, below float64) (correct, n int) {
	for _, a := range group {
		if a.Confidence >= from && a.Confidence < below {
			n++
			if a.Correct {
				correct++
			}
		}
	}
	return correct, n
}

func distinctDescending(group []Answer) []float64 {
	seen := map[float64]struct{}{}
	var out []float64
	for _, a := range group {
		if _, ok := seen[a.Confidence]; ok {
			continue
		}
		seen[a.Confidence] = struct{}{}
		out = append(out, a.Confidence)
	}
	sort.Sort(sort.Reverse(sort.Float64Slice(out)))
	return out
}

// Member is one case with the keys that tie it to near-duplicates: its
// directory and its document number series.
type Member struct {
	ID   string
	Keys []string
}

// Families joins members sharing any key. The family id is the smallest key
// in the joined set, so it does not depend on input order.
func Families(members []Member) map[string]string {
	parent := map[string]string{}
	var find func(string) string
	find = func(x string) string {
		if parent[x] == x {
			return x
		}
		parent[x] = find(parent[x])
		return parent[x]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra == rb {
			return
		}
		if rb < ra {
			ra, rb = rb, ra
		}
		parent[rb] = ra
	}
	for _, m := range members {
		for _, k := range m.Keys {
			if _, ok := parent[k]; !ok {
				parent[k] = k
			}
		}
		for _, k := range m.Keys[1:] {
			union(m.Keys[0], k)
		}
	}
	out := make(map[string]string, len(members))
	for _, m := range members {
		if len(m.Keys) == 0 {
			out[m.ID] = "case:" + m.ID
			continue
		}
		out[m.ID] = find(m.Keys[0])
	}
	return out
}

// SplitOf assigns a family by hash. It is stable when cases are added, and a
// family never straddles the two splits.
func SplitOf(family, salt string, calibrationPercent int) string {
	sum := sha256.Sum256([]byte(salt + "\x00" + family))
	if int(binary.BigEndian.Uint64(sum[:8])%100) < calibrationPercent {
		return SplitCalibration
	}
	return SplitHeldout
}

// Regressions lists fields that lost correct values or gained false
// confident actions against a baseline, and baseline fields now unscored.
func Regressions(baseline, current map[string]FieldMetrics) []string {
	fields := make([]string, 0, len(baseline))
	for f := range baseline {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	var out []string
	for _, f := range fields {
		b := baseline[f]
		c, ok := current[f]
		if !ok {
			out = append(out, f+": no longer scored")
			continue
		}
		if c.Correct < b.Correct {
			out = append(out, fmt.Sprintf("%s: correct %d below baseline %d", f, c.Correct, b.Correct))
		}
		if c.FalseConfident > b.FalseConfident {
			out = append(out, fmt.Sprintf("%s: false confident %d above baseline %d", f, c.FalseConfident, b.FalseConfident))
		}
	}
	return out
}
