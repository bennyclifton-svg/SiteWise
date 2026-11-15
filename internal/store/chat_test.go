package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"sitewise/internal/store"
)

func TestChatConversationBelongsToItsUser(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	if err := s.CreateProject(ctx, orgA, projectA2, "Second"); err != nil {
		t.Fatal(err)
	}
	first, err := s.StartChatTurn(ctx, orgA, projectA, userA, "", "Add a line for joinery")
	if err != nil || first.ConversationID == "" {
		t.Fatalf("turn %+v err %v", first, err)
	}
	first.Status, first.Rounds, first.CacheRead, first.Outside = "done", 1, 6100, true
	first.Answer = json.RawMessage(`{"text":"Added."}`)
	if err := s.FinishChatTurn(ctx, orgA, first, []json.RawMessage{json.RawMessage(`{"role":"user"}`), json.RawMessage(`{"role":"assistant"}`)}); err != nil {
		t.Fatal(err)
	}
	second, err := s.StartChatTurn(ctx, orgA, projectA, userA, first.ConversationID, "And stone")
	if err != nil || second.ConversationID != first.ConversationID {
		t.Fatalf("turn %+v err %v", second, err)
	}
	turns, err := s.ChatTurns(ctx, orgA, projectA, userA, "", 20)
	if err != nil || len(turns) != 2 || turns[0].Text != "Add a line for joinery" || turns[0].Status != "done" || !turns[0].Outside || string(turns[0].Answer) != `{"text": "Added."}` {
		t.Fatalf("turns %+v err %v", turns, err)
	}
	// Another user of the same org, another project and another org see
	// nothing and cannot add to it.
	for name, who := range map[string][3]string{"another user": {orgA, projectA, userA2}, "another project": {orgA, projectA2, userA}, "another org": {orgB, projectB, userB}} {
		if got, err := s.ChatTurns(ctx, who[0], who[1], who[2], first.ConversationID, 20); err != nil || len(got) != 0 {
			t.Errorf("%s reads %d turns (err %v)", name, len(got), err)
		}
		if _, err := s.StartChatTurn(ctx, who[0], who[1], who[2], first.ConversationID, "mine now"); !errors.Is(err, store.ErrNotFound) {
			t.Errorf("%s: start err %v, want not found", name, err)
		}
	}

	// A prepared change is taken once, by its own user.
	id, err := s.AddChatPrepared(ctx, orgA, store.ChatPrepared{TurnID: first.ID, Area: "cost_plan", Ops: json.RawMessage(`[{"op":"cost_plan.freeze"}]`), Summary: "Freeze", Why: "yours to confirm"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.TakeChatPrepared(ctx, orgA, projectA, userA2, id, "applied"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another user took it: %v", err)
	}
	if _, err := s.TakeChatPrepared(ctx, orgB, projectB, userB, id, "applied"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another org took it: %v", err)
	}
	p, err := s.TakeChatPrepared(ctx, orgA, projectA, userA, id, "applied")
	if err != nil || p.Area != "cost_plan" || p.State != "applied" {
		t.Fatalf("prepared %+v err %v", p, err)
	}
	if _, err := s.TakeChatPrepared(ctx, orgA, projectA, userA, id, "applied"); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("second take: %v", err)
	}

	// Deleting the project removes all four tables' rows.
	if _, err := s.DeleteProject(ctx, orgA, projectA); err != nil {
		t.Fatal(err)
	}
	pool := rawPool(t)
	for _, table := range []string{"chat_conversations", "chat_turns", "chat_blocks", "chat_prepared"} {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE org_id=$1::uuid`, orgA).Scan(&n); err != nil || n != 0 {
			t.Errorf("%s keeps %d rows after the project is deleted (err %v)", table, n, err)
		}
	}
}
