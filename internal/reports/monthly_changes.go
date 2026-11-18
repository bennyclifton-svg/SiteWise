package reports

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"sitewise/internal/costs"
	"sitewise/internal/figures"
)

// Changes since report N (Task 4): what moved, computed from the last
// issued report's frozen blocks and this draft's, by block id. Amounts are
// compared exactly, dates in days. A block whose method differs from the
// previous one says "not compared".

type move struct {
	area   int // cost, program, procurement, registers, documents
	size   float64
	id     string
	text   string
	before any
	after  any
}

var areaNames = []string{"Cost", "Program", "Procurement", "Registers", "Documents"}

func (m monthly) changes(now []Section) []Block {
	p := m.in.Previous
	if p == nil {
		return []Block{m.block("changes:first", "", "This is the first issued report; changes are measured from the next one.", map[string]any{"method": "monthly_movement"})}
	}
	moves := movement(p.Sections, now)
	moves = append(moves, m.documentsFiled()...)
	sort.SliceStable(moves, func(i, j int) bool {
		if moves[i].area != moves[j].area {
			return moves[i].area < moves[j].area
		}
		return moves[i].size > moves[j].size
	})
	if len(moves) == 0 {
		return []Block{m.block("changes:none", "", fmt.Sprintf("Nothing in the cost plan, the Program, procurement or the registers has moved since report %d.", p.Number), map[string]any{"method": "monthly_movement", "since": p.Number})}
	}
	more := 0
	if len(moves) > maxMoved {
		more = len(moves) - maxMoved
		moves = moves[:maxMoved]
	}
	var out []Block
	for _, mv := range moves {
		out = append(out, m.block("moved:"+mv.id, areaNames[mv.area], mv.text, map[string]any{"method": "monthly_movement", "since": p.Number, "before": mv.before, "after": mv.after}))
	}
	if more > 0 {
		out = append(out, m.block("moved:more", "", fmt.Sprintf("And %d more changes in SiteWise.", more), map[string]any{"method": "monthly_movement", "more": more}))
	}
	return out
}

// movement compares two reports' sections by block id.
func movement(prev, now []Section) []move {
	var out []move
	out = append(out, costMoves(findBlock(prev, "cost:table"), findBlock(now, "cost:table"))...)
	out = append(out, programMoves(findBlock(prev, "program:completion"), findBlock(now, "program:completion"))...)
	out = append(out, packageMoves(findBlock(prev, "proc:table"), findBlock(now, "proc:table"))...)
	for _, k := range registerKinds {
		id := "reg:" + k.kind + ":summary"
		out = append(out, registerMoves(k.many, findBlock(prev, id), findBlock(now, id))...)
	}
	return out
}

func findBlock(sections []Section, id string) *Block {
	for i := range sections {
		for j := range sections[i].Blocks {
			if sections[i].Blocks[j].ID == id {
				return &sections[i].Blocks[j]
			}
		}
	}
	return nil
}

// sameMethod decodes both bases; false (with a "not compared" move) when the
// methods differ. Both nil is not a change.
func sameMethod(prev, now *Block, method string, into func(*Block) any) (any, any, bool) {
	if prev == nil || now == nil {
		return nil, nil, false
	}
	var pm, nm struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(prev.Basis, &pm)
	_ = json.Unmarshal(now.Basis, &nm)
	if pm.Method != method || nm.Method != method {
		return nil, nil, false
	}
	a, b := into(prev), into(now)
	return a, b, json.Unmarshal(prev.Basis, a) == nil && json.Unmarshal(now.Basis, b) == nil
}

func notCompared(area int, id string, prev, now *Block, method string) []move {
	if prev == nil || now == nil {
		return nil
	}
	var pm struct {
		Method string `json:"method"`
	}
	_ = json.Unmarshal(prev.Basis, &pm)
	if pm.Method == method {
		return nil
	}
	return []move{{area: area, id: id + ":not_compared", text: areaNames[area] + " is not compared: the previous report was built another way."}}
}

