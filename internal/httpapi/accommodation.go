package httpapi

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"sitewise/internal/knowledge"
	"sitewise/internal/profile"
	"sitewise/internal/store"
)

// The Accommodation schedule (docs/plans/2026-11-11-accommodation-schedule-lines.md):
// lines built in code on each read; the user's edits through one route. No
// Jev on read or edit.

type lineCellJSON struct {
	Value        string                `json:"value"`
	Band         string                `json:"band"`
	Sources      []profile.Source      `json:"sources"`
	Alternatives []profile.Alternative `json:"alternatives"`
	Keys         []string              `json:"keys"`
}

type lineJSON struct {
	ID      string       `json:"id"`
	Space   string       `json:"space"`
	Origin  string       `json:"origin"`
	Name    lineCellJSON `json:"name"`
	Level   lineCellJSON `json:"level"`
	Area    lineCellJSON `json:"area"`
	Comment lineCellJSON `json:"comment"`
	Many    int          `json:"many,omitempty"`
	Flag    string       `json:"flag,omitempty"`
}

type accommodationJSON struct {
	Lines   []lineJSON `json:"lines"`
	Notices []string   `json:"notices"`
	// Names are the catalogue's line names, offered while typing a new line.
	Names []string `json:"names"`
}

func accommodationBody(cat *knowledge.Catalog, view store.ProfileView, stored []profile.StoredLine) accommodationJSON {
	whole := ""
	for _, p := range view.Parts {
		if p.Kind == "whole" && whole == "" {
			whole = p.ID
		}
	}
	rows := map[string]profile.Row{}
	for _, r := range view.Rows {
		if r.PartID == whole {
			rows[r.Key] = r
		}
	}
	sched := profile.BuildSchedule(cat, rows, stored)
	out := accommodationJSON{Lines: []lineJSON{}, Notices: sched.Notices, Names: []string{}}
	if out.Notices == nil {
		out.Notices = []string{}
	}
	for _, l := range sched.Lines {
		out.Lines = append(out.Lines, lineJSON{ID: l.ID, Space: l.Space, Origin: l.Origin, Name: lineCell(l.Name), Level: lineCell(l.Level),
			Area: lineCell(l.Area), Comment: lineCell(l.Comment), Many: l.Many, Flag: l.Flag})
	}
	seen := map[string]bool{}
	for _, sp := range cat.Spaces() {
		if !seen[sp.Line] {
			seen[sp.Line] = true
			out.Names = append(out.Names, sp.Line)
		}
	}
	return out
}

func lineCell(c profile.LineCell) lineCellJSON {
	out := lineCellJSON{Value: c.Value, Band: c.Band, Sources: c.Sources, Alternatives: c.Alternatives, Keys: c.Keys}
	if out.Sources == nil {
		out.Sources = []profile.Source{}
	}
	if out.Alternatives == nil {
		out.Alternatives = []profile.Alternative{}
	}
	if out.Keys == nil {
		out.Keys = []string{}
	}
	return out
}

// readAccommodation builds the table as the profile read shows it.
func readAccommodation(ctx context.Context, deps Deps, org, project string) (accommodationJSON, error) {
	view, err := deps.Store.ReadProfile(ctx, org, project, deps.ProfileReading.Kinds())
	if err != nil {
		return accommodationJSON{}, err
	}
	stored, err := deps.Store.AccommodationLines(ctx, org, project)
	if err != nil {
		return accommodationJSON{}, err
	}
	return accommodationBody(deps.Knowledge, view, stored), nil
}

type accommodationEdit struct {
	// Op is set, reset, delete, split, insert, duplicate or order.
	Op    string   `json:"op"`
	Lines []string `json:"lines"`
	Field string   `json:"field"`
	Value string   `json:"value"`
	// Order is every line shown, in the new order; insert puts "new:0"
	// where the new line goes.
	Order []string `json:"order"`
}

