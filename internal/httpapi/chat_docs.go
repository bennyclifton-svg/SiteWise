package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"sitewise/internal/agent"
	"sitewise/internal/store"
)

// The chat's document tools (plan Task 6 and review change 5). They are
// plain Go over the store. Whatever they return came from outside: it is
// material, and a turn that read it prepares its changes for a click.

// chatWords is knowledge/chat/words.yaml: construction synonyms a search is
// widened with, and the words that name each building system. Draft.
type chatWords struct {
	Synonyms []struct {
		Word string   `yaml:"word"`
		Also []string `yaml:"also"`
	} `yaml:"synonyms"`
}

// systemTerms is knowledge/works/system_terms.yaml: a subject to its systems.
type systemTerms struct {
	Terms []struct {
		Term string   `yaml:"term"`
		IDs  []string `yaml:"ids"`
	} `yaml:"terms"`
}

// LoadChatWords reads the search vocabulary. A missing file is no vocabulary.
func LoadChatWords(knowledgeDir string) (map[string][]string, map[string][]string, error) {
	synonyms, systems := map[string][]string{}, map[string][]string{}
	var w chatWords
	if raw, err := os.ReadFile(filepath.Join(knowledgeDir, "chat", "words.yaml")); err == nil {
		if err := yaml.Unmarshal(raw, &w); err != nil {
			return nil, nil, err
		}
	}
	for _, s := range w.Synonyms {
		group := append([]string{s.Word}, s.Also...)
		for _, word := range group {
			synonyms[strings.ToLower(word)] = group
		}
	}
	var t systemTerms
	if raw, err := os.ReadFile(filepath.Join(knowledgeDir, "works", "system_terms.yaml")); err == nil {
		if err := yaml.Unmarshal(raw, &t); err != nil {
			return nil, nil, err
		}
	}
	for _, term := range t.Terms {
		systems[strings.ToLower(term.Term)] = term.IDs
	}
	return synonyms, systems, nil
}

type docInfo struct {
	ID, Title, Kind, Discipline, Number, Revision string
	Pages, Passages                               int
	NoText, Reading                               bool
	SupersededBy                                  string // the newer document's title, or ""
	Firm                                          store.DocumentFirm
}

func (d docInfo) name() string {
	n := d.Title
	if d.Revision != "" && !strings.Contains(strings.ToLower(n), "rev") {
		n += " rev " + d.Revision
	}
	return n
}

// status is what an answer must say about the document it quotes.
func (d docInfo) status() string {
	var parts []string
	if d.SupersededBy != "" {
		parts = append(parts, "superseded by "+d.SupersededBy)
	}
	if d.Firm.Firm != "" {
		s := d.Firm.Firm + "'s submission"
		if d.Firm.Package != "" {
			s += " for " + d.Firm.Package
		}
		if d.Firm.Status == store.SubmissionAwarded {
			s += ", awarded"
		} else {
			s += ", not awarded"
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, "; ")
}

// projectDocs lists the project's filed documents, current ones first.
func (b *chatBox) projectDocs(ctx context.Context) ([]docInfo, error) {
	views, err := b.d.Store.ProjectDocumentViews(ctx, b.who.OrgID, b.project)
	if err != nil {
		return nil, err
	}
	counts, err := b.d.Store.ChatPassageCounts(ctx, b.who.OrgID, b.project)
	if err != nil {
		return nil, err
	}
	firms, err := b.d.Store.DocumentFirms(ctx, b.who.OrgID, b.project)
	if err != nil {
		return nil, err
	}
	out := make([]docInfo, 0, len(views))
	byID := map[string]int{}
	for _, v := range views {
		d := docInfo{ID: v.ID, Title: v.Filename, Number: v.Number, Revision: v.Revision, Pages: int(v.TextPages), Passages: counts[v.ID], Firm: firms[v.ID]}
		for _, f := range v.Fields {
			switch f.Field {
			case "title":
				if strings.TrimSpace(f.Value) != "" {
					d.Title = f.Value
				}
			case "kind":
				d.Kind = f.Value
			case "discipline":
				d.Discipline = f.Value
			}
		}
		d.Reading = v.Status == "pending" || v.Status == "received" || v.TextStatus == "pending"
		d.NoText = d.Passages == 0 && !d.Reading
		byID[v.ID] = len(out)
		out = append(out, d)
	}
	for _, v := range views {
		if i, ok := byID[v.SupersedesID]; ok && v.SupersedesID != "" {
			out[i].SupersededBy = out[byID[v.ID]].name()
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].SupersededBy == "" && out[j].SupersededBy != "" })
	return out, nil
}

