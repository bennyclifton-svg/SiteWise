package httpapi_test

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	"sitewise/internal/store"
)

// budget measures 25 authenticated reads of path and fails above p50/p90.
func budget(t *testing.T, m *member, path string, p50, p90 time.Duration, out any) {
	t.Helper()
	var durations []time.Duration
	for i := 0; i < 25; i++ {
		started := time.Now()
		m.getJSON(t, path, out)
		durations = append(durations, time.Since(started))
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	if durations[12] > p50 || durations[22] > p90 {
		t.Fatalf("%s exceeded p50/p90 budget %s/%s: %s/%s", path, p50, p90, durations[12], durations[22])
	}
}

type progressBody struct {
	Revision  *int64 `json:"revision"`
	Preparing *int   `json:"preparing"`
	Documents []struct {
		DocumentID string `json:"document_id"`
		Filename   string `json:"filename"`
		State      string `json:"state"`
		Pass       string `json:"pass"`
		Units      int    `json:"units"`
		Labelled   int    `json:"labelled"`
		Evidence   int    `json:"evidence"`
	} `json:"documents"`
}

func TestProfileProgressShowsEachDocumentsReading(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Run progress")
	path := "/api/projects/" + project + "/profile/progress"

	var empty progressBody
	budget(t, m, path, 50*time.Millisecond, 150*time.Millisecond, &empty)
	if empty.Revision == nil || empty.Preparing == nil || empty.Documents == nil || len(empty.Documents) != 0 {
		t.Fatalf("an idle project reports nothing to read: %+v", empty)
	}

	// A document whose label pass is queued shows as queued for that pass.
	doc := m.upload(t, project, "identity-page.pdf", http.StatusCreated)
	if err := a.store.EnqueueStage(context.Background(), m.orgID, doc.ID, store.JobKindLabel); err != nil {
		t.Fatal(err)
	}
	var got progressBody
	m.getJSON(t, path, &got)
	var found bool
	for _, d := range got.Documents {
		if d.DocumentID == doc.ID {
			found = true
			if d.State != "queued" || d.Pass != "label" || d.Filename != "identity-page.pdf" {
				t.Fatalf("queued label pass %+v", d)
			}
		}
	}
	if !found {
		t.Fatalf("the queued document is missing: %+v", got)
	}

	other := a.member(t, newUUID(t))
	if code := other.status(t, http.MethodGet, path, nil); code != http.StatusNotFound {
		t.Fatalf("foreign progress read = %d", code)
	}
}

func TestProfileFindsValidatesItsInputAndOrg(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Run finds")
	base := "/api/projects/" + project + "/profile/finds"

	var empty struct {
		Finds []any `json:"finds"`
	}
	budget(t, m, base+"?docs=&after=", 50*time.Millisecond, 150*time.Millisecond, &empty)
	if empty.Finds == nil || len(empty.Finds) != 0 {
		t.Fatalf("no documents, no finds: %+v", empty)
	}
	many := make([]string, 21)
	for i := range many {
		many[i] = newUUID(t)
	}
	for _, q := range []string{"docs=not-a-uuid", "after=yesterday", "docs=" + strings.Join(many, ",")} {
		if code := m.status(t, http.MethodGet, base+"?"+q, nil); code != http.StatusBadRequest {
			t.Fatalf("finds %s = %d", q, code)
		}
	}
	// A document of another project reads as nothing, not an error.
	var foreignDoc struct {
		Finds []any `json:"finds"`
	}
	m.getJSON(t, base+"?docs="+newUUID(t), &foreignDoc)
	if len(foreignDoc.Finds) != 0 {
		t.Fatalf("unknown document finds %+v", foreignDoc)
	}
	other := a.member(t, newUUID(t))
	if code := other.status(t, http.MethodGet, base, nil); code != http.StatusNotFound {
		t.Fatalf("foreign finds read = %d", code)
	}
}

func TestProfileNextCountsWhatAnUpdateChangesElsewhere(t *testing.T) {
	a := newApp(t, withProfile(t))
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Run next")
	path := "/api/projects/" + project + "/profile/next"

	var got struct {
		Procurement *int `json:"procurement"`
		Costs       *int `json:"costs"`
		Program     *int `json:"program"`
		Drafts      *int `json:"drafts"`
	}
	budget(t, m, path, 100*time.Millisecond, 250*time.Millisecond, &got)
	if got.Procurement == nil || got.Costs == nil || got.Program == nil || got.Drafts == nil {
		t.Fatalf("every count is present: %+v", got)
	}
	// The counts are the surfaces' own: what Procurement and the Cost Plan propose.
	var packages, plan struct {
		Suggestions []any `json:"suggestions"`
	}
	m.getJSON(t, "/api/projects/"+project+"/packages", &packages)
	m.getJSON(t, "/api/projects/"+project+"/cost-plan", &plan)
	if *got.Procurement != len(packages.Suggestions) || *got.Costs != len(plan.Suggestions) || *got.Program != 0 || *got.Drafts != 0 {
		t.Fatalf("next counts %d %d %d %d, want %d %d 0 0", *got.Procurement, *got.Costs, *got.Program, *got.Drafts, len(packages.Suggestions), len(plan.Suggestions))
	}
	other := a.member(t, newUUID(t))
	if code := other.status(t, http.MethodGet, path, nil); code != http.StatusNotFound {
		t.Fatalf("foreign next read = %d", code)
	}
}
