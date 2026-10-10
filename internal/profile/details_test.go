package profile_test

import (
	"fmt"
	"strings"
	"testing"

	"sitewise/internal/jev"
	"sitewise/internal/profile"
)

// Phrases are short quotes from the Petersham principal's project
// requirements (PPR) and similar briefs: the address, client and LGA the
// owner expected the profile to fill.
func TestHarvestProjectDetailCandidates(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		text  string
		key   string
		norms []string // each must be among the candidate values
		not   []string // none may be a candidate value
	}{
		{"PRINCIPAL’S PROJECT REQUIREMENTS JW building: 568-572 Parramatta Road Petersham Principal: Joins Win Pty Ltd ABN 20 098 348 031 DRAFT Revision - E",
			"site.address", []string{"568-572 Parramatta Road Petersham"}, []string{"568-572 Parramatta Road Petersham Principal"}},
		{"568-572 Parramatta Rd Petersham – Principal Project Requirements (Rev E– 21.06.2021)",
			"site.address", []string{"568-572 Parramatta Road Petersham"}, nil},
		{"It has two approved street addresses 572 Paramatta Road (commercial) & 82 Queen Street Petersham residential).",
			"site.address", []string{"572 Paramatta Road", "82 Queen Street Petersham"}, nil},
		{"Principal: Joins Win Pty Ltd ABN 20 098 348 031 DRAFT Revision - E",
			"proj.client", []string{"Joins Win Pty Ltd"}, []string{"Joins Win Pty Ltd ABN"}},
		{"Client: Bankstown City Council.\nArchitect: Foo Architects", "proj.client", []string{"Bankstown City Council"}, nil},
		{"Prepared for Acme Developments Pty Ltd by Smith Planning", "proj.client", []string{"Acme Developments Pty Ltd"}, nil},
		{"The site is located within the Inner West local government area.", "proj.lga", []string{"Inner West"}, nil},
		{"The Contractor must pay all authority fees including Inner West Council NBN Co", "proj.lga", []string{"Inner West"}, nil},
		{"The site is in the City of Parramatta.", "proj.lga", []string{"City of Parramatta"}, nil},
		{"The land is described as Lot 12 in DP 123456 and Lots 1 and 2 DP 5432.", "site.lot", []string{"Lot 12 DP 123456", "Lots 1 and 2 DP 5432"}, nil},
		{"Lot 7 Section 3 DP1234 Smith Street", "site.lot", []string{"Lot 7 Section 3 DP1234"}, nil},
		{"The site is zoned B4 Mixed Use under the Inner West LEP 2022.", "proj.zone", []string{"B4 Mixed Use"}, nil},
		{"The building must achieve a 5 Star Green Star Design & As Built rating.", "proj.sustainability", []string{"5 Star Green Star Design & As Built rating"}, nil},
	}
	for _, tc := range cases {
		h := profile.Harvest(tc.text, cat)
		if !h.Has(tc.key) {
			t.Errorf("%q: %s not triggered (got %v)", tc.text, tc.key, h.Triggered)
			continue
		}
		var got []string
		for _, c := range h.Candidates[tc.key] {
			got = append(got, c.Norm)
		}
		for _, want := range tc.norms {
			if !contains(got, want) {
				t.Errorf("%q: %s missing %q; got %q", tc.text, tc.key, want, got)
			}
		}
		for _, bad := range tc.not {
			if contains(got, bad) {
				t.Errorf("%q: %s offered %q", tc.text, tc.key, bad)
			}
		}
	}
}

func TestHarvestProjectDetailsIgnoreLookalikes(t *testing.T) {
	cat := repoCatalog(t)
	cases := map[string]string{
		"PRINCIPAL’S PROJECT REQUIREMENTS":                         "proj.client",
		"requests from another Client of the Contractor":           "proj.client",
		"1 Bedroom: 4 off 2 Bedroom: 5 off":                        "site.address",
		"Provide 12 Car Spaces on Level 1":                         "site.address",
		"The Contractor must hold Lot numbers for each lot":        "site.lot",
		"Allow for the shower zone and the safe construction zone": "proj.zone",
	}
	for text, key := range cases {
		if profile.Harvest(text, cat).Has(key) {
			t.Errorf("%q: %s should not be offered", text, key)
		}
	}
}

