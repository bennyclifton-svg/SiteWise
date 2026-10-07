package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sitewise/internal/costs"
	"sitewise/internal/knowledge"
	"sitewise/internal/procurement"
	"sitewise/internal/store"
	"sitewise/internal/works"
	"sort"
	"sync"
	"testing"
	"time"
)

func cm(s string) *costs.Money { m := costs.Money(s); return &m }
func cv(s string) costs.Value {
	return costs.Value{Amount: cm(s), ValueState: "known", Origin: "user", Meaning: "allowance"}
}
func ci(label string) store.CostItemInput {
	return store.CostItemInput{Content: costs.Content{Label: label, LineKind: "project_wide", Category: "contingency", Posting: true, Origin: "user", Meaning: "allowance"}, Values: map[string]costs.Value{"budget": cv("100.00")}}
}
func cw(p costs.Plan) store.CostWrite {
	return store.CostWrite{PlanVersionID: p.ID, Version: p.Version}
}
func TestCostLifecycleBaselineSubdivisionAndIsolation(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	in := ci("Allowance")
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	id := p.Items[0].ID
	for _, org := range []string{orgB} {
		if _, e = s.ReadCostPlan(ctx, org, projectA, ""); !errors.Is(e, store.ErrNotFound) {
			t.Fatal("foreign read", e)
		}
	}
	in.CostWrite = cw(p)
	if _, e = s.CreateCostItem(ctx, orgA, projectA, userB, in); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("foreign actor", e)
	}
	baseline := p.ID
	p, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
	if e != nil {
		t.Fatal(e)
	}
	if p.ID == baseline || p.Items[0].ID != id {
		t.Fatal("identity not retained", p)
	}
	frozen, e := s.ReadCostPlan(ctx, orgA, projectA, baseline)
	if e != nil || frozen.Status != "baseline" {
		t.Fatal(frozen, e)
	}
	in.CostWrite = cw(frozen)
	if _, e = s.CreateCostItem(ctx, orgA, projectA, userA, in); !errors.Is(e, store.ErrFrozenCostPlan) {
		t.Fatal("frozen write", e)
	}
	pool := rawPool(t)
	if _, e = pool.Exec(ctx, `UPDATE cost_values SET amount=0 WHERE org_id=$1::uuid AND plan_version_id=$2::uuid`, orgA, baseline); e == nil {
		t.Fatal("database allowed frozen value edit")
	}
	child := ci("Detailed allowance")
	child.Values["budget"] = cv("40.00")
	p, e = s.SubdivideCostItem(ctx, orgA, projectA, id, userA, store.CostSubdivision{CostWrite: cw(p), Children: []store.CostItemInput{child}, Residual: "explicit"})
	if e != nil {
		t.Fatal(e)
	}
	total, e := s.CostTotals(ctx, orgA, projectA, "", "overall")
	if e != nil || *total.Overall["budget"].Amount != "100.00" || total.Overall["budget"].Lines != 2 {
		t.Fatal(total, e)
	}
	frozen, e = s.ReadCostPlan(ctx, orgA, projectA, baseline)
	if e != nil || len(frozen.Items) != 1 || *frozen.Items[0].Values["budget"].Amount != "100.00" {
		t.Fatal("baseline changed", frozen, e)
	}
	if _, e = s.PutCostValue(ctx, orgA, projectA, id, userA, "budget", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: cv("1.00")}); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("group money", e)
	}
	if e = s.DeleteOrg(ctx, orgA); e != nil {
		t.Fatal("tenant cleanup", e)
	}
}
func TestCostNullConcurrentAndInvalidAmounts(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	in := ci("Allowance")
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	id := p.Items[0].ID
	unknown := costs.Value{ValueState: "unknown", Origin: "user", Meaning: "forecast"}
	p, e = s.PutCostValue(ctx, orgA, projectA, id, userA, "estimate", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: unknown})
	if e != nil {
		t.Fatal(e)
	}
	if p.Items[0].Values["estimate"].Amount != nil {
		t.Fatal("unknown became zero")
	}
	if _, e = s.PutCostValue(ctx, orgA, projectA, id, userA, "claimed_to_date", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Value: cv("1.00")}); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("missing date", e)
	}
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := ci("Concurrent")
			c.CostWrite = cw(p)
			_, e := s.CreateCostItem(ctx, orgA, projectA, userA, c)
			out <- e
		}()
	}
	wg.Wait()
	close(out)
	success, conflict := 0, 0
	for e := range out {
		if e == nil {
			success++
		} else if errors.Is(e, store.ErrVersionConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal(success, conflict)
	}
}
func TestCostWorkDimensionsAndWrongProject(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	work, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if e != nil {
		t.Fatal(e)
	}
	in := ci("Plant allowance")
	in.LineKind = "works"
	in.Category = ""
	in.WorkItemID = work.ID
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	tot, e := s.CostTotals(ctx, orgA, projectA, "", "system")
	if e != nil || tot.Groups[work.SystemID]["budget"].Amount == nil {
		t.Fatal(tot, e)
	}
	in.CostWrite = cw(p)
	in.PackageID = "22222222-2222-4222-8222-ffffffffffff"
	if _, e = s.CreateCostItem(ctx, orgA, projectA, userA, in); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("wrong package", e)
	}
	other := "11111111-1111-4111-8111-aaaaaaaaaa99"
	e = s.CreateProject(ctx, orgA, other, "Other")
	if e != nil {
		t.Fatal(e)
	}
	in.CostWrite = store.CostWrite{}
	in.PackageID = ""
	if _, e = s.CreateCostItem(ctx, orgA, other, userA, in); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("wrong project work", e)
	}
}

