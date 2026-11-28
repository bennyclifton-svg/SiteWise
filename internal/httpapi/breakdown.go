package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"sitewise/internal/store"
)

// A contract's breakdown: the lines its payment schedules print
// (docs/plans/2026-11-26-payment-schedules.md, slice 3). The PM adds a
// line, renames it, types its note and sum, moves it between works and
// provisional sums, marks a provisional sum final, retires it, or rebuilds
// the lot from the awarded firm's tender lines. Each write answers with the
// ledger, which carries the tab. No route here calls Jev or the model.

// Budget path (bench/budgets.json).
const pathBreakdownWrite = "breakdown_write"

func init() {
	routePaths["POST /projects/{id}/contracts/{pkg}/breakdown"] = pathBreakdownWrite
	routePaths["PATCH /projects/{id}/contracts/{pkg}/breakdown/{line}"] = pathBreakdownWrite
}

type breakdownLineJSON struct {
	ID         string     `json:"id"`
	CostLineID string     `json:"cost_line_id"`
	Section    string     `json:"section"`
	Label      string     `json:"label"`
	Note       string     `json:"note"`
	Amount     moneyField `json:"amount"`
	Final      bool       `json:"final"`
	Retired    bool       `json:"retired"`
	Source     string     `json:"source"`
	// Stored is false while the breakdown is the contract's cost lines:
	// the first edit stores it.
	Stored bool `json:"stored"`
}

type breakdownGapJSON struct {
	CostLineID string `json:"cost_line_id"`
	Label      string `json:"label"`
	// Diff is the breakdown less the contract line.
	Diff string `json:"diff"`
}

func breakdownBody(c store.Contract) ([]breakdownLineJSON, []breakdownGapJSON) {
	lines, gaps := []breakdownLineJSON{}, []breakdownGapJSON{}
	for _, b := range c.Breakdown {
		lines = append(lines, breakdownLineJSON{ID: b.ID, CostLineID: b.CostLineID, Section: b.Section, Label: b.Label, Note: b.Note, Amount: cents(b.Amount),
			Final: b.Final, Retired: b.Retired, Source: b.Source, Stored: b.Stored})
	}
	for _, g := range c.Gaps {
		gaps = append(gaps, breakdownGapJSON{CostLineID: g.CostLineID, Label: g.Label, Diff: *centsValue(g.Diff)})
	}
	return lines, gaps
}

