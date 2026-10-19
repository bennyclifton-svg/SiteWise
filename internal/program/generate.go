package program

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"unicode/utf8"

	"sitewise/internal/knowledge"
)

// Generation (plan 7.2) turns the Profile, the included work items, the live
// packages and the program library into a proposed program. It is pure: no
// database, no Jev, no clock. The same GenInput always gives the same Plan,
// so the store can insert it, the question strip can regenerate it in memory
// and the review can diff it against the live program by source_key.
//
// Every label comes from the library, the stage catalogue, a package, a work
// item or a code template stating what was matched; every duration comes
// from the user, an accepted reading, a Profile planning value or a reviewed
// library row. Anything else stays unknown.

// Row origins, review statuses and duration origins written by generation.
const (
	OriginLibrary  = "library"
	OriginUser     = "user"
	OriginDocument = "document"

	ReviewProposed = "proposed"
	ReviewAccepted = "accepted"

	DurationUser        = "user"
	DurationDocument    = "document"
	DurationProfile     = "profile"
	DurationLibrary     = "library"
	DurationCalculation = "calculation"

	LinkOriginChain     = "chain"
	LinkOriginLibrary   = "library"
	LinkOriginInterface = "interface"
)

// CheckLagUnknown is raised for a library link whose lag names a duration
// that is not reviewed or cannot be computed. The link is kept with no lag
// (see addLibraryLink) and this check makes the gap visible.
const CheckLagUnknown = "lag_unknown"

// CheckLinkSkipped is raised when a library link could not be placed: it
// would close a loop, or ends FF or SF on a stage that has activities.
const CheckLinkSkipped = "link_skipped"

// Planning keys and catalogue stages generation reads by name.
const (
	planConstructionDuration = "construction_duration"
	planTargetCompletion     = "target_completion"
	stageConstruction        = "construction"
	stageProcurement         = "procurement"
	stageInvestigation       = "investigation"
	actionRetain             = "retain"
	actionInvestigate        = "investigate"
	actionRemove             = "remove"
	maxLabel                 = 200
)

// GenInput is everything plan 7.1 lists, as plain values the store reads
// with ordinary queries. Nothing here is fetched by generation itself.
type GenInput struct {
	// Values are known determinant values by id (comma-separated for
	// multi-choice). work_type is not read from here: it comes from WorkTypes.
	Values map[string]string
	// WorkTypes are the Profile's applied work types (taxonomy ids).
	WorkTypes []string
	// Conditions are Profile condition values by taxonomy key, e.g.
	// planning = "da", procurement_route = "traditional" (owner answer Q1).
	Conditions map[string]string
	// Planning are Profile planning values by key, without "plan.".
	Planning map[string]string
	Facts    Facts
	// Present and Existing say whether a system is in the completed building
	// or on site now. A system with no entry (and no true descendant) is
	// unknown. Nil maps make every such test unknown.
	Present  map[string]bool
	Existing map[string]bool
	// WorkItems are the included, live work items in Profile order.
	WorkItems []WorkItem
	// Packages are live, not cancelled packages in Procurement order, each
	// with its live package stages in ordinal order.
	Packages []Package
	// Budget is the Cost Plan construction budget (read-only). Nil is unknown.
	Budget *Budget
	// IndexCost converts a cost at priceDate to a row's cost_base_date for
	// the time_cost basis. Nil means no index is available, so time_cost
	// durations stay unknown rather than use an unindexed cost.
	IndexCost func(cost *big.Rat, priceDate, baseDate string) (*big.Rat, bool)
	// UserDurations are durations the user typed, by source_key.
	UserDurations map[string]int
	// DocumentDurations are accepted document readings by source_key, and
	// StageReadings are accepted readings for a stage by catalogue stage id
	// (plan 7.7: evidence follows the catalogue id, not the row).
	DocumentDurations map[string]Reading
	StageReadings     map[string]Reading
	// WorkingWeek is the program's Mon=1 … Sun=64 mask; 0 is Mon–Fri. It
	// turns construction_duration weeks into working days.
	WorkingWeek int
}

// Facts are the Profile facts generation reads.
type Facts struct {
	ConsentDate            string
	ConsentLapseDate       string
	DefectsLiabilityPeriod string
	ContractForm           string
}

// WorkItem is one included work item.
type WorkItem struct {
	ID           string
	System       string
	Part         string
	Action       string
	Title        string
	LayoutChange string
}

// Package is one live package with its stages.
type Package struct {
	ID           string
	Title        string
	Kind         string // services, works or supply
	DisciplineID string
	Stages       []PackageStage
}

// PackageStage is one live package stage.
type PackageStage struct {
	ID               string
	CatalogueStageID string
	Label            string
}

// Budget is the construction budget, a single value (Low == High) or a range.
type Budget struct {
	Low, High *big.Rat
	PriceDate string
}

// Reading is an accepted document value in working days.
type Reading struct {
	Days      int
	ReadingID string
}

// Row is one program row to insert. ParentKey names the parent stage's
// SourceKey; links name rows by SourceKey too, so the store can insert rows,
// map keys to ids, then insert links.
type Row struct {
	SourceKey        string         `json:"source_key"`
	ParentKey        string         `json:"parent_key,omitempty"`
	Ordinal          int            `json:"ordinal"`
	Kind             string         `json:"kind"`
	Label            string         `json:"label"`
	CatalogueStageID string         `json:"catalogue_stage_id,omitempty"`
	PackageID        string         `json:"package_id,omitempty"`
	PackageStageID   string         `json:"package_stage_id,omitempty"`
	WorkItemID       string         `json:"work_item_id,omitempty"`
	FragmentID       string         `json:"fragment_id,omitempty"`
	InterfaceID      string         `json:"interface_id,omitempty"`
	Systems          []string       `json:"systems"`
	Duration         *int           `json:"duration_days"`
	Low              *int           `json:"duration_low"`
	High             *int           `json:"duration_high"`
	DurationOrigin   string         `json:"duration_origin,omitempty"`
	DurationSource   map[string]any `json:"duration_source"`
	ReviewStatus     string         `json:"review_status"`
	Origin           string         `json:"origin"`

	// Generation-only bookkeeping, not stored.
	durationID  string
	conditional bool
	item        *WorkItem
}

