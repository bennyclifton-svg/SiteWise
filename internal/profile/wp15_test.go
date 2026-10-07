package profile_test

import (
	"fmt"
	"testing"
	"time"

	"sitewise/internal/jev"
	"sitewise/internal/profile"
)

func TestDerivationTrustAndReset(t *testing.T) {
	cat := reviewedCatalog(t)
	for _, verified := range []bool{false, true} {
		in := input()
		review := profile.ReviewAccepted
		if verified {
			review = profile.ReviewVerified
		}
		in.User = []profile.UserValue{
			{PartID: whole, Key: "det.ncc_class", Value: str("7b"), ReviewStatus: review},
			{PartID: whole, Key: "det.rise_in_storeys", Value: str("1"), ReviewStatus: profile.ReviewVerified},
			// An unrelated unverified value must not taint a verified derivation.
			{PartID: whole, Key: "det.bal", Value: str("BAL-LOW")},
		}
		rows := profile.Build(in, cat)
		for key, value := range map[string]string{"det.type_of_construction": "C", "det.max_floor_area_m2": "2000", "det.max_volume_m3": "12000"} {
			r := row(t, rows, whole, key)
			if r.Value != value {
				t.Fatalf("lost derived fact: %+v", r)
			}
			if len(r.Derived.Provenance.Inputs) != 2 {
				t.Fatalf("missing input provenance: %+v", r.Derived)
			}
			for _, input := range r.Derived.Provenance.Inputs {
				if input.Ref == "" || input.Origin == "" || input.ReviewStatus == "" {
					t.Fatalf("incomplete input: %+v", input)
				}
				if input.Ref == whole+"/det.type_of_construction" && (input.Origin != profile.OriginCalculation || input.ReviewStatus != review) {
					t.Fatalf("lost chain provenance: %+v", input)
				}
			}
			if verified {
				if r.Band != "green" || r.ReviewStatus != profile.ReviewVerified || r.Note != "" {
					t.Fatalf("verified inputs: %+v", r)
				}
			} else if r.Band != "amber" || r.ReviewStatus != profile.ReviewAccepted || r.Note != "Planning only — inputs not verified" {
				t.Fatalf("unverified chain: %+v", r)
			}
		}
		// Reset and explicit unknown both remove dependent results, including the chain.
		for _, state := range []string{profile.StateCleared, profile.StateUnknown} {
			in.User[1].State = state
			in.User[1].Value = nil
			for _, r := range profile.Build(in, cat) {
				if r.Derived != nil && (r.Value != "" || r.Band != "blank" || r.ReviewStatus == profile.ReviewVerified) {
					t.Fatalf("stale result after %s: %+v", state, r)
				}
			}
		}
	}
}

func TestBuildingYearHarvestReadAndSiteRouting(t *testing.T) {
	cat := repoCatalog(t)
	for _, year := range []int{1799, 1800, 1985, time.Now().Year(), time.Now().Year() + 1} {
		text := fmt.Sprintf("The existing building was built in %d.", year)
		h := profile.Harvest(text, cat)
		valid := year >= 1800 && year <= time.Now().Year()
		if h.Has("det.existing_building_year") != valid || !h.Has("det.existing_building") {
			t.Fatalf("year %d: %+v", year, h)
		}
		if !valid {
			continue
		}
		qs, cands := profile.LabelQuestions(profile.PassageInfo{Text: text, Ordinal: 30}, h, cat)
		candidate := cands["det.existing_building_year"][0]
		readings := profile.Readings(jev.Result{Answers: map[string]jev.Answer{
			"det.existing_building_year": {Type: jev.TypeChoice, Choice: candidate.ID, Confidence: conf(.9)},
		}}, qs, cands, text)
		if len(readings) != 1 || readings[0].Value != fmt.Sprint(year) || readings[0].Unit != "year" {
			t.Fatalf("readings %+v", readings)
		}
		f := jevFact(readings[0].QuestionID, readings[0].Value, "building-report", .9)
		f.Excerpt, f.FileSHA256, f.Revision, f.Page = readings[0].Excerpt, "hash", "B", 7
		in := input(f, jevFact("det.existing_building", "stated_true", "building-report", .9))
		rows := profile.Build(in, cat)
		r := row(t, rows, whole, "det.existing_building_year")
		if r.Scope != "site" || len(r.Sources) != 1 || r.Sources[0].DocumentID != f.DocumentID || r.Sources[0].FileSHA256 != "hash" || r.Sources[0].Revision != "B" || r.Sources[0].Page != 7 {
			t.Fatalf("site provenance: %+v", r)
		}
		if row(t, rows, whole, "det.existing_building").Scope != "project" {
			t.Fatal("intervention scope changed")
		}
		in.User = []profile.UserValue{{PartID: whole, Key: f.QuestionID, Value: str("1990")}}
		r = row(t, profile.Build(in, cat), whole, f.QuestionID)
		if r.Value != "1990" || r.Band != "user" || len(r.Sources) != 1 {
			t.Fatalf("human decision lost: %+v", r)
		}
	}
}

func TestGreenDocumentConfidenceIsNotVerification(t *testing.T) {
	cat := reviewedCatalog(t)
	in := input(
		jevFact("det.ncc_class.7b", "stated_true", "d1", .99),
		jevFact("det.ncc_class.assertion", "stated", "d1", .99),
	)
	in.Thresholds.Green["determinant"] = conf(.9)
	in.User = []profile.UserValue{{PartID: whole, Key: "det.rise_in_storeys", Value: str("1"), ReviewStatus: profile.ReviewVerified}}
	rows := profile.Build(in, cat)
	if row(t, rows, whole, "det.ncc_class").Band != "green" {
		t.Fatal("fixture must exercise green evidence")
	}
	for _, r := range rows {
		if r.Derived != nil && len(r.Derived.Provenance.Inputs) > 0 && r.Derived.Provenance.Inputs[0].Origin != profile.OriginDocument {
			t.Fatalf("document input origin lost: %+v", r.Derived)
		}
		if r.Derived != nil && r.Value != "" && (r.Band != "amber" || r.ReviewStatus != profile.ReviewAccepted || r.Note != "Planning only — inputs not verified") {
			t.Fatalf("confidence promoted to verification: %+v", r)
		}
	}
}
