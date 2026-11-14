package tender

import (
	"encoding/json"
	"errors"
	"strings"
)

const maxLines = 600

var (
	validG     = map[string]bool{"ex": true, "inc": true, "unknown": true}
	validSt    = map[string]bool{"priced": true, "included": true, "excluded": true, "na": true, "tbc": true}
	validKind  = map[string]bool{"base": true, "ps": true, "pc": true, "option": true, "saving": true, "rate": true, "subtotal": true, "total": true, "gst": true}
	validWhere = map[string]bool{"sum": true, "inside": true, "outside": true}
	validBasis = map[string]bool{"fixed": true, "estimate": true, "cost_plus": true, "rates_only": true, "unknown": true}
)

// ParseAnswer decodes the model's JSON. cut is true when the model ran out
// of tokens: then whatever lines were complete are kept. dropped counts
// lines that were left out.
func ParseAnswer(text string, cut bool) (raw RawReading, dropped int, err error) {
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		if !cut {
			return RawReading{}, 0, errors.New("tender: the answer is not valid JSON")
		}
		raw = salvage(text)
	}
	if !validBasis[raw.Basis] {
		raw.Basis = "unknown"
	}
	if !validG[raw.GST] {
		raw.GST = "unknown"
	}
	kept := raw.Lines[:0]
	for _, l := range raw.Lines {
		if !validG[l.G] || !validSt[l.St] || !validKind[l.Kind] || !validWhere[l.Where] || len(kept) >= maxLines {
			dropped++
			continue
		}
		l.Label, l.Note, l.Group, l.Amt, l.Row = clip(strings.TrimSpace(l.Label), 500), clip(l.Note, 300), clip(l.Group, 300), clip(strings.TrimSpace(l.Amt), 80), strings.TrimSpace(l.Row)
		kept = append(kept, l)
	}
	raw.Lines = kept
	raw.Firm = clip(raw.Firm, 200)
	if len(raw.Exclusions) > 15 {
		raw.Exclusions = raw.Exclusions[:15]
	}
	if len(raw.Notes) > 6 {
		raw.Notes = raw.Notes[:6]
	}
	return raw, dropped, nil
}

// salvage reads the fields that arrived before the answer was cut. The
// schema puts firm, basis and gst before lines.
func salvage(text string) RawReading {
	var raw RawReading
	dec := json.NewDecoder(strings.NewReader(text))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return raw
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return raw
		}
		switch key {
		case "firm":
			_ = dec.Decode(&raw.Firm)
		case "basis":
			_ = dec.Decode(&raw.Basis)
		case "gst":
			_ = dec.Decode(&raw.GST)
		case "lines":
			if t, err := dec.Token(); err != nil || t != json.Delim('[') {
				return raw
			}
			for dec.More() {
				var l RawLine
				if dec.Decode(&l) != nil {
					return raw
				}
				raw.Lines = append(raw.Lines, l)
			}
			return raw
		default:
			var skip json.RawMessage
			if dec.Decode(&skip) != nil {
				return raw
			}
		}
	}
	return raw
}
