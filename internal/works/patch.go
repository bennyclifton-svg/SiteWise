package works

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Patch corrects planning fields without changing the identity or physical owner.
// A null quantity clears its unit too; omitted fields retain their saved values.
type Patch struct {
	PartID                *string         `json:"part_id,omitempty"`
	SystemID              *string         `json:"system_id,omitempty"`
	LayoutChange          *string         `json:"layout_change,omitempty"`
	Version               int64           `json:"version"`
	Action                *string         `json:"action,omitempty"`
	Title                 *string         `json:"title,omitempty"`
	Inclusion             *string         `json:"inclusion,omitempty"`
	ExistingConditionNote *string         `json:"existing_condition_note,omitempty"`
	Target                *Target         `json:"target,omitempty"`
	Quantity              json.RawMessage `json:"quantity,omitempty"`
	Unit                  *string         `json:"unit,omitempty"`
}

func (p Patch) Apply(item *Item) error {
	if p.Version < 1 || (p.LayoutChange == nil && p.PartID == nil && p.SystemID == nil && p.Action == nil && p.Title == nil && p.Inclusion == nil && p.ExistingConditionNote == nil && p.Target == nil && len(p.Quantity) == 0 && p.Unit == nil) {
		return fmt.Errorf("version and a correction are required")
	}
	for dst, src := range map[*string]*string{&item.LayoutChange: p.LayoutChange, &item.PartID: p.PartID, &item.SystemID: p.SystemID, &item.Action: p.Action, &item.Title: p.Title, &item.Inclusion: p.Inclusion, &item.ExistingConditionNote: p.ExistingConditionNote} {
		if src != nil {
			*dst = strings.TrimSpace(*src)
		}
	}
	if item.Title == "" || (item.Inclusion != "included" && item.Inclusion != "excluded") {
		return fmt.Errorf("title and valid inclusion are required")
	}
	if p.Target != nil {
		item.Target = *p.Target
	}
	if p.Unit != nil {
		unit := strings.TrimSpace(*p.Unit)
		item.Unit = &unit
	}
	if len(p.Quantity) > 0 {
		if !json.Valid(p.Quantity) {
			return fmt.Errorf("invalid quantity")
		}
		dec := json.NewDecoder(bytes.NewReader(p.Quantity))
		dec.UseNumber()
		var value any
		if err := dec.Decode(&value); err != nil {
			return fmt.Errorf("invalid quantity")
		}
		switch v := value.(type) {
		case nil:
			if p.Unit != nil && *p.Unit != "" {
				return fmt.Errorf("a cleared quantity cannot have a unit")
			}
			item.Quantity, item.Unit = nil, nil
		case string:
			q := strings.TrimSpace(v)
			item.Quantity = &q
		case json.Number:
			q := v.String()
			item.Quantity = &q
		default:
			return fmt.Errorf("quantity must be a decimal or null")
		}
	}
	return nil
}
