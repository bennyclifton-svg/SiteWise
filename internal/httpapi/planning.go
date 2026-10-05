package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sitewise/internal/knowledge"
	"sitewise/internal/store"
)

// Planning values (plan §4.3, WP-13): assumptions and calculated values with
// a range and limitations, kept apart from document readings. Unknown is a
// value state of its own and is listed under unresolved decisions.

const maxPlanningText = 500

type planningValueJSON struct {
	ID           string    `json:"id"`
	PartID       string    `json:"part_id"`
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	Scope        string    `json:"scope"`
	ValueState   string    `json:"value_state"`
	Value        *string   `json:"value"`
	RangeLow     *float64  `json:"range_low"`
	RangeHigh    *float64  `json:"range_high"`
	Unit         string    `json:"unit"`
	Origin       string    `json:"origin"`
	ReviewStatus string    `json:"review_status"`
	Meaning      string    `json:"meaning"`
	Rationale    string    `json:"rationale"`
	Limitations  string    `json:"limitations"`
	Version      int64     `json:"version"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type planningJSON struct {
	ProjectID string              `json:"project_id"`
	Values    []planningValueJSON `json:"values"`
	// Unresolved are the values recorded as explicitly unknown (L274).
	Unresolved []planningValueJSON `json:"unresolved"`
	// History holds superseded values, when asked for with ?history=1.
	History []planningValueJSON `json:"history,omitempty"`
	// Suggestions are starting values from a reviewed registry only
	// (NW-REQ-185); with none reviewed there are none.
	Suggestions []planningSuggestionJSON `json:"suggestions"`
}

type planningSuggestionJSON struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	Unit  string `json:"unit"`
}

func getPlanning(w http.ResponseWriter, r *http.Request, deps Deps) {
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	writePlanning(w, r, deps, session.OrgID, r.PathValue("id"), r.URL.Query().Get("history") == "1")
}

func writePlanning(w http.ResponseWriter, r *http.Request, deps Deps, orgID, projectID string, history bool) {
	if !uuidPattern.MatchString(projectID) || deps.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if _, err := deps.Store.GetProject(r.Context(), orgID, projectID); errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	values, err := deps.Store.PlanningValues(r.Context(), orgID, projectID, history)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	out := planningJSON{ProjectID: projectID, Values: []planningValueJSON{}, Unresolved: []planningValueJSON{},
		Suggestions: []planningSuggestionJSON{}}
	live := map[string]bool{}
	for _, v := range values {
		label := v.Key
		if pk, ok := deps.Knowledge.PlanningKey(v.Key); ok {
			label = pk.Label
		}
		j := planningValueJSON{ID: v.ID, PartID: v.PartID, Key: v.Key, Label: label, Scope: v.Scope, ValueState: v.State,
			Value: v.Value, RangeLow: v.RangeLow, RangeHigh: v.RangeHigh, Unit: v.Unit, Origin: v.Origin,
			ReviewStatus: v.ReviewStatus, Meaning: v.Meaning, Rationale: v.Rationale, Limitations: v.Limitations,
			Version: v.Version, UpdatedAt: v.UpdatedAt}
		switch {
		case v.ReviewStatus == "superseded":
			out.History = append(out.History, j)
		default:
			live[v.Key] = true
			out.Values = append(out.Values, j)
			if v.State == "unknown" {
				out.Unresolved = append(out.Unresolved, j)
			}
		}
	}
	// Only a key with no value yet is worth suggesting (NW-REQ-186).
	for _, k := range deps.Knowledge.StartingValues() {
		if !live[k.Key] {
			out.Suggestions = append(out.Suggestions, planningSuggestionJSON{Key: k.Key, Label: k.Label, Value: k.StartingValue, Unit: k.Unit})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func putPlanning(w http.ResponseWriter, r *http.Request, deps Deps) {
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
		PartID      string          `json:"part_id"`
		Value       json.RawMessage `json:"value"`
		ValueState  string          `json:"value_state"`
		Unit        string          `json:"unit"`
		RangeLow    *float64        `json:"range_low"`
		RangeHigh   *float64        `json:"range_high"`
		Origin      string          `json:"origin"`
		Meaning     string          `json:"meaning"`
		Rationale   string          `json:"rationale"`
		Limitations string          `json:"limitations"`
		Version     int64           `json:"version"`
	}
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	pk, ok := deps.Knowledge.PlanningKey(r.PathValue("key"))
	if !ok {
		http.Error(w, "not a registered planning key", http.StatusUnprocessableEntity)
		return
	}
	value, msg := planningValue(body.Value)
	if msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	write := store.PlanningWrite{Key: pk.Key, Scope: pk.Scope, Kind: pk.Value, State: body.ValueState, Value: value,
		RangeLow: body.RangeLow, RangeHigh: body.RangeHigh, Unit: pk.Unit, Origin: orDefault(body.Origin, "assumption"),
		Meaning: orDefault(body.Meaning, "stated"), Rationale: strings.TrimSpace(body.Rationale),
		Limitations: strings.TrimSpace(body.Limitations), Version: body.Version}
	if msg := validPlanning(pk, &write, body.Unit); msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	part, status := planningPart(w, r, deps, session.OrgID, projectID, body.PartID)
	if status != 0 {
		return
	}
	write.PartID = part
	current, err := deps.Store.SetPlanningValue(r.Context(), session.OrgID, projectID, session.UserID, write)
	finishPlanning(w, r, deps, session.OrgID, projectID, current, err)
}

// deletePlanning withdraws a key's live value: it is superseded, kept as
// history, never deleted. ?version= is the live version; ?part_id= the part.
func deletePlanning(w http.ResponseWriter, r *http.Request, deps Deps) {
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
	pk, ok := deps.Knowledge.PlanningKey(r.PathValue("key"))
	if !ok {
		http.Error(w, "not a registered planning key", http.StatusUnprocessableEntity)
		return
	}
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil || version < 1 {
		http.Error(w, "version is required", http.StatusUnprocessableEntity)
		return
	}
	part, status := planningPart(w, r, deps, session.OrgID, projectID, r.URL.Query().Get("part_id"))
	if status != 0 {
		return
	}
	current, err := deps.Store.WithdrawPlanningValue(r.Context(), session.OrgID, projectID, part, pk.Key, pk.Scope, version)
	finishPlanning(w, r, deps, session.OrgID, projectID, current, err)
}

// planningPart resolves the part a value belongs to: the given one, or the
// whole part of the project's site. A nonzero status means it answered.
func planningPart(w http.ResponseWriter, r *http.Request, deps Deps, orgID, projectID, partID string) (string, int) {
	if partID != "" {
		if !uuidPattern.MatchString(partID) {
			http.Error(w, "unknown part", http.StatusUnprocessableEntity)
			return "", http.StatusUnprocessableEntity
		}
		return partID, 0
	}
	whole, err := deps.Store.EnsureWholePart(r.Context(), orgID, projectID)
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return "", http.StatusNotFound
	} else if err != nil {
		http.Error(w, "write failed", http.StatusInternalServerError)
		return "", http.StatusInternalServerError
	}
	return whole.ID, 0
}

// finishPlanning answers a planning write: errors per plan §3.4, otherwise
// the code-only profile rebuild and the updated planning view.
func finishPlanning(w http.ResponseWriter, r *http.Request, deps Deps, orgID, projectID string, current int64, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
		return
	case errors.Is(err, store.ErrVersionConflict):
		writeJSON(w, http.StatusConflict, map[string]any{"error": "version_conflict", "current_version": current})
		return
	case err != nil:
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}
	if err := rebuildProfile(r, deps, orgID, projectID); err != nil {
		http.Error(w, "rebuild failed", http.StatusInternalServerError)
		return
	}
	writePlanning(w, r, deps, orgID, projectID, false)
}

// planningValue reads a JSON string, number, boolean or null as text.
func planningValue(raw json.RawMessage) (*string, string) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, ""
	}
	// UseNumber keeps a number's digits exactly; float64 would round.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, "invalid value"
	}
	var s string
	switch t := v.(type) {
	case string:
		s = strings.TrimSpace(t)
	case json.Number:
		s = t.String()
	case bool:
		s = strconv.FormatBool(t)
	default:
		return nil, "value must be text, a number, true or false"
	}
	return &s, ""
}

// validPlanning is the boundary check of one planning write. It fills the
// value state when omitted. "" means valid.
func validPlanning(pk knowledge.PlanningKey, w *store.PlanningWrite, unit string) string {
	if w.State == "" {
		w.State = "set"
		if w.Value == nil {
			w.State = "unknown"
		}
	}
	switch {
	case w.State != "set" && w.State != "unknown":
		return "value_state must be set or unknown"
	case w.State == "set" && (w.Value == nil || *w.Value == ""):
		return "a set value needs a value; record unknown instead of zero or false"
	case w.State == "unknown" && w.Value != nil:
		return "an unknown value has no value"
	case w.Origin != "user" && w.Origin != "assumption":
		// Calculations are made by code, not sent by a client.
		return "origin must be user or assumption"
	case !contains([]string{"stated", "requirement", "allowance", "forecast"}, w.Meaning):
		return "meaning must be stated, requirement, allowance or forecast"
	case unit != "" && unit != pk.Unit:
		return pk.Key + " is recorded in " + orDefault(pk.Unit, "no unit")
	case len([]rune(w.Rationale)) > maxPlanningText || len([]rune(w.Limitations)) > maxPlanningText:
		return "rationale and limitations are at most 500 characters"
	case w.Version < 0:
		return "version must not be negative"
	}
	if w.Value != nil {
		if len([]rune(*w.Value)) > maxProfileText {
			return "value is longer than 200 characters"
		}
		if msg := pk.CheckPlanningValue(*w.Value); msg != "" {
			return msg
		}
		// L276: no planning value says structure, ground or compliance is fine.
		if pk.IsFavourable(*w.Value) {
			return favourableRefused(pk)
		}
	}
	return validRange(pk, w)
}

func validRange(pk knowledge.PlanningKey, w *store.PlanningWrite) string {
	if w.RangeLow == nil && w.RangeHigh == nil {
		return ""
	}
	if pk.Value != "integer" && pk.Value != "number" {
		return "only a number has a range"
	}
	if w.RangeLow != nil && w.RangeHigh != nil && *w.RangeLow > *w.RangeHigh {
		return "range_low is above range_high"
	}
	if w.Value != nil {
		v, _ := strconv.ParseFloat(*w.Value, 64)
		if (w.RangeLow != nil && v < *w.RangeLow) || (w.RangeHigh != nil && v > *w.RangeHigh) {
			return "the value is outside its range"
		}
	}
	return ""
}

func favourableRefused(pk knowledge.PlanningKey) string {
	return pk.Label + " cannot be assumed favourable: record it unknown and propose an investigation, " +
		"or state it as fact on the profile"
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
