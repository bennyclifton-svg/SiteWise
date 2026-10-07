package store

import (
	"encoding/json"
	"strings"
	"testing"

	"sitewise/internal/profile"
)

func TestReportAuditDoesNotBorrowNewerOrOtherScopeAuthor(t *testing.T) {
	row := profile.Row{PartID: "part", Key: "hdr.work_type", Scope: "project", UserVersion: 1}
	audit := map[string]json.RawMessage{
		reportAuditKey("project", "part", row.Key, 2): json.RawMessage(`{"user_id":"newer-author"}`),
		reportAuditKey("site", "part", row.Key, 1):    json.RawMessage(`{"user_id":"other-scope"}`),
	}
	raw, err := reportRowBasis(row, audit)
	if err != nil || strings.Contains(string(raw), "user_id") {
		t.Fatalf("wrong attribution %s %v", raw, err)
	}
	audit[reportAuditKey("project", "part", row.Key, 1)] = json.RawMessage(`{"user_id":"correct-author"}`)
	raw, err = reportRowBasis(row, audit)
	if err != nil || !strings.Contains(string(raw), "correct-author") {
		t.Fatalf("missing attribution %s %v", raw, err)
	}
}
