package rfp

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"sitewise/internal/knowledge"
	"sitewise/internal/reports"
)

// Combined engagements, slice 2 (docs/plans/2026-11-07-combined-engagements-slice2.md):
// one RFP for several disciplines bought under one appointment. Each member
// is resolved and rendered exactly as its own RFP would be; this file only
// composes those parts: the shared parts once, one part per discipline,
// items one discipline gives another shown once, one fee schedule and every
// member's checks. It is pure, like Resolve and Sections.

// GroupPart is one discipline of a combined engagement, in Procurement order.
type GroupPart struct {
	PackageID    string
	PackageTitle string
	DisciplineID string
	// Templated is false for a discipline with no RFP file: its part says
	// the scope is still to be described and a check blocks issue.
	Templated  bool
	Discipline knowledge.RFPDiscipline
	Resolved   Resolved
	Settings   Settings
}

// Title is the part's heading: the discipline file's title, or the
// package's own name when there is no file.
func (p GroupPart) Title() string {
	// The general file's title ("Consultant services") names no discipline;
	// the package title does.
	if p.Templated && p.Resolved.Title != "" && p.Resolved.Discipline != knowledge.GeneralConsultantRFP {
		return p.Resolved.Title
	}
	return p.PackageTitle
}

// Engagement is a combined engagement: the group's name and its parts.
type Engagement struct {
	Title string
	Parts []GroupPart
}

// SharedSetting reports whether a setting key belongs to the whole
// engagement rather than one discipline: the issue details, terms and the
// transmittal print once, so they are stored once, on the group's report.
func SharedSetting(sk knowledge.RFPSkeleton, key string) bool {
	if DocumentListSetting(key) {
		return true
	}
	name, ok := strings.CutPrefix(key, "field:")
	if !ok {
		return false
	}
	for _, f := range sk.Fields {
		if f.Key == name {
			return f.Group != "appointment"
		}
	}
	return false
}

// MergeSettings is a member's settings for the engagement: its own choices,
// with the shared ones taken from the group.
func MergeSettings(sk knowledge.RFPSkeleton, member, group Settings) Settings {
	out := Settings{}
	for k, v := range member {
		if !SharedSetting(sk, k) {
			out[k] = v
		}
	}
	for k, v := range group {
		if SharedSetting(sk, k) {
			out[k] = v
		}
	}
	return out
}

// PartSectionID names a member's part, so the editor can find its member.
func PartSectionID(packageID string) string { return "part:" + packageID }

const groupFeesID = "fees"

// GroupSections renders the engagement's one RFP document.
func GroupSections(e Engagement, template knowledge.ReportTemplate) ([]reports.Section, error) {
	lead := e.lead()
	if lead < 0 {
		return nil, fmt.Errorf("a combined engagement needs at least one discipline with an RFP")
	}
	moved, gained := e.within()
	shared, err := sections(e.sharedResolved(lead), template, partOptions{})
	if err != nil {
		return nil, err
	}
	byID := map[string]reports.Section{}
	for _, s := range shared {
		byID[s.ID] = s
	}
	out := []reports.Section{}
	project := byID["project"]
	project.Blocks = append([]reports.Block{e.block("rfp:group:engagement", "", e.engagementText(), map[string]any{"engagement": e.Title, "disciplines": e.titles()})}, project.Blocks...)
	out = append(out, project)
	out = append(out, reports.Section{ID: "scope", Title: "2. Scope of services", Essential: true, Blocks: []reports.Block{
		e.block("rfp:group:scope", "", "This engagement covers "+strconv.Itoa(len(e.Parts))+" disciplines. Each part below is that discipline's scope, as it would be issued alone. An item one discipline's scope gives to another discipline in this engagement is shown once, under the discipline that does it, marked within this engagement.", nil),
	}})
	reviews := []reports.Section{}
	for i, p := range e.Parts {
		part, review, err := e.part(i, p, template, moved[i], gained[i])
		if err != nil {
			return nil, err
		}
		out = append(out, part)
		reviews = append(reviews, review)
	}
	out = append(out, e.fees())
	for _, id := range []string{"transmittal", "terms", "returnables", "issue_dates"} {
		s, ok := byID[id]
		if !ok {
			continue
		}
		switch id {
		case "transmittal":
			s.Title = "4. Transmittal"
		case "terms":
			s.Title = "5. Terms"
		}
		out = append(out, s)
	}
	return append(out, e.review(reviews)), nil
}

// lead is the first part with an RFP file; it supplies the shared parts.
func (e Engagement) lead() int {
	for i, p := range e.Parts {
		if p.Templated {
			return i
		}
	}
	return -1
}

