package intake

import (
	"encoding/json"
	"os"
	"sitewise/internal/identity"
	"testing"
)

func TestOCRTrialRecordedCells(t *testing.T) {
	var fixtures map[string]identity.Text
	b, err := os.ReadFile("testdata/ocr-titleblocks.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &fixtures); err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{
		"01": {"CC-01", "D", "SITE SETOUT PLAN", "09/07/2015"},
		"02": {"CC-02", "F", "BASEMENT 2", "09/07/2015"},
		"03": {"CC-03", "F", "BASEMENT 1", "09/07/2015"},
		"04": {"CC-04", "J", "GROUND FLOOR PLAN", "14/07/2015"},
		"05": {"CC-05", "H", "LEVEL 1", "07/09/2015"},
	}
	for key, want := range expected {
		t.Run(key, func(t *testing.T) {
			text, ok := fixtures[key]
			if !ok {
				t.Fatal("missing fixture")
			}
			for i, got := range Decide(ocrTrialCandidates(text)) {
				value := got.Display
				if i == 0 {
					value = got.Normalized
				}
				if !got.Settled || value != want[i] {
					t.Errorf("%s got %+v want %q", got.Field, got, want[i])
				}
			}
		})
	}
}