// PlanLink is one link to insert, by row SourceKey.
type PlanLink struct {
	SourceKey   string `json:"source_key"`
	PredKey     string `json:"pred_key"`
	SuccKey     string `json:"succ_key"`
	Type        string `json:"type"`
	Lag         int    `json:"lag_days"`
	LagUnit     string `json:"lag_unit"`
	Origin      string `json:"origin"`
	InterfaceID string `json:"interface_id,omitempty"`
	// LagDurationID is the library lag the link names, and LagUnknown says
	// it could not be resolved, so Lag is 0 and a check says so.
	LagDurationID string `json:"lag_duration_id,omitempty"`
	LagUnknown    bool   `json:"lag_unknown,omitempty"`
}

// StageMapping is where a package stage landed, for writing
// package_stages.program_stage_id back. Rule says how it was placed.
type StageMapping struct {
	PackageID      string `json:"package_id"`
	PackageStageID string `json:"package_stage_id"`
	StageKey       string `json:"stage_key"`
	Rule           string `json:"rule"`
}

// Stage placement rules (see resolveStage).
const (
	PlaceExact    = "exact"
	PlaceParent   = "parent"
	PlaceSubstage = "first_substage"
	PlaceNearest  = "nearest_earlier"
	PlaceFirst    = "first_stage"
)

// Plan is a generated program.
type Plan struct {
	SkeletonID    string `json:"skeleton_id"`
	SkeletonLabel string `json:"skeleton_label"`
	// Reason is why the skeleton was chosen, from the predicate terms that
	// held, for the completion header.
	Reason         string         `json:"reason"`
	SkeletonTie    bool           `json:"skeleton_tie"`
	Ties           []string       `json:"ties,omitempty"`
	LibraryVersion string         `json:"library_version"`
	Rows           []Row          `json:"rows"`
	Links          []PlanLink     `json:"links"`
	StageMap       []StageMapping `json:"stage_map"`
	// ConstructionStageKey is the stage the consent-lapse check reads.
	ConstructionStageKey string `json:"construction_stage_key,omitempty"`
	// StartSuggestion is the consent date, offered as a one-tap start and
	// never applied (plan 7.2 step 9). The program start stays null.
	StartSuggestion string `json:"start_suggestion,omitempty"`
	// Notes are generation checks (skeleton tie, unknown lag, skipped link).
	Notes []Check `json:"notes"`
}

// Row finds a row by source key.
func (p Plan) Row(key string) (Row, bool) {
	for _, r := range p.Rows {
		if r.SourceKey == key {
			return r, true
		}
	}
	return Row{}, false
}

// Input builds the engine input for in-memory scoring and for the first
// computed response. Row ids are source keys.
func (p Plan) Input(cal Calendar) Input {
	in := Input{Calendar: cal, Activities: make([]Activity, 0, len(p.Rows)), Links: make([]Link, 0, len(p.Links))}
	for _, r := range p.Rows {
		in.Activities = append(in.Activities, Activity{ID: r.SourceKey, ParentID: r.ParentKey, Ordinal: r.Ordinal,
			Kind: r.Kind, Label: r.Label, Duration: r.Duration, Low: r.Low, High: r.High})
	}
	for _, l := range p.Links {
		in.Links = append(in.Links, Link{ID: l.SourceKey, Pred: l.PredKey, Succ: l.SuccKey, Type: l.Type, Lag: l.Lag, LagUnit: l.LagUnit})
	}
	return in
}

// CheckInput gathers what Checks needs from the plan and the Profile. When
// the store computes over stored rows it passes its own ids instead; the
// construction stage is then the row whose source key is
// ConstructionStageKey.
func (p Plan) CheckInput(in Input, out Output, gi GenInput, calendarYears []int) CheckInput {
	return CheckInput{Input: in, Output: out, ConstructionStageID: p.ConstructionStageKey,
		ConsentLapseDate: gi.Facts.ConsentLapseDate, TargetCompletion: gi.Planning[planTargetCompletion],
		CalendarYears: calendarYears, SkeletonTie: p.SkeletonTie}
}

// Generate builds the proposed program. It fails only when no program
// library is loaded (no sk.generic).
func Generate(cat *knowledge.Catalog, in GenInput) (Plan, error) {
	if cat == nil {
		return Plan{}, fmt.Errorf("no knowledge catalogue")
	}
	g := newGen(cat, in)
	choice, ok := cat.MatchSkeleton(g.env(nil))
	if !ok {
		return Plan{}, fmt.Errorf("no program library: sk.generic is missing")
	}
	g.plan.SkeletonID, g.plan.SkeletonLabel = choice.Skeleton.ID, choice.Skeleton.Label
	g.plan.Reason = g.reason(choice.Skeleton)
	g.plan.LibraryVersion = cat.LibraryVersion()
	g.plan.StartSuggestion = in.Facts.ConsentDate
	if len(choice.Ties) > 0 {
		g.plan.SkeletonTie, g.plan.Ties = true, append([]string(nil), choice.Ties...)
		g.note(CheckSkeletonTie, "", "Two program templates fit; using the one with more conditions.")
	}

	// Rows first, then links: the loop guard needs to know which stages
	// have activities before any link is placed.
	g.stageRows(choice.Skeleton)
	g.packageRows()
	g.investigationRows()
	g.fragmentRows()
	g.holdPointRows()

	g.stageLinks(choice.Skeleton)
	g.fragmentLinks()
	g.holdPointLinks()
	g.chainLinks()

	g.durations()
	g.plan.ConstructionStageKey, _ = g.resolveStage(stageConstruction)
	return g.plan, nil
}

