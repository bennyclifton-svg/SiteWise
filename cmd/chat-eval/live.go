package main

import (
	"errors"
	"slices"
	"strings"

	"sitewise/internal/chateval"
)

// pick keeps the cases whose id or group is named; an empty list keeps all.
func pick(cases []chateval.Case, only string) []chateval.Case {
	if strings.TrimSpace(only) == "" {
		return cases
	}
	want := strings.Split(only, ",")
	var out []chateval.Case
	for _, c := range cases {
		if slices.Contains(want, c.ID) || slices.Contains(want, c.Group) {
			out = append(out, c)
		}
	}
	return out
}

// liveAgent is built with the chat route (order of work, step 10).
func liveAgent(model string) (chateval.Agent, func(), error) {
	return nil, nil, errors.New("-live is not built yet")
}
