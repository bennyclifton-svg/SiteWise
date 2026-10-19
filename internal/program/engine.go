package program

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

// The engine works on a node graph built once per Compute (plan 6.3, 6.5):
//
//   - node i is activity i;
//   - a stage with children ("summary") reuses its own node as the hidden
//     Start(S) node, with zero duration, and adds one hidden Finish(S) node;
//   - containment edges tie each child between them: Start(S) →SS→ child and
//     child →FF→ Finish(S), both lag 0;
//   - links on a summary are re-anchored: into it on Start(S) (FS, SS), out
//     of it from Finish(S) (FS, FF) or Start(S) (SS, SF).
//
// Hidden nodes are never returned as rows; a summary's row is rolled up from
// its children.

const (
	tFS = iota
	tSS
	tFF
	tSF
)

// edge is kept small (int32 fields) because a 2,000-row program has about
// 8,000 of them and the forward pass walks them three times.
type edge struct {
	from, to int32
	link     int32 // index into Input.Links, −1 for a containment edge
	lag      int32
	typ      uint8
	cal      bool
}

type graph struct {
	in      Input
	cal     *Cal
	s       *scratch
	n       int   // nodes: activities plus one Finish node per summary
	rows    int   // len(in.Activities)
	act     []int // node → activity index (a Finish node maps to its stage)
	finish  []int // activity → its Finish node, −1 when not a summary
	parent  []int // activity → parent activity index, −1 when top level
	kidOff  []int32
	kidIdx  []int32 // children of activity i are kidIdx[kidOff[i]:kidOff[i+1]]
	edges   []edge
	inOff   []int32 // CSR: in-edges of node v are inIdx[inOff[v]:inOff[v+1]]
	inIdx   []int32
	outOff  []int32
	outIdx  []int32
	order   []int // topological order; shorter than n when there is a cycle
	unknown []bool
	snet    []int // node → start-no-earlier index, −1 when none
	mso     []int // node → mandatory start index, −1 when none
}

// scratch is the engine's working memory, reused across calls through a
// pool. A read recomputes the whole program, and allocating about 1 MB of
// arrays every time costs more in garbage collection than the scheduling
// itself. Nothing in an Output points into scratch, so Compute stays pure
// for its callers.
type scratch struct {
	idx                             map[string]int
	act, finish, parent, order, deg []int
	snet, mso                       []int
	kidOff, kidIdx, next            []int32
	inOff, inIdx, outOff, outIdx    []int32
	edges                           []edge
	unknown                         []bool
	durT, durL, durH                []int
	es, ef, ls, lf, res, free       []int
	es2, ef2, res2                  []int
}

var scratchPool = sync.Pool{New: func() any { return &scratch{idx: map[string]int{}} }}

// grow returns *s resized to n and zeroed, reusing its array when it fits.
func grow[T any](s *[]T, n int) []T {
	if cap(*s) < n {
		*s = make([]T, n)
	} else {
		*s = (*s)[:n]
		clear(*s)
	}
	return *s
}

func linkType(s string) (int, bool) {
	switch s {
	case LinkFS, "":
		return tFS, true
	case LinkSS:
		return tSS, true
	case LinkFF:
		return tFF, true
	case LinkSF:
		return tSF, true
	}
	return 0, false
}

