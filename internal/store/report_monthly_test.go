package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/delivery"
	"sitewise/internal/files"
	"sitewise/internal/procurement"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

func blockOf(t *testing.T, d store.ReportDraft, id string) reports.Block {
	t.Helper()
	for _, s := range d.Sections {
		for _, b := range s.Blocks {
			if b.ID == id {
				return b
			}
		}
	}
	t.Fatalf("no block %s in %s", id, blockIDs(d))
	return reports.Block{}
}

func hasBlock(d store.ReportDraft, prefix string) bool {
	for _, s := range d.Sections {
		for _, b := range s.Blocks {
			if strings.HasPrefix(b.ID, prefix) {
				return true
			}
		}
	}
	return false
}

func blockIDs(d store.ReportDraft) string {
	var ids []string
	for _, s := range d.Sections {
		for _, b := range s.Blocks {
			ids = append(ids, b.ID)
		}
	}
	return strings.Join(ids, " ")
}

func sectionText(d store.ReportDraft, section string) string {
	var out []string
	for _, s := range d.Sections {
		if s.ID == section {
			for _, b := range s.Blocks {
				out = append(out, b.Text)
				if b.Table != nil {
					for _, r := range b.Table.Rows {
						out = append(out, strings.Join(r, " | "))
					}
				}
			}
		}
	}
	return strings.Join(out, "\n")
}

// A project report is created once per project and only in its own org.
func TestMonthlyReportCreatesOnceAndIsScoped(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	if err := s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); err != nil {
		t.Fatal(err)
	}
	a, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "")
	if err != nil {
		t.Fatal(err)
	}
	again, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "")
	if err != nil || again.ID != a.ID || a.Title != reports.MonthlyTitle || a.PackageID != "" {
		t.Fatalf("second create %+v %v, first %+v", again, err, a)
	}
	if _, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "11111111-1111-4111-8111-aaaaaaaaaa77"); !errors.Is(err, store.ErrInvalidReport) {
		t.Fatalf("a project report takes no package: %v", err)
	}
	if _, err := s.CreateReport(ctx, orgB, projectA, userB, reports.KindMonthly, ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's create: %v", err)
	}
	if _, err := s.RefreshReport(ctx, orgB, a.ID, userB, "test", true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's update: %v", err)
	}
	if _, err := s.ReadReport(ctx, orgB, a.ID, "test"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org's read: %v", err)
	}
	// An empty project: every surface says so in one honest line.
	d, err := s.RefreshReport(ctx, orgA, a.ID, userA, "test", false)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, sec := range d.Sections {
		ids = append(ids, sec.ID)
	}
	if strings.Join(ids, ",") != "glance,attention,changes,cost,program,procurement,registers,next" {
		t.Fatalf("sections %v", ids)
	}
	for id, want := range map[string]string{"cost:none": "No cost plan yet.", "program:none": "No Program yet.", "proc:none": "No procurement packages yet.", "changes:first": "This is the first issued report"} {
		if b := blockOf(t, d, id); !strings.HasPrefix(b.Text, want) {
			t.Errorf("%s: %q", id, b.Text)
		}
	}
	if text := sectionText(d, "registers"); !strings.Contains(text, "No risks, issues, approvals, decisions, actions, RFIs or consent conditions are recorded in SiteWise.") {
		t.Errorf("registers: %q", text)
	}
	view, err := s.ReadReport(ctx, orgA, a.ID, "test")
	if err != nil || view.PageFit == nil || !view.PageFit.Fits || view.PageFit.Limit != 4 {
		t.Fatalf("page fit %+v %v", view.PageFit, err)
	}
	if strings.Join(view.Stale, ",") != "" {
		t.Fatalf("a fresh draft is stale: %v", view.Stale)
	}
}

