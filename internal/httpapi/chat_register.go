package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"sitewise/internal/delivery"
	"sitewise/internal/reports"
	"sitewise/internal/store"
)

// The register from chat (docs/plans/2026-11-17-rfis-review-comments.md,
// "From chat"). The read prints Go's figures: numbers, due dates, days late
// and who holds a row. The operations run the register's own routes.

var kindWords = map[string]string{"rfi": "RFI", "notice": "Notice", "condition": "Condition", "action": "Action", "risk": "Risk", "issue": "Issue", "decision": "Decision", "approval": "Approval", "milestone": "Hold point"}

// rfiNumber is "RFI-007".
func rfiNumber(n *int) string {
	if n == nil {
		return ""
	}
	return fmt.Sprintf("RFI-%03d", *n)
}

// readRegister renders the register for chat. part narrows it: a kind
// ("rfi"), a top-level system id ("envelope"), or "overdue".
func readRegister(ctx context.Context, d Deps, who opActor, project, part string, r *reader) error {
	q := store.RegisterQuery{}
	overdue, gate := false, ""
	switch {
	case part == "":
	case part == "overdue":
		overdue = true
	case delivery.GateWords[part] != "":
		// A gate: the conditions met before it.
		q.Kind, gate = "condition", part
	case len(delivery.Statuses(part)) > 0:
		q.Kind = part
	default:
		q.System = part
	}
	view, err := d.Store.ReadRegister(ctx, who.OrgID, project, q, todaySydney())
	if err != nil {
		if errors.Is(err, store.ErrInvalidDelivery) {
			return refuse("part", "part is a kind (rfi, notice, risk…), a top-level system id, or overdue.")
		}
		return err
	}
	labels := map[string]string{}
	for _, s := range view.Systems {
		labels[s.ID] = s.Label
	}
	typical := ""
	if view.ResponseTypical {
		typical = " (typical; the PM has not set the project's own)"
	}
	cal := "the Program's calendar"
	if view.Calendar == "weekdays" {
		cal = "Monday to Friday (no Program)"
	}
	r.line("The register: what is owed on the project, from SiteWise's records. Today is %s. RFI response time %d working days%s, on %s. Numbers, due dates, days late and With are computed by SiteWise; With is who holds the row now (You when an RFI is answered).", view.Today, view.ResponseDays, typical, cal)
	consentHeader(view, r)
	if len(view.Kinds) == 0 {
		r.line("Nothing is recorded as owed yet.")
		return nil
	}
	var kinds []string
	for _, k := range view.Kinds {
		kinds = append(kinds, fmt.Sprintf("%s %d (%d open)", strings.ToLower(kindWords[k.Kind]), k.Rows, k.Open))
	}
	r.line("Kinds with rows: %s.", strings.Join(kinds, ", "))
	r.line("Columns: id | kind | No. | item | with | due | late | status | system | possible answers (reading id: document)")
	shown := 0
	for _, it := range view.Items {
		late := 0
		if it.LateDays != nil {
			late = *it.LateDays
		}
		if overdue && !reports.RFIOverdue(reports.AttentionRecord{Kind: it.Kind, Status: it.Status, LateDays: late}) {
			continue
		}
		if gate != "" && delivery.Condition(it.Content).Stage != gate {
			continue
		}
		shown++
		due := "—"
		if it.TargetDate != nil {
			due = *it.TargetDate
		}
		lateText := "—"
		if late > 0 {
			lateText = fmt.Sprintf("%d working days late", late)
		}
		var answers []string
		for _, rd := range it.Readings {
			answers = append(answers, shortID(rd.ID)+": "+rd.Document)
			r.outside = "the titles of filed documents offered as possible answers"
		}
		number, status := rfiNumber(it.Number), strings.ReplaceAll(it.Status, "_", " ")
		if it.Kind == "condition" {
			// The printed heading is the council's words.
			r.outside = "consent conditions' printed headings"
			number, due, status = conditionColumns(it)
		}
		r.line("  %s| %s | %s | %s | %s | %s | %s | %s | %s | %s", r.id(it.ID), kindWords[it.Kind], orDash(number), it.Title, orDash(it.With), due, lateText,
			status, orDash(labels[it.SystemID]), orDash(strings.Join(answers, "; ")))
	}
	if shown == 0 {
		r.line("No row matches.")
	}
	if a := view.AlsoOpen; a != nil {
		line := fmt.Sprintf("Also open on %s: %d specification rows with an unknown inclusion or undecided selection; %d tender exclusions on trade packages that build it", a.Label, a.SpecUnknowns, a.Exclusions)
		if a.MainWorksExclusions > 0 {
			line += fmt.Sprintf("; Main Works lists %d exclusions, not sorted by system", a.MainWorksExclusions)
		}
		r.line("%s.", line)
	}
	return nil
}

