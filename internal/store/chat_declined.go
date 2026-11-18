package store

import (
	"context"
	"time"
)

// The decline measure (docs/plans/2026-11-18-chat-agency.md, Task 7, owner
// decision 3): turns chat declined, over all turns, week by week. Counts
// are read for every org; a message's words are read for one org only, the
// one the caller names.

// ChatDeclineWeek is one week's finished chat turns and how many of them
// chat declined. Week is the Monday the week starts, in Sydney.
type ChatDeclineWeek struct {
	Week     time.Time
	Turns    int
	Declined int
}

// ChatDeclineWeeks counts finished turns (done or stopped) and declined
// ones per week since the given time, across every org. It reads counts
// only: no message, no org and no project leaves the query.
func (s *Store) ChatDeclineWeeks(ctx context.Context, since time.Time) ([]ChatDeclineWeek, error) {
	rows, err := s.pool.Query(ctx, `SELECT date_trunc('week', at AT TIME ZONE 'Australia/Sydney')::date, count(*), count(*) FILTER (WHERE declined)
FROM chat_turns WHERE status IN ('done','stopped') AND at >= $1 GROUP BY 1 ORDER BY 1`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatDeclineWeek
	for rows.Next() {
		var w ChatDeclineWeek
		if err := rows.Scan(&w.Week, &w.Turns, &w.Declined); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ChatDeclined is one declined message of one org.
type ChatDeclined struct {
	At   time.Time
	Text string
}

// ChatDeclinedTexts is one org's declined messages since the given time,
// newest first, at most limit. Every row read is that org's.
func (s *Store) ChatDeclinedTexts(ctx context.Context, org string, since time.Time, limit int) ([]ChatDeclined, error) {
	rows, err := s.pool.Query(ctx, `SELECT at, text FROM chat_turns
WHERE org_id=$1::uuid AND declined AND status IN ('done','stopped') AND at >= $2 ORDER BY at DESC LIMIT $3`, org, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatDeclined
	for rows.Next() {
		var d ChatDeclined
		if err := rows.Scan(&d.At, &d.Text); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