// gen is one generation run.
type gen struct {
	cat  *knowledge.Catalog
	in   GenInput
	plan Plan

	byKey      map[string]int    // row index by source key
	stageByCat map[string]string // program stage key by catalogue id
	stageOrder []string          // program stage keys in order
	kids       map[string]int    // child count per stage key
	catPos     map[string]int    // catalogue position of each stage id
	catParent  map[string]string
	pairs      map[[2]string]int // link index by (pred, succ)
	guard      *loopGuard

	fragInstances []fragInstance
	holds         []holdRow
	anchored      map[string]bool // rows with a placed predecessor
	hasPred       map[string]bool
}

type fragInstance struct {
	frag     knowledge.Fragment
	item     WorkItem
	keys     map[string]string // activity key → row source key
	sources  []string          // rows with no internal predecessor
	sinks    []string          // rows with no internal successor
	stageKey string
}

type holdRow struct {
	key, iface string
	after      []string
	before     []string
}

func newGen(cat *knowledge.Catalog, in GenInput) *gen {
	g := &gen{cat: cat, in: in, byKey: map[string]int{}, stageByCat: map[string]string{}, kids: map[string]int{},
		catPos: map[string]int{}, catParent: map[string]string{}, pairs: map[[2]string]int{},
		anchored: map[string]bool{}, hasPred: map[string]bool{}}
	g.plan.Rows, g.plan.Links, g.plan.StageMap, g.plan.Notes = []Row{}, []PlanLink{}, []StageMapping{}, []Check{}
	for i, s := range cat.DeliveryStages() {
		g.catPos[s.ID] = i
		g.catParent[s.ID] = s.ParentID
	}
	return g
}

func (g *gen) note(code, activity, msg string) {
	g.plan.Notes = append(g.plan.Notes, Check{Code: code, ActivityID: activity, Message: msg})
}

// env is the predicate environment. items nil means every included item;
// a fragment is evaluated against its one work item (see fragmentRows).
func (g *gen) env(items []WorkItem) knowledge.WorksEnv {
	if items == nil {
		items = g.in.WorkItems
	}
	env := knowledge.WorksEnv{Values: g.in.Values, WorkTypes: g.in.WorkTypes, Conditions: g.in.Conditions}
	for _, it := range items {
		env.Items = append(env.Items, knowledge.WorkItem{System: it.System, Action: it.Action, LayoutChange: it.LayoutChange})
	}
	if env.Items == nil {
		env.Items = []knowledge.WorkItem{}
	}
	if g.in.Present != nil {
		env.PresentState = g.systemState(g.in.Present)
	}
	if g.in.Existing != nil {
		env.Existing = g.systemState(g.in.Existing)
	}
	return env
}

// systemState: true when the system or a descendant is recorded true, false
// when the system itself is recorded false, otherwise unknown.
func (g *gen) systemState(m map[string]bool) func(string) knowledge.Truth {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return func(sys string) knowledge.Truth {
		for _, k := range keys {
			if m[k] && g.cat.SystemWithin(k, sys) {
				return knowledge.True
			}
		}
		if v, ok := m[sys]; ok && !v {
			return knowledge.False
		}
		return knowledge.Unknown
	}
}

func (g *gen) addRow(r Row) int {
	if r.Systems == nil {
		r.Systems = []string{}
	}
	r.DurationSource = map[string]any{}
	r.ReviewStatus, r.Origin = ReviewProposed, OriginLibrary
	r.Label = clip(r.Label)
	if r.ParentKey != "" {
		r.Ordinal = g.kids[r.ParentKey]
		g.kids[r.ParentKey]++
	}
	g.byKey[r.SourceKey] = len(g.plan.Rows)
	g.plan.Rows = append(g.plan.Rows, r)
	return len(g.plan.Rows) - 1
}

// clip keeps a label within the schema's 200 characters without splitting a
// character. A package title plus a stage label can exceed it.
func clip(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maxLabel {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:maxLabel]))
}

// Step 2: one stage row per skeleton stage, labelled from the catalogue.
func (g *gen) stageRows(sk knowledge.Skeleton) {
	for i, id := range sk.Stages {
		st, _ := g.cat.DeliveryStage(id)
		key := "skeleton:" + sk.ID + ":" + id
		g.addRow(Row{SourceKey: key, Ordinal: i, Kind: KindStage, Label: st.Label, CatalogueStageID: id})
		g.stageByCat[id] = key
		g.stageOrder = append(g.stageOrder, key)
	}
}

// resolveStage places a catalogue stage in the program. The rule, in order:
//
//  1. exact: the program has a stage with that catalogue id;
//  2. parent: the catalogue stage is a sub-stage and its parent is a program
//     stage (Construction-stage inspections → Construction);
//  3. first_substage: it is a top-level stage the program splits into
//     sub-stages; the first of them in program order (Design → Concept
//     design);
//  4. nearest_earlier: the last program stage that comes no later in the
//     catalogue (sub-stages sit right after their parent), so work is never
//     placed in a stage that the catalogue puts after it; `other` sits at
//     the end of the catalogue and lands in the last program stage;
//  5. first_stage: nothing comes earlier; the first program stage.
func (g *gen) resolveStage(catID string) (string, string) {
	if key, ok := g.stageByCat[catID]; ok {
		return key, PlaceExact
	}
	if parent := g.catParent[catID]; parent != "" {
		if key, ok := g.stageByCat[parent]; ok {
			return key, PlaceParent
		}
	}
	for _, key := range g.stageOrder {
		if g.catParent[g.plan.Rows[g.byKey[key]].CatalogueStageID] == catID {
			return key, PlaceSubstage
		}
	}
	pos, known := g.catPos[catID]
	if !known {
		pos = len(g.catPos)
	}
	best, bestPos := "", -1
	for _, key := range g.stageOrder {
		p := g.catPos[g.plan.Rows[g.byKey[key]].CatalogueStageID]
		if p <= pos && p > bestPos {
			best, bestPos = key, p
		}
	}
	if best != "" {
		return best, PlaceNearest
	}
	if len(g.stageOrder) == 0 {
		return "", PlaceFirst
	}
	return g.stageOrder[0], PlaceFirst
}

