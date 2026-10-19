package program

import (
	"reflect"
	"testing"
)

func TestOneQuestionPicksTheWidestNarrowing(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	// sk.new applies ([procurement, construction]); the only activity is the
	// frame, conditional on flood_hazard: low 0, typical 0, high 8. The
	// procurement stage is given a duration so the finish is known.
	in := GenInput{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "exempt"},
		WorkItems:     []WorkItem{{ID: "wi", System: "structure", Action: "new", Title: "Frame"}},
		UserDurations: map[string]int{"skeleton:sk.new:procurement": 10}}
	q, err := OneQuestion(cat, in, nil)
	if err != nil || q == nil {
		t.Fatalf("no question: %v", err)
	}
	// state (sk.state) comes first in determinants.yaml but changes nothing
	// here; flood_hazard narrows the range from 8 to the worst case 5 (Yes:
	// 3 to 8), so it wins on score, not order.
	if q.Key != "flood_hazard" || q.Kind != QuestionDeterminant || q.NarrowsByDays != 3 {
		t.Fatalf("question %+v", q)
	}
	if !reflect.DeepEqual(q.Options, []string{"Yes", "No", NotSure}) || !reflect.DeepEqual(q.Values, []string{"true", "false"}) {
		t.Fatalf("options %v values %v", q.Options, q.Values)
	}
	if q.Label != "Site in a flood hazard area" {
		t.Fatal(q.Label)
	}

	// Dismissed ("Not sure"): the next referenced unknown is offered even
	// though it narrows nothing, ordered by determinants.yaml.
	q, err = OneQuestion(cat, in, map[string]bool{"flood_hazard": true})
	if err != nil || q == nil || q.Key != "state" || q.NarrowsByDays != 0 {
		t.Fatalf("fallback question %+v %v", q, err)
	}
	if len(q.Options) != len(q.Values) || len(q.Options) == 0 {
		t.Fatalf("choice options %v", q.Options)
	}
}

func TestOneQuestionOrderWhenNothingKnown(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	// Nothing is known and no finish is known, so every score is 0 and
	// order decides: determinants.yaml lists state, flood_hazard, then
	// work_type; conditions come after every determinant.
	q, err := OneQuestion(cat, GenInput{}, nil)
	if err != nil || q == nil {
		t.Fatal(err)
	}
	if q.Key != "state" || q.NarrowsByDays != 0 {
		t.Fatalf("question %+v", q)
	}
	q, _ = OneQuestion(cat, GenInput{}, map[string]bool{"state": true, "flood_hazard": true})
	if q.Key != "work_type" || q.Kind != QuestionWorkType || len(q.Options) != len(cat.Taxonomy().WorkTypes) {
		t.Fatalf("work type question %+v", q)
	}
	q, _ = OneQuestion(cat, GenInput{}, map[string]bool{"state": true, "flood_hazard": true, "work_type": true})
	if q.Key != "planning" || q.Kind != QuestionCondition {
		t.Fatalf("condition question %+v", q)
	}
}

func TestOneQuestionAnswerIsIndependent(t *testing.T) {
	// answer must not write into the caller's maps.
	in := GenInput{Values: map[string]string{"a": "1"}}
	out := answer(in, Question{Key: "b", Kind: QuestionDeterminant}, "2")
	if _, ok := in.Values["b"]; ok || out.Values["b"] != "2" || out.Values["a"] != "1" {
		t.Fatal("answer shares the input map")
	}
}