func (b *chatBox) documents(ctx context.Context, input json.RawMessage) agent.Output {
	var in struct {
		Words, Kind, Discipline string
	}
	_ = json.Unmarshal(input, &in)
	docs, err := b.projectDocs(ctx)
	if err != nil {
		b.d.Log.Printf("chat documents failed: %v", err)
		return toolError("The documents could not be listed.")
	}
	var lines []string
	for _, d := range docs {
		if !hasWords(d.Title+" "+d.Number, in.Words) || !matches(d.Kind, in.Kind) || !matches(d.Discipline, in.Discipline) {
			continue
		}
		lines = append(lines, docLine(d))
	}
	if len(lines) == 0 {
		return agent.Output{Text: fmt.Sprintf("No filed document matches. %d documents are filed in this project.", len(docs))}
	}
	text := "Filed documents (id | title | kind | discipline | number, revision | pages | status). Current revisions first.\n" + strings.Join(lines, "\n") + "\n"
	return agent.Output{Text: capped(text, "Narrow the list with words, kind or discipline."), Outside: "the titles of filed documents"}
}

func docLine(d docInfo) string {
	parts := []string{shortID(d.ID), d.name(), orDash(d.Kind), orDash(d.Discipline), orDash(strings.TrimSpace(d.Number + " " + d.Revision))}
	switch {
	case d.Reading:
		parts = append(parts, "still being read")
	case d.NoText:
		parts = append(parts, "no searchable text (a scan)")
	default:
		parts = append(parts, fmt.Sprintf("%d pages", max(d.Pages, 1)))
	}
	status := d.status()
	if d.SupersededBy == "" {
		status = strings.TrimPrefix("current; "+status, "; ")
		status = strings.TrimSuffix(status, "; ")
	}
	return strings.Join(append(parts, status), " | ")
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func matches(value, want string) bool {
	return want == "" || strings.Contains(strings.ToLower(value), strings.ToLower(strings.TrimSpace(want)))
}

func hasWords(text, words string) bool {
	text = strings.ToLower(text)
	for _, w := range strings.Fields(strings.ToLower(words)) {
		if !strings.Contains(text, strings.Trim(w, ",.")) {
			return false
		}
	}
	return true
}

const (
	maxPassages     = 14   // passages one call returns
	maxPassageChars = 1200 // of each passage shown to the model
	maxReadChars    = 22000
)

// passages searches or reads document text.
func (b *chatBox) passages(ctx context.Context, st *agent.State, input json.RawMessage) agent.Output {
	var in struct {
		Words, Document, Pages string
	}
	_ = json.Unmarshal(input, &in)
	docs, err := b.projectDocs(ctx)
	if err != nil {
		b.d.Log.Printf("chat passages failed: %v", err)
		return toolError("The documents could not be read.")
	}
	byID := map[string]docInfo{}
	for _, d := range docs {
		byID[d.ID] = d
	}
	var one *docInfo
	if strings.TrimSpace(in.Document) != "" {
		d, err := byRef(docs, func(d docInfo) string { return d.ID }, in.Document, "document")
		if err != nil {
			return toolError("%s", err.Error())
		}
		one = &d
		if d.NoText {
			return agent.Output{Text: d.name() + " has no searchable text (it is a scan), so it cannot be read here.", Outside: d.name()}
		}
	}
	if strings.TrimSpace(in.Words) == "" {
		if one == nil {
			return toolError("Give words to search for, or a document id and pages to read.")
		}
		return b.readDocument(ctx, *one, in.Pages)
	}
	// Alternatives are searched with "or": the index wants every word of a
	// query in one passage, so "tree arborist vegetation" would find
	// nothing (review change 5).
	alts := b.alternatives(in.Words)
	quoted := make([]string, len(alts))
	for i, a := range alts {
		quoted[i] = a
		if strings.Contains(a, " ") {
			quoted[i] = `"` + a + `"`
		}
	}
	docID := ""
	if one != nil {
		docID = one.ID
	}
	found, err := b.d.Store.ChatSearchPassages(ctx, b.who.OrgID, b.project, strings.Join(quoted, " or "), docID, 60)
	if err != nil {
		b.d.Log.Printf("chat passages failed: %v", err)
		return toolError("The search failed.")
	}
	coverage := coverageLine(docs, one, alts)
	// The PM named a document or a firm: the answer is from that document
	// only, so another firm's proposal or another report is never quoted
	// beside it. A superseded revision is left out unless the PM asked
	// about it. Go does this; it is not left to the model.
	var kept []string
	if one == nil {
		named := namedDocs(st.Asked, docs)
		old := strings.Contains(strings.ToLower(st.Asked), "supersed") || strings.Contains(strings.ToLower(st.Asked), "previous rev") || strings.Contains(strings.ToLower(st.Asked), "earlier rev")
		var keep []store.ChatPassage
		dropped := map[string]bool{}
		for _, p := range found {
			doc := byID[p.DocumentID]
			switch {
			case len(named) > 0 && !named[p.DocumentID]:
			case len(named) == 0 && doc.SupersededBy != "" && !old:
				dropped[doc.name()+" (superseded by "+doc.SupersededBy+")"] = true
			default:
				keep = append(keep, p)
			}
		}
		found = keep
		if len(named) > 0 {
			for _, d := range docs {
				if named[d.ID] {
					kept = append(kept, d.name())
				}
			}
			coverage = fmt.Sprintf("Searched only %s, which the question names, for: %s.", strings.Join(kept, ", "), strings.Join(alts, ", "))
		}
		for name := range dropped {
			coverage += " Left out: " + name + "."
		}
	}
	if len(found) > 0 {
		coverage += " Only the matching passages were read, not the whole of each document."
	}
	out := agent.Output{Coverage: coverage, Outside: "filed documents"}
	var text strings.Builder
	if len(found) == 0 {
		fmt.Fprintf(&text, "No mention found for: %s.\n", strings.Join(alts, ", "))
	} else {
		// Documents with their hit counts first, current ones first.
		hits := map[string]int{}
		var order []string
		for _, p := range found {
			if _, seen := hits[p.DocumentID]; !seen {
				order = append(order, p.DocumentID)
			}
			hits[p.DocumentID] = p.Hits
		}
		sort.SliceStable(order, func(i, j int) bool { return byID[order[i]].SupersededBy == "" && byID[order[j]].SupersededBy != "" })
		text.WriteString("Documents with matches (current first):\n")
		for _, id := range order {
			fmt.Fprintf(&text, "  %s | %d passages match\n", docLine(byID[id]), hits[id])
		}
		text.WriteString("Passages (quote exactly, then the tag):\n")
		shown := 0
		for _, id := range order {
			for _, p := range found {
				if p.DocumentID != id || shown == maxPassages {
					continue
				}
				shown++
				b.writePassage(&text, &out, byID[id], p)
			}
		}
		if shown < len(found) {
			fmt.Fprintf(&text, "[Capped: %d of %d matching passages are shown. Search one document, or narrower words, for the rest.]\n", shown, len(found))
		}
	}
	b.systemRoute(ctx, &text, &out, byID, alts)
	text.WriteString(coverage + "\n")
	out.Text = text.String()
	return out
}

// namedDocs finds the documents the PM's message names: by a title's own
// words ("the arborist report"), or by the firm whose submission it is
// ("Northline"). An empty result means the message names none.
func namedDocs(asked string, docs []docInfo) map[string]bool {
	asked = " " + strings.ToLower(asked) + " "
	named := map[string]bool{}
	for _, d := range docs {
		title := strings.ToLower(d.Title)
		if i := strings.Index(title, " rev "); i > 0 {
			title = title[:i]
		}
		if len(title) >= 6 && strings.Contains(asked, title) {
			named[d.ID] = true
		}
		if firm := strings.Fields(strings.ToLower(d.Firm.Firm)); len(firm) > 0 && len(firm[0]) >= 4 && strings.Contains(asked, firm[0]) {
			named[d.ID] = true
		}
		// A proposal filed but not yet tied to a firm still carries the
		// firm's name in its title ("Northline fee proposal").
		if first := strings.Fields(title); len(first) > 1 && len(first[0]) >= 5 && (d.Kind == "commercial" || strings.Contains(title, "proposal") || strings.Contains(title, "tender")) &&
			strings.Contains(asked, " "+first[0]) {
			named[d.ID] = true
		}
	}
	// A superseded revision of a named document gives way to the current one.
	current := false
	for _, d := range docs {
		current = current || (named[d.ID] && d.SupersededBy == "")
	}
	if current && !strings.Contains(asked, "supersed") {
		for _, d := range docs {
			if named[d.ID] && d.SupersededBy != "" {
				delete(named, d.ID)
			}
		}
	}
	return named
}

// alternatives splits the model's words into search alternatives and
// widens them with the construction synonyms.
func (b *chatBox) alternatives(words string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(w string) {
		w = strings.ToLower(strings.TrimSpace(w))
		if w != "" && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	for _, alt := range strings.FieldsFunc(words, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		alt = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(alt), "or "), " or"))
		add(alt)
		if b.d.Chat != nil {
			for _, also := range b.d.Chat.Synonyms[strings.ToLower(alt)] {
				add(also)
			}
		}
	}
	return out
}