// Step 3: one activity per package stage; a package with no stages gets one
// activity under Construction when the program has it, else Procurement,
// else wherever Construction resolves to.
func (g *gen) packageRows() {
	for _, p := range g.in.Packages {
		if len(p.Stages) == 0 {
			key := g.stageByCat[stageConstruction]
			if key == "" {
				key = g.stageByCat[stageProcurement]
			}
			if key == "" {
				key, _ = g.resolveStage(stageConstruction)
			}
			g.addRow(Row{SourceKey: "package:" + p.ID + ":", ParentKey: key, Kind: KindActivity, Label: p.Title, PackageID: p.ID})
			continue
		}
		for _, s := range p.Stages {
			key, rule := g.resolveStage(s.CatalogueStageID)
			label := s.Label
			if strings.TrimSpace(label) == "" {
				st, _ := g.cat.DeliveryStage(s.CatalogueStageID)
				label = st.Label
			}
			g.addRow(Row{SourceKey: "package:" + p.ID + ":" + s.ID, ParentKey: key, Kind: KindActivity,
				Label: p.Title + ": " + label, PackageID: p.ID, PackageStageID: s.ID})
			g.plan.StageMap = append(g.plan.StageMap, StageMapping{PackageID: p.ID, PackageStageID: s.ID, StageKey: key, Rule: rule})
		}
	}
}

// Revision 2 of 4.2: an investigate work item brings one activity under the
// Investigation stage, labelled with the item's title as the user wrote it.
// Its source key is investigate:<work item id> (no fragment is involved).
func (g *gen) investigationRows() {
	for i := range g.in.WorkItems {
		it := g.in.WorkItems[i]
		if it.Action != actionInvestigate || strings.TrimSpace(it.Title) == "" {
			continue
		}
		key, _ := g.resolveStage(stageInvestigation)
		g.addRow(Row{SourceKey: "investigate:" + it.ID, ParentKey: key, Kind: KindActivity, Label: it.Title,
			WorkItemID: it.ID, Systems: []string{it.System}, item: &g.in.WorkItems[i]})
	}
}

// Steps 4–5: fragments follow included work items. A fragment is considered
// once per work item on one of its systems (or below one), and its
// applies_when is evaluated with that one item as the works in scope, so a
// work_action term reads that item's action. retain items bring nothing;
// investigate items are handled above; a remove item selects only demolition
// fragments (those whose predicate names `remove` in a work_action). True
// adds the fragment; unknown adds it conditionally (low 0, label "If …");
// false skips it. A fragment with no matching work item is not added, even
// when its predicate holds: fragments follow work items (revision 2).
func (g *gen) fragmentRows() {
	for _, f := range g.cat.Fragments() {
		demolition := namesAction(f.AppliesWhen, actionRemove)
		for i := range g.in.WorkItems {
			it := g.in.WorkItems[i]
			if it.Action == actionRetain || it.Action == actionInvestigate || !g.onSystems(it.System, f.Systems) {
				continue
			}
			if it.Action == actionRemove && !demolition {
				continue
			}
			env := g.env([]WorkItem{it})
			truth := knowledge.True
			if f.AppliesWhen != nil {
				truth = g.cat.Holds(f.AppliesWhen, env)
			}
			if truth == knowledge.False {
				continue
			}
			prefix := ""
			if truth == knowledge.Unknown {
				prefix = "If " + g.unknownLabel(f.AppliesWhen, env) + ": "
			}
			stageKey, _ := g.resolveStage(f.Stage)
			inst := fragInstance{frag: f, item: it, keys: map[string]string{}, stageKey: stageKey}
			for _, a := range f.Activities {
				label := a.Label
				if len(f.Activities) == 1 && strings.TrimSpace(it.Title) != "" {
					label = it.Title
				}
				key := "fragment:" + f.ID + ":" + it.ID + ":" + a.Key
				r := Row{SourceKey: key, ParentKey: stageKey, Kind: KindActivity, Label: prefix + label,
					WorkItemID: it.ID, FragmentID: f.ID, Systems: append([]string(nil), a.Systems...),
					conditional: truth == knowledge.Unknown, item: &g.in.WorkItems[i]}
				if a.Duration != nil {
					r.durationID = *a.Duration
				}
				g.addRow(r)
				inst.keys[a.Key] = key
			}
			internalPred, internalSucc := map[string]bool{}, map[string]bool{}
			for _, l := range f.Links {
				internalSucc[l.From], internalPred[l.To] = true, true
			}
			for _, a := range f.Activities {
				if !internalPred[a.Key] {
					inst.sources = append(inst.sources, inst.keys[a.Key])
				}
				if !internalSucc[a.Key] {
					inst.sinks = append(inst.sinks, inst.keys[a.Key])
				}
			}
			g.fragInstances = append(g.fragInstances, inst)
		}
	}
}

func (g *gen) onSystems(system string, systems []string) bool {
	for _, s := range systems {
		if system != "" && g.cat.SystemWithin(system, s) {
			return true
		}
	}
	return false
}

// namesAction reports whether a predicate has a work_action listing action.
func namesAction(p any, action string) bool {
	found := false
	walkPred(p, func(m map[string]any) {
		wa, ok := m["work_action"].(map[string]any)
		if !ok {
			return
		}
		for _, a := range anyList(wa["any_of"]) {
			if fmt.Sprint(a) == action {
				found = true
			}
		}
	})
	return found
}

// walkPred visits every mapping in a predicate, depth first in key order.
func walkPred(p any, visit func(map[string]any)) {
	switch v := p.(type) {
	case map[string]any:
		visit(v)
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walkPred(v[k], visit)
		}
	case []any:
		for _, x := range v {
			walkPred(x, visit)
		}
	}
}

func anyList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	return nil
}

