package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"testing"
	"time"

	"sitewise/internal/files"
	"sitewise/internal/invoices"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

func cp(n int64) *int64 { return &n }

// dropClaim drops a claim on the contract and commits a read of it: claim
// number 4, 46,000 claimed ex GST across three contract rows and a
// retention row, GST 4,600, total 50,600.
func dropClaim(t *testing.T, s *store.Store, pkg string, lines []string, name string) string {
	t.Helper()
	ctx := context.Background()
	up := upload(name)
	up.ClaimPackage = pkg
	created, err := s.CreateInvoice(ctx, orgA, projectA, up, invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	d := func(f, v string) invoices.Decision {
		return invoices.Decision{Field: f, Value: v, Band: "green", DecidedBy: "rule"}
	}
	out := invoices.Outcome{Asked: true}
	out.Decisions = []invoices.Decision{d(invoices.FieldNumber, "4"), d(invoices.FieldDate, "2026-10-14"), d(invoices.FieldTotal, "5060000"),
		d(invoices.FieldGST, "460000"), d(invoices.FieldPeriod, "2026-10-01"), d(invoices.FieldCurrency, "AUD")}
	out.Claim = &invoices.ClaimReading{TableFound: true, PrintedGross: cp(4600000), Rows: []invoices.ClaimRow{
		{Kind: invoices.RowContract, Label: "Substructure", LineID: lines[0], MappedBy: "rule", Band: "green", ToDate: cp(6000000), Previous: cp(4000000), ThisClaim: cp(2000000)},
		{Kind: invoices.RowContract, Label: "Superstructure", LineID: lines[1], MappedBy: "rule", Band: "green", ToDate: cp(2600000), Previous: cp(0), ThisClaim: cp(2600000)},
		{Kind: invoices.RowRetention, Label: "Retention", MappedBy: "rule", Band: "green", ToDate: cp(-230000), Previous: cp(0), ThisClaim: cp(-230000)},
	}}
	if ok, err := s.CommitInvoiceRead(ctx, orgA, created.ID, out, false, invoiceNow); err != nil || !ok {
		t.Fatalf("commit %v %v", ok, err)
	}
	return created.ID
}

func claimOf(t *testing.T, v store.LedgerView, id string) store.ClaimView {
	t.Helper()
	for _, c := range v.Claims {
		if c.InvoiceID == id {
			return c
		}
	}
	t.Fatalf("claim %s not in the ledger", id)
	return store.ClaimView{}
}

func ledgerNow(t *testing.T, s *store.Store) store.LedgerView {
	t.Helper()
	v, err := s.InvoiceLedger(context.Background(), orgA, projectA, "2026-10-16")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// Done means 3: accept, cut one row with a reason, issue; Claimed to Date
// rises by the scheduled amounts; the tax invoice links by rule and the
// ledger total does not double.
func TestClaimAssessIssueAndCount(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	claimID := dropClaim(t, s, pkg, lines, "claim-4")

	v := ledgerNow(t, s)
	c := claimOf(t, v, claimID)
	if len(c.Rows) != 3 || c.Figures.Counts || c.Figures.Unassessed != 3 || lineInvoiced(v, lines[0]) != 0 {
		t.Fatalf("assessing: %+v", c.Figures)
	}
	// The builder's previous (40,000) is not SiteWise's certified (0).
	flagged := false
	for _, f := range v.Flags[claimID] {
		flagged = flagged || f.Code == "previous_differs"
	}
	if !flagged {
		t.Fatalf("a previous unlike SiteWise's is flagged: %+v", v.Flags[claimID])
	}
	if err := s.AssessClaim(ctx, orgA, projectA, userA, claimID, store.ClaimAssess{AcceptClaimed: true}); err != nil {
		t.Fatal(err)
	}
	// Cut Superstructure to 20,000 to date: the reason is needed to issue.
	cut := store.ClaimAssess{}
	cut.Rows = append(cut.Rows, struct {
		ID        string  `json:"id"`
		Assessed  *string `json:"assessed_to_date"`
		Reason    *string `json:"reason"`
		Line      *string `json:"line"`
		Variation *string `json:"variation"`
		Retention bool    `json:"retention"`
	}{ID: c.Rows[1].ID, Assessed: sp("20000")})
	if err := s.AssessClaim(ctx, orgA, projectA, userA, claimID, cut); err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssuePaymentSchedule(ctx, orgA, projectA, userA, claimID, "2026-10-16", blobs); !errors.Is(err, store.ErrInvalidClaim) {
		t.Fatalf("a cut row with no reason must block issue: %v", err)
	}
	cut.Rows[0].Reason = sp("Frame not complete on level 2")
	if err := s.AssessClaim(ctx, orgA, projectA, userA, claimID, cut); err != nil {
		t.Fatal(err)
	}
	c = claimOf(t, ledgerNow(t, s), claimID)
	// Scheduled: 60,000 + 20,000 = 80,000; retention 2,300; net 77,700;
	// GST 7,770; payable 85,470.
	f := c.Figures
	if *f.Gross != 8000000 || *f.Retention != 230000 || *f.Net != 7770000 || *f.GST != 777000 || *f.Payable != 8547000 || f.MissingReasons != 0 {
		t.Fatalf("figures: gross %d retention %d net %d gst %d payable %d", *f.Gross, *f.Retention, *f.Net, *f.GST, *f.Payable)
	}
	issued, err := s.IssuePaymentSchedule(ctx, orgA, projectA, userA, claimID, "2026-10-16", blobs)
	if err != nil {
		t.Fatal(err)
	}
	v = ledgerNow(t, s)
	if lineInvoiced(v, lines[0]) != 6000000 || lineInvoiced(v, lines[1]) != 2000000 || v.Summary.Total != 8000000 {
		t.Fatalf("claimed to date: %d %d total %d", lineInvoiced(v, lines[0]), lineInvoiced(v, lines[1]), v.Summary.Total)
	}
	ledgerTotals(t, v)
	// The remaining on the line is the contract sum to date less claimed.
	for _, l := range v.Summary.Lines {
		if l.ID == lines[0] && (l.ContractRemaining == nil || *l.ContractRemaining != 4000000) {
			t.Fatalf("remaining %v", l.ContractRemaining)
		}
	}
	// The tax invoice for 85,470 links by rule and is not counted again.
	tax, err := s.CreateInvoice(ctx, orgA, projectA, upload("tax-invoice-4"), invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitInvoiceRead(ctx, orgA, tax.ID, reading(pkg, lines[0], "INV-77", 8547000, 777000), false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v = ledgerNow(t, s)
	if v.Summary.Total != 8000000 {
		t.Fatalf("the tax invoice doubled the ledger: %d", v.Summary.Total)
	}
	for _, inv := range v.Invoices {
		if inv.ID == tax.ID && (inv.SettlesClaim != claimID || inv.SettlesBy != invoices.DecidedByRule || invoices.Counted(inv)) {
			t.Fatalf("link rule: %+v", inv)
		}
	}
	// Frozen: a write after issue is refused, here and by the trigger.
	if err := s.AssessClaim(ctx, orgA, projectA, userA, claimID, store.ClaimAssess{AcceptClaimed: true}); !errors.Is(err, store.ErrClaimIssued) {
		t.Fatalf("write after issue: %v", err)
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE progress_claim_lines SET reason='x' WHERE org_id=$1 AND invoice_id=$2`, orgA, claimID); err == nil {
		t.Fatal("the trigger let an issued claim's row change")
	}
	// The snapshot reads back with the hash it was issued with.
	report, err := s.ReadIssuedReport(ctx, orgA, issued.ReportID, issued.VersionID)
	if err != nil {
		t.Fatal(err)
	}
	if _, hash, err := reports.CanonicalSnapshot(report.Snapshot); err != nil || hash != issued.SHA256 {
		t.Fatalf("snapshot hash %s != %s (%v)", hash, issued.SHA256, err)
	}
	// Re-issue is a new version linked to the first.
	again, err := s.IssuePaymentSchedule(ctx, orgA, projectA, userA, claimID, "2026-10-17", blobs)
	if err != nil || again.Number != 2 || again.ReportID != issued.ReportID {
		t.Fatalf("re-issue: %+v %v", again, err)
	}
	var previous string
	if err := rawPool(t).QueryRow(ctx, `SELECT previous_issue_id::text FROM report_versions WHERE org_id=$1 AND id=$2`, orgA, again.VersionID).Scan(&previous); err != nil || previous != issued.VersionID {
		t.Fatalf("previous %q %v", previous, err)
	}
	// A payment schedule is never a package's report: the list leaves it out.
	list, err := s.ListReports(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range list {
		if r.Kind == reports.KindPaymentSchedule {
			t.Fatal("ListReports includes a payment schedule")
		}
	}
	// Another org cannot assess or issue.
	if err := s.AssessClaim(ctx, orgB, projectB, userB, claimID, store.ClaimAssess{AcceptClaimed: true}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign assess: %v", err)
	}
	if _, err := s.IssuePaymentSchedule(ctx, orgB, projectB, userB, claimID, "2026-10-16", blobs); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign issue: %v", err)
	}
}

// An assessment typed under a marked change has Undo; Undo after issue is
// refused.
func TestClaimAssessUndo(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	claimID := dropClaim(t, s, pkg, lines, "claim-undo")
	marked := store.WithChange(ctx, store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "claims"})
	if err := s.AssessClaim(marked, orgA, projectA, userA, claimID, store.ClaimAssess{AcceptClaimed: true}); err != nil {
		t.Fatal(err)
	}
	change, _ := store.ChangeOf(marked)
	if c := claimOf(t, ledgerNow(t, s), claimID); c.Figures.Unassessed != 0 {
		t.Fatalf("accepted: %+v", c.Figures)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err != nil {
		t.Fatal(err)
	}
	if c := claimOf(t, ledgerNow(t, s), claimID); c.Figures.Unassessed != 3 {
		t.Fatalf("undo: %+v", c.Figures)
	}
	second := store.WithChange(ctx, store.Change{ID: store.NewChangeID(), OrgID: orgA, ProjectID: projectA, Actor: userA, Via: "screen", Area: "claims"})
	if err := s.AssessClaim(second, orgA, projectA, userA, claimID, store.ClaimAssess{AcceptClaimed: true}); err != nil {
		t.Fatal(err)
	}
	change, _ = store.ChangeOf(second)
	blobs, _ := files.Open(t.TempDir(), 20<<20)
	if _, err := s.IssuePaymentSchedule(ctx, orgA, projectA, userA, claimID, "2026-10-16", blobs); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UndoChange(ctx, orgA, projectA, userA, change.ID); err == nil {
		t.Fatal("undo after issue must be refused")
	}
}

// A claim not read still gets its contract's lines, amounts unknown; a
// claim cannot be dropped on a package with no live works contract.
func TestClaimDropRules(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	up := upload("scanned-claim")
	up.ClaimPackage, up.NotRead = pkg, invoices.NotReadUnsupported
	created, err := s.CreateInvoice(ctx, orgA, projectA, up, invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	c := claimOf(t, ledgerNow(t, s), created.ID)
	if len(c.Rows) != 3 || c.Rows[0].LineID != lines[0] || c.Rows[0].ToDate != nil || c.Rows[2].Kind != invoices.RowRetention {
		t.Fatalf("unread claim rows: %+v", c.Rows)
	}
	// Re-dropping the same bytes is "Already filed".
	again, err := s.CreateInvoice(ctx, orgA, projectA, up, invoiceNow)
	if err != nil || again.Created || again.ID != created.ID {
		t.Fatalf("re-drop: %+v %v", again, err)
	}
	other := upload("not-a-contract")
	other.ClaimPackage = newUUID(t)
	if _, err := s.CreateInvoice(ctx, orgA, projectA, other, invoiceNow); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("no live contract: %v", err)
	}
}

// Due dates come from the contract's periods and the state's Act row; with
// the state on the Profile unset the dates are unknown.
func TestClaimDueDatesAndPeriods(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, lines := contractFixture(t, s, "100000.00", "50000.00")
	claimID := dropClaim(t, s, pkg, lines, "claim-due")
	c := claimOf(t, ledgerNow(t, s), claimID)
	if c.Schedule.Date != "" || c.Alert != invoices.AlertCheck {
		t.Fatalf("no state: %+v", c.Schedule)
	}
	five := 5
	if err := s.SetContractPeriods(ctx, orgA, projectA, userA, pkg, store.ContractPeriods{ScheduleDays: &five}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetContractPeriods(ctx, orgB, projectB, userB, pkg, store.ContractPeriods{ScheduleDays: &five}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("foreign periods: %v", err)
	}
	cs, _, _ := s.VariationRegister(ctx, orgA, projectA)
	if cs[0].ScheduleDays == nil || *cs[0].ScheduleDays != 5 || cs[0].ScheduleBasis != "business" {
		t.Fatalf("periods: %+v", cs[0])
	}
	raw, _ := json.Marshal(c.Schedule)
	_ = raw
	_ = strconv.Itoa
	_ = time.Now
}