func (e Engagement) titles() []string {
	out := []string{}
	for _, p := range e.Parts {
		out = append(out, p.Title())
	}
	return out
}

func (e Engagement) engagementText() string {
	return "This RFP is for one engagement, " + e.Title + ", covering " + joinAnd(e.titles()) + ". One agreement covers every discipline. You may deliver any discipline yourself or through a subconsultant you name in R4. Price each discipline in the fee schedule (section 3)."
}

func joinAnd(list []string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	}
	return strings.Join(list[:len(list)-1], ", ") + " and " + list[len(list)-1]
}

// block is a composed block; it is provisional while any member's rules are
// a draft, like every block of a draft discipline.
func (e Engagement) block(id, label, text string, extra map[string]any) reports.Block {
	basis := map[string]any{"method": "rfp_group", "status": e.status()}
	for k, v := range extra {
		basis[k] = v
	}
	raw, _ := json.Marshal(basis)
	return reports.Block{ID: id, Label: label, Text: text, Origin: "calculation", ReviewStatus: "proposed", Meaning: "requirement", Basis: raw, Provisional: e.status() != "reviewed"}
}

func (e Engagement) tableBlock(id, label, text string, table *reports.Table, extra map[string]any) reports.Block {
	b := e.block(id, label, text, extra)
	b.Table = table
	return b
}

// status is reviewed only when every member's rules are.
func (e Engagement) status() string {
	for _, p := range e.Parts {
		if !p.Templated || p.Resolved.Status != "reviewed" {
			return "draft"
		}
	}
	return "reviewed"
}

// sharedResolved is the lead's resolution with the terms and returnable
// schedules of every member merged: a clause every member carries prints
// once; one only some carry is labelled with their disciplines.
func (e Engagement) sharedResolved(lead int) Resolved {
	r := e.Parts[lead].Resolved
	templated := []GroupPart{}
	for _, p := range e.Parts {
		if p.Templated {
			templated = append(templated, p)
		}
	}
	type entry struct {
		row    Row
		owners []string
	}
	terms := []*entry{}
	byText := map[string]*entry{}
	for _, p := range templated {
		for _, t := range p.Resolved.Terms {
			key := strings.TrimSpace(t.Text)
			if x, ok := byText[key]; ok {
				x.owners = append(x.owners, p.Title())
				continue
			}
			x := &entry{row: t, owners: []string{p.Title()}}
			byText[key] = x
			terms = append(terms, x)
		}
	}
	r.Terms = []Row{}
	for _, t := range terms {
		row := t.row
		if len(t.owners) < len(templated) {
			row.Text = strings.Join(t.owners, ", ") + ": " + row.Text
		}
		r.Terms = append(r.Terms, row)
	}
	returnables := make([]Row, len(r.Returnables))
	for i, ret := range r.Returnables {
		ret.Text = mergeSentences(ret.Key, templated)
		returnables[i] = ret
	}
	r.Returnables = returnables
	return r
}

