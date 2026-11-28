package reports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"sitewise/internal/delivery"
	"sitewise/internal/invoices"
)

var goldenTitles = map[string]string{
	"cover": "Cover", "quality": "Quality information", "context": "01 Site and project context", "summary": "02 Executive summary",
	"statutory": "03 Statutory approvals", "programme": "04 Programme review", "cost_planning": "05 Cost planning", "design": "06 Design management",
	"site": "07 Site activity", "change": "08 Variation and change control", "insurances": "09 Insurances, security and bonds",
	"attach_a": "Attachment A Payment schedules", "attach_b": "Attachment B",
	"glance": "At a glance", "attention": "Needs attention", "changes": "Changes since the last report", "cost": "Cost",
	"program": "Program", "procurement": "Procurement", "registers": "Registers", "next": "Next period",
}

func item(kind, id, title, status string, number int, details any) delivery.Item {
	raw, _ := json.Marshal(details)
	n := number
	return delivery.Item{ID: id, Number: &n, Content: delivery.Content{Kind: kind, Title: title, Status: status, Details: raw}}
}

func sp(s string) *string { return &s }
func ip(n int) *int       { return &n }

// goldenInput is a project with a works contract (original 994,642.00, one
// approved variation of 56,379.00), an EOT claim granted 3 days on an agreed
// date of 17 November 2027, a notice of delay and a determined approval.
func goldenInput() MonthlyInput {
	original, adjusted := cents(99464200), cents(105102100)
	in := MonthlyInput{ProjectName: "Harbour Street", AsAt: "2026-10-31", Number: 2}
	in.Ledger = &AttentionLedger{Variations: []invoices.Variation{
		{ID: "v1", AwardID: "a1", Number: "V01", Title: "Rock excavation", Status: invoices.VariationApproved, Approved: cents(5637900), SubmittedOn: "2026-10-05"},
		{ID: "v2", AwardID: "a1", Number: "V02", Title: "Extra pier", Status: invoices.VariationSubmitted, Claimed: cents(1200000), SubmittedOn: "2026-10-20"},
	}}
	in.Delivery = []delivery.Item{
		item("eot_claim", "e1", "Rain", "closed", 2, delivery.EOTClaimDetails{ReceivedOn: sp("2026-10-10"), DaysClaimed: ip(5),
			Result: &delivery.EOTResult{Outcome: delivery.OutcomeGrantedInPart, DaysGranted: ip(3), AssessedOn: sp("2026-10-15")}}),
		item("delay_notice", "n1", "Late steel", "open", 3, delivery.DelayNoticeDetails{ReceivedOn: sp("2026-10-12"), Cause: "Steel supplier late"}),
		item("approval", "p1", "S68 approval", "approved", 0, delivery.ApprovalDetails{Authority: "Council", Reference: "S68-12", SubmittedOn: sp("2026-07-01"), DeterminedOn: sp("2026-08-13")}),
	}
	in.Packages = []MonthlyPackage{{ID: "k1", Title: "Main Works", Kind: "works", AwardedTo: "Acme Building", AwardedOn: "2026-09-01"},
		{ID: "k2", Title: "Structural engineer", Kind: "services", AwardedTo: "Beam Co", AwardedOn: "2026-10-03"}}
	in.Golden = &GoldenInput{Title: "Project report", Client: "Harbour Trust", OrgName: "Example PM", Address: "12 Harbour Street", LGA: "Sydney",
		Consent: "DA-2026/14", ConsentOn: "2026-03-02", Period: Period{From: "2026-10-01", To: "2026-10-31", Label: "October 2026"},
		Issues:    []IssueRecord{{Number: 1, Date: "2026-09-30", By: "pm@example.test"}},
		Contracts: []invoices.ContractSummary{{AwardID: "a1", PackageID: "k1", Title: "Main Works", Kind: "works", Original: original, Variations: 5637900, Adjusted: adjusted}},
		Packages:  map[string]string{"a1": "Main Works"}, Agreed: "2027-11-17", Adjusted: "2027-11-20",
		Systems: []GoldenSystem{{ID: "substructure", Label: "Substructure"}}}
	return in
}

