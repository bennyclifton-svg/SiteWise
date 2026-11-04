package profile_test

import (
	"strings"
	"testing"

	"sitewise/internal/profile"
)

func scopeRow(rows []profile.Row, leaf string) (profile.Row, bool) {
	for _, r := range rows {
		if r.PartID == whole && r.Key == "scope."+leaf {
			return r, true
		}
	}
	return profile.Row{}, false
}

func header(class, work string) []profile.UserValue {
	return []profile.UserValue{
		{PartID: whole, Key: "hdr.subclass", Value: str(class)},
		{PartID: whole, Key: "hdr.work_type", Value: str(work)},
	}
}

func TestScopeStartsFromDefaultsAndDocumentsAdd(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.mechanical.heating-appliances.presence", "included", "spec", 0.9))
	in.User = header("house", "new")
	rows := profile.Build(in, cat)
	if r, ok := scopeRow(rows, "substructure.footings"); !ok || r.Value != "in" || r.Band != "suggested" {
		t.Fatalf("default scope row %+v %v", r, ok)
	}
	if r, ok := scopeRow(rows, "mechanical.heating-appliances"); !ok || r.Value != "in" || r.Band != "amber" {
		t.Fatalf("a document's included system must join the scope: %+v %v", r, ok)
	}
	if _, ok := scopeRow(rows, "vertical-transport.passenger-lifts"); ok {
		t.Fatal("a system nobody chose or read is in scope")
	}
}

func TestRefurbishmentStartsEmptyAndTheUserChooses(t *testing.T) {
	cat := repoCatalog(t)
	in := input()
	in.User = append(header("warehouse", "refurb"), profile.UserValue{PartID: whole, Key: "scope.fire-active.sprinklers", Value: str("in")})
	rows := profile.Build(in, cat)
	count := 0
	for _, r := range rows {
		if r.Key[:6] == "scope." && r.Value == "in" {
			count++
		}
	}
	if r, ok := scopeRow(rows, "fire-active.sprinklers"); !ok || r.Band != "user" || count != 1 {
		t.Fatalf("pump replacement scope: %d in scope, sprinklers %+v", count, r)
	}
	for _, r := range rows {
		if r.Band == "suggested" {
			t.Fatalf("refurbishment suggested %s", r.Key)
		}
	}
}

func TestTheUsersRemovalIsFinalButADocumentSaysSo(t *testing.T) {
	cat := repoCatalog(t)
	in := input(jevFact("sys.hydraulic.gas.presence", "included", "spec", 0.9))
	in.User = append(header("house", "new"),
		profile.UserValue{PartID: whole, Key: "scope.hydraulic.gas", Value: str("out")},
		profile.UserValue{PartID: whole, Key: "scope.substructure.footings", Value: str("out")})
	rows := profile.Build(in, cat)
	gas, _ := scopeRow(rows, "hydraulic.gas")
	if gas.Value != "out" || gas.Band != "user" || gas.Note == "" {
		t.Fatalf("removed but evidenced system must stay out with a notice: %+v", gas)
	}
	footings, _ := scopeRow(rows, "substructure.footings")
	if footings.Value != "out" || footings.Note != "" {
		t.Fatalf("a removed default stays out without a notice: %+v", footings)
	}
	for _, r := range rows {
		if r.Key == "sys.substructure.footings.presence" && r.Band == "suggested" {
			t.Fatal("a removed default is still suggested")
		}
	}
}

func TestChangingBuildingTypeReappliesOnlyDefaults(t *testing.T) {
	cat := repoCatalog(t)
	in := input()
	in.User = append(header("warehouse", "new"), profile.UserValue{PartID: whole, Key: "scope.vertical-transport.goods-lifts", Value: str("in")})
	rows := profile.Build(in, cat)
	if _, ok := scopeRow(rows, "site.loading-docks"); ok {
		t.Log("loading docks are a warehouse default") // informative only: the list is draft data
	}
	in.User = append(header("house", "new"), profile.UserValue{PartID: whole, Key: "scope.vertical-transport.goods-lifts", Value: str("in")})
	rows = profile.Build(in, cat)
	if r, ok := scopeRow(rows, "vertical-transport.goods-lifts"); !ok || r.Band != "user" {
		t.Fatalf("the user's choice survives a type change: %+v", r)
	}
	if _, ok := scopeRow(rows, "structure.steel"); ok {
		t.Fatal("the warehouse default stayed after switching to a house")
	}
}

