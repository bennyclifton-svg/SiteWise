package store_test

import (
	"context"
	"testing"
	"time"
)

// The decline measure (agency plan, Task 7, owner decision 3): counts are
// read for every org, a declined message's words for the named org only.
func TestChatDeclinedCountsEveryOrgAndTextsOneOrg(t *testing.T) {
	ctx := context.Background()
	s := profileStore(t)
	since := time.Now().Add(-time.Minute)
	weeks := func() (turns, declined int) {
		t.Helper()
		got, err := s.ChatDeclineWeeks(ctx, since)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range got {
			turns, declined = turns+w.Turns, declined+w.Declined
		}
		return turns, declined
	}
	turns0, declined0 := weeks()
	say := func(org, project, user, text string, declined bool, status string) {
		t.Helper()
		turn, err := s.StartChatTurn(ctx, org, project, user, "", text)
		if err != nil {
			t.Fatal(err)
		}
		turn.Status, turn.Declined = status, declined
		if err := s.FinishChatTurn(ctx, org, turn, nil); err != nil {
			t.Fatal(err)
		}
	}
	say(orgA, projectA, userA, "Show me the messages", true, "done")
	say(orgA, projectA, userA, "Add a joinery line", false, "done")
	say(orgB, projectB, userB, "Their secret wording about the RFIs", true, "done")
	// A failed turn (chat off, the model down) is not a turn chat answered.
	say(orgA, projectA, userA, "Failed turn", false, "failed")

	if turns, declined := weeks(); turns-turns0 != 3 || declined-declined0 != 2 {
		t.Fatalf("turns +%d declined +%d, want +3 and +2", turns-turns0, declined-declined0)
	}
	mine, err := s.ChatDeclinedTexts(ctx, orgA, since, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || mine[0].Text != "Show me the messages" {
		t.Fatalf("org A's declined messages %+v", mine)
	}
	theirs, err := s.ChatDeclinedTexts(ctx, orgB, since, 50)
	if err != nil || len(theirs) != 1 || theirs[0].Text != "Their secret wording about the RFIs" {
		t.Fatalf("org B's declined messages %+v err %v", theirs, err)
	}
}
