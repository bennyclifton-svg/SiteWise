package works

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sitewise/internal/knowledge"
	"strings"
)

func targetType(cat *knowledge.Catalog, key string) (knowledge.PlanningKey, bool) {
	if id, ok := strings.CutPrefix(key, "plan."); ok {
		return cat.PlanningKey(id)
	}
	var d knowledge.Determinant
	var ok bool
	if id, found := strings.CutPrefix(key, "det."); found {
		d, ok = cat.Determinant(id)
	}
	if id, found := strings.CutPrefix(key, "fact."); found {
		d, ok = cat.ProjectFact(id)
	}
	if ok && !d.Deprecated() {
		k := knowledge.PlanningKey{Key: key, Value: d.Value, Unit: d.Unit}
		for _, o := range d.Options {
			k.Options = append(k.Options, knowledge.PlanningOption{ID: o.ID, Label: o.Describes})
		}
		return k, true
	}
	if id, found := strings.CutPrefix(key, "hdr.scale."); found {
		for _, field := range cat.ScaleFields() {
			if field.Key == id {
				return knowledge.PlanningKey{Key: key, Value: field.Type, Unit: field.Unit}, true
			}
		}
	}
	return knowledge.PlanningKey{}, false
}

func validateTargetValue(cat *knowledge.Catalog, v TargetValue) error {
	key, ok := targetType(cat, v.Key)
	if !ok {
		return fmt.Errorf("target key is not registered")
	}
	dec := json.NewDecoder(bytes.NewReader(v.Value))
	dec.UseNumber()
	var scalar any
	if err := dec.Decode(&scalar); err != nil {
		return fmt.Errorf("invalid target value")
	}
	var text string
	switch value := scalar.(type) {
	case string:
		text = value
	case json.Number:
		text = value.String()
	case bool:
		text = fmt.Sprint(value)
	default:
		return fmt.Errorf("target value must be a scalar")
	}
	if key.Value == "multi_choice" {
		key.Value = "choice"
		for _, value := range strings.Split(text, ",") {
			if msg := key.CheckPlanningValue(strings.TrimSpace(value)); msg != "" {
				return fmt.Errorf("%s", msg)
			}
		}
	} else if msg := key.CheckPlanningValue(text); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	if v.Unit != "" && key.Unit != v.Unit {
		return fmt.Errorf("target unit does not match its key")
	}
	return nil
}
