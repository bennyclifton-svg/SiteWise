package profile

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"sitewise/internal/knowledge"
)

// The Accommodation schedule as lines
// (docs/plans/2026-11-11-accommodation-schedule-lines.md): one line per room,
// built in code on every profile read from the building type's typical
// spaces, the documents' readings (space.* rows), the card's counts and the
// user's stored edits. Nothing here calls Jev. Unknown stays unknown: no
// area is divided, no count is guessed.

const (
	// lineCap is the most rooms of one kind shown as numbered lines before
	// the user asks; above it one line stands for them all.
	lineCap = 30
	// splitCap bounds a split, so one typed number cannot make a huge table.
	splitCap = 500
)

// StoredLine is what the user changed on one line: a generated line, by
// its key, or a line the user added (Key empty).
type StoredLine struct {
	ID, Key                    string
	Name, Level, Area, Comment string
	// Set lists the fields the user set: name, level, area, comment. A
	// field set blank stays blank.
	Set      []string
	Position *float64
	Split    bool
	Removed  bool
}

// LineCell is one value of a line and how it was decided.
type LineCell struct {
	Value string
	// Band is green, amber or red (the documents), user, typical (a
	// placeholder's name) or empty.
	Band         string
	Sources      []Source
	Alternatives []Alternative
	// Keys are the readings behind the value, for the update's marks.
	Keys []string
}

// Line is one line of the schedule.
type Line struct {
	// ID is "<space>#<n>" for a generated line, the row id for a line the
	// user added.
	ID    string
	Space string
	// Origin is typical, document, card or user.
	Origin                     string
	Name, Level, Area, Comment LineCell
	// Many is how many rooms this one line stands for until it is split.
	Many int
	// Flag says, in words, why the line needs a look.
	Flag string
}

// Schedule is the table and what needs saying under it.
type Schedule struct {
	Lines   []Line
	Notices []string
}

// LineKey names generated line n of a space.
func LineKey(space string, n int) string { return space + "#" + strconv.Itoa(n) }

// ParseLineKey splits a generated line's key.
func ParseLineKey(key string) (space string, n int, ok bool) {
	space, num, found := strings.Cut(key, "#")
	if !found || space == "" {
		return "", 0, false
	}
	n, err := strconv.Atoi(num)
	if err != nil || n < 1 || n > splitCap || strconv.Itoa(n) != num {
		return "", 0, false
	}
	return space, n, true
}

// BuildSchedule lays out the schedule. rows are the whole project's profile
// rows by key; stored are the user's edits.
func BuildSchedule(cat *knowledge.Catalog, rows map[string]Row, stored []StoredLine) Schedule {
	r := newReading(rows)
	byKey := map[string]StoredLine{}
	var added []StoredLine
	for _, s := range stored {
		if s.Key == "" {
			added = append(added, s)
		} else {
			byKey[s.Key] = s
		}
	}
	type placed struct {
		line Line
		at   float64
	}
	var out []placed
	natural := 0.0
	generated := map[string]bool{}
	lastOf := map[string]float64{}
	counted := map[string]int{} // rooms the documents or the card list, by space
	carded := map[string]int{}  // rooms the card lists, by space
	for _, e := range scheduleSpaces(cat, r) {
		lines, n, card := spaceLines(cat, r, e, byKey[LineKey(e.sp.ID, 1)].Split)
		if n > 0 {
			counted[e.sp.ID] = n
		}
		if card && e.sp.Kind == "room" && n <= lineCap {
			carded[e.sp.ID] = n
		}
		for _, l := range lines {
			generated[l.ID] = true
			s, edited := byKey[l.ID]
			if edited && s.Removed {
				continue
			}
			at := natural
			natural++
			if edited {
				applyEdits(&l, s)
				if s.Position != nil {
					at = *s.Position
				}
			}
			lastOf[e.sp.ID] = at
			out = append(out, placed{l, at})
		}
	}
	// An edited line its space no longer generates stays, flagged; one that
	// only held its place goes.
	for _, s := range stored {
		if s.Key == "" || generated[s.Key] || s.Removed || len(s.Set) == 0 {
			continue
		}
		id, k, ok := ParseLineKey(s.Key)
		sp, known := cat.Space(id)
		if !ok || !known {
			continue
		}
		l := Line{ID: s.Key, Space: id, Origin: "user", Name: LineCell{Value: numbered(sp.Line, k, 2)}}
		applyEdits(&l, s)
		if n, listed := counted[id]; listed {
			l.Flag = fmt.Sprintf("The documents now list %d %s.", n, sp.Unit)
		} else {
			l.Flag = "No longer in the documents or the typical list."
		}
		at, near := lastOf[id]
		if !near {
			at = natural
		}
		at += 0.5
		if s.Position != nil {
			at = *s.Position
		}
		out = append(out, placed{l, at})
	}
	for _, s := range added {
		l := Line{ID: s.ID, Origin: "user"}
		applyEdits(&l, s)
		at := natural
		natural++
		if s.Position != nil {
			at = *s.Position
		}
		out = append(out, placed{l, at})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].at < out[j].at })
	var sched Schedule
	for _, p := range out {
		sched.Lines = append(sched.Lines, p.line)
	}
	sched.Notices = cardNotices(cat, carded, sched.Lines)
	return sched
}