// mergeSentences joins one returnable schedule's text across members: the
// sentences every member asks for once, then each member's own, labelled.
func mergeSentences(key string, parts []GroupPart) string {
	split := func(text string) []string {
		out := []string{}
		for _, s := range strings.SplitAfter(text, ". ") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	count := map[string]int{}
	lists := make([][]string, len(parts))
	for i, p := range parts {
		for _, ret := range p.Resolved.Returnables {
			if ret.Key == key {
				lists[i] = split(ret.Text)
			}
		}
		seen := map[string]bool{}
		for _, s := range lists[i] {
			if !seen[s] {
				count[s]++
				seen[s] = true
			}
		}
	}
	common := []string{}
	done := map[string]bool{}
	for _, list := range lists {
		for _, s := range list {
			if count[s] == len(parts) && !done[s] {
				common = append(common, s)
				done[s] = true
			}
		}
	}
	out := common
	for i, list := range lists {
		own := []string{}
		for _, s := range list {
			if count[s] < len(parts) && !done[s] {
				own = append(own, s)
				done[s] = true
			}
		}
		if len(own) > 0 {
			out = append(out, parts[i].Title()+": "+strings.Join(own, " "))
		}
	}
	return strings.Join(out, " ")
}

// part renders one member: its role, stage table and services as its own
// RFP prints them (without a fee column), its allowances, and the items
// other members give it. It returns the member's Before-issue section too.
func (e Engagement) part(i int, p GroupPart, template knowledge.ReportTemplate, moved map[string]bool, gained []withinItem) (reports.Section, reports.Section, error) {
	id := PartSectionID(p.PackageID)
	title := "2." + strconv.Itoa(i+1) + " " + p.Title()
	sec := reports.Section{ID: id, Title: title, Essential: true, Blocks: []reports.Block{}}
	if !p.Templated {
		sec.Blocks = append(sec.Blocks, e.block(id+":rfp:role", "", "Scope to be described. SiteWise has no RFP rules for this discipline yet; describe its scope before issue.", map[string]any{"package_id": p.PackageID}))
		review := reports.Section{ID: reports.ReviewSectionID, Blocks: []reports.Block{}}
		review.Blocks = append(review.Blocks, e.reviewBlock("rfp:check:scope_described", "Blocks issue — Scope to be described", "There are no RFP rules for "+p.PackageTitle+". Describe its scope before issue.", map[string]any{"blocking": true}))
		return sec, review, nil
	}
	rendered, err := sections(p.Resolved, template, partOptions{part: true, moved: moved})
	if err != nil {
		return sec, reports.Section{}, err
	}
	var review reports.Section
	for _, s := range rendered {
		switch s.ID {
		case "role", "stages", "positions":
			sec.Blocks = append(sec.Blocks, s.Blocks...)
		case "allowances":
			for _, b := range s.Blocks {
				if b.Label == "" {
					b.Label = "Allowances"
				}
				sec.Blocks = append(sec.Blocks, b)
			}
		case reports.ReviewSectionID:
			review = s
		}
	}
	if len(gained) > 0 {
		lines := []string{"Other disciplines' scopes in this engagement give these items to you. They are part of your fee and are priced once."}
		for _, g := range gained {
			text := upperFirst(g.Label)
			if g.Detail != "" {
				text += ". " + g.Detail
			}
			lines = append(lines, "- "+text+" (from the "+g.From+" scope; within this engagement)")
		}
		sec.Blocks = append(sec.Blocks, e.block("rfp:within", "Within this engagement", strings.Join(lines, "\n"), map[string]any{"within": gained}))
	}
	for j := range sec.Blocks {
		sec.Blocks[j].ID = id + ":" + sec.Blocks[j].ID
	}
	return sec, review, nil
}

// withinItem is a service one member's template gives to another member.
type withinItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Detail is the service's own description, printed with it.
	Detail string `json:"detail,omitempty"`
	From   string `json:"from"`
	// FromPackage is the member whose template allocates the item.
	FromPackage string `json:"from_package"`
}

// within finds each excluded service whose responsible party is another
// member of the engagement. moved[i] are part i's services printed
// elsewhere; gained[j] are the items part j prints as its own. A party that
// names no member, or more than one, leaves the wording unchanged.
func (e Engagement) within() ([]map[string]bool, [][]withinItem) {
	moved := make([]map[string]bool, len(e.Parts))
	gained := make([][]withinItem, len(e.Parts))
	for i, p := range e.Parts {
		moved[i] = map[string]bool{}
		if !p.Templated {
			continue
		}
		responsible := map[string]string{}
		for _, pos := range p.Discipline.Positions {
			responsible[pos.Key] = pos.Responsible
		}
		for _, s := range p.Resolved.Scope {
			if s.Value != "X" || s.Inactive {
				continue
			}
			party := responsible[s.Key]
			if raw, ok := p.Settings["responsible:"+s.Key]; ok {
				if v := strings.TrimSpace(settingString(raw.Value)); v != "" {
					party = v
				}
			}
			j := e.memberFor(party)
			if j < 0 || j == i {
				continue
			}
			moved[i][s.Key] = true
			gained[j] = append(gained[j], withinItem{Key: s.Key, Label: s.Label, Detail: s.Detail, From: p.Title(), FromPackage: p.PackageID})
		}
	}
	return moved, gained
}

// memberFor is the one member a responsible party names, by the longest
// discipline word it contains; -1 when none or the party is shared.
func (e Engagement) memberFor(party string) int {
	party = " " + strings.ToLower(strings.ReplaceAll(party, "-", " ")) + " "
	if strings.TrimSpace(party) == "" {
		return -1
	}
	for _, sep := range []string{" or ", " and ", "/", ","} {
		if strings.Contains(party, sep) {
			return -1
		}
	}
	best, length, tie := -1, 0, false
	for j, p := range e.Parts {
		for _, w := range MemberWords(p) {
			if !strings.Contains(party, " "+w) {
				continue
			}
			switch {
			case len(w) > length:
				best, length, tie = j, len(w), false
			case len(w) == length && j != best:
				tie = true
			}
		}
	}
	if tie {
		return -1
	}
	return best
}

// genericWords are the first words of discipline ids that name no
// discipline on their own ("fire" could be services or engineering).
var genericWords = map[string]bool{"fire": true, "land": true, "bca": true, "town": true, "cost": true, "waste": true}