func (b *chatBox) writePassage(text *strings.Builder, out *agent.Output, d docInfo, p store.ChatPassage) {
	tag := b.nextTag("p")()
	body := p.Body
	if len(body) > maxPassageChars {
		cut := maxPassageChars
		for cut > 0 && body[cut]&0xC0 == 0x80 {
			cut--
		}
		body = body[:cut] + " …[cut]"
	}
	where := d.name()
	if p.Page > 0 {
		where += ", page " + strconv.Itoa(p.Page)
	}
	if p.Section != "" {
		where += ", " + p.Section
	}
	if s := d.status(); s != "" {
		where += " (" + s + ")"
	}
	fmt.Fprintf(text, "[%s] %s:\n%s\n", tag, where, body)
	out.Passages = append(out.Passages, agent.Passage{ID: tag, Document: d.name(), DocID: d.ID, Page: p.Page, Text: p.Body, Note: d.status()})
}

// coverageLine is Go's own statement of what a search covered. It goes on
// every document answer, so "not found" is never read as "not there".
func coverageLine(docs []docInfo, one *docInfo, alts []string) string {
	if one != nil {
		return fmt.Sprintf("Searched %s only, for: %s.", one.name(), strings.Join(alts, ", "))
	}
	var noText []string
	reading := 0
	for _, d := range docs {
		switch {
		case d.Reading:
			reading++
		case d.NoText:
			noText = append(noText, d.name())
		}
	}
	line := fmt.Sprintf("Searched %d filed documents for: %s.", len(docs)-len(noText)-reading, strings.Join(alts, ", "))
	if len(noText) > 0 {
		line += fmt.Sprintf(" %d could not be searched because they have no text: %s.", len(noText), strings.Join(noText, ", "))
	}
	if reading > 0 {
		line += fmt.Sprintf(" %d are still being read.", reading)
	}
	return line + " Emails are not searched: this project has no inbox yet."
}