func TestCostCalculationTaxBasisAndRollback(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	in := ci("Measured allowance")
	in.Quantity = cm("3.333")
	in.Unit = "m2"
	in.Rate = cm("0.3350")
	in.RateBasis = "per m2"
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	id := p.Items[0].ID
	p, e = s.PutCostValue(ctx, orgA, projectA, id, userA, "estimate", store.CostValueInput{PlanVersionID: p.ID, PlanVersion: p.Version, Calculate: true, Value: costs.Value{Meaning: "allowance"}})
	if e != nil {
		t.Fatal(e)
	}
	v := p.Items[0].Values["estimate"]
	if v.Amount == nil || *v.Amount != "1.12" || v.Origin != "calculation" {
		t.Fatal(v)
	}
	settings := p.Settings
	settings.TaxBasis = "inc_tax"
	if _, e = s.SaveCostSettings(ctx, orgA, projectA, userA, store.CostSettingsInput{CostWrite: cw(p), Settings: settings}); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("mixed tax", e)
	}
	child := ci("Overallocated")
	child.Values["budget"] = cv("101.00")
	if _, e = s.SubdivideCostItem(ctx, orgA, projectA, id, userA, store.CostSubdivision{CostWrite: cw(p), Children: []store.CostItemInput{child}, Residual: "explicit"}); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("overallocated", e)
	}
	after, e := s.ReadCostPlan(ctx, orgA, projectA, "")
	if e != nil || after.Version != p.Version || len(after.Items) != 1 || !after.Items[0].Posting {
		t.Fatal("partial allocation", after, e)
	}
}
func TestCostWorkRetirementBlocker(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	w, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if e != nil {
		t.Fatal(e)
	}
	in := ci("Plant")
	in.LineKind = "works"
	in.Category = ""
	in.WorkItemID = w.ID
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	var blocker *store.WorkReferencedError
	if e = s.RetireWorkItem(ctx, orgA, projectA, w.ID, userA, w.Version); !errors.As(e, &blocker) {
		t.Fatal("live cost failed to block retirement", e)
	}
	in.CostWrite = cw(p)
	in.Excluded = true
	in.Values = nil
	if _, e = s.PatchCostItem(ctx, orgA, projectA, p.Items[0].ID, userA, in); e != nil {
		t.Fatal(e)
	}
	if e = s.RetireWorkItem(ctx, orgA, projectA, w.ID, userA, w.Version); e != nil {
		t.Fatal("excluded cost blocked retirement", e)
	}
}

