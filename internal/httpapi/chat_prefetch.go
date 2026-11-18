package httpapi

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"sitewise/internal/agent"
	"sitewise/internal/llm"
)

// Reading ahead (speed rule 4: nothing a model call is spent on that Go can
// do). A question about figures used to cost two model calls: one to ask
// for a read, one to answer. When the PM's words already say which area
// the question is about, Go reads that area before the model is asked and
// the answer takes one call. A wrong guess costs input tokens, not a
// call: the model can still read whatever else it needs.
//
// Only questions are read ahead. A command needs names and ids, which the
// outline has, and a read on a command turn could make its change wait
// for a click (a profile read can carry document text).

// prefetchWords maps words in the PM's question to the area that holds the
// answer. Order within an area does not matter; a word is matched whole.
var prefetchWords = map[string][]string{
	"cost_plan": {"budget", "budgets", "cost", "costs", "contract", "claimed", "remaining", "left", "contingency",
		"spend", "spent", "how much", "amount", "amounts", "allowance", "allowances", "cost line", "cost lines", "cost plan"},
	"program": {"program", "programme", "schedule", "when", "finish", "finishes", "start", "starts", "duration",
		"milestone", "milestones", "late", "completion", "stage", "stages", "hold point", "hold points"},
	"profile": {"site area", "zone", "zoning", "gfa", "floor area", "storeys", "classification", "class", "client",
		"address", "lot", "council", "lga", "height", "unit mix"},
	"procurement": {"tender", "tenders", "tendered", "quote", "quotes", "quoted", "firm", "firms", "package",
		"packages", "award", "awarded", "invited", "consultant", "consultants"},
	// What is risky comes from both lists (knowledge/chat/system.md), so
	// the risk words name both.
	"attention":     {"attention", "outstanding", "to do", "next", "risk", "risks", "risky"},
	"implications":  {"risk", "risks", "risky", "implication", "implications"},
	"invoices":      {"invoice", "invoices", "paid"},
	"accommodation": {"room", "rooms", "accommodation", "bedroom", "bedrooms"},
	"shares":        {"link", "links", "shared", "guest", "guests", "opened", "can see"},
}

// questionStart is a first word that makes a message a question even
// without a question mark ("show me the budget").
var questionStart = map[string]bool{"what": true, "what's": true, "whats": true, "how": true, "when": true, "which": true,
	"who": true, "where": true, "why": true, "is": true, "are": true, "does": true, "do": true, "did": true, "has": true,
	"have": true, "can": true, "could": true, "show": true, "list": true, "tell": true, "give": true, "any": true,
	"summarise": true, "summarize": true}

var nonWord = regexp.MustCompile(`[^a-z0-9']+`)

// maxPrefetch bounds what a question adds to the first call.
const maxPrefetch = 2

// prefetchFor is the reads to run ahead for the PM's message: none unless
// it is a question, then the areas its words name, earliest named first.
func prefetchFor(text string) []agent.Prefetch {
	words := strings.Fields(nonWord.ReplaceAllString(strings.ToLower(text), " "))
	if len(words) == 0 || !(strings.Contains(text, "?") || questionStart[words[0]]) {
		return nil
	}
	padded := " " + strings.Join(words, " ") + " "
	type hit struct {
		area string
		at   int
	}
	var hits []hit
	for area, list := range prefetchWords {
		first := -1
		for _, w := range list {
			if at := strings.Index(padded, " "+w+" "); at >= 0 && (first < 0 || at < first) {
				first = at
			}
		}
		if first >= 0 {
			hits = append(hits, hit{area, first})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].at != hits[j].at {
			return hits[i].at < hits[j].at
		}
		return hits[i].area < hits[j].area
	})
	if len(hits) > maxPrefetch {
		hits = hits[:maxPrefetch]
	}
	out := make([]agent.Prefetch, 0, len(hits))
	for _, h := range hits {
		input, _ := json.Marshal(map[string]string{"area": h.area})
		out = append(out, agent.Prefetch{Area: h.area, Call: llm.ToolCall{Name: "read", Input: input}})
	}
	return out
}
