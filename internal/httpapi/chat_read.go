package httpapi

import (
	"context"
	"errors"
	"fmt"
	"math/bits"
	"sort"
	"strings"

	"sitewise/internal/agent"
	"sitewise/internal/costs"
	"sitewise/internal/figures"
	"sitewise/internal/store"
)

// The chat reads. Each renders one area as compact text from the same Go
// rollups its screen uses: no figure here is computed by the model. A
// reader never takes the org or project from the model.
//
// A figure the model may quote carries a tag ("[f12]"); the model writes
// "{f12}" and Go prints the value with its row label (review change 3).
// The plain form has no ids and no tags: the command key compares it
// before and after a turn.

// reader collects one read's text, facts and outside source.
type reader struct {
	b       strings.Builder
	plain   bool
	facts   map[string]agent.Fact
	next    func() string // the next fact tag
	outside string
}

func (r *reader) line(format string, args ...any) {
	fmt.Fprintf(&r.b, format+"\n", args...)
}

// id is a row's short id for the model, or nothing in the plain form.
func (r *reader) id(full string) string {
	if r.plain {
		return ""
	}
	return shortID(full) + " "
}

// fig writes a figure and, for the model, its tag.
func (r *reader) fig(value, shown, label, source string) string {
	if r.plain || r.next == nil {
		return value
	}
	tag := r.next()
	r.facts[tag] = agent.Fact{Label: label, Value: shown, Source: source}
	return value + " [" + tag + "]"
}

const maxReadBytes = 24000

// capped says so when a read is cut, and how to ask for the rest.
func capped(text, how string) string {
	if len(text) <= maxReadBytes {
		return text
	}
	cut := strings.LastIndexByte(text[:maxReadBytes], '\n')
	return text[:cut+1] + "[Capped: this is the first part only. " + how + "]\n"
}

func plainMoney(m *costs.Money) string {
	if m == nil {
		return "unknown"
	}
	if p, ok := figures.Plain(string(*m)); ok {
		return p
	}
	return string(*m)
}

var sectionNames = map[string]string{"authority": "Authority fees", "consultants": "Consultants", "construction": "Construction", "contingency": "Contingency and project-wide"}
var sectionOrder = []string{"construction", "consultants", "contingency", "authority"}