func postBreakdown(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !uuidPattern.MatchString(r.PathValue("pkg")) {
		http.NotFound(w, r)
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.BreakdownInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.AddBreakdownLine(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg"), in); err != nil {
		breakdownError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

func patchBreakdown(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !uuidPattern.MatchString(r.PathValue("pkg")) || !uuidPattern.MatchString(r.PathValue("line")) {
		http.NotFound(w, r)
		return
	}
	session, ok := deliverySession(w, r, deps, true)
	if !ok {
		return
	}
	var in store.BreakdownInput
	if readProposalJSON(w, r, deps.MaxBodyBytes, &in) != nil || in.Rebuild {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := deps.Store.EditBreakdownLine(r.Context(), session.OrgID, r.PathValue("id"), session.UserID, r.PathValue("pkg"), r.PathValue("line"), in); err != nil {
		breakdownError(w, deps, err)
		return
	}
	invoiceWritten(w, r, deps, session)
}

func breakdownError(w http.ResponseWriter, deps Deps, err error) {
	if errors.Is(err, store.ErrBreakdownIssued) {
		http.Error(w, "A payment schedule is issued on this contract, so its breakdown can no longer be rebuilt. Add, rename or retire lines instead.", http.StatusConflict)
		return
	}
	claimError(w, deps, err)
}

// breakdownRef finds one line of a contract's breakdown by its id, short id
// or description.
func breakdownRef(c store.Contract, ref string) (store.BreakdownLine, error) {
	ref = strings.TrimSpace(ref)
	if b, err := byRef(c.Breakdown, func(b store.BreakdownLine) string { return b.ID }, ref, "breakdown line"); err == nil {
		return b, nil
	}
	var found []store.BreakdownLine
	for _, b := range c.Breakdown {
		if strings.EqualFold(strings.TrimSpace(b.Label), ref) {
			found = append(found, b)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return store.BreakdownLine{}, refuse("line", "%s has no one breakdown line %q; read variations lists its lines.", c.Title, ref)
}

func breakdownContract(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (store.Contract, error) {
	cs, _, err := d.Store.VariationRegister(ctx, who.OrgID, project)
	if err != nil {
		return store.Contract{}, err
	}
	return contractRef(cs, fieldText(f, "package"))
}

// yesNo reads a yes or no the PM gave; ok is false when the field is absent.
func yesNo(f map[string]any, name string) (value, ok bool, err error) {
	raw, present := f[name]
	if !present {
		return false, false, nil
	}
	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(raw))) {
	case "yes", "true", "y":
		return true, true, nil
	case "no", "false", "n":
		return false, true, nil
	}
	return false, false, refuse(name, "%s is yes or no.", name)
}

var breakdownForwards = []forward{
	{Name: "claims.breakdown_add", Undo: true, About: "Add a line to a contract's breakdown, the lines its payment schedules print: a works line or a provisional sum, with its sum when the PM gave one.",
		Fields:  `package (the contract's package id or title); label (the line's description, in the PM's words); section (works|provisional; works when not said); amount (the line's sum ex GST, as the PM gave it); note (a short note printed beside the label)`,
		Figures: map[string]string{"amount": "money"}, Route: "POST /projects/{id}/contracts/{pkg}/breakdown",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			c, err := breakdownContract(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			if err := need(f, "label"); err != nil {
				return sent{}, err
			}
			body := map[string]any{}
			textFields(f, body, "label", "section", "amount", "note")
			f["_label"], f["_contract"] = fieldText(f, "label"), c.Title
			return sent{Path: map[string]string{"pkg": c.PackageID}, Body: body}, nil
		},
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Added %q to %s's breakdown.", fieldText(f, "_label"), fieldText(f, "_contract"))
		}},
	{Name: "claims.breakdown_edit", Undo: true, About: "Change one line of a contract's breakdown: rename it, type its note or sum, move it between works and provisional sums, mark a provisional sum final (settled at what was completed) or not final, or take it off the breakdown.",
		Fields:  `package (the contract's package id or title); line (the line's id or its description); label; note; amount (its sum ex GST, as the PM gave it; empty is unknown); section (works|provisional); final (yes|no); remove (yes takes the line off the breakdown, no puts it back)`,
		Figures: map[string]string{"amount": "money"}, Route: "PATCH /projects/{id}/contracts/{pkg}/breakdown/{line}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			c, err := breakdownContract(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			if err := need(f, "line"); err != nil {
				return sent{}, err
			}
			b, err := breakdownRef(c, fieldText(f, "line"))
			if err != nil {
				return sent{}, err
			}
			body := map[string]any{}
			textFields(f, body, "label", "section", "amount", "note")
			if v, ok, err := yesNo(f, "final"); err != nil {
				return sent{}, err
			} else if ok {
				body["final"] = v
			}
			if v, ok, err := yesNo(f, "remove"); err != nil {
				return sent{}, err
			} else if ok {
				body["retired"] = v
			}
			f["_label"], f["_contract"] = b.Label, c.Title
			return sent{Path: map[string]string{"pkg": c.PackageID, "line": b.ID}, Body: body}, nil
		},
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Changed %q on %s's breakdown.", fieldText(f, "_label"), fieldText(f, "_contract"))
		}},
	{Name: "claims.breakdown_rebuild", Undo: true, About: "Rebuild a contract's breakdown from the awarded firm's own tender lines, where they add up to each cost line. It replaces the breakdown, so from chat it waits for the PM's click; it is refused once a payment schedule is issued on the contract.",
		Fields: `package (the contract's package id or title)`, Route: "POST /projects/{id}/contracts/{pkg}/breakdown",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			c, err := breakdownContract(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			f["_contract"] = c.Title
			return sent{Path: map[string]string{"pkg": c.PackageID}, Body: map[string]any{"rebuild": true}}, nil
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			c, err := breakdownContract(ctx, d, who, project, f)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Rebuild %s's breakdown from the awarded tender's lines? The lines there now are replaced.", c.Title), nil
		},
		Done: func(f map[string]any) string {
			return "Rebuilt " + fieldText(f, "_contract") + "'s breakdown from the tender."
		}},
}

func init() {
	claimsForwards = append(claimsForwards, breakdownForwards...)
	claimVouches["claims.breakdown_rebuild"] = "Rebuilding a contract's breakdown replaces its lines; it is yours to confirm."
}
