package intake

import (
	"bytes"
	"encoding/json"
	"sitewise/internal/identity"
	"testing"
)

func TestVisualCallDoesNotRepeatAllFieldsInState(t *testing.T) {
	c := visualCall(map[string][]VisualCandidate{"number": {{Value: "unique literal number", Page: 1}}}, identity.VisualEvidence{})
	raw, _ := json.Marshal(c.State)
	if bytes.Contains(raw, []byte("unique literal number")) {
		t.Fatal("candidate duplicated into shared state, exhausting image context")
	}
}

func TestVisualImageEvidencePrecedesTitleContext(t *testing.T) {
	c := visualCall(map[string][]VisualCandidate{FieldTitle: {{Value: "Sheet title", Page: 1}}}, identity.VisualEvidence{Images: []identity.VisualImage{{Label: "Whole page", Page: 1, Data: "fixture"}}})
	raw, err := json.Marshal(c.State)
	if err != nil || !json.Valid(raw) {
		t.Fatal("invalid state", err)
	}
	if bytes.Index(raw, []byte("page_1_Whole page")) > bytes.Index(raw, []byte("candidate_title_lines")) {
		t.Fatal("image evidence must precede candidate context")
	}
}

func TestVisualCandidatesJoinRotatedLiteralLinesAndExcludeUnreadPages(t *testing.T) {
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "LOWER GROUND", Source: identity.Source{Page: 1, Rotation: 270, X: 10, Y: 20, Width: 10, Height: 90}},
		{Text: "FLOOR PLAN", Source: identity.Source{Page: 1, Rotation: 270, X: 22, Y: 20, Width: 10, Height: 70}},
		{Text: "OTHER SHEET", Source: identity.Source{Page: 2}},
	}}
	candidates := visualCandidates("invented filename title.pdf", text, identity.VisualEvidence{Pages: 1})
	joined := false
	for _, c := range candidates[FieldTitle] {
		if c.Value == "LOWER GROUND FLOOR PLAN" {
			joined = true
		}
		if c.Page > 1 || c.Value == "invented filename title" {
			t.Fatal("unsupported evidence", c)
		}
	}
	if !joined {
		t.Fatal("rotated multiline title absent")
	}
}

func TestVisualCandidatesJoinMixedSizeReportTitle(t *testing.T) {
	text := identity.Text{TextLayer: true, Runs: []identity.Run{
		{Text: "ACCESSIBILTY", Source: identity.Source{Page: 1, X: 217.42, Y: 552.07, Width: 148, Height: 20.96}},
		{Text: "Design Review and Report", Source: identity.Source{Page: 1, X: 217.44, Y: 534.70, Width: 192, Height: 14.69}},
	}}
	for _, c := range visualCandidates("cover.pdf", text, identity.VisualEvidence{Pages: 1})[FieldTitle] {
		if c.Value == "ACCESSIBILTY Design Review and Report" {
			return
		}
	}
	t.Fatal("mixed font-size literal title missing")
}