// MemberWords are the words a responsible party uses for a member: its
// discipline file's aliases, its id, and the id's first word when specific.
// A fee proposal names a discipline in the same words (slice 3 placement).
func MemberWords(p GroupPart) []string {
	if !p.Templated {
		if p.PackageTitle == "" {
			return nil
		}
		return []string{strings.ToLower(p.PackageTitle)}
	}
	words := []string{strings.ReplaceAll(p.Discipline.ID, "-", " ")}
	for _, a := range p.Discipline.TitleAliases {
		words = append(words, strings.ToLower(strings.ReplaceAll(a, "-", " ")))
	}
	if first, _, _ := strings.Cut(p.Discipline.ID, "-"); !genericWords[first] {
		words = append(words, first)
	}
	return words
}

// MemberKey prefixes a member's fee rows on the response form and in the
// group's price table: its RFP file's id, or its package name when it has
// none. One key in both places lets a returned form read back by key.
func MemberKey(p GroupPart) string {
	if p.Templated {
		return p.Discipline.ID
	}
	return slug(p.PackageTitle)
}

// FeeSection is one discipline's part of the engagement's fee schedule.
type FeeSection struct {
	PackageID string `json:"package_id"`
	// Key prefixes the discipline's rows on the response form: its RFP
	// file's id, or its package name when it has none.
	Key   string         `json:"key"`
	Title string         `json:"title"`
	Items []ScheduleItem `json:"items"`
	// Rates are the discipline's hourly-rate roles (R3).
	Rates []ScheduleItem `json:"rates"`
}

// GroupSchedule is the engagement's fee schedule: each discipline's priced
// items (included stages, reimbursables, options), in Procurement order. A
// discipline with no RFP file is one lump sum.
func GroupSchedule(e Engagement) []FeeSection {
	out := []FeeSection{}
	for _, p := range e.Parts {
		var items []ScheduleItem
		if p.Templated {
			items = Schedule(p.Resolved)
		} else {
			items = GenericSchedule(nil)
		}
		f := FeeSection{PackageID: p.PackageID, Key: MemberKey(p), Title: p.Title(), Items: []ScheduleItem{}, Rates: []ScheduleItem{}}
		for _, it := range items {
			switch {
			case it.Price():
				f.Items = append(f.Items, it)
			case it.Kind == ItemRate:
				f.Rates = append(f.Rates, it)
			}
		}
		out = append(out, f)
	}
	return out
}

// CombinedStageNote is how a bidder prices a stage as one amount for
// several disciplines (owner decision 2): the stage is then recorded as
// combined, and no discipline's share is guessed.
const CombinedStageNote = "Give a sum for each discipline and the total for the engagement. If one amount covers a stage for more than one discipline, enter it once, against the first of them, and write \"Included in\" with that discipline and stage against the others. That stage is then recorded as combined; it is not split."

// fees prints the fee schedule: a section per discipline with its included
// stages and a sum, then the engagement total. Options sit below each sum.
func (e Engagement) fees() reports.Section {
	schedule := GroupSchedule(e)
	rows := [][]string{}
	for _, f := range schedule {
		rows = append(rows, []string{f.Title, ""})
		options := [][]string{}
		for _, it := range f.Items {
			if !it.Base {
				options = append(options, []string{"Option: " + it.Label, ""})
				continue
			}
			rows = append(rows, []string{it.Label, ""})
		}
		rows = append(rows, []string{"Sum for " + f.Title, ""})
		rows = append(rows, options...)
	}
	rows = append(rows, []string{"Engagement total (the sums above)", ""})
	text := "Fees exclude GST; state GST and the total in your proposal. Options are priced separately and are not in the sums. " + CombinedStageNote
	return reports.Section{ID: groupFeesID, Title: "3. Fee schedule", Essential: true, Blocks: []reports.Block{
		e.tableBlock("rfp:fees", "", text, doc([]string{"Discipline and stage", "Fee $"}, rows), map[string]any{"fee_schedule": feeKeys(schedule), "layout": "group_fees"}),
	}}
}

// feeKeys is the fee schedule as the block's basis records it: each
// discipline's item keys in order, without the descriptions the parts
// already print.
func feeKeys(schedule []FeeSection) []map[string]any {
	out := []map[string]any{}
	for _, f := range schedule {
		keys := []string{}
		for _, it := range f.Items {
			keys = append(keys, it.Key)
		}
		out = append(out, map[string]any{"package_id": f.PackageID, "key": f.Key, "items": keys})
	}
	return out
}

