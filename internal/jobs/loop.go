package jobs

import (
	"context"
	"errors"
	"time"

	"sitewise/internal/store"
)

// Run works background jobs until ctx ends: for each org with queued work it
// runs jobs one at a time until that org is idle, then waits poll. One job at
// a time per process keeps filing ahead; background Jev calls already cannot
// take the interactive reserve.
func Run(ctx context.Context, w *Worker, orgs func(context.Context) ([]string, error), poll time.Duration, logf func(string, ...any)) {
	if poll <= 0 {
		poll = 2 * time.Second
	}
	for {
		list, err := orgs(ctx)
		if err != nil && ctx.Err() == nil {
			logf("background: list orgs: %v", err)
		}
		for _, org := range list {
			for ctx.Err() == nil {
				err := w.Once(ctx, org)
				if errors.Is(err, store.ErrIdle) {
					break
				}
				if err != nil {
					// The job is failed with backoff; move on rather than spin.
					logf("background: org %s: %v", org, err)
					break
				}
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(poll):
		}
	}
}