func tableByID(t *testing.T, sections []Section, id string) *Table {
	t.Helper()
	for _, s := range sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID == id {
				return b.Table
			}
		}
	}
	t.Fatalf("no table %q", id)
	return nil
}

func rowText(tb *Table) string {
	var out []string
	for _, r := range tb.Rows {
		out = append(out, strings.Join(r, "|"))
	}
	return strings.Join(out, "\n")
}

func TestGoldenSectionsInTheOwnersOrder(t *testing.T) {
	sections, err := AssembleMonthly(goldenInput(), goldenTitles, nil)
	if err != nil {
		t.Fatal(err)
	}
	var issued, review []string
	for _, s := range sections {
		if s.Internal {
			review = append(review, s.ID)
		} else {
			issued = append(issued, s.ID)
		}
	}
	if got := strings.Join(issued, " "); got != "cover quality context summary statutory programme cost_planning design site change insurances attach_a attach_b" {
		t.Fatalf("issued sections %q", got)
	}
	if got := strings.Join(review, " "); got != "glance attention changes cost program procurement registers next" {
		t.Fatalf("review lists %q", got)
	}
	// Every table has rows: records, or the template's placeholders; never blank.
	for _, s := range sections {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Table.ID != "" && (len(b.Table.Rows) == 0 || len(b.Table.Rows) != len(b.Table.Meta)) {
				t.Fatalf("table %s: %d rows, %d facts", b.Table.ID, len(b.Table.Rows), len(b.Table.Meta))
			}
		}
	}
	// 05: the contract's own rollup, and the total.
	if got := rowText(tableByID(t, sections, "cost_planning")); !strings.Contains(got, "Main Works|994,642.00|56,379.00|1,051,021.00") || !strings.Contains(got, "Total|994,642.00|56,379.00|1,051,021.00") {
		t.Fatalf("cost planning:\n%s", got)
	}
	// 08: approved at its amount; an open one at its claim, labelled.
	if got := rowText(tableByID(t, sections, "change")); !strings.Contains(got, "Main Works|V01|Rock excavation|Approved|56,379.00|") || !strings.Contains(got, "Main Works|V02|Extra pier|Submitted|12,000.00 (claimed)|") {
		t.Fatalf("variations:\n%s", got)
	}
	// 04: the agreed date, the EOT with days claimed and granted, the adjusted date in a key cell.
	ms := tableByID(t, sections, "milestones:a1")
	got := rowText(ms)
	for _, want := range []string{"Date for practical completion|17 Nov 2027", "EOT-2 Rain||5|3|3 days granted", "Adjusted date for practical completion|20 Nov 2027"} {
		if !strings.Contains(got, want) {
			t.Fatalf("milestones lack %q:\n%s", want, got)
		}
	}
	last := ms.Meta[len(ms.Meta)-1]
	if last.Key != "pc:adjusted" || len(last.Tones) < 2 || last.Tones[1] != ToneKey {
		t.Fatalf("adjusted row %+v", last)
	}
	// 03: the consent first, the determined approval done, the standing list grey.
	ap := tableByID(t, sections, "approvals")
	if !strings.Contains(rowText(ap), "Council|S68 approval|S68-12|1 Jul 2026|13 Aug 2026|Approved") || ap.Meta[0].Key != "approval:consent" || ap.Meta[1].Tones[4] != ToneDone {
		t.Fatalf("approvals:\n%s\n%+v", rowText(ap), ap.Meta[:2])
	}
	if f := ap.Meta[len(ap.Meta)-1]; f.Kind != RowPlaceholder || !strings.Contains(strings.Join(ap.Rows[len(ap.Rows)-1], "|"), "Fire service|Nil to report") {
		t.Fatalf("placeholder %+v %v", f, ap.Rows[len(ap.Rows)-1])
	}
	// 02: this period's events from the records, under headings.
	sum := rowText(tableByID(t, sections, "summary"))
	for _, want := range []string{"October 2026|", "**Main Works**", "- V02 Extra pier submitted", "**Project**", "- EOT-2 received, 5 days claimed", "- EOT-2 assessed: 3 days granted", "- NOD-3 received: Late steel", "- Structural engineer awarded to Beam Co"} {
		if !strings.Contains(sum, want) {
			t.Fatalf("summary lacks %q:\n%s", want, sum)
		}
	}
	// 07: who is engaged.
	if site := rowText(tableByID(t, sections, "site")); !strings.Contains(site, "Consultant procurement|Engaged:\n- Structural engineer: Beam Co") {
		t.Fatalf("site:\n%s", site)
	}
	// Cover and revision history.
	if c := rowText(tableByID(t, sections, "cover")); !strings.Contains(c, "Period|October 2026") || !strings.Contains(c, "Prepared for|Harbour Trust") {
		t.Fatalf("cover:\n%s", c)
	}
	if r := rowText(tableByID(t, sections, "revisions")); r != "Issue 1|30 Sep 2026|pm@example.test\nIssue 2|TBD|TBD" {
		t.Fatalf("revisions:\n%s", r)
	}
	// No "month" or "monthly" a client reads, whatever the period.
	in := goldenInput()
	in.Golden.Period = Period{From: "2026-10-20", To: "2026-10-26", Label: PeriodLabel("2026-10-20", "2026-10-26")}
	weekly, _ := AssembleMonthly(in, goldenTitles, nil)
	raw, _ := json.Marshal(publicOnly(weekly))
	if strings.Contains(strings.ToLower(string(raw)), "month") {
		t.Fatalf("a weekly report says month: %s", raw)
	}
}