// postAccommodation applies one edit to the schedule and returns the table.
func postAccommodation(w http.ResponseWriter, r *http.Request, deps Deps) {
	if !originOK(r, deps.PublicOrigin) {
		http.Error(w, "origin rejected", http.StatusForbidden)
		return
	}
	session, ok := memberSession(w, r, deps)
	if !ok {
		return
	}
	project := r.PathValue("id")
	if !uuidPattern.MatchString(project) || deps.Knowledge == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	var body accommodationEdit
	if err := readJSON(w, r, deps.MaxBodyBytes, &body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if msg := validLineRefs(deps.Knowledge, body.Lines, false); msg != "" {
		http.Error(w, msg, http.StatusUnprocessableEntity)
		return
	}
	ctx, org := r.Context(), session.OrgID
	var err error
	switch body.Op {
	case "set":
		value, msg := lineValue(body.Field, body.Value)
		if msg != "" || len(body.Lines) == 0 {
			http.Error(w, firstOf(msg, "lines required"), http.StatusUnprocessableEntity)
			return
		}
		err = deps.Store.SetAccommodationField(ctx, org, project, body.Lines, body.Field, value)
	case "reset":
		err = deps.Store.ResetAccommodationLines(ctx, org, project, body.Lines)
	case "delete":
		err = deps.Store.RemoveAccommodationLines(ctx, org, project, body.Lines)
	case "split":
		if len(body.Lines) != 1 {
			http.Error(w, "split takes one line", http.StatusUnprocessableEntity)
			return
		}
		err = deps.Store.SplitAccommodationLine(ctx, org, project, body.Lines[0])
	case "order", "insert":
		if msg := validLineRefs(deps.Knowledge, body.Order, body.Op == "insert"); msg != "" || len(body.Order) == 0 {
			http.Error(w, firstOf(msg, "order required"), http.StatusUnprocessableEntity)
			return
		}
		var added []profile.StoredLine
		if body.Op == "insert" {
			name, msg := lineValue("name", firstOf(strings.TrimSpace(body.Value), "New space"))
			if msg != "" {
				http.Error(w, msg, http.StatusUnprocessableEntity)
				return
			}
			added = []profile.StoredLine{{Name: name, Set: []string{"name"}}}
		}
		_, err = deps.Store.PlaceAccommodationLines(ctx, org, project, body.Order, added)
	case "duplicate":
		err = duplicateLines(ctx, deps, org, project, body.Lines)
	default:
		http.Error(w, "unknown op", http.StatusUnprocessableEntity)
		return
	}
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
		return
	case errors.Is(err, store.ErrInvalidInput):
		http.Error(w, "invalid line", http.StatusUnprocessableEntity)
		return
	case err != nil:
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}
	if deps.Broker != nil {
		deps.Broker.Wake(org)
	}
	out, err := readAccommodation(ctx, deps, org, project)
	if err != nil {
		http.Error(w, "read failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// duplicateLines adds a copy of each line straight after it, with the
// next free number ("Office 9") and the line's level, area and comment as
// the user's own.
func duplicateLines(ctx context.Context, deps Deps, org, project string, refs []string) error {
	if len(refs) == 0 {
		return store.ErrInvalidInput
	}
	table, err := readAccommodation(ctx, deps, org, project)
	if err != nil {
		return err
	}
	picked := map[string]bool{}
	for _, ref := range refs {
		picked[ref] = true
	}
	var names []string
	for _, l := range table.Lines {
		names = append(names, l.Name.Value)
	}
	var order []string
	var added []profile.StoredLine
	for _, l := range table.Lines {
		order = append(order, l.ID)
		if !picked[l.ID] {
			continue
		}
		name := nextName(l.Name.Value, names)
		names = append(names, name)
		c := profile.StoredLine{Name: name, Level: l.Level.Value, Area: l.Area.Value, Comment: l.Comment.Value, Set: []string{"name"}}
		for f, v := range map[string]string{"level": c.Level, "area": c.Area, "comment": c.Comment} {
			if v != "" {
				c.Set = append(c.Set, f)
			}
		}
		order = append(order, "new:"+strconv.Itoa(len(added)))
		added = append(added, c)
	}
	if len(added) != len(picked) {
		return store.ErrNotFound // a line not in this project's table
	}
	_, err = deps.Store.PlaceAccommodationLines(ctx, org, project, order, added)
	return err
}

var trailingNumber = regexp.MustCompile(`^(.*?)\s*(\d+)$`)

// nextName numbers a copy after the highest line of the same name.
func nextName(name string, taken []string) string {
	stem := name
	if m := trailingNumber.FindStringSubmatch(name); m != nil && m[1] != "" {
		stem = m[1]
	}
	high := 1
	for _, t := range taken {
		if m := trailingNumber.FindStringSubmatch(t); m != nil && strings.EqualFold(m[1], stem) {
			if n, _ := strconv.Atoi(m[2]); n > high {
				high = n
			}
		}
	}
	return stem + " " + strconv.Itoa(high+1)
}

// validLineRefs checks line references at the boundary: a line the user
// added (its id), a generated line of a known space ("offices#3"), and in
// an insert's order the new line ("new:0").
func validLineRefs(cat *knowledge.Catalog, refs []string, insert bool) string {
	if len(refs) > 2000 {
		return "too many lines"
	}
	news := 0
	for _, ref := range refs {
		if uuidPattern.MatchString(ref) {
			continue
		}
		if ref == "new:0" && insert {
			news++
			continue
		}
		space, _, ok := profile.ParseLineKey(ref)
		if _, known := cat.Space(space); !ok || !known {
			return "unknown line"
		}
	}
	if insert && news != 1 {
		return "insert places one new line"
	}
	return ""
}

// lineValue checks and tidies a typed value. A name is never blank; an
// area is a number of square metres or blank.
func lineValue(field, v string) (string, string) {
	v = strings.TrimSpace(v)
	limit := map[string]int{"name": 120, "level": 80, "area": 20, "comment": 500}[field]
	switch {
	case limit == 0:
		return "", "unknown field"
	case utf8.RuneCountInString(v) > limit || len(v) > limit*4:
		return "", field + " is too long"
	case field == "name" && v == "":
		return "", "a space needs a name"
	case field == "area" && v != "":
		v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(v, "m²"), "m2"), "sqm"))
		f, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", ""), 64)
		if err != nil || f < 0 || f > 1e7 {
			return "", "area must be square metres"
		}
		return strconv.FormatFloat(f, 'f', -1, 64), ""
	}
	return v, ""
}

func firstOf(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
