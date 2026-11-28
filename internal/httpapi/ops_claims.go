package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"sitewise/internal/invoices"
	"sitewise/internal/store"
)

// Variations and progress claims from chat and the screen
// (docs/plans/2026-11-17-variations-progress-claims.md, "From chat" and
// Task 11). Recording and editing a variation and typing an assessment
// apply at once with Undo. Approving or rejecting a variation, Accept
// claimed and a contract's periods wait for the PM's click. Issuing the
// payment schedule is never an operation: chat opens the claim's panel and
// the PM presses Issue (decision 9).

// contractRef finds one live contract by its package id, short id or title.
func contractRef(cs []store.Contract, ref string) (store.Contract, error) {
	ref = strings.TrimSpace(ref)
	if c, err := byRef(cs, func(c store.Contract) string { return c.PackageID }, ref, "contract"); err == nil {
		return c, nil
	}
	var found []store.Contract
	for _, c := range cs {
		if strings.EqualFold(c.Title, ref) || ref != "" && strings.EqualFold(c.Firm, ref) {
			found = append(found, c)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	if len(cs) == 1 && ref == "" {
		return cs[0], nil
	}
	return store.Contract{}, refuse("package", "Name the contract by its package id or title from read variations.")
}

var variationNumberRef = regexp.MustCompile(`(?i)^\s*(?:variation\s+)?(V\s*-?\s*\d{1,4}|[A-Z]{1,3}\d{1,4})\s*$`)

// variationRef finds one variation by its id, short id or number. A number
// two contracts share needs the contract too.
func variationRef(cs []store.Contract, vs []invoices.Variation, ref, contract string) (invoices.Variation, error) {
	if m := variationNumberRef.FindStringSubmatch(ref); m != nil {
		want := strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(m[1]))
		var pkg string
		if contract != "" {
			c, err := contractRef(cs, contract)
			if err != nil {
				return invoices.Variation{}, err
			}
			pkg = c.PackageID
		}
		var found []invoices.Variation
		for _, v := range vs {
			if (pkg == "" || v.PackageID == pkg) && variationNumberKey(v.Number) == variationNumberKey(want) {
				found = append(found, v)
			}
		}
		switch len(found) {
		case 1:
			return found[0], nil
		case 0:
			return invoices.Variation{}, refuse("variation", "There is no variation %s.", want)
		}
		return invoices.Variation{}, refuse("variation", "Two contracts have a %s; name the contract too.", want)
	}
	return byRef(vs, func(v invoices.Variation) string { return v.ID }, ref, "variation")
}

// variationNumberKey reads V3, V03 and V-003 as one number.
func variationNumberKey(n string) string {
	n = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(n))
	head := strings.TrimRight(n, "0123456789")
	digits := strings.TrimLeft(n[len(head):], "0")
	return head + digits
}

// lineRef finds one of a contract's lines by id or label; blank is the
// contract's only line, which the store chooses.
func lineRef(c store.Contract, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", nil
	}
	// A line of the breakdown first: it is what a schedule prints.
	if b, err := byRef(c.Breakdown, func(b store.BreakdownLine) string { return b.ID }, ref, "breakdown line"); err == nil && !b.Retired {
		return b.ID, nil
	}
	var named []string
	for _, b := range c.Breakdown {
		if !b.Retired && strings.EqualFold(strings.TrimSpace(b.Label), ref) {
			named = append(named, b.ID)
		}
	}
	if len(named) == 1 {
		return named[0], nil
	}
	if l, err := byRef(c.Lines, func(l store.ContractLine) string { return l.ID }, ref, "contract line"); err == nil {
		return l.ID, nil
	}
	for _, l := range c.Lines {
		if strings.EqualFold(l.Label, ref) {
			return l.ID, nil
		}
	}
	return "", refuse("line", "%s has no line %q; read variations lists its lines.", c.Title, ref)
}

func variationFigures() map[string]string {
	return map[string]string{"claimed": "money", "estimate": "money", "approved": "money", "time_claimed_days": "number", "submitted_on": "date", "decided_on": "date"}
}

