package delivery

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Consent conditions (docs/plans/2026-11-17-consent-conditions.md). A
// condition is an obligation the development consent places on the project.
// Its number, heading and words come from the notice itself and are never
// renumbered or reworded; the PM edits who acts, the gate, the system and
// whether evidence is needed.

// Gates a condition is met before, in the order a project reaches them. ""
// is unknown: no heading said and nothing was picked.
const (
	GateConstructionCertificate = "before_cc"
	GateWorkStarts              = "before_work"
	GateDuringWorks             = "during_works"
	GateOccupation              = "before_occupation"
	GateSubdivision             = "before_subdivision"
	GateOngoing                 = "ongoing"
	GateThroughout              = "throughout"
)

// Gates lists every gate in project order.
var Gates = []string{GateConstructionCertificate, GateWorkStarts, GateDuringWorks, GateOccupation, GateSubdivision, GateOngoing, GateThroughout}

// GateWords are the register's Due words and chat's.
var GateWords = map[string]string{
	GateConstructionCertificate: "Before CC",
	GateWorkStarts:              "Before work starts",
	GateDuringWorks:             "During works",
	GateOccupation:              "Before OC",
	GateSubdivision:             "Before SC",
	GateOngoing:                 "Ongoing after occupation",
	GateThroughout:              "Throughout",
}

// GateLong names a gate in full, for a hold point label and a card.
var GateLong = map[string]string{
	GateConstructionCertificate: "before the construction certificate",
	GateWorkStarts:              "before work starts",
	GateDuringWorks:             "during works",
	GateOccupation:              "before the occupation certificate",
	GateSubdivision:             "before the subdivision certificate",
	GateOngoing:                 "ongoing after occupation",
	GateThroughout:              "throughout (no single gate)",
}

// HeldAt reports a gate a Program hold point stands for. During works,
// ongoing and throughout are obligations, not gates (owner decision 4).
func HeldAt(gate string) bool {
	return gate == GateConstructionCertificate || gate == GateWorkStarts || gate == GateOccupation || gate == GateSubdivision
}

// MaxConditionText is how much of a condition's printed words a row keeps.
const MaxConditionText = 12000

// ConditionDetails is what is particular to a consent condition.
type ConditionDetails struct {
	ConsentRef     string              `json:"consent_ref"`
	Number         string              `json:"number"` // as printed: "46", "A1"
	Heading        string              `json:"heading"`
	Text           string              `json:"text"` // the council's words, verbatim
	Stage          string              `json:"stage"`
	StageHeading   string              `json:"stage_heading"`
	Discipline     string              `json:"discipline"`
	EvidenceNeeded *bool               `json:"evidence_needed"` // nil is unknown
	Evidence       []ConditionEvidence `json:"evidence"`
	Source         *ConditionSource    `json:"source"`
	RFP            *ConditionRFP       `json:"rfp"`
}

// ConditionEvidence is a filed document the PM linked as evidence.
type ConditionEvidence struct {
	DocumentID string `json:"document_id"`
	LinkedBy   string `json:"linked_by"`
	LinkedAt   string `json:"linked_at"`
}

// ConditionSource is the notice and page the condition was read from.
type ConditionSource struct {
	DocumentID string `json:"document_id"`
	Page       int    `json:"page"`
}

// ConditionRFP is the scope line Add to RFP wrote.
type ConditionRFP struct {
	ReportID string `json:"report_id"`
	Stage    string `json:"stage"`
	LineID   string `json:"line_id"`
}

func (d ConditionDetails) check() error {
	if !bounded(d.ConsentRef, 100) || !bounded(d.Number, 20) || !bounded(d.Heading, 300) || !bounded(d.Text, MaxConditionText) ||
		!bounded(d.StageHeading, 300) || !bounded(d.Discipline, 100) || len(d.Evidence) > 50 {
		return fmt.Errorf("invalid condition details")
	}
	if d.Stage != "" && !slices.Contains(Gates, d.Stage) {
		return fmt.Errorf("invalid condition gate")
	}
	seen := map[string]bool{}
	for _, e := range d.Evidence {
		if !uuidLike(e.DocumentID) || seen[e.DocumentID] || !bounded(e.LinkedBy, 100) || !bounded(e.LinkedAt, 40) {
			return fmt.Errorf("invalid or repeated evidence document")
		}
		seen[e.DocumentID] = true
	}
	if d.Source != nil && (!uuidLike(d.Source.DocumentID) || d.Source.Page < 0) {
		return fmt.Errorf("invalid condition source")
	}
	if d.RFP != nil && (!uuidLike(d.RFP.ReportID) || !bounded(d.RFP.Stage, 100) || !bounded(d.RFP.LineID, 100)) {
		return fmt.Errorf("invalid condition RFP line")
	}
	return nil
}

// Condition reads a condition's details; anything else reads as none.
func Condition(c Content) ConditionDetails {
	var d ConditionDetails
	if c.Kind == "condition" {
		_ = json.Unmarshal(c.Details, &d)
	}
	return d
}

// ConditionTitle is the row's title: "Condition 46 · Landscape plan". With
// no printed heading Go takes the opening words, cut to 120 characters.
func ConditionTitle(number, heading, text string) string {
	words := strings.TrimSpace(heading)
	if words == "" {
		words = strings.Join(strings.Fields(text), " ")
		if r := []rune(words); len(r) > 120 {
			words = strings.TrimSpace(string(r[:119])) + "…"
		}
	}
	title := "Condition " + number
	if words != "" {
		title += " · " + words
	}
	if r := []rune(title); len(r) > 200 {
		title = string(r[:199]) + "…"
	}
	return title
}

// ToCheck reports a row the PM should look at: a possible answer or
// possible evidence to check, or a condition SiteWise read that the PM has
// not yet edited (amber "to check", owner decision 9).
func ToCheck(it Item) bool {
	return len(it.Readings) > 0 || it.Kind == "condition" && it.ReviewStatus == "proposed"
}

// MergeConditionEdit keeps what the instrument printed and what Go
// recorded: an edit may change who acts, the gate and whether evidence is
// needed, never the number, heading, words, source, evidence list or RFP
// line. The PM adds evidence through LinkEvidence, not by rewriting the list.
func MergeConditionEdit(before, after ConditionDetails) ConditionDetails {
	out := before
	out.Stage, out.Discipline, out.EvidenceNeeded = after.Stage, after.Discipline, after.EvidenceNeeded
	return out
}