func TestCostPackageResponsibilityFeeStageAndLinks(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	pkg, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Design", LifecycleStatus: "planned"})
	if e != nil {
		t.Fatal(e)
	}
	w, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if e != nil {
		t.Fatal(e)
	}
	in := ci("Works")
	in.LineKind = "works"
	in.Category = ""
	in.WorkItemID = w.ID
	in.PackageID = pkg.ID
	if _, e = s.CreateCostItem(ctx, orgA, projectA, userA, in); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("missing responsibility", e)
	}
	scope, e := s.CreatePackageScope(ctx, orgA, projectA, pkg.ID, userA, store.ScopeInput{ScopeContent: procurement.ScopeContent{ItemKind: "responsibility", WorkItemID: w.ID, Role: "design", UserText: "Design plant", StageID: pkg.Stages[0].ID, Inclusion: "included"}})
	if e != nil {
		t.Fatal(e)
	}
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	id := p.Items[0].ID
	p, e = s.LinkScopeCost(ctx, orgA, projectA, userA, store.CostLinkInput{CostWrite: cw(p), CostItemID: id, ScopeItemID: scope.ID})
	if e != nil || len(p.Links) != 1 {
		t.Fatal(p, e)
	}
	fee := ci("Design fee")
	fee.CostWrite = cw(p)
	fee.LineKind = "fee"
	fee.Category = ""
	fee.PackageID = pkg.ID
	if _, e = s.CreateCostItem(ctx, orgA, projectA, userA, fee); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("fee missing stage", e)
	}
	fee.PackageStageID = pkg.Stages[0].ID
	p, e = s.CreateCostItem(ctx, orgA, projectA, userA, fee)
	if e != nil {
		t.Fatal(e)
	}
	totals, e := s.CostTotals(ctx, orgA, projectA, "", "system")
	if e != nil || *totals.Works["budget"].Amount != "100.00" || *totals.Fees["budget"].Amount != "100.00" || *totals.Overall["budget"].Amount != "200.00" {
		t.Fatal(totals, e)
	}
	frozen := p.ID
	p, e = s.BaselineCostPlan(ctx, orgA, projectA, userA, cw(p))
	if e != nil || len(p.Links) != 1 {
		t.Fatal(p, e)
	}
	p, e = s.RemoveCostItem(ctx, orgA, projectA, id, userA, cw(p))
	if e != nil || len(p.Links) != 0 {
		t.Fatal(p, e)
	}
	old, e := s.ReadCostPlan(ctx, orgA, projectA, frozen)
	if e != nil || len(old.Links) != 1 {
		t.Fatal(old, e)
	}
}
func TestCostLatencyBudget(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, ci("Allowance"))
	if e != nil {
		t.Fatal(e)
	}
	writes, reads := []time.Duration{}, []time.Duration{}
	for n := 0; n < 40; n++ {
		in := ci("Allowance")
		in.CostWrite = cw(p)
		in.Values = nil
		start := time.Now()
		p, e = s.PatchCostItem(ctx, orgA, projectA, p.Items[0].ID, userA, in)
		writes = append(writes, time.Since(start))
		if e != nil {
			t.Fatal(e)
		}
		start = time.Now()
		_, e = s.CostTotals(ctx, orgA, projectA, "", "overall")
		reads = append(reads, time.Since(start))
		if e != nil {
			t.Fatal(e)
		}
	}
	for name, samples := range map[string][]time.Duration{"cost_write": writes, "cost_totals": reads} {
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p50, p90 := samples[len(samples)/2], samples[(len(samples)*9)/10]
		t.Logf("%s p50=%s p90=%s (100/250ms)", name, p50, p90)
		if p50 > 100*time.Millisecond || p90 > 250*time.Millisecond {
			t.Fatalf("%s budget exceeded", name)
		}
	}
}