func buildGraph(in Input, sc *scratch) *graph {
	g := &graph{in: in, cal: NewCalendar(in.Calendar), s: sc, rows: len(in.Activities)}
	idx := sc.idx
	clear(idx)
	for i, a := range in.Activities {
		if _, dup := idx[a.ID]; !dup {
			idx[a.ID] = i
		}
	}
	g.parent = grow(&sc.parent, g.rows)
	g.kidOff = grow(&sc.kidOff, g.rows+1)
	for i, a := range in.Activities {
		g.parent[i] = -1
		p, ok := idx[a.ParentID]
		// Two tiers: only a top-level row can hold children. A row whose
		// parent is missing or is itself a child is scheduled as top level.
		if a.ParentID != "" && ok && p != i && in.Activities[p].ParentID == "" {
			g.parent[i] = p
			g.kidOff[p+1]++
		}
	}
	for i := 0; i < g.rows; i++ {
		g.kidOff[i+1] += g.kidOff[i]
	}
	g.kidIdx = grow(&sc.kidIdx, int(g.kidOff[g.rows]))
	next := grow(&sc.next, g.rows)
	copy(next, g.kidOff[:g.rows])
	for i, p := range g.parent {
		if p >= 0 {
			g.kidIdx[next[p]] = int32(i)
			next[p]++
		}
	}

	g.finish = grow(&sc.finish, g.rows)
	g.act = grow(&sc.act, g.rows)
	for i := range in.Activities {
		g.act[i] = i
		g.finish[i] = -1
	}
	for i := range in.Activities {
		if len(g.children(i)) > 0 {
			g.finish[i] = len(g.act)
			g.act = append(g.act, i)
		}
	}
	sc.act = g.act
	g.n = len(g.act)

	g.edges = sc.edges[:0]
	for li, l := range in.Links {
		p, okp := idx[l.Pred]
		s, oks := idx[l.Succ]
		t, okt := linkType(l.Type)
		if !okp || !oks || !okt || p == s {
			continue // Validate refuses these; Compute ignores them.
		}
		from, to := p, s
		if g.finish[p] >= 0 && (t == tFS || t == tFF) {
			from = g.finish[p]
		}
		// FF and SF into a summary are refused by ValidateLink. One can
		// still arrive when a stage with such a link gains its first child;
		// anchoring it on Finish(S) keeps its meaning ("the stage finishes
		// after …") until the user re-links it.
		if g.finish[s] >= 0 && (t == tFF || t == tSF) {
			to = g.finish[s]
		}
		g.edges = append(g.edges, edge{from: int32(from), to: int32(to), typ: uint8(t), lag: int32(l.Lag), cal: l.LagUnit == LagCalendar, link: int32(li)})
	}
	for s := range in.Activities {
		for _, c := range g.children(s) {
			g.edges = append(g.edges, edge{from: int32(s), to: c, typ: tSS, link: -1})
			g.edges = append(g.edges, edge{from: c, to: int32(g.finish[s]), typ: tFF, link: -1})
		}
	}
	sc.edges = g.edges
	g.inOff, g.inIdx = csr(g.n, g.edges, &sc.inOff, &sc.inIdx, &sc.next, func(e edge) int32 { return e.to })
	g.outOff, g.outIdx = csr(g.n, g.edges, &sc.outOff, &sc.outIdx, &sc.next, func(e edge) int32 { return e.from })
	g.topo()
	return g
}

func (g *graph) children(i int) []int32 { return g.kidIdx[g.kidOff[i]:g.kidOff[i+1]] }

// csr groups edge indices by node, keeping input order within a node so
// "first link wins" ties are stable.
func csr(n int, edges []edge, offBuf, idxBuf, nextBuf *[]int32, key func(edge) int32) ([]int32, []int32) {
	off := grow(offBuf, n+1)
	for _, e := range edges {
		off[key(e)+1]++
	}
	for v := 0; v < n; v++ {
		off[v+1] += off[v]
	}
	next := grow(nextBuf, n)
	copy(next, off[:n])
	idx := grow(idxBuf, len(edges))
	for i, e := range edges {
		k := key(e)
		idx[next[k]] = int32(i)
		next[k]++
	}
	return off, idx
}

