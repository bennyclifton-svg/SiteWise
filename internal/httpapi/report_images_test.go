package httpapi_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"strings"
	"testing"

	"sitewise/internal/files"
	"sitewise/internal/httpapi"
	"sitewise/internal/store"
)

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, x%h, color.RGBA{0x1F, 0x4E, 0xA8, 0xFF})
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// Slice 5 of the project report plan (Lane C): images are decoded, capped,
// and served only through their own report to its own org; an image whose
// bytes are gone holds issue, naming its cell.
func TestReportImages(t *testing.T) {
	blobs, err := files.Open(t.TempDir(), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	a := newApp(t, withProfile(t), func(o *httpapi.Options) { o.Blobs = blobs; o.MaxUploadBytes = 100 << 20 })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Images")
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	base := "/api/reports/" + report

	// A PNG: saved, with the line that shows it.
	var saved struct {
		SHA256 string `json:"sha256"`
		Line   string `json:"line"`
	}
	sketch := pngBytes(t, 120, 60)
	m.call(t, http.MethodPost, base+"/images?caption=Hydraulic+servery+sketch", sketch, http.StatusCreated, &saved)
	if len(saved.SHA256) != 64 || saved.Line != "![Hydraulic servery sketch](image:"+saved.SHA256+")" {
		t.Fatalf("saved %+v", saved)
	}
	// Served inline, as an image, to this report's org only.
	resp := m.do(t, http.MethodGet, base+"/images/"+saved.SHA256, nil)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("serve = %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	foreign := a.member(t, newUUID(t))
	if got := foreign.status(t, http.MethodGet, base+"/images/"+saved.SHA256, nil); got != http.StatusNotFound {
		t.Fatalf("foreign read = %d", got)
	}
	if got := foreign.status(t, http.MethodPost, base+"/images", sketch); got != http.StatusNotFound {
		t.Fatalf("foreign upload = %d", got)
	}
	// Refusals: not an image (whatever its name), too big, a hash this report does not hold.
	if got := m.status(t, http.MethodPost, base+"/images?caption=x.png", []byte("%PDF-1.4 not an image")); got != http.StatusUnprocessableEntity {
		t.Fatalf("a PDF = %d", got)
	}
	if got := m.status(t, http.MethodPost, base+"/images", bytes.Repeat([]byte{0x89}, store.ReportImageMaxBytes+10)); got != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversize = %d", got)
	}
	second := m.createProject(t, "Other")
	m.ops(t, second, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	other := projectReportID(t, m, second)
	if got := m.status(t, http.MethodGet, "/api/reports/"+other+"/images/"+saved.SHA256, nil); got != http.StatusNotFound {
		t.Fatalf("another report's hash = %d", got)
	}
	if got := m.status(t, http.MethodGet, base+"/images/nothex", nil); got != http.StatusNotFound {
		t.Fatalf("a bad hash = %d", got)
	}

	// In a cell, then issued: the PDF draws it.
	v := reportView(t, m, report)
	body, _ := json.Marshal(map[string]any{"table": "design", "key": draftTable(t, v.Draft, "design").Meta[0].Key, "cells": map[string]string{"1": "Servery moved.\n" + saved.Line}, "version": v.Draft.Version})
	var out rowReplyJSON
	m.call(t, http.MethodPost, base+"/rows", body, http.StatusOK, &out)
	// The bytes are gone from disk: issue is refused, naming the cell.
	raw, _ := hex.DecodeString(saved.SHA256)
	path, err := blobs.Path(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	v = reportView(t, m, report)
	ib, _ := json.Marshal(store.IssueOptions{Version: v.Draft.Version, ReportingDate: "2026-11-28", AcceptStale: true, StaleReason: "Fixture"})
	status, text := m.text(t, http.MethodPost, base+"/issue", ib)
	if status != http.StatusUnprocessableEntity || !strings.Contains(text, "report_image_missing") || !strings.Contains(text, "Design management") {
		t.Fatalf("issue with a missing image = %d %s", status, text)
	}
}

// With the bytes there, the issued PDF draws the image.
func TestReportImageIssues(t *testing.T) {
	blobs, err := files.Open(t.TempDir(), 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	a := newApp(t, withProfile(t), func(o *httpapi.Options) { o.Blobs = blobs; o.MaxUploadBytes = 100 << 20 })
	m := a.member(t, newUUID(t))
	project := m.createProject(t, "Images issue")
	m.ops(t, project, "reports", http.StatusOK, op("reports.update", map[string]any{}))
	report := projectReportID(t, m, project)
	base := "/api/reports/" + report
	var saved struct {
		Line string `json:"line"`
	}
	m.call(t, http.MethodPost, base+"/images?caption=Cover", pngBytes(t, 400, 200), http.StatusCreated, &saved)
	v := reportView(t, m, report)
	body, _ := json.Marshal(map[string]any{"table": "context", "cells": map[string]string{"0": "Site photo", "1": saved.Line}, "version": v.Draft.Version})
	m.call(t, http.MethodPost, base+"/rows", body, http.StatusOK, nil)
	v = reportView(t, m, report)
	ib, _ := json.Marshal(store.IssueOptions{Version: v.Draft.Version, ReportingDate: "2026-11-28", AcceptStale: true, StaleReason: "Fixture"})
	var issued store.IssuedReport
	m.call(t, http.MethodPost, base+"/issue", ib, http.StatusCreated, &issued)
	resp := m.do(t, http.MethodGet, base+"/versions/"+issued.ID+"/file", nil)
	defer resp.Body.Close()
	var pdf bytes.Buffer
	_, _ = pdf.ReadFrom(resp.Body)
	if resp.StatusCode != http.StatusOK || !bytes.HasPrefix(pdf.Bytes(), []byte("%PDF")) || !bytes.Contains(pdf.Bytes(), []byte("/Subtype /Image")) {
		t.Fatalf("issued PDF %d, %d bytes, has image %v", resp.StatusCode, pdf.Len(), bytes.Contains(pdf.Bytes(), []byte("/Subtype /Image")))
	}
}
