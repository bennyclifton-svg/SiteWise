package eval

import (
	"testing"

	"sitewise/internal/intake"
)

// The eval expands exactly the cases the application would split into sheets
// (store.CommitFiling): multi-page PDFs filed as a drawing in an applied band.
func TestExpandsMatchesApplication(t *testing.T) {
	drawing := func(band string) []intake.Decision {
		return []intake.Decision{{Field: intake.FieldKind, Value: "drawing", Band: band}}
	}
	for _, c := range []struct {
		name      string
		format    string
		pages     int
		decisions []intake.Decision
		want      bool
	}{
		{"multi-page amber drawing", "pdf", 3, drawing("amber"), true},
		{"multi-page green drawing", "pdf", 3, drawing("green"), true},
		{"single page", "pdf", 1, drawing("green"), false},
		{"grey kind is not confirmed", "pdf", 3, drawing("grey"), false},
		{"a report is not split", "pdf", 3, []intake.Decision{{Field: intake.FieldKind, Value: "report", Band: "green"}}, false},
		{"not a pdf", "docx", 3, drawing("green"), false},
	} {
		if got := expands(Case{Format: c.format}, c.pages, c.decisions); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
}
