package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"sitewise/internal/costs"
)

// The read areas beyond the four screens chat can change (review change 1):
// what SiteWise already knows, read-only. Chat may show records that have
// no screen yet (owner, 15 November).

func readMore(ctx context.Context, d Deps, who opActor, project, area string, r *reader) error {
	switch area {
	case "implications":
		return readImplications(ctx, d, who, project, r)
	case "attention":
		return readAttention(ctx, d, who, project, r)
	case "gaps":
		items, err := d.Store.ReadGaps(ctx, who.OrgID, project)
		if err != nil {
			return err
		}
		r.line("Package gap check: work items and who is responsible for them. A state of unassigned is a gap.")
		return compact(r, items, 0)
	case "delivery":
		items, err := d.Store.ReadDelivery(ctx, who.OrgID, project)
		if err != nil {
			return err
		}
		r.outside = "delivery records, some of which were read from documents"
		r.line("Delivery records: risks, approvals, decisions and actions. A record marked addressed means a document states it; say \"a document states…\".")
		return compact(r, items, 0)
	case "invoices":
		v, err := d.Store.InvoiceLedger(ctx, who.OrgID, project, time.Now().Format("2006-01-02"))
		if err != nil {
			return err
		}
		r.outside = "invoices, which are suppliers' documents"
		r.line("Invoice ledger: %d invoices. Amounts are as the ledger holds them; the checks are SiteWise's.", len(v.Invoices))
		return compact(r, map[string]any{"summary": v.Summary, "invoices": v.Invoices, "flags": v.Flags, "status": v.Status}, 0)
	case "specifications":
		snap, err := d.Store.ReadSpecifications(ctx, who.OrgID, project)
		if err != nil {
			return err
		}
		r.outside = "the specification workspace, which holds text from documents"
		r.line("Specifications workspace.")
		return compact(r, snap, 0)
	}
	return refuse("area", "Areas are %s.", strings.Join(readAreas, ", "))
}

// noisy are keys a reader of the record does not need.
var noisy = map[string]bool{"provenance": true, "version": true, "inputs_fingerprint": true, "knowledge_version": true, "org_id": true,
	"project_id": true, "created_at": true, "updated_at": true, "projection_hash": true, "source_key": true}

// compact writes a store read as trimmed JSON: no empty values and none of
// the bookkeeping keys. It is the plain form for records with no chat
// renderer of their own yet.
func compact(r *reader, v any, _ int) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var x any
	if err := json.Unmarshal(raw, &x); err != nil {
		return err
	}
	var trim func(any) any
	trim = func(x any) any {
		switch n := x.(type) {
		case map[string]any:
			for k, c := range n {
				c = trim(c)
				if noisy[k] || c == nil || c == "" {
					delete(n, k)
					continue
				}
				n[k] = c
			}
			if len(n) == 0 {
				return nil
			}
		case []any:
			out := n[:0]
			for _, c := range n {
				if c = trim(c); c != nil {
					out = append(out, c)
				}
			}
			if len(out) == 0 {
				return nil
			}
			return out
		case string:
			if uuidPattern.MatchString(n) {
				return shortID(n)
			}
		}
		return x
	}
	x = trim(x)
	if x == nil {
		r.line("Nothing is recorded.")
		return nil
	}
	out, _ := json.Marshal(x)
	r.line("%s", out)
	return nil
}

// readImplications renders the proposals table in Go's own order: rank,
// severity, the critical flag, and whether each is open, accepted or
// dismissed.
func readImplications(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	items, err := d.Store.ReadProposals(ctx, who.OrgID, project)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		r.line("Implications: none are recorded for this project.")
		return nil
	}
	r.line("Implications, in SiteWise's order (rank). They come from a draft checklist applied to this project's profile; they are prompts to check, not findings. \"Addressed\" means a document states it.")
	open := 0
	for _, it := range items {
		state := "open"
		if it.Decision != nil {
			state = it.Decision.Decision
		} else {
			open++
		}
		parts := []string{fmt.Sprintf("%d", it.Rank), it.Label, "severity " + orDash(it.Severity), state}
		if !r.plain {
			parts = append(parts, "key "+it.Key)
		}
		if it.Critical {
			parts = append(parts, "critical")
		}
		if it.Draft {
			parts = append(parts, "draft checklist")
		}
		// Rule and clause numbers behind an implication are not printed
		// here: chat cannot yet show each with its verified flag, and an
		// unverified number must never read as a verified one.
		r.line("  %s", strings.Join(parts, " | "))
	}
	r.line("%d open of %d.", open, len(items))
	return nil
}

