// Package bench holds the committed latency budgets. They are embedded so
// the speed page in the deployed binary judges against the same file the
// build gate uses.
package bench

import (
	_ "embed"
	"encoding/json"
	"errors"

	"sitewise/internal/latency"
)

//go:embed budgets.json
var budgetsJSON []byte

// Budgets parses the embedded budget file.
func Budgets() (latency.Budgets, error) {
	var b latency.Budgets
	if err := json.Unmarshal(budgetsJSON, &b); err != nil {
		return latency.Budgets{}, err
	}
	if b.MinSamples < 1 || len(b.Paths) == 0 {
		return latency.Budgets{}, errors.New("embedded budgets are empty")
	}
	return b, nil
}