// topo is Kahn's algorithm with a FIFO queue seeded in node order: O(V+E),
// no recursion. Dates do not depend on the order chosen; only the reported
// cycle would, and findCycle picks it by outline key instead.
func (g *graph) topo() {
	deg := grow(&g.s.deg, g.n)
	for _, e := range g.edges {
		deg[e.to]++
	}
	g.order = g.s.order[:0]
	for v := 0; v < g.n; v++ {
		if deg[v] == 0 {
			g.order = append(g.order, v)
		}
	}
	for k := 0; k < len(g.order); k++ {
		v := g.order[k]
		for _, ei := range g.outIdx[g.outOff[v]:g.outOff[v+1]] {
			w := g.edges[ei].to
			if deg[w]--; deg[w] == 0 {
				g.order = append(g.order, int(w))
			}
		}
	}
	g.s.order = g.order
}

func (g *graph) summary(i int) bool { return g.finish[i] >= 0 }

// leafDuration is a schedulable row's duration in one run. Milestones are
// always 0; every other kind is unknown until someone states a duration.
func leafDuration(a Activity, run int) (int, bool) {
	if a.Kind == KindMilestone {
		return 0, true
	}
	if a.Duration == nil {
		return 0, false
	}
	switch {
	case run == runLow && a.Low != nil:
		return *a.Low, true
	case run == runHigh && a.High != nil:
		return *a.High, true
	}
	return *a.Duration, true
}

const (
	runTypical = iota
	runLow
	runHigh
)

// durations fills per-node durations for one run. Hidden nodes are 0.
func (g *graph) durations(buf *[]int, run int) []int {
	d := grow(buf, g.n)
	for i, a := range g.in.Activities {
		if !g.summary(i) {
			d[i], _ = leafDuration(a, run)
		}
	}
	return d
}

// mark sets unknown propagation and constraint indices; both are the same in
// every run.
func (g *graph) mark() {
	g.unknown = grow(&g.s.unknown, g.n)
	g.snet = grow(&g.s.snet, g.n)
	g.mso = grow(&g.s.mso, g.n)
	for v := 0; v < g.n; v++ {
		g.snet[v], g.mso[v] = -1, -1
	}
	for i, a := range g.in.Activities {
		if !g.summary(i) {
			_, ok := leafDuration(a, runTypical)
			g.unknown[i] = !ok
		}
		// A summary's constraint holds its hidden start, and so its children.
		if idx, ok := g.cal.IndexOnOrAfter(a.ConstraintDate); ok {
			switch a.Constraint {
			case ConstraintSNET:
				g.snet[i] = idx
			case ConstraintMSO:
				g.mso[i] = max(0, idx) // nothing starts before the program
			}
		}
	}
	for _, v := range g.order {
		for _, ei := range g.inIdx[g.inOff[v]:g.inOff[v+1]] {
			if g.unknown[g.edges[ei].from] {
				g.unknown[v] = true
				break
			}
		}
	}
}

// pass holds one run's numbers per node; res is each edge's forward result
// (the earliest successor start that edge allows, before the clamp at 0).
type pass struct {
	dur, es, ef, ls, lf []int
	res                 []int
	pf                  int
	known               bool
	pfLast              bool // some row with duration > 0 ends at pf
}

// fwd is the forward rule of plan 6.2 for one edge.
func (g *graph) fwd(e edge, p *pass) int {
	u, db, lag := e.from, p.dur[e.to], int(e.lag)
	if !e.cal {
		switch e.typ {
		case tFS:
			return p.ef[u] + lag
		case tSS:
			return p.es[u] + lag
		case tFF:
			return p.ef[u] + lag - db
		default:
			return p.es[u] + lag - db
		}
	}
	c := g.cal
	switch e.typ {
	case tFS:
		return c.startOnOrAfter(c.finishDay(p.ef[u], p.dur[u]) + lag)
	case tSS:
		return c.startOnOrAfter(c.startDay(p.es[u]) + lag)
	case tFF:
		return c.finishOnOrAfter(c.finishDay(p.ef[u], p.dur[u])+lag, db) - db
	default:
		return c.finishOnOrAfter(c.startDay(p.es[u])+lag, db) - db
	}
}

