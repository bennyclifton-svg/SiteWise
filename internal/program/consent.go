package program

import (
	"fmt"
	"slices"
	"strings"
)

// Consent gates (docs/plans/2026-11-17-consent-conditions.md, Task 7). One
// hold point per gate that has a live, applicable consent condition, linked
// finish-to-start before the activity the gate stands for. The label carries
// no count, so a condition added or satisfied never churns the review; the
// register says how many are open. During works, ongoing and throughout are
// obligations, not gates (owner decision 4): no hold point.

// SourceConsent is the review source of a gate's rows and links.
const SourceConsent = "consent"

// CheckConsentGateOpen fires when a gate is released with conditions open.
const CheckConsentGateOpen = "consent_gate_open"

// ConsentGate is one gate a hold point stands for. The ids are the
// register's (delivery.Gate*).
type ConsentGate struct {
	ID, Label string
	// target finds the activity the gate holds, by source key.
	target func(key string) bool
	// fallback names what is missing when no activity is found.
	missing string
}

// ConsentGates are the gates in project order.
var ConsentGates = []ConsentGate{
	{ID: "before_cc", Label: "Consent conditions before the construction certificate",
		target: prefix("fragment:fr.construction-certificate:"), missing: "construction certificate"},
	{ID: "before_work", Label: "Consent conditions before work starts", missing: "construction"},
	{ID: "before_occupation", Label: "Consent conditions before the occupation certificate",
		target: prefix("fragment:fr.occupation-certificate:"), missing: "occupation certificate"},
	{ID: "before_subdivision", Label: "Consent conditions before the subdivision certificate",
		target: func(k string) bool {
			return strings.HasPrefix(k, "fragment:") && strings.Contains(k, "subdivision-certificate")
		},
		missing: "subdivision certificate"},
}

func prefix(p string) func(string) bool {
	return func(k string) bool { return strings.HasPrefix(k, p) }
}

// ConsentGateKey is a gate hold point's source key.
func ConsentGateKey(gate string) string { return "consent:" + gate }

// ConsentGateOf is the gate of a source key, or "".
func ConsentGateOf(key string) string {
	g, ok := strings.CutPrefix(key, "consent:")
	if !ok {
		return ""
	}
	return g
}

// consentGateRows adds a hold point for each gate with conditions, in the
// outline and the detailed program alike.
func (g *gen) consentGateRows() {
	for _, gate := range ConsentGates {
		if g.in.ConsentGates[gate.ID] == 0 {
			continue
		}
		target, after := g.gateTarget(gate), ""
		if target == "" {
			// No certificate activity (the generic outline has none): a gate
			// before the certificate holds construction from starting; one
			// after construction follows the last construction activity.
			first, last := g.constructionEnds()
			if gate.ID == "before_cc" || gate.ID == "before_work" {
				target = first
			} else {
				after = last
			}
			g.note(CheckHoldPointUnlinked, ConsentGateKey(gate.ID), fmt.Sprintf("No %s activity in the program; the gate is placed by the construction activities.", gate.missing))
		}
		anchor := target
		if anchor == "" {
			anchor = after
		}
		if anchor == "" {
			continue
		}
		key := ConsentGateKey(gate.ID)
		zero := 0
		row := Row{SourceKey: key, ParentKey: g.plan.Rows[g.byKey[anchor]].ParentKey, Kind: KindHoldPoint, Label: gate.Label, Systems: []string{}, Duration: &zero}
		switch prev := g.stageBefore(row.ParentKey); {
		case target != "" && g.firstChild(target) && prev != "":
			// The activity opens its stage: the gate closes the stage
			// before, so Adopt chains it there and it holds the activity
			// across the stage boundary.
			row.ParentKey = prev
			g.addRow(row)
		case target != "":
			// The gate sits just before the activity it holds, so Adopt
			// places it there and links it into it.
			g.insertBefore(row, target)
		default:
			g.addRow(row)
		}
		g.gates = append(g.gates, gateRow{key: key, target: target, after: after})
	}
}