// reading answers questions about the profile rows. bySpace holds each
// space's readings by attribute ("count", "need.av_ready"), so a space's
// comment never scans the whole catalogue for adjacencies.
type reading struct {
	rows    map[string]Row
	bySpace map[string]map[string]Row
}

func newReading(rows map[string]Row) reading {
	r := reading{rows: rows, bySpace: map[string]map[string]Row{}}
	for key, row := range rows {
		rest, ok := strings.CutPrefix(key, knowledge.SpacePrefix)
		if !ok {
			continue
		}
		id, attr, ok := strings.Cut(rest, ".")
		if !ok {
			continue
		}
		if r.bySpace[id] == nil {
			r.bySpace[id] = map[string]Row{}
		}
		r.bySpace[id][attr] = row
	}
	return r
}

// value is a row's value when it is applied: from the documents (green or
// amber) or the user.
func (r reading) value(key string) (Row, bool) {
	row, ok := r.rows[key]
	if !ok || row.Value == "" {
		return row, false
	}
	switch row.Band {
	case "green", "amber", "user", "planning":
		return row, true
	}
	return row, false
}

// disputed is a row the documents disagree on.
func (r reading) disputed(key string) (Row, bool) {
	row, ok := r.rows[key]
	return row, ok && row.Band == "red"
}

func (r reading) number(key string) (int, Row, bool) {
	row, ok := r.value(key)
	if !ok {
		return 0, row, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(row.Value), 64)
	if err != nil || f < 0 || f != float64(int(f)) {
		return 0, row, false
	}
	return int(f), row, true
}

// shown says whether the documents put a space in the schedule: they say
// the project provides it, give it a value, or disagree about it. Their
// "no" takes it out.
func (r reading) shown(id string) (yes, no bool) {
	attrs := r.bySpace[id]
	if len(attrs) == 0 {
		return false, false
	}
	base := knowledge.SpacePrefix + id + "."
	if p, ok := r.value(base + "provided"); ok && p.Value == "no" {
		return false, true
	}
	if p, ok := r.value(base + "provided"); ok && p.Value == "yes" {
		return true, false
	}
	for _, a := range knowledge.SpaceAttributes {
		if _, ok := attrs[a]; !ok {
			continue
		}
		if _, ok := r.value(base + a); ok && a != "provided" {
			return true, false
		}
		if _, ok := r.disputed(base + a); ok {
			return true, false
		}
	}
	return false, false
}

type scheduleSpace struct {
	sp      knowledge.Space
	line    string // the building type's name for its line, when it has one
	typical bool
}

// scheduleSpaces lists the spaces in the table: the building type's
// typical spaces in their list's order, then each space only the documents
// name, placed among them by the catalogue's order.
func scheduleSpaces(cat *knowledge.Catalog, r reading) []scheduleSpace {
	at := func(key string) string {
		row, _ := r.value(key)
		return row.Value
	}
	index := map[string]int{}
	for i, s := range cat.Spaces() {
		index[s.ID] = i
	}
	var out []scheduleSpace
	listed := map[string]bool{}
	for _, t := range cat.TypicalSpaces(at("hdr.subclass"), at("hdr.building_class"), at("hdr.work_type")) {
		sp, ok := cat.Space(t.Space)
		if _, no := r.shown(t.Space); !ok || no {
			continue
		}
		listed[t.Space] = true
		out = append(out, scheduleSpace{sp, t.Line, true})
	}
	for _, sp := range cat.Spaces() {
		if yes, _ := r.shown(sp.ID); !yes || listed[sp.ID] {
			continue
		}
		e := scheduleSpace{sp: sp}
		at := slices.IndexFunc(out, func(o scheduleSpace) bool { return index[o.sp.ID] > index[sp.ID] })
		if at < 0 {
			out = append(out, e)
		} else {
			out = slices.Insert(out, at, e)
		}
	}
	return out
}

