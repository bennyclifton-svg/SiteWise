package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

// Profile routes read precomputed rows and write user values. Neither waits
// on Jev: a write re-runs the code-only reconciliation under the project lock.

const (
	maxProfileText = 200
	maxProfileNote = 120
)

type optionJSON struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type cellJSON struct {
	Value        string                `json:"value"`
	Band         string                `json:"band"`
	Assertion    string                `json:"assertion,omitempty"`
	Tenders      string                `json:"tenders,omitempty"`
	Note         string                `json:"note,omitempty"`
	Sources      []profile.Source      `json:"sources"`
	Alternatives []profile.Alternative `json:"alternatives"`
	Derived      *profile.Derived      `json:"derived,omitempty"`
}

type fieldJSON struct {
	Key      string       `json:"key"`
	Label    string       `json:"label"`
	PartID   string       `json:"part_id"`
	Kind     string       `json:"kind"` // choice | multi | number | text
	Unit     string       `json:"unit,omitempty"`
	Options  []optionJSON `json:"options,omitempty"`
	StatedIn []string     `json:"stated_in,omitempty"`
	cellJSON
}

type systemRowJSON struct {
	Leaf     string   `json:"leaf"`
	Label    string   `json:"label"`
	PartID   string   `json:"part_id"`
	Presence cellJSON `json:"presence"`
	Provider cellJSON `json:"provider"`
	Note     cellJSON `json:"note"`
	Shown    bool     `json:"shown_by_default"`
}

type systemGroupJSON struct {
	ID    string          `json:"id"`
	Label string          `json:"label"`
	Rows  []systemRowJSON `json:"rows"`
}

type complianceGroupJSON struct {
	Group string      `json:"group"`
	Rows  []fieldJSON `json:"rows"`
}

type partJSON struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	NCCClass string `json:"ncc_class"`
}

type profileJSON struct {
	Coverage         []store.SourceCoverage `json:"coverage"`
	ProjectID        string                 `json:"project_id"`
	BuiltAt          *time.Time             `json:"built_at"`
	PendingDocuments int                    `json:"pending_documents"`
	ActiveDocuments  int                    `json:"active_documents"`
	UnreadDocuments  int                    `json:"unread_documents"`
	ReadDocuments    int                    `json:"read_documents"`
	SkippedDocuments int                    `json:"skipped_documents"`
	SkippedKind      string                 `json:"skipped_kind"`
	FailedDocuments  int                    `json:"failed_documents"`
	PaymentRequired  bool                   `json:"payment_required"`
	// Queued answers a profile read request: documents it sent to Jev.
	Queued     *int64                `json:"queued,omitempty"`
	Thresholds map[string]any        `json:"thresholds"`
	Parts      []partJSON            `json:"parts"`
	Header     []fieldJSON           `json:"header"`
	Facts      []fieldJSON           `json:"facts"`
	Systems    []systemGroupJSON     `json:"systems"`
	Compliance []complianceGroupJSON `json:"compliance"`
}

func getProfile(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	writeProfile(w, r, deps, session.OrgID, r.PathValue("id"))
}

// requestProfileRead is the "Update project profile" button: it queues Jev
// reading for the project's unread documents and returns the profile, which
// then fills in as each document's profile event arrives.
func requestProfileRead(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID := r.PathValue("id")
	if !uuidPattern.MatchString(projectID) || deps.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	// The read below 404s an unknown or foreign project; the queue is
	// org-scoped, so a foreign id queues nothing first.
	queued, err := deps.Store.RequestProfileRead(r.Context(), session.OrgID, projectID, deps.ProfileReading.Kinds())
	if err != nil {
		http.Error(w, "request failed", http.StatusInternalServerError)
		return
	}
	writeProfileQueued(w, r, deps, session.OrgID, projectID, &queued)
}

func writeProfile(w http.ResponseWriter, r *http.Request, deps Deps, orgID, projectID string) {
	writeProfileQueued(w, r, deps, orgID, projectID, nil)
}

func writeProfileQueued(w http.ResponseWriter, r *http.Request, deps Deps, orgID, projectID string, queued *int64) {
	if !uuidPattern.MatchString(projectID) || deps.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := deps.Store.EnsureWholePart(r.Context(), orgID, projectID); errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	view, err := deps.Store.ReadProfile(r.Context(), orgID, projectID, deps.ProfileReading.Kinds())
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	body := profileBody(projectID, view, deps)
	body.Queued = queued
	writeJSON(w, http.StatusOK, body)
}

