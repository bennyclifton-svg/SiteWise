package rfp

import "sitewise/internal/knowledge"

// Every project fact an RFP reads has one home in Profile (owner decision
// D1, 10 October 2026). A fact edited in the RFP is written there.

// Home kinds.
const (
	// HomeDeterminant is a yes/no determinant: yes is stated_true.
	HomeDeterminant = "determinant"
	// HomeDetail is a project detail, proj.<key> or proj.flag.<key>.
	HomeDetail = "detail"
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