func TestCostPackageStageAndScopeRetirement(t *testing.T) {
	ctx := context.Background()
	s, _, part := workStore(t)
	pkg, e := s.CreatePackage(ctx, orgA, projectA, userA, procurement.Package{Kind: "services", Title: "Design", LifecycleStatus: "planned"})
	if e != nil {
		t.Fatal(e)
	}
	fee := ci("Fee")
	fee.LineKind = "fee"
	fee.Category = ""
	fee.PackageID = pkg.ID
	fee.PackageStageID = pkg.Stages[0].ID
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, fee)
	if e != nil {
		t.Fatal(e)
	}
	retire := true
	if _, e = s.PatchPackageStage(ctx, orgA, projectA, pkg.ID, pkg.Stages[0].ID, userA, procurement.StagePatch{Version: 1, Retired: &retire}); !errors.Is(e, store.ErrInvalidPackage) {
		t.Fatal("live fee stage retired", e)
	}
	if _, e = s.PatchPackage(ctx, orgA, projectA, pkg.ID, userA, procurement.PackagePatch{Version: 1, Retired: &retire}); !errors.Is(e, store.ErrInvalidPackage) {
		t.Fatal("live fee package retired", e)
	}
	w, e := s.CreateWorkItem(ctx, orgA, projectA, userA, works.Item{PartID: part, SystemID: "mechanical.air-conditioning", Action: "new", Title: "Plant"})
	if e != nil {
		t.Fatal(e)
	}
	scope, e := s.CreatePackageScope(ctx, orgA, projectA, pkg.ID, userA, store.ScopeInput{ScopeContent: procurement.ScopeContent{ItemKind: "responsibility", WorkItemID: w.ID, Role: "design", UserText: "Design", Inclusion: "included"}})
	if e != nil {
		t.Fatal(e)
	}
	line := ci("Plant")
	line.CostWrite = cw(p)
	line.LineKind = "works"
	line.Category = ""
	line.WorkItemID = w.ID
	line.PackageID = pkg.ID
	p, e = s.CreateCostItem(ctx, orgA, projectA, userA, line)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PatchPackageScope(ctx, orgA, projectA, pkg.ID, scope.ID, userA, store.ScopePatch{Version: 1, Retired: &retire}); !errors.Is(e, store.ErrInvalidPackage) {
		t.Fatal("last priced responsibility removed", e)
	}
	var lineID string
	for _, i := range p.Items {
		if i.WorkItemID == w.ID {
			lineID = i.ID
		}
	}
	p, e = s.LinkScopeCost(ctx, orgA, projectA, userA, store.CostLinkInput{CostWrite: cw(p), CostItemID: lineID, ScopeItemID: scope.ID})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PatchPackageScope(ctx, orgA, projectA, pkg.ID, scope.ID, userA, store.ScopePatch{Version: 1, Retired: &retire}); !errors.Is(e, store.ErrInvalidPackage) {
		t.Fatal("linked scope removed", e)
	}
}
func TestCostMissingBenchmarkDoesNotInventEstimate(t *testing.T) {
	ctx := context.Background()
	s, _, _ := workStore(t)
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, ci("Allowance"))
	if e != nil {
		t.Fatal(e)
	}
	item := p.Items[0].ID
	found, e := s.ReadCostBenchmarks(ctx, orgA, projectA, item, "NSW Sydney metro", "standard")
	if e != nil || len(found) != 0 {
		t.Fatal(found, e)
	}
	if _, e = s.ApplyCostBenchmark(ctx, orgA, projectA, item, userA, store.CostBenchmarkInput{CostWrite: cw(p), BenchmarkID: "bm.missing", BenchmarkVersion: 1, Geography: "NSW Sydney metro", Quality: "standard", AcknowledgeBasis: true}); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("invented missing benchmark", e)
	}
	p, e = s.ReadCostPlan(ctx, orgA, projectA, "")
	if e != nil || p.Items[0].Values["estimate"].Amount != nil {
		t.Fatal(p, e)
	}
}