type costBasis struct {
	Figures map[string]map[string]any `json:"figures"`
}

func figureText(row map[string]any, key string) *string {
	if s, ok := row[key].(string); ok {
		return &s
	}
	return nil
}

// knownPart is a partly known total's known subtotal and unknown count.
func knownPart(row map[string]any, metric string) (costs.Money, int, bool) {
	s, ok := row[metric+"_known"].(string)
	if !ok {
		return "", 0, false
	}
	n, _ := row[metric+"_unknown"].(float64)
	return costs.Money(s), int(n), true
}

var costMetricNames = map[string]string{"forecast": "forecast final cost", "commitment": "approved contract", "budget": "budget", "claimed": "claimed to date"}

func costMoves(prev, now *Block) []move {
	a, b, ok := sameMethod(prev, now, methodCost, func(*Block) any { return &costBasis{} })
	if !ok {
		return notCompared(0, "cost", prev, now, methodCost)
	}
	pa, pb := a.(*costBasis), b.(*costBasis)
	var out []move
	for _, row := range append([]string{"total"}, CostSectionOrder...) {
		name := "Total"
		if row != "total" {
			name = CostSectionNames[row]
		}
		metrics := []string{"forecast", "commitment", "budget"}
		if row == "total" {
			metrics = append(metrics, "claimed")
		}
		for _, metric := range metrics {
			before, after := figureText(pa.Figures[row], metric), figureText(pb.Figures[row], metric)
			id := "cost:" + row + ":" + metric
			what := capitalise(costMetricNames[metric])
			if row != "total" {
				what = name + " " + costMetricNames[metric]
			}
			switch {
			case before == nil && after == nil:
				// Neither total is complete: say how the known part moved.
				wasKnown, _, wasOK := knownPart(pa.Figures[row], metric)
				isKnown, unknown, isOK := knownPart(pb.Figures[row], metric)
				if !isOK {
					continue
				}
				if !wasOK {
					wasKnown = "0"
				}
				diff := costs.Sub(isKnown, wasKnown)
				c := costs.Compare(diff, "0")
				if c == 0 {
					continue
				}
				dir := "up"
				if c < 0 {
					dir = "down"
				}
				out = append(out, move{area: 0, id: id + ":known", size: moneySize(diff), before: string(wasKnown), after: string(isKnown),
					text: fmt.Sprintf("%s: the known part is %s %s, from %s to %s; %s.", what, dir, figures.Dollars(strings.TrimPrefix(string(diff), "-")), figures.Dollars(string(wasKnown)), figures.Dollars(string(isKnown)), plural(unknown, "line is still not known", "lines are still not known"))})
			case before != nil && after == nil:
				out = append(out, move{area: 0, id: id, text: fmt.Sprintf("%s is no longer known; it was %s.", what, figures.Dollars(*before)), before: *before})
			case before == nil:
				out = append(out, move{area: 0, id: id, text: fmt.Sprintf("%s is now known: %s.", what, figures.Dollars(*after)), after: *after})
			default:
				diff := costs.Sub(costs.Money(*after), costs.Money(*before))
				c := costs.Compare(diff, "0")
				if c == 0 {
					continue
				}
				dir := "up"
				if c < 0 {
					dir = "down"
				}
				out = append(out, move{area: 0, id: id, size: moneySize(diff), before: *before, after: *after,
					text: fmt.Sprintf("%s %s %s, from %s to %s.", what, dir, figures.Dollars(strings.TrimPrefix(string(diff), "-")), figures.Dollars(*before), figures.Dollars(*after))})
			}
		}
	}
	return out
}

func moneySize(m costs.Money) float64 {
	r, ok := new(big.Rat).SetString(strings.TrimPrefix(string(m), "-"))
	if !ok {
		return 0
	}
	f, _ := r.Float64()
	return f
}

