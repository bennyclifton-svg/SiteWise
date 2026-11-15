package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sitewise/internal/procurement"
	"sitewise/internal/store"
)

// Procurement operations. procurement.tender is passed whole to
// ApplyTenderOps, which is one transaction and returns the change the
// change log can undo.

func procurementOperations() []operation {
	return []operation{
		{Name: "procurement.tender", About: "One edit to a package's tender comparison. Edits of a batch are for one package and apply together.",
			Fields: `package (id); op (add_adjustment|set_adjustment|remove_adjustment|set_adjustment_amount|apply_adjustment|move_line|take_out_line|edit_line); ` +
				`add_adjustment: label, note; set_adjustment_amount: adjustment (id), submission (the firm's column id), amount; ` +
				`apply_adjustment: adjustment, submission, applied (true|false); move_line: line (id), row (a row key); take_out_line: line`,
			Figures: map[string]string{"amount": "money"},
			Routes:  []string{"POST /projects/{id}/procurement/packages/{pkg}/tender/ops"}, Hold: tenderHold, Batch: tenderBatch},
	}
}

// tenderEdit is a TenderOp as chat sends it, with the package it is for.
type tenderEdit struct {
	Package string `json:"package"`
	store.TenderOp
}

func packageByRef(pkgs []procurement.Package, ref string) (procurement.Package, error) {
	if strings.TrimSpace(ref) == "" {
		return procurement.Package{}, refuse("package", "Name the package by its id.")
	}
	return byRef(pkgs, func(p procurement.Package) string { return p.ID }, ref, "package")
}

func decodeTender(call opCall) (tenderEdit, error) {
	var e tenderEdit
	dec := json.NewDecoder(bytes.NewReader(call.Fields))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		return e, refuse("", "The fields are not in the shape a tender edit takes.")
	}
	return e, nil
}

// tenderHold: checking a firm's column is the PM vouching for it, so it
// waits for a click (review change 4).
func tenderHold(_ context.Context, _ Deps, _ opActor, _ string, raw json.RawMessage) (string, error) {
	e, err := decodeTender(opCall{Fields: raw})
	if err != nil {
		return "", err
	}
	if e.Checked != nil || strings.Contains(e.Op.Op, "check") {
		return "Checking a tender column is yours to confirm, so it waits for your click.", nil
	}
	return "", nil
}

func tenderBatch(ctx context.Context, d Deps, who opActor, project string, in opBatch) (opApplied, error) {
	pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
	if err != nil {
		return opApplied{}, err
	}
	var pkg procurement.Package
	ops := make([]store.TenderOp, len(in.Calls))
	for i, call := range in.Calls {
		e, err := decodeTender(call)
		if err != nil {
			return opApplied{}, asOpError(i, err)
		}
		p, err := packageByRef(pkgs, e.Package)
		if err != nil {
			return opApplied{}, asOpError(i, err)
		}
		if i > 0 && p.ID != pkg.ID {
			return opApplied{}, &opError{Index: i, Field: "package", Message: "A batch of tender edits is for one package."}
		}
		pkg, ops[i] = p, e.TenderOp
	}
	table, err := d.Store.ReadPriceTable(ctx, who.OrgID, project, pkg.ID)
	if err != nil {
		return opApplied{}, err
	}
	if table.Tender == nil {
		return opApplied{}, &opError{Index: -1, Message: fmt.Sprintf("%q has no tender comparison yet.", pkg.Title)}
	}
	change, err := d.Store.ApplyTenderOps(ctx, who.OrgID, project, pkg.ID, who.UserID, table.Tender.Version, ops)
	if err != nil {
		return opApplied{}, err
	}
	out := opApplied{ChangeID: change, Undo: opUndo{Kind: "change"}}
	if change == "" {
		out.Undo.Kind = "none"
	}
	for i, op := range ops {
		out.Results = append(out.Results, opResult{Op: in.Calls[i].Op, Summary: tenderSummary(op, pkg.Title)})
	}
	return out, nil
}

func tenderSummary(op store.TenderOp, pkg string) string {
	str := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	switch op.Op.Op {
	case "add_adjustment":
		return fmt.Sprintf("Added the adjustment %q to the %s comparison.", str(op.Label), pkg)
	case "set_adjustment_amount":
		return fmt.Sprintf("Set an adjustment amount of %s on the %s comparison.", dollarsText(str(op.Amount)), pkg)
	case "apply_adjustment":
		return fmt.Sprintf("Applied an adjustment on the %s comparison.", pkg)
	case "move_line":
		return fmt.Sprintf("Moved a tender line on the %s comparison.", pkg)
	}
	return fmt.Sprintf("Tender edit on %s: %s.", pkg, op.Op.Op)
}

func dollarsText(s string) string {
	if s == "" {
		return "unknown"
	}
	m, err := typedAmount("amount", s)
	if err != nil {
		return s
	}
	return dollars(m)
}
