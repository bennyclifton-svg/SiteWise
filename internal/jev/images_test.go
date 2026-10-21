package jev

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImageContractAndModelIsolation(t *testing.T) {
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 280, 280)), nil); err != nil {
		t.Fatal(err)
	}
	asset := ImageAsset{Type: "image", Format: "jpg", Data: base64.StdEncoding.EncodeToString(b.Bytes())}
	ref := "00000000-0000-4000-8000-000000000001"
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("TypeSafe-Danger-Zone-Images") != "true" {
			t.Error("missing image opt-in")
		}
		w.Write([]byte(`{"model":"jev-1.14.0","answers":{}}`))
	}))
	defer srv.Close()
	c, err := New(Options{APIKey: "test", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	call := Call{State: map[string]string{"page": ref}, Images: map[string]ImageAsset{ref: asset}, Priority: PriorityBackground, Questions: map[string]Question{"title": {Type: TypeNoul, Instructions: "Check source"}}}
	if _, err = c.Ask(context.Background(), call); err != nil {
		t.Fatal(err)
	}
	call.Priority = PriorityInteractive
	if _, err = c.Ask(context.Background(), call); err == nil {
		t.Fatal("interactive image accepted")
	}
	call.Priority = PriorityBackground
	call.Images[ref] = ImageAsset{Type: "image", Format: "jpg", Data: "bad"}
	if _, err = c.Ask(context.Background(), call); err == nil {
		t.Fatal("malformed image accepted")
	}
	if hits != 1 {
		t.Fatalf("invalid images reached transport: %d", hits)
	}
	if c.callModel(Call{}) != "jev-1.13.0" {
		t.Fatal("text model changed")
	}
}
