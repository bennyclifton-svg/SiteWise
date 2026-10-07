package knowledge_test

import (
	"reflect"
	"testing"

	"sitewise/internal/knowledge"
)

func TestWorksTraceIncludesOnlyReadInputs(t *testing.T) {
	cat := loadRepo(t)
	p := map[string]any{"all": []any{
		map[string]any{"works": map[string]any{"action": []any{"upgrade"}, "system": []any{"fire-active.hydrants"}}},
		map[string]any{"det": "existing_building", "is": true},
		map[string]any{"system_existing": "fire-active.fire-water"},
		map[string]any{"system_present": "fire-active.sprinklers"},
	}}
	env := knowledge.WorksEnv{Values: map[string]string{"existing_building": "true", "unrelated": "42"}, Items: []knowledge.WorkItem{{System: "electrical", Action: "new"}, {System: "fire-active.hydrants", Action: "upgrade"}, {System: "fire-active.fire-water", Action: "retain"}}, Present: func(string) bool { return true }}
	trace := cat.TraceWorks(p, env)
	if trace.Truth != knowledge.True || trace.Specificity != 3 || !reflect.DeepEqual(trace.Determinants, []string{"existing_building"}) || !reflect.DeepEqual(trace.ItemIndexes, []int{1, 2}) {
		t.Fatal(trace)
	}
	if trace.Existing["fire-active.fire-water"] != knowledge.True || trace.Present["fire-active.sprinklers"] != knowledge.True {
		t.Fatal(trace)
	}
	env.Items[2].Action = "replace"
	trace = cat.TraceWorks(p, env)
	if trace.Truth != knowledge.False || trace.Existing["fire-active.fire-water"] != knowledge.False {
		t.Fatal(trace)
	}
}

func TestWorksTraceRetainsUnknownInputs(t *testing.T) {
	cat := loadRepo(t)
	p := map[string]any{"all": []any{map[string]any{"det": "work_type", "eq": "refurb"}, map[string]any{"system_existing": "structure"}}}
	trace := cat.TraceWorks(p, knowledge.WorksEnv{})
	if trace.Truth != knowledge.Unknown || trace.Existing["structure"] != knowledge.Unknown || !reflect.DeepEqual(trace.Determinants, []string{"work_type"}) {
		t.Fatal(trace)
	}
}

func TestPresentStateUnknownDoesNotHidePredicate(t *testing.T) {
	cat := loadRepo(t)
	p := map[string]any{"system_present": "structure"}
	env := knowledge.WorksEnv{Present: func(string) bool { return false }, PresentState: func(string) knowledge.Truth { return knowledge.Unknown }}
	if got := cat.Holds(p, env); got != knowledge.Unknown {
		t.Fatal(got)
	}
	trace := cat.TraceWorks(p, env)
	if trace.Truth != knowledge.Unknown || trace.Present["structure"] != knowledge.Unknown {
		t.Fatal(trace)
	}
}
