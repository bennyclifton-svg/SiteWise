package eval

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/intake"
	"sitewise/internal/latency"
)

// Report is the committed result of one evaluation. It holds counts, hashes
// and aggregate metrics only: no document text, titles or numbers.
type Report struct {
	SchemaVersion   int                                           `json:"schema_version"`
	Mode            string                                        `json:"mode"`
	Model           string                                        `json:"model"`
	QuestionVersion string                                        `json:"question_version"`
	GeneratedAt     time.Time                                     `json:"generated_at"`
	Inputs          map[string]string                             `json:"inputs"`
	Cases           CaseCounts                                    `json:"cases"`
	Metrics         map[string]map[string]map[string]FieldMetrics `json:"metrics"`
	Supersession    map[string]SupersessionCounts                 `json:"supersession"`
	Jev             JevCounts                                     `json:"jev"`
	Fit             []FitResult                                   `json:"fit,omitempty"`
	Notes           []string                                      `json:"notes"`
	Verdict         GateResult                                    `json:"gate"`
}

// CaseCounts discloses sample sizes per split and what was left out.
type CaseCounts struct {
	Total        int            `json:"total"`
	BySplit      map[string]int `json:"by_split"`
	Families     map[string]int `json:"families"`
	NotFiled     map[string]int `json:"not_filed"`
	Excluded     map[string]int `json:"excluded"`
	Adjudication map[string]int `json:"adjudication"`
}

// JevCounts are call outcomes. Recorded latency comes from the live
// recording: a replay's own timing says nothing about the provider.
type JevCounts struct {
	Calls             int   `json:"calls"`
	Errors            int   `json:"errors"`
	GreyFilings       int   `json:"grey_filings"`
	ReplayMisses      int   `json:"replay_misses"`
	RecordedP50US     int64 `json:"recorded_p50_us,omitempty"`
	RecordedP90US     int64 `json:"recorded_p90_us,omitempty"`
	OverDeadline      int   `json:"recorded_over_interactive_deadline"`
	InteractiveDeadUS int64 `json:"interactive_deadline_us"`
}

// GateResult is the held-out verdict.
type GateResult struct {
	Passed   bool     `json:"passed"`
	Failures []string `json:"failures"`
}

// Baseline is the accepted held-out result a later run must not fall below.
type Baseline struct {
	AcceptedAt   time.Time               `json:"accepted_at"`
	Inputs       map[string]string       `json:"inputs"`
	Metrics      map[string]FieldMetrics `json:"metrics"`
	Supersession SupersessionCounts      `json:"supersession"`
}

// NewReport summarises run per split and label tier.
func NewReport(m Manifest, set CaseSet, run *RunResult, mode string) Report {
	r := Report{
		SchemaVersion:   1,
		Mode:            mode,
		Model:           config.PinnedJevModel,
		QuestionVersion: intake.QuestionVersion,
		GeneratedAt:     time.Now().UTC(),
		Inputs:          map[string]string{},
		Metrics:         map[string]map[string]map[string]FieldMetrics{},
		Supersession:    map[string]SupersessionCounts{},
		Cases: CaseCounts{
			Total:        len(set.Cases),
			BySplit:      map[string]int{},
			Families:     map[string]int{},
			NotFiled:     map[string]int{},
			Excluded:     set.Excluded,
			Adjudication: map[string]int{},
		},
		Jev: JevCounts{Calls: run.Calls, Errors: run.Errors, ReplayMisses: run.Misses, InteractiveDeadUS: interactiveDeadline.Microseconds()},
	}
	for k, v := range set.Keys {
		r.Inputs["key:"+k] = v
	}
	families := map[string]map[string]bool{}
	for _, c := range set.Cases {
		r.Cases.BySplit[c.Split]++
		r.Cases.Adjudication[c.Adjudged]++
		if families[c.Split] == nil {
			families[c.Split] = map[string]bool{}
		}
		families[c.Split][c.Family] = true
	}
	for split, f := range families {
		r.Cases.Families[split] = len(f)
	}
	for _, c := range run.Cases {
		if c.NotFiled != "" {
			r.Cases.NotFiled[c.NotFiled]++
		}
		if c.Grey {
			r.Jev.GreyFilings++
		}
	}
	for _, split := range []string{SplitCalibration, SplitHeldout} {
		r.Metrics[split] = map[string]map[string]FieldMetrics{
			TierAuthoritative: Summarize(run.Outcomes(split, true), m.Gate.Z),
			TierDiagnostic:    Summarize(run.Outcomes(split, false), m.Gate.Z),
		}
		r.Supersession[split] = run.Supersession(split)
	}
	r.Notes = []string{
		"Every answer key is adjudication: unreviewed. Authoritative means transcribed from a register or transmittal, not owner-reviewed.",
		"Accuracy is over applied values with a Wilson interval at the gate z; coverage is applied over scored. Read both with the sample sizes above.",
		"Discipline and kind labels are model or folder judgements in every key and are diagnostic only; they neither fit thresholds nor gate.",
	}
	return r
}

