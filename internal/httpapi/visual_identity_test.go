package httpapi_test

import (
	"net/http"
	"sitewise/internal/store"
	"testing"
)

func TestVisualReviewEndpointIsolation(t *testing.T) {
	a := newApp(t)
	owner := a.member(t, "Visual owner")
	other := a.member(t, "Other org")
	p := owner.createProject(t, "Review")
	stream := owner.events(t, 0)
	doc := owner.upload(t, p, "identity-page.pdf", http.StatusCreated)
	stream.next(t, "filing", doc.ID)
	var review store.VisualReview
	owner.getJSON(t, "/api/documents/"+doc.ID+"/visual-review", &review)
	if review.Status != "queued" {
		t.Fatalf("new PDF not queued: %s", review.Status)
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		if got := other.status(t, method, "/api/documents/"+doc.ID+"/visual-review", nil); got != 404 {
			t.Fatalf("other org %s: %d", method, got)
		}
		if got := owner.status(t, method, "/api/documents/not-a-uuid/visual-review", nil); got != 404 {
			t.Fatalf("bad id %d", got)
		}
	}
	if got := owner.status(t, http.MethodPost, "/api/documents/"+doc.ID+"/visual-review", nil); got != 200 {
		t.Fatal(got)
	}
}
