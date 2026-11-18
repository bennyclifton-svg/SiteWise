package httpapi

import (
	"fmt"
	"strings"
	"testing"

	"sitewise/internal/agent"
)

// The history names at most fifty ids, then how many more.
func TestHistoryIDsAreCapped(t *testing.T) {
	var ids []string
	for i := 0; i < 64; i++ {
		ids = append(ids, fmt.Sprintf("%08x-0000-4000-8000-000000000000", i))
	}
	said := viewSaid(agent.View{Do: "select", Table: "documents", IDs: ids})
	if !strings.HasPrefix(said, "[Selected 64 documents: 00000000 00000001 ") || !strings.HasSuffix(said, " 00000031 and 14 more]") {
		t.Fatalf("said %q", said)
	}
	if got := viewSaid(agent.View{Do: "open", Dialog: "award", Fields: map[string]string{"package": ids[1], "column": ids[2]}}); got != "[Opened the award dialog: package 00000001, column 00000002]" {
		t.Fatalf("dialog said %q", got)
	}
}

func TestCardRefsTakeIDsFromFields(t *testing.T) {
	calls := []opCall{{Op: "cost_plan.set_lines", Fields: []byte(`{"lines":["3f9a1b2c","3F9A1B2C","9e8d7c6b-1111-4111-8111-111111111111"],"excluded":true,"label":"Footings"}`)}}
	if got := cardRefs(calls); strings.Join(got, " ") != "3f9a1b2c 9e8d7c6b" {
		t.Fatalf("refs %v", got)
	}
}