// Named filters on the register table. Overdue is reports.RFIOverdue, the
// same definition the attention list uses.
func registerFilter(keep func(it delivery.Item, who opActor) bool) func(context.Context, Deps, opActor, string) ([]string, error) {
	return func(ctx context.Context, d Deps, who opActor, project string) ([]string, error) {
		items, err := d.Store.ReadDelivery(ctx, who.OrgID, project)
		if err != nil {
			return nil, err
		}
		var ids []string
		for _, it := range items {
			if keep(it, who) {
				ids = append(ids, it.ID)
			}
		}
		return ids, nil
	}
}

func registerOverdue(it delivery.Item, _ opActor) bool {
	return reports.RFIOverdue(attentionRecord(it, ""))
}

// registerWithMe is what is the PM's move: answered RFIs, and the drafts
// the PM raised.
func registerWithMe(it delivery.Item, who opActor) bool {
	if it.Kind != "rfi" {
		return false
	}
	if it.Status == "answered" {
		return true
	}
	var p struct {
		Actor string `json:"actor"`
	}
	_ = json.Unmarshal(it.Provenance, &p)
	return it.Status == "draft" && p.Actor == who.UserID
}

func registerToCheck(it delivery.Item, _ opActor) bool { return delivery.ToCheck(it) }

// conditionOpenBefore is the named filter for one gate, by the shared rule.
func conditionOpenBefore(gate string) func(delivery.Item, opActor) bool {
	return func(it delivery.Item, _ opActor) bool {
		return reports.ConditionOpenBefore(attentionRecord(it, ""), gate)
	}
}

// conditionNoOne is an open consent condition no one acts on: no party and
// no package.
func conditionNoOne(it delivery.Item, _ opActor) bool {
	return it.Kind == "condition" && delivery.Open(it.Kind, it.Status) && strings.TrimSpace(it.OwnerText) == "" && it.PackageID == ""
}

// rfiRef is "RFI-007", "RFI 7" or "rfi#7".
var rfiRef = regexp.MustCompile(`(?i)^\s*RFI[\s\-#]*0*(\d+)\s*$`)

// registerItem finds one live row by its id, its short id or its RFI
// number.
func registerItem(ctx context.Context, d Deps, who opActor, project, ref string) (delivery.Item, error) {
	items, err := d.Store.ReadDelivery(ctx, who.OrgID, project)
	if err != nil {
		return delivery.Item{}, err
	}
	if m := rfiRef.FindStringSubmatch(ref); m != nil {
		n, _ := strconv.Atoi(m[1])
		for _, it := range items {
			if it.Number != nil && *it.Number == n {
				return it, nil
			}
		}
		return delivery.Item{}, refuse("item", "There is no RFI-%03d on the register.", n)
	}
	return byRef(items, func(it delivery.Item) string { return it.ID }, ref, "register row")
}

// rfiFields are the RFI's own words, given as their own fields so a date
// in raised_on is checked like any other figure.
var rfiFields = []string{"question", "ref", "raised_by", "raised_on", "answer", "to", "cc", "subject"}

// rfiAddressing resolves an RFI's to and cc (a list, or comma-separated
// text, of addresses or firm names) through Send's resolver, so they are
// saved as lists of addresses the send route accepts.
func rfiAddressing(ctx context.Context, d Deps, who opActor, f map[string]any) error {
	for _, key := range []string{"to", "cc"} {
		v, ok := f[key]
		if !ok {
			continue
		}
		parts, bad := addressParts(v)
		if bad != "" {
			return refuse(key, "%s", bad)
		}
		list := []any{}
		for _, p := range parts {
			resolved, bad := resolveAddresses(ctx, d, who, p)
			if bad != "" {
				return refuse(key, "%s", bad)
			}
			for _, a := range resolved {
				list = append(list, a)
			}
		}
		f[key] = list
	}
	return nil
}