// reviewBlock is one Before-issue item of the engagement.
func (e Engagement) reviewBlock(id, label, text string, extra map[string]any) reports.Block {
	b := e.block(id, label, text, extra)
	b.Meaning = "stated"
	b.Provisional = true
	return b
}

// review merges every member's Before-issue items. An item every member
// raises in the same words (an issue detail, a project fact) prints once;
// any other is labelled with its discipline.
func (e Engagement) review(parts []reports.Section) reports.Section {
	sec := reports.Section{ID: reports.ReviewSectionID, Title: "Before issue — resolve or accept", Internal: true, Blocks: []reports.Block{}}
	key := func(b reports.Block) string { return b.ID + "\x00" + b.Label + "\x00" + b.Text }
	count := map[string]int{}
	templated := 0
	for i, s := range parts {
		if !e.Parts[i].Templated {
			continue
		}
		templated++
		for _, b := range s.Blocks {
			count[key(b)]++
		}
	}
	done := map[string]bool{}
	for i, s := range parts {
		p := e.Parts[i]
		for _, b := range s.Blocks {
			k := key(b)
			if p.Templated && count[k] == templated {
				if !done[k] {
					done[k] = true
					sec.Blocks = append(sec.Blocks, b)
				}
				continue
			}
			b.ID = PartSectionID(p.PackageID) + ":" + b.ID
			b.Label = labelCheck(p.Title(), b.Label)
			sec.Blocks = append(sec.Blocks, b)
		}
	}
	return sec
}

// labelCheck names the discipline after "Blocks issue — ", so the marker
// still leads.
func labelCheck(title, label string) string {
	if rest, ok := strings.CutPrefix(label, "Blocks issue — "); ok {
		return "Blocks issue — " + title + ": " + rest
	}
	return title + ": " + label
}

// GroupChecks are every member's checks, labelled with the discipline. A
// check every member raises in the same words is listed once. A member's
// control key means nothing on the group's report, so only a shared check
// keeps it.
func GroupChecks(e Engagement) []Check {
	type keyed struct {
		part  int
		check Check
	}
	all := []keyed{}
	count := map[string]int{}
	key := func(c Check) string { return c.ID + "\x00" + c.Label + "\x00" + c.Text }
	templated := 0
	for i, p := range e.Parts {
		if !p.Templated {
			all = append(all, keyed{i, Check{ID: "scope_described", Label: "Scope to be described", Text: "There are no RFP rules for " + p.PackageTitle + ". Describe its scope before issue.", Blocking: true}})
			continue
		}
		templated++
		seen := map[string]bool{}
		for _, c := range p.Resolved.Checks {
			all = append(all, keyed{i, c})
			if !seen[key(c)] {
				count[key(c)]++
				seen[key(c)] = true
			}
		}
	}
	out := []Check{}
	done := map[string]bool{}
	for _, k := range all {
		c := k.check
		if e.Parts[k.part].Templated && count[key(c)] == templated {
			if !done[key(c)] {
				done[key(c)] = true
				out = append(out, c)
			}
			continue
		}
		p := e.Parts[k.part]
		c.ID = p.PackageID + ":" + c.ID
		c.Label = p.Title() + ": " + c.Label
		c.Control = ""
		c.Changes = nil
		out = append(out, c)
	}
	return out
}

// GroupBlocking are the checks that block the engagement's issue.
func GroupBlocking(e Engagement) []Check {
	out := []Check{}
	for _, c := range GroupChecks(e) {
		if c.Blocking {
			out = append(out, c)
		}
	}
	return out
}

// GroupControls turns the lead member's controls into the engagement's
// shared ones: page one (less the parts a discipline covers, which are each
// member's own), the terms and the proposal dates, with every member's
// checks merged. Each member's own controls come from its own RFP.
func GroupControls(lead Controls, e Engagement) Controls {
	out := lead
	out.Discipline, out.Title, out.Status = "", e.Title, e.status()
	out.Decisions, out.Checks = GroupBlocking(e), GroupChecks(e)
	out.Groups = []Group{}
	for _, g := range lead.Groups {
		switch g.ID {
		case "project":
			kept := []Control{}
			for _, c := range g.Controls {
				if c.Key != "parts:affected" {
					kept = append(kept, c)
				}
			}
			g.Controls = kept
		case "terms", "issue":
		default:
			continue
		}
		out.Groups = append(out.Groups, g)
	}
	return out
}

// GroupSharedReturnables are the engagement's returnable schedules, merged
// as the document prints them.
func GroupSharedReturnables(e Engagement) []Row {
	lead := e.lead()
	if lead < 0 {
		return nil
	}
	return e.sharedResolved(lead).Returnables
}