// readCostPlan renders the draft plan: totals, then each section's lines.
func readCostPlan(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	p, err := d.Store.ReadCostPlan(ctx, who.OrgID, project, "")
	if err != nil {
		return err
	}
	systems := func() {
		if r.plain {
			return
		}
		var ids []string
		for _, s := range p.Systems {
			ids = append(ids, s.ID+" ("+s.Label+")")
		}
		r.line("Systems a construction line can belong to: %s", strings.Join(ids, "; "))
	}
	if len(p.Items) == 0 {
		r.line("Cost plan: no lines yet.")
		systems()
		return nil
	}
	totals, err := costs.Summarize(p, "overall")
	if err != nil {
		return err
	}
	basis := "ex GST"
	if p.TaxBasis == "inc_tax" {
		basis = "inc GST"
	}
	rate := "unknown"
	if p.TaxRate != nil {
		rate = plainMoney(p.TaxRate)
	}
	r.line("Cost plan: %s, amounts %s, GST rate %s. Every figure here is computed by SiteWise.", p.Status, basis, rate)
	money := func(metric, label string, t costs.Total) string {
		known := t.KnownSubtotal
		out := metric + " " + r.fig(plainMoney(&known), dollars(known), label+", "+metric, "Cost Plan")
		if t.Unknown > 0 {
			out += fmt.Sprintf(" (+%d lines unknown)", t.Unknown)
		}
		return out
	}
	r.line("Total: %s | %s | %s", money("budget", "Total", totals.Overall["budget"]), money("contract", "Total", totals.Overall["commitment"]), money("claimed", "Total", totals.Overall["claimed_to_date"]))
	sections := costs.CostSections(p.Items)
	system := map[string]string{}
	for _, s := range p.Systems {
		system[s.ID] = s.Label
	}
	children := map[string][]costs.Item{}
	for _, i := range p.Items {
		children[i.ParentItemID] = append(children[i.ParentItemID], i)
	}
	var write func(i costs.Item, depth int)
	write = func(i costs.Item, depth int) {
		parts := []string{r.id(i.ID) + i.Label}
		val := func(metric, name string, always bool) {
			v := i.Values[metric]
			switch {
			case v.ValueState == "known" && v.Amount != nil:
				parts = append(parts, name+" "+r.fig(plainMoney(v.Amount), dollars(*v.Amount), i.Label+", "+name, "Cost Plan"))
			case always:
				parts = append(parts, name+" unknown")
			}
		}
		if i.Posting || i.Values["budget"].ValueState == "known" {
			val("budget", "budget", i.Posting)
			val("commitment", "contract", false)
			val("claimed_to_date", "claimed", false)
			if rem := totals.Items[i.ID].Remaining; rem != nil && i.Values["commitment"].ValueState == "known" {
				parts = append(parts, "remaining "+r.fig(plainMoney(rem), dollars(*rem), i.Label+", contract less claimed", "Cost Plan"))
			}
		} else {
			parts = append(parts, "group")
		}
		if i.Excluded {
			parts = append(parts, "excluded")
		}
		if !r.plain && i.SystemID != "" {
			parts = append(parts, "system "+i.SystemID)
		}
		r.line("%s%s", strings.Repeat("  ", depth), strings.Join(parts, " | "))
		for _, c := range children[i.ID] {
			write(c, depth+1)
		}
	}
	for _, section := range sectionOrder {
		var roots []costs.Item
		for _, i := range children[""] {
			if sections[i.ID] == section {
				roots = append(roots, i)
			}
		}
		head := strings.ToUpper(sectionNames[section])
		if t, ok := totals.Sections[section]; ok && len(roots) > 0 {
			head += ": " + money("budget", sectionNames[section], t["budget"]) + " | " + money("contract", sectionNames[section], t["commitment"]) + " | " + money("claimed", sectionNames[section], t["claimed_to_date"])
		}
		r.line("%s", head)
		for _, i := range roots {
			write(i, 1)
		}
	}
	systems()
	return nil
}

// readProfile renders the values the PM states or the documents gave: the
// header, the project facts and details, and the site.
func readProfile(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	if d.Knowledge == nil {
		r.line("Profile: not available.")
		return nil
	}
	view, err := d.Store.ReadProfile(ctx, who.OrgID, project, d.ProfileReading.Kinds())
	if err != nil {
		return err
	}
	body := profileBody(project, view, d)
	r.line("Profile (key | label: value (where it came from)). A dash is blank.")
	blank := 0
	group := func(name string, fields []fieldJSON) {
		if len(fields) == 0 {
			return
		}
		r.line("%s", name)
		for _, f := range fields {
			// The many yes/no flags are listed only once they hold a value.
			if f.Value == "" && f.ValueState != "unknown" && (strings.HasPrefix(f.Key, "proj.flag.") || strings.HasPrefix(f.Key, "hdr.cond.")) {
				blank++
				continue
			}
			value, origin := f.Value, f.Origin
			if f.ValueState == "unknown" {
				value = "unknown"
			}
			if value == "" {
				value, origin = "—", ""
			}
			if f.Unit != "" && value != "—" && value != "unknown" {
				value += " " + f.Unit
			}
			if origin != "" {
				value += " (" + origin + ")"
			}
			// A long value read from a document is the document's words.
			if f.Origin != "user" && len(strings.Fields(f.Value)) > 4 {
				r.outside = "what the documents say in the profile"
			}
			key := ""
			if !r.plain {
				key = f.Key + " | "
			}
			r.line("  %s%s: %s", key, f.Label, value)
		}
	}
	// Planning values (site area, floor space and the like) are profile
	// rows keyed plan.<key>; a value the PM states outranks the others.
	var planning []fieldJSON
	for _, pk := range d.Knowledge.PlanningKeys() {
		f := fieldJSON{Key: "plan." + pk.Key, Label: pk.Label, Unit: pk.Unit}
		for _, row := range view.Rows {
			if row.Key == f.Key && row.Value != "" {
				f.Value, f.Origin, f.ValueState = row.Value, row.Origin, row.ValueState
				break
			}
		}
		planning = append(planning, f)
	}
	// The size of the building (storeys, floor area and the like). The
	// screen shows the ones that fit the building class; chat lists them
	// all, so a size the PM states always has its own key.
	header := body.Header
	listed := map[string]bool{}
	for _, f := range header {
		listed[f.Key] = true
	}
	for _, f := range d.Knowledge.ScaleFields() {
		if key := "hdr.scale." + f.Key; !listed[key] {
			listed[key] = true
			header = append(header, fieldJSON{Key: key, Label: f.Label, Unit: f.Unit})
		}
	}
	group("Building", header)
	group("Planning", planning)
	group("Project", body.Details)
	group("Site", body.SiteDetails)
	group("Facts", body.Facts)
	if blank > 0 && !r.plain {
		r.line("(%d yes/no flags with no value are not listed; their keys start proj.flag. or hdr.cond.)", blank)
	}
	return nil
}

