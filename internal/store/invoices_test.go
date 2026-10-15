package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

var invoiceNow = time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)

func upload(name string) store.InvoiceUpload {
	sum := sha256.Sum256([]byte(name))
	return store.InvoiceUpload{SHA256: sum[:], Size: int64(len(name)), MediaType: "application/pdf", Filename: name + ".pdf", UserID: userA}
}

func sp(s string) *string { return &s }

// ledgerFixture is a project with one consultant package, its fee line with
// a budget, and a second construction-free line for re-maps.
func ledgerFixture(t *testing.T) (*store.Store, string, string, string) {
	t.Helper()
	ctx := context.Background()
	s, _, _ := workStore(t)
	pkg, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Architect", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Structural engineer", LifecycleStatus: "planned"})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	plan := ready.Plan
	var line, line2 string
	for _, i := range plan.Items {
		switch i.PackageID {
		case pkg.ID:
			line = i.ID
		case other.ID:
			line2 = i.ID
		}
	}
	if line == "" || line2 == "" {
		t.Fatalf("fee lines missing: %+v", plan.Items)
	}
	if _, err := s.PutCostValue(ctx, orgA, projectA, line, userA, "budget", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: cv("10000.00")}); err != nil {
		t.Fatal(err)
	}
	return s, pkg.ID, line, line2
}

func reading(pkg, line string, number string, total, gst int64) invoices.Outcome {
	d := func(f, v, band, by string) invoices.Decision {
		return invoices.Decision{Field: f, Value: v, Band: band, DecidedBy: by}
	}
	out := invoices.Outcome{Asked: true}
	out.Decisions = []invoices.Decision{
		d(invoices.FieldKind, "invoice", "green", "rule"), d(invoices.FieldNumber, number, "green", "rule"),
		d(invoices.FieldDate, "2026-10-01", "green", "rule"), d(invoices.FieldTotal, strconv.FormatInt(total, 10), "green", "rule"),
		d(invoices.FieldGST, strconv.FormatInt(gst, 10), "green", "rule"), d(invoices.FieldSupplier, pkg, "green", "jev"),
		d(invoices.FieldLine, line, "green", "rule"), d(invoices.FieldPeriod, "2026-10-01", "green", "rule"), d(invoices.FieldCurrency, "AUD", "green", "rule"),
	}
	out.InvoiceCount = 1
	return out
}

func ledgerTotals(t *testing.T, v store.LedgerView) (lines int64) {
	t.Helper()
	for _, l := range v.Summary.Lines {
		lines += l.Invoiced
	}
	if lines+v.Summary.Unallocated.Invoiced != v.Summary.Total {
		t.Fatalf("invariant broken: lines %d + unallocated %d != total %d", lines, v.Summary.Unallocated.Invoiced, v.Summary.Total)
	}
	return lines
}

func lineInvoiced(v store.LedgerView, id string) int64 {
	for _, l := range v.Summary.Lines {
		if l.ID == id {
			return l.Invoiced
		}
	}
	return -1
}