// spaceLines makes a space's lines, and says how many rooms the documents
// or the card list for it (0 when unknown) and whether the card's count did.
func spaceLines(cat *knowledge.Catalog, r reading, e scheduleSpace, split bool) ([]Line, int, bool) {
	sp := e.sp
	base := knowledge.SpacePrefix + sp.ID + "."
	lineName := sp.Line
	if e.line != "" {
		lineName = e.line
	}
	origin := "typical"
	name := LineCell{Value: lineName, Band: "typical"}
	if yes, _ := r.shown(sp.ID); yes {
		origin = "document"
		name = documentName(r, base, lineName)
	}
	var notes []note
	n, countRow, cardKey, known := roomCount(r, sp)
	if cardKey != "" {
		origin = "card"
		name = LineCell{Value: lineName, Band: countRow.Band, Sources: countRow.Sources, Keys: []string{cardKey}}
	}
	if !known {
		if row, ok := r.disputed(base + "count"); ok {
			notes = append(notes, note{text: strings.Join(altValues(row), " or ") + " in the documents", row: row, key: base + "count"})
		}
	}
	countKey := base + "count"
	if cardKey != "" {
		countKey = cardKey
	}
	lines := 1
	many := 0
	switch {
	case !known:
	case sp.Kind == "setting":
		notes = append(notes, note{text: fmt.Sprintf("%d %s", n, sp.Unit), row: countRow, key: countKey})
	case n > splitCap || n > lineCap && !split:
		many = n
		notes = append(notes, note{text: fmt.Sprintf("%d %s", n, sp.Unit), row: countRow, key: countKey})
	case n > 1:
		lines = n
	}
	if many > 0 {
		name.Value = sp.Label
	}
	level := cellOf(r, base+"level")
	area := LineCell{}
	if each := cellOf(r, base+"area_each_sqm"); each.Value != "" || each.Band == "red" {
		area = each
	} else if whole := cellOf(r, base+"area_sqm"); whole.Value != "" || whole.Band == "red" {
		if lines == 1 && many == 0 {
			area = whole
		} else if row, ok := r.value(base + "area_sqm"); ok {
			rooms := max(lines, many)
			notes = append(notes, note{text: fmt.Sprintf("%s m² for all %d", row.Value, rooms), row: row, key: base + "area_sqm"})
		}
	}
	sizes := roomSizes(r, base, lines)
	var capacity *note
	if row, ok := r.value(base + "capacity"); ok && sizes == nil {
		capacity = &note{text: "Capacity " + row.Value, row: row, key: base + "capacity"}
	}
	var tail []note
	if len(r.bySpace[sp.ID]) > 0 {
		for _, q := range cat.SpaceRequirements() {
			if row, ok := r.value(base + "need." + q.ID); ok && row.Value == "yes" {
				tail = append(tail, note{text: q.Label, row: row, key: base + "need." + q.ID})
			}
		}
		tail = append(tail, adjacencies(cat, r, sp.ID)...)
	}
	out := make([]Line, 0, lines)
	for i := 1; i <= lines; i++ {
		l := Line{ID: LineKey(sp.ID, i), Space: sp.ID, Origin: origin, Name: name, Level: level, Area: area, Many: many}
		l.Name.Value = numbered(name.Value, i, lines)
		var parts []note
		if sizes != nil {
			parts = append(parts, sizes[i-1])
		} else if capacity != nil {
			parts = append(parts, *capacity)
		}
		parts = append(parts, notes...)
		parts = append(parts, tail...)
		l.Comment = comment(parts)
		out = append(out, l)
	}
	if !known {
		n = 0
	}
	return out, n, cardKey != ""
}

// adjacencies are the spaces the documents place this one next to, in the
// catalogue's order: "adjacent to dining".
func adjacencies(cat *knowledge.Catalog, r reading, id string) []note {
	var others []string
	for attr := range r.bySpace[id] {
		if other, ok := strings.CutPrefix(attr, "next_to."); ok {
			others = append(others, other)
		}
	}
	if len(others) == 0 {
		return nil
	}
	var out []note
	base := knowledge.SpacePrefix + id + ".next_to."
	for _, o := range cat.Spaces() {
		if !slices.Contains(others, o.ID) {
			continue
		}
		if row, ok := r.value(base + o.ID); ok && row.Value == "yes" {
			out = append(out, note{text: "adjacent to " + strings.ToLower(o.Line), row: row, key: base + o.ID})
		}
	}
	return out
}

// roomCount is the number of rooms: the card's count when it states one
// (the card is the count RFPs use), else the documents'.
func roomCount(r reading, sp knowledge.Space) (n int, row Row, cardKey string, ok bool) {
	for _, k := range sp.CountFrom {
		if n, row, ok := r.number(k); ok {
			return n, row, k, true
		}
	}
	n, row, ok = r.number(knowledge.SpacePrefix + sp.ID + ".count")
	return n, row, "", ok
}

