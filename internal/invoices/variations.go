package invoices

import "time"

// Variations to a live contract (docs/plans/2026-11-17-variations-progress-claims.md,
// Tasks 3 and 4). A variation's money reaches the cost plan through its
// line: approved amounts as Approved Variations, open ones as Forecast
// Variations at the PM's estimate, else the claimed amount. Nothing here is
// stored as a total; every sum is worked out at read.

// Variation statuses.
const (
	VariationProposed  = "proposed"
	VariationSubmitted = "submitted"
	VariationApproved  = "approved"
	VariationRejected  = "rejected"
	VariationWithdrawn = "withdrawn"
)

// VariationStatuses in the order the status menu lists them.
var VariationStatuses = []string{VariationProposed, VariationSubmitted, VariationApproved, VariationRejected, VariationWithdrawn}

// Variation is one row of a contract's variation register.
type Variation struct {
	ID, PackageID, AwardID, LineID string
	Number, Title, Details, Status string
	Claimed, Estimate, Approved    *int64
	TimeDays                       *int
	SubmittedOn, DecidedOn         string
	Documents                      []string
	DocumentNames                  []string // the documents' file names, in Documents' order
	Version                        int64
	CreatedAt                      time.Time
}

// Open is a variation still waiting for a decision.
func (v Variation) Open() bool {
	return v.Status == VariationProposed || v.Status == VariationSubmitted
}

// Forecast is what an open variation is expected to cost: the PM's
// estimate, else the amount claimed (labelled so), else unknown. A decided
// variation forecasts nothing.
func (v Variation) Forecast() (*int64, string) {
	switch {
	case !v.Open():
		return nil, ""
	case v.Estimate != nil:
		return v.Estimate, "estimate"
	case v.Claimed != nil:
		return v.Claimed, "claimed"
	}
	return nil, "unknown"
}

// LineVariations are one cost line's variation sums. Forecast holds only
// the known forecasts; Unknown counts open variations with none, which
// leaves the forecast partial.
type LineVariations struct {
	Approved      int64
	HasApproved   bool
	Forecast      int64
	HasForecast   bool
	Unknown       int
	ClaimedBasis  int // open variations forecast at the claimed amount
	ApprovedCount int
	OpenCount     int
}

// VariationsByLine sums each line's variations.
func VariationsByLine(vs []Variation) map[string]LineVariations {
	out := map[string]LineVariations{}
	for _, v := range vs {
		l := out[v.LineID]
		switch {
		case v.Status == VariationApproved && v.Approved != nil:
			l.Approved += *v.Approved
			l.HasApproved = true
			l.ApprovedCount++
		case v.Open():
			l.OpenCount++
			f, basis := v.Forecast()
			if f == nil {
				l.Unknown++
			} else {
				l.Forecast += *f
				l.HasForecast = true
				if basis == "claimed" {
					l.ClaimedBasis++
				}
			}
		default:
			continue
		}
		out[v.LineID] = l
	}
	return out
}

// ContractTotals are a contract's header figures: the contract sum (its
// lines' approved contract, unknown when any line's is), its approved
// variations, and the contract sum to date (the two together).
func ContractTotals(lines []*int64, vs []Variation) (sum *int64, approved int64, toDate *int64) {
	var s int64
	known := len(lines) > 0
	for _, l := range lines {
		if l == nil {
			known = false
			break
		}
		s += *l
	}
	for _, v := range vs {
		if v.Status == VariationApproved && v.Approved != nil {
			approved += *v.Approved
		}
	}
	if !known {
		return nil, approved, nil
	}
	t := s + approved
	return &s, approved, &t
}
