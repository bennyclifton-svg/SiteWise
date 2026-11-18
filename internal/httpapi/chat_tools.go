package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"sitewise/internal/agent"
	"sitewise/internal/figures"
	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// chatBox is the agent's toolbox for one turn. The org, the user and the
// project come from the session and the route; no tool takes them from its
// input (plan, "Tools").
type chatBox struct {
	d       Deps
	who     opActor
	project string
	turn    string
	// screen is the selection the strip sent with this turn, checked.
	screen *screenSel
	tags   atomic.Int64
}

var readAreas = []string{"cost_plan", "profile", "accommodation", "program", "procurement", "implications", "delivery", "gaps", "invoices", "specifications", "attention", "report"}

func object(required []string, props map[string]any) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false, "required": required, "properties": props}
}

func str(desc string, enum ...string) map[string]any {
	m := map[string]any{"type": "string", "description": desc}
	if len(enum) > 0 {
		m["enum"] = enum
	}
	return m
}

// chatTools is the fixed tool list. It changes only with a deploy, so the
// prompt prefix stays cached.
func chatTools() []llm.Tool {
	return []llm.Tool{
		{Name: "read", Description: "Read one area of this project as SiteWise holds it, with ids. Figures carry a tag like [f3].",
			Schema: object([]string{"area"}, map[string]any{
				"area": str("the area", readAreas...),
				"part": str("optional: a package id for its tender lines (procurement)"),
			})},
		{Name: "find", Description: "Find firms, packages, cost lines, Program rows and documents by name.",
			Schema: object([]string{"words"}, map[string]any{"words": str("words from the name")})},
		{Name: "apply", Description: "Apply operations to one area. All apply or none. Set final true when the operations are everything the PM asked for and you have nothing to add: the turn then ends and SiteWise shows what changed.",
			Schema: object([]string{"area", "ops"}, map[string]any{
				"area": str("the area the operations belong to", opAreas...),
				"ops": map[string]any{"type": "array", "items": map[string]any{"type": "object", "required": []string{"op", "fields"},
					"properties": map[string]any{"op": str("the operation name"), "fields": map[string]any{"type": "object"}}}},
				"final": map[string]any{"type": "boolean"},
			})},
		{Name: "calc", Description: "Arithmetic done by SiteWise. Use it for every sum, difference, percentage, GST conversion and date shift; never work one out yourself. The result carries a tag.",
			Schema: object([]string{"op"}, map[string]any{
				"op":     str("sum, difference (a minus b), percentage (a as a share of b), percent_of (a percent of b), ratio (a over b), gst_ex (take GST out of a), gst_inc (add GST to a), working_days (a duration to working days), shift_date (a date moved by working days)", "sum", "difference", "percentage", "percent_of", "ratio", "gst_ex", "gst_inc", "working_days", "shift_date"),
				"values": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "the numbers, as read or as the PM typed them"},
				"date":   str("shift_date: the date, YYYY-MM-DD"),
				"span":   str("working_days and shift_date: the duration, e.g. \"2 weeks\""),
			})},
		{Name: "documents", Description: "List the project's filed documents: title, kind, discipline, number, revision, pages, whether current or superseded. With about, SiteWise finds the documents about a subject by title, text and building-system labels, strongest first, each with why it is included; superseded revisions are left out unless superseded is true.",
			Schema: object(nil, map[string]any{
				"words":      str("words in the title"),
				"kind":       str("filing kind, e.g. specification, report, drawing, contract, correspondence"),
				"discipline": str("discipline, e.g. mechanical"),
				"about":      str("what the documents are about, e.g. the basement; separate alternatives with commas"),
				"select":     map[string]any{"type": "boolean", "description": "true to select the documents found on screen"},
				"superseded": map[string]any{"type": "boolean", "description": "true to include superseded revisions in an about search"},
			})},
		{Name: "passages", Description: "Search or read document text. With words it searches every filed document (or one): give alternatives separated by commas and each is tried. With a document id and pages it reads those pages. Each passage has an id like p4 to quote from.",
			Schema: object(nil, map[string]any{
				"words":    str("words to search for; separate alternatives with commas"),
				"document": str("a document id, to search or read only that document"),
				"pages":    str("a page range to read, e.g. 1-10"),
			})},
		{Name: "messages", Description: "The project's inbox: the latest messages, or one message's body by id.",
			Schema: object(nil, map[string]any{"id": str("a message id, to read its body")})},
		{Name: "view", Description: "Move the PM's screen: show a table, select rows, show only some rows, clear that, open a row, open the award dialog on one firm, or open the project report (with its issue form when the PM asks to issue it). It changes no record, so it is done at once, even after reading a document; it never presses Award, Send or Issue. Use ids from the outline, a read or the screen line. Set final true when moving the screen is all the PM asked for.",
			Schema: object([]string{"do"}, map[string]any{
				"do":     str("show, open (a row by its one id, or a dialog), select, only (hide the other rows) or clear (all rows back)", "show", "open", "select", "only", "clear"),
				"table":  str("the table", viewTableNames...),
				"ids":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "row ids, at most 200"},
				"filter": str("with select or only, in place of ids: a set SiteWise computes by rule. " + filterList() + ". Any other set: pick ids from a read"),
				"dialog": str("a dialog to open: award, or report (the project report)", "award", "report"),
				"fields": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "award: package (a package id) and column (the firm's column id). report: issue (yes opens its issue form)"},
				"final":  map[string]any{"type": "boolean"},
			})},
	}
}

