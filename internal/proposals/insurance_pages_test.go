package proposals

import "testing"

// Willow Tree's town planning proposal (Hale 2, 20 Nov 2026) prints its
// stage fees and its insurance table on one page, then attaches its
// certificates of currency. The cover limits and the workers compensation
// wages are not fees: read as fees, they outweighed the $11,000 total and
// the stated total was never settled.
func TestInsuranceTableAndCertificateAreNotFees(t *testing.T) {
	ev := Harvest("Fee Town Planning (Willow Tree).pdf", text(
		page(5,
			[]string{"Appendix B – Remuneration Schedule"},
			[]string{"Stage Costing – Lump Sum"},
			[]string{"Stage 1: Schematic Design (Lodgement of DA)", "$8,500"},
			[]string{"Stage 2: Detailed Design", "N/A"},
			[]string{"Stage 3: Post Lodgement", "$2,500"},
			[]string{"Total (excl. GST)", "$11,000*"},
			[]string{"Appendix C – Insurances"},
			[]string{"Insurance Type", "Company", "Policy Number", "Extent of Cover", "Expiry"},
			[]string{"BIA", "201909-0444 R3", "$20,000,000.00", "31/08/2023"},
			[]string{"icare", "116558901", "$1,259,757.61", "31/08/2023"},
		),
		// OCR reads the certificate's two-line title across its logo.
		page(16,
			[]string{"Certificate icare Workers Insurance of currency"},
			[]string{"WILLOWTREE PLANNING PTY LTD"},
			[]string{"Statement of coverage"},
			[]string{"Policy number:", "116558901"},
			[]string{"782300 Consulting Engineering Services", "$1,259,757.61"},
		),
	))
	for _, fee := range []string{"$8,500", "$2,500", "$11,000"} {
		if _, ok := find(ev, fee); !ok {
			t.Errorf("fee %s not read", fee)
		}
	}
	for _, a := range ev.Amounts {
		if a.Display == "$20,000,000.00" || a.Display == "$1,259,757.61" {
			t.Errorf("insurance figure read as a fee: page %d %s %q", a.Page, a.Display, a.Label)
		}
	}
}
