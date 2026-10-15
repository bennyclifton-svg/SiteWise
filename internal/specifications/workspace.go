package specifications

import (
	"fmt"
	"slices"
	"strings"

	"sitewise/internal/knowledge"
)

// Coverage is a team's explicit scope decision, not an inference from upload.
type Coverage struct {
	ID         string              `json:"id"`
	SystemID   string              `json:"system_id"`
	TypeID     string              `json:"type_id,omitempty"`
	Mode       string              `json:"mode"`
	Extent     string              `json:"extent"`
	Exclusions string              `json:"exclusions"`
	Maintainer string              `json:"maintainer"`
	Reviewer   string              `json:"reviewer"`
	References []DocumentReference `json:"references"`
	Reason     string              `json:"reason"`
	Version    int64               `json:"version"`
}

type DocumentReference struct {
	DocumentID  string `json:"document_id"`
	Filename    string `json:"filename"`
	Number      string `json:"number"`
	Revision    string `json:"revision"`
	Hash        string `json:"hash"`
	Location    string `json:"location"`
	Unavailable bool   `json:"unavailable,omitempty"`
	Stale       bool   `json:"stale,omitempty"`
}

// Scope is read from work_items and the Profiler. It is never edited here.
type Scope struct {
	WorkItemID string `json:"work_item_id"`
	SystemID   string `json:"system_id"`
	PartID     string `json:"part_id"`
	Location   string `json:"location"`
	Title      string `json:"title"`
	Action     string `json:"action"`
	Inclusion  string `json:"inclusion"`
	Provider   string `json:"provider"`
	Retired    bool   `json:"retired"`
	Version    int64  `json:"version"`
}

type Section struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Systems  []SystemOption `json:"systems"`
	Starters []Starter      `json:"starters"`
}
type SystemOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type Starter struct {
	Key      string `json:"key"`
	Code     string `json:"code"`
	Title    string `json:"title"`
	SystemID string `json:"system_id"`
	Kind     string `json:"kind"`
}

// These are empty authoring prompts, not products, scope or technical rules.
var starterItems = []Starter{
	{"earthworks", "EW1", "Earthworks", "site.earthworks", "assembly"},
	{"paving", "PV1", "External paving", "site.vehicle-access", "material"},
	{"landscape", "LA1", "Landscape works", "site.landscape", "assembly"},
	{"footings", "FO1", "Footings", "substructure.footings", "assembly"},
	{"slab", "SL1", "Slab on ground", "substructure.slab-on-ground", "assembly"},
	{"frame", "FR1", "Structural framing", "structure", "assembly"},
	{"cladding", "CL1", "External cladding", "envelope.external-walls", "assembly"},
	{"roof", "RF1", "Roof covering", "envelope.roof-coverings", "material"},
	{"windows", "WN1", "Windows and external doors", "envelope.windows-doors", "product"},
	{"insulation", "IN1", "Thermal insulation", "envelope.thermal-insulation", "material"},
	{"basin", "BS1", "Basin", "hydraulic.sanitary", "product"},
	{"mixer", "BM1", "Basin mixer", "hydraulic.cold-water", "product"},
	{"shower", "SH1", "Shower fittings", "hydraulic.cold-water", "product"},
	{"toilet", "WC1", "Toilet suite", "hydraulic.sanitary", "product"},
	{"waste", "FW1", "Floor waste", "hydraulic.sanitary", "product"},
	{"hot-water", "HW1", "Hot-water system", "hydraulic.hot-water", "equipment"},
	{"air-conditioning", "AC1", "Heating and cooling equipment", "mechanical.air-conditioning", "equipment"},
	{"ventilation", "VE1", "Ventilation", "mechanical.ventilation", "assembly"},
	{"lighting", "LF1", "Lighting fittings", "electrical.lighting", "product"},
	{"power", "EL1", "Power and distribution", "electrical.switchboards", "assembly"},
	{"solar", "PV2", "Solar equipment", "electrical.solar-pv", "equipment"},
	{"wall-linings", "WL1", "Wall linings", "interiors.walls-linings", "material"},
	{"ceilings", "CE1", "Ceilings", "interiors.ceilings", "assembly"},
	{"floor-tile", "FT1", "Floor tile", "interiors.tiling", "finish"},
	{"wall-tile", "WT1", "Wall tile", "interiors.tiling", "finish"},
	{"waterproofing", "WP1", "Wet-area waterproofing", "interiors.wet-area-waterproofing", "assembly"},
	{"flooring", "FL1", "Floor finishes", "interiors.floor-finishes", "finish"},
	{"paint", "P1", "Paint system", "interiors.painting", "finish"},
	{"joinery", "JN1", "Joinery", "interiors.joinery", "assembly"},
	{"doors", "ID1", "Internal doors and hardware", "interiors.internal-doors", "product"},
	{"mirror", "MR1", "Mirror", "interiors.ffe", "product"},
	{"hook", "RH1", "Robe hook", "interiors.ffe", "product"},
	{"rail", "TR1", "Towel rail", "interiors.ffe", "product"},
}