func programMoves(prev, now *Block) []move {
	a, b, ok := sameMethod(prev, now, methodCompletion, func(*Block) any { return &completionBasis{} })
	if !ok {
		return notCompared(1, "program", prev, now, methodCompletion)
	}
	pa, pb := a.(*completionBasis), b.(*completionBasis)
	var out []move
	nowProgram := &MonthlyProgram{Relative: pb.Relative, Cycle: pb.Cycle, Typical: pb.Typical, TypicalIndex: pb.TypicalIndex}
	if d, unit, ok := completionDelta(prev, nowProgram); ok && d != 0 {
		dir := "later"
		if d < 0 {
			dir = "earlier"
		}
		out = append(out, move{area: 1, id: "program:completion", size: math.Abs(float64(d)) * 1e9, before: pa.TypicalLabel, after: pb.TypicalLabel,
			text: fmt.Sprintf("Planned completion %s %s, from %s to %s.", plural(abs(d), unit, unit+"s"), dir, pa.TypicalLabel, pb.TypicalLabel)})
	} else if pa.TypicalLabel != "" && pb.TypicalLabel == "" {
		out = append(out, move{area: 1, id: "program:completion", size: 1e9, before: pa.TypicalLabel, text: "Planned completion is no longer known; it was " + pa.TypicalLabel + "."})
	}
	ids := make([]string, 0, len(pa.Milestones)+len(pb.Milestones))
	for id := range pa.Milestones {
		ids = append(ids, id)
	}
	for id := range pb.Milestones {
		if _, ok := pa.Milestones[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		was, wasOK := pa.Milestones[id]
		is, isOK := pb.Milestones[id]
		switch {
		case wasOK && !isOK:
			out = append(out, move{area: 1, id: "program:ms:" + id, size: 1, before: was.LabelDate, text: was.Label + " was removed from the Program."})
		case !wasOK && isOK:
			text := is.Label + " was added to the Program"
			if is.LabelDate != "" && !is.Unknown {
				text += ", planned for " + is.LabelDate
			}
			out = append(out, move{area: 1, id: "program:ms:" + id, size: 1, after: is.LabelDate, text: text + "."})
		default:
			d, ok := dayDiff(was.Date, is.Date)
			if !ok || d == 0 {
				continue
			}
			// The current label is printed: a renamed milestone is matched by id.
			out = append(out, move{area: 1, id: "program:ms:" + id, size: math.Abs(float64(d)), before: was.LabelDate, after: is.LabelDate,
				text: fmt.Sprintf("%s %s, from %s to %s.", is.Label, movedDays(d), was.LabelDate, is.LabelDate)})
		}
	}
	return out
}

type packagesBasis struct {
	Packages map[string]packageBasis `json:"packages"`
}

func packageMoves(prev, now *Block) []move {
	if prev != nil && now == nil {
		now = &Block{Basis: json.RawMessage(`{"method":"` + methodPackages + `","packages":{}}`)}
	}
	a, b, ok := sameMethod(prev, now, methodPackages, func(*Block) any { return &packagesBasis{} })
	if !ok {
		return notCompared(2, "procurement", prev, now, methodPackages)
	}
	pa, pb := a.(*packagesBasis), b.(*packagesBasis)
	ids := []string{}
	for id := range pa.Packages {
		ids = append(ids, id)
	}
	for id := range pb.Packages {
		if _, ok := pa.Packages[id]; !ok {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var out []move
	for _, id := range ids {
		was, wasOK := pa.Packages[id]
		is, isOK := pb.Packages[id]
		switch {
		case wasOK && !isOK:
			out = append(out, move{area: 2, id: "proc:" + id, size: 1, before: was.Status, text: was.Title + " was removed from procurement."})
		case !wasOK && isOK:
			out = append(out, move{area: 2, id: "proc:" + id, size: 1, after: is.Status, text: is.Title + " was added: " + lowerFirst(is.Status) + "."})
		case was.AwardedTo != "" && is.AwardedTo == "":
			out = append(out, move{area: 2, id: "proc:" + id, size: 3, before: was.Status, after: is.Status, text: is.Title + ": award to " + was.AwardedTo + " withdrawn; now " + lowerFirst(is.Status) + "."})
		case was.Status != is.Status:
			text := is.Title + ": " + was.Status + " → " + lowerFirst(is.Status)
			size := 1.0
			if is.AwardedTo != "" && was.AwardedTo == "" {
				text = is.Title + ": " + was.Status + " → awarded to " + is.AwardedTo
				if s, ok := is.Contract.(string); ok {
					text += ", " + figures.Dollars(s)
				}
				size = 2
			}
			out = append(out, move{area: 2, id: "proc:" + id, size: size, before: was.Status, after: is.Status, text: text + "."})
		}
	}
	return out
}

type registerKindBasis struct {
	Items map[string]registerBasis `json:"items"`
}

func registerMoves(many string, prev, now *Block) []move {
	if prev == nil && now == nil {
		return nil
	}
	empty := json.RawMessage(`{"method":"` + methodRegister + `","items":{}}`)
	if prev == nil {
		prev = &Block{Basis: empty}
	}
	if now == nil {
		now = &Block{Basis: empty}
	}
	a, b, ok := sameMethod(prev, now, methodRegister, func(*Block) any { return &registerKindBasis{} })
	if !ok {
		return notCompared(3, "registers:"+many, prev, now, methodRegister)
	}
	pa, pb := a.(*registerKindBasis), b.(*registerKindBasis)
	var opened, closed, removed []string
	for id, is := range pb.Items {
		was, ok := pa.Items[id]
		if is.Open && (!ok || !was.Open) {
			opened = append(opened, is.Title)
		}
		if ok && was.Open && !is.Open {
			closed = append(closed, is.Title)
		}
	}
	for id, was := range pa.Items {
		if _, ok := pb.Items[id]; !ok && was.Open {
			removed = append(removed, was.Title)
		}
	}
	var out []move
	for _, l := range []struct {
		titles []string
		verb   string
	}{{opened, "opened"}, {closed, "closed"}, {removed, "removed"}} {
		if len(l.titles) == 0 {
			continue
		}
		sort.Strings(l.titles)
		shown := l.titles
		if len(shown) > 3 {
			shown = append(append([]string{}, shown[:3]...), fmt.Sprintf("%d more", len(l.titles)-3))
		}
		out = append(out, move{area: 3, id: "reg:" + many + ":" + l.verb, size: float64(len(l.titles)), after: len(l.titles),
			text: fmt.Sprintf("%s %s: %s.", capitalise(plural(len(l.titles), strings.TrimSuffix(many, "s"), many)), l.verb, joinAnd(shown))})
	}
	return out
}

// documentsFiled is how many documents arrived this period, by discipline.
func (m monthly) documentsFiled() []move {
	start := m.periodStart()
	by := map[string]int{}
	n := 0
	for _, d := range m.in.Documents {
		if d.Received < start {
			continue
		}
		n++
		k := d.Discipline
		if k == "" {
			k = "with no discipline"
		}
		by[k]++
	}
	if n == 0 {
		return nil
	}
	keys := make([]string, 0, len(by))
	for k := range by {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if by[keys[i]] != by[keys[j]] {
			return by[keys[i]] > by[keys[j]]
		}
		return keys[i] < keys[j]
	})
	parts := []string{}
	for i, k := range keys {
		if i == 4 {
			parts = append(parts, fmt.Sprintf("%d other", n-sumFirst(by, keys[:4])))
			break
		}
		parts = append(parts, fmt.Sprintf("%d %s", by[k], strings.ReplaceAll(k, "_", " ")))
	}
	return []move{{area: 4, id: "documents", size: float64(n), after: n, text: fmt.Sprintf("%s filed since %s: %s.", capitalise(plural(n, "document", "documents")), LongDate(start), strings.Join(parts, ", "))}}
}

func sumFirst(by map[string]int, keys []string) int {
	n := 0
	for _, k := range keys {
		n += by[k]
	}
	return n
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
