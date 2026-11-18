package httpapi

import "testing"

func TestPointsAtSelection(t *testing.T) {
	for text, want := range map[string]bool{"Exclude these": true, "delete those.": true, "move them to Fitout": true, "exclude the selected lines": true,
		"Exclude the theme lines": false, "Add a line for thesis": false, "Show the cost plan": false} {
		if got := pointsAtSelection(text); got != want {
			t.Errorf("%q: %v", text, got)
		}
	}
}