// textFields copies the named fields as the store's strings.
func textFields(f map[string]any, body map[string]any, names ...string) {
	for _, n := range names {
		v, ok := f[n]
		if !ok {
			continue
		}
		switch x := v.(type) {
		case nil:
			body[n] = ""
		case string:
			body[n] = x
		case float64:
			body[n] = strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", x), "0"), ".")
		default:
			body[n] = fmt.Sprint(x)
		}
	}
}

var claimsForwards = []forward{
	{Name: "variations.add", Undo: true, About: "Record a variation on a live contract (saved as submitted unless the PM says proposed). Amounts stay unknown unless the PM gave them.",
		Fields:  `package (the contract's package id or title from read variations); title (what the variation is for); line (the contract line, only when the contract has more than one); number (only when the PM gave one, else SiteWise numbers it V01, V02…); status (proposed|submitted); claimed (the builder's amount ex GST, only as the PM gave it); estimate (the PM's own estimate); time_claimed_days (days of time claimed, only as the PM said); submitted_on; details`,
		Figures: variationFigures(), Route: "POST /projects/{id}/variations",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			cs, _, err := d.Store.VariationRegister(ctx, who.OrgID, project)
			if err != nil {
				return sent{}, err
			}
			c, err := contractRef(cs, fieldText(f, "package"))
			if err != nil {
				return sent{}, err
			}
			if err := need(f, "title"); err != nil {
				return sent{}, err
			}
			if st := fieldText(f, "status"); st != "" && st != invoices.VariationProposed && st != invoices.VariationSubmitted {
				return sent{}, refuse("status", "A new variation is proposed or submitted; approving or rejecting it is variations.decide.")
			}
			if _, ok := f["approved"]; ok {
				return sent{}, refuse("approved", "An approved amount is given by approving the variation (variations.decide).")
			}
			line, err := lineRef(c, fieldText(f, "line"))
			if err != nil {
				return sent{}, err
			}
			body := map[string]any{"package": c.PackageID}
			if line != "" {
				body["line"] = line
			}
			textFields(f, body, "title", "number", "status", "claimed", "estimate", "time_claimed_days", "submitted_on", "details")
			f["_contract"] = c.Title
			return sent{Body: body}, nil
		},
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Recorded the variation %q on %s.", fieldText(f, "title"), fieldText(f, "_contract"))
		}},
	{Name: "variations.edit", Undo: true, About: "Change a variation's description, line, claimed amount, the PM's estimate, time claimed, dates or details, or mark it proposed, submitted or withdrawn.",
		Fields:  `variation (its id, or its number such as V03); package (only when two contracts share that number); title; line; number; status (proposed|submitted|withdrawn); claimed; estimate; time_claimed_days; submitted_on; details; link_documents (document ids from documents); unlink_documents`,
		Figures: variationFigures(), Route: "PATCH /projects/{id}/variations/{v}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			cs, vs, err := d.Store.VariationRegister(ctx, who.OrgID, project)
			if err != nil {
				return sent{}, err
			}
			if err := need(f, "variation"); err != nil {
				return sent{}, err
			}
			v, err := variationRef(cs, vs, fieldText(f, "variation"), fieldText(f, "package"))
			if err != nil {
				return sent{}, err
			}
			if st := fieldText(f, "status"); st == invoices.VariationApproved || st == invoices.VariationRejected {
				return sent{}, refuse("status", "Approving or rejecting is yours to confirm: use variations.decide.")
			}
			if _, ok := f["approved"]; ok {
				return sent{}, refuse("approved", "An approved amount is given by approving the variation (variations.decide).")
			}
			body := map[string]any{"version": v.Version}
			if ref := fieldText(f, "line"); ref != "" {
				for _, c := range cs {
					if c.AwardID == v.AwardID {
						line, err := lineRef(c, ref)
						if err != nil {
							return sent{}, err
						}
						body["line"] = line
					}
				}
			}
			textFields(f, body, "title", "number", "status", "claimed", "estimate", "time_claimed_days", "submitted_on", "details")
			for _, k := range []string{"link_documents", "unlink_documents"} {
				if list, ok := f[k].([]any); ok {
					ids := []string{}
					for _, x := range list {
						ids = append(ids, fmt.Sprint(x))
					}
					body[k] = ids
				}
			}
			f["_number"] = v.Number
			return sent{Path: map[string]string{"v": v.ID}, Body: body}, nil
		},
		Done: func(f map[string]any) string { return "Changed variation " + fieldText(f, "_number") + "." }},
	{Name: "variations.decide", Undo: true, About: "Approve a variation at an amount, or reject it: the PM decides it for the client, so from chat it waits for the PM's click.",
		Fields:  `variation (its id or number); package (only when two contracts share that number); decision (approve|reject); approved (the approved amount ex GST, as the PM gave it; needed to approve, zero or negative allowed); decided_on (only a date the PM gave)`,
		Figures: variationFigures(), Route: "PATCH /projects/{id}/variations/{v}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			v, body, err := decideBody(ctx, d, who, project, f)
			return sent{Path: map[string]string{"v": v.ID}, Body: body}, err
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			v, body, err := decideBody(ctx, d, who, project, f)
			if err != nil {
				return "", err
			}
			if body["status"] == invoices.VariationRejected {
				return fmt.Sprintf("Reject variation %s (%s)?", v.Number, v.Title), nil
			}
			return fmt.Sprintf("Approve variation %s (%s) at $%s ex GST? The contract sum to date and Forecast Final Cost move by that amount.", v.Number, v.Title, body["approved"]), nil
		},
		Done: func(f map[string]any) string { return "Decided the variation." }},
}