func StarterFor(key string) (Starter, bool) {
	for _, s := range starterItems {
		if s.Key == key {
			return s, true
		}
	}
	return Starter{}, false
}

func Sections(cat *knowledge.Catalog, snap Snapshot) []Section {
	out := []Section{}
	if cat == nil {
		return out
	}
	scoped := map[string]bool{}
	scopeKnown := false
	for _, s := range snap.Scope {
		scopeKnown = scopeKnown || (!s.Retired && (s.Inclusion == "included" || s.Inclusion == "excluded"))
		if !s.Retired && s.Inclusion != "excluded" {
			scoped[s.SystemID] = true
		}
	}
	for _, top := range cat.BuildingOrderSystems() {
		if top.Status == "deprecated" {
			continue
		}
		section := Section{ID: top.ID, Label: top.Label, Systems: []SystemOption{}, Starters: []Starter{}}
		for _, leaf := range cat.Leaves() {
			if leaf.Parent == top.ID && leaf.Status != "deprecated" {
				section.Systems = append(section.Systems, SystemOption{leaf.ID, leaf.Label})
			}
		}
		for _, starter := range starterItems {
			if strings.Split(starter.SystemID, ".")[0] != top.ID {
				continue
			}
			// No scope yet: show the framework. Otherwise follow the Profiler's scope.
			applicable := scoped[starter.SystemID] || scoped[top.ID]
			if starter.SystemID == top.ID {
				for id := range scoped {
					applicable = applicable || strings.HasPrefix(id, top.ID+".")
				}
			}
			if scopeKnown && !applicable {
				continue
			}
			section.Starters = append(section.Starters, starter)
		}
		// Specialised commercial/industrial systems get their own scoped prompt.
		for _, leaf := range section.Systems {
			if !scoped[leaf.ID] {
				continue
			}
			found := false
			for _, s := range section.Starters {
				found = found || s.SystemID == leaf.ID
			}
			if !found {
				section.Starters = append(section.Starters, Starter{Key: "system:" + leaf.ID, Title: leaf.Label, SystemID: leaf.ID, Kind: "assembly"})
			}
		}
		out = append(out, section)
	}
	return out
}

func ValidateCoverage(c Coverage) error {
	if !slices.Contains([]string{"needs_attention", "managed_here", "external", "partial", "not_applicable"}, c.Mode) || !small(c.Extent, 1000) || !small(c.Exclusions, 1000) || !small(c.Maintainer, 200) || !small(c.Reviewer, 200) || !small(c.Reason, 1000) || len(c.References) > 20 {
		return fmt.Errorf("invalid specification coverage")
	}
	if (c.Mode == "external" || c.Mode == "partial") && (len(c.References) == 0 || strings.TrimSpace(c.Extent) == "" || strings.TrimSpace(c.Maintainer) == "") {
		return fmt.Errorf("external coverage needs a document, scope extent and specification maintainer")
	}
	for _, r := range c.References {
		if r.DocumentID == "" || !small(r.Location, 300) {
			return fmt.Errorf("provide a valid document reference")
		}
	}
	return nil
}

func EffectiveCoverage(s Snapshot, t Type) *Coverage {
	var result *Coverage
	for i := range s.Coverage {
		c := &s.Coverage[i]
		if c.TypeID == t.ID {
			return c
		}
		if c.TypeID == "" && (c.SystemID == t.SystemID || c.SystemID == strings.Split(t.SystemID, ".")[0]) {
			if result == nil || c.SystemID == t.SystemID {
				result = c
			}
		}
	}
	return result
}

func External(s Snapshot, t Type) bool {
	if c := EffectiveCoverage(s, t); c != nil && c.Mode == "external" {
		return true
	}
	return t.Authority == "external" || (t.Authority == "" && t.Origin == "document")
}
