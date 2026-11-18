package httpapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"sitewise/internal/procurement"
)

// The operations beyond the first slice (plan Task 3, the rest). Each runs
// the screen's own handler (ops_forward.go). One with Undo applies at once;
// one without waits for the PM's click; a vouching one always waits.

type sender = func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error)

// forwardedOperations lists them by area.
func forwardedOperations(area string) []operation {
	var out []operation
	for _, fw := range append(append(append(append([]forward{}, forwards...), readForwards...), shareForwards...), consentForwards...) {
		if opArea(fw.Name) == area {
			out = append(out, fw.operation())
		}
	}
	return out
}

func pkgOf(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (procurement.Package, error) {
	pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
	if err != nil {
		return procurement.Package{}, err
	}
	ref := fieldText(f, "package")
	if p, err := packageByRef(pkgs, ref); err == nil {
		return p, nil
	}
	// A package may also be named by its title.
	var found []procurement.Package
	for _, p := range pkgs {
		if p.RetiredAt == nil && strings.EqualFold(p.Title, ref) {
			found = append(found, p)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return procurement.Package{}, refuse("package", "Name the package by its id from the outline.")
}

// firmOn finds a firm on a package's row by name or id.
func firmOn(ctx context.Context, d Deps, who opActor, project, pkg, name string) (id, submission, shown string, err error) {
	board, _, err := d.Store.ProcurementBoard(ctx, who.OrgID, project)
	if err != nil {
		return "", "", "", err
	}
	n := 0
	for _, c := range board[pkg] {
		if strings.EqualFold(c.FirmName, name) || c.FirmID == name || (len(name) >= 4 && strings.Contains(strings.ToLower(c.FirmName), strings.ToLower(name))) {
			id, submission, shown = c.FirmID, c.SubmissionID, c.FirmName
			n++
		}
	}
	if n != 1 {
		return "", "", "", refuse("firm", "There is not exactly one firm called %q on that package.", name)
	}
	return id, submission, shown, nil
}

// withPlan adds the draft plan's id and version to a cost plan body.
func withPlan(path func(f map[string]any) map[string]string) sender {
	return func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
		p, err := draftPlan(ctx, d, who, project)
		if err != nil {
			return sent{}, err
		}
		s := sent{Body: f}
		if path != nil {
			s.Path = path(f)
		}
		if ref := fieldText(f, "line"); ref != "" {
			item, err := costLine(p, ref)
			if err != nil {
				return sent{}, err
			}
			s.Path = map[string]string{"item": item.ID}
			delete(f, "line")
		}
		f["plan_version_id"], f["version"] = p.ID, p.Version
		return s, nil
	}
}

func take(f map[string]any, name string) string {
	v := fieldText(f, name)
	delete(f, name)
	return v
}

var forwards = []forward{
	// ---- Procurement
	{Name: "procurement.add_package", About: "Add a consultant or trade package for one discipline.",
		Fields: `discipline (an id such as consultant.structural, consultant.acoustic, trade.mechanical; read procurement lists them)`,
		Route:  "POST /projects/{id}/procurement/rows",
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			if err := need(f, "discipline"); err != nil {
				return sent{}, err
			}
			return sent{Body: map[string]any{"discipline_id": fieldText(f, "discipline")}}, nil
		},
		Done: func(f map[string]any) string { return "Added a package for " + fieldText(f, "discipline") + "." }},
	{Name: "procurement.edit_package", About: "Rename a package or change how it is engaged.",
		Fields: `package (id); title; novation (no|option|yes); engaged_by (client|lead)`,
		Route:  "PATCH /projects/{id}/packages/{pkg}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			delete(f, "package")
			delete(f, "retired")
			delete(f, "remove_cost_lines")
			f["version"] = p.Version
			return sent{Path: map[string]string{"pkg": p.ID}, Body: f}, nil
		},
		Done: func(map[string]any) string { return "Changed the package." }},
	{Name: "procurement.remove_package", Click: true, About: "remove a package from Procurement; its draft cost lines go with it, frozen baselines and issued history stay.",
		Fields: `package (id)`, Route: "PATCH /projects/{id}/packages/{pkg}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			plan, err := draftPlan(ctx, d, who, project)
			if err != nil {
				return sent{}, err
			}
			retired := true
			body := map[string]any{"retired": retired, "version": p.Version}
			if plan.ID != "" {
				body["remove_cost_lines"], body["cost_plan_id"], body["cost_plan_version"] = true, plan.ID, plan.Version
			}
			return sent{Path: map[string]string{"pkg": p.ID}, Body: body}, nil
		},
		Done: func(map[string]any) string { return "Removed the package and its draft cost lines." }},
	{Name: "procurement.invite_firm", Undo: true, About: "Name a firm on a package before its quote arrives.",
		Fields: `package (id); name (the firm)`, Route: "POST /projects/{id}/procurement/packages/{pkg}/invitations",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			if err := need(f, "name"); err != nil {
				return sent{}, err
			}
			f["_title"] = p.Title
			return sent{Path: map[string]string{"pkg": p.ID}, Body: map[string]any{"name": fieldText(f, "name")}}, nil
		},
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Invited %s to %s.", fieldText(f, "name"), fieldText(f, "_title"))
		}},
	{Name: "procurement.remove_invitation", Undo: true, About: "Take an invited firm that has not quoted off a package.",
		Fields: `package (id); firm (its name)`, Route: "DELETE /projects/{id}/procurement/packages/{pkg}/invitations/{firm}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			id, sub, shown, err := firmOn(ctx, d, who, project, p.ID, fieldText(f, "firm"))
			if err != nil {
				return sent{}, err
			}
			if sub != "" {
				return sent{}, refuse("firm", "%s has quoted, so it is not an invitation to remove.", shown)
			}
			f["_title"], f["firm"] = p.Title, shown
			return sent{Path: map[string]string{"pkg": p.ID, "firm": id}}, nil
		},
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Removed %s from %s.", fieldText(f, "firm"), fieldText(f, "_title"))
		}},
	{Name: "procurement.rename_invitation", About: "Correct the name of an invited firm.",
		Fields: `package (id); firm (its name now); name (the right name)`, Route: "PUT /projects/{id}/procurement/packages/{pkg}/invitations/{firm}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			id, _, _, err := firmOn(ctx, d, who, project, p.ID, fieldText(f, "firm"))
			if err != nil {
				return sent{}, err
			}
			return sent{Path: map[string]string{"pkg": p.ID, "firm": id}, Body: map[string]any{"name": fieldText(f, "name")}}, nil
		}},
	{Name: "procurement.award", Click: true, About: "award a package to a firm the PM named.",
		Fields: `package (id); firm (its name)`, Route: "POST /projects/{id}/procurement/packages/{pkg}/award",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			// The lines an award writes are chosen by the PM in the award
			// dialog; chat does not choose them. The click takes them there.
			p, err := pkgOf(ctx, d, who, project, f)
			if err != nil {
				return sent{}, err
			}
			return sent{}, refuse("", "An award's lines are chosen in the award dialog. Open Procurement, %s, and press Award beside the firm.", p.Title)
		}},
	{Name: "procurement.undo_award", Click: true, About: "undo a package's award.",
		Fields: `package (id)`, Route: "DELETE /projects/{id}/procurement/packages/{pkg}/award",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			return sent{Path: map[string]string{"pkg": p.ID}}, err
		}},
	{Name: "procurement.sign_off", Click: true, About: "sign off a package's comparison report with a recommendation.",
		Fields: `package (id); text (the recommendation, the PM's words)`, Route: "PUT /projects/{id}/procurement/packages/{pkg}/report",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			return sent{Path: map[string]string{"pkg": p.ID}, Body: map[string]any{"text": fieldText(f, "text"), "sign_off": true}}, err
		}},
	{Name: "procurement.create_report", About: "Prepare the RFP or RFT draft for a package.",
		Fields: `package (id)`, Route: "POST /projects/{id}/procurement/create",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			p, err := pkgOf(ctx, d, who, project, f)
			return sent{Body: map[string]any{"package_id": p.ID}}, err
		}},

	// ---- Profile
	{Name: "profile.set_scope", About: "Say whether building systems are in or out of scope.",
		Fields: `systems (an object: system id to "in", "out" or null to clear; ids are from read profile's scope list, e.g. site.landscaping)`,
		Route:  "PUT /projects/{id}/profile/scope"},
	{Name: "profile.accommodation", Undo: true, About: "Edit lines of the accommodation schedule.",
		Fields:  `op (set|reset|delete|split|duplicate); lines (line ids from read accommodation); set: field (name|level|area|comment), value`,
		Figures: map[string]string{"value": "value"},
		Route:   "POST /projects/{id}/profile/accommodation"},
	{Name: "profile.set_planning", About: "Record a planning value (gross floor area, construction duration and the like) with why.",
		Fields: `key (without plan., e.g. gross_floor_area); value; rationale (the PM's reason)`, Figures: map[string]string{"value": "value"},
		Route: "PUT /projects/{id}/planning/{key}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			key := strings.TrimPrefix(take(f, "key"), "plan.")
			live, err := d.Store.PlanningValues(ctx, who.OrgID, project, false)
			if err != nil {
				return sent{}, err
			}
			f["origin"], f["meaning"] = "user", "stated"
			for _, v := range live {
				if v.Key == key {
					f["version"] = v.Version
				}
			}
			return sent{Path: map[string]string{"key": key}, Body: f}, nil
		}},
	{Name: "profile.clear_planning", About: "Withdraw a planning value; it is kept as history.",
		Fields: `key`, Route: "DELETE /projects/{id}/planning/{key}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			key := strings.TrimPrefix(take(f, "key"), "plan.")
			live, err := d.Store.PlanningValues(ctx, who.OrgID, project, false)
			if err != nil {
				return sent{}, err
			}
			for _, v := range live {
				if v.Key == key {
					return sent{Path: map[string]string{"key": key}, Query: url.Values{"version": {strconv.FormatInt(v.Version, 10)}, "part_id": {v.PartID}}}, nil
				}
			}
			return sent{}, refuse("key", "There is no planning value %q to withdraw.", key)
		}},
	{Name: "profile.add_part", About: "Add a part of the building (a stage, a block, a tenancy).",
		Fields: `label; kind (as read profile lists them); ncc_class (optional)`, Route: "POST /projects/{id}/parts"},
	{Name: "profile.edit_part", About: "Rename or reclassify a part of the building.",
		Fields: `part (id); label; kind; ncc_class`, Route: "PATCH /projects/{id}/parts/{part}"},
	{Name: "profile.add_work_item", About: "Add a work item to the scope.",
		Fields: `part_id; system_id; action (new|upgrade|replace|retain|remove); title`, Route: "POST /projects/{id}/works"},
	{Name: "profile.edit_work_item", About: "Change a work item.",
		Fields: `wi (id); version; title; action; the other fields the Works screen sends`, Route: "PATCH /projects/{id}/works/{wi}"},
	{Name: "profile.retire_work_item", About: "Retire a work item.",
		Fields: `wi (id); the fields the Works screen sends`, Route: "POST /projects/{id}/works/{wi}/retire"},
	{Name: "profile.accept_implication", Click: true, About: "accept an implication: you are saying it applies and should be acted on.",
		Fields: `key (from read implications); action (optional)`, Route: "POST /projects/{id}/proposals/{key}/accept", Send: implicationSend},
	{Name: "profile.dismiss_implication", Click: true, About: "dismiss an implication: you are saying it does not apply.",
		Fields: `key (from read implications); rationale (the PM's reason)`, Route: "POST /projects/{id}/proposals/{key}/dismiss", Send: implicationSend},
	{Name: "profile.reopen_implication", About: "Undo a decision on an implication.",
		Fields: `key`, Route: "DELETE /projects/{id}/proposals/{key}/decision", Send: implicationSend},

	// ---- Program
	{Name: "program.accept_readings", Click: true, About: "accept proposed Program rows or document readings into the program.",
		Fields: `ids (the proposed rows or readings), or all: true`, Route: "POST /projects/{id}/program/accept",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			st, err := loadProgramForOps(ctx, d, who, project)
			f["version"] = st.Program.Version
			return sent{Body: f}, err
		}},
	{Name: "program.dismiss_readings", About: "Dismiss proposed document readings on the Program.",
		Fields: `the fields the Program screen sends`, Route: "POST /projects/{id}/program/readings/dismiss"},
	{Name: "program.decide_changes", Click: true, About: "accept or decline changes the Program proposes after its inputs changed.",
		Fields: `the fields the Program screen sends`, Route: "POST /projects/{id}/program/changes/decide"},

	// ---- Cost Plan: the writes the first slice left out. All are covered by Undo.
	{Name: "cost_plan.reorder", Undo: true, About: "Put cost lines in a new order.",
		Fields: `ids (every line of one section, in the new order)`, Route: "PUT /projects/{id}/cost-plan/order", Send: withPlan(nil)},
	{Name: "cost_plan.subdivide", Undo: true, About: "Split one line into a group of lines.",
		Fields: `line (id); parts (the screen's list: label and amount each)`, Figures: map[string]string{"amount": "money"},
		Route: "POST /projects/{id}/cost-plan/items/{item}/subdivide", Send: withPlan(nil)},
	{Name: "cost_plan.settings", Undo: true, About: "Set the plan's GST basis, GST rate, price date or coverage note.",
		Fields: `currency (AUD); tax_basis (ex_tax|inc_tax); tax_rate (0.1 is 10%); price_date; coverage`, Figures: map[string]string{"tax_rate": "value", "price_date": "date"},
		Route: "PUT /projects/{id}/cost-plan", Send: withPlan(nil)},
	{Name: "cost_plan.accept_suggestions", Undo: true, About: "Add the cost lines SiteWise suggests: one by its key, or all.",
		Fields: `key (one suggestion), or none for all`, Route: "POST /projects/{id}/cost-plan/suggestions/accept",
		Also: []string{"POST /projects/{id}/cost-plan/suggestions/{key}/accept"},
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			key := take(f, "key")
			s, err := withPlan(nil)(ctx, d, who, project, f)
			if key != "" {
				s.Route, s.Path = "POST /projects/{id}/cost-plan/suggestions/{key}/accept", map[string]string{"key": key}
			}
			return s, err
		}},
	{Name: "cost_plan.dismiss_suggestion", Undo: true, About: "Dismiss a suggested cost line, or bring a dismissed one back.",
		Fields: `key; restore (true brings it back)`, Route: "POST /projects/{id}/cost-plan/suggestions/{key}/dismiss",
		Also: []string{"DELETE /projects/{id}/cost-plan/suggestions/{key}/dismiss"},
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			key := take(f, "key")
			restore, _ := f["restore"].(bool)
			delete(f, "restore")
			s, err := withPlan(nil)(ctx, d, who, project, f)
			s.Path = map[string]string{"key": key}
			if restore {
				s.Route = "DELETE /projects/{id}/cost-plan/suggestions/{key}/dismiss"
			}
			return s, err
		}},
	{Name: "cost_plan.link_scope", Undo: true, About: "Link or unlink a package scope item and a cost line.",
		Fields: `the fields the Cost Plan screen sends`, Route: "PUT /projects/{id}/cost-plan/links", Send: withPlan(nil)},
	{Name: "cost_plan.apply_benchmark", Undo: true, About: "Set a line's budget from one of its benchmarks.",
		Fields: `line (id); the benchmark fields the screen sends`, Route: "POST /projects/{id}/cost-plan/items/{item}/benchmarks", Send: withPlan(nil)},
	{Name: "cost_plan.prepare", Undo: true, About: "Fill the cost plan with the lines SiteWise proposes from the packages and scope.",
		Fields: `(none)`, Route: "POST /projects/{id}/cost-plan/prepare"},

	// ---- Invoices
	{Name: "invoices.edit", About: "Correct an invoice's fields, or void, restore or mark it as replacing another.",
		Fields: `invoice (id from read invoices); fields (an object of field to value, null clears); action (void|restore|keep_both|replaces); target (the other invoice for replaces)`,
		Route:  "PATCH /projects/{id}/invoices/{inv}", Send: renamePath("invoice", "inv")},
	{Name: "invoices.allocate", About: "Split an invoice across cost lines.",
		Fields: `invoice (id); shares (the screen's list: a cost line and an amount each)`, Figures: map[string]string{"amount": "money"},
		Route: "PUT /projects/{id}/invoices/{inv}/allocations", Send: renamePath("invoice", "inv")},
	{Name: "invoices.confirm_supplier", Click: true, About: "confirm who an invoice is from.",
		Fields: `invoice (id); fields (an object with the supplier fields as the Invoices screen sends them)`,
		Route:  "PATCH /projects/{id}/invoices/{inv}", Send: renamePath("invoice", "inv")},

	// ---- Delivery records
	{Name: "delivery.add", Undo: true, About: "Add a register row: an RFI (saved as Draft; nothing is sent), a notice, a risk, an approval, a decision, an issue or an action.",
		Fields:  `kind (rfi|notice|risk|approval|decision|issue|action); title (an RFI's subject); owner_text (who it is with; leave it out for an RFI to a package and SiteWise fills the appointed firm); package (a package id or title); system (a top-level system id, e.g. envelope); status; for an RFI: question (its wording), ref (the sender's number), raised_by, raised_on (only a date the PM gave); target_date (only a date the PM gave; SiteWise works out an RFI's due date from raised_on)`,
		Figures: map[string]string{"target_date": "date", "forecast_date": "date", "actual_date": "date", "baseline_date": "date", "raised_on": "date"},
		Route:   "POST /projects/{id}/delivery", Send: sendRegisterAdd,
		Done: func(f map[string]any) string { return "Added " + quoted(fieldText(f, "title")) + " to the register." }},
	{Name: "delivery.edit", Undo: true, About: "Change a register row: close, withdraw or reopen it, set who it is with, its due date or its system, edit an RFI's wording, dismiss possible answers; for a consent condition set its gate, who acts and whether evidence is needed, link a document as its evidence, or mark it not applicable.",
		Fields:  `item (a row id from read delivery, an RFI number such as RFI-004, or a condition number such as condition 46); status (an RFI: draft|open|answered|closed|withdrawn; a notice: open|closed|withdrawn; a condition: open|evidence_filed|not_applicable, never satisfied, which is delivery.satisfy); title; owner_text; target_date; system; package; question; ref; raised_on; answer; for a condition: gate (before_cc|before_work|during_works|before_occupation|before_subdivision|ongoing|throughout), discipline (who acts, a discipline id), evidence_needed (true|false), link_evidence (a document id from documents: its evidence); dismiss_readings (reading ids from read delivery); retired`,
		Figures: map[string]string{"target_date": "date", "forecast_date": "date", "actual_date": "date", "baseline_date": "date", "raised_on": "date"},
		Route:   "PATCH /projects/{id}/delivery/{item}", Send: sendRegisterEdit,
		Done: func(f map[string]any) string { return "Changed the register row." }},
	{Name: "delivery.accept", Click: true, About: "accept a possible answer: you vouch that the document answers the RFI.",
		Fields: `item (the RFI's id or number); reading (the possible answer's id from read delivery)`, Route: "PATCH /projects/{id}/delivery/{item}",
		Send: sendRegisterAccept, Card: registerAcceptCard,
		Done: func(f map[string]any) string { return "Accepted the answer." }},
	{Name: "delivery.settings", Undo: true, About: "Set the project's RFI response time in working days.",
		Fields: `rfi_response_days (a whole number of working days, 1 to 60, as the PM said it)`, Figures: map[string]string{"rfi_response_days": "number"},
		Route: "PUT /projects/{id}/delivery/settings",
		Done: func(f map[string]any) string {
			return fmt.Sprintf("Set the RFI response time to %v working days.", f["rfi_response_days"])
		}},
	{Name: "delivery.verify", Click: true, About: "mark a delivery record as verified by you.",
		Fields: `item (id); version`, Route: "PATCH /projects/{id}/delivery/{item}",
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			return sent{Path: map[string]string{"item": take(f, "item")}, Body: map[string]any{"version": f["version"], "review_status": "verified"}}, nil
		}},
	{Name: "delivery.link", About: "Say one delivery record depends on another, or remove that.",
		Fields: `remove (true removes); the fields the screen sends`, Route: "POST /projects/{id}/delivery/dependencies",
		Also: []string{"DELETE /projects/{id}/delivery/dependencies"},
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			s := sent{Body: f}
			if remove, _ := f["remove"].(bool); remove {
				s.Route = "DELETE /projects/{id}/delivery/dependencies"
			}
			delete(f, "remove")
			return s, nil
		}},

	// ---- Reports
	{Name: "reports.create", About: "Create a report draft.",
		Fields: `kind; package_id (for a package's report)`, Route: "POST /projects/{id}/reports"},

	// ---- Documents
	{Name: "documents.correct_field", About: "Correct how a document is filed: its kind, discipline, title, number or revision.",
		Fields: `document (id from documents); field (kind|discipline|title|number|revision); value`,
		Route:  "PUT /documents/{id}/fields/{field}",
		Send: func(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
			views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project)
			if err != nil {
				return sent{}, err
			}
			ids := make([]string, len(views))
			for i, v := range views {
				ids[i] = v.ID
			}
			id, err := byRef(ids, func(s string) string { return s }, fieldText(f, "document"), "document")
			if err != nil {
				return sent{}, err
			}
			return sent{Path: map[string]string{"id": id, "field": fieldText(f, "field")}, Body: map[string]any{"value": f["value"]}}, nil
		}},

	// ---- Specifications
	{Name: "specifications.edit", About: "A direct edit to the specification schedule.",
		Fields: `what (cells|types|applications|assertions|relationships|coverage); type, application (ids, when editing one); body (what the Specifications screen sends for that edit)`,
		Route:  "POST /projects/{id}/specifications/cells",
		Also: []string{"POST /projects/{id}/specifications/types", "PUT /projects/{id}/specifications/types/{type}", "POST /projects/{id}/specifications/types/{type}/link",
			"POST /projects/{id}/specifications/applications", "PUT /projects/{id}/specifications/applications/{application}",
			"POST /projects/{id}/specifications/assertions", "POST /projects/{id}/specifications/relationships", "POST /projects/{id}/specifications/coverage"},
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			typ, app := fieldText(f, "type"), fieldText(f, "application")
			s := sent{Body: f["body"]}
			switch what := fieldText(f, "what"); {
			case what == "cells" || what == "assertions" || what == "relationships" || what == "coverage":
				s.Route = "POST /projects/{id}/specifications/" + what
			case what == "types" && typ == "":
				s.Route = "POST /projects/{id}/specifications/types"
			case what == "types":
				s.Route, s.Path = "PUT /projects/{id}/specifications/types/{type}", map[string]string{"type": typ}
			case what == "link" && typ != "":
				s.Route, s.Path = "POST /projects/{id}/specifications/types/{type}/link", map[string]string{"type": typ}
			case what == "applications" && app == "":
				s.Route = "POST /projects/{id}/specifications/applications"
			case what == "applications":
				s.Route, s.Path = "PUT /projects/{id}/specifications/applications/{application}", map[string]string{"application": app}
			default:
				return sent{}, refuse("what", "what is cells, types, link, applications, assertions, relationships or coverage.")
			}
			return s, nil
		}},
	{Name: "specifications.adopt", Click: true, About: "adopt a specification type from a consultant's document into the schedule.",
		Fields: `type (id); body (what the screen sends)`, Route: "POST /projects/{id}/specifications/types/{type}/adopt",
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			return sent{Path: map[string]string{"type": fieldText(f, "type")}, Body: f["body"]}, nil
		}},
	{Name: "specifications.review", Click: true, About: "mark a specification value as reviewed by you.",
		Fields: `assertion (id); body (what the screen sends)`, Route: "POST /projects/{id}/specifications/assertions/{assertion}/review",
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			return sent{Path: map[string]string{"assertion": fieldText(f, "assertion")}, Body: f["body"]}, nil
		}},
	{Name: "specifications.issue", Click: true, About: "issue the specification: it is frozen and leaves the building.",
		Fields: `body (what the screen sends)`, Route: "POST /projects/{id}/specifications/issues",
		Send: func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
			return sent{Body: f["body"]}, nil
		}},
}

func renamePath(field, name string) sender {
	return func(_ context.Context, _ Deps, _ opActor, _ string, f map[string]any) (sent, error) {
		if err := need(f, field); err != nil {
			return sent{}, err
		}
		return sent{Path: map[string]string{name: take(f, field)}, Body: f}, nil
	}
}

// implicationSend fills the fingerprint the decision is pinned to, so a
// decision made on a stale reading is refused by the handler as it is for
// the screen.
func implicationSend(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
	key := take(f, "key")
	items, err := d.Store.ReadProposals(ctx, who.OrgID, project)
	if err != nil {
		return sent{}, err
	}
	for _, it := range items {
		if it.Key == key {
			f["inputs_fingerprint"] = it.InputsFingerprint
			return sent{Path: map[string]string{"key": key}, Body: f}, nil
		}
	}
	return sent{}, refuse("key", "There is no implication with key %q. Use a key from read implications.", key)
}
