package httpapi_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"sitewise/internal/chatfix"
	"sitewise/internal/llm"
)

// The basement set: one document for each way a document can be about the
// basement, and the ones that must stay out.
var basementDocs = []chatfix.Document{
	// Title only: the title says basement, the text does not.
	{Title: "Basement excavation plan", Kind: "drawing", Discipline: "structural", Pages: []string{"Set out and levels for the bulk dig."}},
	// Text only.
	{Title: "Geotechnical investigation", Kind: "report", Discipline: "geotechnical", Pages: []string{
		"Groundwater was met at 2.1 m, above the basement floor level.\n\nThe basement walls must be designed as tanked.",
		"Borehole BH2 met weathered shale at 3.4 m."}},
	// Labelled by the profile's reading with the basement system only.
	{Title: "Shoring design", Kind: "report", Discipline: "structural", Pages: []string{"Soldier piles at 2.4 m centres with shotcrete infill."},
		Systems: []string{"substructure.basement-retention"}},
	// Title and text: listed once, with both reasons.
	{Title: "Basement drainage specification", Kind: "specification", Discipline: "hydraulic", Pages: []string{
		"The basement sump pump shall be duty and standby."}},
	// A scan: no text, so only its title can match.
	{Title: "Basement survey", Kind: "drawing"},
	// A superseded revision and its current one, both mentioning it.
	{Title: "Structural notes rev A", Kind: "drawing", Number: "S-001", Revision: "A", Pages: []string{"Basement slab 200 thick."}},
	{Title: "Structural notes rev B", Kind: "drawing", Number: "S-001", Revision: "B", Supersedes: "Structural notes rev A", Pages: []string{"Basement slab 250 thick."}},
	// Out: a label on the wider system is not about the basement, and an
	// unrelated document.
	{Title: "Footing report", Kind: "report", Pages: []string{"Pad footings on rock."}, Systems: []string{"substructure"}},
	{Title: "Landscape plan", Kind: "drawing", Pages: []string{"Planting along the street frontage."}},
}

func seedBasement(t *testing.T, ta *toolApp) {
	t.Helper()
	if _, err := chatfix.SeedDocuments(context.Background(), ta.a.store, ta.m.orgID, ta.project, basementDocs); err != nil {
		t.Fatal(err)
	}
}

// rowOf is the result row for a title, or "" when it is not listed.
func rowOf(result, title string) string {
	for _, line := range strings.Split(result, "\n") {
		if parts := strings.Split(line, " | "); len(parts) > 1 && parts[1] == title {
			return line
		}
	}
	return ""
}

func TestDocumentsAboutRanksBySignals(t *testing.T) {
	ta := newToolApp(t)
	seedBasement(t, ta)
	// The same org's other project, and another org, each hold a basement
	// document. Neither is ever in the set.
	other := ta.m.createProject(t, "Other job")
	if _, err := chatfix.SeedDocuments(context.Background(), ta.a.store, ta.m.orgID, other, []chatfix.Document{{Title: "Basement plan other job", Kind: "drawing", Pages: []string{"Basement level B1."}}}); err != nil {
		t.Fatal(err)
	}
	stranger := ta.a.member(t, newUUID(t))
	theirs := stranger.createProject(t, "Theirs")
	if _, err := chatfix.SeedDocuments(context.Background(), ta.a.store, stranger.orgID, theirs, []chatfix.Document{{Title: "Basement plan other org", Kind: "drawing", Pages: []string{"Basement level B2."}}}); err != nil {
		t.Fatal(err)
	}

	ta.run(t, "Which documents are about the basement?", tool("documents", map[string]any{"about": "the basement"}))
	got := ta.results[0]
	// Each row: id | title | kind | why it is included | state.
	want := map[string]string{
		"Basement excavation plan":        "title",
		"Geotechnical investigation":      "2 passages",
		"Shoring design":                  "labelled Basement excavation, retention and dewatering (1 passage)",
		"Basement drainage specification": "title; 1 passage",
		"Basement survey":                 "title",
		"Structural notes rev B":          "1 passage",
	}
	for title, why := range want {
		row := strings.Split(rowOf(got, title), " | ")
		if len(row) != 5 || row[3] != why {
			t.Errorf("%s: row %q, want why %q in:\n%s", title, row, why, got)
		}
	}
	if row := rowOf(got, "Basement survey"); !strings.HasSuffix(row, "| current, a scan with no text") {
		t.Errorf("the scan is not marked: %q", row)
	}
	for _, out := range []string{"Structural notes rev A", "Footing report", "Landscape plan", "Basement plan other job", "Basement plan other org"} {
		if rowOf(got, out) != "" {
			t.Errorf("%s is in the set:\n%s", out, got)
		}
	}
	if n := strings.Count(got, "| Basement drainage specification |"); n != 1 {
		t.Errorf("a document matched twice is listed %d times", n)
	}
	// Two signals rank first; the scan is in the set and counted as matched
	// by title only.
	if first := strings.Index(got, "| Basement drainage specification |"); first < 0 || first > strings.Index(got, "| Basement excavation plan |") || first > strings.Index(got, "| Shoring design |") {
		t.Errorf("the two-signal document is not first:\n%s", got)
	}
	for _, line := range []string{
		"Searched the titles of 8 filed documents, and their text and building-system labels, for: basement.",
		"1 has no text (scans) and could be matched by title only, 1 of them in this set: Basement survey.",
		"Labels cover only documents the profile has read.",
		"Left out 1 superseded revision; ask with superseded true to include them.",
	} {
		if !strings.Contains(got, line) {
			t.Errorf("the result lacks %q:\n%s", line, got)
		}
	}
	if strings.Contains(got, "select_ids") {
		t.Errorf("ids to select without select:\n%s", got)
	}

	// Asked for: the superseded revision comes back, after the current one.
	ta.run(t, "Include old revisions", tool("documents", map[string]any{"about": "basement", "superseded": true}))
	if row := rowOf(ta.results[0], "Structural notes rev A"); !strings.Contains(row, "superseded by Structural notes rev B") ||
		strings.Index(ta.results[0], "| Structural notes rev B |") > strings.Index(ta.results[0], "| Structural notes rev A |") {
		t.Errorf("superseded asked for:\n%s", ta.results[0])
	}
}