func profileBody(projectID string, view store.ProfileView, deps Deps) profileJSON {
	cat := deps.Knowledge
	whole := ""
	out := profileJSON{Coverage: view.Coverage, ProjectID: projectID, BuiltAt: view.BuiltAt, PendingDocuments: view.PendingDocuments, UnreadDocuments: view.UnreadDocuments,
		ActiveDocuments: view.ActiveDocuments, FailedDocuments: view.FailedDocuments, PaymentRequired: view.PaymentRequired,
		ReadDocuments: view.ReadDocuments, SkippedDocuments: view.SkippedDocuments, SkippedKind: view.SkippedKind,
		Thresholds: map[string]any{"version": deps.ProfileThresholds.Version, "provisional": !deps.ProfileThresholds.Approved,
			"applied": len(deps.ProfileThresholds.Amber) > 0}}
	for _, p := range view.Parts {
		out.Parts = append(out.Parts, partJSON{ID: p.ID, Label: p.Label, Kind: p.Kind, NCCClass: p.NCCClass})
		if p.Kind == "whole" && whole == "" {
			whole = p.ID
		}
	}
	rows := map[string]profile.Row{}
	for _, r := range view.Rows {
		rows[r.PartID+"|"+r.Key] = r
	}
	cell := func(part, key string) cellJSON {
		r, ok := rows[part+"|"+key]
		if !ok {
			return cellJSON{Sources: []profile.Source{}, Alternatives: []profile.Alternative{}}
		}
		c := cellJSON{Value: r.Value, Band: r.Band, Assertion: r.Assertion, Tenders: r.Tenders, Note: r.Note,
			Sources: r.Sources, Alternatives: r.Alternatives, Derived: r.Derived}
		if c.Sources == nil {
			c.Sources = []profile.Source{}
		}
		if c.Alternatives == nil {
			c.Alternatives = []profile.Alternative{}
		}
		return c
	}

	tax := cat.Taxonomy()
	var classes, subclasses, works []optionJSON
	for _, c := range tax.BuildingClasses {
		classes = append(classes, optionJSON{c.ID, c.Label})
	}
	subclass := cell(whole, "hdr.subclass").Value
	class := cell(whole, "hdr.building_class").Value
	for _, c := range tax.BuildingClasses {
		if class != "" && c.ID != class {
			continue
		}
		for _, s := range c.Subclasses {
			subclasses = append(subclasses, optionJSON{s.ID, s.Label})
		}
	}
	for _, wt := range tax.WorkTypes {
		works = append(works, optionJSON{wt.ID, wt.Label})
	}
	field := func(key, label, kind string, opts []optionJSON) fieldJSON {
		return fieldJSON{Key: key, Label: label, PartID: whole, Kind: kind, Options: opts, cellJSON: cell(whole, key)}
	}
	out.Header = append(out.Header, field("hdr.building_class", "Building class", "choice", classes),
		field("hdr.subclass", "Building type", "choice", subclasses), field("hdr.work_type", "Work type", "choice", works))
	var scaleFields []knowledge.ScaleField
	seenScale := map[string]bool{}
	if sub, ok := tax.Subclass(subclass); ok {
		for _, f := range sub.ScaleFields {
			scaleFields = append(scaleFields, f)
			seenScale[f.Key] = true
		}
	}
	// A type's default fields must not hide evidence in another measurement
	// basis (for example a warehouse brief states GLA, while defaults ask GFA).
	for _, f := range cat.ScaleFields() {
		c := cell(whole, "hdr.scale."+f.Key)
		if !seenScale[f.Key] && (c.Value != "" || len(c.Sources) > 0) {
			scaleFields = append(scaleFields, f)
		}
	}
	for _, f := range scaleFields {
		kind := "number"
		if f.Type == "text" {
			kind = "text"
		}
		sf := field("hdr.scale."+f.Key, f.Label, kind, nil)
		sf.Unit = f.Unit
		out.Header = append(out.Header, sf)
	}
	for _, c := range tax.Conditions {
		if len(c.AppliesTo) > 0 && !contains(c.AppliesTo, class) {
			continue
		}
		var opts []optionJSON
		for _, o := range c.Options {
			opts = append(opts, optionJSON{o.ID, o.Label})
		}
		out.Header = append(out.Header, field("hdr.cond."+c.Key, c.Label, "choice", opts))
	}
	for _, f := range cat.ProjectFacts() {
		fj := field("fact."+f.ID, f.Label, valueKind(f), detOptions(f))
		fj.Unit = f.Unit
		out.Facts = append(out.Facts, fj)
	}

	groups := map[string]*systemGroupJSON{}
	var order []string
	for _, top := range cat.TopSystems() {
		groups[top.ID] = &systemGroupJSON{ID: top.ID, Label: top.Label}
		order = append(order, top.ID)
	}
	for _, leaf := range cat.Leaves() {
		g := groups[leaf.Parent]
		if g == nil {
			continue
		}
		base := "sys." + leaf.ID
		row := systemRowJSON{Leaf: leaf.ID, Label: leaf.Label, PartID: whole,
			Presence: cell(whole, base+".presence"), Provider: cell(whole, base+".provider"), Note: cell(whole, base+".note")}
		row.Shown = row.Presence.Band != "" || row.Provider.Band != "" || row.Note.Band != ""
		g.Rows = append(g.Rows, row)
	}
	for _, id := range order {
		out.Systems = append(out.Systems, *groups[id])
	}

	byGroup := map[string][]fieldJSON{}
	for _, d := range cat.ProfileDeterminants() {
		for _, p := range view.Parts {
			key := "det." + d.ID
			c := cell(p.ID, key)
			if p.ID != whole && c.Band == "" {
				continue // other parts show only what applies to them
			}
			fj := fieldJSON{Key: key, Label: d.Label, PartID: p.ID, Kind: valueKind(d), Unit: d.Unit,
				Options: detOptions(d), cellJSON: c}
			for _, s := range d.StatedIn {
				fj.StatedIn = append(fj.StatedIn, s.Label)
			}
			if d.Derived && fj.Derived == nil {
				fj.Derived = &profile.Derived{Rule: d.By, State: knowledge.StateUnknown, Reason: "not_built"}
			}
			byGroup[d.ProfileGroup] = append(byGroup[d.ProfileGroup], fj)
		}
	}
	for _, g := range []string{"classification", "site", "services", "fire"} {
		out.Compliance = append(out.Compliance, complianceGroupJSON{Group: g, Rows: byGroup[g]})
	}
	return out
}

