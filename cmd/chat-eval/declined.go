package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"sitewise/internal/store"
)

// The decline measure (docs/plans/2026-11-18-chat-agency.md, Task 7):
//
//	chat-eval -declined [-weeks 12] [-org <the owner's org id>]
//
// reads SITEWISE_DATABASE_URL and prints, week by week, the chat turns that
// ended in "I can't do that from chat yet" over all finished turns. That
// share, not the key, is the measure of how much of what a PM asks chat can
// do. Counts are for every org. The words of a declined message are listed
// for the one org named by -org, the owner's own, and never for any other
// (owner decision 3, 18 November 2026).

// declineReader is the two store reads the report makes.
type declineReader interface {
	ChatDeclineWeeks(ctx context.Context, since time.Time) ([]store.ChatDeclineWeek, error)
	ChatDeclinedTexts(ctx context.Context, org string, since time.Time, limit int) ([]store.ChatDeclined, error)
}

// maxDeclinedListed bounds the list of one org's declined messages.
const maxDeclinedListed = 200

func reportDeclined(ctx context.Context, st declineReader, org string, since time.Time, w io.Writer) error {
	if org != "" && !uuidLike(org) {
		return fmt.Errorf("-org must be an org id")
	}
	weeks, err := st.ChatDeclineWeeks(ctx, since)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "Chat turns declined with \"I can't do that from chat yet\", every org, by week (Monday, Sydney).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Week from | Turns | Declined | Share |")
	fmt.Fprintln(w, "|---|---|---|---|")
	turns, declined := 0, 0
	for _, wk := range weeks {
		turns, declined = turns+wk.Turns, declined+wk.Declined
		fmt.Fprintf(w, "| %s | %d | %d | %s |\n", wk.Week.Format("2006-01-02"), wk.Turns, wk.Declined, share(wk.Declined, wk.Turns))
	}
	fmt.Fprintf(w, "\nAll weeks: %d of %d turns declined (%s).\n", declined, turns, share(declined, turns))
	if org == "" {
		fmt.Fprintln(w, "\nThe words of declined messages are listed only for the org named by -org.")
		return nil
	}
	texts, err := st.ChatDeclinedTexts(ctx, org, since, maxDeclinedListed)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "\nDeclined messages of org %s, newest first (at most %d):\n", org, maxDeclinedListed)
	if len(texts) == 0 {
		fmt.Fprintln(w, "(none)")
	}
	for _, d := range texts {
		fmt.Fprintf(w, "%s  %s\n", d.At.Format("2006-01-02 15:04"), d.Text)
	}
	return nil
}

func share(part, whole int) string {
	if whole == 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", 100*float64(part)/float64(whole))
}

func uuidLike(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, r := range s {
		switch {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if r != '-' {
				return false
			}
		case (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F'):
			return false
		}
	}
	return true
}
