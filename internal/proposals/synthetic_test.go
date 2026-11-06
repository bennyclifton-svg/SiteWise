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

// F37: a directory firm that matches only by name is not the firm when the
// page prints a different valid ABN. T02's civil proposal carries the
// descriptor "Civil and stormwater engineers", the name another firm was once
// saved under; T02's Basalt Ridge prints a different ABN from T11's.
func TestNameMatchedDirectoryFirmYieldsToPrintedABN(t *testing.T) {
	for _, tc := range []struct{ file, firm, abn, stored, storedABN string }{
		{"Northfall - civil fee - Calloway Street.md", "NORTHFALL CIVIL ENGINEERING PTY LTD", "36979312905", "Civil and stormwater engineers", "29559935012"},
		{"Geotechnical investigation fee proposal - Basalt Ridge.md", "Basalt Ridge Geotechnical Pty Ltd", "49518838449", "Basalt Ridge Geotechnical Pty Ltd", "27811849176"},
	} {
		t.Run(tc.file, func(t *testing.T) {
			stored := Firm{ID: "f-old", Name: tc.stored, ABN: tc.storedABN, Address: "14 Taren Point Road, Taren Point NSW 2229"}
			// Mirrors FirmLookup: ABN first, then the name key.
			lookup := func(abns, keys []string) *Firm {
				for _, a := range abns {
					if a == stored.ABN {
						return &stored
					}
				}
				for _, k := range keys {
					if k == NameKey(stored.Name) {
						return &stored
					}
				}
				return nil
			}
			d := NewDraft(Harvest(tc.file, syntheticText(t, tc.file)), Context{Lookup: lookup}, nil)
			if d.firm != nil {
				t.Fatalf("matched directory firm %+v", d.firm)
			}
			if got, _ := d.Decided(FieldFirmABN); got.Value != tc.abn {
				t.Errorf("ABN %+v, want %s", got, tc.abn)
			}
			if got, _ := d.Decided(FieldFirmName); !strings.EqualFold(got.Value, tc.firm) {
				t.Errorf("firm %+v, want %s", got, tc.firm)
			}
		})
	}
}

// A name match still fills the firm when the page prints no ABN, or the
// same one.
func TestNameMatchedDirectoryFirmKeptWithoutConflictingABN(t *testing.T) {
	file := "Geotechnical investigation fee proposal - Basalt Ridge.md"
	for _, abn := range []string{"", "49518838449"} {
		stored := Firm{ID: "f1", Name: "Basalt Ridge Geotechnical Pty Ltd", ABN: abn}
		lookup := func(_, keys []string) *Firm {
			for _, k := range keys {
				if k == NameKey(stored.Name) {
					return &stored
				}
			}
			return nil
		}
		d := NewDraft(Harvest(file, syntheticText(t, file)), Context{Lookup: lookup}, nil)
		if d.firm == nil || d.firm.ID != "f1" {
			t.Errorf("stored ABN %q: directory firm not matched", abn)
		}
	}
}

// F38: a larger optional fee ("Option: post-novation services ... $210,000
// ex GST") does not block the printed total (T02 structural, d06).
// F39: a QS service named "DA cost estimate report" is a fee row, not a
// total; the total is $49,500 inc GST, $45,000 ex (T17 QS, d11).
func TestStatedTotalBesideOptionalFeeAndCostEstimateService(t *testing.T) {
	for _, tc := range []struct {
		file  string
		total int64
	}{
		{"Structural engineering fee proposal KBS-Q-2604.md", 15600000},
		{"Quoin fee submission - Tideway Crescent.md", 4500000},
	} {
		t.Run(tc.file, func(t *testing.T) {
			r := NewDraft(Harvest(tc.file, syntheticText(t, tc.file)), Context{}, nil).Finish()
			if r.StatedTotal == nil || *r.StatedTotal != tc.total {
				t.Fatalf("stated total %v, want %d ex GST", r.StatedTotal, tc.total)
			}
		})
	}
}

// F39: the QS's cost estimate service is not marked a total.
func TestCostEstimateServiceIsNotATotal(t *testing.T) {
	file := "Quoin fee submission - Tideway Crescent.md"
	for _, a := range Harvest(file, syntheticText(t, file)).Amounts {
		if a.Total && a.Cents != 4950000 {
			t.Errorf("%s %q marked total", a.Display, a.Context)
		}
	}
}

// The larger-fee guard stays for fees that are not optional: a
// disbursements total beside a larger certification fee is not the stated
// total (the guard F24 relies on).
func TestTotalBesideLargerNonOptionalFeeStaysUnset(t *testing.T) {
	md := "# Fee proposal\n\nFees exclude GST.\n\n## Certification\n\n| Item | Fee (ex GST) |\n|---|---|\n| Construction certificate and inspections | $12,000 |\n\n## Disbursements\n\n| Item | Fee (ex GST) |\n|---|---|\n| Council lodgement | $400 |\n| | **Total** | **$900** |\n"
	r := NewDraft(Harvest("fee.md", mdText(t, md)), Context{}, nil).Finish()
	if r.StatedTotal != nil {
		t.Fatalf("stated total %d beside a larger fee", *r.StatedTotal)
	}
}
