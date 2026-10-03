package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestSheetVersionGrid(t *testing.T) {
	run := func(v string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: v, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	text := identity.Text{Format: "pdf", TextLayer: true, Runs: []identity.Run{
		run("SHEET:", 995, 110, 30, 14), run("SCALE:", 1079, 110, 30, 14), run("DRAWN BY:", 995, 77, 46, 14), run("DATE:", 1079, 77, 24, 14), run("VERSION No.", 995, 45, 52, 14), run("CONSTRUCTION No.", 1079, 45, 81, 14),
		run("7 OF 20", 1019, 95, 31, 14), run("1:100", 1108, 95, 21, 14), run("AB", 1028, 62, 14, 14), run("07.03.24", 1102, 62, 33, 14), run("3", 1032, 30, 5, 14), run("-", 1117, 30, 3, 14),
		run("JOB ADDRESS:", 659, 77, 74, 17), run("COUNCIL:", 836, 81, 39, 14), run("ROOF PLAN", 710, 102, 68, 18), run("SUBMISSION PLANS", 474, 101, 121, 21), run("LOT 9 DP 271363", 659, 62, 71, 14),
		run("Version: 1, Version Date: 06/12/2022", 30, 150, 180, 10), run("This plan/document forms part of the Approved Construction Certificate X999 Issued By Example", 1000, 700, 160, 20),
	}}
	candidates := intake.Harvest("", text)
	for _, c := range candidates {
		if c.Field == intake.FieldTitle && c.Display == text.Runs[len(text.Runs)-1].Text {
			t.Error("approval stamp offered as title")
		}
	}
	wants := map[string]string{"number": "7", "revision": "3", "title": "ROOF PLAN", "date": "07.03.24"}
	for _, r := range intake.Decide(candidates) {
		if w, ok := wants[r.Field]; ok && (!r.Settled || r.Display != w) {
			t.Errorf("%s got %+v want %s", r.Field, r, w)
		}
	}
	// The bare sheet counter of a report must not become a drawing number.
	plain := intake.Harvest("", identity.Text{Format: "pdf", TextLayer: true, Runs: text.Runs[:2]})
	for _, c := range plain {
		if c.Provenance.OwnNumber {
			t.Error("incomplete grid admitted")
		}
	}
	for _, index := range []int{0, 1, 2, 3, 4, 5} {
		broken := text
		broken.Runs = append([]identity.Run(nil), text.Runs...)
		broken.Runs[index].Source.Page = 2
		for _, c := range intake.Harvest("", broken) {
			if c.Provenance.BlockTitle || c.Provenance.OwnNumber {
				t.Errorf("grid admitted with anchor %d on another page", index)
			}
		}
	}
}

func TestApprovalStampIsNotDocumentTitle(t *testing.T) {
	text := identity.Text{Format: "pdf", TextLayer: true, Runs: []identity.Run{
		{Text: "This plan/document forms part of the", Source: identity.Source{Page: 1, X: 100, Y: 500, Width: 200, Height: 14}},
		{Text: "Approved Construction Certificate", Source: identity.Source{Page: 1, X: 100, Y: 482, Width: 190, Height: 14}},
		{Text: "X2024-00001", Source: identity.Source{Page: 1, X: 100, Y: 464, Width: 100, Height: 14}},
		{Text: "Issued By Example Certifier", Source: identity.Source{Page: 1, X: 100, Y: 446, Width: 180, Height: 14}},
	}}
	for i := 0; i < 8; i++ {
		text.Runs = append(text.Runs, identity.Run{Text: "body annotation", Source: identity.Source{Page: 1, X: 30, Y: float64(20 + i*10), Width: 70, Height: 7}})
	}
	for _, c := range intake.Harvest("", text) {
		if c.Field == intake.FieldTitle {
			t.Fatalf("stamp offered as title: %+v", c)
		}
	}
}
