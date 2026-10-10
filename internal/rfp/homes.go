package rfp

import (
	"strings"
	"unicode/utf8"

	"sitewise/internal/knowledge"
)

// Every project fact an RFP reads has one home in Profile (owner decision
// D1, 10 October 2026). A fact edited in the RFP is written there.

// Home kinds.
const (
	// HomeDeterminant is a yes/no determinant: yes is stated_true.
	HomeDeterminant = "determinant"
	// HomeDetail is a project detail, proj.<key> or proj.flag.<key>.
	HomeDetail = "detail"
	// HomeHeader is a profile header value (hdr.*), written as the value.
	HomeHeader = "header"
	// HomeSite is the site record's address or lot.
	HomeSite = "site"
	// HomeCostPlan is the cost plan's budget; the report links to Costs.
	HomeCostPlan = "cost_plan"
	// HomeLink is a profile field the RFP cannot write as yes/no; the
	// report links to Profile instead.
	HomeLink = "link"
	// HomeDerived follows from other facts and is never written.
	HomeDerived = "derived"
)

type Home struct {
	Key  string `json:"key"`
	Kind string `json:"kind"`
}

// flagDeterminants are the flags whose determinant the profile can write.
// det.groundwater_above_excavation has no profile group, so groundwater is
// a project detail; its reading still counts as evidence (readFlags).
var flagDeterminants = map[string]string{
	"bushfire": "det.bushfire_prone_land", "flood": "det.flood_hazard",
	"contamination": "det.potentially_contaminated_land", "acid_sulfate": "det.acid_sulfate_soils_mapped",
	"existing_building": "det.existing_building",
}

// FlagHome is where a skeleton flag lives in Profile.
func FlagHome(flag string) Home {
	if key, ok := flagDeterminants[flag]; ok {
		return Home{key, HomeDeterminant}
	}
	switch flag {
	case "heritage":
		return Home{"det.heritage_status", HomeLink}
	case "live_operations":
		return Home{"hdr.cond.operational_constraints", HomeLink}
	case "mixed_use":
		return Home{"", HomeDerived}
	}
	return Home{knowledge.DetailPrefix + "flag." + flag, HomeDetail}
}

// DetailFlags are the skeleton flags stored as proj.flag.<key>.
func DetailFlags(sk knowledge.RFPSkeleton) []knowledge.RFPOption {
	var out []knowledge.RFPOption
	for _, f := range sk.Flags {
		if FlagHome(f.Key).Kind == HomeDetail {
			out = append(out, f)
		}
	}
	return out
}

// factReader is a DriverSet or a Resolved RFP.
type factReader interface{ Fact(string) Fact }

// FactHome is where an RFP fact lives in Profile. Name is a driver fact
// (work_type, flag.basement) or a profile-homed field (client, zone).
// Writable reports whether an edit in the RFP may write it there.
func FactHome(sk knowledge.RFPSkeleton, facts factReader, name string) Home {
	if flag, ok := strings.CutPrefix(name, "flag."); ok {
		for _, f := range sk.Flags {
			if f.Key == flag {
				return FlagHome(flag)
			}
		}
		return Home{}
	}
	for _, f := range sk.Fields {
		if f.Key == name && f.Profile != "" {
			if strings.HasPrefix(f.Profile, "site.") {
				return Home{f.Profile, HomeSite}
			}
			return Home{f.Profile, HomeDetail}
		}
	}
	switch name {
	case "work_type":
		return Home{"hdr.work_type", HomeHeader}
	case "pathway":
		return Home{"hdr.cond.planning", HomeHeader}
	case "procurement":
		return Home{"hdr.cond.procurement_route", HomeHeader}
	case "site_area":
		return Home{"hdr.scale.site_sqm", HomeHeader}
	case "gfa":
		// Write where the figure was read, so the edit is the one the RFP uses.
		if d := facts.Fact("gfa").Detail; strings.HasPrefix(d, "hdr.scale.") {
			return Home{d, HomeHeader}
		}
		return Home{"hdr.scale.gfa_sqm", HomeHeader}
	case "regulated_design":
		return Home{knowledge.DetailPrefix + "regulated_design", HomeDetail}
	case "budget":
		return Home{"", HomeCostPlan}
	}
	return Home{}
}

// Writable reports whether the RFP may write this home.
func (h Home) Writable() bool {
	switch h.Kind {
	case HomeDeterminant, HomeDetail, HomeHeader, HomeSite:
		return true
	}
	return false
}

// ValidateFact checks a value the user gives a fact in the RFP. "unknown"
// records that it is not known; "" clears the user's word.
func ValidateFact(sk knowledge.RFPSkeleton, name, v string) error {
	if v == "" || v == "unknown" {
		return nil
	}
	if strings.HasPrefix(name, "flag.") || name == "regulated_design" {
		if v == "yes" || v == "no" {
			return nil
		}
		return ErrInvalidSetting
	}
	var list []knowledge.RFPOption
	switch name {
	case "pathway":
		list = sk.Pathways
	case "procurement":
		list = sk.Procurement
	case "work_type":
		list = sk.WorkTypes
	case "site_area", "gfa":
		if numberRE.MatchString(v) {
			return nil
		}
		return ErrInvalidSetting
	default:
		// Profile-homed text fields; Profile keeps at most 200 characters.
		if utf8.RuneCountInString(v) <= 200 && printable(v) && !strings.Contains(v, "\n") {
			return nil
		}
		return ErrInvalidSetting
	}
	for _, o := range list {
		if o.Key == v {
			return nil
		}
	}
	return ErrInvalidSetting
}
