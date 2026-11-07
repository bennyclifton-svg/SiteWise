package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sitewise/internal/costs"
	"sitewise/internal/files"
	"sitewise/internal/invoices"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// groupFixture makes consultant rows for the named disciplines and prepares
// the cost plan, so each has its draft fee line.
func groupFixture(t *testing.T, disciplines ...string) (*store.Store, []procurement.Package) {
	t.Helper()
	ctx := context.Background()
	s, _, _ := workStore(t)
	labels := map[string]string{"consultant.structural": "Structural engineer", "consultant.civil": "Civil engineer", "consultant.hydraulic": "Hydraulic engineer", "consultant.electrical": "Electrical engineer"}
	out := []procurement.Package{}
	for _, d := range disciplines {
		row, ok := procurement.DisciplinePackage(d, labels[d])
		if !ok {
			t.Fatal("unknown discipline", d)
		}
		p, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, row)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return s, out
}

func feeLine(plan costs.Plan, pkg string) *costs.Item {
	for i := range plan.Items {
		if plan.Items[i].LineKind == "fee" && plan.Items[i].PackageID == pkg {
			return &plan.Items[i]
		}
	}
	return nil
}

func feeLines(plan costs.Plan) int {
	n := 0
	for _, i := range plan.Items {
		if i.LineKind == "fee" {
			n++
		}
	}
	return n
}

func packageByID(t *testing.T, s *store.Store, id string) procurement.Package {
	t.Helper()
	items, err := s.ReadPackages(context.Background(), orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range items {
		if p.ID == id {
			return p
		}
	}
	return procurement.Package{}
}

func setBudget(t *testing.T, s *store.Store, plan costs.Plan, line, amount string) costs.Plan {
	t.Helper()
	m := costs.Money(amount)
	plan, err := s.PutCostValue(context.Background(), orgA, projectA, line, userA, "budget", store.CostValueInput{PlanVersionID: plan.ID, PlanVersion: plan.Version, Value: costs.Value{ValueState: "known", Amount: &m, Origin: "user", Meaning: "allowance"}})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestCombineNestsFeeLinesMovesProposalsAndUncombineReverses(t *testing.T) {
	ctx := context.Background()
	s, pkgs := groupFixture(t, "consultant.structural", "consultant.civil")
	structural, civil := pkgs[0], pkgs[1]
	ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	plan := ready.Plan
	sLine, cLine := feeLine(plan, structural.ID), feeLine(plan, civil.ID)
	if sLine == nil || cLine == nil {
		t.Fatal("fee lines missing")
	}
	plan = setBudget(t, s, plan, sLine.ID, "40000.00")
	plan = setBudget(t, s, plan, cLine.ID, "25000.00")
	before := feeLines(plan)
	sub, err := s.OpenSubmission(ctx, orgA, projectA, docA, structural.ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{structural.ID, civil.ID}, "")
	t.Logf("combine: %s (p90 budget 250ms)", time.Since(start))
	if err != nil {
		t.Fatal(err)
	}
	if !group.IsGroup() || group.Kind != "services" || group.DisciplineID != "" || group.Title != "Structural engineer and Civil engineer" {
		t.Fatalf("group %+v", group)
	}
	for _, m := range pkgs {
		if got := packageByID(t, s, m.ID); got.GroupID != group.ID {
			t.Fatalf("member %s not in group: %+v", m.Title, got)
		}
	}
	plan, err = s.ReadCostPlan(ctx, orgA, projectA, "")
	if err != nil {
		t.Fatal(err)
	}
	gLine := feeLine(plan, group.ID)
	if gLine == nil || gLine.Posting || gLine.ParentItemID != "" || gLine.SourceKey != "fee:"+group.ID {
		t.Fatalf("group line %+v", gLine)
	}
	for _, m := range []string{structural.ID, civil.ID} {
		if l := feeLine(plan, m); l.ParentItemID != gLine.ID {
			t.Fatalf("member line not under the group: %+v", l)
		}
	}
	if b := feeLine(plan, structural.ID).Values["budget"].Amount; b == nil || *b != "40000.00" {
		t.Fatal("member budget lost")
	}
	totals, err := costs.Summarize(plan, "overall")
	if err != nil {
		t.Fatal(err)
	}
	if r := totals.Rollups[gLine.ID]; r.Values["budget"].Amount == nil || *r.Values["budget"].Amount != "65000.00" {
		t.Fatalf("group roll-up %+v", r)
	}
	// Other lines have no amount yet, so the known part is what counts.
	if b := totals.Overall["budget"].KnownSubtotal; b != "65000.00" {
		t.Fatalf("overall budget counted twice or lost: %v", b)
	}
	if pkgOf(t, sub) != group.ID {
		t.Fatal("proposal not moved to the group")
	}
	// Prepare keeps the nesting and adds no line for a member.
	again, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	if feeLines(again.Plan) != before+1 || feeLine(again.Plan, civil.ID).ParentItemID != gLine.ID {
		t.Fatalf("prepare changed the group: %d fee lines, want %d", feeLines(again.Plan), before+1)
	}

	// Rename is the package title patch.
	title := "Structural and civil engineering"
	renamed, err := s.PatchPackage(ctx, orgA, projectA, group.ID, userA, procurement.PackagePatch{Version: group.Version, Title: &title})
	if err != nil || renamed.Title != title {
		t.Fatalf("rename %+v %v", renamed, err)
	}

	start = time.Now()
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, group.ID); err != nil {
		t.Fatal(err)
	}
	t.Logf("uncombine: %s (p90 budget 250ms)", time.Since(start))
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	if feeLine(plan, group.ID) != nil || feeLines(plan) != before {
		t.Fatal("group line left behind")
	}
	for _, m := range []string{structural.ID, civil.ID} {
		if l := feeLine(plan, m); l == nil || l.ParentItemID != "" {
			t.Fatalf("member line not back at the top level: %+v", l)
		}
		if got := packageByID(t, s, m); got.GroupID != "" {
			t.Fatal("member still grouped")
		}
	}
	if b := feeLine(plan, civil.ID).Values["budget"].Amount; b == nil || *b != "25000.00" {
		t.Fatal("member budget lost on uncombine")
	}
	if pkgOf(t, sub) != structural.ID {
		t.Fatal("proposal not returned to its member")
	}
	if packageByID(t, s, group.ID).ID != "" {
		t.Fatal("group row still live")
	}
	// The members can be combined again.
	if _, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{structural.ID, civil.ID}, "Again"); err != nil {
		t.Fatal(err)
	}
}

