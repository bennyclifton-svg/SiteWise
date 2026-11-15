package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"sitewise/internal/knowledge"
)

func profileOperations() []operation {
	return []operation{
		{Name: "profile.set_value", About: "Set one profile value as the user's own, mark it unknown, or reset it to what the documents say.",
			Fields:  `key (from a profile read, e.g. hdr.scale.site_area); value (text or a plain number, no unit), or unknown: true, or reset: true; note (optional)`,
			Figures: map[string]string{"value": "value"}, Routes: []string{"PUT /projects/{id}/profile/{key}"}, Run: profileSetValue},
	}
}

func profileSetValue(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
	var f struct {
		Key     string  `json:"key"`
		Part    string  `json:"part_id"`
		Value   *string `json:"value"`
		Note    string  `json:"note"`
		Unknown bool    `json:"unknown"`
		Reset   bool    `json:"reset"`
	}
	if err := decodeFields(raw, &f); err != nil {
		return "", err
	}
	if d.Knowledge == nil {
		return "", refuse("", "The profile is not available.")
	}
	n := 0
	for _, set := range []bool{f.Value != nil, f.Unknown, f.Reset} {
		if set {
			n++
		}
	}
	if n != 1 {
		return "", refuse("value", "Give exactly one of value, unknown: true or reset: true.")
	}
	// The user's word: a favourable planning value is accepted only so.
	in := profileValueInput{PartID: f.Part, Value: f.Value, Note: f.Note, Unknown: f.Unknown, Reset: f.Reset, Origin: "user", Meaning: "stated"}
	_, status, msg := setProfileValue(ctx, d, who, project, f.Key, in)
	switch status {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", refuse("key", "There is no profile value with key %q. Use a key from a profile read.", f.Key)
	case http.StatusUnprocessableEntity:
		return "", refuse("value", "%s", profileRefusal(d.Knowledge, f.Key, msg))
	default:
		return "", &opError{Message: "The profile value could not be saved.", status: status}
	}
	label := profileLabel(d.Knowledge, f.Key)
	switch {
	case f.Reset:
		return fmt.Sprintf("Reset %s to what the documents say.", label), nil
	case f.Unknown:
		return fmt.Sprintf("Marked %s unknown.", label), nil
	}
	return fmt.Sprintf("Set %s to %s.", label, *f.Value), nil
}

// profileRefusal keeps the handler's own message and says it plainly when
// the key itself is not one the profile holds.
func profileRefusal(cat *knowledge.Catalog, key, msg string) string {
	if profileLabel(cat, key) == key && !strings.Contains(msg, "value") {
		return "There is no profile value with key " + key + "."
	}
	return strings.ToUpper(msg[:1]) + msg[1:] + "."
}

// profileLabel is the screen's name for a profile key, or the key.
func profileLabel(cat *knowledge.Catalog, key string) string {
	switch {
	case strings.HasPrefix(key, "hdr.scale."):
		for _, f := range cat.ScaleFields() {
			if "hdr.scale."+f.Key == key {
				return f.Label
			}
		}
	case strings.HasPrefix(key, knowledge.DetailPrefix):
		if d, ok := cat.ProjectDetail(strings.TrimPrefix(key, knowledge.DetailPrefix)); ok {
			return d.Label
		}
	case strings.HasPrefix(key, knowledge.PlanningPrefix):
		if pk, ok := cat.PlanningKey(strings.TrimPrefix(key, knowledge.PlanningPrefix)); ok {
			return pk.Label
		}
	case key == "hdr.building_class":
		return "Building class"
	case key == "hdr.work_type":
		return "Work type"
	}
	return key
}
