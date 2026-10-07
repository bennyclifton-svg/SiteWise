package store

import (
	"encoding/json"
	"sitewise/internal/reports"
	"testing"
)

func TestUnreviewedReportClauseIncludesPackageScope(t *testing.T) {
	for _, b := range []reports.Block{{ID: "clause:services:standard", Provisional: true, Basis: json.RawMessage(`{}`)}, {ID: "scope:stable-scope-id", Provisional: true, Basis: json.RawMessage(`{"clause_id":"cl.design","clause_version":1}`)}} {
		if !unreviewedReportClause(b) {
			t.Fatal("draft obligation accepted", b)
		}
	}
	for _, b := range []reports.Block{{ID: "scope:reviewed", Basis: json.RawMessage(`{"clause_id":"cl.design","clause_version":1}`)}, {ID: "cost:unknown", Provisional: true, Basis: json.RawMessage(`{"method":"cost_plan_availability"}`)}} {
		if unreviewedReportClause(b) {
			t.Fatal("unrelated unknown blocked", b)
		}
	}
}
