package reports

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RFPBudgetBlockID carries a consultant RFP's internal construction budget.
const RFPBudgetBlockID = "rfp:budget"

// IssueContent strips undisclosed pricing from BOTH rendered wording and its
// citation basis. Hiding an amount on the page alone would leak it in snapshots.
func IssueContent(sections []Section, kind string, disclose bool) ([]Section, []Reference, error) {
	raw, err := json.Marshal(sections)
	if err != nil {
		return nil, nil, err
	}
	var out []Section
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, nil, err
	}
	issued := out[:0]
	for _, section := range out {
		if !section.Internal {
			issued = append(issued, section)
		}
	}
	out = issued
	for i := range out {
		// A consultant RFP's budget block is internal unless disclosed.
		if !disclose && kind != "pmp" {
			kept := out[i].Blocks[:0]
			for _, b := range out[i].Blocks {
				if b.ID != RFPBudgetBlockID {
					kept = append(kept, b)
				}
			}
			out[i].Blocks = kept
		}
		for j := range out[i].Blocks {
			b := &out[i].Blocks[j]
			if !strings.HasPrefix(b.ID, "cost:line:") {
				continue
			}
			var basis map[string]json.RawMessage
			if err = json.Unmarshal(b.Basis, &basis); err != nil {
				return nil, nil, err
			}
			budget := basis["budget"]
			delete(basis, "budget")
			if disclose || kind == "pmp" {
				basis["budget"] = budget
				var value struct {
					Amount *string `json:"amount"`
					Low    *string `json:"low"`
					High   *string `json:"high"`
				}
				if err := json.Unmarshal(budget, &value); err != nil {
					return nil, nil, err
				}
				if value.Amount != nil {
					b.Text += " Internal budget disclosed: " + *value.Amount + "."
				} else if value.Low != nil && value.High != nil {
					b.Text += fmt.Sprintf(" Internal budget range disclosed: %s to %s.", *value.Low, *value.High)
				} else {
					b.Text += " Internal budget not available."
				}
			}
			b.Basis, err = json.Marshal(basis)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return Cite(out)
}
