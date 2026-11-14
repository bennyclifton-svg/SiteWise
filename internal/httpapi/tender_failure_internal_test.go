package httpapi

import (
	"errors"
	"testing"
	"time"

	"sitewise/internal/llm"
	"sitewise/internal/tender"
)

func TestTenderFailureWordsAndWaits(t *testing.T) {
	plain := errors.New("connection reset")
	for _, c := range []struct {
		name     string
		cause    error
		attempts int
		words    string
		wait     time.Duration
		off      bool
	}{
		{"first transport error", plain, 1, tenderUnreachable, 30 * time.Second, false},
		{"second", plain, 2, tenderUnreachable, 2 * time.Minute, false},
		{"third", plain, 3, tenderUnreachable, 10 * time.Minute, false},
		{"the fourth failure stops retrying", plain, 4, tenderGaveUp, 0, false},
		{"an empty answer is asked again", tender.ErrEmpty, 1, tenderListedNone, 30 * time.Second, false},
		{"a refusal is not retried", llm.ErrRefused, 1, tenderDeclined, 0, false},
		{"no renderer is not retried", tender.ErrNoRenderer, 1, tenderNoImages, 0, false},
		{"a file kind that cannot be read", errTenderKind, 1, tenderWrongKind, 0, false},
		{"the pass is off", llm.ErrOff, 1, "", 0, true},
	} {
		words, wait, off := tenderFailure(c.cause, c.attempts)
		if words != c.words || wait != c.wait || off != c.off {
			t.Errorf("%s: %q %s %v", c.name, words, wait, off)
		}
	}
}