// bwd is the backward rule of plan 6.4: the latest finish the edge allows its
// predecessor.
func (g *graph) bwd(e edge, p *pass) int {
	u, w, lag := e.from, e.to, int(e.lag)
	da := p.dur[u]
	if !e.cal {
		switch e.typ {
		case tFS:
			return p.ls[w] - lag
		case tSS:
			return p.ls[w] - lag + da
		case tFF:
			return p.lf[w] - lag
		default:
			return p.lf[w] - lag + da
		}
	}
	c := g.cal
	switch e.typ {
	case tFS:
		return c.finishOnOrBefore(c.startDay(p.ls[w])-lag, da)
	case tSS:
		return c.startOnOrBefore(c.startDay(p.ls[w])-lag) + da
	case tFF:
		return c.finishOnOrBefore(c.finishDay(p.lf[w], p.dur[w])-lag, da)
	default:
		return c.startOnOrBefore(c.finishDay(p.lf[w], p.dur[w])-lag) + da
	}
}

// forward fills es, ef, res and pf; the caller sizes the arrays. conflict is
// called for each mandatory start held later by a link, with the holding
// edge.
func (g *graph) forward(p *pass, conflict func(v, held, ei int)) {
	for _, v := range g.order {
		if g.unknown[v] {
			continue
		}
		best, bestEdge := math.MinInt, int32(-1)
		for _, ei := range g.inIdx[g.inOff[v]:g.inOff[v+1]] {
			r := g.fwd(g.edges[ei], p)
			p.res[ei] = r
			if r > best {
				best, bestEdge = r, ei
			}
		}
		es := max(0, g.snet[v], best)
		if m := g.mso[v]; m >= 0 {
			es = m
			if best > m && conflict != nil {
				conflict(v, best, int(bestEdge))
			}
		}
		p.es[v], p.ef[v] = es, es+p.dur[v]
		if !p.known || p.ef[v] > p.pf {
			p.pf, p.known, p.pfLast = p.ef[v], true, false
		}
		if p.ef[v] == p.pf && p.dur[v] > 0 {
			p.pfLast = true
		}
	}
}

// backward fills ls, lf and free float per node.
func (g *graph) backward(p *pass, free []int) {
	for k := len(g.order) - 1; k >= 0; k-- {
		v := g.order[k]
		if g.unknown[v] {
			continue
		}
		lf, ff, any := p.pf, math.MaxInt, false
		for _, ei := range g.outIdx[g.outOff[v]:g.outOff[v+1]] {
			e := g.edges[ei]
			if g.unknown[e.to] {
				continue
			}
			lf = min(lf, g.bwd(e, p))
			ff = min(ff, p.es[e.to]-p.res[ei])
			any = true
		}
		if !any {
			ff = p.pf - p.ef[v]
		}
		p.lf[v], p.ls[v], free[v] = lf, lf-p.dur[v], ff
	}
}