// SetRecordedLatency fills provider latency from the live recording.
func (r *Report) SetRecordedLatency(records []Record) {
	var us []int64
	for _, rec := range records {
		us = append(us, rec.LatencyUS)
		if rec.LatencyUS > interactiveDeadline.Microseconds() {
			r.Jev.OverDeadline++
		}
	}
	if len(us) == 0 {
		return
	}
	r.Jev.RecordedP50US, _ = latency.Percentile(us, 0.5)
	r.Jev.RecordedP90US, _ = latency.Percentile(us, 0.9)
}

// Gate lists every reason the held-out result cannot be released. A nil
// baseline fails: the first baseline is accepted explicitly after review.
func (r Report) Gate(m Manifest, base *Baseline) []string {
	var out []string
	if r.Cases.Total < m.Gate.MinCases {
		out = append(out, fmt.Sprintf("too few cases: %d, need %d", r.Cases.Total, m.Gate.MinCases))
	}
	if r.Jev.ReplayMisses > 0 {
		out = append(out, fmt.Sprintf("%d requests had no recording", r.Jev.ReplayMisses))
	}
	held := r.Metrics[SplitHeldout][TierAuthoritative]
	fields := make([]string, 0, len(m.Gate.MinHeldout))
	for f := range m.Gate.MinHeldout {
		fields = append(fields, f)
	}
	sort.Strings(fields)
	for _, f := range fields {
		if got := held[f].Scored; got < m.Gate.MinHeldout[f] {
			out = append(out, fmt.Sprintf("too few held-out %s samples: %d, need %d", f, got, m.Gate.MinHeldout[f]))
		}
	}
	for _, split := range []string{SplitCalibration, SplitHeldout} {
		s := r.Supersession[split]
		if bad := s.Incorrect + s.Unverified; bad > m.Gate.MaxIncorrectSupersession {
			out = append(out, fmt.Sprintf("%s: %d incorrect or unverified automatic supersession links", split, bad))
		}
	}
	if base == nil {
		out = append(out, "no accepted baseline: review this report, then run with -accept")
		return out
	}
	out = append(out, Regressions(base.Metrics, held)...)
	return out
}

// Baseline is this report's held-out authoritative result.
func (r Report) Baseline() Baseline {
	return Baseline{
		AcceptedAt:   time.Now().UTC(),
		Inputs:       r.Inputs,
		Metrics:      r.Metrics[SplitHeldout][TierAuthoritative],
		Supersession: r.Supersession[SplitHeldout],
	}
}

// thresholdFields are the questions intake asks Jev.
var thresholdFields = []string{
	intake.FieldKind, intake.FieldDiscipline, intake.FieldLifecycle,
	intake.FieldNumber, intake.FieldRevision, intake.FieldTitle, intake.FieldSupersedes,
}

// ThresholdsFrom turns fits into the production calibration file. A fit with
// unknown cut-offs is left out, so that shape stays unknown at runtime.
func ThresholdsFrom(fits []FitResult, reconciliation string, meta any) intake.Thresholds {
	t := intake.Thresholds{
		QuestionVersion: intake.QuestionVersion,
		Reconciliation:  reconciliation,
		Questions:       map[string][]intake.Threshold{},
	}
	for _, f := range thresholdFields {
		t.Questions[f] = []intake.Threshold{}
	}
	for _, fit := range fits {
		if fit.Amber == nil {
			continue
		}
		lo, hi := fit.Shape.Min, fit.Shape.Max
		a := *fit.Amber
		th := intake.Threshold{Options: &lo, Amber: &a, N: fit.N}
		if fit.Green != nil {
			g := *fit.Green
			th.Green = &g
		}
		if hi != lo {
			th.MaxOptions = &hi
		}
		t.Questions[fit.Field] = append(t.Questions[fit.Field], th)
	}
	if meta != nil {
		if raw, err := json.Marshal(meta); err == nil {
			t.Fit = raw
		}
	}
	return t
}