func workingDays(week int) int { return bits.OnesCount8(uint8(week)) }

// readProgram renders stages and their rows with Go's dates.
func readProgram(ctx context.Context, d Deps, who opActor, project string, r *reader, stagesOnly bool) error {
	st, err := d.Store.LoadProgram(ctx, who.OrgID, project)
	if errors.Is(err, store.ErrNotFound) {
		r.line("Program: none yet. Adding an activity creates it.")
		return nil
	}
	if err != nil {
		return err
	}
	times := map[string]int{}
	for i, t := range st.Computed.Activities {
		times[t.ID] = i
	}
	label := map[string]string{}
	unknown, holds := 0, 0
	for _, a := range st.Activities {
		label[a.ID] = a.Label
		if a.Kind != "stage" && a.DurationDays == nil && a.Kind != "milestone" {
			unknown++
		}
		if a.Kind == "hold_point" {
			holds++
		}
	}
	after := map[string][]string{}
	for _, l := range st.Links {
		after[l.SuccID] = append(after[l.SuccID], label[l.PredID])
	}
	start := string(st.Program.StartDate)
	if start == "" {
		start = "not set"
	}
	finish := "unknown"
	if f := st.Computed.Finish.Typical; f != nil {
		finish = r.fig(*f, *f, "Program finish", "Program")
		if !st.Computed.Finish.Complete {
			finish += " at the earliest (some durations are unknown)"
		}
	}
	r.line("Program: start %s, working week of %d days, finish %s. Rows with no duration: %d. Hold points: %d. Every date is computed by SiteWise; the program holds no progress, so nothing here says what is late.",
		start, workingDays(st.Program.WorkingWeek), finish, unknown, holds)
	for _, a := range st.Activities {
		if stagesOnly && a.Kind != "stage" && len(st.Activities) > 160 {
			continue
		}
		parts := []string{r.id(a.ID) + a.Label}
		indent := "  "
		if a.Kind == "stage" {
			indent = ""
			parts[0] += " (stage)"
		} else {
			if a.Kind != "activity" {
				parts[0] += " (" + strings.ReplaceAll(a.Kind, "_", " ") + ")"
			}
			if a.DurationDays != nil {
				parts = append(parts, fmt.Sprintf("%d d", *a.DurationDays))
			} else if a.Kind != "milestone" {
				parts = append(parts, "? d")
			}
		}
		if i, ok := times[a.ID]; ok && !stagesOnly {
			if t := st.Computed.Activities[i]; !t.Unknown && t.Start != "" {
				span := t.Start
				if t.Finish != "" {
					span += " to " + t.Finish
				}
				parts = append(parts, r.fig(span, span, a.Label, "Program"))
			}
		}
		if preds := after[a.ID]; len(preds) > 0 && !stagesOnly {
			sort.Strings(preds)
			parts = append(parts, "after "+strings.Join(preds, ", "))
		}
		r.line("%s%s", indent, strings.Join(parts, " | "))
	}
	return nil
}