// leaves lists a predicate's leaf terms in a stable order: each mapping that
// is a det or cond comparison, or a single system, works or work_action test.
func leaves(p any) []map[string]any {
	var out []map[string]any
	var walk func(any)
	walk = func(v any) {
		switch x := v.(type) {
		case map[string]any:
			if _, ok := x["det"]; ok {
				out = append(out, x)
				return
			}
			if _, ok := x["cond"]; ok {
				out = append(out, x)
				return
			}
			keys := make([]string, 0, len(x))
			for k := range x {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				switch k {
				case "all", "any", "not":
					walk(x[k])
				default:
					out = append(out, map[string]any{k: x[k]})
				}
			}
		case []any:
			for _, y := range x {
				walk(y)
			}
		}
	}
	walk(p)
	return out
}

// unknownLabel names the first unknown term of a predicate with the
// library's own label for it (a determinant, a condition or a system).
func (g *gen) unknownLabel(p any, env knowledge.WorksEnv) string {
	for _, leaf := range leaves(p) {
		if g.cat.Holds(leaf, env) != knowledge.Unknown {
			continue
		}
		if l := g.termLabel(leaf); l != "" {
			return l
		}
	}
	return "this applies"
}

// termLabel is the library label of the thing a leaf term reads.
func (g *gen) termLabel(leaf map[string]any) string {
	if det, ok := leaf["det"]; ok {
		if d, ok := g.cat.Determinant(fmt.Sprint(det)); ok {
			return d.Label
		}
	}
	if key, ok := leaf["cond"]; ok {
		if c, ok := g.condition(fmt.Sprint(key)); ok {
			return c.Label
		}
	}
	for _, k := range []string{"system_present", "system_existing"} {
		if v, ok := leaf[k]; ok {
			if s, ok := g.cat.System(fmt.Sprint(v)); ok {
				return s.Label
			}
		}
	}
	if wa, ok := leaf["work_action"].(map[string]any); ok {
		if s, ok := g.cat.System(fmt.Sprint(wa["system"])); ok {
			return s.Label
		}
	}
	return ""
}

func (g *gen) condition(key string) (knowledge.Condition, bool) {
	for _, c := range g.cat.Taxonomy().Conditions {
		if c.Key == key {
			return c, true
		}
	}
	return knowledge.Condition{}, false
}

// Step 6: a hold point for each `sequences` interface with a hold_point
// whose before and after are system ids tagged on program activities. It
// goes in the stage of the last `after` activity in program order. An
// interface whose applies_when is false is skipped; unknown keeps it, since
// leaving out an inspection is the unsafe direction. Today the library's
// before/after are free text, so none resolve and none are inserted.
func (g *gen) holdPointRows() {
	for _, iface := range g.cat.Interfaces {
		hp := iface.HoldPoint
		if iface.Type != "sequences" || hp == nil {
			continue
		}
		if _, ok := g.cat.System(hp.After); !ok {
			continue
		}
		if _, ok := g.cat.System(hp.Before); !ok {
			continue
		}
		if iface.AppliesWhen != nil && g.cat.Holds(iface.AppliesWhen, g.env(nil)) == knowledge.False {
			continue
		}
		after, before := g.tagged(hp.After), g.tagged(hp.Before)
		if len(after) == 0 || len(before) == 0 || strings.TrimSpace(hp.Evidence) == "" {
			continue
		}
		last := after[len(after)-1]
		key := "hold:" + iface.ID
		zero := 0
		g.addRow(Row{SourceKey: key, ParentKey: g.plan.Rows[g.byKey[last]].ParentKey, Kind: KindHoldPoint,
			Label: "Hold point: " + hp.Evidence, InterfaceID: iface.ID, Systems: []string{hp.After, hp.Before}, Duration: &zero})
		g.holds = append(g.holds, holdRow{key: key, iface: iface.ID, after: after, before: before})
	}
}

// tagged lists non-stage rows tagged with the system or a descendant, in
// program order (stage order, then ordinal).
func (g *gen) tagged(system string) []string {
	var out []string
	for _, stage := range g.stageOrder {
		for _, r := range g.plan.Rows {
			if r.ParentKey != stage || r.Kind == KindHoldPoint {
				continue
			}
			for _, s := range r.Systems {
				if g.cat.SystemWithin(s, system) {
					out = append(out, r.SourceKey)
					break
				}
			}
		}
	}
	return out
}

// Step 2 links: chain links between neighbouring stages, then the
// skeleton's own links. A skeleton link on a neighbouring pair that is FS
// with no lag is that chain link (origin chain, so a reorder may rewrite
// it); any other skeleton link on a neighbouring pair replaces the chain
// link with the library's type and lag.
func (g *gen) startGuard() {
	g.guard = newLoopGuard(g.plan.Rows, g.kids)
}

func (g *gen) stageLinks(sk knowledge.Skeleton) {
	g.startGuard()
	neighbour := map[[2]string]bool{}
	for i := 1; i < len(g.stageOrder); i++ {
		neighbour[[2]string{g.stageOrder[i-1], g.stageOrder[i]}] = true
	}
	replaced := map[[2]string]knowledge.ProgramLink{}
	var extra []knowledge.ProgramLink
	for _, l := range sk.Links {
		pair := [2]string{g.stageByCat[l.From], g.stageByCat[l.To]}
		switch {
		case neighbour[pair] && l.Type == LinkFS && l.Lag == nil:
		case neighbour[pair]:
			replaced[pair] = l
		default:
			extra = append(extra, l)
		}
	}
	for i := 1; i < len(g.stageOrder); i++ {
		pair := [2]string{g.stageOrder[i-1], g.stageOrder[i]}
		if l, ok := replaced[pair]; ok {
			g.addLibraryLink(pair[0], pair[1], l, LinkOriginLibrary, "")
			continue
		}
		g.addLink(PlanLink{PredKey: pair[0], SuccKey: pair[1], Type: LinkFS, Origin: LinkOriginChain})
	}
	for _, l := range extra {
		g.addLibraryLink(g.stageByCat[l.From], g.stageByCat[l.To], l, LinkOriginLibrary, "")
	}
}