// Compute schedules the program (plan 6.3–6.5). It is pure and O(V+E) per
// run; it runs the forward pass three times (typical, low, high) and the
// backward pass once, on the typical durations the bars draw.
func Compute(in Input) Output {
	sc := scratchPool.Get().(*scratch)
	defer scratchPool.Put(sc)
	g := buildGraph(in, sc)
	out := Output{Activities: RowTimes{}, DrivingLinks: []string{}, Conflicts: []Conflict{}}
	out.Status = status(g)
	if len(g.order) < g.n {
		out.Cycle = g.findCycle()
		return out
	}
	g.mark()

	n, m := g.n, len(g.edges)
	typ := &pass{dur: g.durations(&sc.durT, runTypical), es: grow(&sc.es, n), ef: grow(&sc.ef, n),
		ls: grow(&sc.ls, n), lf: grow(&sc.lf, n), res: grow(&sc.res, m)}
	held := map[int]Conflict{}
	g.forward(typ, func(v, h, ei int) {
		c := Conflict{ActivityID: in.Activities[g.act[v]].ID, Index: g.mso[v], HeldIndex: h, HeldBy: in.Activities[g.act[g.edges[ei].from]].ID}
		if l := g.edges[ei].link; l >= 0 {
			c.LinkID = in.Links[l].ID
		}
		held[v] = c
	})
	free := grow(&sc.free, n)
	g.backward(typ, free)
	low, high := typ, typ
	if hasRanges(in.Activities) {
		// Only the project finish is wanted from these runs, so they share
		// one set of arrays.
		low = &pass{dur: g.durations(&sc.durL, runLow), es: grow(&sc.es2, n), ef: grow(&sc.ef2, n), res: grow(&sc.res2, m)}
		g.forward(low, nil)
		high = &pass{dur: g.durations(&sc.durH, runHigh), es: low.es, ef: low.ef, res: low.res}
		g.forward(high, nil)
	}

	g.rowTimes(&out, typ, free)
	out.Finish = Finish{Complete: out.Status.Known == out.Status.Total}
	out.Finish.Typical, out.Finish.TypicalIndex = g.finishOf(typ)
	out.Finish.Low, out.Finish.LowIndex = g.finishOf(low)
	out.Finish.High, out.Finish.HighIndex = g.finishOf(high)

	// Driving links follow input link order, so both engines list them the
	// same way. Each link is one edge, so a link appears at most once.
	flags, count := make([]bool, len(in.Links)), 0
	for ei, e := range g.edges {
		if e.link >= 0 && !g.unknown[e.to] && typ.res[ei] == typ.es[e.to] {
			flags[e.link] = true
			count++
		}
	}
	out.DrivingLinks = make([]string, 0, count)
	for li, l := range in.Links {
		if flags[li] {
			out.DrivingLinks = append(out.DrivingLinks, l.ID)
		}
	}
	// Conflicts follow input row order, not the topological order.
	for i := range in.Activities {
		if c, ok := held[i]; ok {
			out.Conflicts = append(out.Conflicts, c)
		}
	}
	return out
}

func hasRanges(acts []Activity) bool {
	for _, a := range acts {
		if a.Low != nil || a.High != nil {
			return true
		}
	}
	return false
}

func (g *graph) finishOf(p *pass) (*string, *int) {
	if !p.known {
		return nil, nil
	}
	pf := p.pf
	var date *string
	if !g.cal.Relative() {
		// The finish date is the last working day, unless only zero-length
		// rows reach pf (a closing milestone), which show their own day.
		s := g.cal.DateOf(pf)
		if p.pfLast {
			s = g.cal.DateOf(pf - 1)
		}
		date = &s
	}
	return date, &pf
}

// rowTimes writes one Times per activity. Dates are substrings of the
// calendar's warm cache, so a 2,000-row program costs a handful of
// allocations, not one per date.
func (g *graph) rowTimes(out *Output, p *pass, free []int) {
	lo, hi := math.MaxInt, math.MinInt
	for i := range g.in.Activities {
		if es, ef, _, _, _, _, ok := g.rowNumbers(i, p, free); ok {
			lo, hi = min(lo, es), max(hi, ef-1, es)
		}
	}
	g.cal.warm(lo, hi, g.rows)
	out.Activities = make(RowTimes, 0, g.rows)
	for i, a := range g.in.Activities {
		es, ef, ls, lf, tf, ff, ok := g.rowNumbers(i, p, free)
		if !ok {
			out.Activities = append(out.Activities, Times{ID: a.ID, Unknown: true})
			continue
		}
		start, finish := g.cal.Display(es, ef-es)
		out.Activities = append(out.Activities, Times{ID: a.ID, ES: es, EF: ef, LS: ls, LF: lf, TF: tf, FF: ff,
			Critical: tf <= 0, Start: start, Finish: finish})
	}
}

