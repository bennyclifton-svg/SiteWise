package reports

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/invoices"
)

// printedSheet is a small contract at its second claim: two works lines (one
// cut below claimed), a provisional sum marked final, an approved variation,
// retention at 5% of the works, and an earlier claim in the payment list.
func printedSheet() invoices.ScheduleSheet {
	rate := 500
	return invoices.ScheduleSheet{
		ClaimID: "c2",
		Sections: []invoices.SheetSection{
			{Kind: invoices.SectionWorks, TotalLet: cents(15000000), Completed: cents(11000000), Prior: 6000000, ThisClaim: cents(5000000), Lines: []invoices.SheetLine{
				{Key: "b:1", Label: "Substructure", TotalLet: cents(10000000), Completed: cents(9000000), Percent: "90.0", Prior: 6000000, ThisClaim: cents(3000000),
					ClaimedToDate: cents(10000000), BelowClaimed: true, Reason: "Footings to grid C not poured"},
				{Key: "b:2", Label: "Superstructure", Note: "Levels 1 to 3", TotalLet: cents(5000000), Completed: cents(2000000), Percent: "40.0", ThisClaim: cents(2000000)},
			}},
			{Kind: invoices.SectionProvisional, TotalLet: cents(480000), Completed: cents(240000), ThisClaim: cents(240000), Lines: []invoices.SheetLine{
				{Key: "b:3", Label: "Rock, 40 tonne", TotalLet: cents(480000), Completed: cents(240000), Percent: "50.0", ThisClaim: cents(240000), Final: true},
			}},
			{Kind: invoices.SectionVariations, TotalLet: cents(120000), Completed: cents(0), ThisClaim: cents(0), Lines: []invoices.SheetLine{
				{Key: "v:1", Label: "V-001 Extra pier", TotalLet: cents(120000), Completed: cents(0), Percent: "0.0", ThisClaim: cents(0)},
			}},
		},
		TotalLet: cents(15600000), Completed: cents(11240000), Prior: 6000000, ThisClaim: cents(5240000),
		Retention: invoices.SheetRetention{RateBP: &rate, BaseKind: invoices.RetentionBaseWorks, Base: cents(15000000), Limit: cents(750000),
			HeldBefore: 300000, Held: cents(250000), Percent: "1.7", HeldToDate: cents(550000)},
		Net: cents(4990000), GST: cents(499000), Payable: cents(5489000),
		Payments: []invoices.PaymentRow{
			{ClaimID: "c1", Period: "SEP26", Number: "01", InvoiceNo: "8642", InvoiceDate: "2026-09-30", Ex: cents(5700000), GST: cents(570000), Inc: cents(6270000)},
			{ClaimID: "c2", Period: "OCT26", Number: "02", Ex: cents(4990000), GST: cents(499000), Inc: cents(5489000), Current: true},
		},
		Paid: invoices.PaymentTotals{Ex: cents(10690000), GST: cents(1069000), Inc: cents(11759000)},
	}
}

func printedHead() ScheduleHead {
	return ScheduleHead{Project: "Harbour Street", Address: "12 Harbour Street, Sydney", Contract: "Main Works",
		Principal: "Harbour Trust", PrincipalABN: "51 824 753 556", Contractor: "Acme Building", ContractorABN: "53 004 085 616",
		AssessorOrg: "Example PM Pty Ltd", AssessorNo: "605 081 488", Approver: "A. Manager",
		Number: "02", Period: "OCT26", Received: "2026-10-12", ReferenceDate: "2026-09-30", ContractDate: "2026-02-25",
		IssuedOn: "2026-10-16", Note: "Retention is held on the works only.", RetentionReason: "The contract holds 5% of the works"}
}