// firstChild reports the first activity of its stage.
func (g *gen) firstChild(key string) bool {
	r := g.plan.Rows[g.byKey[key]]
	for _, o := range g.plan.Rows {
		if o.ParentKey == r.ParentKey && o.Ordinal < r.Ordinal {
			return false
		}
	}
	return true
}

// stageBefore is the stage before a stage, or "".
func (g *gen) stageBefore(stage string) string {
	for i, k := range g.stageOrder {
		if k == stage && i > 0 {
			return g.stageOrder[i-1]
		}
	}
	return ""
}

// insertBefore adds a row in its parent just before another, in plan order
// and ordinal.
func (g *gen) insertBefore(r Row, before string) {
	at := g.byKey[before]
	ord := g.plan.Rows[at].Ordinal
	g.addRow(r)
	added := g.plan.Rows[len(g.plan.Rows)-1]
	for i := range g.plan.Rows[:len(g.plan.Rows)-1] {
		if g.plan.Rows[i].ParentKey == added.ParentKey && g.plan.Rows[i].Ordinal >= ord {
			g.plan.Rows[i].Ordinal++
		}
	}
	added.Ordinal = ord
	rows := append([]Row{}, g.plan.Rows[:at]...)
	rows = append(rows, added)
	rows = append(rows, g.plan.Rows[at:len(g.plan.Rows)-1]...)
	g.plan.Rows = rows
	for i, row := range g.plan.Rows {
		g.byKey[row.SourceKey] = i
	}
}

// constructionEnds are the first and last activities of the construction
// stage.
func (g *gen) constructionEnds() (first, last string) {
	stage, _ := g.resolveStage(stageConstruction)
	for _, r := range g.plan.Rows {
		if r.ParentKey == stage && r.Kind == KindActivity {
			if first == "" {
				first = r.SourceKey
			}
			last = r.SourceKey
		}
	}
	return first, last
}

// gateTarget is the activity a gate holds: the certificate's activity, or
// for work starting the first activity of the construction stage.
func (g *gen) gateTarget(gate ConsentGate) string {
	if gate.target == nil {
		first, _ := g.constructionEnds()
		return first
	}
	for _, stage := range g.stageOrder {
		for _, r := range g.plan.Rows {
			if r.ParentKey == stage && r.Kind == KindActivity && gate.target(r.SourceKey) {
				return r.SourceKey
			}
		}
	}
	return ""
}

type gateRow struct{ key, target, after string }

// consentGateLinks: the gate waits on what its activity waits on, and its
// activity waits on the gate, finish-to-start with no lag.
func (g *gen) consentGateLinks() {
	for _, h := range g.gates {
		if h.target == "" {
			g.addGuarded(PlanLink{PredKey: h.after, SuccKey: h.key, Type: LinkFS, Origin: LinkOriginInterface})
			continue
		}
		for _, l := range slices.Clone(g.plan.Links) {
			if l.SuccKey == h.target && l.PredKey != h.key && g.plan.Rows[g.byKey[l.PredKey]].Kind != KindStage {
				g.addGuarded(PlanLink{PredKey: l.PredKey, SuccKey: h.key, Type: LinkFS, Origin: LinkOriginInterface})
			}
		}
		g.addGuarded(PlanLink{PredKey: h.key, SuccKey: h.target, Type: LinkFS, Origin: LinkOriginInterface})
	}
}

// ConsentGateOpenCheck is the check for a released gate with conditions
// still open.
func ConsentGateOpenCheck(activityID, label string, open int) Check {
	words := "1 consent condition is"
	if open != 1 {
		words = fmt.Sprintf("%d consent conditions are", open)
	}
	return Check{Code: CheckConsentGateOpen, ActivityID: activityID, Message: fmt.Sprintf("%s released, but %s still open.", label, words)}
}