// rowNumbers rolls a summary up from its children: the bar spans the earliest
// start to the latest finish; late dates likewise; total float is the least
// float of any child (the stage is critical when any child is); free float is
// that of the hidden Finish node, how far the stage end can slip before it
// holds a successor.
func (g *graph) rowNumbers(i int, p *pass, free []int) (es, ef, ls, lf, tf, ff int, ok bool) {
	if !g.summary(i) {
		if g.unknown[i] {
			return
		}
		return p.es[i], p.ef[i], p.ls[i], p.lf[i], p.ls[i] - p.es[i], free[i], true
	}
	fin := g.finish[i]
	if g.unknown[fin] {
		return
	}
	es, ef, ls, lf, tf = math.MaxInt, math.MinInt, math.MaxInt, math.MinInt, math.MaxInt
	for _, c := range g.children(i) {
		es, ef = min(es, p.es[c]), max(ef, p.ef[c])
		ls, lf = min(ls, p.ls[c]), max(lf, p.lf[c])
		tf = min(tf, p.ls[c]-p.es[c])
	}
	return es, ef, ls, lf, tf, free[fin], true
}

// status counts schedulable rows: every row except a stage with children,
// whose span comes from them.
func status(g *graph) Status {
	s := Status{}
	for i, a := range g.in.Activities {
		if g.summary(i) {
			continue
		}
		s.Total++
		if _, ok := leafDuration(a, runTypical); ok {
			s.Known++
		}
	}
	s.Words = StatusWords(s.Known, s.Total)
	return s
}

// StatusWords is the toolbar line, e.g. "Indicative: 31 of 44 durations known".
func StatusWords(known, total int) string {
	switch {
	case total == 0:
		return "No activities yet"
	case known == total:
		return fmt.Sprintf("All %d durations known", total)
	}
	return fmt.Sprintf("Indicative: %d of %d durations known", known, total)
}

// key orders nodes for the cycle walk: the outline path of ordinals, then id.
// A stage sorts as (ordinal, −1), its child as (stage ordinal, ordinal), and a
// hidden Finish node as (stage ordinal, MaxInt).
func (g *graph) less(a, b int) bool {
	ka0, ka1, ida := g.key(a)
	kb0, kb1, idb := g.key(b)
	if ka0 != kb0 {
		return ka0 < kb0
	}
	if ka1 != kb1 {
		return ka1 < kb1
	}
	if ida != idb {
		return ida < idb
	}
	return a < b // the finish node of a stage sorts after the stage
}

func (g *graph) key(v int) (int, int, string) {
	i := g.act[v]
	a := g.in.Activities[i]
	if v >= g.rows {
		return a.Ordinal, math.MaxInt, a.ID
	}
	if p := g.parent[i]; p >= 0 {
		return g.in.Activities[p].Ordinal, a.Ordinal, a.ID
	}
	return a.Ordinal, -1, a.ID
}

