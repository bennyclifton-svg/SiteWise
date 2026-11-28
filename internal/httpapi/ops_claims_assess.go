package httpapi

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"sitewise/internal/invoices"
	"sitewise/internal/store"
)

// Claim operations (variations plan, "From chat"). Typing an assessed
// amount applies at once with Undo; Accept claimed and a contract's periods
// wait for the PM's click from chat. Served dates go through the existing
// invoices.edit (fields served_on, schedule_served_on, reference_date).

var claimNumberRef = regexp.MustCompile(`(?i)^\s*(?:progress\s+)?(?:claim|pc)\s*(?:no\.?|#)?\s*-?\s*([A-Za-z0-9\-]{1,20})\s*$`)

// claimRef finds one live claim by its invoice id, short id or number.
func claimRef(v store.LedgerView, ref string) (store.ClaimView, error) {
	ref = strings.TrimSpace(ref)
	if c, err := byRef(v.Claims, func(c store.ClaimView) string { return c.InvoiceID }, ref, "progress claim"); err == nil {
		return c, nil
	}
	want := ref
	if m := claimNumberRef.FindStringSubmatch(ref); m != nil {
		want = m[1]
	}
	key := func(s string) string {
		return strings.TrimLeft(strings.ToUpper(strings.NewReplacer("PC", "", "-", "", " ", "").Replace(s)), "0")
	}
	var found []store.ClaimView
	for _, c := range v.Claims {
		if c.Number != "" && key(c.Number) == key(want) {
			found = append(found, c)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return store.ClaimView{}, refuse("claim", "There is no progress claim %s; read variations lists the claims.", ref)
	}
	return store.ClaimView{}, refuse("claim", "Two contracts have a claim %s; name it by its id.", ref)
}

var rowOrdinal = regexp.MustCompile(`(?i)^\s*row\s*(\d{1,3})\s*$`)

// claimRowRef finds one row of a claim by id, "row 3" (as read variations
// numbers them) or its printed label.
func claimRowRef(c store.ClaimView, ref string) (invoices.ClaimRow, error) {
	if m := rowOrdinal.FindStringSubmatch(ref); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n >= 1 && n <= len(c.Rows) {
			return c.Rows[n-1], nil
		}
	}
	if r, err := byRef(c.Rows, func(r invoices.ClaimRow) string { return r.ID }, ref, "claim row"); err == nil {
		return r, nil
	}
	var found []invoices.ClaimRow
	for _, r := range c.Rows {
		if strings.EqualFold(strings.TrimSpace(r.Label), strings.TrimSpace(ref)) {
			found = append(found, r)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return invoices.ClaimRow{}, refuse("row", "Name the row as read variations shows it (row 3, or its id).")
}

// claimRowTarget names the row an assessment is for: a row of the claim, or
// a line, a variation or retention of its contract that the claim left out,
// whose row the write then adds.
func claimRowTarget(v store.LedgerView, c store.ClaimView, ref string) (map[string]any, string, error) {
	r, err := claimRowRef(c, ref)
	if err == nil {
		return map[string]any{"id": r.ID}, r.Label, nil
	}
	ref = strings.TrimSpace(ref)
	// rowOn is the claim's own row on that line when it has one after all.
	rowOn := func(match func(invoices.ClaimRow) bool, add map[string]any, label string) (map[string]any, string, error) {
		for _, row := range c.Rows {
			if match(row) {
				return map[string]any{"id": row.ID}, row.Label, nil
			}
		}
		return add, label, nil
	}
	if strings.EqualFold(ref, "retention") {
		return rowOn(func(r invoices.ClaimRow) bool { return r.Kind == invoices.RowRetention }, map[string]any{"retention": true}, "Retention")
	}
	for _, ct := range v.Contracts {
		if ct.PackageID != c.PackageID {
			continue
		}
		if id, lerr := lineRef(ct, ref); lerr == nil && id != "" {
			label := ""
			for _, l := range ct.Lines {
				if l.ID == id {
					label = l.Label
				}
			}
			for _, b := range ct.Breakdown {
				if b.ID == id {
					label = b.Label
				}
			}
			// The id is a breakdown line's, or a cost line's on a contract
			// whose breakdown is its cost lines.
			return rowOn(func(r invoices.ClaimRow) bool {
				return r.Kind == invoices.RowContract && r.VariationID == "" && (r.BreakdownID == id || r.BreakdownID == "" && r.LineID == id)
			}, map[string]any{"line": id}, label)
		}
	}
	for _, va := range v.Variations {
		name := strings.TrimSpace(va.Number + " " + va.Title)
		if va.PackageID == c.PackageID && (va.ID == ref || strings.EqualFold(va.Number, ref) || strings.EqualFold(name, ref)) {
			id := va.ID
			return rowOn(func(r invoices.ClaimRow) bool { return r.VariationID == id }, map[string]any{"variation": id}, name)
		}
	}
	return nil, "", err
}

func claimLedgerFor(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (store.LedgerView, store.ClaimView, error) {
	v, err := d.Store.InvoiceLedger(ctx, who.OrgID, project, todaySydney())
	if err != nil {
		return v, store.ClaimView{}, err
	}
	if err := need(f, "claim"); err != nil {
		return v, store.ClaimView{}, err
	}
	c, err := claimRef(v, fieldText(f, "claim"))
	return v, c, err
}

var claimForwards = []forward{
	{Name: "claims.assess", Undo: true, About: "Type what the PM assesses as complete to date on one row of a progress claim, as an amount or as a percent of the line's sum, with the reason when it is less than claimed; or change the line the row charges for. A contract line, a variation or retention the claim left out can be named in place of a row: its row is added. One operation per row.",
		Fields:  `claim (its id or number, e.g. claim 4); row (row 3, its id, its printed label, or a contract line, a variation number or retention the claim left out); assessed_to_date (the amount to date ex GST, as the PM gave it; empty clears it); percent (complete to date as a percent of the line's sum, as the PM gave it; SiteWise works out the dollars); reason (the PM's words); line (a contract line id or label, or retention). For retention, in place of row, exactly one of: retention_hold_percent or retention_return_percent (what this claim holds or returns, as a percent of the contract's retention base), retention_hold or retention_return (the same in dollars; retention_hold 0 is no change this claim)`,
		Figures: map[string]string{"assessed_to_date": "money", "percent": "number", "retention_hold_percent": "number", "retention_return_percent": "number", "retention_hold": "money", "retention_return": "money"},
		Route:   "PUT /projects/{id}/invoices/{inv}/claim-lines",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			v, c, err := claimLedgerFor(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			if len(c.Issued) > 0 {
				return sent{}, refuse("claim", "Claim %s's payment schedule is issued; its rows can no longer change.", c.Number)
			}
			// What this claim holds or returns of retention needs no row:
			// SiteWise adds it to what was held before.
			move := map[string]any{}
			for from, to := range map[string]string{"retention_hold_percent": "hold_percent", "retention_return_percent": "return_percent", "retention_hold": "hold", "retention_return": "return"} {
				if _, ok := f[from]; ok {
					move[to] = fieldText(f, from)
				}
			}
			if len(move) > 0 {
				f["_row"], f["_claim"] = "retention", c.Number
				return sent{Path: map[string]string{"inv": c.InvoiceID}, Body: map[string]any{"retention": move}}, nil
			}
			if err := need(f, "row"); err != nil {
				return sent{}, err
			}
			row, label, err := claimRowTarget(v, c, fieldText(f, "row"))
			if err != nil {
				return sent{}, err
			}
			textFields(f, row, "assessed_to_date", "reason", "percent")
			// Only a row the claim already has can move to another line.
			if ref := fieldText(f, "line"); ref != "" && row["id"] != nil {
				if strings.EqualFold(ref, "retention") {
					row["retention"] = true
				} else {
					found := false
					for _, ct := range v.Contracts {
						if ct.PackageID == c.PackageID {
							id, err := lineRef(ct, ref)
							if err != nil {
								return sent{}, err
							}
							row["line"], found = id, true
						}
					}
					if !found {
						return sent{}, refuse("line", "That claim's contract is not live.")
					}
				}
			}
			f["_row"], f["_claim"] = label, c.Number
			return sent{Path: map[string]string{"inv": c.InvoiceID}, Body: map[string]any{"rows": []any{row}}}, nil
		},
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Assessed %s on claim %s.", fieldText(f, "_row"), fieldText(f, "_claim"))
		}},
	{Name: "claims.accept_claimed", Undo: true, About: "Accept every row of a progress claim at its claimed amount to date: the PM vouches for the builder's figures, so from chat it waits for the PM's click.",
		Fields: `claim (its id or number)`, Route: "PUT /projects/{id}/invoices/{inv}/claim-lines",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			_, c, err := claimLedgerFor(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			f["_claim"] = c.Number
			return sent{Path: map[string]string{"inv": c.InvoiceID}, Body: map[string]any{"accept_claimed": true}}, nil
		},
		Card: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
			_, c, err := claimLedgerFor(ctx, d, who, project, f)
			if err != nil {
				return "", err
			}
			if len(c.Issued) > 0 {
				return "", refuse("claim", "Claim %s's payment schedule is issued; its rows can no longer change.", c.Number)
			}
			return fmt.Sprintf("Accept claim %s at every row's claimed amount to date? You can still cut a row before you issue the schedule.", c.Number), nil
		},
		Done: func(f map[string]any) string { return "Accepted claim " + fieldText(f, "_claim") + " as claimed." }},
	{Name: "claims.contract_periods", Click: true, About: "set a contract's own terms as the contract states them: its payment schedule and payment periods (days, business or calendar), its contract date, and its retention as a percent of a base.",
		Fields:  `package (the contract's package id or title); schedule_days; schedule_basis (business|calendar); payment_days; payment_basis (business|calendar); contract_date (the contract's own date); retention_percent (the retention rate, a percent); retention_base (works: the works sub total or original sum | let: works plus provisional sums | to_date: the contract sum to date); act_applies (yes|no to name the Security of Payment Act on this contract's schedules, empty for the default: on for works, off for consultants); each only as the PM gave it`,
		Figures: map[string]string{"schedule_days": "number", "payment_days": "number", "contract_date": "date", "retention_percent": "number"}, Route: "PATCH /projects/{id}/procurement/packages/{pkg}/award",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			cs, _, err := d.Store.VariationRegister(ctx, who.OrgID, project)
			if err != nil {
				return sent{}, err
			}
			c, err := contractRef(cs, fieldText(f, "package"))
			if err != nil {
				return sent{}, err
			}
			body := map[string]any{"schedule_basis": fieldText(f, "schedule_basis"), "payment_basis": fieldText(f, "payment_basis"),
				"schedule_days": c.ScheduleDays, "payment_days": c.PaymentDays}
			for _, k := range []string{"schedule_days", "payment_days"} {
				if raw, ok := f[k]; ok {
					n, err := strconv.Atoi(strings.TrimSpace(fmt.Sprint(raw)))
					if err != nil {
						return sent{}, refuse(k, "%s is a whole number of days.", k)
					}
					body[k] = n
				}
			}
			if body["schedule_basis"] == "" {
				body["schedule_basis"] = c.ScheduleBasis
			}
			if body["payment_basis"] == "" {
				body["payment_basis"] = c.PaymentBasis
			}
			for _, k := range []string{"contract_date", "retention_percent", "retention_base", "act_applies"} {
				if _, ok := f[k]; ok {
					body[k] = fieldText(f, k)
				}
			}
			return sent{Path: map[string]string{"pkg": c.PackageID}, Body: body}, nil
		},
		Done: func(map[string]any) string { return "Set the contract's terms." }},
}

func init() {
	claimsForwards = append(claimsForwards, claimForwards...)
	claimVouches["claims.accept_claimed"] = "Accepting a claim as claimed is yours to confirm."
}