// viewOf is the turn's view event, or nil when the screen did not move.
func viewOf(events []chatEvent) *chatEvent {
	for i := range events {
		if events[i].Type == "view" {
			return &events[i]
		}
	}
	return nil
}

func TestDocumentsAboutSelect(t *testing.T) {
	ta := newToolApp(t)
	seedBasement(t, ta)
	// The rows as the model would be shown them, in rank order.
	ta.run(t, "Which documents are about the basement?", tool("documents", map[string]any{"about": "basement documents"}))
	var listed []string
	for _, r := range regexp.MustCompile(`(?m)^([0-9a-f]{8}) \| `).FindAllStringSubmatch(ta.results[0], -1) {
		listed = append(listed, r[1])
	}
	// With select the screen moves in the same round and Go's summary is the
	// answer: the model is not asked again, so it cannot restate the counts.
	events := ta.run(t, "Select the basement documents", tool("documents", map[string]any{"about": "basement documents", "select": true}))
	view := viewOf(events)
	if view == nil || view.View.Do != "select" || view.View.Table != "documents" ||
		view.View.Line != "Selected 6 documents: 3 by title, 3 by what they say. 1 scan could be matched by title only." ||
		events[len(events)-1].Rounds != 1 {
		t.Fatalf("the set was not selected in one round: %+v", view)
	}
	// The ids are the listed rows, in the same order, as full ids.
	var selected []string
	for _, id := range view.View.IDs {
		if len(id) != 36 {
			t.Fatalf("not a full id: %q", id)
		}
		selected = append(selected, id[:8])
	}
	if len(listed) != 6 || strings.Join(selected, " ") != strings.Join(listed, " ") {
		t.Errorf("selected %v, listed %v", selected, listed)
	}
}

func TestDocumentsAboutNoMatch(t *testing.T) {
	ta := newToolApp(t)
	seedBasement(t, ta)
	ta.run(t, "Anything about asbestos?", tool("documents", map[string]any{"about": "asbestos", "select": true}))
	got := ta.results[0]
	for _, want := range []string{"No document mentions asbestos. Tried in titles, text and labels: asbestos, hazardous materials, hazmat, acm,",
		"Searched the titles of 8 filed documents", "1 has no text (scans)"} {
		if !strings.Contains(got, want) {
			t.Errorf("no-match result lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "select_ids") {
		t.Errorf("nothing to select, yet:\n%s", got)
	}
	// Only filler is not a subject.
	ta.run(t, "Select the documents", tool("documents", map[string]any{"about": "the documents"}))
	if !strings.Contains(ta.results[0], "Say what the documents are about") {
		t.Errorf("filler only:\n%s", ta.results[0])
	}
}

func TestDocumentsAboutCap(t *testing.T) {
	ta := newToolApp(t)
	docs := make([]chatfix.Document, 205)
	for i := range docs {
		docs[i] = chatfix.Document{Title: fmt.Sprintf("Basement sheet %03d", i), Kind: "drawing"}
	}
	if _, err := chatfix.SeedDocuments(context.Background(), ta.a.store, ta.m.orgID, ta.project, docs); err != nil {
		t.Fatal(err)
	}
	ta.reply = func([]string) llm.Turn { return llm.Turn{Text: "ok"} }
	ta.run(t, "Which drawings are about the basement?", tool("documents", map[string]any{"about": "basement"}))
	if got := ta.results[0]; !strings.Contains(got, "[Capped: the strongest 200 of 205 matching documents are listed.") {
		t.Errorf("cap:\n%s", got[:min(len(got), 1500)])
	}
	view := viewOf(ta.run(t, "Select the basement drawings", tool("documents", map[string]any{"about": "basement", "select": true})))
	if view == nil || len(view.View.IDs) != 200 ||
		view.View.Line != "Selected the first 200 of 205 matching documents: 200 by title, 0 by what they say. 205 scans could be matched by title only." {
		t.Errorf("cap, selected: %+v", view)
	}
}