// liftRFIFields moves the RFI's own fields into details, over what the row
// has now. A raised date hidden inside details is refused: it must be its
// own field so chat's figure check sees it.
func liftRFIFields(f map[string]any, current json.RawMessage) error {
	details := map[string]any{}
	if len(current) > 0 {
		_ = json.Unmarshal(current, &details)
	}
	if given, ok := f["details"].(map[string]any); ok {
		if _, hidden := given["raised_on"]; hidden {
			return refuse("details", "Give raised_on as its own field, not inside details.")
		}
		for k, v := range given {
			details[k] = v
		}
	}
	lifted := false
	for _, k := range rfiFields {
		if v, ok := f[k]; ok {
			details[k] = v
			delete(f, k)
			lifted = true
		}
	}
	if lifted || f["details"] != nil {
		f["details"] = details
	}
	return nil
}

// sendRegisterAdd is delivery.add's body: the package named by id or
// title, the RFI's fields in details.
func sendRegisterAdd(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
	if ref := fieldText(f, "package"); ref != "" {
		p, err := pkgOf(ctx, d, who, project, f)
		if err != nil {
			return sent{}, err
		}
		f["package_id"] = p.ID
		delete(f, "package")
	}
	if sys, ok := f["system"]; ok {
		f["system_id"] = sys
		delete(f, "system")
	}
	kind := fieldText(f, "kind")
	if kind == "rfi" {
		if err := rfiAddressing(ctx, d, who, f); err != nil {
			return sent{}, err
		}
		if err := liftRFIFields(f, nil); err != nil {
			return sent{}, err
		}
	} else if _, ok := f["raised_on"]; ok {
		return sent{}, refuse("raised_on", "Only an RFI has a raised date.")
	}
	return sent{Body: f}, nil
}

// sendRegisterEdit is delivery.edit's: the row by id or RFI number, its
// version filled from the record when the caller has none, and the RFI's
// own fields merged into its details.
func sendRegisterEdit(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
	if err := need(f, "item"); err != nil {
		return sent{}, err
	}
	if _, ok := f["accept_reading"]; ok {
		// The screen's Accept is the PM's own click, captured with Undo.
		// From chat, accepting is a vouch and waits for the card's click.
		if c, marked := store.ChangeOf(ctx); !marked || c.Via != "screen" {
			return sent{}, refuse("accept_reading", "Accepting an answer is the PM's to confirm: use delivery.accept.")
		}
	}
	ref := take(f, "item")
	var it delivery.Item
	var err error
	if conditionRef.MatchString(ref) {
		it, err = conditionItem(ctx, d, who, project, ref)
	} else {
		it, err = registerItem(ctx, d, who, project, ref)
	}
	if err != nil {
		return sent{}, err
	}
	if it.Kind == "condition" {
		if fieldText(f, "status") == "satisfied" {
			// Only the PM marks a condition satisfied (owner decision 6): the
			// screen's own click, or delivery.satisfy's card.
			if c, marked := store.ChangeOf(ctx); !marked || c.Via != "screen" {
				return sent{}, refuse("status", "Marking a condition satisfied is the PM's to confirm: use delivery.satisfy.")
			}
		}
		if doc := fieldText(f, "link_evidence"); doc != "" {
			v, _, err := projectDocument(ctx, d, who, project, doc)
			if err != nil {
				return sent{}, err
			}
			f["link_evidence"] = v.ID
		}
		if err := liftConditionFields(f, it.Details); err != nil {
			return sent{}, err
		}
	}
	if _, ok := f["version"]; !ok {
		f["version"] = it.Version
	}
	if sys, ok := f["system"]; ok {
		f["system_id"] = sys
		delete(f, "system")
	}
	if ref := fieldText(f, "package"); ref != "" {
		p, err := pkgOf(ctx, d, who, project, f)
		if err != nil {
			return sent{}, err
		}
		f["package_id"] = p.ID
		delete(f, "package")
	}
	if it.Kind == "rfi" {
		if err := rfiAddressing(ctx, d, who, f); err != nil {
			return sent{}, err
		}
		if err := liftRFIFields(f, it.Details); err != nil {
			return sent{}, err
		}
		// An RFI's attached documents, as its transmittal: the whole list,
		// each checked against the project by the store.
		if docs, ok := f["documents"]; ok {
			refs := []map[string]string{}
			list, ok := docs.([]any)
			if !ok {
				return sent{}, refuse("documents", "Send documents as a list of document ids.")
			}
			for _, raw := range list {
				id, _ := raw.(string)
				v, _, err := projectDocument(ctx, d, who, project, id)
				if err != nil {
					return sent{}, err
				}
				refs = append(refs, map[string]string{"document_id": v.ID})
			}
			f["source_refs"] = refs
			delete(f, "documents")
		}
	}
	return sent{Path: map[string]string{"item": it.ID}, Body: f}, nil
}

