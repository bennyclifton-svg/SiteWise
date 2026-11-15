// Command chat-probe measures the model before chat is built on it (plan
// Task 2). It is a developer tool and is not part of the release build.
//
// It times, through the same llm.Converse the chat loop uses:
//
//	a  a cached prefix of about 6,000 tokens and a one-tool answer
//	b  the same with a tool result of about 30,000 tokens in the history
//
// The key comes from SITEWISE_ANTHROPIC_HAIKU_KEY and is never printed.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"sitewise/internal/llm"
)

var tools = []llm.Tool{
	{Name: "apply", Description: "Apply operations to one area of the project. Set final true when nothing more needs saying.", Schema: map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"area", "ops"},
		"properties": map[string]any{
			"area":  map[string]any{"type": "string", "enum": []string{"cost_plan", "profile", "program", "procurement"}},
			"ops":   map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
			"final": map[string]any{"type": "boolean"},
		}}},
	{Name: "read", Description: "Read one area of the project.", Schema: map[string]any{
		"type": "object", "additionalProperties": false, "required": []string{"area"},
		"properties": map[string]any{"area": map[string]any{"type": "string"}}}},
}

// filler is deterministic text of roughly n tokens, so the prefix caches.
func filler(n int, word string) string {
	var b strings.Builder
	for i := 0; b.Len() < n*4; i++ {
		fmt.Fprintf(&b, "Line %d: the %s line of the reference list holds item %d with amount unknown.\n", i, word, i*7%1000)
	}
	return b.String()
}

func main() {
	models := flag.String("models", "claude-haiku-5-5,claude-sonnet-5-5", "comma-separated model ids")
	n := flag.Int("n", 20, "samples per case")
	flag.Parse()
	client := llm.New(llm.Options{APIKey: os.Getenv("SITEWISE_ANTHROPIC_HAIKU_KEY")})
	if !client.On() {
		fmt.Fprintln(os.Stderr, "chat-probe: SITEWISE_ANTHROPIC_HAIKU_KEY is not set")
		os.Exit(1)
	}
	system := "You are the chat strip of a construction project app. Turn the user's instruction into one apply call with final true. " +
		"Operation cost_plan.add_line takes label and optional amount. Do not write prose before a tool call.\n\nReference list:\n" + filler(6000, "prefix")
	big := filler(30000, "result")
	fmt.Println("| Model | Case | Samples | First token p50 | First token p90 | Total p50 | Total p90 | Input tok (median) | Cache read (median) | Output tok (median) | Errors |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|")
	for _, model := range strings.Split(*models, ",") {
		ctx := context.Background()
		warm, err := client.Converse(ctx, llm.Conversation{Name: "probe_warm", Model: model, System: system, Tools: tools,
			History: []json.RawMessage{llm.UserText("warm")}, Warm: true}, nil)
		fmt.Fprintf(os.Stderr, "%s warm: %v, cache write %d, cache read %d, %d ms\n", model, err, warm.CacheWrite, warm.CacheRead, warm.Latency.Milliseconds())
		a := func(i int) llm.Conversation {
			return llm.Conversation{Name: "probe_a", Model: model, System: system, Tools: tools, MaxTokens: 300,
				History: []json.RawMessage{llm.UserText(fmt.Sprintf("Add a cost line for joinery, $%d,000.", 40+i))}}
		}
		run(client, model, "a: 6k cached prefix, one tool call", *n, a)
		// Case b needs a real assistant tool call to answer, so take one.
		first, err := client.Converse(ctx, llm.Conversation{Name: "probe_b0", Model: model, System: system, Tools: tools, MaxTokens: 300,
			History: []json.RawMessage{llm.UserText("Read the cost plan with the read tool, then tell me in one sentence how many lines it has.")}}, nil)
		if err != nil || len(first.Calls) == 0 {
			fmt.Fprintf(os.Stderr, "%s: case b skipped: no tool call (%v)\n", model, err)
			continue
		}
		b := func(i int) llm.Conversation {
			return llm.Conversation{Name: "probe_b", Model: model, System: system, Tools: tools, MaxTokens: 800,
				History: []json.RawMessage{
					llm.UserText("Read the cost plan with the read tool, then tell me in one sentence how many lines it has."),
					first.Message,
					llm.ToolResults([]llm.ToolResult{{CallID: first.Calls[0].ID, Text: fmt.Sprintf("Run %d.\n%s", i, big)}}),
				}}
		}
		run(client, model, "b: plus a 30k-token tool result, text answer", *n, b)
	}
}

func run(client *llm.Client, model, label string, n int, conv func(int) llm.Conversation) {
	var first, total, in, read, out []int
	errs := 0
	for i := 0; i < n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		turn, err := client.Converse(ctx, conv(i), nil)
		cancel()
		if err != nil {
			errs++
			fmt.Fprintf(os.Stderr, "%s %s: %v\n", model, label, err)
			continue
		}
		first = append(first, int(turn.FirstToken.Milliseconds()))
		total = append(total, int(turn.Latency.Milliseconds()))
		in, read, out = append(in, int(turn.InTokens)), append(read, int(turn.CacheRead)), append(out, int(turn.OutTokens))
	}
	fmt.Printf("| %s | %s | %d | %d ms | %d ms | %d ms | %d ms | %d | %d | %d | %d |\n", model, label, len(total),
		pct(first, 50), pct(first, 90), pct(total, 50), pct(total, 90), pct(in, 50), pct(read, 50), pct(out, 50), errs)
}

func pct(s []int, p int) int {
	if len(s) == 0 {
		return 0
	}
	sort.Ints(s)
	i := (len(s)*p + 99) / 100
	if i < 1 {
		i = 1
	}
	return s[i-1]
}