func sectionByID(t *testing.T, sections []Section, id string) Section {
	t.Helper()
	for _, s := range sections {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no section %q", id)
	return Section{}
}

func tableOf(t *testing.T, s Section) *Table {
	t.Helper()
	for _, b := range s.Blocks {
		if b.Table != nil {
			return b.Table
		}
	}
	t.Fatalf("section %q has no table", s.ID)
	return nil
}

// The issued schedule prints the golden's sections, in the golden's order,
// with the golden's columns; every figure is the sheet's.
func TestScheduleSheetSectionsFollowTheGolden(t *testing.T) {
	sections := ScheduleSheetSections(printedSheet(), printedHead())
	var order []string
	for _, s := range sections {
		order = append(order, s.ID)
	}
	want := "details works provisional variations totals note payments approval"
	if got := strings.Join(order, " "); got != want {
		t.Fatalf("sections %q, want %q", got, want)
	}
	// A section is named in its table's first head, as on screen.
	works := tableOf(t, sectionByID(t, sections, "works"))
	if got := strings.Join(works.Columns, "|"); got != "Contract summary: Works|Total let ($)|(%)|Completed ($)|Less prior ($)|This claim ($)" {
		t.Fatalf("columns %q", got)
	}
	for id, want := range map[string]string{"provisional": "Provisional sum adjustment", "variations": "Variations", "totals": "Total and retention"} {
		if got := tableOf(t, sectionByID(t, sections, id)).Columns[0]; got != want || sectionByID(t, sections, id).Title != "" {
			t.Fatalf("%s is headed %q under the title %q", id, got, sectionByID(t, sections, id).Title)
		}
	}
	rows := func(tb *Table) string {
		var out []string
		for _, r := range tb.Rows {
			out = append(out, strings.Join(r, "|"))
		}
		return strings.Join(out, "\n")
	}
	if got, want := rows(works), strings.Join([]string{
		"Substructure|100,000.00|90.0|90,000.00|60,000.00|30,000.00",
		"Reason: Footings to grid C not poured|||||",
		"Superstructure\nLevels 1 to 3|50,000.00|40.0|20,000.00|–|20,000.00",
		"Sub total ex GST ($)|150,000.00||110,000.00|60,000.00|50,000.00",
	}, "\n"); got != want {
		t.Fatalf("works rows:\n%s\nwant:\n%s", got, want)
	}
	// A reason is one cell of words across the table; a sub total is bold.
	var meta struct {
		Layout string `json:"layout"`
		Bold   []int  `json:"bold"`
		Notes  []int  `json:"notes"`
	}
	for _, b := range sectionByID(t, sections, "works").Blocks {
		if b.Table != nil {
			if err := json.Unmarshal(b.Basis, &meta); err != nil {
				t.Fatal(err)
			}
		}
	}
	if meta.Layout != "schedule" || len(meta.Bold) != 1 || meta.Bold[0] != 3 || len(meta.Notes) != 1 || meta.Notes[0] != 1 {
		t.Fatalf("layout %+v", meta)
	}
	if got := rows(tableOf(t, sectionByID(t, sections, "provisional"))); !strings.Contains(got, "Rock, 40 tonne (final)|4,800.00|50.0|2,400.00|–|2,400.00") {
		t.Fatalf("provisional rows:\n%s", got)
	}
	if got, want := rows(tableOf(t, sectionByID(t, sections, "totals"))), strings.Join([]string{
		"Total (ex GST)|156,000.00||112,400.00|60,000.00|52,400.00",
		"Retention limit: 5% of the works sub total|−7,500.00||||",
		"Retention held before this claim||||3,000.00|",
		"Retention held this claim||1.7|||−2,500.00",
		"Reason: The contract holds 5% of the works|||||",
		"Retention held to date|||5,500.00||",
		"Total claim inc retention (ex GST)|||||49,900.00",
		"GST|||||4,990.00",
		"Amount payable (inc GST)|||||54,890.00",
	}, "\n"); got != want {
		t.Fatalf("totals rows:\n%s\nwant:\n%s", got, want)
	}
	if got, want := rows(tableOf(t, sectionByID(t, sections, "payments"))), strings.Join([]string{
		"SEP26|01|8642|30 Sep 2026|57,000.00|5,700.00|62,700.00",
		"OCT26|02|Awaiting invoice||49,900.00|4,990.00|54,890.00",
		"Total||||106,900.00|10,690.00|117,590.00",
	}, "\n"); got != want {
		t.Fatalf("payment list:\n%s\nwant:\n%s", got, want)
	}
	text := func(id string) string {
		var out []string
		for _, b := range sectionByID(t, sections, id).Blocks {
			if b.Text != "" {
				out = append(out, b.Text)
			}
			if b.Table != nil {
				out = append(out, rows(b.Table))
			}
		}
		return strings.Join(out, "\n")
	}
	details := text("details")
	for _, want := range []string{"Project: Harbour Street", "Address: 12 Harbour Street, Sydney", "Principal: Harbour Trust (ABN 51 824 753 556)", "Contractor: Acme Building (ABN 53 004 085 616)",
		"Principal's rep: Example PM Pty Ltd (ACN 605 081 488)", "Claim no.: 02", "Claim received: 12 October 2026", "Period: OCT26", "Contract date: 25 February 2026",
		"Invoice no.: Awaiting invoice", "Issued: 16 October 2026", "This is a payment schedule."} {
		if !strings.Contains(details, want) {
			t.Fatalf("details lack %q:\n%s", want, details)
		}
	}
	// No Act is named, and no section number, unless the head carries one.
	if strings.Contains(details, "Act") {
		t.Fatalf("an Act was printed with none verified:\n%s", details)
	}
	if got := text("note"); got != "Scheduled for less than claimed. The reason is printed under each line.\nRetention is held on the works only." {
		t.Fatalf("note %q", got)
	}
	approval := tableOf(t, sectionByID(t, sections, "approval"))
	if got := strings.Join(approval.Columns, "|") + "\n" + rows(approval); got != "Approved by|Signature|Date\nA. Manager||16 October 2026" {
		t.Fatalf("approval %q", got)
	}
}

func TestScheduleSheetSectionsSayOnlyWhatIsKnown(t *testing.T) {
	sheet, head := printedSheet(), printedHead()
	// Paid in full, no terms, nothing provisional, no variations, an unknown
	// sum, a claim no schedule answered in time.
	sheet.PaidInFull = true
	sheet.Sections = sheet.Sections[:1]
	sheet.Sections[0].Lines[0].BelowClaimed, sheet.Sections[0].Lines[0].Reason = false, ""
	sheet.Sections[0].Lines[1].TotalLet, sheet.Sections[0].Lines[1].Percent = nil, ""
	sheet.Sections[0].TotalLet, sheet.TotalLet = nil, nil
	nothing := int64(0)
	sheet.Retention = invoices.SheetRetention{Held: &nothing, HeldToDate: &nothing}
	sheet.Payments[0].Unserved = true
	head.Act, head.Note, head.RetentionReason, head.Address, head.ContractDate, head.Principal, head.PrincipalABN = "Building and Construction Industry Security of Payment Act 1999 (NSW)", "", "", "", "", "", ""
	head.DueOn, head.Late = "2026-10-14", true
	sections := ScheduleSheetSections(sheet, head)
	var order []string
	all := ""
	for _, s := range sections {
		order = append(order, s.ID)
		for _, b := range s.Blocks {
			all += b.Text + "\n"
			if b.Table != nil {
				for _, r := range b.Table.Rows {
					all += strings.Join(r, "|") + "\n"
				}
			}
		}
	}
	if got := strings.Join(order, " "); got != "details works totals note payments approval" {
		t.Fatalf("sections %q", got)
	}
	for _, want := range []string{
		"This is a payment schedule made under the Building and Construction Industry Security of Payment Act 1999 (NSW).",
		"Superstructure\nLevels 1 to 3|unknown||20,000.00|–|20,000.00",
		"Sub total ex GST ($)|unknown||110,000.00|60,000.00|50,000.00",
		"No retention change this claim|||||–",
		"Progress claim to be paid in full.",
		"SEP26|01|8642 (no schedule served)|30 Sep 2026|57,000.00|5,700.00|62,700.00",
	} {
		if !strings.Contains(all, want) {
			t.Fatalf("lacks %q:\n%s", want, all)
		}
	}
	for _, not := range []string{"due date", "Retention limit", "Address:", "Contract date:", "Principal:", "Reason:"} {
		if strings.Contains(all, not) {
			t.Fatalf("prints %q with nothing behind it:\n%s", not, all)
		}
	}
}

// The same sheet freezes to the same bytes: an issued snapshot's hash is
// stable.
func TestScheduleSheetSnapshotIsStable(t *testing.T) {
	snap := func() string {
		s := IssueSnapshot{SchemaVersion: 1, ReportID: "r", VersionID: "v", Kind: KindPaymentSchedule, Title: "Payment schedule 02", ReportingDate: "2026-10-16",
			SourceRevisions: SourceState{TemplateID: "tpl.payment-schedule-sheet", Domains: map[string]int64{}}, Sections: ScheduleSheetSections(printedSheet(), printedHead()),
			References: []Reference{}, RendererVersion: "x"}
		_, hash, err := CanonicalSnapshot(s)
		if err != nil {
			t.Fatal(err)
		}
		return hash
	}
	if a, b := snap(), snap(); a != b || a == "" {
		t.Fatalf("hashes %q %q", a, b)
	}
}
