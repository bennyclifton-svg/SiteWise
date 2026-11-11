package proposals

import (
	"fmt"
	"strings"
	"testing"

	"sitewise/internal/identity"
)

// builderQuote lays out an estimating tool's quote as its PDF prints it:
// trade notes, an allowances table, a category table whose long
// descriptions wrap around the row's number and amount, and a summary whose
// GST was summed per row (8 cents off 10% of the subtotal).
func builderQuote() identity.Text {
	t := identity.Text{TextLayer: true, Format: "pdf"}
	page, y := 1, 790.0
	put := func(x, h float64, s string) {
		t.Runs = append(t.Runs, identity.Run{Text: s, Source: identity.Source{Page: page, X: x, Y: y, Width: float64(len(s)) * 4.5, Height: h}})
	}
	row := func(cells ...any) {
		for i := 0; i+1 < len(cells); i += 2 {
			put(cells[i].(float64), 11.6, cells[i+1].(string))
		}
		y -= 17
	}
	footer := func() {
		y = 60
		row(430.0, "Seaview Builders")
		row(430.0, "Email: office@seaviewbuilders.com.au")
		row(430.0, "ABN: 51824753556")
	}
	newPage := func() { footer(); page++; y = 790 }

	row(34.0, "Dear Priya & Tom,")
	row(34.0, "Please find attached a quote from Seaview Builders for your new home.")
	row(62.0, "• PRICE - Total price for demolition services listed above is $46,100 (incl GST).")
	row(62.0, "• Please note an impending glass price increase of 4% as of September 2026. This will add approximately $950 (ex GST).")
	// Enough trade notes to pass the old 20,000-character section cap.
	for i := 0; i < 260; i++ {
		if y < 100 {
			newPage()
		}
		row(62.0, fmt.Sprintf("• Trade note %03d: supply and install as specified, including all fixings, flashings and trims required.", i))
	}
	row(62.0, "• Excludes dropped pelmets to living/dining room.")
	newPage()

	put(210, 17.4, "Allowances For Provisional Sums")
	y -= 20
	row(79.0, "Description of Provisional Sum Item", 424.0, "Quantity UOM")
	put(505, 11.6, "Allowance incl.")
	y -= 10
	put(542, 11.6, "GST")
	y -= 17
	row(36.0, "19", 79.0, "Security / Intercom / Data - See Specifications")
	row(63.0, "19.1 Security / Intercom / Data", 434.0, "1.000", 521.0, "$35,000.00")
	row(407.0, "Allowance including GST: $35,000.00")
	newPage()

	put(247, 17.4, "Quoted Categories")
	y -= 20
	row(65.0, "Description of category", 505.0, "Total incl. GST")
	// A wrapped description sits half a line above and below its number and
	// amount.
	y -= 5
	put(65, 11.9, "Preliminaries Including Builders Warranty Insurance, Construction & Public Liability")
	y -= 5
	put(36, 11.9, "1")
	put(505, 11.9, "$159,123.20")
	y -= 5
	put(65, 11.9, "Insurances & Office Administration")
	y -= 17
	row(36.0, "2", 65.0, "Location of Services", 523.0, "$916.01")
	row(36.0, "3", 65.0, "Hire Items - Site Toilet, Temporary Fencing, Bin Hire & Fall Protection", 510.0, "$49,879.75")
	for i := 4; i <= 18; i++ {
		row(36.0, fmt.Sprint(i), 65.0, fmt.Sprintf("Trade category %d - See Specifications", i), 510.0, "$10,000.00")
	}
	row(36.0, "19", 65.0, "Security / Intercom / Data - See Specifications", 510.0, "$35,000.00")
	row(36.0, "20", 65.0, "Solar Panels & Battery System - as per quotation provided", 510.0, "$90,576.51")
	newPage()

	put(240, 17.4, "Terms and Conditions")
	y -= 30
	// 159,123.20 + 916.01 + 49,879.75 + 15 x 10,000 + 35,000 + 90,576.51
	row(390.0, "Quote Total:", 494.0, "$441,359.47")
	row(390.0, "Tax (GST):", 503.0, "$44,136.00")
	row(390.0, "Total:", 485.0, "$485,495.47")
	footer()
	return t
}