// claimsOperations are the forwards above. A decision applies at once from
// the screen (the PM's own click, with Undo) and waits for the click from
// chat.
func claimsOperations() []operation {
	var out []operation
	for _, fw := range claimsForwards {
		op := fw.operation()
		if why, ok := claimVouches[fw.Name]; ok {
			op.Hold = screenClick(why)
		}
		out = append(out, op)
	}
	return out
}

// claimVouches are the operations the PM vouches for.
var claimVouches = map[string]string{
	"variations.decide": "Approving or rejecting a variation is yours to confirm.",
}

// screenClick lets the screen's own operations call through: the PM
// pressed the button. From chat the operation waits for that click.
func screenClick(why string) func(ctx context.Context, _ Deps, _ opActor, _ string, _ json.RawMessage) (string, error) {
	return func(ctx context.Context, _ Deps, _ opActor, _ string, _ json.RawMessage) (string, error) {
		if r, ok := ctx.Value(opRequestKey{}).(*http.Request); ok && strings.HasSuffix(r.URL.Path, "/ops") {
			return "", nil
		}
		return why, nil
	}
}

func decideBody(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (invoices.Variation, map[string]any, error) {
	cs, vs, err := d.Store.VariationRegister(ctx, who.OrgID, project)
	if err != nil {
		return invoices.Variation{}, nil, err
	}
	if err := need(f, "variation", "decision"); err != nil {
		return invoices.Variation{}, nil, err
	}
	v, err := variationRef(cs, vs, fieldText(f, "variation"), fieldText(f, "package"))
	if err != nil {
		return v, nil, err
	}
	body := map[string]any{"version": v.Version}
	textFields(f, body, "decided_on")
	switch fieldText(f, "decision") {
	case "approve":
		textFields(f, body, "approved")
		if strings.TrimSpace(fmt.Sprint(body["approved"])) == "" || body["approved"] == nil {
			return v, nil, refuse("approved", "Approving needs the approved amount, as you give it.")
		}
		amount, ok, err := invoices.ParseCents(fmt.Sprint(body["approved"]))
		if err != nil || !ok {
			return v, nil, refuse("approved", "The approved amount is not an amount.")
		}
		body["approved"] = invoices.FormatCents(amount)
		body["status"] = invoices.VariationApproved
	case "reject":
		body["status"] = invoices.VariationRejected
	default:
		return v, nil, refuse("decision", "decision is approve or reject.")
	}
	return v, body, nil
}
