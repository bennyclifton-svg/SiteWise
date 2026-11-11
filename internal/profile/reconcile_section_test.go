package profile_test

import (
	"testing"

	"sitewise/internal/profile"
)

// A builder's quote: the demolition section says its subcontractor does no
// earthworks, and the price table carries an Earthworks category. The
// exclusion limits the demolition trade; it does not contradict the price.
func TestTradeSectionExclusionDoesNotContradictThePricedRow(t *testing.T) {
	demo := tender(jevFact("sys.site.earthworks.presence", "not_included", "coastal", 1))
	demo.PassageID, demo.Section, demo.Excerpt = "coastal-demo", "Demolition", "No site earthworks/excavation are included."
	priced := tender(jevFact("sys.site.earthworks.presence", "included", "coastal", 0.68))
	priced.PassageID, priced.Section, priced.Excerpt = "coastal-cat4", "Quoted Categories", "4 Earthworks - Site Cut, Footings, Trenching $33,356.50"
	r := row(t, profile.Reconcile(input(demo, priced), repoCatalog(t)), whole, "sys.site.earthworks.presence")
	if r.Value != "included" || r.Band != "amber" || r.Note != priced.Excerpt {
		t.Fatalf("%+v", r)
	}
	// The exclusion stays listed, so the reader sees it and can choose it.
	if len(r.Alternatives) != 1 || r.Alternatives[0].Value != "not_included" || r.Alternatives[0].Sources[0].Excerpt != demo.Excerpt {
		t.Fatalf("alternatives %+v", r.Alternatives)
	}
	if r.Tenders != "" {
		t.Fatalf("one tender cannot differ from itself: %q", r.Tenders)
	}
}

// An exclusion in the document's own list of exclusions is the project's,
// so the conflict with a priced row stands.
func TestGeneralExclusionStillConflicts(t *testing.T) {
	excluded := tender(jevFact("sys.site.landscape.presence", "not_included", "coastal", 0.84))
	excluded.PassageID, excluded.Section = "coastal-excl", "Excluded Items"
	demo := tender(jevFact("sys.site.landscape.presence", "not_included", "coastal", 0.8))
	demo.PassageID, demo.Section = "coastal-demo", "Demolition"
	priced := tender(jevFact("sys.site.landscape.presence", "included", "coastal", 0.7))
	priced.PassageID, priced.Section = "coastal-cat34", "Quoted Categories"
	r := row(t, profile.Reconcile(input(excluded, demo, priced), repoCatalog(t)), whole, "sys.site.landscape.presence")
	if r.Band != "red" || r.Value != "" {
		t.Fatalf("%+v", r)
	}
}

// The exclusion of a trade's own work, under that trade's heading, is not
// local to another trade.
func TestExclusionUnderItsOwnHeadingStillConflicts(t *testing.T) {
	own := jevFact("sys.site.demolition.presence", "not_included", "d1", 0.9)
	own.PassageID, own.Section = "d1-a", "Demolition"
	other := jevFact("sys.site.demolition.presence", "included", "d1", 0.9)
	other.PassageID, other.Section = "d1-b", "Quoted Categories"
	r := row(t, profile.Reconcile(input(own, other), repoCatalog(t)), whole, "sys.site.demolition.presence")
	if r.Band != "red" {
		t.Fatalf("%+v", r)
	}
}

// The note and the first source are the strongest reading, not the first
// passage in the document.
func TestPresenceNoteIsTheStrongestSource(t *testing.T) {
	weak := jevFact("sys.hydraulic.hot-water.presence", "included", "d1", 0.8)
	weak.PassageID, weak.Excerpt = "d1-a", "Vegetable Garden: 1 x double w/proof power point."
	below := jevFact("sys.hydraulic.hot-water.presence", "included", "d1", 0.3)
	below.PassageID, below.Excerpt = "d1-0", "Quoted Categories"
	strong := jevFact("sys.hydraulic.hot-water.presence", "included", "d1", 1)
	strong.PassageID, strong.Excerpt = "d1-b", "Hot & Cold water supply to all required fixtures."
	r := row(t, profile.Reconcile(input(below, weak, strong), repoCatalog(t)), whole, "sys.hydraulic.hot-water.presence")
	if r.Note != strong.Excerpt || r.Sources[0].Excerpt != strong.Excerpt || r.Sources[2].Excerpt != below.Excerpt {
		t.Fatalf("note %q sources %+v", r.Note, r.Sources)
	}
}
