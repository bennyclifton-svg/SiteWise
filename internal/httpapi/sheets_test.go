package httpapi_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/httpapi"
	"sitewise/internal/identity"
	"sitewise/internal/jev"
	"sitewise/internal/store"
)

type sheetAsker struct {
	kind  string
	pages atomic.Int32
}

func (a *sheetAsker) Ask(ctx context.Context, call jev.Call) (jev.Result, error) {
	c := .99
	kind := a.kind
	if kind == "mixed" || kind == "drawing-schedule" {
		kind = "drawing"
		if call.Priority == jev.PriorityBackground && a.pages.Add(1) == 2 {
			kind = "report"
			if a.kind == "drawing-schedule" {
				kind = "schedule"
			}
		}
	}
	return jev.Result{Answers: map[string]jev.Answer{"kind": {Type: jev.TypeChoice, Choice: kind, Confidence: &c}}}, nil
}

func TestUploadDrawingSheetsAndReportRetention(t *testing.T) {
	for _, kind := range []string{"drawing", "drawing-schedule", "report", "schedule", "mixed"} {
		t.Run(kind, func(t *testing.T) {
			app := newApp(t, func(o *httpapi.Options) { o.Jev = &sheetAsker{kind: kind} })
			m := app.member(t, "Sheets "+kind)
			p := m.createProject(t, "Sheet integration")
			doc := m.upload(t, p, "rotated-title-block.pdf", http.StatusCreated)
			var list struct {
				Documents []store.DocumentView `json:"documents"`
			}
			deadline := time.Now().Add(8 * time.Second)
			for {
				m.getJSON(t, "/api/projects/"+p+"/documents", &list)
				done := false
				for _, d := range list.Documents {
					if d.ID == doc.ID {
						done = ((kind == "drawing" || kind == "drawing-schedule") && d.Status == "split") || ((kind == "report" || kind == "schedule") && d.Status == "filed")
						if kind == "mixed" {
							done = d.Expansion != nil && d.Expansion.Status == "review"
						}
					}
				}
				if done {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("not finished: %+v", list.Documents)
				}
				time.Sleep(10 * time.Millisecond)
			}
			want := 1
			if kind == "drawing" || kind == "drawing-schedule" {
				want = 3
			}
			if len(list.Documents) != want {
				t.Fatalf("%d documents want %d", len(list.Documents), want)
			}
			original, err := os.ReadFile(filepath.Join("..", "..", "testdata", "identity", "rotated-title-block.pdf"))
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range list.Documents {
				resp, err := m.client.Get(app.url + "/api/documents/" + d.ID + "/file")
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil || resp.StatusCode != 200 {
					t.Fatalf("download %d %v", resp.StatusCode, err)
				}
				if d.ID == doc.ID {
					if !bytes.Equal(body, original) {
						t.Fatal("source bytes changed")
					}
					continue
				}
				if d.SourceID != doc.ID || d.SheetPage < 1 || d.SheetTotal != 2 || d.SourceFilename != "rotated-title-block.pdf" {
					t.Fatalf("provenance %+v", d)
				}
				if kind == "drawing-schedule" && d.SheetPage == 2 {
					schedule := false
					for _, f := range d.Fields {
						schedule = schedule || f.Field == "kind" && f.Value == "schedule"
					}
					if !schedule {
						t.Fatal("schedule sheet was relabelled as drawing")
					}
				}
				text, err := identity.Extract(context.Background(), "pdf", bytes.NewReader(body), int64(len(body)), identity.DefaultLimits())
				if err != nil || text.PageCount != 1 {
					t.Fatalf("sheet download %+v %v", text, err)
				}
			}
			again := m.upload(t, p, "rotated-title-block.pdf", http.StatusOK)
			if again.ID != doc.ID {
				t.Fatal("duplicate source")
			}
			m.getJSON(t, "/api/projects/"+p+"/documents", &list)
			if len(list.Documents) != want {
				t.Fatal("duplicate children")
			}
			other := app.member(t, "Other tenant")
			if got := other.status(t, http.MethodGet, "/api/documents/"+doc.ID+"/file", nil); got != 404 {
				t.Fatal("cross-org download", got)
			}
		})
	}
}
