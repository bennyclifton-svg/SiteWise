package reports

import (
	"fmt"
	"strings"
)

// These are the saved shapes produced by the shared assembler. Keep the
// explanation tied to the actual inputs, without consulting today's catalogue.
func calculationDetails(source map[string]any) []string {
	parts := []string{}
	text := func(v any) string {
		if v == nil {
			return "not recorded"
		}
		return fmt.Sprint(v)
	}
	if clause, ok := source["clause"].(map[string]any); ok {
		parts = append(parts, fmt.Sprintf("Catalogue clause: %s; version: %s; review: %s", text(clause["ID"]), text(clause["Version"]), text(clause["Status"])))
	}
	if derived, ok := source["Derived"].(map[string]any); ok {
		parts = append(parts, "Rule: "+text(derived["Rule"]), "Result: "+text(derived["State"]))
		if reason, ok := derived["Reason"].(string); ok && reason != "" {
			parts = append(parts, reason)
		}
	}
	if finding, ok := source["finding"].(map[string]any); ok {
		parts = append(parts, fmt.Sprintf("Allocation input: work %s; role %s; state %s", text(finding["work_item_id"]), text(finding["role"]), text(finding["state"])))
	}
	if stage, ok := source["stage"].(map[string]any); ok {
		parts = append(parts, "Stage: "+text(stage["label"]), "Fee requested from supplier; no amount calculated")
	}
	if edge, ok := source["interface"].(map[string]any); ok {
		parts = append(parts, "Interface: "+text(edge["id"])+" — "+text(edge["summary"]), "Interface review: "+text(edge["status"]))
	}
	if reason, ok := source["reason"].(map[string]any); ok {
		parts = append(parts, "Knowledge record: "+text(reason["record"]))
		if triggers, ok := reason["triggers"].([]any); ok {
			for _, v := range triggers {
				if trigger, ok := v.(map[string]any); ok {
					parts = append(parts, fmt.Sprintf("Trigger: %s %s in %s (work %s)", text(trigger["action"]), text(trigger["system"]), text(trigger["part"]), text(trigger["work_item_id"])))
				}
			}
		}
		if values, ok := reason["determinants"].([]any); ok {
			for _, v := range values {
				if value, ok := v.(map[string]any); ok {
					parts = append(parts, fmt.Sprintf("Input: %s = %s (%s)", text(value["key"]), text(value["value"]), text(value["origin"])))
				}
			}
		}
	}
	// Accepted calculations retain the exact proposal in provenance.
	if provenance, ok := source["provenance"].(map[string]any); ok {
		if proposal, ok := provenance["proposal"].(map[string]any); ok {
			parts = append(parts, calculationDetails(proposal)...)
		}
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