// roomSizes gives each line its own capacity from a size list whose rooms
// add up to the lines ("4x8,2x16" over six rooms); nil otherwise.
func roomSizes(r reading, base string, lines int) []note {
	row, ok := r.value(base + "mix")
	if !ok {
		return nil
	}
	var out []note
	for _, part := range strings.Split(row.Value, ",") {
		count, size, found := strings.Cut(strings.TrimSpace(part), "x")
		k, err := strconv.Atoi(count)
		if !found || err != nil || k < 1 {
			return nil
		}
		for range k {
			out = append(out, note{text: "Capacity " + size, row: row, key: base + "mix"})
		}
	}
	if len(out) != lines {
		return nil
	}
	return out
}

// documentName is a document line's name cell: how the documents decided
// the space is provided.
func documentName(r reading, base, name string) LineCell {
	for _, a := range knowledge.SpaceAttributes {
		if row, ok := r.rows[base+a]; ok && (row.Band == "green" || row.Band == "amber" || row.Band == "red" || row.Band == "user") {
			c := LineCell{Value: name, Band: row.Band, Sources: row.Sources, Keys: []string{base + a}}
			if a == "provided" && row.Band == "red" {
				c.Alternatives = row.Alternatives
			} else if row.Band == "red" {
				c.Band = "amber" // a disputed value, not a disputed space
			}
			return c
		}
	}
	return LineCell{Value: name, Band: "amber"}
}

// cellOf is a reading as a line's cell: its applied value, or the
// documents' disagreement.
func cellOf(r reading, key string) LineCell {
	if row, ok := r.value(key); ok {
		return LineCell{Value: row.Value, Band: row.Band, Sources: row.Sources, Keys: []string{key}}
	}
	if row, ok := r.disputed(key); ok {
		return LineCell{Band: "red", Sources: row.Sources, Alternatives: row.Alternatives, Keys: []string{key}}
	}
	return LineCell{}
}

type note struct {
	text string
	row  Row
	key  string
}

// comment joins a line's notes; it is as sure as its least sure note.
func comment(parts []note) LineCell {
	if len(parts) == 0 {
		return LineCell{}
	}
	var texts []string
	c := LineCell{}
	rank := map[string]int{"green": 1, "user": 1, "planning": 1, "amber": 2, "red": 3}
	seen := map[string]bool{}
	for _, p := range parts {
		texts = append(texts, p.text)
		band := p.row.Band
		if rank[band] > rank[c.Band] {
			c.Band = band
		}
		if !seen[p.key] {
			seen[p.key] = true
			c.Keys = append(c.Keys, p.key)
			c.Sources = append(c.Sources, p.row.Sources...)
		}
	}
	text := strings.Join(texts, ", ")
	c.Value = strings.ToUpper(text[:1]) + text[1:]
	return c
}

func altValues(row Row) []string {
	var out []string
	for _, a := range row.Alternatives {
		if a.Value != "" && !slices.Contains(out, a.Value) {
			out = append(out, a.Value)
		}
	}
	return out
}

// numbered names line i of n: "Office 3", or "Office" when it is the only one.
func numbered(name string, i, n int) string {
	if n <= 1 {
		return name
	}
	return name + " " + strconv.Itoa(i)
}

// applyEdits puts the user's fields over a line's; they are final.
func applyEdits(l *Line, s StoredLine) {
	for _, f := range s.Set {
		c := LineCell{Band: "user"}
		switch f {
		case "name":
			c.Value = s.Name
			l.Name = c
		case "level":
			c.Value = s.Level
			l.Level = c
		case "area":
			c.Value = s.Area
			l.Area = c
		case "comment":
			c.Value = s.Comment
			l.Comment = c
		}
	}
}

// cardNotices says where the schedule and the card count a room
// differently. The card keeps its count; the user decides. A line the user
// added counts when its name starts with the room's ("Bedroom 5").
func cardNotices(cat *knowledge.Catalog, carded map[string]int, lines []Line) []string {
	var out []string
	for _, sp := range cat.Spaces() {
		n, ok := carded[sp.ID]
		if !ok {
			continue
		}
		listed := 0
		for _, l := range lines {
			if l.Space == sp.ID || l.Space == "" && strings.HasPrefix(strings.ToLower(l.Name.Value), strings.ToLower(sp.Line)) {
				listed++
			}
		}
		if listed != n {
			out = append(out, fmt.Sprintf("The card has %d %s; the schedule lists %d.", n, sp.Unit, listed))
		}
	}
	return out
}