// readAttention is one Go rollup of what needs attention, from explicit
// states only (docs/design/2026-11-12-project-report.md, section 4 row 2):
// open critical implications, cost sections where the contract exceeds the
// budget, packages with firms invited and no quote, documents that could
// not be read, and Program rows with no duration. Each line names its
// source. It ranks nothing beyond the order below.
func readAttention(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	r.line("What needs attention, from SiteWise's records only, each with its source. This is a list of explicit states, not a judgment of risk.")
	n := 0
	say := func(source, format string, args ...any) {
		n++
		r.line("  %d. %s (source: %s)", n, fmt.Sprintf(format, args...), source)
	}
	if items, err := d.Store.ReadProposals(ctx, who.OrgID, project); err == nil {
		sort.SliceStable(items, func(i, j int) bool { return items[i].Rank < items[j].Rank })
		shown := 0
		for _, it := range items {
			if it.Decision != nil || shown == 8 {
				continue
			}
			if it.Critical || it.Severity == "high" {
				shown++
				flag := "severity " + it.Severity
				if it.Critical {
					flag = "critical"
				}
				say("Implications, draft checklist", "Open implication, %s: %s", flag, it.Label)
			}
		}
	}
	if p, err := d.Store.ReadCostPlan(ctx, who.OrgID, project, ""); err == nil && len(p.Items) > 0 {
		if totals, err := costs.Summarize(p, "overall"); err == nil {
			for _, section := range sectionOrder {
				t := totals.Sections[section]
				b, c := t["budget"], t["commitment"]
				if costs.Compare(c.KnownSubtotal, b.KnownSubtotal) > 0 && costs.Compare(b.KnownSubtotal, "0") > 0 {
					over := costs.Sub(c.KnownSubtotal, b.KnownSubtotal)
					say("Cost Plan", "%s: contract %s is over budget %s by %s", sectionNames[section],
						r.fig(plainMoney(&c.KnownSubtotal), dollars(c.KnownSubtotal), sectionNames[section]+", contract", "Cost Plan"),
						r.fig(plainMoney(&b.KnownSubtotal), dollars(b.KnownSubtotal), sectionNames[section]+", budget", "Cost Plan"),
						r.fig(plainMoney(&over), dollars(over), sectionNames[section]+", contract less budget", "Cost Plan"))
				}
			}
			if u := totals.Overall["budget"].Unknown; u > 0 {
				say("Cost Plan", "%d cost lines have no budget amount", u)
			}
		}
	}
	if pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project); err == nil {
		board, _, _ := d.Store.ProcurementBoard(ctx, who.OrgID, project)
		for _, p := range pkgs {
			if p.RetiredAt != nil {
				continue
			}
			invited, quoted, review := 0, 0, 0
			for _, c := range board[p.ID] {
				if c.SubmissionID == "" {
					invited++
				} else {
					quoted++
				}
				if c.NeedsReview {
					review++
				}
			}
			if invited > 0 && quoted == 0 {
				say("Procurement", "%s: %d firms invited and no quote received", p.Title, invited)
			}
			if review > 0 {
				say("Procurement", "%s: %d quotes have readings to check", p.Title, review)
			}
		}
	}
	if views, err := d.Store.ProjectDocumentViews(ctx, who.OrgID, project); err == nil {
		failed := 0
		for _, v := range views {
			if v.Status == "not_filed" || v.Status == "failed" {
				failed++
			}
		}
		if failed > 0 {
			say("Documents", "%d documents could not be filed or read", failed)
		}
	}
	if st, err := d.Store.LoadProgram(ctx, who.OrgID, project); err == nil {
		unknown := 0
		for _, a := range st.Activities {
			if a.Kind == "activity" && a.DurationDays == nil {
				unknown++
			}
		}
		if unknown > 0 {
			say("Program", "%d Program rows have no duration, so the finish date is a lower bound", unknown)
		}
	}
	if n == 0 {
		r.line("  Nothing is flagged by the records. That is not a statement that the project is low risk: it means no explicit state is flagged.")
	}
	r.line("The Program holds no progress and there is no inbox, so nothing here says what is late or which emails are unanswered.")
	return nil
}

