package tender

import _ "embed"

// PromptVersion changes whenever prompt.txt or Schema changes. A reading
// stores the version that produced it.
const PromptVersion = "t2"

//go:embed prompt.txt
var Prompt string

func enum(values ...string) map[string]any {
	return map[string]any{"type": "string", "enum": values}
}

// Schema is the JSON schema the answer must follow.
func Schema() map[string]any {
	str := map[string]any{"type": "string"}
	line := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"p": map[string]any{"type": "integer"}, "label": str, "amt": str,
			"g":     enum("ex", "inc", "unknown"),
			"st":    enum("priced", "included", "excluded", "na", "tbc"),
			"kind":  enum("base", "ps", "pc", "option", "saving", "rate", "subtotal", "total", "gst"),
			"where": enum("sum", "inside", "outside"),
			"group": str, "row": str, "note": str,
			"sure": map[string]any{"type": "boolean"},
		},
		"required":             []string{"p", "label", "amt", "g", "st", "kind", "where", "group", "row", "note", "sure"},
		"additionalProperties": false,
	}
	exclusion := map[string]any{
		"type":                 "object",
		"properties":           map[string]any{"p": map[string]any{"type": "integer"}, "text": str},
		"required":             []string{"p", "text"},
		"additionalProperties": false,
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"firm":       str,
			"basis":      enum("fixed", "estimate", "cost_plus", "rates_only", "unknown"),
			"gst":        enum("ex", "inc", "unknown"),
			"lines":      map[string]any{"type": "array", "items": line},
			"exclusions": map[string]any{"type": "array", "items": exclusion},
			"notes":      map[string]any{"type": "array", "items": str},
		},
		"required":             []string{"firm", "basis", "gst", "lines", "exclusions", "notes"},
		"additionalProperties": false,
	}
}
