package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sitewise/internal/store"
	"sitewise/internal/works"
)

// Work-tree writes have separate samples so quick creates cannot mask a slow
// split or retirement. Setup is outside those samples; HTTP/rebuild is inside.
func (a *apiClient) workTree(ctx context.Context, st *store.Store, n int) error {
	project, err := a.createProject(ctx, "Work subdivision benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/works"
	for i := 0; i < n; i++ {
		part, err := st.CreatePart(ctx, benchOrg, project, fmt.Sprintf("Subdivision area %d", i), "part", "")
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{"part_id": part.ID, "system_id": "structure", "action": "investigate", "title": "Inspect structure"})
		raw, err := a.timed(ctx, "works_write", http.MethodPost, base, body, http.StatusCreated)
		if err != nil {
			return err
		}
		var parent works.Item
		if err = json.Unmarshal(raw, &parent); err != nil {
			return err
		}
		body, _ = json.Marshal(works.SplitRequest{Version: parent.Version, Children: []works.SplitChild{{Title: "Columns"}, {Title: "Beams"}}})
		raw, err = a.timed(ctx, "works_split", http.MethodPost, base+"/"+parent.ID+"/split", body, http.StatusOK)
		if err != nil {
			return err
		}
		var children []works.Item
		if err = json.Unmarshal(raw, &children); err != nil {
			return err
		}
		if len(children) != 2 {
			return fmt.Errorf("split benchmark expected 2 children, got %d", len(children))
		}
		for _, child := range children {
			if child.ParentID != parent.ID {
				return fmt.Errorf("split benchmark child lost parent")
			}
			body, _ = json.Marshal(map[string]int64{"version": child.Version})
			if _, err = a.timed(ctx, "works_retire", http.MethodPost, base+"/"+child.ID+"/retire", body, http.StatusOK); err != nil {
				return err
			}
		}
		body, _ = json.Marshal(map[string]int64{"version": parent.Version + 1})
		if _, err = a.timed(ctx, "works_retire", http.MethodPost, base+"/"+parent.ID+"/retire", body, http.StatusOK); err != nil {
			return err
		}
	}
	return nil
}

func (a *apiClient) layoutWork(ctx context.Context, st *store.Store, n int) error {
	project, err := a.createProject(ctx, "Partition layout benchmark")
	if err != nil {
		return err
	}
	base := "/projects/" + project + "/works"
	for i := 0; i < n; i++ {
		part, err := st.CreatePart(ctx, benchOrg, project, fmt.Sprintf("Partition area %d", i), "part", "")
		if err != nil {
			return err
		}
		body, _ := json.Marshal(map[string]string{"part_id": part.ID, "system_id": "interiors.walls-linings", "action": "alter", "title": "Alter partitions", "layout_change": "yes"})
		raw, err := a.timed(ctx, "works_write", http.MethodPost, base, body, http.StatusCreated)
		if err != nil {
			return err
		}
		var item works.Item
		if err = json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if item.LayoutChange != "yes" {
			return fmt.Errorf("layout benchmark lost explicit yes")
		}
		body, _ = json.Marshal(map[string]any{"version": item.Version, "layout_change": "no"})
		raw, err = a.timed(ctx, "works_write", http.MethodPatch, base+"/"+item.ID, body, http.StatusOK)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(raw, &item); err != nil {
			return err
		}
		if item.LayoutChange != "no" {
			return fmt.Errorf("layout benchmark lost explicit no")
		}
	}
	return nil
}
