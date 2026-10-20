package programread

import (
	"fmt"
	"strings"

	"sitewise/internal/jev"
	"sitewise/internal/knowledge"
)

// Reading C: hold-point release evidence (plan
// docs/plans/2026-10-18-program.md 10.4). The release noul is authored per
// interface in knowledge (hold_point.release) and joins a passage's
// existing evidence fan-out in internal/jobs; this file holds what that
// reading needs from here: its version, its family and its banding.

// HoldPointReleaseVersion is the release question map's own version. The
// passage evidence call keeps its version; when release questions join it,
// this is appended so the passage_calls cache and the recorded version
// both name it. Any change to the release wording or state is a new
// version and needs a new calibration (plan section 2, Jev rule 11).
const HoldPointReleaseVersion = "hold-point-release-1"

// HoldPointFamily is the threshold family of every release noul.
const HoldPointFamily = "hold_point_release"

// FieldReleased is the program_readings field a release proposal writes.
const FieldReleased = "released"

// HoldPointStateKey is the state field the release question reads: the
// hold point's evidence text from the interface.
const HoldPointStateKey = "hold_point"

// HoldPointLimit is the known limit shown in the evidence tooltip (plan
// 10.4 "Known limit (v1)").
const HoldPointLimit = "One hold point per interface per program, so evidence for one pour cannot be told apart from another."

// ReleaseQuestion is the Jev noul for a knowledge release question, with
// string criteria keys: YAML reads `true:` as a boolean key, which cannot
// be encoded as a JSON object (https://docs.typesafe.ai/api).
func ReleaseQuestion(q knowledge.Question) jev.Question {
	criteria := map[string]string{}
	switch m := q.Criteria.(type) {
	case map[string]any:
		for k, v := range m {
			criteria[strings.ToLower(k)] = strings.TrimSpace(fmt.Sprint(v))
		}
	case map[any]any:
		for k, v := range m {
			criteria[strings.ToLower(fmt.Sprint(k))] = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	return jev.Question{Type: jev.TypeNoul, Instructions: strings.TrimSpace(q.Instructions), Criteria: criteria}
}

// BandRelease bands one release answer. A noul carries a probability only,
// never a confidence (https://docs.typesafe.ai/confidence), so the cut-off
// is on the probability of true, from entries with no option shape. No
// cut-off for this version, or an invalid one, authorises nothing: the
// answer stays blank. Only green or amber writes a proposal; a proposal
// never marks the hold point released (plan 10.4 "Never automatic").
func (t Thresholds) BandRelease(probability float64) (band string, apply bool) {
	if !t.versionOK(HoldPointFamily) {
		return BandBlank, false
	}
	for _, q := range t.Questions[HoldPointFamily] {
		if q.Options != nil || q.Amber == nil || !noulCutOK(q) {
			continue
		}
		switch {
		case q.Green != nil && probability >= *q.Green:
			return BandGreen, true
		case probability >= *q.Amber:
			return BandAmber, true
		default:
			return BandBlank, false
		}
	}
	return BandBlank, false
}

// ReleaseCalibrated reports whether a release cut-off exists, for the dev
// status line ("hold_point_release: not calibrated").
func (t Thresholds) ReleaseCalibrated() bool {
	if !t.versionOK(HoldPointFamily) {
		return false
	}
	for _, q := range t.Questions[HoldPointFamily] {
		if q.Options == nil && q.Amber != nil && noulCutOK(q) {
			return true
		}
	}
	return false
}

func noulCutOK(q Threshold) bool {
	if q.Amber == nil || *q.Amber < 0 || *q.Amber > 1 {
		return false
	}
	return q.Green == nil || *q.Green >= *q.Amber && *q.Green <= 1
}