// monthlyFixture is a project with a cost plan whose fee lines hold budgets,
// invoices on them, a Program and a register item.
func monthlyFixture(t *testing.T) (*store.Store, string, string) {
	t.Helper()
	ctx := context.Background()
	s, pkg, line, _ := ledgerFixture(t)
	for i, total := range []int64{110000, 220000} {
		created, err := s.CreateInvoice(ctx, orgA, projectA, upload("monthly-"+string(rune('a'+i))), time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CommitInvoiceRead(ctx, orgA, created.ID, reading(pkg, line, "M-"+string(rune('a'+i)), total, total/11), false, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	newProgram(t, s)
	target := "2020-01-01"
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "risk", Title: "Latent ground conditions", Status: "open", TargetDate: &target, Details: json.RawMessage(`{}`)}}); err != nil {
		t.Fatal(err)
	}
	return s, pkg, line
}

// Every cost, Program and procurement figure in the draft is the owning
// screen's figure (Done means 2).
func TestMonthlyReportFiguresEqualTheScreens(t *testing.T) {
	ctx := context.Background()
	s, pkg, _ := monthlyFixture(t)
	r, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	totals, err := s.CostTotals(ctx, orgA, projectA, "", "overall")
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := s.InvoiceLedger(ctx, orgA, projectA, store.SydneyDate(time.Now()))
	if err != nil {
		t.Fatal(err)
	}
	var cost struct {
		TaxBasis string                    `json:"tax_basis"`
		Figures  map[string]map[string]any `json:"figures"`
	}
	table := blockOf(t, d, "cost:table")
	if err := json.Unmarshal(table.Basis, &cost); err != nil {
		t.Fatal(err)
	}
	want := func(t costs.Total) any {
		if t.Lines == 0 || t.Amount == nil || !t.Complete {
			return nil
		}
		return string(*t.Amount)
	}
	for _, section := range reports.CostSectionOrder {
		for _, metric := range []string{"budget", "commitment", "forecast"} {
			if got := cost.Figures[section][metric]; got != want(totals.Sections[section][metric]) {
				t.Errorf("%s %s: report %v, cost plan %v", section, metric, got, want(totals.Sections[section][metric]))
			}
		}
	}
	for _, metric := range []string{"budget", "commitment", "forecast"} {
		if got := cost.Figures["total"][metric]; got != want(totals.Overall[metric]) {
			t.Errorf("total %s: report %v, cost plan %v", metric, got, want(totals.Overall[metric]))
		}
	}
	if ledger.Summary.Total == 0 {
		t.Fatal("fixture has no invoices")
	}
	if got := cost.Figures["total"]["claimed"]; got != centsText(ledger.Summary.Total) {
		t.Errorf("claimed to date: report %v, ledger %d", got, ledger.Summary.Total)
	}
	// The printed row is the grid's: the total row's claimed column.
	last := table.Table.Rows[len(table.Table.Rows)-1]
	if last[0] != "Total" || last[4] != dollarText(ledger.Summary.Total) {
		t.Errorf("total row %v, ledger %d", last, ledger.Summary.Total)
	}
	program, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	var completion struct {
		Typical string `json:"typical"`
	}
	if err := json.Unmarshal(blockOf(t, d, "program:completion").Basis, &completion); err != nil {
		t.Fatal(err)
	}
	if program.Computed.Finish.Typical == nil || completion.Typical != *program.Computed.Finish.Typical {
		t.Errorf("completion: report %q, Program %v", completion.Typical, program.Computed.Finish.Typical)
	}
	if !strings.Contains(blockOf(t, d, "program:completion").Text, reports.LongDate(*program.Computed.Finish.Typical)) {
		t.Errorf("completion text %q", blockOf(t, d, "program:completion").Text)
	}
	var proc struct {
		Packages map[string]map[string]any `json:"packages"`
	}
	if err := json.Unmarshal(blockOf(t, d, "proc:table").Basis, &proc); err != nil {
		t.Fatal(err)
	}
	if _, ok := proc.Packages[pkg]; !ok {
		t.Errorf("package %s missing from %v", pkg, proc.Packages)
	}
	award, err := s.ReadAward(ctx, orgA, projectA, pkg)
	if err != nil || award != nil {
		t.Fatalf("fixture award %+v %v", award, err)
	}
	if got := proc.Packages[pkg]["awarded_to"]; got != "" {
		t.Errorf("awarded to %v with no award", got)
	}
	// The overdue risk is in the registers and in Needs attention's words.
	if text := sectionText(d, "registers"); !strings.Contains(text, "1 risk open, 1 past the target date of 1 recorded.") || !strings.Contains(text, "Latent ground conditions") {
		t.Errorf("registers %q", text)
	}
}