// readDocument reads one document by page range, up to a cap, and says how
// much of it that was (review change 6).
func (b *chatBox) readDocument(ctx context.Context, d docInfo, pages string) agent.Output {
	from, to := 0, 0
	if pages != "" {
		a, z, ok := strings.Cut(strings.ReplaceAll(pages, " ", ""), "-")
		from, _ = strconv.Atoi(a)
		to = from
		if ok {
			to, _ = strconv.Atoi(z)
		}
		if from < 1 || to < from {
			return toolError("Give pages as a range, for example 1-10.")
		}
	}
	found, err := b.d.Store.ChatDocumentPassages(ctx, b.who.OrgID, b.project, d.ID, from, to)
	if err != nil {
		b.d.Log.Printf("chat passages failed: %v", err)
		return toolError("The document could not be read.")
	}
	out := agent.Output{Outside: d.name()}
	var text strings.Builder
	last, shown := 0, 0
	for _, p := range found {
		if text.Len() > maxReadChars {
			break
		}
		shown++
		if p.Page > last {
			last = p.Page
		}
		b.writePassage(&text, &out, d, p)
	}
	total := max(d.Pages, last)
	switch {
	case shown < len(found) && last > 0:
		out.Coverage = fmt.Sprintf("Read %s to page %d of %d; the rest was not read.", d.name(), last, total)
		fmt.Fprintf(&text, "[Capped at page %d of %d. Read on with pages %d-%d. Until you have, say which pages you read and that any list is incomplete.]\n", last, total, last, total)
	case shown < len(found):
		out.Coverage = fmt.Sprintf("Read the first %d of %d passages of %s; the rest was not read.", shown, len(found), d.name())
		fmt.Fprintf(&text, "[Capped: %d of %d passages. Say that any list is incomplete.]\n", shown, len(found))
	case from > 0:
		out.Coverage = fmt.Sprintf("Read pages %d to %d of %s (%d pages).", from, to, d.name(), total)
	default:
		out.Coverage = fmt.Sprintf("Read all of %s.", d.name())
	}
	text.WriteString(out.Coverage + "\n")
	out.Text = text.String()
	return out
}

