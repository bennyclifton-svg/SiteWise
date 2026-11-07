package proposals

import (
	"regexp"
	"sort"
	"strings"

	"sitewise/internal/rfp"
)

// Combined engagements, slice 3 (docs/plans/2026-11-07-combined-engagements-slice3.md):
// a proposal for a group of disciplines is read onto one schedule. Each
// member's priced items keep their own keys behind the member's key (the
// response form's keys), and a stage two or more members share has a
// combined cell for a bid that prices that stage once. Jev answers the
// existing price questions over the members' items, in the one fan-out;
// code then decides where each amount sits. Nothing is ever split.

// GroupMember is one discipline of a group as the reader needs it: the key
// its items carry, its name, and the words a bid uses for it.
type GroupMember struct {
	Key   string
	Title string
	Words []string
}

// GroupLayout is a group's members in Procurement order, and its combined
// cells by key with their labels (WithCombined fills them from the schedule).
type GroupLayout struct {
	Members  []GroupMember
	Combined map[string]string
}

// WithCombined records the schedule's combined cells on the layout.
func (l GroupLayout) WithCombined(schedule []rfp.ScheduleItem) GroupLayout {
	l.Combined = map[string]string{}
	for _, it := range schedule {
		if _, _, combined := SplitItemKey(it.Key); combined {
			l.Combined[it.Key] = it.Label
		}
	}
	return l
}

// CombinedMember is the key prefix of a combined cell. A member whose own
// key would collide is given another by the caller (MemberKeyFor).
const CombinedMember = "combined"

// MemberKeyFor keeps a member's key clear of the combined prefix.
func MemberKeyFor(key string) string {
	if key == CombinedMember || key == "" {
		return key + "-member"
	}
	return key
}

// SplitItemKey parses a group schedule key: the member's key and its own
// item key, or combined for a combined cell. ok is false for a key that
// belongs to no member (a shared criterion).
func SplitItemKey(key string) (member, item string, combined bool) {
	member, item, ok := strings.Cut(key, ":")
	if !ok || strings.HasPrefix(key, "criterion:") {
		return "", "", false
	}
	return member, item, member == CombinedMember
}

// stageOf is the stage an item prices: a stage, a stage's element, or the
// reimbursables. Options and rates belong to no stage and are never combined.
func stageOf(item string) string {
	switch {
	case strings.HasPrefix(item, "stage:"):
		return item
	case strings.HasPrefix(item, "element:"):
		parts := strings.SplitN(item, ":", 3)
		if len(parts) == 3 {
			return "stage:" + parts[1]
		}
	case item == rfp.ReimbursablesKey:
		return item
	}
	return ""
}

// CombinedKey is the combined cell for a member item's stage, or "".
func CombinedKey(item string) string {
	if s := stageOf(item); s != "" {
		return CombinedMember + ":" + s
	}
	return ""
}

// GroupSchedule is the group's price table rows: each member's priced
// items in member order, then a combined cell for every base stage two or
// more members price, then the first member's non-price criteria once (the
// group RFP asks them once). schedules[i] is member i's own schedule.
func GroupSchedule(layout GroupLayout, schedules [][]rfp.ScheduleItem) []rfp.ScheduleItem {
	out := []rfp.ScheduleItem{}
	add := func(it rfp.ScheduleItem) {
		it.Ordinal = len(out)
		out = append(out, it)
	}
	type shared struct {
		label string
		count int
	}
	stages := map[string]*shared{}
	order := []string{}
	var criteria []rfp.ScheduleItem
	for i, m := range layout.Members {
		if i >= len(schedules) {
			break
		}
		for _, it := range schedules[i] {
			if it.Kind == rfp.ItemCriterion && i == 0 {
				criteria = append(criteria, it)
			}
			if !it.Price() {
				continue
			}
			if s := stageOf(it.Key); s != "" && it.Base {
				if stages[s] == nil {
					stages[s] = &shared{label: stageLabel(it)}
					order = append(order, s)
				}
				stages[s].count++
			}
			add(rfp.ScheduleItem{Key: m.Key + ":" + it.Key, Kind: it.Kind, Label: m.Title + " — " + it.Label, Detail: it.Detail, Base: it.Base, Unit: it.Unit})
		}
	}
	for _, s := range order {
		// Each member prices an element of a stage separately; one element
		// per member still makes one shared stage.
		if st := stages[s]; st.count >= 2 && sharedByMembers(layout, schedules, s) {
			add(rfp.ScheduleItem{Key: CombinedMember + ":" + s, Kind: rfp.ItemStage, Label: "Combined — " + st.label, Detail: "One amount the bid gives for this stage across disciplines; not split.", Base: true})
		}
	}
	for _, it := range criteria {
		add(it)
	}
	return out
}

