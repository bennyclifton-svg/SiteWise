package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"sitewise/internal/agent"
	"sitewise/internal/store"
)

// Memory of ids, and undo (docs/plans/2026-11-18-chat-agency.md, Task 5).

// idList is a card's or a move's ids for the history: fifty, then how many
// more, so a long selection does not crowd the prompt.
func idList(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	shown := ids
	if len(shown) > 50 {
		shown = shown[:50]
	}
	short := make([]string, len(shown))
	for i, id := range shown {
		short[i] = shortID(id)
	}
	out := " " + strings.Join(short, " ")
	if more := len(ids) - len(shown); more > 0 {
		out += fmt.Sprintf(" and %d more", more)
	}
	return out
}

// viewSaid is a screen move as the history replays it, with its ids.
func viewSaid(v agent.View) string {
	t := viewTables[v.Table]
	switch {
	case v.Dialog != "":
		return "[Opened " + viewDialogs[v.Dialog] + ": package " + shortID(v.Fields["package"]) + ", column " + shortID(v.Fields["column"]) + "]"
	case v.Do == "select":
		return "[Selected " + t.count(len(v.IDs)) + filterWords(v) + ":" + idList(v.IDs) + "]"
	case v.Do == "only":
		return "[Showing only " + t.count(len(v.IDs)) + filterWords(v) + ":" + idList(v.IDs) + "]"
	case v.Do == "open" && len(v.IDs) > 0:
		return "[Opened " + t.one + " " + shortID(v.IDs[0]) + "]"
	case v.Do == "clear":
		return "[Cleared the " + t.many + " back to all rows]"
	}
	return "[Showed the " + t.many + "]"
}

// filterWords are a named filter's words after the count, or "".
func filterWords(v agent.View) string {
	if v.Shown == "" {
		return ""
	}
	return " " + v.Shown
}

// cardRefs are the row ids a batch's fields name: any string, or string in
// a list, that is a full id or a short one. A label that happens to look
// like one costs a word in the history, nothing more.
func cardRefs(calls []opCall) []string {
	var out []string
	add := func(v any) {
		s, ok := v.(string)
		if !ok {
			return
		}
		s = strings.ToLower(strings.TrimSpace(s))
		if (uuidPattern.MatchString(s) || shortIDPattern.MatchString(s)) && !containsID(out, shortID(s)) {
			out = append(out, shortID(s))
		}
	}
	for _, c := range calls {
		var m map[string]any
		_ = json.Unmarshal(c.Fields, &m)
		for _, v := range m {
			if list, ok := v.([]any); ok {
				for _, x := range list {
					add(x)
				}
				continue
			}
			add(v)
		}
	}
	return out
}

var shortIDPattern = regexp.MustCompile(`^[0-9a-f]{8}$`)

// undoPhrases are answered by Go with no model call (agency plan, Task 5).
var undoPhrases = map[string]bool{"undo": true, "undo that": true, "undo the last change": true}

// undoLast undoes the newest change this user made in this conversation
// that is not undone yet, with the same store undo the card's button uses.
// It never reaches the model. A change another screen has since altered is
// left as it is.
func undoLast(ctx context.Context, d Deps, who opActor, project string, earlier []store.ChatTurn, emit func(agent.Event)) {
	answer := func(text string) { emit(agent.Event{Type: "answer", Answer: &agent.Answer{Text: text}}) }
	undone := map[string]bool{}
	for i := len(earlier) - 1; i >= 0; i-- {
		var stored struct {
			Events []agent.Event `json:"events"`
		}
		_ = json.Unmarshal(earlier[i].Answer, &stored)
		for j := len(stored.Events) - 1; j >= 0; j-- {
			e := stored.Events[j]
			if e.Type == "undone" {
				undone[e.Text] = true
			}
			if e.Type != "change" || e.Card == nil {
				continue
			}
			key, done := undoKey(*e.Card), ""
			if key == "" || undone[key] {
				continue
			}
			switch e.Card.Undo {
			case "change":
				done = undoChangeCard(ctx, d, who, project, *e.Card)
			case "program":
				done = undoProgramCard(ctx, d, who, project, *e.Card)
			}
			if done == "already" {
				continue
			}
			if done == "" {
				if d.Broker != nil {
					d.Broker.Wake(who.OrgID)
				}
				emit(agent.Event{Type: "undone", Text: key})
				answer("Undone: " + strings.Join(e.Card.Lines, " "))
				return
			}
			answer(done)
			return
		}
	}
	answer("There is nothing to undo in this conversation.")
}

// undoKey names a card's change for the strip and for later undo turns.
func undoKey(c agent.Card) string {
	switch {
	case c.Undo == "change" && c.ChangeID != "":
		return c.ChangeID
	case c.Undo == "program" && c.ProgramVersion > 0:
		return fmt.Sprintf("program:%d", c.ProgramVersion)
	case c.Undo == "program":
		return "program"
	}
	return ""
}

const cantUndo = "Can't undo: this has changed since."

// undoChangeCard returns "" when undone, "already" when it was undone
// before (the next older change is tried), or the answer to give.
func undoChangeCard(ctx context.Context, d Deps, who opActor, project string, c agent.Card) string {
	_, err := d.Store.UndoChange(ctx, who.OrgID, project, who.UserID, c.ChangeID)
	switch {
	case err == nil:
		return ""
	case errors.Is(err, store.ErrChangeUndone), errors.Is(err, store.ErrNotFound):
		return "already"
	case errors.Is(err, store.ErrChangedSince):
		return cantUndo
	}
	d.Log.Printf("chat: undo failed: %v", err)
	return "Undo didn't go through. Try again."
}

// undoProgramCard undoes a Program change only while the Program is still
// at the version that change left: Program's own undo takes the user's
// latest batch, which must be this one.
func undoProgramCard(ctx context.Context, d Deps, who opActor, project string, c agent.Card) string {
	if c.ProgramVersion == 0 {
		return "Use Undo on that change's card; chat can't tell whether the Program has changed since."
	}
	st, err := d.Store.LoadProgram(ctx, who.OrgID, project)
	if err != nil {
		return cantUndo
	}
	if st.Program.Version != c.ProgramVersion {
		return cantUndo
	}
	res, err := d.Store.UndoProgram(ctx, who.OrgID, project, who.UserID, c.ProgramVersion)
	if err != nil {
		var bad *store.ProgramOpError
		if errors.As(err, &bad) || errors.Is(err, store.ErrVersionConflict) {
			return cantUndo
		}
		d.Log.Printf("chat: program undo failed: %v", err)
		return "Undo didn't go through. Try again."
	}
	if d.Broker != nil {
		payload, _ := json.Marshal(map[string]any{"project_id": res.Program.ProjectID, "program_id": res.Program.ID, "version": res.Program.Version})
		if _, err := d.Broker.Commit(ctx, who.OrgID, eventProgramChanged, "", string(payload)); err != nil {
			d.Log.Printf("program event: %v", err)
		}
	}
	return ""
}