func TestInvoiceLedgerLifecycle(t *testing.T) {
	ctx := context.Background()
	s, pkg, line, line2 := ledgerFixture(t)
	first, err := s.CreateInvoice(ctx, orgA, projectA, upload("inv-1"), invoiceNow)
	if err != nil || !first.Created || first.Status != invoices.StatusReading {
		t.Fatalf("%+v %v", first, err)
	}
	again, err := s.CreateInvoice(ctx, orgA, projectA, upload("inv-1"), invoiceNow)
	if err != nil || again.Created || again.ID != first.ID {
		t.Fatalf("re-drop made a new row: %+v %v", again, err)
	}
	if ok, err := s.CommitInvoiceRead(ctx, orgA, first.ID, reading(pkg, line, "INV-1", 110000, 10000), false, invoiceNow); !ok || err != nil {
		t.Fatal(ok, err)
	}
	// A second commit of the same read is dropped: the invoice is no longer waiting.
	if ok, _ := s.CommitInvoiceRead(ctx, orgA, first.ID, reading(pkg, line, "OTHER", 1, 0), false, invoiceNow); ok {
		t.Fatal("late duplicate read applied")
	}
	v, err := s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Invoices) != 1 || lineInvoiced(v, line) != 100000 || v.Status[first.ID] != invoices.StatusFiled {
		t.Fatalf("ledger %+v status %v", v.Summary, v.Status)
	}
	ledgerTotals(t, v)

	// The user's number is final: a later re-check cannot replace it.
	if err := s.EditInvoice(ctx, orgA, projectA, first.ID, userA, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldNumber: sp("INV-1A")}}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.CommitInvoiceRead(ctx, orgA, first.ID, reading(pkg, line, "INV-9", 110000, 10000), true, invoiceNow); !ok || err != nil {
		t.Fatal(ok, err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if v.Invoices[0].Number != "INV-1A" || v.Invoices[0].Decisions[invoices.FieldNumber].DecidedBy != "user" {
		t.Fatalf("user correction overwritten: %+v", v.Invoices[0].Decisions[invoices.FieldNumber])
	}

	// Re-map to no line: the spend moves to Unallocated in the same read.
	if err := s.EditInvoice(ctx, orgA, projectA, first.ID, userA, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldLine: sp("")}}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if lineInvoiced(v, line) != 0 || v.Summary.Unallocated.Invoiced != 100000 {
		t.Fatalf("re-map %+v", v.Summary)
	}
	// Split across two lines; a split that does not add up is kept and flagged.
	if err := s.SplitInvoice(ctx, orgA, projectA, first.ID, userA, []store.ShareInput{{LineID: line, Amount: "600.00"}, {LineID: line2, Amount: "300.00"}}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if lineInvoiced(v, line) != 60000 || lineInvoiced(v, line2) != 30000 || v.Summary.Unallocated.Invoiced != 10000 {
		t.Fatalf("split %+v", v.Summary)
	}
	if !hasFlag(v.Flags[first.ID], "split_total") || v.Status[first.ID] != invoices.StatusNeedsReview {
		t.Fatalf("flags %v", v.Flags[first.ID])
	}
	if err := s.EditInvoice(ctx, orgA, projectA, first.ID, userA, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldLine: sp(line)}}, invoiceNow); !errors.Is(err, store.ErrInvoiceSplit) {
		t.Fatal("line edit on a split", err)
	}
	ledgerTotals(t, v)

	// A revised invoice with the same number: flagged, then it replaces the old one.
	second, _ := s.CreateInvoice(ctx, orgA, projectA, upload("inv-1-revised"), invoiceNow)
	if _, err := s.CommitInvoiceRead(ctx, orgA, second.ID, reading(pkg, line, "INV-1A", 220000, 20000), false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if !hasFlag(v.Flags[second.ID], "duplicate") {
		t.Fatalf("duplicate not flagged: %v", v.Flags[second.ID])
	}
	if err := s.EditInvoice(ctx, orgA, projectA, second.ID, userA, store.InvoiceEdit{Action: "replaces", Target: first.ID}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if v.Status[first.ID] != invoices.StatusSuperseded || v.Summary.Total != 200000 || hasFlag(v.Flags[second.ID], "duplicate") {
		t.Fatalf("supersede %+v %v", v.Summary, v.Status)
	}
	// A credit note reduces the line; void takes an invoice out of totals.
	credit, _ := s.CreateInvoice(ctx, orgA, projectA, upload("credit"), invoiceNow)
	out := reading(pkg, line, "CN-1", 11000, 1000)
	out.Decisions[0].Value = "credit_note"
	if _, err := s.CommitInvoiceRead(ctx, orgA, credit.ID, out, false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if lineInvoiced(v, line) != 200000-10000 {
		t.Fatalf("credit note %d", lineInvoiced(v, line))
	}
	if err := s.EditInvoice(ctx, orgA, projectA, credit.ID, userA, store.InvoiceEdit{Action: "void"}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	v, _ = s.InvoiceLedger(ctx, orgA, projectA, "2026-10-16")
	if lineInvoiced(v, line) != 200000 || v.Status[credit.ID] != invoices.StatusVoid {
		t.Fatal("void still counts")
	}
	ledgerTotals(t, v)
}

// A grey read is re-checked in the background, once per due time, and a
// bounded number of times.
func TestGreyReadSchedulesABackgroundRecheck(t *testing.T) {
	ctx := context.Background()
	s, pkg, line, _ := ledgerFixture(t)
	inv, _ := s.CreateInvoice(ctx, orgA, projectA, upload("grey"), invoiceNow)
	out := reading(pkg, line, "INV-G", 110000, 10000)
	out.Grey = true
	out.Decisions[5] = invoices.Decision{Field: invoices.FieldSupplier, Band: "grey", DecidedBy: "jev"}
	if _, err := s.CommitInvoiceRead(ctx, orgA, inv.ID, out, false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if due, _ := s.DueInvoiceRechecks(ctx, invoiceNow.Add(10*time.Second)); len(due) != 0 {
		t.Fatal("re-check due before its time")
	}
	due, err := s.DueInvoiceRechecks(ctx, invoiceNow.Add(31*time.Second))
	if err != nil || len(due) != 1 || due[0].InvoiceID != inv.ID {
		t.Fatalf("due %+v %v", due, err)
	}
	if ok, _ := s.ClaimInvoiceRecheck(ctx, due[0], invoiceNow.Add(31*time.Second)); !ok {
		t.Fatal("claim failed")
	}
	if ok, _ := s.ClaimInvoiceRecheck(ctx, due[0], invoiceNow.Add(31*time.Second)); ok {
		t.Fatal("claimed twice")
	}
	// The re-check keeps every answered field and fills the grey one.
	in, err := s.InvoiceForReading(ctx, orgA, inv.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := in.Keep[invoices.FieldSupplier]; kept || in.Keep[invoices.FieldNumber].Value != "INV-G" {
		t.Fatalf("keep %+v", in.Keep)
	}
	again := reading(pkg, line, "INV-OTHER", 1, 0)
	if _, err := s.CommitInvoiceRead(ctx, orgA, inv.ID, again, true, invoiceNow.Add(32*time.Second)); err != nil {
		t.Fatal(err)
	}
	v, _ := s.InvoiceLedger(ctx, orgA, projectA, "")
	if v.Invoices[0].PackageID != pkg {
		t.Fatalf("grey supplier not filled: %+v", v.Invoices[0])
	}
}

func hasFlag(flags []invoices.Flag, code string) bool {
	for _, f := range flags {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestConfirmedSupplierTeachesTheABNRule(t *testing.T) {
	ctx := context.Background()
	s, pkg, line, _ := ledgerFixture(t)
	inv, _ := s.CreateInvoice(ctx, orgA, projectA, upload("abn"), invoiceNow)
	out := reading("", "", "INV-7", 110000, 10000)
	out.Decisions = append(out.Decisions[:5], invoices.Decision{Field: invoices.FieldABN, Value: "54165354218", Band: "amber", DecidedBy: "jev"}, out.Decisions[7], out.Decisions[8])
	if _, err := s.CommitInvoiceRead(ctx, orgA, inv.ID, out, false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if err := s.EditInvoice(ctx, orgA, projectA, inv.ID, userA, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldSupplier: sp(pkg), invoices.FieldLine: sp(line)}}, invoiceNow); err != nil {
		t.Fatal(err)
	}
	next, err := s.CreateInvoice(ctx, orgA, projectA, upload("abn-2"), invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	in, err := s.InvoiceForReading(ctx, orgA, next.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range in.Context.Packages {
		if p.ID == pkg && p.ABN == "54165354218" {
			found = true
		}
	}
	if !found || len(in.Context.History[pkg]) != 1 || in.Context.History[pkg][0] != line {
		t.Fatalf("context %+v", in.Context)
	}
}

func TestInvoiceOrgIsolation(t *testing.T) {
	ctx := context.Background()
	s, pkg, line, _ := ledgerFixture(t)
	inv, _ := s.CreateInvoice(ctx, orgA, projectA, upload("private"), invoiceNow)
	if _, err := s.CommitInvoiceRead(ctx, orgA, inv.ID, reading(pkg, line, "INV-P", 110000, 10000), false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateInvoice(ctx, orgB, projectA, upload("x"), invoiceNow); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("upload into another org's project", err)
	}
	if _, err := s.InvoiceLedger(ctx, orgB, projectA, ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("ledger read across orgs", err)
	}
	if err := s.EditInvoice(ctx, orgB, projectB, inv.ID, userB, store.InvoiceEdit{Action: "void"}, invoiceNow); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("edit across orgs", err)
	}
	if err := s.SplitInvoice(ctx, orgB, projectB, inv.ID, userB, []store.ShareInput{{LineID: ""}}, invoiceNow); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("split across orgs", err)
	}
	if _, _, err := s.InvoiceFile(ctx, orgB, projectB, inv.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("file across orgs", err)
	}
	if _, err := s.InvoiceForReading(ctx, orgB, inv.ID, false); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("read input across orgs", err)
	}
	// Org B's own invoice cannot name org A's package (supplier) or cost line.
	own, err := s.CreateInvoice(ctx, orgB, projectB, upload("b-own"), invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EditInvoice(ctx, orgB, projectB, own.ID, userB, store.InvoiceEdit{Fields: map[string]*string{invoices.FieldSupplier: sp(pkg)}}, invoiceNow); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign supplier accepted", err)
	}
	if err := s.SplitInvoice(ctx, orgB, projectB, own.ID, userB, []store.ShareInput{{LineID: line}}, invoiceNow); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("foreign line accepted", err)
	}
	// The schema refuses a cross-org allocation or decision outright.
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, `INSERT INTO invoice_allocations (org_id,id,project_id,invoice_id,decided_by,band) VALUES ($1,gen_random_uuid(),$2,$3,'user','green')`, orgB, projectB, inv.ID); err == nil {
		t.Fatal("allocation in org B for org A's invoice")
	}
	if _, err := conn.Exec(ctx, `INSERT INTO invoice_allocations (org_id,id,project_id,invoice_id,cost_item_id,decided_by,band) VALUES ($1,gen_random_uuid(),$2,$3,$4,'user','green')`, orgB, projectB, own.ID, line); err == nil {
		t.Fatal("allocation in org B to org A's cost line")
	}
	if _, err := conn.Exec(ctx, `UPDATE invoices SET package_id=$3 WHERE org_id=$1 AND id=$2`, orgB, own.ID, pkg); err == nil {
		t.Fatal("org B invoice linked to org A's package")
	}
	if _, err := conn.Exec(ctx, `INSERT INTO invoice_decisions (org_id,project_id,invoice_id,field,band,decided_by) VALUES ($1,$2,$3,'number','green','user')`, orgB, projectB, inv.ID); err == nil {
		t.Fatal("decision in org B for org A's invoice")
	}
	v, err := s.InvoiceLedger(ctx, orgB, projectB, "")
	if err != nil || len(v.Invoices) != 1 || v.Invoices[0].ID != own.ID {
		t.Fatalf("org B sees %+v %v", v.Invoices, err)
	}
}

func TestDeletesKeepOrRemoveInvoiceBytesCorrectly(t *testing.T) {
	ctx := context.Background()
	s, pkg, line, _ := ledgerFixture(t)
	up := upload("shared")
	inv, err := s.CreateInvoice(ctx, orgA, projectA, up, invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	// The same bytes dropped in the register become a document of their own.
	filed, err := s.CommitIntake(ctx, orgA, store.CommitIntake{ProjectID: projectA, SHA256: up.SHA256, ByteSize: up.Size, MediaType: up.MediaType, Filename: up.Filename})
	if err != nil || !filed.Created {
		t.Fatalf("register drop of invoice bytes: %+v %v", filed, err)
	}
	if _, err := s.DeleteDocuments(ctx, orgA, projectA, []string{filed.DocumentID}, userA); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.InvoiceFile(ctx, orgA, projectA, inv.ID); err != nil {
		t.Fatal("document delete took the invoice's bytes", err)
	}
	if _, err := s.CommitInvoiceRead(ctx, orgA, inv.ID, reading(pkg, line, "INV-D", 110000, 10000), false, invoiceNow); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteProject(ctx, orgA, projectA); err != nil {
		t.Fatal("project delete with invoices", err)
	}
	if _, err := s.InvoiceLedger(ctx, orgA, projectA, ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatal("ledger survived project delete", err)
	}
}
