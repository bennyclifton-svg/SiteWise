package jobs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"sitewise/internal/jev"
)

// A profile read keeps every background Jev slot busy, and no more: the pool
// is the client's background allowance, so filing keeps its reserve and no
// passage waits in admission against its own deadline.
func TestEachPassageFillsTheBackgroundAllowance(t *testing.T) {
	want := jev.TotalSlots - jev.InteractiveReserve
	var mu sync.Mutex
	inFlight, peak := 0, 0
	full := make(chan struct{})
	err := eachPassage(context.Background(), 3*want, func(ctx context.Context, i int) error {
		mu.Lock()
		inFlight++
		if inFlight > peak {
			peak = inFlight
			if peak == want {
				close(full)
			}
		}
		mu.Unlock()
		select {
		case <-full:
		case <-time.After(300 * time.Millisecond):
		}
		mu.Lock()
		inFlight--
		mu.Unlock()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if peak != want {
		t.Fatalf("%d passages in flight at most, want %d", peak, want)
	}
}

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
