package delivery

import "testing"

func TestDependencyCycle(t *testing.T) {
	edges := []Dependency{{PredecessorID: "a", SuccessorID: "b"}, {PredecessorID: "b", SuccessorID: "c"}}
	if Cycle(edges, "a", "c") != nil {
		t.Fatal("forward edge rejected")
	}
	if p := Cycle(edges, "c", "a"); len(p) != 4 || p[0] != "c" || p[3] != "c" {
		t.Fatalf("cycle %v", p)
	}
	if len(Cycle(nil, "a", "a")) != 2 {
		t.Fatal("self cycle")
	}
}
