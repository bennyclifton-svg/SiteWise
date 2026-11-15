package httpapi

import (
	"strings"
	"testing"
)

// Every write route is an operation, is excluded with a reason, or is
// pending with its area. A new write route fails here until it is classed.
func TestWriteRoutesHaveOperations(t *testing.T) {
	covered := map[string]bool{}
	for _, op := range operations() {
		for _, r := range op.Routes {
			covered[r] = true
		}
	}
	routes := apiRoutes()
	pending := 0
	for pattern := range routes {
		if strings.HasPrefix(pattern, "GET ") {
			continue
		}
		homes := 0
		if covered[pattern] {
			homes++
		}
		if noOperation[pattern] != "" {
			homes++
		}
		if pendingOperation[pattern] != "" {
			homes++
			pending++
		}
		if homes != 1 {
			t.Errorf("%s: want exactly one of operation, noOperation, pendingOperation; got %d", pattern, homes)
		}
	}
	for _, list := range []map[string]string{noOperation, pendingOperation} {
		for pattern := range list {
			if routes[pattern] == nil {
				t.Errorf("%s is listed but is not a route", pattern)
			}
		}
	}
	for pattern := range covered {
		if routes[pattern] == nil {
			t.Errorf("an operation names %s, which is not a route", pattern)
		}
	}
	t.Logf("%d write routes are pending an operation", pending)
}

// The registry is what the agent reads, so it must be well formed: one
// area per name, a way to run, and a figure kind the guard knows.
func TestOperationsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, op := range operations() {
		area := opArea(op.Name)
		ok := false
		for _, a := range opAreas {
			ok = ok || a == area
		}
		if !ok || seen[op.Name] || op.About == "" || op.Fields == "" {
			t.Errorf("%s: bad area, repeated name, or no description", op.Name)
		}
		seen[op.Name] = true
		if (op.Run == nil) == (op.Batch == nil) {
			t.Errorf("%s: wants exactly one of Run and Batch", op.Name)
		}
		for field, kind := range op.Figures {
			switch kind {
			case "money", "duration", "date", "number", "value":
			default:
				t.Errorf("%s.%s: unknown figure kind %q", op.Name, field, kind)
			}
		}
	}
}
