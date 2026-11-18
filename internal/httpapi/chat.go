package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"sitewise/internal/agent"
	"sitewise/internal/llm"
	"sitewise/internal/store"
)

// Chat is the chat strip's server side (docs/plans/2026-11-15-chat-agent.md).
// It is the one path that waits on the model; no screen read or write
// reaches it. Nil switches chat off for every org.
type Chat struct {
	Model agent.Model
	// On reports whether the model can be called at all (a key is set).
	On func() bool
	// Name is the model id, a config value.
	Name string
	// Prompt is the stable system text: the rules and the area guides. The
	// operation list is added from the registry.
	Prompt string
	// Synonyms widen a document search with the words a builder would use;
	// SystemTerms name the building systems a subject belongs to. Both are
	// draft knowledge (LoadChatWords).
	// Think turns the model's hidden thinking on (adaptive, low effort).
	Think       bool
	Synonyms    map[string][]string
	SystemTerms map[string][]string
	warmed      sync.Map // user id to the time of its last warming call
}

// LoadChatPrompt reads knowledge/chat: system.md, then each area guide in
// name order. The text is fixed between deploys so the prefix caches.
func LoadChatPrompt(dir string) (string, error) {
	base, err := os.ReadFile(filepath.Join(dir, "system.md"))
	if err != nil {
		return "", err
	}
	guides, err := filepath.Glob(filepath.Join(dir, "areas", "*.md"))
	if err != nil {
		return "", err
	}
	sort.Strings(guides)
	out := strings.TrimSpace(string(base))
	for _, g := range guides {
		raw, err := os.ReadFile(g)
		if err != nil {
			return "", err
		}
		out += "\n\n" + strings.TrimSpace(string(raw))
	}
	return strings.ReplaceAll(out, "\r\n", "\n"), nil
}

// chatSystem is the whole cached prefix: the prompt, then the operation
// list rendered from the registry, so the list the model reads is the list
// the server runs.
func chatSystem(prompt string) string {
	var b strings.Builder
	b.WriteString(prompt)
	b.WriteString("\n\n# Operations\n\nEach is sent in apply as {\"op\": name, \"fields\": {…}}. Ids are the short ids shown in the outline and in reads.\n")
	for _, op := range operations() {
		click := ""
		if op.Click {
			click = " (waits for the PM's click)"
		}
		fmt.Fprintf(&b, "\n- %s%s: %s Fields: %s.", op.Name, click, op.About, op.Fields)
	}
	b.WriteString("\n\nAn operation marked as waiting for the PM's click, or one that cannot be undone from chat, is still sent with apply: SiteWise prepares it. Awards are made in the award dialog on Procurement; view opens it on the firm and the PM presses Award.\n")
	b.WriteString("Not available from chat: uploading or re-reading a document, editing a report's wording, combining packages. Say \"" + declineWords + "\" and name the screen.\n")
	b.WriteString("Never from chat: sign-in, billing, team and invitations, deleting or archiving a project, issuing anything.\n")
	return b.String()
}

// declineWords are chat's fixed words for a request it cannot do yet. The
// prompt (knowledge/chat/system.md) tells the model to use exactly these
// words, and a turn whose answer carries them is stored as declined: the
// share of such turns is the measure of what chat cannot do (agency plan,
// Task 7). TestDeclineWordsAreThePrompts holds the two together.
const declineWords = "I can't do that from chat yet"

// declined reports whether an answer carries the fixed words, with either
// apostrophe and in any case.
func declined(answer string) bool {
	answer = strings.ToLower(strings.ReplaceAll(answer, "’", "'"))
	return strings.Contains(answer, strings.ToLower(declineWords))
}

// answerText is the checked answer among a turn's events, or "".
func answerText(events []agent.Event) string {
	for _, e := range events {
		if e.Type == "answer" && e.Answer != nil {
			return e.Answer.Text
		}
	}
	return ""
}

// movedScreen reports whether a turn moved the screen (chat_screen_command).
func movedScreen(events []agent.Event) bool {
	for _, e := range events {
		if e.Type == "view" {
			return true
		}
	}
	return false
}

// goTo answers a message that is exactly a known read with the screen, and
// no model call (speed rule 7). The list is short and exact on purpose.
var goTo = map[string]string{
	"show the cost plan": "costs", "open the cost plan": "costs", "open cost plan": "costs", "show cost plan": "costs",
	"show the program": "program", "open the program": "program", "open program": "program", "show program": "program",
	"open procurement": "procurement", "show procurement": "procurement", "open the profile": "profile", "show the profile": "profile",
	"open specifications": "specifications", "show the documents": "documents", "open documents": "documents",
	"open the register": "register", "show the register": "register", "open register": "register", "show register": "register",
}