func (b *chatBox) Tools() []llm.Tool       { return chatTools() }
func (b *chatBox) Writes(name string) bool { return name == "apply" }

func (b *chatBox) nextTag(prefix string) func() string {
	return func() string { return fmt.Sprintf("%s%d", prefix, b.tags.Add(1)) }
}

// Run runs one tool. Every read is timed against chat_tool_read.
func (b *chatBox) Run(ctx context.Context, st *agent.State, call llm.ToolCall) agent.Output {
	began := time.Now()
	var out agent.Output
	switch call.Name {
	case "apply":
		return b.apply(ctx, st, call.Input)
	case "read":
		out = b.read(ctx, call.Input)
	case "find":
		out = b.find(ctx, call.Input)
	case "calc":
		out = b.calc(ctx, st, call.Input)
	case "documents":
		out = b.documents(ctx, call.Input)
	case "passages":
		out = b.passages(ctx, st, call.Input)
	case "view":
		out = b.view(ctx, call.Input)
	case "messages":
		// Email intake is not built. The tool keeps its place in the list
		// so the prompt does not change when it is.
		out = agent.Output{Text: "This project has no inbox yet."}
	default:
		out = agent.Output{Text: "There is no tool named " + call.Name + ".", IsError: true}
	}
	if b.d.Speed != nil {
		b.d.Speed.Observe("chat_tool_read", time.Since(began))
	}
	return out
}

func toolError(format string, args ...any) agent.Output {
	return agent.Output{Text: fmt.Sprintf(format, args...), IsError: true}
}

func (b *chatBox) read(ctx context.Context, input json.RawMessage) agent.Output {
	var in struct {
		Area string `json:"area"`
		Part string `json:"part"`
	}
	_ = json.Unmarshal(input, &in)
	r := &reader{facts: map[string]agent.Fact{}, next: b.nextTag("f")}
	var err error
	switch in.Area {
	case "cost_plan":
		err = readCostPlan(ctx, b.d, b.who, b.project, r)
	case "profile":
		err = readProfile(ctx, b.d, b.who, b.project, r)
	case "accommodation":
		err = chatAccommodation(ctx, b.d, b.project, r)
	case "program":
		err = readProgram(ctx, b.d, b.who, b.project, r, false)
	case "procurement":
		if in.Part != "" {
			err = readTender(ctx, b.d, b.who, b.project, in.Part, r)
		} else {
			err = readProcurement(ctx, b.d, b.who, b.project, r)
		}
	default:
		err = readMore(ctx, b.d, b.who, b.project, in.Area, r)
	}
	if err != nil {
		if e, ok := err.(*opError); ok {
			return toolError("%s", e.Message)
		}
		b.d.Log.Printf("chat read %s failed: %v", in.Area, err)
		return toolError("That could not be read.")
	}
	return agent.Output{Text: capped(r.b.String(), "Ask about one section or one package."), Facts: r.facts, Outside: r.outside}
}

