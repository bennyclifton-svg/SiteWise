package invoices

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"sitewise/internal/jev"
)

// QuestionVersion is the invoice question map this process asks. Thresholds
// calibrated for another version do not apply, and changing any wording or
// option shape is a new version that needs a new calibration run.
const QuestionVersion = "invoice-1"

// Threshold is one calibrated cut-off for a question and option shape. A nil
// cut-off is unknown and authorises nothing. N is the calibration evidence.
type Threshold struct {
	Options    *int     `json:"options"`
	MaxOptions *int     `json:"max_options,omitempty"`
	Green      *float64 `json:"green"`
	Amber      *float64 `json:"amber"`
	N          int      `json:"n,omitempty"`
	Reason     string   `json:"reason,omitempty"`
}

// Thresholds is data/invoice/thresholds.json, written by cmd/invoice-eval.
type Thresholds struct {
	QuestionVersion string                 `json:"question_version"`
	Calibration     string                 `json:"calibration"`
	Policy          json.RawMessage        `json:"policy,omitempty"`
	Questions       map[string][]Threshold `json:"questions"`
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
		return Thresholds{}, fmt.Errorf("invoice threshold version %q, want %q", t.QuestionVersion, QuestionVersion)
	}
	for field, list := range t.Questions {
		for _, q := range list {
			if q.Options == nil {
				continue
			}
			lo, hi := q.shape()
			if lo < 2 || hi > jev.MaxChoiceOptions || hi < lo {
				return Thresholds{}, fmt.Errorf("%s option shape", field)
			}
			if q.Amber != nil && (*q.Amber < 0 || *q.Amber > 1) || q.Green != nil && (*q.Green < 0 || *q.Green > 1 || q.Amber == nil || *q.Amber > *q.Green) {
				return Thresholds{}, fmt.Errorf("%s threshold is out of range", field)
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

// Band reports the colour for one answer, or apply false when no calibrated
// cut-off covers this question, version and option count.
//
//	https://docs.typesafe.ai/confidence
//	https://docs.typesafe.ai/patterns/confidence-routing
func (t Thresholds) Band(field string, confidence *float64, options int) (band string, apply bool) {
	if t.QuestionVersion != QuestionVersion || confidence == nil {
		return BandBlank, false
	}
	for _, q := range t.Questions[field] {
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