func pkgOf(t *testing.T, submission string) string {
	t.Helper()
	var pkg string
	if err := rawPool(t).QueryRow(context.Background(), `SELECT COALESCE(package_id::text,'') FROM submissions WHERE org_id=$1 AND id=$2`, orgA, submission).Scan(&pkg); err != nil {
		t.Fatal(err)
	}
	return pkg
}

func TestCombineRefusals(t *testing.T) {
	ctx := context.Background()
	s, pkgs := groupFixture(t, "consultant.structural", "consultant.civil", "consultant.hydraulic")
	structural, civil, hydraulic := pkgs[0], pkgs[1], pkgs[2]
	works, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, mustRow(t, "trade.electrical", "Electrical"))
	if err != nil {
		t.Fatal(err)
	}
	const otherProject = "33333333-3333-4333-8333-444444444444"
	if err := s.CreateProject(ctx, orgA, otherProject, "Other project"); err != nil {
		t.Fatal(err)
	}
	foreign, err := s.CreateDisciplinePackage(ctx, orgA, otherProject, userA, mustRow(t, "consultant.civil", "Civil engineer"))
	if err != nil {
		t.Fatal(err)
	}
	combine := func(ids ...string) error {
		_, err := s.CombinePackages(ctx, orgA, projectA, userA, ids, "")
		return err
	}
	for name, c := range map[string]struct {
		ids  []string
		want error
	}{
		"fewer than two":  {[]string{structural.ID}, store.ErrInvalidPackage},
		"the same id":     {[]string{structural.ID, structural.ID}, store.ErrInvalidPackage},
		"works package":   {[]string{structural.ID, works.ID}, store.ErrInvalidPackage},
		"another project": {[]string{structural.ID, foreign.ID}, store.ErrNotFound},
		"not a uuid":      {[]string{structural.ID, "x"}, store.ErrNotFound},
		"missing package": {[]string{structural.ID, "44444444-4444-4444-8444-444444444444"}, store.ErrNotFound},
	} {
		if err := combine(c.ids...); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v, want %v", name, err, c.want)
		}
	}
	if _, err := s.CombinePackages(ctx, orgB, projectA, userB, []string{structural.ID, civil.ID}, ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong org: %v", err)
	}
	if _, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{structural.ID, civil.ID}, string(make([]byte, 201))); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("overlong title: %v", err)
	}
	// The schema refuses a member pointing at a row that is not a group.
	if _, err := rawPool(t).Exec(ctx, `UPDATE packages SET group_id=$3 WHERE org_id=$1 AND id=$2`, orgA, hydraulic.ID, civil.ID); err == nil {
		t.Fatal("a member joined a discipline row")
	}
	// A live award on a member refuses, inside the transaction.
	award := awardRow(t, s, hydraulic.ID)
	if err := combine(civil.ID, hydraulic.ID); !errors.Is(err, store.ErrPackageAwarded) {
		t.Fatalf("awarded member: %v", err)
	}
	if _, err := rawPool(t).Exec(ctx, `UPDATE awards SET undone_at=now() WHERE org_id=$1 AND id=$2`, orgA, award); err != nil {
		t.Fatal(err)
	}
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{structural.ID, civil.ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	// A duplicate request finds its members already grouped.
	if err := combine(structural.ID, civil.ID); !errors.Is(err, store.ErrPackageGrouped) {
		t.Fatalf("duplicate combine: %v", err)
	}
	if err := combine(group.ID, hydraulic.ID); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("group as a member: %v", err)
	}
	if err := s.UncombinePackages(ctx, orgB, projectA, userB, group.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("wrong-org uncombine: %v", err)
	}
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, structural.ID); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("uncombine a member: %v", err)
	}
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, hydraulic.ID); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("remove an ungrouped row: %v", err)
	}
	// A member or group is not retired while combined, and a member's kind
	// and a group's discipline do not change.
	retire := true
	m := packageByID(t, s, structural.ID)
	if _, err := s.PatchPackage(ctx, orgA, projectA, m.ID, userA, procurement.PackagePatch{Version: m.Version, Retired: &retire}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("retire a member: %v", err)
	}
	if _, err := s.PatchPackage(ctx, orgA, projectA, group.ID, userA, procurement.PackagePatch{Version: group.Version, Retired: &retire}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("retire a group: %v", err)
	}
	d := "consultant.civil"
	if _, err := s.PatchPackage(ctx, orgA, projectA, group.ID, userA, procurement.PackagePatch{Version: group.Version, DisciplineID: &d}); !errors.Is(err, store.ErrInvalidPackage) {
		t.Fatalf("give a group a discipline: %v", err)
	}
	// A live award on the group refuses uncombine and remove.
	awardRow(t, s, group.ID)
	if err := s.UncombinePackages(ctx, orgA, projectA, userA, group.ID); !errors.Is(err, store.ErrPackageAwarded) {
		t.Fatalf("awarded uncombine: %v", err)
	}
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, civil.ID); !errors.Is(err, store.ErrPackageAwarded) {
		t.Fatalf("awarded remove: %v", err)
	}
}