// sharedByMembers is true when at least two members price stage s.
func sharedByMembers(layout GroupLayout, schedules [][]rfp.ScheduleItem, s string) bool {
	n := 0
	for i := range layout.Members {
		if i >= len(schedules) {
			break
		}
		for _, it := range schedules[i] {
			if it.Price() && it.Base && stageOf(it.Key) == s {
				n++
				break
			}
		}
	}
	return n >= 2
}

// stageLabel is the stage's own name: an element row reads "Stage: element".
func stageLabel(it rfp.ScheduleItem) string {
	if strings.HasPrefix(it.Key, "element:") {
		if name, _, ok := strings.Cut(it.Label, ":"); ok {
			return strings.TrimSpace(name)
		}
	}
	return it.Label
}

// GroupTotals is a group column's totals: a combined cell no one priced is
// not a gap, and a member's stage covered by a priced combined cell is not
// unread. Everything else is Totals, so a figure is never counted twice.
func GroupTotals(items []rfp.ScheduleItem, cells map[string]Cell, others []Cell, stated *int64) Column {
	priced := func(key string) bool {
		c, ok := cells[key]
		return ok && c.State == StatePriced
	}
	counted := make([]rfp.ScheduleItem, 0, len(items))
	for _, it := range items {
		member, item, combined := SplitItemKey(it.Key)
		switch {
		case combined && !priced(it.Key):
			continue
		case member != "" && !combined && cells[it.Key].State == "" && priced(CombinedKey(item)):
			continue
		}
		counted = append(counted, it)
	}
	return Totals(counted, cells, others, stated)
}

// placeGroup decides where each amount Jev matched to a member's item sits.
// chosen holds the amount behind each machine-read cell. It returns the
// amounts no cell keeps, which the caller lists as unplaced.
func (d *Draft) placeGroup(layout GroupLayout, prices []Price, chosen map[string]Amount) ([]Price, map[string]bool) {
	kept := map[string]bool{}
	dropped := map[string]bool{}
	unplace := func(i int, why string) {
		if a, ok := chosen[prices[i].Key]; ok {
			dropped[a.ID] = true
		}
		prices[i].State, prices[i].Cents, prices[i].Band = "", nil, BandBlank
		prices[i].Page, prices[i].Passage, prices[i].Basis = 0, "", why
	}
	type candidate struct {
		index int
		a     Amount
	}
	byStage := map[string][]candidate{}
	for i, p := range prices {
		member, item, combined := SplitItemKey(p.Key)
		a, read := chosen[p.Key]
		if member == "" || combined || !read {
			continue
		}
		names := d.namedMembers(layout, a)
		switch {
		case len(names) == 1 && names[0] == member:
			kept[a.ID] = true
		case len(names) > 0 && !containsString(names, member):
			unplace(i, "The quoted line names another discipline; place it with a cell correction.")
		case layout.Combined[CombinedKey(item)] != "":
			byStage[CombinedKey(item)] = append(byStage[CombinedKey(item)], candidate{i, a})
		default:
			unplace(i, "The quoted line does not name this discipline; place it with a cell correction.")
		}
	}
	stages := make([]string, 0, len(byStage))
	for s := range byStage {
		stages = append(stages, s)
	}
	sort.Strings(stages)
	for _, s := range stages {
		cs := byStage[s]
		same := true
		for _, c := range cs[1:] {
			same = same && c.a.ID == cs[0].a.ID
		}
		if !same {
			for _, c := range cs {
				unplace(c.index, "Different amounts were read for this stage; place them with a cell correction.")
			}
			continue
		}
		cell := prices[cs[0].index]
		cell.Key, cell.Label = s, layout.Combined[s]
		for _, c := range cs {
			cell.Band = lowerBand(cell.Band, prices[c.index].Band)
			markIncluded(&prices[c.index], cell.Label)
		}
		kept[cs[0].a.ID] = true
		prices = append(prices, cell)
	}
	prices = d.formCombined(prices, layout.Combined)
	for id := range kept {
		delete(dropped, id)
	}
	return prices, dropped
}