func publicOnly(sections []Section) []Section {
	var out []Section
	for _, s := range sections {
		if !s.Internal {
			out = append(out, s)
		}
	}
	return out
}

func TestUnknownIsTBDAndNothingIsInvented(t *testing.T) {
	in := MonthlyInput{ProjectName: "Empty", Number: 1, Golden: &GoldenInput{Unawarded: []string{"Main Works"}}}
	sections, err := AssembleMonthly(in, goldenTitles, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := rowText(tableByID(t, sections, "cost_planning")); got != "Main Works|TBD|TBD|TBD\nTotal|TBD|TBD|TBD" {
		t.Fatalf("an unawarded package:\n%s", got)
	}
	if got := rowText(tableByID(t, sections, "milestones")); !strings.Contains(got, "Date for practical completion|TBD") {
		t.Fatalf("no contract:\n%s", got)
	}
}

func TestPeriodLabel(t *testing.T) {
	for _, c := range [][3]string{
		{"2026-10-01", "2026-10-31", "October 2026"},
		{"2026-11-03", "2026-11-17", "3 to 17 November 2026"},
		{"2026-10-20", "2026-11-05", "20 October to 5 November 2026"},
		{"2026-12-20", "2027-01-05", "20 December 2026 to 5 January 2027"},
		{"", "2026-11-27", "To 27 November 2026"},
	} {
		if got := PeriodLabel(c[0], c[1]); got != c[2] {
			t.Errorf("PeriodLabel(%s, %s) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}

// The PM's rows: a typed-over cell keeps its words; when SiteWise's value
// under it moves the row is flagged and the block holds issue; a row of
// their own stays; a hidden row goes; a moved row moves.
func TestApplyRows(t *testing.T) {
	sections, err := AssembleMonthly(goldenInput(), goldenTitles, nil)
	if err != nil {
		t.Fatal(err)
	}
	ap := tableByID(t, sections, "approvals")
	consentBase := CellHash(ap.Rows[0][2])
	pos := -1.0
	rows := []ReportRow{
		{ID: "r1", TableID: "approvals", SourceKey: "approval:consent", Cells: map[string]RowCell{"2": {Text: "DA-2026/14 (modified)", Base: consentBase}}},
		{ID: "r2", TableID: "approvals", SourceKey: "approval:p1", Cells: map[string]RowCell{"5": {Text: "Approved with conditions", Base: CellHash("an old value")}}},
		{ID: "r3", TableID: "approvals", Cells: map[string]RowCell{"0": {Text: "Sydney Water"}, "1": {Text: "Section 73"}}},
		{ID: "r4", TableID: "approvals", SourceKey: "ph:approvals:0", Hidden: true},
		{ID: "r5", TableID: "approvals", SourceKey: "ph:approvals:1", Cells: map[string]RowCell{"1": {Text: "Tree permit lodged"}}, Position: &pos},
		{ID: "r6", TableID: "change", SourceKey: "var:gone", Cells: map[string]RowCell{"2": {Text: "My words on a deleted variation", Base: "x"}}},
	}
	out := ApplyRows(sections, rows, time.Time{})
	ap = tableByID(t, out, "approvals")
	text := rowText(ap)
	if ap.Meta[0].Key != "ph:approvals:1" || ap.Meta[0].Kind != RowOwn || !strings.Contains(text, "Arborist|Tree permit lodged") {
		t.Fatalf("moved and typed placeholder first:\n%s\n%+v", text, ap.Meta[0])
	}
	if strings.Contains(text, "Energy distributor") || ap.Hidden != 1 || len(ap.HiddenIDs) != 1 || ap.HiddenIDs[0] != "r4" {
		t.Fatalf("hidden placeholder:\n%s", text)
	}
	if !strings.Contains(text, "DA-2026/14 (modified)") || !strings.Contains(text, "Sydney Water|Section 73") || !strings.Contains(text, "Approved with conditions") {
		t.Fatalf("words:\n%s", text)
	}
	var consent, p1 RowFacts
	for _, f := range ap.Meta {
		switch f.Key {
		case "approval:consent":
			consent = f
		case "approval:p1":
			p1 = f
		}
	}
	if consent.Flag != "" || !consent.Typed[2] || consent.Base[2] != "DA-2026/14" || p1.Flag != "moved" {
		t.Fatalf("consent %+v p1 %+v", consent, p1)
	}
	held := map[string]bool{}
	for _, s := range out {
		for _, b := range s.Blocks {
			if b.Table != nil && b.Conflict {
				held[b.Table.ID] = true
			}
		}
	}
	if !held["approvals"] || !held["change"] {
		t.Fatalf("issue must wait on approvals and change: %v", held)
	}
	ch := tableByID(t, out, "change")
	if f := ch.Meta[len(ch.Meta)-1]; f.Flag != "missing" || !strings.Contains(rowText(ch), "My words on a deleted variation") {
		t.Fatalf("missing source: %+v", f)
	}
	// Move up: halfway between the new neighbours.
	if p, ok := MovePosition(*ap, 2, -1); !ok || p >= ap.Meta[1].Pos {
		t.Fatalf("move up %v %v", p, ok)
	}
	if _, ok := MovePosition(*ap, 0, -1); ok {
		t.Fatal("the first row moved up")
	}
}

// New this period: a row not in the last issue is new, a changed cell is
// marked, placeholders and history are not.
func TestMarkNew(t *testing.T) {
	first, _ := AssembleMonthly(goldenInput(), goldenTitles, nil)
	in := goldenInput()
	in.Ledger.Variations = append(in.Ledger.Variations, invoices.Variation{ID: "v3", AwardID: "a1", Number: "V03", Title: "Handrail", Status: invoices.VariationSubmitted, Estimate: cents(50000)})
	in.Ledger.Variations[1].Claimed = cents(1300000)
	second, _ := AssembleMonthly(in, goldenTitles, nil)
	MarkNew(second, first)
	ch := tableByID(t, second, "change")
	for i, f := range ch.Meta {
		switch f.Key {
		case "var:v3":
			if !f.New {
				t.Fatal("V03 is new")
			}
		case "var:v2":
			if f.New || len(f.Changed) < 5 || !f.Changed[4] || f.Changed[2] {
				t.Fatalf("V02 changed amount only: %+v", f)
			}
		case "var:v1":
			if f.New || f.Changed != nil {
				t.Fatalf("V01 carried: %+v", f)
			}
		}
		if f.Kind == RowPlaceholder && (f.New || f.Changed != nil) {
			t.Fatalf("placeholder %d marked", i)
		}
	}
	// The first report marks nothing.
	MarkNew(first, nil)
	for _, f := range tableByID(t, first, "change").Meta {
		if f.New {
			t.Fatal("first report marked new")
		}
	}
}
