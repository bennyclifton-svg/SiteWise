package program

import (
	"testing"
)

func TestProgramAsksNoProfileQuestion(t *testing.T) {
	cat := testCatalog(t, testLibrary, "")
	in := GenInput{WorkTypes: []string{"new"}, Conditions: map[string]string{"planning": "exempt"},
		WorkItems:     []WorkItem{{ID: "wi", System: "structure", Action: "new", Title: "Frame"}},
		UserDurations: map[string]int{"skeleton:sk.new:procurement": 10}}
	q, err := OneQuestion(cat, in, nil)
	if err != nil || q != nil {
		t.Fatalf("question %+v %v", q, err)
	}
	q, err = OneQuestion(cat, GenInput{}, nil)
	if err != nil || q != nil {
		t.Fatalf("empty project question %+v %v", q, err)
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