// Fragment internal links, then attach links. attach after/before
// stage_start or stage_finish adds no link (containment already holds an
// activity inside its stage, and a row cannot link to its own stage), but an
// `after: stage_start` activity counts as placed, so step 7 does not chain
// it behind the previous activity. attach to another fragment links every
// sink of that fragment's instances to every source of this one (or the
// reverse for before), FS with no lag.
func (g *gen) fragmentLinks() {
	for _, inst := range g.fragInstances {
		for _, l := range inst.frag.Links {
			g.addLibraryLink(inst.keys[l.From], inst.keys[l.To], l, LinkOriginLibrary, "")
		}
	}
	byFrag := map[string][]fragInstance{}
	for _, inst := range g.fragInstances {
		byFrag[inst.frag.ID] = append(byFrag[inst.frag.ID], inst)
	}
	for _, inst := range g.fragInstances {
		a := inst.frag.Attach
		if a == nil {
			continue
		}
		if a.After == "stage_start" || a.After == "stage_finish" {
			for _, k := range inst.sources {
				g.anchored[k] = true
			}
		}
		if strings.HasPrefix(a.After, "fr.") {
			for _, other := range byFrag[a.After] {
				for _, from := range other.sinks {
					for _, to := range inst.sources {
						g.addGuarded(PlanLink{PredKey: from, SuccKey: to, Type: LinkFS, Origin: LinkOriginLibrary})
					}
				}
			}
		}
		if strings.HasPrefix(a.Before, "fr.") {
			for _, other := range byFrag[a.Before] {
				for _, from := range inst.sinks {
					for _, to := range other.sources {
						g.addGuarded(PlanLink{PredKey: from, SuccKey: to, Type: LinkFS, Origin: LinkOriginLibrary})
					}
				}
			}
		}
	}
}

// Hold point links: every `after` activity → hold point → every `before`
// activity, FS with no lag, origin interface.
func (g *gen) holdPointLinks() {
	for _, h := range g.holds {
		for _, a := range h.after {
			g.addGuarded(PlanLink{PredKey: a, SuccKey: h.key, Type: LinkFS, Origin: LinkOriginInterface, InterfaceID: h.iface})
		}
		for _, b := range h.before {
			g.addGuarded(PlanLink{PredKey: h.key, SuccKey: b, Type: LinkFS, Origin: LinkOriginInterface, InterfaceID: h.iface})
		}
	}
}

// Step 7: inside each stage, an activity with no predecessor from steps 3–6
// (and not anchored at its stage start) is chained FS behind the activity
// before it in ordinal order.
func (g *gen) chainLinks() {
	for _, stage := range g.stageOrder {
		var prev string
		for _, r := range g.plan.Rows {
			if r.ParentKey != stage {
				continue
			}
			if prev != "" && !g.hasPred[r.SourceKey] && !g.anchored[r.SourceKey] {
				g.addGuarded(PlanLink{PredKey: prev, SuccKey: r.SourceKey, Type: LinkFS, Origin: LinkOriginChain})
			}
			prev = r.SourceKey
		}
	}
}

// addLibraryLink resolves a library lag. An unresolved lag id (no reviewed
// row, or a driver that is not known) is never turned into a number: the
// link is kept with no lag, flagged LagUnknown, and a lag_unknown check
// names it. Dropping the link would lose the sequence, which is the more
// misleading error; the check keeps the optimistic gap visible.
func (g *gen) addLibraryLink(pred, succ string, l knowledge.ProgramLink, origin, iface string) {
	pl := PlanLink{PredKey: pred, SuccKey: succ, Type: l.Type, Origin: origin, InterfaceID: iface}
	if l.Lag != nil {
		pl.LagDurationID = *l.Lag
		days, unit, ok := g.lagDays(*l.Lag)
		if ok {
			pl.Lag, pl.LagUnit = days, unit
		} else {
			pl.LagUnknown = true
		}
	}
	if g.addGuarded(pl) && pl.LagUnknown {
		g.note(CheckLagUnknown, succ, fmt.Sprintf("The lag between %s and %s is not known yet; shown as no lag.",
			g.label(pred), g.label(succ)))
	}
}

func (g *gen) label(key string) string { return g.plan.Rows[g.byKey[key]].Label }

// addGuarded places a link unless it would close a loop, ends FF or SF on a
// stage with activities, or joins a row and its own stage; a skipped
// library link becomes a check.
func (g *gen) addGuarded(l PlanLink) bool {
	if l.PredKey == "" || l.SuccKey == "" || l.PredKey == l.SuccKey {
		return false
	}
	if _, dup := g.pairs[[2]string{l.PredKey, l.SuccKey}]; dup {
		return false
	}
	p, s := g.plan.Rows[g.byKey[l.PredKey]], g.plan.Rows[g.byKey[l.SuccKey]]
	if p.ParentKey == l.SuccKey || s.ParentKey == l.PredKey {
		return false
	}
	reason := ""
	switch {
	case g.kids[l.SuccKey] > 0 && (l.Type == LinkFF || l.Type == LinkSF):
		reason = MsgLinkIntoStage
	case g.guard.closes(g.byKey[l.PredKey], g.byKey[l.SuccKey], l.Type):
		reason = "it would make a loop"
	}
	if reason != "" {
		if l.Origin != LinkOriginChain {
			g.note(CheckLinkSkipped, l.SuccKey, fmt.Sprintf("Link from %s to %s left out: %s.", p.Label, s.Label, reason))
		}
		return false
	}
	g.addLink(l)
	return true
}

func (g *gen) addLink(l PlanLink) {
	if l.LagUnit == "" {
		l.LagUnit = LagWorking
	}
	l.SourceKey = LinkKey(l.PredKey, l.SuccKey)
	g.pairs[[2]string{l.PredKey, l.SuccKey}] = len(g.plan.Links)
	g.plan.Links = append(g.plan.Links, l)
	g.hasPred[l.SuccKey] = true
	g.guard.add(g.byKey[l.PredKey], g.byKey[l.SuccKey], l.Type)
}

