package jobs

import (
	"fmt"
	"testing"

	"sitewise/internal/jev"
)

// An open breaker fails every call until its cooldown passes, so retrying
// sooner only spends the attempt budget without asking Jev anything.
func TestCircuitOpenWaitsOutBreakerCooldown(t *testing.T) {
	w := &Worker{Backoff: 0}
	if got := w.retryAfter(fmt.Errorf("evidence: %w", jev.ErrCircuitOpen)); got < jev.BreakerCooldown {
		t.Fatalf("circuit open backoff %s, want >= %s", got, jev.BreakerCooldown)
	}
	if got := w.retryAfter(fmt.Errorf("parse failed")); got != 0 {
		t.Fatalf("ordinary failure backoff %s, want 0", got)
	}
}