func centsMoney(c *int64) *costs.Money {
	if c == nil {
		return nil
	}
	m := costs.Money(fmt.Sprintf("%d.%02d", *c/100, *c%100))
	return &m
}

// readProcurement renders each package with its firms: invited, quoted or
// awarded. A firm's name is SiteWise's record; a tender's lines are the
// firm's words and are read with read(procurement, part: a package id).
func readProcurement(ctx context.Context, d Deps, who opActor, project string, r *reader) error {
	pkgs, err := d.Store.ReadPackages(ctx, who.OrgID, project)
	if err != nil {
		return err
	}
	board, _, err := d.Store.ProcurementBoard(ctx, who.OrgID, project)
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		r.line("Procurement: no packages yet.")
		return nil
	}
	r.line("Procurement (package, then its firms). Amounts are ex GST.")
	for _, p := range pkgs {
		if p.RetiredAt != nil {
			continue
		}
		kind := "consultant"
		if p.Kind == "works" {
			kind = "trade or builder"
		}
		r.line("%s%s (%s) | %s", r.id(p.ID), p.Title, kind, strings.ReplaceAll(p.LifecycleStatus, "_", " "))
		cards := board[p.ID]
		if len(cards) == 0 {
			r.line("  no firms yet")
		}
		for _, c := range cards {
			parts := []string{c.FirmName}
			switch {
			case c.SubmissionID == "":
				parts = append(parts, "invited")
			default:
				total := c.BaseTotal
				if total == nil {
					total = c.StatedTotal
				}
				if m := centsMoney(total); m != nil {
					parts = append(parts, "quoted "+r.fig(plainMoney(m), dollars(*m), c.FirmName+", "+p.Title, "Procurement"))
				} else {
					parts = append(parts, "quote not priced")
				}
				parts = append(parts, strings.ReplaceAll(c.Status, "_", " "))
				if c.Lowest {
					parts = append(parts, "lowest")
				}
				if !r.plain {
					parts = append(parts, "column "+shortID(c.SubmissionID))
				}
			}
			r.line("  %s", strings.Join(parts, " | "))
		}
	}
	return nil
}

// chatOutline is the compact picture of the project that opens every turn,
// so a simple command needs no read first (speed rule 2): the cost plan
// with ids, the Program's stages and rows (stages only on a very large
// program), the packages and the profile's values. Dates and links are
// left to read(program).
func chatOutline(ctx context.Context, d Deps, who opActor, project, name, today string) string {
	parts := make([]string, 4)
	run := func(i int, f func(r *reader) error) func() {
		return func() {
			r := &reader{}
			if err := f(r); err != nil {
				parts[i] = "(could not be read)\n"
				return
			}
			parts[i] = r.b.String()
		}
	}
	jobs := []func(){
		run(0, func(r *reader) error { return readCostPlan(ctx, d, who, project, r) }),
		run(1, func(r *reader) error { return readProgram(ctx, d, who, project, r, true) }),
		run(2, func(r *reader) error { return readProcurement(ctx, d, who, project, r) }),
		run(3, func(r *reader) error { return readProfile(ctx, d, who, project, r) }),
	}
	done := make(chan struct{}, len(jobs))
	for _, job := range jobs {
		go func() { job(); done <- struct{}{} }()
	}
	for range jobs {
		<-done
	}
	head := fmt.Sprintf("Project: %s. Today is %s.\n", name, today)
	return capped(head+strings.Join(parts, "\n"), "Use read for the area you need.")
}
