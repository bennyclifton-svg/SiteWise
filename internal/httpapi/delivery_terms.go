package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"sitewise/internal/store"
)

// termsPatch reads the settings body: rfi_response_days, eot_response_days
// and agreed_completion, each optional, null to clear. Anything else is
// refused, as the strict reader refuses unknown fields elsewhere.
func termsPatch(body map[string]json.RawMessage) (store.TermsPatch, error) {
	var p store.TermsPatch
	for k, raw := range body {
		null := bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
		switch k {
		case "rfi_response_days", "eot_response_days":
			o := store.Optional[int]{Present: true}
			if !null {
				var n int
				if err := json.Unmarshal(raw, &n); err != nil {
					return p, fmt.Errorf("%s is a whole number of days", k)
				}
				o.Value = &n
			}
			if k == "rfi_response_days" {
				p.RFIDays = o
			} else {
				p.EOTDays = o
			}
		case "agreed_completion":
			o := store.Optional[string]{Present: true}
			if !null {
				var d string
				if err := json.Unmarshal(raw, &d); err != nil {
					return p, fmt.Errorf("agreed_completion is a date, YYYY-MM-DD")
				}
				o.Value = &d
			}
			p.AgreedCompletion = o
		default:
			return p, fmt.Errorf("unknown field %q", k)
		}
	}
	return p, nil
}

// termsDone is delivery.settings's sentence, naming what was set.
func termsDone(f map[string]any) string {
	var parts []string
	say := func(k, set, cleared string) {
		v, ok := f[k]
		if !ok {
			return
		}
		if v == nil {
			parts = append(parts, cleared)
			return
		}
		parts = append(parts, fmt.Sprintf(set, v))
	}
	say("rfi_response_days", "the RFI response time to %v working days", "the RFI response time back to the default")
	say("eot_response_days", "the EOT response time to %v days", "the EOT response time back to the default")
	say("agreed_completion", "the agreed completion date to %v", "the agreed completion date cleared")
	if len(parts) == 0 {
		return "Set the project's terms."
	}
	return "Set " + strings.Join(parts, "; ") + "."
}