// systemRoute is the route through the building model: when the subject
// names a building system, Go adds the passages the profile's reading
// labelled with it, and the open implications for it.
func (b *chatBox) systemRoute(ctx context.Context, text *strings.Builder, out *agent.Output, docs map[string]docInfo, alts []string) {
	if b.d.Chat == nil {
		return
	}
	var systems []string
	for _, a := range alts {
		systems = append(systems, b.d.Chat.SystemTerms[a]...)
	}
	if len(systems) == 0 {
		return
	}
	labelled, err := b.d.Store.ChatSystemPassages(ctx, b.who.OrgID, b.project, systems, 6)
	if err != nil {
		return
	}
	fmt.Fprintf(text, "Building model: the subject matches system %s. Labels cover only documents the profile has read.\n", strings.Join(systems, ", "))
	for _, p := range labelled {
		if _, ok := docs[p.DocumentID]; ok {
			b.writePassage(text, out, docs[p.DocumentID], p)
		}
	}
	under := func(id string) bool {
		for _, s := range systems {
			if id != "" && (id == s || strings.HasPrefix(id, s+".") || strings.HasPrefix(s, id+".")) {
				return true
			}
		}
		return false
	}
	// Scope status: what the profile holds for the system, in or out.
	if view, err := b.d.Store.ReadProfile(ctx, b.who.OrgID, b.project, b.d.ProfileReading.Kinds()); err == nil {
		for _, row := range view.Rows {
			if leaf, ok := strings.CutPrefix(row.Key, "scope."); ok && under(leaf) && row.Value != "" {
				fmt.Fprintf(text, "  Scope: %s is %s (%s)\n", leaf, row.Value, orDash(row.Origin))
			}
		}
	}
	if works, err := b.d.Store.ReadWorks(ctx, b.who.OrgID, b.project); err == nil {
		for _, w := range works {
			if under(w.SystemID) {
				fmt.Fprintf(text, "  Work item: %s (%s, %s)\n", w.Title, w.SystemID, w.Action)
			}
		}
	}
	if st, err := b.d.Store.LoadProgram(ctx, b.who.OrgID, b.project); err == nil {
		for _, act := range st.Activities {
			for _, s := range act.Systems {
				if under(s) {
					fmt.Fprintf(text, "  Program row: %s\n", act.Label)
					break
				}
			}
		}
	}
	items, err := b.d.Store.ReadProposals(ctx, b.who.OrgID, b.project)
	if err != nil {
		return
	}
	for _, it := range items {
		for _, s := range systems {
			if it.Decision == nil && (it.TargetSystemID == s || strings.HasPrefix(it.TargetSystemID, s+".") || strings.HasPrefix(s, it.TargetSystemID+".")) && it.TargetSystemID != "" {
				fmt.Fprintf(text, "  Open implication (draft checklist): %s\n", it.Label)
				break
			}
		}
	}
}

// find looks names up across the project's records.
func (b *chatBox) find(ctx context.Context, input json.RawMessage) agent.Output {
	var in struct {
		Words string `json:"words"`
	}
	_ = json.Unmarshal(input, &in)
	if strings.TrimSpace(in.Words) == "" {
		return toolError("Give words from the name.")
	}
	var lines []string
	out := agent.Output{}
	if p, err := b.d.Store.ReadCostPlan(ctx, b.who.OrgID, b.project, ""); err == nil {
		for _, i := range p.Items {
			if hasWords(i.Label, in.Words) {
				lines = append(lines, "cost line "+shortID(i.ID)+" | "+i.Label)
			}
		}
	}
	if st, err := b.d.Store.LoadProgram(ctx, b.who.OrgID, b.project); err == nil {
		for _, a := range st.Activities {
			if hasWords(a.Label, in.Words) {
				lines = append(lines, "program "+a.Kind+" "+shortID(a.ID)+" | "+a.Label)
			}
		}
	}
	if pkgs, err := b.d.Store.ReadPackages(ctx, b.who.OrgID, b.project); err == nil {
		board, _, _ := b.d.Store.ProcurementBoard(ctx, b.who.OrgID, b.project)
		for _, p := range pkgs {
			if p.RetiredAt == nil && hasWords(p.Title, in.Words) {
				lines = append(lines, "package "+shortID(p.ID)+" | "+p.Title)
			}
			for _, c := range board[p.ID] {
				if hasWords(c.FirmName, in.Words) {
					col := ""
					if c.SubmissionID != "" {
						col = " | column " + shortID(c.SubmissionID)
					}
					lines = append(lines, "firm | "+c.FirmName+" | on package "+shortID(p.ID)+" "+p.Title+col)
				}
			}
		}
	}
	if docs, err := b.projectDocs(ctx); err == nil {
		for _, d := range docs {
			if hasWords(d.Title+" "+d.Number, in.Words) {
				lines = append(lines, "document "+docLine(d))
				out.Outside = "the titles of filed documents"
			}
		}
	}
	if len(lines) == 0 {
		out.Text = fmt.Sprintf("Nothing in this project is named %q.", in.Words)
		return out
	}
	out.Text = capped(strings.Join(lines, "\n")+"\n", "Use more words.")
	return out
}