// readTender renders one package's tender comparison: each firm's column,
// its lines and exclusions, and the adds and subs. These are the firms'
// words, so the turn is marked as having read outside text.
func readTender(ctx context.Context, d Deps, who opActor, project, ref string, r *reader) error {
	pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
	if err != nil {
		return err
	}
	pkg, err := packageByRef(pkgs, ref)
	if err != nil {
		return err
	}
	t, err := d.Store.ReadPriceTable(ctx, who.OrgID, project, pkg.ID)
	if err != nil {
		return err
	}
	firm := map[string]string{}
	var names []string
	for _, c := range t.Columns {
		name := "a firm"
		if c.Firm != nil {
			name = c.Firm.Name
		}
		firm[c.Submission.ID] = name
		names = append(names, name)
	}
	r.outside = "the tenders for " + pkg.Title
	if len(names) > 0 {
		r.outside = "the tender of " + strings.Join(names, ", ")
	}
	r.line("Tender comparison for %s (package %s). Amounts are ex GST. The lines are each firm's own words.", pkg.Title, shortID(pkg.ID))
	if t.Tender == nil || len(t.Tender.Columns) == 0 {
		for _, c := range t.Columns {
			r.line("  %s | column %s | %s | no tender comparison has been read for it", firm[c.Submission.ID], shortID(c.Submission.ID), c.Submission.Status)
		}
		if len(t.Columns) == 0 {
			r.line("  No firm has quoted.")
		}
		return nil
	}
	row := map[string]string{}
	var rows []string
	for _, tr := range t.Tender.Rows {
		row[tr.Key] = tr.Label
		rows = append(rows, tr.Key+" ("+tr.Label+")")
	}
	r.line("Rows a line can be moved to: %s", strings.Join(rows, "; "))
	for _, c := range t.Tender.Columns {
		name := firm[c.SubmissionID]
		checked := "not checked"
		if c.Checked {
			checked = "checked"
		}
		sum := "tender sum unknown"
		if m := centsMoney(c.Sum); m != nil {
			sum = "tender sum " + r.fig(plainMoney(m), dollars(*m), name+", tender sum", "Procurement")
		}
		if m := centsMoney(c.LikeForLike); m != nil {
			sum += " | like for like " + r.fig(plainMoney(m), dollars(*m), name+", like for like", "Procurement")
		}
		r.line("%s | column %s | %s | %s", name, shortID(c.SubmissionID), sum, checked)
		for _, l := range c.Lines {
			amount := "no amount"
			if m := centsMoney(l.Cents); m != nil {
				amount = plainMoney(m)
			}
			counted := ""
			if !l.Counted {
				counted = " | not counted"
			}
			r.line("    line %s | %s | %s | row %s | page %d%s", shortID(l.ID), l.Label, amount, orDash(row[l.Row]), l.Page, counted)
		}
		for _, e := range c.Exclusions {
			r.line("    exclusion (page %d): %s", e.P, e.Text)
		}
	}
	for _, a := range t.Tender.Adjustments {
		var amounts []string
		for sub, amt := range a.Amounts {
			if m := centsMoney(amt.Cents); m != nil {
				amounts = append(amounts, firm[sub]+" "+plainMoney(m)+" ("+amt.State+")")
			}
		}
		sort.Strings(amounts)
		r.line("  adjustment %s | %s | %s", shortID(a.ID), a.Label, strings.Join(amounts, "; "))
	}
	return nil
}

// getChatRead is GET /projects/{id}/chat/read?area=…: an area exactly as
// chat reads it, in the plain form (no ids, no tags). The command key
// compares it before and after a turn; it is also how to see what the
// agent sees. It calls no model.
func getChatRead(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, false)
	if !ok {
		return
	}
	who, project := opActor{s.OrgID, s.UserID}, r.PathValue("id")
	rd := &reader{plain: true}
	var err error
	switch r.URL.Query().Get("area") {
	case "outline":
		// The outline exactly as a turn opens with it.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(chatOutline(r.Context(), d, who, project, "", "")))
		return
	case "cost_plan":
		err = readCostPlan(r.Context(), d, who, project, rd)
	case "profile":
		err = readProfile(r.Context(), d, who, project, rd)
	case "accommodation":
		err = chatAccommodation(withOpRequest(r.Context(), r), d, project, rd)
	case "program":
		err = readProgram(r.Context(), d, who, project, rd, false)
	case "procurement":
		err = readProcurement(r.Context(), d, who, project, rd)
	default:
		http.Error(w, "area is cost_plan, profile, accommodation, program, procurement or outline", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		d.Log.Printf("chat read failed: %v", err)
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(rd.b.String()))
}