// formCombined records a returned form's stage as combined when one member
// carries the amount and another writes "Included in" against it, as the
// form's note asks (rfp.CombinedStageNote).
func (d *Draft) formCombined(prices []Price, labels map[string]string) []Price {
	if d.ev.Form == nil {
		return prices
	}
	priced, included := map[string][]int{}, map[string]int{}
	for i, p := range prices {
		member, item, combined := SplitItemKey(p.Key)
		s := CombinedKey(item)
		if member == "" || combined || labels[s] == "" || p.DecidedBy != ByRule {
			continue
		}
		switch p.State {
		case StatePriced:
			priced[s] = append(priced[s], i)
		case StateIncluded:
			included[s]++
		}
	}
	for s, list := range priced {
		if len(list) != 1 || included[s] == 0 {
			continue
		}
		cell := prices[list[0]]
		cell.Key, cell.Label = s, labels[s]
		markIncluded(&prices[list[0]], cell.Label)
		prices = append(prices, cell)
	}
	return prices
}

func markIncluded(p *Price, label string) {
	p.State, p.Cents, p.Basis = StateIncluded, nil, "Included in "+label
}

var nonWord = regexp.MustCompile(`[^a-z0-9]+`)

// namedMembers are the members whose words the amount's own row, column,
// heading or section heading print. A word matches whole, so "civil" is
// not found in "civilian".
func (d *Draft) namedMembers(layout GroupLayout, a Amount) []string {
	parts := []string{a.Label, a.Column, a.Under}
	for _, s := range d.ev.Sections {
		if s.Ordinal == a.Section {
			parts = append(parts, s.Heading)
		}
	}
	text := " " + strings.TrimSpace(nonWord.ReplaceAllString(strings.ToLower(strings.Join(parts, " ")), " ")) + " "
	var out []string
	for _, m := range layout.Members {
		for _, w := range m.Words {
			w = strings.TrimSpace(nonWord.ReplaceAllString(strings.ToLower(w), " "))
			if w != "" && strings.Contains(text, " "+w+" ") {
				out = append(out, m.Key)
				break
			}
		}
	}
	return out
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// lowerBand is the less certain of two bands: a combined cell is no surer
// than the least sure answer behind it.
func lowerBand(a, b string) string {
	rank := map[string]int{BandGreen: 3, BandAmber: 2, BandBlank: 1, BandGrey: 0}
	if rank[b] < rank[a] {
		return b
	}
	return a
}

// groupBase is the placed prices' base sum and whether it is complete, for
// the stated-total check.
func groupBase(items []rfp.ScheduleItem, layout GroupLayout, prices []Price) (int64, bool) {
	all := append([]rfp.ScheduleItem{}, items...)
	for key := range layout.Combined {
		all = append(all, rfp.ScheduleItem{Key: key, Kind: rfp.ItemStage, Base: true})
	}
	cells := map[string]Cell{}
	for _, p := range prices {
		if p.Key != "" {
			cells[p.Key] = Cell{Key: p.Key, Cents: p.Cents, State: p.State}
		}
	}
	col := GroupTotals(all, cells, nil, nil)
	return col.KnownBase, col.BaseTotal != nil
}