// findCycle returns one loop as closed activity ids. Every node Kahn could not
// order has a predecessor that also could not be ordered, so walking back
// from any such node must revisit one; the revisited stretch is a real loop.
// The walk starts at the least key and always steps to the least-key
// predecessor, then the loop is rotated to begin at its least key, so the
// answer depends only on the input, not on the order Kahn ran in.
func (g *graph) findCycle() []string {
	ordered := make([]bool, g.n)
	for _, v := range g.order {
		ordered[v] = true
	}
	start := -1
	for v := 0; v < g.n; v++ {
		if !ordered[v] && (start < 0 || g.less(v, start)) {
			start = v
		}
	}
	seen := map[int]int{}
	var walk []int
	for v := start; ; {
		if at, ok := seen[v]; ok {
			walk = walk[at:]
			break
		}
		seen[v] = len(walk)
		walk = append(walk, v)
		next := -1
		for _, ei := range g.inIdx[g.inOff[v]:g.inOff[v+1]] {
			u := int(g.edges[ei].from)
			if !ordered[u] && (next < 0 || g.less(u, next)) {
				next = u
			}
		}
		v = next
	}
	// walk runs against the links; reverse it to read forwards.
	for l, r := 0, len(walk)-1; l < r; l, r = l+1, r-1 {
		walk[l], walk[r] = walk[r], walk[l]
	}
	least := 0
	for k := range walk {
		if g.less(walk[k], walk[least]) {
			least = k
		}
	}
	walk = append(walk[least:], walk[:least]...)
	// Start(S) → child → Finish(S) is the loop passing through stage S; name
	// the stage, not whichever child the walk happened to cross.
	through := func(k int) bool {
		v := walk[k]
		p := -1
		if v < g.rows {
			p = g.parent[v]
		}
		prev, next := walk[(k+len(walk)-1)%len(walk)], walk[(k+1)%len(walk)]
		return p >= 0 && prev == p && next == g.finish[p]
	}
	// Hidden nodes read as their stage; collapse repeats they leave behind.
	ids := make([]string, 0, len(walk)+1)
	for k, v := range walk {
		if len(walk) > 2 && through(k) {
			continue
		}
		id := g.in.Activities[g.act[v]].ID
		if len(ids) == 0 || ids[len(ids)-1] != id {
			ids = append(ids, id)
		}
	}
	if len(ids) > 1 && ids[len(ids)-1] == ids[0] {
		ids = ids[:len(ids)-1]
	}
	return append(ids, ids[0])
}

// FindCycle returns one loop in the program as closed activity ids, or nil.
func FindCycle(in Input) []string {
	sc := scratchPool.Get().(*scratch)
	defer scratchPool.Put(sc)
	g := buildGraph(in, sc)
	if len(g.order) == g.n {
		return nil
	}
	return g.findCycle()
}

// CycleWords names a loop by label in order, e.g. "Framing → Roof → Framing".
func CycleWords(in Input, cycle []string) string {
	label := make(map[string]string, len(in.Activities))
	for _, a := range in.Activities {
		label[a.ID] = a.Label
		if a.Label == "" {
			label[a.ID] = a.ID
		}
	}
	words := make([]string, len(cycle))
	for k, id := range cycle {
		words[k] = label[id]
	}
	return strings.Join(words, " → ")
}

// LinkError is a refused link. Cycle holds the loop's ids when the link would
// close one, so the response can name and highlight it.
type LinkError struct {
	Message string
	Cycle   []string
}

func (e *LinkError) Error() string { return e.Message }

// Refusal messages shared by the store and the API.
const (
	MsgLinkIntoStage = "Link to an activity inside the stage instead"
	MsgLinkOwnStage  = "A row can't link to its own stage."
	MsgLinkSelf      = "A row can't link to itself."
	MsgLinkMissing   = "That link joins a row that is not in this program."
	MsgLinkDuplicate = "These two rows are already linked."
	MsgLinkType      = "Link type must be FS, SS, FF or SF."
	MsgLinkLagUnit   = "Lag unit must be working or calendar."
	MsgLinkLag       = "Lag must be between −365 and 365 days."
)

