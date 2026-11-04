package proposals

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sitewise/internal/identity"
)

// Fee proposals from the synthetic test corpus (testing/synthetic, scenarios
// ST01, ST05 and ST09; expected values from each scenario's fees.json). They
// exposed a table total row with an empty first cell left unmarked (F14a), a
// total stated only inc GST dropped (F14b), an inc-GST total kept as if ex
// (F3) and a letterhead refused because of the header's "Client:" line.
var synthetic = []struct {
	file, firm, abn, gst string
	total                int64
	gstInc               bool
}{
	{"Fee proposal - Parapet Building Consultants - Kurrajong Apartments.md", "Parapet Building Consultants Pty Ltd", "66049460444", "ex", 8100000, false},
	{"Halyard fee submission - Kurrajong Parade.md", "HALYARD REMEDIAL CONSULTANTS PTY LTD", "41434467555", "inc", 9000000, true},
	{"Structural engineering fee proposal KCE-Q-2611.md", "Karrak Consulting Engineers Pty Ltd", "77650369205", "ex", 1700000, false},
	{"Hazardous materials survey fee proposal FHM-P-0309.md", "Fibrecheck Hazardous Materials Pty Ltd", "23824966838", "ex", 700000, false},
	{"Lindqvist Civil - stormwater fee - Tallowwood Ave.md", "LINDQVIST CIVIL & STORMWATER PTY LTD", "20748603875", "inc", 660000, true},
	{"Brennan Civil - stormwater fee - Corella Rise.md", "BRENNAN CIVIL ENGINEERING PTY LTD", "31864452323", "inc", 900000, true},
}

func syntheticText(t *testing.T, file string) identity.Text {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", "synthetic", file))
	if err != nil {
		t.Fatal(err)
	}
	text, err := identity.Extract(context.Background(), "md", bytes.NewReader(body), int64(len(body)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func mdText(t *testing.T, s string) identity.Text {
	t.Helper()
	text, err := identity.Extract(context.Background(), "md", strings.NewReader(s), int64(len(s)), ExtractLimits())
	if err != nil {
		t.Fatal(err)
	}
	return text
}

func TestSyntheticFirmGSTAndTotalSettleByRule(t *testing.T) {
	for _, s := range synthetic {
		t.Run(s.file, func(t *testing.T) {
			d := NewDraft(Harvest(s.file, syntheticText(t, s.file)), Context{}, nil)
			for field, value := range map[string]string{FieldFirmName: s.firm, FieldFirmABN: s.abn, FieldGSTBasis: s.gst} {
				got, ok := d.Decided(field)
				if !ok || got.Value != value || got.DecidedBy != ByRule {
					t.Errorf("%s = %+v, want %q by rule", field, got, value)
				}
			}
			if got, ok := d.Decided(FieldStatedTotal); !ok || got.DecidedBy != ByRule {
				t.Errorf("stated total not settled by rule: %+v", got)
			}
			r := d.Finish()
			if r.StatedTotal == nil || *r.StatedTotal != s.total {
				t.Errorf("stated total %v, want %d ex GST", r.StatedTotal, s.total)
			}
			if hasFlag(r.Flags, FlagGSTInc) != s.gstInc {
				t.Errorf("flags %v, want %s=%v", r.Flags, FlagGSTInc, s.gstInc)
			}
		})
	}
}

// A total row whose words sit in the second column under an empty first
// cell is the stated total (F14a).
func TestTotalWordsInSecondColumnMarkTotal(t *testing.T) {
	ev := Harvest("fee.md", mdText(t, "| Item | Description | Fee (ex GST) |\n|---|---|---|\n| 1 | Survey | $3,900 |\n| 2 | Report | $3,100 |\n| | Total $7,000 excluding GST | $7,000 |\n"))
	totals := 0
	for _, a := range ev.Amounts {
		if a.Cents == 700000 && !a.Total {
			t.Errorf("%s %q not marked total", a.ID, a.Context)
		}
		if a.Total {
			totals++
		}
	}
	if totals == 0 {
		t.Fatal("no total")
	}
}

// An inc-GST stated total is stored ex GST, computed in Go to the cent.
func TestIncGSTStatedTotalStoredExGST(t *testing.T) {
	ev := Harvest("fee.md", mdText(t, "Design $5,000.00\nCertification $5,001.00\nTotal $10,001 including GST\n\nAll amounts include GST.\n"))
	d := NewDraft(ev, Context{}, nil)
	r := d.Finish()
	if r.GSTBasis != "inc" || r.StatedTotal == nil || *r.StatedTotal != 909182 {
		t.Fatalf("basis %q total %v, want inc and 909182", r.GSTBasis, r.StatedTotal)
	}
}
