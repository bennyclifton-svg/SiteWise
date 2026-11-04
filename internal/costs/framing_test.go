package costs

import "testing"

func TestReadBudgetFraming(t *testing.T) {
	budget := ReadBudgetFraming("t confirmed:** working budget ceiling **$920,000** (including contingency for latent con")
	if !budget.IncludesContingency || budget.Additional || budget.GST != "not_stated" {
		t.Fatalf("Walsh budget framing %+v", budget)
	}
	held := ReadBudgetFraming("ns and heritage tie-ins). Owners hold **$85,000** additional contingency for discovery")
	if !held.Additional || held.IncludesContingency {
		t.Fatalf("Walsh contingency framing %+v", held)
	}
	est, ok := ReadEstimate("$880,000 – $980,000", "**ROM construction range:** $880,000 – $980,000 ex GST for scope as described")
	if !ok || est.Low != "880000.00" || est.High != "980000.00" || est.GST != "ex" {
		t.Fatalf("Walsh estimate %+v %v", est, ok)
	}
	if _, ok := ReadEstimate("about a million", ""); ok {
		t.Fatal("an estimate read from no figure")
	}
	plain := ReadBudgetFraming("The construction budget is $2.4m excluding GST.")
	if plain.IncludesContingency || plain.Additional || plain.GST != "ex" {
		t.Fatalf("plain budget framing %+v", plain)
	}
}