func TestBuilderQuoteCategoryRowsKeepTheirOwnWords(t *testing.T) {
	ev := Harvest("Seaview-Builders.pdf", builderQuote())
	for display, label := range map[string]string{
		"$159,123.20": "Preliminaries Including Builders Warranty Insurance, Construction & Public Liability Insurances & Office Administration",
		"$916.01":     "Location of Services",
		"$49,879.75":  "Hire Items - Site Toilet, Temporary Fencing, Bin Hire & Fall Protection",
		"$90,576.51":  "Solar Panels & Battery System - as per quotation provided",
	} {
		a, ok := find(ev, display)
		if !ok || !strings.HasPrefix(label, strings.TrimSuffix(a.Label, "…")) || len(a.Label) < 20 || a.GST != "inc" || a.Title != "Quoted Categories" {
			t.Errorf("%s: %+v (found %v)", display, a, ok)
		}
	}
	if a, ok := find(ev, "$35,000.00"); !ok || a.Title != "Allowances For Provisional Sums" || a.GST != "inc" || a.Label != "19.1 Security / Intercom / Data" {
		t.Errorf("allowance row %+v", a)
	}
	// Nothing after the old cap is dropped from the stored text.
	var all strings.Builder
	for _, s := range ev.Sections {
		if len(s.Body) > 20000 {
			t.Fatalf("section %d holds %d characters", s.Ordinal, len(s.Body))
		}
		all.WriteString(s.Body)
	}
	if !strings.Contains(all.String(), "Excludes dropped pelmets") || !strings.Contains(all.String(), "Trade note 259") {
		t.Fatal("text after the first 20,000 characters is lost")
	}
}

func TestBuilderQuoteTotalFirmAndGSTSettleByRule(t *testing.T) {
	d := NewDraft(Harvest("Seaview-Builders.pdf", builderQuote()), Context{}, nil)
	r := d.Finish()
	if got := r.FirmFields[FieldFirmName]; got != "Seaview Builders" {
		t.Errorf("firm %q", got)
	}
	if r.GSTBasis != "inc" {
		t.Errorf("GST basis %q", r.GSTBasis)
	}
	// Stored ex GST: the printed "Quote Total".
	if r.StatedTotal == nil || *r.StatedTotal != 44135947 {
		t.Errorf("stated total %v", r.StatedTotal)
	}
	for _, f := range []string{FieldFirmName, FieldGSTBasis, FieldStatedTotal} {
		if dec := d.by[f]; dec.DecidedBy != ByRule || dec.Band != BandGreen {
			t.Errorf("%s decided %+v", f, dec)
		}
	}
	roles := map[string]string{}
	for _, f := range r.Quote.Fees {
		roles[f.Printed] = f.Role
		if f.Printed == "$159,123.20" && (f.Amount == nil || *f.Amount != "144657.45") {
			t.Errorf("category amount ex GST %v", f.Amount)
		}
	}
	if _, listed := roles["$44,136.00"]; listed {
		t.Error("the GST line is listed as a price")
	}
	if _, listed := roles["$485,495.47"]; listed {
		t.Error("the inclusive copy of the total is listed as a second total")
	}
	if roles["$441,359.47"] != RoleTotal {
		t.Errorf("quote total role %q", roles["$441,359.47"])
	}
	if roles["$916.01"] != RoleComponent || roles["$90,576.51"] != RoleComponent {
		t.Errorf("category rows %q %q", roles["$916.01"], roles["$90,576.51"])
	}
	if roles["$46,100"] == RoleTotal {
		t.Error("one trade's price is called the package total")
	}
	if !strings.HasPrefix(roles["$950"], "Conditional") {
		t.Errorf("price increase note %q", roles["$950"])
	}
	for _, issue := range r.Quote.Issues {
		if strings.Contains(issue, "add") {
			t.Errorf("rows reconcile, yet: %s", issue)
		}
	}
}

// The allowance table repeats one category's figure; it is inside the
// price, not beside it.
func TestBuilderQuoteAllowancesAreInsideThePrice(t *testing.T) {
	r := NewDraft(Harvest("Seaview-Builders.pdf", builderQuote()), Context{}, nil).Finish()
	n := 0
	for _, f := range r.Quote.Fees {
		if f.Printed == "$35,000.00" && f.Page < 12 && f.Role == RoleAllowance {
			n++
		}
	}
	if n == 0 {
		t.Fatalf("no allowance role: %+v", r.Quote.Fees)
	}
}

// A category table that does not add up to the printed total says so.
func TestBuilderQuoteRowsThatDoNotAddUpAreFlagged(t *testing.T) {
	text := builderQuote()
	for i, run := range text.Runs {
		if run.Text == "$916.01" {
			text.Runs[i].Text = "$900.00"
		}
	}
	r := NewDraft(Harvest("Seaview-Builders.pdf", text), Context{}, nil).Finish()
	found := false
	for _, issue := range r.Quote.Issues {
		found = found || strings.Contains(issue, "Quoted Categories")
	}
	if !found {
		t.Fatalf("issues %q", r.Quote.Issues)
	}
}
