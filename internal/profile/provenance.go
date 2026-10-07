package profile

import (
	"strings"

	"sitewise/internal/knowledge"
)

// Provenance vocabularies (plan §4.1). Origin, review status and meaning are
// independent: accepting a value never makes it verified, and an assumption
// stays an assumption.
const (
	OriginDocument    = "document"
	OriginUser        = "user"
	OriginCalculation = "calculation"
	OriginAssumption  = "assumption"

	ReviewProposed  = "proposed"
	ReviewAccepted  = "accepted_for_planning"
	ReviewVerified  = "verified"
	ReviewSupersede = "superseded"

	MeaningStated      = "stated"
	MeaningRequirement = "requirement"
	MeaningAllowance   = "allowance"
	MeaningForecast    = "forecast"

	StateSet     = "set"
	StateCleared = "cleared"
	StateUnknown = "unknown"
	// StateAbsent is a row with no value that nobody marked unknown: the
	// documents did not settle it. Only a person (or a planning value) can
	// record an explicit unknown (L274).
	StateAbsent = "absent"
)

// eligibleUser is decision D-06: a value the user states as fact may feed a
// regulatory derivation; an assumption, allowance, forecast or requirement
// never does. Accepting a planning assumption must not make it verifiable.
func eligibleUser(r *Row) bool {
	return r.Band == bandUser && r.Value != "" &&
		orDefault(r.Origin, OriginUser) == OriginUser && orDefault(r.Meaning, MeaningStated) == MeaningStated
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func userState(u UserValue) string {
	if u.State != "" {
		return u.State
	}
	if u.Value == nil {
		return StateCleared
	}
	return StateSet
}

// Annotate fills each row's scope, origin, review status, meaning and value
// state from how the row was made. It never changes a value or band.
func Annotate(rows []Row, cat *knowledge.Catalog) {
	for i := range rows {
		r := &rows[i]
		r.Scope = cat.KeyScope(r.Key)
		if r.Scope == "" {
			r.Scope = knowledge.ScopeProject
		}
		switch {
		case r.Band == bandUser:
			r.ReviewStatus = orDefault(r.ReviewStatus, ReviewAccepted)
			// Origin, meaning and state were copied from the user value.
		case r.Band == bandPlanning:
			// Copied from the planning value; never eligible (D-06).
		case r.Derived != nil:
			r.Origin, r.ReviewStatus, r.Meaning = OriginCalculation, orDefault(r.ReviewStatus, ReviewAccepted), MeaningStated
		case r.Band == bandSuggest:
			r.Origin, r.ReviewStatus, r.Meaning = OriginCalculation, ReviewProposed, MeaningStated
		default:
			r.Origin, r.ReviewStatus, r.Meaning = OriginDocument, ReviewProposed, assertionMeaning(r.Assertion)
		}
		if r.ValueState == "" {
			r.ValueState = StateSet
			if strings.TrimSpace(r.Value) == "" {
				r.ValueState = StateAbsent
			}
		}
	}
}

// assertionMeaning maps how a passage presents a value to its meaning.
func assertionMeaning(a string) string {
	switch a {
	case "required":
		return MeaningRequirement
	case "allowance":
		return MeaningAllowance
	}
	return MeaningStated
}