// sendRegisterAccept is delivery.accept's: the reading by its id from the
// read. Status, the answer document and the date answered are set by Go.
func sendRegisterAccept(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (sent, error) {
	it, rd, err := registerReading(ctx, d, who, project, f)
	if err != nil {
		return sent{}, err
	}
	if it.Kind == "condition" {
		return sent{}, refuse("item", "That is a consent condition: use delivery.accept_evidence.")
	}
	return sent{Path: map[string]string{"item": it.ID}, Body: map[string]any{"version": it.Version, "accept_reading": rd}}, nil
}

func registerReading(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (delivery.Item, string, error) {
	if err := need(f, "item", "reading"); err != nil {
		return delivery.Item{}, "", err
	}
	it, err := registerItem(ctx, d, who, project, fieldText(f, "item"))
	if err != nil {
		return it, "", err
	}
	rd, err := byRef(it.Readings, func(r delivery.Reading) string { return r.ID }, fieldText(f, "reading"), "possible answer on that row")
	if err != nil {
		return it, "", err
	}
	return it, rd.ID, nil
}

// registerAcceptCard is the card's words: which document answers which RFI.
func registerAcceptCard(ctx context.Context, d Deps, who opActor, project string, f map[string]any) (string, error) {
	it, id, err := registerReading(ctx, d, who, project, f)
	if err != nil {
		return "", err
	}
	for _, rd := range it.Readings {
		if rd.ID == id {
			name := rfiNumber(it.Number)
			if name == "" {
				name = it.Title
			}
			return fmt.Sprintf("Accept %s as the answer to %s (%s). It becomes Answered, with you.", rd.Document, name, it.Title), nil
		}
	}
	return "", refuse("reading", "That possible answer is no longer there.")
}

// conditionColumns are a condition's No., Due and Status words: its printed
// number, its gate, and what is needed ("open · evidence needed · to
// check").
func conditionColumns(it delivery.Item) (number, due, status string) {
	d := delivery.Condition(it.Content)
	due = delivery.GateWords[d.Stage]
	if due == "" {
		due = "gate not known"
	}
	status = strings.ReplaceAll(it.Status, "_", " ")
	if d.EvidenceNeeded != nil && *d.EvidenceNeeded && delivery.Open(it.Kind, it.Status) && len(d.Evidence) == 0 {
		status += " · evidence needed"
	}
	if len(it.Readings) > 0 {
		status += " · possible evidence"
	}
	if it.ReviewStatus == "proposed" {
		status += " · to check (read by SiteWise)"
	}
	var p struct {
		Flag string `json:"flag"`
	}
	if json.Unmarshal(it.Provenance, &p) == nil && p.Flag != "" {
		status += " · " + p.Flag
	}
	return "Condition " + d.Number, due, status
}

// consentHeader prints the consent's number, date and lapse date from the
// Profile, and each notice read: how many conditions, which numbers are
// missing.
func consentHeader(view store.RegisterView, r *reader) {
	c := view.Consent
	if c.Number != "" || c.Date != "" || c.LapseDate != "" {
		r.line("Development consent (from the Profile): number %s, date %s, lapses %s.", orDash(c.Number), orDash(c.Date), orDash(c.LapseDate))
	}
	for _, read := range view.Consents {
		switch {
		case read.Status == "queued" || read.Status == "reading":
			r.line("Consent conditions of %s: %s.", read.Title, read.Status)
		case read.Status == "failed":
			r.line("Consent conditions of %s: the read stopped (%s).", read.Title, read.Error)
		case read.Notice != nil:
			n := read.Notice
			line := fmt.Sprintf("Consent conditions of %s (consent %s): %d numbered, %d rows on the register", read.Title, orDash(n.ConsentRef), n.Conditions, read.Rows)
			if len(n.Gaps) > 0 {
				line += "; not found: " + strings.Join(n.Gaps, ", ")
			}
			r.line("%s.", line)
		}
	}
}