func TestLabelQuestionsOfferDetailCandidatesAndPickRecordsThem(t *testing.T) {
	cat := repoCatalog(t)
	text := "Principal: Joins Win Pty Ltd ABN 20 098 348 031 at 568-572 Parramatta Road Petersham"
	qs, cands := profile.LabelQuestions(profile.PassageInfo{Ordinal: 1, Text: text}, profile.Harvest(text, cat), cat)
	for _, key := range []string{"proj.client", "site.address"} {
		q, ok := qs[key]
		if !ok {
			t.Fatalf("no question %s (got %d questions)", key, len(qs))
		}
		if text := fmt.Sprint(q.Instructions); !strings.Contains(text, "candidates") {
			t.Errorf("%s instructions do not mention candidates: %s", key, text)
		}
		if len(cands[key]) == 0 {
			t.Errorf("%s: no candidates in state", key)
		}
	}
}

func TestProjectDetailReadingsAreProfileQuestions(t *testing.T) {
	for _, id := range []string{"proj.client", "proj.flag.coastal", "site.address", "site.lot"} {
		if !profile.IsProfileQuestion(id) {
			t.Errorf("%s is not a profile question, so its answer would be dropped", id)
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// The Petersham cover page, end to end: code offers candidates, Jev picks,
// the pick is stored verbatim, and Reconcile shows it once the detail shape
// has a confidence floor.
func TestPetershamCoverFillsClientAndAddress(t *testing.T) {
	cat := repoCatalog(t)
	text := "PRINCIPAL’S PROJECT REQUIREMENTS JW building: 568-572 Parramatta Road Petersham Principal: Joins Win Pty Ltd ABN 20 098 348 031 DRAFT Revision - E"
	qs, cands := profile.LabelQuestions(profile.PassageInfo{Ordinal: 1, Text: text}, profile.Harvest(text, cat), cat)
	answers := map[string]jev.Answer{}
	for _, key := range []string{"proj.client", "site.address"} {
		answers[key] = jev.Answer{Type: jev.TypeChoice, Choice: cands[key][0].ID, Confidence: conf(.9)}
	}
	var facts []profile.Fact
	for _, r := range profile.Readings(jev.Result{Answers: answers}, qs, cands, text) {
		facts = append(facts, jevFact(r.QuestionID, r.Value, "ppr", .9))
	}
	in := input(facts...)
	in.Thresholds.Amber["detail"] = .6
	rows := profile.Reconcile(in, cat)
	if r := row(t, rows, whole, "proj.client"); r.Value != "Joins Win Pty Ltd" || r.Band != "amber" || r.Scope != "project" {
		t.Errorf("client = %+v", r)
	}
	if r := row(t, rows, whole, "site.address"); r.Value != "568-572 Parramatta Road Petersham" || r.Band != "amber" || r.Scope != "site" {
		t.Errorf("address = %+v", r)
	}
	// With no floor for the shape the reading is kept but not applied.
	in = input(facts...)
	if r := row(t, profile.Reconcile(in, cat), whole, "proj.client"); r.Value != "" || r.Band != "blank" {
		t.Errorf("uncalibrated client = %+v", r)
	}
}

func TestPickingNoCandidateStoresNothing(t *testing.T) {
	cat := repoCatalog(t)
	text := "Client: Bankstown City Council."
	qs, cands := profile.LabelQuestions(profile.PassageInfo{Ordinal: 1, Text: text}, profile.Harvest(text, cat), cat)
	res := jev.Result{Answers: map[string]jev.Answer{"proj.client": {Type: jev.TypeChoice, Choice: "none", Confidence: conf(.9)}}}
	if got := profile.Readings(res, qs, cands, text); len(got) != 0 {
		t.Errorf("none stored %+v", got)
	}
}

func TestFlagsAreAskedYesNoAndStoredAsTheUserWouldEnterThem(t *testing.T) {
	cat := repoCatalog(t)
	cases := map[string]string{
		"proj.flag.staged":       "The works are delivered in two stages: stage 1 is the carpark.",
		"proj.flag.basement":     "Excavate the basement to RL 12.5.",
		"proj.flag.cl46":         "A clause 4.6 variation to the height standard was approved.",
		"proj.flag.rail":         "The site adjoins the Sydney Trains rail corridor.",
		"proj.flag.osd_required": "On-site detention is required by council.",
	}
	for key, text := range cases {
		h := profile.Harvest(text, cat)
		if !h.Has(key) {
			t.Errorf("%s not triggered by %q (got %v)", key, text, h.Triggered)
			continue
		}
		qs, _ := profile.LabelQuestions(profile.PassageInfo{Ordinal: 1, Text: text}, h, cat)
		q, ok := qs[key]
		if !ok {
			t.Errorf("no question for %s", key)
			continue
		}
		crit, _ := q.Criteria.(map[string]string)
		if crit["yes"] == "" || crit["no"] == "" || crit["not_stated"] == "" {
			t.Errorf("%s criteria = %v", key, crit)
		}
		got := profile.Readings(jev.Result{Answers: map[string]jev.Answer{key: {Type: jev.TypeChoice, Choice: "yes", Confidence: conf(.9)}}}, qs, nil, text)
		if len(got) != 1 || got[0].QuestionID != key || got[0].Value != "yes" {
			t.Errorf("%s readings = %+v", key, got)
		}
	}
	if profile.Harvest("Allow for a staircase between levels.", cat).Has("proj.flag.staged") {
		t.Error("a staircase is not a stage")
	}
	// The judgements no document states are never asked.
	for _, key := range []string{"objections", "super_flat_floor", "contract_admin"} {
		if _, ok := cat.FlagReading(key); ok {
			t.Errorf("%s must stay a user judgement", key)
		}
	}
}

func TestEveryReadFlagIsARealDetailFlagOfTheSkeleton(t *testing.T) {
	cat := repoCatalog(t)
	sk, ok := cat.RFPSkeleton()
	if !ok {
		t.Fatal("no RFP skeleton")
	}
	byKey := map[string]bool{}
	for _, f := range sk.Flags {
		byKey[f.Key] = true
	}
	for _, f := range cat.FlagReadings() {
		if !byKey[f.Key] {
			t.Errorf("flag reading %s is not a skeleton flag", f.Key)
		}
		if _, det := map[string]bool{"bushfire": true, "flood": true, "contamination": true, "acid_sulfate": true, "existing_building": true,
			"heritage": true, "live_operations": true, "mixed_use": true}[f.Key]; det {
			t.Errorf("flag %s already has a determinant home; reading it twice would disagree", f.Key)
		}
	}
}

// A passage inherits its section heading and context; a detail must come from
// the passage's own text. Petersham's list of the contractor's past projects
// ("81 MacDonald Street Erskineville") rode along into 195 later passages and
// out-voted the real address.
func TestDetailsAreFoundInThePassagesOwnTextOnly(t *testing.T) {
	cat := repoCatalog(t)
	inherited := "Reference projects\n19-21 Eve Street Erskineville; 81 MacDonald Street Erskineville\n"
	own := "All bedroom areas to be carpet unless noted otherwise."
	if h := profile.HarvestOwn(inherited+own, own, cat); h.Has("site.address") {
		t.Errorf("an inherited address was offered: %+v", h.Candidates["site.address"])
	}
	own = "The works are at 568-572 Parramatta Road Petersham."
	h := profile.HarvestOwn(inherited+own, own, cat)
	if !h.Has("site.address") || len(h.Candidates["site.address"]) != 1 {
		t.Errorf("own address missing or mixed with inherited ones: %+v", h.Candidates["site.address"])
	}
	// Everything else still reads the whole excerpt.
	if !profile.HarvestOwn("Design life 50 years", "50 years", cat).Has("fact.design_life") {
		t.Error("non-detail questions must still read the excerpt")
	}
}

func TestAddressStreetTypesAreWrittenOutSoSpellingsAgree(t *testing.T) {
	cat := repoCatalog(t)
	for _, text := range []string{"568-572 Parramatta Rd Petersham", "568-572 Parramatta Road Petersham"} {
		cands := profile.Harvest(text, cat).Candidates["site.address"]
		if len(cands) != 1 || cands[0].Norm != "568-572 Parramatta Road Petersham" {
			t.Errorf("%q = %+v", text, cands)
		}
		if cands[0].Value != text {
			t.Errorf("the verbatim text must be kept: %q", cands[0].Value)
		}
	}
}

func TestAddressDoesNotTakeTheNextSentenceAsASuburb(t *testing.T) {
	cat := repoCatalog(t)
	for _, word := range []string{"Additional", "Proposed", "Refer", "Principal", "Residential", "Existing"} {
		cands := profile.Harvest("81 MacDonald Street "+word+" works", cat).Candidates["site.address"]
		if len(cands) != 1 || cands[0].Norm != "81 MacDonald Street" {
			t.Errorf("%s: %+v", word, cands)
		}
	}
}