func valueKind(d knowledge.Determinant) string {
	switch {
	case d.Value == "multi_choice":
		return "multi"
	case len(d.Options) > 0 || d.Value == "boolean":
		return "choice"
	case d.Value == "number" || d.Value == "integer":
		return "number"
	}
	return "text"
}

func detOptions(d knowledge.Determinant) []optionJSON {
	if d.Value == "boolean" {
		return []optionJSON{{"stated_true", "Yes"}, {"stated_false", "No"}}
	}
	var out []optionJSON
	for _, o := range d.Options {
		out = append(out, optionJSON{o.ID, o.ID})
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// putProfileValue records the user's word for one key and rebuilds.
// Body: {"part_id": "...", "value": "..." | null, "note": "...", "reset": bool}.
// reset removes the user's value so evidence shows again.
func putProfileValue(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID, key := r.PathValue("id"), r.PathValue("key")
	if !uuidPattern.MatchString(projectID) || deps.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		PartID string  `json:"part_id"`
		Value  *string `json:"value"`
		Note   string  `json:"note"`
		Reset  bool    `json:"reset"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Value != nil {
		v := strings.TrimSpace(*body.Value)
		body.Value = &v
	}
	if msg := validProfileValue(deps.Knowledge, key, body.Value, body.Note); msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	ctx := r.Context()
	part := body.PartID
	if part == "" {
		whole, err := deps.Store.EnsureWholePart(ctx, session.OrgID, projectID)
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "write failed", http.StatusInternalServerError)
			return
		}
		part = whole.ID
	} else if !uuidPattern.MatchString(part) {
		http.Error(w, "unknown part", http.StatusUnprocessableEntity)
		return
	}
	var err error
	if body.Reset {
		err = deps.Store.DeleteUserValue(ctx, session.OrgID, projectID, part, key)
	} else {
		err = deps.Store.SetUserValue(ctx, session.OrgID, projectID, part, session.UserID, key, body.Value, body.Note)
	}
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}
	if err := rebuildProfile(r, deps, session.OrgID, projectID); err != nil {
		http.Error(w, "rebuild failed", http.StatusInternalServerError)
		return
	}
	writeProfile(w, r, deps, session.OrgID, projectID)
}

func rebuildProfile(r *http.Request, deps Deps, orgID, projectID string) error {
	return deps.Store.RebuildProfile(r.Context(), orgID, projectID, deps.ProfileThresholds.Version,
		func(s store.ProfileSnapshot) []profile.Row {
			return profile.Build(profile.Input{Parts: s.Parts, Facts: s.Facts, User: s.User, Thresholds: deps.ProfileThresholds, Read: deps.ProfileReading}, deps.Knowledge)
		})
}

// validProfileValue is the boundary check: the key exists and the value is
// one of its options, a number, or short text. "" means valid.
func validProfileValue(cat *knowledge.Catalog, key string, value *string, note string) string {
	if len([]rune(note)) > maxProfileNote {
		return "note is longer than 120 characters"
	}
	v := ""
	if value != nil {
		v = *value
	}
	if len([]rune(v)) > maxProfileText {
		return "value is longer than 200 characters"
	}
	oneOf := func(opts ...string) string {
		if value == nil || v == "" || contains(opts, v) {
			return ""
		}
		return "value is not an option for " + key
	}
	number := func() string {
		if value == nil || v == "" {
			return ""
		}
		if _, err := strconv.ParseFloat(v, 64); err != nil {
			return "value must be a number"
		}
		return ""
	}
	tax := cat.Taxonomy()
	switch {
	case key == "hdr.building_class":
		var ids []string
		for _, c := range tax.BuildingClasses {
			ids = append(ids, c.ID)
		}
		return oneOf(ids...)
	case key == "hdr.subclass":
		var ids []string
		for _, c := range tax.BuildingClasses {
			for _, s := range c.Subclasses {
				ids = append(ids, s.ID)
			}
		}
		return oneOf(ids...)
	case key == "hdr.work_type":
		var ids []string
		for _, wt := range tax.WorkTypes {
			ids = append(ids, wt.ID)
		}
		return oneOf(ids...)
	case strings.HasPrefix(key, "hdr.cond."):
		for _, c := range tax.Conditions {
			if "hdr.cond."+c.Key == key {
				var ids []string
				for _, o := range c.Options {
					ids = append(ids, o.ID)
				}
				return oneOf(ids...)
			}
		}
	case strings.HasPrefix(key, "hdr.scale."):
		for _, f := range cat.ScaleFields() {
			if "hdr.scale."+f.Key == key {
				if f.Type == "text" {
					return ""
				}
				return number()
			}
		}
	case strings.HasPrefix(key, "fact.") || strings.HasPrefix(key, "det."):
		var d knowledge.Determinant
		var ok bool
		if strings.HasPrefix(key, "fact.") {
			d, ok = cat.ProjectFact(strings.TrimPrefix(key, "fact."))
		} else {
			d, ok = cat.Determinant(strings.TrimPrefix(key, "det."))
			ok = ok && d.ProfileGroup != "" && !d.Deprecated()
		}
		if !ok {
			break
		}
		switch valueKind(d) {
		case "multi":
			if value == nil || v == "" {
				return ""
			}
			var ids []string
			for _, o := range detOptions(d) {
				ids = append(ids, o.ID)
			}
			for _, part := range strings.Split(v, ",") {
				if !contains(ids, strings.TrimSpace(part)) {
					return "value is not an option for " + key
				}
			}
			return ""
		case "choice":
			var ids []string
			for _, o := range detOptions(d) {
				ids = append(ids, o.ID)
			}
			if d.Extraction == "pre_parsed" && len(ids) == 0 {
				return ""
			}
			return oneOf(ids...)
		case "number":
			return number()
		}
		return ""
	case strings.HasPrefix(key, "sys."):
		rest := strings.TrimPrefix(key, "sys.")
		dot := strings.LastIndex(rest, ".")
		if dot < 0 {
			break
		}
		sys, ok := cat.System(rest[:dot])
		if !ok || sys.Parent == "" || sys.Status == "deprecated" {
			break
		}
		switch rest[dot+1:] {
		case "presence":
			return oneOf("included", "not_included")
		case "provider":
			return oneOf("contractor", "owner", "others")
		case "note":
			if len([]rune(v)) > maxProfileNote {
				return "note is longer than 120 characters"
			}
			return ""
		}
	}
	return "unknown profile key " + key
}

type partBody struct {
	Label    *string `json:"label"`
	Kind     *string `json:"kind"`
	NCCClass *string `json:"ncc_class"`
}

var partKinds = []string{"building", "part", "storey", "compartment", "tenancy", "outbuilding"}

func (b partBody) invalid() string {
	if b.Label != nil {
		l := strings.TrimSpace(*b.Label)
		if l == "" || len([]rune(l)) > 80 || l == "Whole project" {
			return "label must be 1 to 80 characters and not 'Whole project'"
		}
		b.Label = &l
	}
	if b.Kind != nil && !contains(partKinds, *b.Kind) {
		return "kind must be one of " + strings.Join(partKinds, ", ")
	}
	if b.NCCClass != nil && len(*b.NCCClass) > 12 {
		return "ncc_class is too long"
	}
	return ""
}

func createPart(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID := r.PathValue("id")
	var body partBody
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil || body.Label == nil || body.Kind == nil {
		http.Error(w, "label and kind required", http.StatusBadRequest)
		return
	}
	if msg := body.invalid(); msg != "" || !uuidPattern.MatchString(projectID) {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	ncc := ""
	if body.NCCClass != nil {
		ncc = *body.NCCClass
	}
	p, err := deps.Store.CreatePart(r.Context(), session.OrgID, projectID, strings.TrimSpace(*body.Label), *body.Kind, ncc)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "a part with that label exists", http.StatusConflict)
		return
	}
	_ = rebuildProfile(r, deps, session.OrgID, projectID)
	writeJSON(w, http.StatusCreated, partJSON{ID: p.ID, Label: p.Label, Kind: p.Kind, NCCClass: p.NCCClass})
}

func updatePart(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID, partID := r.PathValue("id"), r.PathValue("part")
	var body partBody
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if msg := body.invalid(); msg != "" || !uuidPattern.MatchString(projectID) || !uuidPattern.MatchString(partID) {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	if body.Label != nil {
		l := strings.TrimSpace(*body.Label)
		body.Label = &l
	}
	p, err := deps.Store.UpdatePart(r.Context(), session.OrgID, projectID, partID, body.Label, body.Kind, body.NCCClass)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "a part with that label exists", http.StatusConflict)
		return
	}
	_ = rebuildProfile(r, deps, session.OrgID, projectID)
	writeJSON(w, http.StatusOK, partJSON{ID: p.ID, Label: p.Label, Kind: p.Kind, NCCClass: p.NCCClass})
}

const maxReadingIDs = 1000

// setProfileReading records which documents the profile reads, for a
// selection in the register. The rebuild is code only: a document turned
// off leaves the profile at once and its readings stay stored.
func setProfileReading(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	projectID := r.PathValue("id")
	if !uuidPattern.MatchString(projectID) || deps.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body struct {
		DocumentIDs []string `json:"document_ids"`
		Setting     string   `json:"setting"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !profile.ValidReadSetting(body.Setting) || len(body.DocumentIDs) == 0 || len(body.DocumentIDs) > maxReadingIDs {
		http.Error(w, "setting must be auto, read or skip, for 1 to 1000 documents", http.StatusBadRequest)
		return
	}
	for _, id := range body.DocumentIDs {
		if !uuidPattern.MatchString(id) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	}
	_, err := deps.Store.SetProfileReading(r.Context(), session.OrgID, projectID, body.DocumentIDs, body.Setting, deps.ProfileReading.Kinds())
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "update failed", http.StatusInternalServerError)
		return
	}
	if err := rebuildProfile(r, deps, session.OrgID, projectID); err != nil {
		http.Error(w, "rebuild failed", http.StatusInternalServerError)
		return
	}
	writeProfile(w, r, deps, session.OrgID, projectID)
}
