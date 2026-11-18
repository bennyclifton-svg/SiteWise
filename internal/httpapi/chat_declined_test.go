package httpapi_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"sitewise/internal/llm"
)

// A turn whose answer carries the fixed words is stored as declined, and
// only such a turn (agency plan, Task 7).
func TestChatTurnDeclinedIsStored(t *testing.T) {
	_, m, project, _ := chatApp(t, func(_ int, conv llm.Conversation) llm.Turn {
		if strings.Contains(lastUser(conv), "messages") {
			return llm.Turn{Text: "I can’t do that from chat yet. Messages come with email intake.", StopReason: "end_turn"}
		}
		return llm.Turn{Text: "The plan has no lines yet.", StopReason: "end_turn"}
	})
	m.chat(t, project, "", "Show me the messages")
	m.chat(t, project, "", "What cost lines are there?")
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, os.Getenv("SITEWISE_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, `SELECT text, declined FROM chat_turns WHERE org_id=$1::uuid ORDER BY n`, m.orgID)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for rows.Next() {
		var text string
		var declined bool
		if err := rows.Scan(&text, &declined); err != nil {
			t.Fatal(err)
		}
		got[text] = declined
	}
	if len(got) != 2 || !got["Show me the messages"] || got["What cost lines are there?"] {
		t.Fatalf("declined by message: %v", got)
	}
}
