package jobs

import (
	"testing"

	"sitewise/internal/store"
)

// A reply in the thread of the email an RFI went out on matches by rule,
// though its text never names the RFI.
func TestRuleMatchesAThreadedReply(t *testing.T) {
	hits, left := RuleMatches("It is 1.2 m deep.", []store.RegisterRFI{{ID: "a", Number: 3, Threaded: true}, {ID: "b", Number: 4}})
	if len(hits) != 1 || hits[0].ItemID != "a" || hits[0].DecidedBy != "rule" || len(left) != 1 || left[0].ID != "b" {
		t.Fatalf("hits %+v left %+v", hits, left)
	}
}