func mustRow(t *testing.T, id, label string) procurement.Package {
	t.Helper()
	p, ok := procurement.DisciplinePackage(id, label)
	if !ok {
		t.Fatal(id)
	}
	return p
}

// awardRow records a live award on pkg directly: the refusals only ask
// whether one exists.
func awardRow(t *testing.T, s *store.Store, pkg string) string {
	t.Helper()
	ctx := context.Background()
	sub := newUUID(t)
	if _, err := rawPool(t).Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,filename,status,received_on) VALUES($1,$2,$3,$4,'fee.md','received','2026-11-01')`, orgA, sub, projectA, pkg); err != nil {
		t.Fatal(err)
	}
	ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	line := ready.Plan.Items[0].ID
	id := newUUID(t)
	if _, err := rawPool(t).Exec(ctx, `INSERT INTO awards(org_id,id,project_id,package_id,submission_id,parent_item_id,parent_created,created_by) VALUES($1,$2,$3,$4,$5,$6,false,$7)`, orgA, id, projectA, pkg, sub, line, userA); err != nil {
		t.Fatal(err)
	}
	return id
}

func newUUID(t *testing.T) string {
	t.Helper()
	var id string
	if err := rawPool(t).QueryRow(context.Background(), `SELECT gen_random_uuid()::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestRemoveFromGroupAndLastMemberUncombines(t *testing.T) {
	ctx := context.Background()
	s, pkgs := groupFixture(t, "consultant.structural", "consultant.civil", "consultant.hydraulic")
	ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	plan := setBudget(t, s, ready.Plan, feeLine(ready.Plan, pkgs[2].ID).ID, "9000.00")
	_ = plan
	sub, err := s.OpenSubmission(ctx, orgA, projectA, docA, pkgs[2].ID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{pkgs[0].ID, pkgs[1].ID, pkgs[2].ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	if group.Title != "Structural engineer, Civil engineer and Hydraulic engineer" {
		t.Fatalf("default title %q", group.Title)
	}
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, pkgs[2].ID); err != nil {
		t.Fatal(err)
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	gLine := feeLine(plan, group.ID)
	if gLine == nil || feeLine(plan, pkgs[0].ID).ParentItemID != gLine.ID {
		t.Fatal("group lost its remaining members")
	}
	if l := feeLine(plan, pkgs[2].ID); l.ParentItemID != "" || l.Values["budget"].Amount == nil || *l.Values["budget"].Amount != "9000.00" {
		t.Fatalf("removed member line %+v", l)
	}
	if pkgOf(t, sub) != pkgs[2].ID || packageByID(t, s, pkgs[2].ID).GroupID != "" {
		t.Fatal("removed member kept in the group")
	}
	// The default name follows the members; a name the PM chose would stay.
	if g := packageByID(t, s, group.ID); g.Title != "Structural engineer and Civil engineer" {
		t.Fatalf("title after remove %q", g.Title)
	}
	// Removing one of the last two leaves no group.
	if err := s.RemoveFromGroup(ctx, orgA, projectA, userA, pkgs[0].ID); err != nil {
		t.Fatal(err)
	}
	if packageByID(t, s, group.ID).ID != "" || packageByID(t, s, pkgs[1].ID).GroupID != "" {
		t.Fatal("a one-member group remains")
	}
	plan, _ = s.ReadCostPlan(ctx, orgA, projectA, "")
	if feeLine(plan, group.ID) != nil || feeLine(plan, pkgs[1].ID).ParentItemID != "" {
		t.Fatal("group line left behind")
	}
}

// A frozen baseline keeps the lines as they were when it was frozen; only
// the draft moves.
func TestCombineLeavesTheBaselineAlone(t *testing.T) {
	ctx := context.Background()
	s, pkgs := groupFixture(t, "consultant.structural", "consultant.civil")
	ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	frozenID := ready.Plan.ID
	if _, err := s.BaselineCostPlan(ctx, orgA, projectA, userA, store.CostWrite{PlanVersionID: ready.Plan.ID, Version: ready.Plan.Version}); err != nil {
		t.Fatal(err)
	}
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{pkgs[0].ID, pkgs[1].ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := s.ReadCostPlan(ctx, orgA, projectA, frozenID)
	if err != nil {
		t.Fatal(err)
	}
	if feeLine(frozen, group.ID) != nil || feeLine(frozen, pkgs[0].ID).ParentItemID != "" {
		t.Fatal("baseline changed")
	}
	draft, _ := s.ReadCostPlan(ctx, orgA, projectA, "")
	if g := feeLine(draft, group.ID); g == nil || feeLine(draft, pkgs[0].ID).ParentItemID != g.ID {
		t.Fatal("draft not nested")
	}
}

// Combined before any cost plan: prepare builds the nesting itself.
func TestPrepareNestsAGroupCombinedBeforeTheCostPlan(t *testing.T) {
	ctx := context.Background()
	s, pkgs := groupFixture(t, "consultant.structural", "consultant.civil")
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{pkgs[0].ID, pkgs[1].ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	ready, err := s.PrepareCostPlan(ctx, orgA, projectA, userA)
	if err != nil {
		t.Fatal(err)
	}
	g := feeLine(ready.Plan, group.ID)
	if g == nil || g.Posting {
		t.Fatalf("group line %+v", g)
	}
	for _, m := range pkgs {
		if l := feeLine(ready.Plan, m.ID); l == nil || l.ParentItemID != g.ID {
			t.Fatalf("member line %+v", l)
		}
	}
}

func TestGroupRulesForDisciplinesRFPsAwardsAndInvoices(t *testing.T) {
	ctx := context.Background()
	s, pkgs := groupFixture(t, "consultant.structural", "consultant.civil")
	structural := pkgs[0]
	report := mustReport(t, s, structural.ID)
	pool := rawPool(t)
	if _, err := pool.Exec(ctx, `INSERT INTO firms(org_id,id,name,name_key,abn,status) VALUES($1,'77777777-7777-4777-8777-777777777781','Karrak Consulting Engineers Pty Ltd','karrakengineers','77650369205','draft')`, orgA); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO submissions(org_id,id,project_id,package_id,firm_id,filename,status,received_on) VALUES($1,'77777777-7777-4777-8777-777777777782',$2,$3,'77777777-7777-4777-8777-777777777781','fee.md','received','2026-11-01')`, orgA, projectA, structural.ID); err != nil {
		t.Fatal(err)
	}
	group, err := s.CombinePackages(ctx, orgA, projectA, userA, []string{pkgs[0].ID, pkgs[1].ID}, "")
	if err != nil {
		t.Fatal(err)
	}
	// A member still holds its discipline; the group holds none.
	if _, err := s.CreateDisciplinePackage(ctx, orgA, projectA, userA, mustRow(t, "consultant.civil", "Civil engineer")); !errors.Is(err, store.ErrDuplicatePackage) {
		t.Fatalf("member discipline offered again: %v", err)
	}
	o, err := s.ReadPackageProposals(ctx, orgA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	for _, sug := range o.Suggestions {
		if sug.Title == structural.Title {
			t.Fatal("member re-suggested")
		}
	}
	// A member's own RFP is not issued while it is in a group, and the group
	// has no RFP yet.
	blobs, err := files.Open(t.TempDir(), 20<<20)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.IssueReport(ctx, orgA, report, userA, "test", store.IssueOptions{Version: 1, ReportingDate: "2026-11-07"}, blobs); !errors.Is(err, store.ErrReportPackageGrouped) {
		t.Fatalf("member RFP issue: %v", err)
	}
	if _, _, _, err := s.ProcurementCreate(ctx, orgA, projectA, userA, group.ID, "", ""); !errors.Is(err, store.ErrInvalidReport) {
		t.Fatalf("group RFP: %v", err)
	}
	// A member is not awarded on its own while grouped.
	if _, err := s.Award(ctx, orgA, projectA, structural.ID, userA, store.AwardInput{SubmissionID: "77777777-7777-4777-8777-777777777782"}); !errors.Is(err, store.ErrPackageGrouped) {
		t.Fatalf("member award: %v", err)
	}
	// An invoice from the firm finds the group as its supplier.
	inv, err := s.CreateInvoice(ctx, orgA, projectA, upload("group"), invoiceNow)
	if err != nil {
		t.Fatal(err)
	}
	in, err := s.InvoiceForReading(ctx, orgA, inv.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	ev := invoices.Evidence{ABNs: []invoices.Candidate{{Value: "77650369205", Labels: []string{"ABN"}, Labeled: true, Count: 1}}}
	if got := invoices.NewDraft(ev, in.Context, nil).Finish(invoiceNow).PackageID; got != group.ID {
		t.Fatalf("supplier %q, want the group %q", got, group.ID)
	}
}

func mustReport(t *testing.T, s *store.Store, pkg string) string {
	t.Helper()
	_, r, _, err := s.ProcurementCreate(context.Background(), orgA, projectA, userA, pkg, "", "")
	if err != nil {
		t.Fatal(err)
	}
	return r.ID
}