type chatDone struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversation_id"`
	TurnID         string `json:"turn_id"`
	Rounds         int    `json:"rounds"`
	Millis         int64  `json:"millis"`
	FirstMillis    int64  `json:"first_millis"`
	InTokens       int64  `json:"in_tokens"`
	OutTokens      int64  `json:"out_tokens"`
	CacheRead      int64  `json:"cache_read_tokens"`
	CacheWrite     int64  `json:"cache_write_tokens"`
}

// chatUnavailable says why chat cannot run, or "" when it can. Nothing
// else on the page depends on it.
func chatUnavailable(ctx context.Context, d Deps, org string) string {
	if d.Chat == nil || d.Chat.Model == nil || (d.Chat.On != nil && !d.Chat.On()) {
		return "Chat is not set up on this server."
	}
	on, err := d.Store.ModelEnabled(ctx, org)
	if err != nil || !on {
		return "Chat is off for your organisation. The owner can switch it on under Account."
	}
	return ""
}

// postChat is POST /projects/{id}/chat {"conversation_id","text","screen"}.
// It answers with a stream of events: text, tool, change, prepared, view,
// answer, go, error and, last, done.
func postChat(w http.ResponseWriter, r *http.Request, d Deps) {
	began := time.Now()
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	var in struct {
		ConversationID string `json:"conversation_id"`
		Text           string `json:"text"`
		// Screen is what the strip says is on screen (agency plan, Task 1).
		Screen *screenIn `json:"screen"`
	}
	if readProposalJSON(w, r, d.MaxBodyBytes, &in) != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	in.Text = strings.TrimSpace(in.Text)
	if in.Text == "" || len(in.Text) > 4000 || (in.ConversationID != "" && !uuidPattern.MatchString(in.ConversationID)) {
		http.Error(w, "a message is 1 to 4,000 characters", http.StatusUnprocessableEntity)
		return
	}
	project := r.PathValue("id")
	ctx := r.Context()
	who := opActor{s.OrgID, s.UserID}
	// Earlier turns are read before this one is stored. Another user's
	// conversation reads as empty and is refused when the turn starts.
	var earlier []store.ChatTurn
	if in.ConversationID != "" {
		var err error
		if earlier, err = d.Store.ChatTurns(ctx, s.OrgID, project, s.UserID, in.ConversationID, 12); err != nil {
			http.Error(w, "chat failed", http.StatusInternalServerError)
			return
		}
	}
	turn, err := d.Store.StartChatTurn(ctx, s.OrgID, project, s.UserID, in.ConversationID, in.Text)
	if errors.Is(err, store.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		d.Log.Printf("chat: start failed: %v", err)
		http.Error(w, "chat failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	var events []agent.Event
	firstOut := time.Duration(0)
	send := func(v any) {
		raw, _ := json.Marshal(v)
		fmt.Fprintf(w, "data: %s\n\n", raw)
		if flusher != nil {
			flusher.Flush()
		}
	}
	emit := func(e agent.Event) {
		if firstOut == 0 && e.Type != "tool" {
			firstOut = time.Since(began)
		}
		if e.Type != "text" {
			events = append(events, e)
		}
		send(e)
	}
	finish := func(res agent.Result, status string) {
		turn.Status, turn.Outside, turn.Model, turn.Rounds = status, res.Outside, "", res.Rounds
		if d.Chat != nil {
			turn.Model = d.Chat.Name
		}
		turn.InTokens, turn.OutTokens, turn.CacheRead, turn.CacheWrite = res.InTokens, res.OutTokens, res.CacheRead, res.CacheWrite
		turn.FirstMS, turn.LatencyMS = int(firstOut.Milliseconds()), int(time.Since(began).Milliseconds())
		turn.Answer, _ = json.Marshal(map[string]any{"events": events})
		turn.Declined = declined(answerText(events))
		// The turn is stored even if the PM has gone: the changes it made are real.
		save, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := d.Store.FinishChatTurn(save, s.OrgID, turn, res.Messages); err != nil {
			d.Log.Printf("chat: storing the turn failed: %v", err)
		}
		// One line per turn, without content.
		d.Log.Printf("chat turn status=%s rounds=%d tools=%d cards=%d outside=%t in_tokens=%d out_tokens=%d cache_read=%d cache_write=%d first_ms=%d model_ms=%d total_ms=%d",
			status, res.Rounds, len(res.Tools), len(res.Cards), res.Outside, res.InTokens, res.OutTokens, res.CacheRead, res.CacheWrite,
			turn.FirstMS, res.ModelTime.Milliseconds(), turn.LatencyMS)
		if d.Speed != nil && status == "done" {
			d.Speed.Observe("chat_first_response", firstOut)
			switch {
			case contains(res.Tools, "passages"):
				d.Speed.Observe("chat_document_answer", time.Since(began))
			case res.Rounds == 1 && len(res.Cards) > 0:
				d.Speed.Observe("chat_simple_command", time.Since(began))
			case res.Rounds == 1 && movedScreen(events):
				d.Speed.Observe("chat_screen_command", time.Since(began))
			}
		}
		send(chatDone{Type: "done", ConversationID: turn.ConversationID, TurnID: turn.ID, Rounds: res.Rounds, Millis: int64(turn.LatencyMS),
			FirstMillis: int64(turn.FirstMS), InTokens: res.InTokens, OutTokens: res.OutTokens, CacheRead: res.CacheRead, CacheWrite: res.CacheWrite})
	}
	if why := chatUnavailable(ctx, d, s.OrgID); why != "" {
		emit(agent.Event{Type: "error", Text: why})
		finish(agent.Result{}, "failed")
		return
	}
	said := strings.ToLower(strings.TrimRight(in.Text, ".!? "))
	if area, ok := goTo[said]; ok {
		emit(agent.Event{Type: "go", Text: area})
		finish(agent.Result{}, "done")
		return
	}
	if undoPhrases[said] {
		undoLast(ctx, d, who, project, earlier, emit)
		finish(agent.Result{}, "done")
		return
	}
	name := ""
	if p, err := d.Store.GetProject(ctx, s.OrgID, project); err == nil {
		name = p.Name
	}
	today := time.Now()
	req := agent.Request{Model: d.Chat.Name, System: chatSystem(d.Chat.Prompt), Today: today, Think: d.Chat.Think,
		Outline: chatOutline(ctx, d, who, project, name, today.Format("Monday 2 January 2006"))}
	for _, t := range earlier {
		req.Typed = append(req.Typed, t.Text)
		if said := saidBefore(t); said != "" {
			req.History = append(req.History, llm.UserText(t.Text), assistantText(said))
		}
	}
	req.Typed = append(req.Typed, in.Text)
	req.Prefetch = prefetchFor(in.Text)
	var sel *screenSel
	req.Screen, sel = readScreen(ctx, d, who, project, in.Screen)
	box := &chatBox{d: d, who: who, project: project, turn: turn.ID, screen: sel}
	// The turn runs to its end even if the PM closes the strip: a change
	// half way through a batch must be finished or put back, and the card
	// must be stored.
	res := agent.Run(withOpRequest(context.WithoutCancel(ctx), r), d.Chat.Model, box, req, emit)
	if res.Err != nil {
		d.Log.Printf("chat: model call failed: %s", llmErrorClass(res.Err))
		emit(agent.Event{Type: "error", Text: agent.Unavailable(res.Err, res.Cards)})
		finish(res, "failed")
		return
	}
	status := "done"
	if res.Stopped != "" {
		status = "stopped"
	}
	finish(res, status)
}

func llmErrorClass(err error) string {
	switch {
	case errors.Is(err, llm.ErrOff):
		return "off"
	case errors.Is(err, llm.ErrBusy):
		return "breaker open"
	case errors.Is(err, llm.ErrRefused):
		return "refused"
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout"
	}
	return "error"
}

func assistantText(text string) json.RawMessage {
	raw, _ := json.Marshal(map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": text}}})
	return raw
}

// saidBefore is what an earlier turn came to, in a few plain lines, for the
// model's history: the answer, what the cards said and the ids they and the
// screen moves named, so "those" in a later turn has rows to point at. Tool
// results are not replayed: they are stale by the next turn and cost tokens
// on every call.
func saidBefore(t store.ChatTurn) string {
	if t.Status != "done" && t.Status != "stopped" {
		return ""
	}
	var stored struct {
		Events []agent.Event `json:"events"`
	}
	_ = json.Unmarshal(t.Answer, &stored)
	var lines []string
	for _, e := range stored.Events {
		switch {
		case e.Type == "answer" && e.Answer != nil && e.Answer.Text != "":
			lines = append(lines, e.Answer.Text)
		case e.Card != nil && e.Type == "change":
			lines = append(lines, "[Changed"+idList(e.Card.Refs)+": "+strings.Join(e.Card.Lines, " ")+"]")
		case e.Card != nil && e.Type == "prepared":
			lines = append(lines, "[Prepared for the PM's click: "+strings.Join(e.Card.Lines, " ")+"]")
		case e.Type == "go":
			lines = append(lines, "[Opened "+e.Text+"]")
		case e.Type == "view" && e.View != nil:
			lines = append(lines, viewSaid(*e.View))
		}
	}
	return strings.Join(lines, "\n")
}

// getChat is GET /projects/{id}/chat: the caller's latest conversation on
// the project, whether chat can run, and why not.
func getChat(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, false)
	if !ok {
		return
	}
	turns, err := d.Store.ChatTurns(r.Context(), s.OrgID, r.PathValue("id"), s.UserID, "", 30)
	if err != nil {
		http.Error(w, "chat failed", http.StatusInternalServerError)
		return
	}
	if turns == nil {
		turns = []store.ChatTurn{}
	}
	why := chatUnavailable(r.Context(), d, s.OrgID)
	writeJSON(w, http.StatusOK, map[string]any{"available": why == "", "unavailable": why, "turns": turns})
}

// postChatWarm is POST /projects/{id}/chat/warm: the strip was focused, so
// the cached prefix is warmed with a call that generates nothing (speed
// rule 6). At most once per five minutes per user; it never fails loudly.
func postChatWarm(w http.ResponseWriter, r *http.Request, d Deps) {
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	if chatUnavailable(r.Context(), d, s.OrgID) != "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	now := time.Now()
	if last, ok := d.Chat.warmed.Load(s.UserID); ok && now.Sub(last.(time.Time)) < 5*time.Minute {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	d.Chat.warmed.Store(s.UserID, now)
	conv := llm.Conversation{Name: "chat_warm", Model: d.Chat.Name, System: chatSystem(d.Chat.Prompt), Tools: chatTools(),
		History: []json.RawMessage{llm.UserText("warm")}, Warm: true, Timeout: 20 * time.Second, Think: d.Chat.Think}
	go func() {
		if _, err := d.Chat.Model.Converse(context.Background(), conv, nil); err != nil {
			d.Log.Printf("chat: warming failed: %s", llmErrorClass(err))
		}
	}()
	w.WriteHeader(http.StatusNoContent)
}

// postChatPrepared is POST /projects/{id}/chat/prepared/{prepared}/apply
// and …/dismiss: the PM's click on a prepared change, with their session.
func postChatPrepared(w http.ResponseWriter, r *http.Request, d Deps) {
	if !pathIDs(w, r, "prepared") {
		return
	}
	s, ok := deliverySession(w, r, d, true)
	if !ok {
		return
	}
	project, id := r.PathValue("id"), r.PathValue("prepared")
	state := "applied"
	if strings.HasSuffix(r.URL.Path, "/dismiss") {
		state = "dismissed"
	}
	p, err := d.Store.TakeChatPrepared(r.Context(), s.OrgID, project, s.UserID, id, state)
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.NotFound(w, r)
		return
	case errors.Is(err, store.ErrConflict):
		http.Error(w, "already "+p.State, http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "chat failed", http.StatusInternalServerError)
		return
	}
	if state == "dismissed" {
		writeJSON(w, http.StatusOK, map[string]any{"dismissed": true})
		return
	}
	batch := opBatch{Area: p.Area, TurnID: p.TurnID, Via: "chat", Clicked: true}
	if err := json.Unmarshal(p.Ops, &batch.Calls); err != nil {
		http.Error(w, "chat failed", http.StatusInternalServerError)
		return
	}
	// The operations read the current values now, not those of the turn
	// that prepared them (review change 7).
	done, bad := applyOps(withOpRequest(r.Context(), r), d, opActor{s.OrgID, s.UserID}, project, batch)
	if bad != nil {
		if err := d.Store.ReopenChatPrepared(r.Context(), s.OrgID, id); err != nil {
			d.Log.Printf("chat: reopening a prepared change failed: %v", err)
		}
		writeOpError(w, bad)
		return
	}
	if d.Broker != nil {
		d.Broker.Wake(s.OrgID)
	}
	card := agent.Card{Kind: "change", Area: p.Area, ChangeID: done.ChangeID, Undo: done.Undo.Kind}
	for _, res := range done.Results {
		card.Lines = append(card.Lines, res.Summary)
	}
	writeJSON(w, http.StatusOK, card)
}
