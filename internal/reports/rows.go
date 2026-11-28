package reports

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"time"
)

// What the PM owns in the report's tables (project report plan, section 6):
// a row of their own, or their words laid over a generated row. Update
// never touches either; a typed-over cell whose SiteWise value has moved is
// flagged and holds issue until the PM keeps theirs or takes SiteWise's.

// ReportRow is one report_rows row.
type ReportRow struct {
	ID        string             `json:"id"`
	TableID   string             `json:"table_id"`
	SourceKey string             `json:"source_key,omitempty"` // "" is the PM's own row
	Position  *float64           `json:"position,omitempty"`
	Cells     map[string]RowCell `json:"cells"` // by column index
	Hidden    bool               `json:"hidden"`
	Source    string             `json:"source,omitempty"`
	CreatedAt time.Time          `json:"created_at"`
	Version   int64              `json:"version"`
}

// RowCell is the PM's words in one cell. Base is the hash of SiteWise's
// words in that cell when they typed; empty on their own row.
type RowCell struct {
	Text string `json:"text"`
	Base string `json:"base,omitempty"`
}

// CellHash is the fingerprint of a generated cell's words.
func CellHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:8])
}

// FreshTables start empty each period: the PM's rows made before the last
// issue are not drawn again (the revision history).
var FreshTables = map[string]bool{"revisions": true}

// ApplyRows lays the PM's rows over the generated tables. lastIssue is when
// the last issue was made (zero for none). It returns the sections with the
// rows in place; a table that holds a flagged row marks its block Conflict,
// so issue waits as it does for a protected paragraph.
func ApplyRows(sections []Section, rows []ReportRow, lastIssue time.Time) []Section {
	byTable := map[string][]ReportRow{}
	for _, r := range rows {
		byTable[r.TableID] = append(byTable[r.TableID], r)
	}
	for si := range sections {
		for bi := range sections[si].Blocks {
			b := &sections[si].Blocks[bi]
			if b.Table == nil || b.Table.ID == "" {
				continue
			}
			t := layRows(*b.Table, byTable[b.Table.ID], lastIssue)
			b.Table = &t
			for _, f := range t.Meta {
				if f.Flag != "" {
					b.Conflict = true
				}
			}
		}
	}
	return sections
}

type placed struct {
	row  []string
	fact RowFacts
	pos  float64
	seq  int
}

func layRows(t Table, own []ReportRow, lastIssue time.Time) Table {
	over := map[string]ReportRow{}
	var mine []ReportRow
	for _, r := range own {
		if r.SourceKey == "" {
			if FreshTables[t.ID] && !lastIssue.IsZero() && r.CreatedAt.Before(lastIssue) {
				continue
			}
			mine = append(mine, r)
			continue
		}
		over[r.SourceKey] = r
	}
	width := len(t.Columns)
	var out []placed
	var hiddenIDs, hiddenKeys []string
	for i, row := range t.Rows {
		f := RowFacts{}
		if i < len(t.Meta) {
			f = t.Meta[i]
		}
		p := placed{row: append([]string(nil), row...), fact: f, pos: float64(i), seq: len(out)}
		if r, ok := over[f.Key]; ok {
			delete(over, f.Key)
			if r.Hidden {
				hiddenIDs, hiddenKeys = append(hiddenIDs, r.ID), append(hiddenKeys, f.Key)
				continue
			}
			p.fact.ID, p.fact.Source = r.ID, r.Source
			if r.Position != nil {
				p.pos = *r.Position
			}
			typedAny := false
			for c := 0; c < width && c < len(p.row); c++ {
				cell, ok := r.Cells[strconv.Itoa(c)]
				if !ok {
					continue
				}
				if p.fact.Typed == nil {
					p.fact.Typed, p.fact.Base = make([]bool, width), make([]string, width)
				}
				p.fact.Typed[c], p.fact.Base[c] = true, p.row[c]
				if f.Kind != RowPlaceholder && cell.Base != CellHash(p.row[c]) {
					p.fact.Flag = "moved"
				}
				p.row[c] = cell.Text
				typedAny = true
			}
			// A placeholder typed in is the PM's row from then on.
			if f.Kind == RowPlaceholder && typedAny {
				p.fact.Kind = RowOwn
			}
		}
		out = append(out, p)
	}
	// The PM's words over a record that is gone stay, flagged.
	var gone []string
	for key := range over {
		gone = append(gone, key)
	}
	sort.Strings(gone)
	for _, key := range gone {
		r := over[key]
		if r.Hidden || len(r.Cells) == 0 {
			continue
		}
		p := placed{row: make([]string, width), fact: RowFacts{Key: key, Kind: RowOwn, ID: r.ID, Flag: "missing", Typed: make([]bool, width)}, pos: float64(len(t.Rows)), seq: len(out)}
		if len(key) > 3 && key[:3] == "ph:" {
			p.fact.Flag = "" // a placeholder the template no longer lists: the PM's row
		}
		for c := 0; c < width; c++ {
			if cell, ok := r.Cells[strconv.Itoa(c)]; ok {
				p.row[c], p.fact.Typed[c] = cell.Text, true
			}
		}
		if r.Position != nil {
			p.pos = *r.Position
		}
		out = append(out, p)
	}
	for _, r := range mine {
		if r.Hidden {
			continue
		}
		p := placed{row: make([]string, width), fact: RowFacts{Kind: RowOwn, ID: r.ID, Source: r.Source}, pos: float64(len(t.Rows)) + 0.5, seq: len(out)}
		for c := 0; c < width; c++ {
			p.row[c] = r.Cells[strconv.Itoa(c)].Text
		}
		if r.Position != nil {
			p.pos = *r.Position
		}
		out = append(out, p)
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].pos != out[b].pos {
			return out[a].pos < out[b].pos
		}
		return out[a].seq < out[b].seq
	})
	t.Rows, t.Meta, t.Hidden, t.HiddenIDs, t.HiddenKeys = make([][]string, 0, len(out)), make([]RowFacts, 0, len(out)), len(hiddenIDs), hiddenIDs, hiddenKeys
	for _, p := range out {
		p.fact.Pos = p.pos
		t.Rows = append(t.Rows, p.row)
		t.Meta = append(t.Meta, p.fact)
	}
	return t
}

// MovePosition is the position that puts row index i one place up (dir
// -1) or down (+1) in a laid table: halfway between its new neighbours.
// ok is false at either end.
func MovePosition(t Table, i, dir int) (float64, bool) {
	pos := func(j int) float64 { return t.Meta[j].Pos }
	j := i + dir
	if i < 0 || i >= len(t.Meta) || j < 0 || j >= len(t.Meta) {
		return 0, false
	}
	beyond := j + dir
	if beyond < 0 || beyond >= len(t.Meta) {
		return pos(j) + float64(dir), true
	}
	return (pos(j) + pos(beyond)) / 2, true
}
