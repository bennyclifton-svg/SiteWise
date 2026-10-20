package programread

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sitewise/internal/jev"
)

// QuestionVersion is the program_row question map this process asks
// (plan 10.2). Thresholds calibrated for another version do not apply, and
// any change to the wording or the option shape is a new version that
// needs a new calibration run (plan section 2, Jev rule 11).
const QuestionVersion = "program-row-1"

// Family is the threshold family of the one question this reading asks.
const Family = "program_row"

// familyVersions is the question map each family's cut-offs belong to. A
// family not listed is part of QuestionVersion's map. Reading B asks its
// own map (plan 13.3 Q3), so its cut-offs are versioned on their own.
var familyVersions = map[string]string{Family: QuestionVersion, StageWeeksFamily: StageWeeksQuestionVersion,
	HoldPointFamily: HoldPointReleaseVersion}

// Threshold is one calibrated cut-off for a family and option shape. A nil
// cut-off is unknown and authorises nothing. N is the calibration evidence.
type Threshold struct {
	Options    *int     `json:"options"`
	MaxOptions *int     `json:"max_options,omitempty"`
	Green      *float64 `json:"green"`
	Amber      *float64 `json:"amber"`
	N          int      `json:"n"`
	Reason     string   `json:"reason,omitempty"`
}

// Thresholds is data/program/thresholds.json, written by cmd/program-eval in
// the shape of data/proposal/thresholds.json.
type Thresholds struct {
	QuestionVersion string          `json:"question_version"`
	Calibration     string          `json:"calibration"`
	Policy          json.RawMessage `json:"policy,omitempty"`
	// Versions names the question version a family was calibrated for
	// when it is not QuestionVersion: "program_stage_weeks":
	// "proposal-program-1".
	Versions  map[string]string      `json:"question_versions,omitempty"`
	Questions map[string][]Threshold `json:"questions"`
}

// versionOK reports whether the file's cut-offs for family were fitted on
// the question map this build asks for that family.
func (t Thresholds) versionOK(family string) bool {
	want, ok := familyVersions[family]
	if !ok {
		want = QuestionVersion
	}
	got := t.Versions[family]
	if got == "" {
		got = t.QuestionVersion
	}
	return got == want
}

// LoadThresholds reads <dir>/thresholds.json. A missing file is no
// calibration: every Jev answer stays blank until one is fitted.
func LoadThresholds(dir string) (Thresholds, error) {
	body, err := os.ReadFile(filepath.Join(dir, "thresholds.json"))
	if os.IsNotExist(err) {
		return Thresholds{QuestionVersion: QuestionVersion}, nil
	}
	if err != nil {
		return Thresholds{}, err
	}
	var t Thresholds
	if err := json.Unmarshal(body, &t); err != nil {
		return Thresholds{}, err
	}
	if t.QuestionVersion != QuestionVersion {
		return Thresholds{}, fmt.Errorf("program threshold version %q, want %q", t.QuestionVersion, QuestionVersion)
	}
	for family := range t.Versions {
		if !t.versionOK(family) {
			return Thresholds{}, fmt.Errorf("%s threshold version %q, want %q", family, t.Versions[family], familyVersions[family])
		}
	}
	for family, list := range t.Questions {
		for _, q := range list {
			if q.Options == nil {
				continue
			}
			lo, hi := q.shape()
			if lo < 2 || hi > jev.MaxChoiceOptions || hi < lo {
				return Thresholds{}, fmt.Errorf("%s option shape", family)
			}
			if q.Amber != nil && (*q.Amber < 0 || *q.Amber > 1) || q.Green != nil && (*q.Green < 0 || *q.Green > 1 || q.Amber == nil || *q.Amber > *q.Green) {
				return Thresholds{}, fmt.Errorf("%s threshold is out of range", family)
			}
		}
	}
	return t, nil
}

func (q Threshold) shape() (lo, hi int) {
	lo = *q.Options
	hi = lo
	if q.MaxOptions != nil {
		hi = *q.MaxOptions
	}
	return lo, hi
}

// Calibrated reports whether any cut-off exists for the family, for the dev
// status line ("program_row: not calibrated").
func (t Thresholds) Calibrated(family string) bool {
	if !t.versionOK(family) {
		return false
	}
	for _, q := range t.Questions[family] {
		if q.Amber != nil && q.Options != nil {
			return true
		}
	}
	return false
}

// Band reports the colour for one answer, or apply false when no calibrated
// cut-off covers this family, version and option count. Choice confidence
// depends on the option count, so a cut-off never transfers between shapes.
//
//	https://docs.typesafe.ai/confidence
//	https://docs.typesafe.ai/patterns/confidence-routing
func (t Thresholds) Band(family string, confidence *float64, options int) (band string, apply bool) {
	if !t.versionOK(family) || confidence == nil {
		return BandBlank, false
	}
	for _, q := range t.Questions[family] {
		if q.Amber == nil || q.Options == nil {
			continue
		}
		if lo, hi := q.shape(); options < lo || options > hi {
			continue
		}
		switch {
		case q.Green != nil && *confidence >= *q.Green:
			return BandGreen, true
		case *confidence >= *q.Amber:
			return BandAmber, true
		default:
			return BandBlank, false
		}
	}
	return BandBlank, false
}