// ValidateLink checks a link about to be added or updated (same id replaces
// the stored one) against the program, including the cycle it would close.
// It returns nil or a *LinkError.
func ValidateLink(in Input, l Link) error {
	idx := map[string]Activity{}
	kids := map[string]bool{}
	for _, a := range in.Activities {
		idx[a.ID] = a
		if a.ParentID != "" {
			kids[a.ParentID] = true
		}
	}
	p, okp := idx[l.Pred]
	s, oks := idx[l.Succ]
	t, okt := linkType(l.Type)
	switch {
	case !okp || !oks:
		return &LinkError{Message: MsgLinkMissing}
	case l.Pred == l.Succ:
		return &LinkError{Message: MsgLinkSelf}
	case p.ParentID == s.ID || s.ParentID == p.ID:
		return &LinkError{Message: MsgLinkOwnStage}
	case !okt:
		return &LinkError{Message: MsgLinkType}
	case l.LagUnit != "" && l.LagUnit != LagWorking && l.LagUnit != LagCalendar:
		return &LinkError{Message: MsgLinkLagUnit}
	case l.Lag < -365 || l.Lag > 365:
		return &LinkError{Message: MsgLinkLag}
	case kids[s.ID] && (t == tFF || t == tSF):
		return &LinkError{Message: MsgLinkIntoStage}
	}
	links := make([]Link, 0, len(in.Links)+1)
	for _, x := range in.Links {
		if l.ID != "" && x.ID == l.ID {
			continue
		}
		if x.Pred == l.Pred && x.Succ == l.Succ {
			return &LinkError{Message: MsgLinkDuplicate}
		}
		links = append(links, x)
	}
	trial := in
	trial.Links = append(links, l)
	if c := FindCycle(trial); c != nil {
		return &LinkError{Message: "This link would make a loop: " + CycleWords(in, c), Cycle: c}
	}
	return nil
}

// Validate checks inputs at the boundary, where the store turns rows into an
// Input. Compute itself tolerates anything and never fails.
func Validate(in Input) error {
	if in.Calendar.Start != "" {
		if _, ok := dayNumber(in.Calendar.Start); !ok {
			return fmt.Errorf("start date %q is not YYYY-MM-DD", in.Calendar.Start)
		}
	}
	if w := in.Calendar.WorkingWeek; w < 0 || w > 127 {
		return fmt.Errorf("working week %d is not a Mon=1 … Sun=64 mask", w)
	}
	for _, d := range in.Calendar.Days {
		if _, ok := dayNumber(d.Day); !ok {
			return fmt.Errorf("calendar day %q is not YYYY-MM-DD", d.Day)
		}
		if !validDayKind(d.Kind) {
			return fmt.Errorf("calendar day %s has unknown kind %q", d.Day, d.Kind)
		}
	}
	for _, a := range in.Activities {
		if a.Duration != nil && (*a.Duration < 0 || *a.Duration > MaxDuration) {
			return fmt.Errorf("%s: duration must be 0 to 3,650 working days", a.ID)
		}
		if (a.Low == nil) != (a.High == nil) || (a.Low != nil && *a.Low > *a.High) {
			return fmt.Errorf("%s: low and high come together, low first", a.ID)
		}
		switch a.Constraint {
		case "", ConstraintASAP:
			if a.ConstraintDate != "" {
				return fmt.Errorf("%s: as soon as possible takes no date", a.ID)
			}
		case ConstraintSNET, ConstraintMSO:
			if _, ok := dayNumber(a.ConstraintDate); !ok {
				return fmt.Errorf("%s: constraint date %q is not YYYY-MM-DD", a.ID, a.ConstraintDate)
			}
		default:
			return fmt.Errorf("%s: constraint must be asap, snet or mso", a.ID)
		}
	}
	return nil
}

// FirstChildMove gives a leaf stage's duration (and range) to its first
// activity and clears it from the stage, which keeps its links (they move to
// its hidden nodes by themselves). moved is false when the stage had no
// duration to give. The caller writes both rows and shows the toast
// ("Design's 40 days moved to Concept design").
func FirstChildMove(stage, child Activity) (Activity, Activity, bool) {
	if stage.Duration == nil {
		return stage, child, false
	}
	child.Duration, child.Low, child.High = stage.Duration, stage.Low, stage.High
	stage.Duration, stage.Low, stage.High = nil, nil, nil
	return stage, child, true
}

// StageSpan is a stage's current span in working days (ef − es), the
// duration it takes back when its last activity is removed. ok is false when
// the span is unknown, which leaves the stage's duration unknown too.
func StageSpan(out Output, stageID string) (int, bool) {
	t, ok := out.Activities.Get(stageID)
	if !ok || t.Unknown {
		return 0, false
	}
	return t.EF - t.ES, true
}