// LinkKey is a generated link's source key. The pair is unique per program
// (program_links_pair_uq), so it identifies the link across regenerations.
func LinkKey(pred, succ string) string { return "link:" + pred + "|" + succ }

// lagDays turns a library lag id into whole days and its unit. Only a
// reviewed, applicable row with a known typical value resolves.
func (g *gen) lagDays(id string) (int, string, bool) {
	d, ok := g.cat.ReviewedDuration(id)
	if !ok || !g.cat.DurationApplies(d, g.env(nil)) {
		return 0, "", false
	}
	r, ok := g.compute(d)
	if !ok || !r.TypicalKnown || r.Typical > 365 {
		return 0, "", false
	}
	unit := LagWorking
	if d.UnitDays == "calendar" {
		unit = LagCalendar
	}
	return int(r.Typical), unit, true
}

// compute feeds a reviewed row its driver: a planning value or determinant
// for per_unit and floor_cycle, the indexed budget for time_cost.
func (g *gen) compute(d knowledge.Duration) (knowledge.DayRange, bool) {
	var in knowledge.DurationInput
	switch d.Basis {
	case "per_unit", "floor_cycle":
		if d.Driver == nil {
			return knowledge.DayRange{}, false
		}
		raw := g.in.Planning[d.Driver.PlanningKey]
		if d.Driver.Determinant != "" {
			raw = g.in.Values[d.Driver.Determinant]
		}
		v, ok := new(big.Rat).SetString(strings.TrimSpace(raw))
		if !ok || v.Sign() < 0 {
			return knowledge.DayRange{}, false
		}
		in = knowledge.ExactInput(v)
	case "time_cost":
		b := g.in.Budget
		if b == nil || b.Low == nil || b.High == nil || g.in.IndexCost == nil {
			return knowledge.DayRange{}, false
		}
		lo, ok1 := g.in.IndexCost(b.Low, b.PriceDate, d.CostBaseDate)
		hi, ok2 := g.in.IndexCost(b.High, b.PriceDate, d.CostBaseDate)
		if !ok1 || !ok2 {
			return knowledge.DayRange{}, false
		}
		in = knowledge.DurationInput{Low: lo, High: hi}
		if lo.Cmp(hi) == 0 {
			in.Typical = lo
		}
	}
	return knowledge.ComputeDuration(d, in)
}

// Step 8: durations, first match wins: the user's value; an accepted
// document reading (by source key, or for a stage by catalogue id); a
// Profile planning value (construction_duration on the Construction stage
// only while none of its activities has a duration); a reviewed library row
// whose predicate holds and whose driver is known; otherwise unknown.
// Activities are settled before stages so the Construction rule can see
// its children.
func (g *gen) durations() {
	for i := range g.plan.Rows {
		if g.plan.Rows[i].Kind != KindStage {
			g.duration(i)
		}
	}
	for i := range g.plan.Rows {
		if g.plan.Rows[i].Kind == KindStage {
			g.duration(i)
		}
	}
}

func (g *gen) duration(i int) {
	r := &g.plan.Rows[i]
	if r.Kind == KindHoldPoint {
		// A hold point is a zero-length milestone by definition.
		r.DurationOrigin, r.DurationSource = DurationCalculation, map[string]any{"hold_point": r.InterfaceID}
		return
	}
	set := func(days int, origin string, source map[string]any) {
		d := days
		r.Duration, r.Low, r.High, r.DurationOrigin, r.DurationSource = &d, nil, nil, origin, source
	}
	if d, ok := g.in.UserDurations[r.SourceKey]; ok && validDays(d) {
		set(d, DurationUser, map[string]any{"user": true})
		return
	}
	if rd, ok := g.in.DocumentDurations[r.SourceKey]; ok && validDays(rd.Days) {
		set(rd.Days, DurationDocument, map[string]any{"reading": rd.ReadingID})
		return
	}
	if r.Kind == KindStage {
		if rd, ok := g.in.StageReadings[r.CatalogueStageID]; ok && validDays(rd.Days) {
			set(rd.Days, DurationDocument, map[string]any{"reading": rd.ReadingID, "catalogue_stage_id": r.CatalogueStageID})
			return
		}
		if r.CatalogueStageID == stageConstruction && !g.childHasDuration(r.SourceKey) {
			if days, raw, ok := g.constructionDays(); ok {
				set(days, DurationProfile, map[string]any{"planning_key": planConstructionDuration, "value": raw})
			}
		}
		return
	}
	if r.durationID == "" {
		return
	}
	d, ok := g.cat.ReviewedDuration(r.durationID)
	env := g.env(nil)
	if r.item != nil {
		env = g.env([]WorkItem{*r.item})
	}
	if !ok || !g.cat.DurationApplies(d, env) || d.UnitDays != "working" {
		// A calendar-day activity duration needs dates to become working
		// days; with no start date that conversion would be a guess.
		return
	}
	dr, ok := g.compute(d)
	if !ok || !validDays(int(dr.High)) {
		return
	}
	low, high := int(dr.Low), int(dr.High)
	source := map[string]any{"library": d.ID, "version": d.Version}
	switch {
	case r.conditional:
		// Step 5: an unknown predicate lengthens only the high run.
		zero, zl := 0, 0
		r.Duration, r.Low, r.High = &zero, &zl, &high
		source["conditional"] = true
	case dr.TypicalKnown:
		typ := int(dr.Typical)
		r.Duration, r.Low, r.High = &typ, &low, &high
	default:
		// A range driver with no typical value: the ends are kept, the
		// typical run stays unknown rather than take an invented middle.
		r.Low, r.High = &low, &high
	}
	r.DurationOrigin, r.DurationSource = DurationLibrary, source
}

func validDays(d int) bool { return d >= 0 && d <= MaxDuration }