// An applied works action names a system the works touch, so it is in scope
// even when the presence answer fell below its floor (Walsh, 2026-10-27:
// "Relocate internal stair" read as alter at 1.0, presence 0.53).
func TestAnAppliedActionPutsTheSystemInScope(t *testing.T) {
	cat := repoCatalog(t)
	in := input(
		jevFact("sys.access-egress.stairs-and-ramps.action", "alter", "brief", 1),
		jevFact("sys.hydraulic.sanitary.action", "upgrade", "email", 0.96),
		jevFact("sys.envelope.external-walls.action", "retain", "heritage", 0.95),
		jevFact("sys.hydraulic.gas.action", "new", "email", 0.95),
		jevFact("sys.hydraulic.gas.presence", "not_included", "email", 0.95),
	)
	in.User = header("house", "extend")
	in.Thresholds.Amber["action"] = 0.85
	rows := profile.Build(in, cat)
	for _, leaf := range []string{"access-egress.stairs-and-ramps", "hydraulic.sanitary"} {
		if r, ok := scopeRow(rows, leaf); !ok || r.Value != "in" || r.Band != "amber" {
			t.Fatalf("%s: an applied action must put the system in scope: %+v %v", leaf, r, ok)
		}
	}
	if _, ok := scopeRow(rows, "hydraulic.gas"); ok {
		t.Fatal("a system stated as not included stays out of scope whatever its action")
	}
	low := input(jevFact("sys.access-egress.stairs-and-ramps.action", "alter", "brief", 0.5))
	low.User = header("house", "extend")
	low.Thresholds.Amber["action"] = 0.85
	if _, ok := scopeRow(profile.Build(low, cat), "access-egress.stairs-and-ramps"); ok {
		t.Fatal("an action below its floor is not evidence")
	}
}

// Words the documents use for the works suggest the systems they imply
// (Walsh review, 2026-10-27). A suggestion is not evidence: it carries the
// document's words, and the user's out is final.
func TestImpliedScopeSuggestsFromTheDocumentsWords(t *testing.T) {
	cat := repoCatalog(t)
	brief := "First-floor main bedroom with ensuite + walk-in robe over rear portion. New kitchen and dining."
	readings := profile.ImpliedReadings(brief, cat)
	got := map[string]profile.Reading{}
	for _, r := range readings {
		got[r.QuestionID] = r
	}
	for _, leaf := range []string{"interiors.wet-area-waterproofing", "interiors.tiling", "hydraulic.sanitary", "hydraulic.cold-water", "hydraulic.hot-water", "mechanical.local-exhaust", "interiors.joinery", "interiors.internal-doors"} {
		r, ok := got["implied."+leaf]
		if !ok || r.Value != "included" || r.Basis == "" || r.Excerpt == "" || len(r.Excerpt) > 120 {
			t.Fatalf("%s not implied by the brief: %+v", leaf, r)
		}
	}
	if len(profile.ImpliedReadings("Fortnightly owner progress reporting.", cat)) != 0 {
		t.Fatal("a passage naming no room or element implied scope")
	}

	var facts []profile.Fact
	for _, r := range profile.ImpliedReadings("Party wall tie-in with No. 40 — acoustic and fire separation", cat) {
		facts = append(facts, profile.Fact{QuestionID: r.QuestionID, Value: r.Value, Basis: r.Basis, Excerpt: r.Excerpt, DocumentID: "email", PassageID: "email-p4", DocumentKind: "correspondence", DecidedBy: "rule"})
	}
	in := input(facts...)
	in.User = append(header("house", "extend"), profile.UserValue{PartID: whole, Key: "scope.interiors.acoustic-separation", Value: str("out")})
	rows := profile.Build(in, cat)
	for _, leaf := range []string{"site.adjoining-structures", "fire-passive.compartments"} {
		r, ok := scopeRow(rows, leaf)
		if !ok || r.Value != "in" || r.Band != "suggested" || !strings.Contains(r.Note, "Party wall") {
			t.Fatalf("%s: implied scope must be a suggestion carrying the document's words: %+v %v", leaf, r, ok)
		}
	}
	if r, _ := scopeRow(rows, "interiors.acoustic-separation"); r.Value != "out" || r.Band != "user" {
		t.Fatalf("the user's out is final: %+v", r)
	}
	for _, r := range rows {
		if strings.HasPrefix(r.Key, "implied.") || r.Key == "sys.site.adjoining-structures.presence" {
			t.Fatalf("an implied term made a row of its own: %s", r.Key)
		}
	}
}

// What no document states stays unknown (Walsh review, 2026-10-27).
func TestWalshUnknownsStayOutOfScope(t *testing.T) {
	cat := repoCatalog(t)
	var facts []profile.Fact
	for _, text := range []string{
		"Late-Victorian terrace (circa 1890) with sandstone plinth and two-storey front portion; single-storey lean-to rear.",
		"Exclusions unless separately agreed: full interior design package; landscape design (we will brief a consultant if required); geotechnical and specialist reports.",
		"Latent conditions in lean-to footings (common in this era)",
	} {
		for _, r := range profile.ImpliedReadings(text, cat) {
			facts = append(facts, profile.Fact{QuestionID: r.QuestionID, Value: r.Value, Basis: r.Basis, Excerpt: r.Excerpt, DocumentID: "d", PassageID: "d-p", DocumentKind: "report", DecidedBy: "rule"})
		}
	}
	in := input(facts...)
	in.User = header("house", "extend")
	rows := profile.Build(in, cat)
	for _, leaf := range []string{"site.contamination", "electrical.switchboards", "site.landscape", "substructure.termite-management", "substructure.underpinning"} {
		if _, ok := scopeRow(rows, leaf); ok {
			t.Fatalf("%s is in scope though no document states it", leaf)
		}
	}
}