// figureCheck refuses an operation that carries an amount, a date or a
// duration nobody gave: not in the PM's messages (as Go normalises them)
// and not in a tool result of this turn (review change 2).
func figureCheck(op operation, fields json.RawMessage, st *agent.State) *opError {
	if len(op.Figures) == 0 {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(fields, &m) != nil {
		return nil // the operation itself refuses a bad shape
	}
	names := make([]string, 0, len(op.Figures))
	for name := range op.Figures {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		raw, ok := m[name]
		if !ok || raw == nil {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(raw))
		if n, isNum := raw.(float64); isNum {
			value = strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", n), "0"), ".")
		}
		given := false
		switch op.Figures[name] {
		case "money":
			a, ok := figures.Amount(value)
			given = ok && st.Book.HasNumber(a)
			if !ok {
				continue // not an amount at all: the operation says so
			}
		case "duration":
			n, _, ok := figures.Duration(value)
			if !ok {
				continue
			}
			given = st.Book.HasNumber(fmt.Sprint(n)) || strings.Contains(strings.ToLower(value), "fortnight")
		case "date":
			given = st.Book.HasDate(value, st.Today)
		case "number":
			given = st.Book.HasNumber(value)
		default:
			given = st.Book.HasNumbersOf(value)
		}
		if !given {
			return &opError{Field: name, Message: fmt.Sprintf("Say the figure. %q is not in the PM's message or in anything read this turn, so it was not used. Ask the PM for it, or leave it out.", value)}
		}
	}
	return nil
}

// apply is the one tool that changes records. It applies the batch at once
// with Undo, or prepares it for the PM's click when the turn has read text
// from outside or an operation is the PM's to confirm.
func (b *chatBox) apply(ctx context.Context, st *agent.State, input json.RawMessage) agent.Output {
	var in struct {
		opBatch
		Final bool `json:"final"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return toolError("The apply input is not valid.")
	}
	in.Via, in.TurnID = "chat", b.turn
	refused := func(e *opError) agent.Output {
		where := "The batch"
		if e.Index >= 0 && e.Index < len(in.Calls) {
			where = fmt.Sprintf("Operation %d (%s)", e.Index+1, in.Calls[e.Index].Op)
		}
		if e.Field != "" {
			where += ", " + e.Field
		}
		return agent.Output{Text: "Refused, nothing was changed. " + where + ": " + e.Message, IsError: true, Refused: true}
	}
	if why := tooManySelected(b.screen, in.Area, st.Asked); why != "" {
		return agent.Output{Text: "Refused, nothing was changed. " + why, IsError: true, Refused: true}
	}
	in.opBatch.Clicked = true // check the shape as if clicked; clicks are judged below
	ops, bad := checkBatch(in.opBatch)
	in.opBatch.Clicked = false
	if bad != nil {
		return refused(bad)
	}
	for i, op := range ops {
		if bad := figureCheck(op, in.Calls[i].Fields, st); bad != nil {
			bad.Index = i
			return refused(bad)
		}
	}
	// A prepared read checks its ids and whether it is already under way
	// before any card exists (ops_reads.go).
	words, bad := cardWords(ctx, b.d, b.who, b.project, ops, in.Calls)
	if bad != nil {
		return refused(bad)
	}
	why := ""
	if st.Outside {
		why = "This used text from " + st.Why + ", so it waits for your click."
	} else {
		done, bad := applyOps(ctx, b.d, b.who, b.project, in.opBatch)
		switch {
		case bad == nil:
			if b.d.Broker != nil {
				b.d.Broker.Wake(b.who.OrgID)
			}
			card := agent.Card{Kind: "change", Area: in.Area, ChangeID: done.ChangeID, Undo: done.Undo.Kind, Refs: cardRefs(in.Calls), ProgramVersion: done.ProgramVersion}
			for _, r := range done.Results {
				card.Lines = append(card.Lines, r.Summary)
			}
			return agent.Output{Text: "Applied. " + strings.Join(card.Lines, " "), Cards: []agent.Card{card}, Final: in.Final}
		case !bad.NeedsClick:
			return refused(bad)
		}
		why = bad.Message
	}
	raw, _ := json.Marshal(in.Calls)
	lines := describeOps(ops, in.Calls)
	for i, w := range words {
		if w != "" {
			lines[i] = w
		}
	}
	id, err := b.d.Store.AddChatPrepared(ctx, b.who.OrgID, store.ChatPrepared{TurnID: b.turn, Area: in.Area, Ops: raw, Summary: strings.Join(lines, "\n"), Why: why})
	if err != nil {
		b.d.Log.Printf("chat: storing a prepared change failed: %v", err)
		return toolError("The change could not be prepared.")
	}
	card := agent.Card{Kind: "prepared", Area: in.Area, Lines: lines, Prepared: id, Why: why}
	return agent.Output{Text: "Prepared, not applied: it waits for the PM's click. " + why, Cards: []agent.Card{card}, Final: in.Final}
}

// describeOps says in plain words what a prepared batch would do, from the
// operations' own descriptions and the fields as sent. Nothing is run.
func describeOps(ops []operation, calls []opCall) []string {
	var out []string
	for i, op := range ops {
		var m map[string]any
		_ = json.Unmarshal(calls[i].Fields, &m)
		names := make([]string, 0, len(m))
		for name := range m {
			names = append(names, name)
		}
		sort.Strings(names)
		var parts []string
		for _, name := range names {
			v := m[name]
			if s, ok := v.(string); ok && uuidPattern.MatchString(s) {
				continue // an id says nothing to a reader
			}
			raw, _ := json.Marshal(v)
			parts = append(parts, name+" "+strings.Trim(string(raw), `"`))
		}
		line := strings.TrimSuffix(op.About, ".")
		if i := strings.Index(line, ". "); i > 0 {
			line = line[:i]
		}
		if len(parts) > 0 {
			line += ": " + strings.Join(parts, ", ")
		}
		out = append(out, strings.ToUpper(line[:1])+line[1:]+".")
	}
	return out
}
