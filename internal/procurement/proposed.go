package procurement

import "sitewise/internal/works"

// WorksProposalKey names the one head contract proposal per project.
const WorksProposalKey = "proposed:works:head_contract"

// BuilderDesigns reports whether the procurement route puts design with the
// head contract. Unanswered defaults to Design & Construct, which most projects
// use (design note 2026-10-08-proposed-packages).
func BuilderDesigns(route string) bool { return route != "traditional" }

// RouteNeedsConfirm marks an assumed route, or a route proposed as D&C that
// does not cleanly fit it, so the user is asked to confirm.
func RouteNeedsConfirm(route string) bool {
	switch route {
	case "", "managing_contractor", "alliance", "ppp":
		return true
	}
	return false
}

type ProposedLine struct {
	WorkItemID string `json:"work_item_id"`
	Role       string `json:"role"`
	Text       string `json:"text"`
}

// WorksProposal is transient: nothing is saved until the user accepts it.
// PackageID is the accepted head contract that would receive the lines, or
// empty when accepting creates it.
type WorksProposal struct {
	Key       string         `json:"key"`
	Title     string         `json:"title"`
	PackageID string         `json:"package_id,omitempty"`
	Lines     []ProposedLine `json:"lines"`
}

// ProposeHeadContract turns gap-check findings into head contract lines, so
// the proposal fills exactly what nobody holds. Investigations are left to the
// user because they usually belong to a consultant. A trade split without a
// head contract is the user's choice and is not overridden.
func ProposeHeadContract(gaps []Gap, items []works.Item, packages []Package, route string) *WorksProposal {
	target, trades := "", false
	for _, p := range packages {
		if p.RetiredAt != nil || p.Kind != "works" {
			continue
		}
		if p.WorksScope == "head_contract" && target == "" {
			target = p.ID
		}
		trades = trades || p.WorksScope == "trade"
	}
	if target == "" && trades {
		return nil
	}
	titles := map[string]string{}
	for _, item := range items {
		titles[item.ID] = item.Title
	}
	wording := map[string][2]string{
		"install":                       {"install", "Carry out the works: "},
		"design":                        {"design", "Design: "},
		"maintain_operation_or_protect": {"protect", "Protect and keep operating: "},
	}
	lines := []ProposedLine{}
	for _, g := range gaps {
		w, ok := wording[g.Role]
		if g.State != "gap" || !ok || (g.Role == "design" && !BuilderDesigns(route)) {
			continue
		}
		lines = append(lines, ProposedLine{WorkItemID: g.WorkItemID, Role: w[0], Text: w[1] + titles[g.WorkItemID]})
	}
	if len(lines) == 0 {
		return nil
	}
	return &WorksProposal{Key: WorksProposalKey, Title: "Head contract", PackageID: target, Lines: lines}
}
