package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestProfileRevisionJSONAndDurableEvent(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Revision contract")
	type state struct {
		Revision    int64    `json:"revision"`
		Fingerprint string   `json:"input_fingerprint"`
		Knowledge   string   `json:"knowledge_version"`
		StaleFor    []string `json:"stale_for"`
	}
	var before, after state
	hits := a.jevHits.Load()
	m.getJSON(t, "/api/projects/"+project+"/profile", &before)
	if before.Revision != 0 || len(before.StaleFor) == 0 {
		t.Fatal("unbuilt profile claimed freshness")
	}
	put(t, m, project, "hdr.subclass", map[string]any{"value": "house"}, http.StatusOK, &after)
	if after.Revision != 1 || len(after.Fingerprint) != 64 || len(after.Knowledge) != 64 || after.StaleFor == nil || len(after.StaleFor) != 0 {
		t.Fatalf("bad revision response: %+v", after)
	}
	events, err := a.store.EventsAfter(context.Background(), m.orgID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.Kind != "profile" {
			continue
		}
		var event struct {
			ProjectID string `json:"project_id"`
			Revision  int64  `json:"revision"`
		}
		if err := json.Unmarshal([]byte(e.Payload), &event); err != nil {
			t.Fatal(err)
		}
		if event.ProjectID == project && event.Revision == after.Revision {
			found = true
		}
	}
	if !found {
		t.Fatal("revision missing from durable SSE payload")
	}
	m.getJSON(t, "/api/projects/"+project+"/profile", &before)
	if before.Revision != after.Revision || before.Fingerprint != after.Fingerprint {
		t.Fatal("read rebuilt the profile")
	}
	if a.jevHits.Load() != hits {
		t.Fatal("read or edit called Jev")
	}
}