func centsText(c int64) string {
	sign := ""
	if c < 0 {
		sign, c = "-", -c
	}
	return fmt.Sprintf("%s%d.%02d", sign, c/100, c%100)
}

func dollarText(c int64) string {
	whole := strconv.FormatInt(c/100, 10)
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := "$" + b.String()
	if c%100 != 0 {
		out += "." + strings.TrimRight(fmt.Sprintf("%02d", c%100), "0")
	}
	return out
}

// Two issues: report 1 says it is the first; report 2 lists what moved and
// what was removed, and a carried comment holds issue until it is kept
// (Done means 3, 4 and 5).
func TestMonthlyReportTwoIssuesMovementAndComments(t *testing.T) {
	ctx := context.Background()
	s, _, line := monthlyFixture(t)
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if b := blockOf(t, d, "changes:first"); !strings.HasPrefix(b.Text, "This is the first issued report") {
		t.Fatalf("report 1 changes %q", b.Text)
	}
	d, err = s.EditReport(ctx, orgA, r.ID, userA, "comment:cost", "Fees are tracking to the budget.", d.Version)
	if err != nil {
		t.Fatal(err)
	}
	// A comment survives Update while nothing moved.
	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if b := blockOf(t, d, "comment:cost"); b.Text != "Fees are tracking to the budget." || !b.Edited || b.Conflict {
		t.Fatalf("comment after Update %+v", b)
	}
	first, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-17", AcceptStale: true, StaleReason: "Fixture"}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	if first.Number != 1 || first.Snapshot.Title != "Project report No. 1" {
		t.Fatalf("issue 1 %d %q", first.Number, first.Snapshot.Title)
	}
	saved, err := s.ReadIssuedReport(ctx, orgA, r.ID, first.ID)
	if err != nil || saved.SnapshotSHA256 != first.SnapshotSHA256 {
		t.Fatalf("issued read %v", err)
	}

	// Between reports: a contract appears on the fee line, a Program row
	// gets longer, the risk is closed and a decision is opened.
	plan, err := s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.PutCostValue(ctx, orgA, projectA, line, userA, "commitment", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: cv("12000.00")}); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadProgram(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	apply(t, s, st, store.ProgramOp{Op: "update", ID: rowID(t, st, "Framing"), DurationDays: dur(25)})
	items, err := s.ReadDelivery(ctx, orgA, projectA)
	if err != nil || len(items) != 1 {
		t.Fatalf("delivery %v %v", items, err)
	}
	if _, err := s.PatchDelivery(ctx, orgA, projectA, items[0].ID, userA, store.DeliveryPatch{Patch: delivery.Patch{Version: items[0].Version, Status: sp("closed")}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDelivery(ctx, orgA, projectA, userA, store.DeliveryInput{Content: delivery.Content{Kind: "decision", Title: "Facade colour", Status: "open", Details: json.RawMessage(`{}`)}}); err != nil {
		t.Fatal(err)
	}

	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if d.Number != 2 {
		t.Fatalf("draft number %d", d.Number)
	}
	changes := sectionText(d, "changes")
	for _, want := range []string{
		"Consultants approved contract: the known part is up $12,000, from $0 to $12,000; 1 line is still not known.",
		"Planned completion 14 days later, from 28 May 2027 to 11 June 2027.",
		"1 risk closed: Latent ground conditions.",
		"1 decision opened: Facade colour.",
	} {
		if !strings.Contains(changes, want) {
			t.Errorf("changes lack %q:\n%s", want, changes)
		}
	}
	if glance := blockOf(t, d, "glance:period"); !strings.Contains(glance.Text, "Since report 1 (17 October 2026)") {
		t.Errorf("period %q", glance.Text)
	}
	// The completion moved, so Needs attention says so, from the one rule.
	if text := sectionText(d, "attention"); !strings.Contains(text, "later than report 1 printed") {
		t.Errorf("attention %q", text)
	}
	// Last period's comment came forward, flagged, and holds issue.
	comment := blockOf(t, d, "comment:cost")
	if comment.Text != "Fees are tracking to the budget." || !comment.Conflict {
		t.Fatalf("carried comment %+v", comment)
	}
	if _, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-11-17", AcceptStale: true, StaleReason: "Fixture"}, blobs); !errors.Is(err, store.ErrVersionConflict) {
		t.Fatalf("issue with a flagged comment: %v", err)
	}
	// One save keeps it.
	d, err = s.EditReport(ctx, orgA, r.ID, userA, "comment:cost", comment.Text, d.Version)
	if err != nil {
		t.Fatal(err)
	}
	for _, sec := range d.Sections {
		for _, b := range sec.Blocks {
			if b.Conflict && b.ID != "comment:cost" {
				d, err = s.EditReport(ctx, orgA, r.ID, userA, b.ID, b.Text, d.Version)
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	second, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-11-17", AcceptStale: true, StaleReason: "Fixture"}, blobs)
	if err != nil {
		t.Fatal(err)
	}
	if second.Number != 2 || second.Snapshot.Title != "Project report No. 2" {
		t.Fatalf("issue 2 %d %q", second.Number, second.Snapshot.Title)
	}
	// Both issues stay listable and frozen, newest first.
	view, err := s.ReadReport(ctx, orgA, r.ID, "test")
	if err != nil || len(view.Issues) != 2 || view.Issues[0].Number != 2 || view.Issues[1].Number != 1 {
		t.Fatalf("issues %+v %v", view.Issues, err)
	}
	again, err := s.ReadIssuedReport(ctx, orgA, r.ID, first.ID)
	if err != nil || again.SnapshotSHA256 != first.SnapshotSHA256 {
		t.Fatalf("issue 1 changed: %v", err)
	}
	last, err := s.LastReportCompletion(ctx, orgA, projectA)
	if err != nil || last == nil || last.Number != 2 {
		t.Fatalf("last completion %+v %v", last, err)
	}
	if _, err := s.LastReportCompletion(ctx, orgB, projectA); err != nil {
		t.Fatal(err)
	}
	if other, _ := s.LastReportCompletion(ctx, orgB, projectA); other != nil {
		t.Fatal("another org read this project's report")
	}
}

// A package removed between reports is listed as removed.
func TestMonthlyReportListsARemovedPackage(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Acoustic engineer", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateReport(ctx, orgA, projectA, userA, reports.KindMonthly, "")
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssueReport(ctx, orgA, r.ID, userA, "test", store.IssueOptions{Version: d.Version, ReportingDate: "2026-10-17", AcceptStale: true, StaleReason: "Fixture"}, blobs); err != nil {
		t.Fatal(err)
	}
	retired := true
	if _, err := s.PatchPackage(ctx, orgA, projectA, pkg.ID, userA, procurement.PackagePatch{Version: pkg.Version, Retired: &retired}); err != nil {
		t.Fatal(err)
	}
	d, err = s.RefreshReport(ctx, orgA, r.ID, userA, "test", true)
	if err != nil {
		t.Fatal(err)
	}
	if text := sectionText(d, "changes"); !strings.Contains(text, "Acoustic engineer was removed from procurement.") {
		t.Fatalf("changes %q", text)
	}
}
