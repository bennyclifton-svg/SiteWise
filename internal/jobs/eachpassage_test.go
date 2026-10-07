package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// One passage failing must not cancel calls already in flight: a cancelled
// half-open probe would keep the Jev circuit open on every retry.
func TestEachPassageLetsInFlightCallsFinish(t *testing.T) {
	var mu sync.Mutex
	var cancelled []int
	err := eachPassage(context.Background(), 2, func(ctx context.Context, i int) error {
		if i == 1 {
			return errors.New("refused")
		}
		select {
		case <-ctx.Done():
			mu.Lock()
			cancelled = append(cancelled, i)
			mu.Unlock()
		case <-time.After(100 * time.Millisecond):
		}
		return nil
	})
	if err == nil || err.Error() != "refused" {
		t.Fatalf("err %v", err)
	}
	if len(cancelled) != 0 {
		t.Fatalf("in-flight passages cancelled: %v", cancelled)
	}
}
