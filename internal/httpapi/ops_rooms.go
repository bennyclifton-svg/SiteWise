package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// profile.add_room adds one line to the accommodation schedule the way the
// screen does: an insert at the end, then each field the PM gave. The
// schedule's lines are covered by Undo, so the whole thing is one change.

type roomLines struct {
	Lines []struct {
		ID    string `json:"id"`
		Space string `json:"space"`
		Name  struct {
			Value string `json:"value"`
		} `json:"name"`
		Level struct {
			Value string `json:"value"`
		} `json:"level"`
		Area struct {
			Value string `json:"value"`
		} `json:"area"`
		Comment struct {
			Value string `json:"value"`
		} `json:"comment"`
	} `json:"lines"`
}

func accommodationLines(ctx context.Context, d Deps, project string) (roomLines, error) {
	var out struct {
		Accommodation roomLines `json:"accommodation"`
	}
	status, body, err := callRoute(ctx, d, project, "GET /projects/{id}/profile", sent{})
	if err != nil {
		return roomLines{}, err
	}
	if status != 200 {
		return roomLines{}, routeRefusal(status, body)
	}
	return out.Accommodation, json.Unmarshal(body, &out)
}

func roomOperations() []operation {
	return []operation{{
		Name: "profile.add_room", About: "Add a space to the accommodation schedule.",
		Fields:  `name; level (optional); area (square metres, optional); comment (optional)`,
		Figures: map[string]string{"area": "number"},
		Run: func(ctx context.Context, d Deps, who opActor, project string, raw json.RawMessage) (string, error) {
			var f struct {
				Name    string `json:"name"`
				Level   string `json:"level"`
				Area    any    `json:"area"`
				Comment string `json:"comment"`
			}
			if err := decodeFields(raw, &f); err != nil {
				return "", err
			}
			if strings.TrimSpace(f.Name) == "" {
				return "", refuse("name", "A space needs a name.")
			}
			before, err := accommodationLines(ctx, d, project)
			if err != nil {
				return "", err
			}
			had := map[string]bool{}
			order := []string{}
			for _, l := range before.Lines {
				had[l.ID] = true
				order = append(order, l.ID)
			}
			post := func(body map[string]any) (roomLines, error) {
				var after roomLines
				status, resp, err := callRoute(ctx, d, project, "POST /projects/{id}/profile/accommodation", sent{Body: body})
				if err != nil {
					return after, err
				}
				if status < 200 || status > 299 {
					return after, routeRefusal(status, resp)
				}
				// The handler answers with the table, or with the profile holding it.
				var wrapped struct {
					Accommodation *roomLines `json:"accommodation"`
				}
				if json.Unmarshal(resp, &wrapped) == nil && wrapped.Accommodation != nil {
					return *wrapped.Accommodation, nil
				}
				return after, json.Unmarshal(resp, &after)
			}
			after, err := post(map[string]any{"op": "insert", "order": append(order, "new:0")})
			if err != nil {
				return "", err
			}
			id := ""
			for _, l := range after.Lines {
				if !had[l.ID] {
					id = l.ID
				}
			}
			if id == "" {
				return "", refuse("", "The new line could not be found after it was added.")
			}
			area := ""
			if f.Area != nil {
				area = strings.TrimSpace(fmt.Sprint(f.Area))
			}
			for _, set := range [][2]string{{"name", strings.TrimSpace(f.Name)}, {"level", strings.TrimSpace(f.Level)}, {"area", area}, {"comment", strings.TrimSpace(f.Comment)}} {
				if set[1] == "" {
					continue
				}
				if _, err := post(map[string]any{"op": "set", "lines": []string{id}, "field": set[0], "value": set[1]}); err != nil {
					return "", err
				}
			}
			said := fmt.Sprintf("Added %q to the accommodation schedule", strings.TrimSpace(f.Name))
			if f.Level != "" {
				said += ", " + strings.TrimSpace(f.Level)
			}
			if area != "" {
				said += ", " + area + " m²"
			}
			return said + ".", nil
		},
	}}
}

// readAccommodation renders the schedule's lines with their ids.
func chatAccommodation(ctx context.Context, d Deps, project string, r *reader) error {
	t, err := accommodationLines(ctx, d, project)
	if err != nil {
		return err
	}
	if len(t.Lines) == 0 {
		r.line("Accommodation schedule: no lines.")
		return nil
	}
	r.line("Accommodation schedule (id | space | level | area m² | comment). A dash is blank.")
	for _, l := range t.Lines {
		name := l.Name.Value
		if name == "" {
			name = l.Space
		}
		// A line's id may be a catalogue key, so it is given whole.
		id := ""
		if !r.plain {
			id = l.ID + " "
		}
		r.line("  %s%s | %s | %s | %s", id, name, orDash(l.Level.Value), orDash(l.Area.Value), orDash(l.Comment.Value))
	}
	return nil
}
