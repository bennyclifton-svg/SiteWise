package intake_test

import (
	"testing"

	"sitewise/internal/identity"
	"sitewise/internal/intake"
)

func TestUncaptionedSheetCellRequiresControlColumn(t *testing.T) {
	run := func(s string, x, y, w, h float64) identity.Run {
		return identity.Run{Text: s, Source: identity.Source{Page: 1, X: x, Y: y, Width: w, Height: h}}
	}
	base := []identity.Run{
		run("Project Number", 952, 72, 45, 9), run("Print Date", 952, 48, 29, 9),
		run("Drawn By", 952, 24, 28, 9), run("Scale", 1094, 24, 16, 9),
		run("CC02.6", 1120, 55, 48, 18), run("Typical Floor Plan & Roof Plan", 960, 99, 200, 20),
	}
	for _, tc := range []struct {
		name   string
		remove int
		want   bool
	}{
		{"complete", -1, true}, {"no project", 0, false}, {"no print date", 1, false},
		{"no drawn by", 2, false}, {"no scale", 3, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var runs []identity.Run
			for i, r := range base {
				if i != tc.remove {
					runs = append(runs, r)
				}
			}
			found := map[string]bool{}
			for _, c := range intake.Harvest("", identity.Text{Format: "pdf", TextLayer: true, Runs: runs}) {
				if c.Display == "CC02.6" || c.Display == "Typical Floor Plan & Roof Plan" {
					found[c.Field] = true
					if c.Provenance.OwnNumber || c.Provenance.OwnTitle || c.Provenance.Labeled {
						t.Error("unlabelled cell given explicit-caption provenance")
					}
				}
			}
			for _, field := range []string{intake.FieldNumber, intake.FieldTitle} {
				if found[field] != tc.want {
					t.Errorf("%s candidate=%v want %v", field, found[field], tc.want)
				}
			}
		})
	}
	for _, tc := range []struct {
		name   string
		change func([]identity.Run)
	}{
		{"another page", func(r []identity.Run) { r[4].Source.Page = 2 }},
		{"another rotation", func(r []identity.Run) { r[4].Source.Rotation = 90 }},
		{"body annotation", func(r []identity.Run) { r[4].Source.Y = 500 }},
		{"small reference", func(r []identity.Run) { r[4].Source.Height = 9 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runs := append([]identity.Run(nil), base...)
			tc.change(runs)
			for _, c := range intake.Harvest("", identity.Text{Format: "pdf", TextLayer: true, Runs: runs}) {
				if c.Field == intake.FieldNumber && c.Display == "CC02.6" {
					t.Error("unrelated reference treated as sheet cell")
				}
			}
		})
	}
}