func (g *gen) childHasDuration(stage string) bool {
	for _, r := range g.plan.Rows {
		if r.ParentKey == stage && r.Kind != KindHoldPoint && r.Duration != nil {
			return true
		}
	}
	return false
}

// constructionDays turns construction_duration (weeks) into working days:
// weeks × working days per week of the program's working week, rounded up.
func (g *gen) constructionDays() (int, string, bool) {
	raw := strings.TrimSpace(g.in.Planning[planConstructionDuration])
	w, ok := new(big.Rat).SetString(raw)
	if !ok || w.Sign() < 0 {
		return 0, raw, false
	}
	week := g.in.WorkingWeek
	if week <= 0 || week > 127 {
		week = DefaultWorkingWeek
	}
	perWeek := 0
	for b := week; b > 0; b >>= 1 {
		perWeek += b & 1
	}
	days := new(big.Rat).Mul(w, big.NewRat(int64(perWeek), 1))
	q, m := new(big.Int).QuoRem(days.Num(), days.Denom(), new(big.Int))
	if m.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() || !validDays(int(q.Int64())) {
		return 0, raw, false
	}
	return int(q.Int64()), raw, true
}

// reason explains the skeleton choice from the terms that held, using the
// library's labels and the Profile's own option labels.
func (g *gen) reason(sk knowledge.Skeleton) string {
	if sk.ID == knowledge.GenericSkeletonID {
		return sk.Label + ": no other program template matches the Profile yet."
	}
	env := g.env(nil)
	var terms []string
	var walk func(p any)
	walk = func(p any) {
		m, ok := p.(map[string]any)
		if !ok {
			return
		}
		if _, ok := m["det"]; ok {
			if g.cat.Holds(m, env) == knowledge.True {
				terms = append(terms, g.termText(m))
			}
			return
		}
		if _, ok := m["cond"]; ok {
			if g.cat.Holds(m, env) == knowledge.True {
				terms = append(terms, g.termText(m))
			}
			return
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			switch k {
			case "all":
				for _, sub := range anyList(m[k]) {
					walk(sub)
				}
			case "any":
				for _, sub := range anyList(m[k]) {
					if g.cat.Holds(sub, env) == knowledge.True {
						walk(sub)
					}
				}
			case "not":
				// A negated term held because something is absent; the
				// header names what is present, so it is left out.
			default:
				leaf := map[string]any{k: m[k]}
				if g.cat.Holds(leaf, env) == knowledge.True {
					if l := g.termLabel(leaf); l != "" {
						terms = append(terms, l)
					}
				}
			}
		}
	}
	walk(sk.AppliesWhen)
	if len(terms) == 0 {
		return sk.Label
	}
	return sk.Label + ": " + strings.Join(terms, "; ")
}

// termText is "<label>: <value label>" for a det or cond term that held.
func (g *gen) termText(leaf map[string]any) string {
	if key, ok := leaf["cond"]; ok {
		c, _ := g.condition(fmt.Sprint(key))
		return c.Label + ": " + choiceLabel(c.AllOptions(), g.in.Conditions[c.Key])
	}
	id := fmt.Sprint(leaf["det"])
	d, _ := g.cat.Determinant(id)
	if id == "work_type" {
		var labels []string
		for _, w := range g.in.WorkTypes {
			labels = append(labels, choiceLabel(g.cat.Taxonomy().WorkTypes, w))
		}
		return d.Label + ": " + strings.Join(labels, ", ")
	}
	v := g.in.Values[id]
	if d.Value == "boolean" {
		switch v {
		case "true", "stated_true":
			v = "Yes"
		case "false", "stated_false":
			v = "No"
		}
	}
	return d.Label + ": " + v
}

func choiceLabel(options []knowledge.Choice, id string) string {
	for _, o := range options {
		if o.ID == id {
			if o.DisplayLabel != "" {
				return o.DisplayLabel
			}
			return o.Label
		}
	}
	return id
}

// loopGuard answers "would this link close a loop?" over the same graph the
// engine builds (engine.go header): a stage with activities has a start node
// (its own) and a finish node, with containment edges start → child → finish;
// links into it land on its start, links out leave from its finish (FS, FF)
// or start (SS, SF).
type loopGuard struct {
	n      int
	finish []int // finish node per row, −1 for a leaf
	out    [][]int
	seen   []int
	stamp  int
}

func newLoopGuard(rows []Row, kids map[string]int) *loopGuard {
	g := &loopGuard{n: len(rows), finish: make([]int, len(rows))}
	idx := map[string]int{}
	for i, r := range rows {
		idx[r.SourceKey] = i
		g.finish[i] = -1
	}
	total := len(rows)
	for i, r := range rows {
		if kids[r.SourceKey] > 0 {
			g.finish[i] = total
			total++
		}
	}
	g.out = make([][]int, total)
	g.seen = make([]int, total)
	for i, r := range rows {
		if r.ParentKey == "" {
			continue
		}
		p := idx[r.ParentKey]
		g.out[p] = append(g.out[p], i)
		g.out[i] = append(g.out[i], g.finish[p])
	}
	return g
}

func (g *loopGuard) anchors(pred, succ int, typ string) (int, int) {
	from := pred
	if f := g.finish[pred]; f >= 0 && (typ == LinkFS || typ == LinkFF || typ == "") {
		from = f
	}
	return from, succ
}

func (g *loopGuard) add(pred, succ int, typ string) {
	from, to := g.anchors(pred, succ, typ)
	g.out[from] = append(g.out[from], to)
}

// closes reports whether to already reaches from, iteratively (no recursion:
// chains can be long).
func (g *loopGuard) closes(pred, succ int, typ string) bool {
	from, to := g.anchors(pred, succ, typ)
	g.stamp++
	stack := []int{to}
	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if v == from {
			return true
		}
		if g.seen[v] == g.stamp {
			continue
		}
		g.seen[v] = g.stamp
		stack = append(stack, g.out[v]...)
	}
	return false
}