func TestCostReviewedBenchmarkApplySnapshotsBasis(t *testing.T) {
	ctx := context.Background()
	s, b, _ := workStore(t)
	root := t.TempDir()
	if e := os.CopyFS(root, os.DirFS("../../knowledge")); e != nil {
		t.Fatal(e)
	}
	if e := os.MkdirAll(filepath.Join(root, "costs"), 0755); e != nil {
		t.Fatal(e)
	}
	data := `version: 1
benchmarks:
 - id: bm.fixture
   version: 1
   basis: rate
   unit: m2
   amount: "12.3456"
   currency: AUD
   tax_basis: ex_tax
   price_date: "2026-10-07"
   geography: NSW Sydney metro
   quality: standard
   inclusions: [installation]
   exclusions: [demolition]
   applies_when: {}
   status: reviewed
   sources: [{design: synthetic-test-only}]
`
	if e := os.WriteFile(filepath.Join(root, "costs", "benchmarks.yaml"), []byte(data), 0600); e != nil {
		t.Fatal(e)
	}
	cat, e := knowledge.Load(root)
	if e != nil {
		t.Fatal(e)
	}
	b.Catalog = cat
	b.KnowledgeVersion = cat.Version()
	s = s.WithProfile(b)
	if e = s.RebuildProfile(ctx, orgA, projectA, "", b.Compute); e != nil {
		t.Fatal(e)
	}
	in := ci("Measured")
	in.Quantity = cm("3.00")
	in.Rate = cm("0.00")
	in.Unit = "m2"
	in.RateBasis = "per m2"
	p, e := s.CreateCostItem(ctx, orgA, projectA, userA, in)
	if e != nil {
		t.Fatal(e)
	}
	date := "2026-10-07"
	settings := p.Settings
	settings.PriceDate = &date
	p, e = s.SaveCostSettings(ctx, orgA, projectA, userA, store.CostSettingsInput{CostWrite: cw(p), Settings: settings})
	if e != nil {
		t.Fatal(e)
	}
	item := p.Items[0].ID
	found, e := s.ReadCostBenchmarks(ctx, orgA, projectA, item, "NSW Sydney metro", "standard")
	if e != nil || len(found) != 1 || found[0].Amount != "37.04" {
		t.Fatal(found, e)
	}
	apply := store.CostBenchmarkInput{CostWrite: cw(p), BenchmarkID: "bm.fixture", BenchmarkVersion: 1, Geography: "NSW Sydney metro", Quality: "standard"}
	if _, e = s.ApplyCostBenchmark(ctx, orgA, projectA, item, userA, apply); !errors.Is(e, costs.ErrInvalid) {
		t.Fatal("unacknowledged basis accepted", e)
	}
	apply.AcknowledgeBasis = true
	p, e = s.ApplyCostBenchmark(ctx, orgA, projectA, item, userA, apply)
	if e != nil {
		t.Fatal(e)
	}
	v := p.Items[0].Values["estimate"]
	if v.Amount == nil || *v.Amount != "37.04" || v.Origin != "calculation" {
		t.Fatal(v)
	}
	var provenance struct {
		Benchmark knowledge.CostBenchmark `json:"benchmark"`
		Catalogue string                  `json:"catalogue_version"`
	}
	if e = json.Unmarshal(v.Provenance, &provenance); e != nil || provenance.Benchmark.ID != "bm.fixture" || provenance.Catalogue != cat.Version() {
		t.Fatal(provenance, e)
	}
}
