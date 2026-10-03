package intake_test

import (
	"sitewise/internal/identity"
	"sitewise/internal/intake"
	"testing"
)

func TestAdjacentNumberCaptionsAreNotSheetTitle(t *testing.T) {
	for _, value := range []string{"C.A.P. No FILE No", "Drawing No. Sheet No."} {
		text := identity.Text{TextLayer: true, Format: "pdf", Runs: []identity.Run{
			{Text: "Drawing No.", Source: identity.Source{Page: 1}},
			{Text: "A-101", Source: identity.Source{Page: 1}},
			{Text: value, Source: identity.Source{Page: 1}},
		}}
		for _, c := range intake.Harvest("", text) {
			if c.Field == intake.FieldTitle && c.Display == value {
				t.Errorf("caption offered as sheet title: %q", value)
			}
		}
	}
}
