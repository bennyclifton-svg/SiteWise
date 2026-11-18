package httpapi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The prompt tells the model to use the fixed words, and Go detects the
// same words: if either drifts, the decline measure counts nothing.
func TestDeclineWordsAreThePrompts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "knowledge", "chat", "system.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `say exactly "`+declineWords+`"`) {
		t.Fatalf("knowledge/chat/system.md does not tell the model to say exactly %q", declineWords)
	}
	prompt, err := LoadChatPrompt(filepath.Join("..", "..", "knowledge", "chat"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(chatSystem(prompt), `"`+declineWords+`"`) {
		t.Fatal("the operation list does not give the fixed words")
	}
	for text, want := range map[string]bool{
		"I can't do that from chat yet. Messages arrive with email intake.": true,
		"I can’t do that from chat yet; use Procurement.":                   true,
		"i CAN'T do that from chat yet":                                     true,
		"I can do that from chat.":                                          false,
		"This cannot be undone from chat yet, so it waits for your click.":  false,
	} {
		if declined(text) != want {
			t.Errorf("declined(%q) = %t", text, !want)
		}
	}
}
