package profile_test

import (
	"slices"
	"testing"

	"sitewise/internal/profile"
)

// The details a fit-out brief, planning advice, engagement letter and
// landlord email state, offered as candidates for Jev to pick from
// (docs/plans/2026-11-11-accommodation-and-full-capture.md).
func TestHarvestFitoutDetails(t *testing.T) {
	cat := repoCatalog(t)
	cases := []struct {
		text string
		key  string
		want string // a candidate norm that must be offered
		not  string // a candidate norm that must not be
	}{
		{"- Building: **Commercial office tower**, BCA Class 5.\n4. **No heritage constraints** — building constructed 1984, not heritage-listed.",
			"det.existing_building_year", "1984", ""},
		{"1. **Occupancy load increase** from 98 persons (existing) to **142 persons** proposed — egress and sanitary provision to be demonstrated.",
			"det.occupant_load", "142", ""},
		{"2. **Mezzanine insert** on Level 4 (~185 m²) triggers structural and fire-compartment review.",
			"hdr.scale.added_gfa_sqm", "185", ""},
		{"**Project:** Meridian Legal Group fit-out, Levels 3 & partial Level 4, 88 Westgate Street, Parramatta NSW 2150",
			"proj.works_levels", "Level 3 and part of Level 4", "Level 3 and part of Level 4, 88"},
		{"Please find below our draft base building information schedule for Levels 3 & 4.",
			"proj.works_levels", "Levels 3 and 4", ""},
		{"- Target possession for fit-out: **1 November 2026** (subject to SSD approval).",
			"proj.target_construction_start", "1 November 2026", ""},
		{"- SSD lodgement target: **September 2026** (subject to fire engineering and architectural package).",
			"proj.target_da_lodgement", "September 2026", ""},
		{"- Rent-free fit-out period requested: **1 February 2027 to 30 June 2027**.\n- Firm occupation from **1 July 2027**.",
			"proj.target_completion", "1 July 2027", ""},
		{"**Programme ROM:** 22–26 weeks on site after possession and CC issue, if **after-hours works** available 5 nights per week.",
			"proj.preliminary_construction_months", "22–26 weeks", ""},
		{"## Budget aspiration (not confirmed)\nInternal feasibility allowance **$1.35M – $1.55M** for tenant works excluding furniture.",
			"proj.budget_aspiration", "$1.35M – $1.55M", ""},
		{"| HVAC plant upgrade | Landlord will upgrade central plant capacity on Level 3 riser — allowance **$180,000** contributed (ex GST) subject to lease incentive deed |",
			"proj.contributions_stated", "HVAC plant upgrade: Landlord will upgrade central plant capacity on Level 3 riser — allowance $180,000 contributed (ex GST) subject to lease incentive deed", ""},
		{"4. Landlord design review and base-building interface coordination (Westgate Properties / CBRE)",
			"proj.landlord", "Westgate Properties", ""},
		{"| After-hours works | Permitted **10:00 pm – 6:00 am** weekdays and Saturdays until 2:00 pm; **no Sunday works** |",
			"proj.work_hours", "After-hours works: Permitted 10:00 pm – 6:00 am weekdays and Saturdays until 2:00 pm; no Sunday works", ""},
		{"Subject: Meridian Legal — base building schedule\nDate: 8 March 2026 3:15 PM\nWe require after-hours access where necessary.",
			"proj.work_hours", "", "Date: 8 March 2026 3:15 PM"},
		{"Tenant works must not commence until SSD consent and landlord fit-out consent deed are executed.",
			"proj.start_conditions", "Tenant works must not commence until SSD consent and landlord fit-out consent deed are executed.", ""},
		{"5. Head builder procurement — preparation of tender documentation, evaluation of **two** invited builders, and written recommendation",
			"proj.tenderers", "2", ""},
		{"Recommend keeping Apex on the invite list with one other commercial fit-out builder experienced in occupied buildings.",
			"proj.tenderers", "", "1"},
	}
	for _, c := range cases {
		h := profile.Harvest(c.text, cat)
		got := spaceNorms(h.Candidates[c.key])
		if c.want != "" && !slices.Contains(got, c.want) {
			t.Errorf("%s: candidates %q, want %q (asked %v)", c.key, got, c.want, h.Has(c.key))
		}
		if c.not != "" && slices.Contains(got, c.not) {
			t.Errorf("%s: offered %q", c.key, c.not)
		}
	}
}

// Floors said more or less precisely are one answer; amounts written two
// ways are one contribution. Different floors or sums stay a conflict.
func TestReconcileFloorsAndAmounts(t *testing.T) {
	cat := repoCatalog(t)
	conf := 1.0
	fact := func(key, value, doc string) profile.Fact {
		return profile.Fact{QuestionID: key, Value: value, DocumentID: doc, PassageID: doc + value, Confidence: &conf, DecidedBy: "jev"}
	}
	th := profile.Thresholds{Amber: map[string]float64{"detail": 0.6}}
	row := func(facts ...profile.Fact) profile.Row {
		rows := profile.Reconcile(profile.Input{Parts: []profile.Part{{ID: "w", Kind: "whole"}}, Facts: facts, Thresholds: th}, cat)
		for _, r := range rows {
			if r.Key == facts[0].QuestionID {
				return r
			}
		}
		t.Fatalf("no row for %s in %d rows", facts[0].QuestionID, len(rows))
		return profile.Row{}
	}
	r := row(
		fact("proj.works_levels", "Level 3 and part of Level 4", "d1"), fact("proj.works_levels", "Level 3 and part of Level 4", "d2"),
		fact("proj.works_levels", "Levels 3 and 4", "d3"), fact("proj.works_levels", "Levels 3 and 4", "d4"),
		fact("proj.works_levels", "Level 4", "d5"), fact("proj.works_levels", "Levels 2 and 5", "d6"),
	)
	if r.Value != "Level 3 and part of Level 4" || r.Band != "amber" || len(r.Alternatives) != 1 || r.Alternatives[0].Value != "Levels 2 and 5" {
		t.Fatalf("floors = %q %s %+v", r.Value, r.Band, r.Alternatives)
	}
	r = row(
		fact("proj.contributions_stated", "HVAC plant upgrade: allowance $180,000 contributed (ex GST) subject to lease incentive deed", "d1"),
		fact("proj.contributions_stated", "James's schedule confirms landlord HVAC contribution ($180k) but leaves supplementary sprinklers, after-hours rules, and switchboard extension as tenant/ shared-cost items.", "d1"),
	)
	if r.Band != "amber" || r.Value != "HVAC plant upgrade: allowance $180,000 contributed (ex GST) subject to lease incentive deed" {
		t.Fatalf("contribution = %q %s", r.Value, r.Band)
	}
	r = row(
		fact("proj.contributions_stated", "landlord contribution $180,000", "d1"),
		fact("proj.contributions_stated", "landlord contribution $150,000", "d2"),
	)
	if r.Band != "red" {
		t.Fatalf("different sums must conflict, got %q %s", r.Value, r.Band)
	}
}
