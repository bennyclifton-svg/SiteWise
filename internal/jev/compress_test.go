package jev_test

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"sitewise/internal/config"
	"sitewise/internal/jev"
)

// seen is what the fake System One received: the encoding header and the
// request JSON after undoing that encoding.
type seen struct {
	encoding string
	body     map[string]any
}

func compressionServer(t *testing.T, got *seen) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.encoding = r.Header.Get("Content-Encoding")
		var in io.Reader = r.Body
		if got.encoding == "gzip" {
			zr, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Errorf("gzip body: %v", err)
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			in = zr
		}
		if err := json.NewDecoder(in).Decode(&got.body); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "success.json"))
	})
}

// longCall is a fan-out whose JSON body is well over 1 KiB, like a profile
// label call.
func longCall() jev.Call {
	call := sampleCall()
	call.State = map[string]string{"title_block": canaryDocument + strings.Repeat(" Footing F1 to engineer's detail.", 80)}
	return call
}

// A label call is ~25 KB of JSON, and this machine uploads ~600 KB/s, so
// plain bodies capped a profile read at ~25 calls a second. System One takes
// a gzip body (https://docs.typesafe.ai/api/errors.md, Compression).
func TestCompressedClientSendsLargeBodiesAsGzip(t *testing.T) {
	var got seen
	c := startClient(t, compressionServer(t, &got), jev.Options{CompressRequests: true})
	res, err := c.Ask(context.Background(), longCall())
	if err != nil {
		t.Fatal(err)
	}
	if got.encoding != "gzip" {
		t.Fatalf("Content-Encoding %q, want gzip", got.encoding)
	}
	state, _ := got.body["state"].(map[string]any)
	if got.body["model"] != config.PinnedJevModel || !strings.HasPrefix(state["title_block"].(string), canaryDocument) {
		t.Fatalf("decoded body %v", got.body)
	}
	if res.Answers["kind"].Choice != "drawing" {
		t.Fatalf("answers %+v", res.Answers)
	}
}

// Under 1 KiB compression saves nothing worth a header, as in TypeSafe's own
// clients; and a client without the option (replay and recording tools,
// which hash the plain body) sends what it always sent.
func TestSmallBodiesAndPlainClientsStayUncompressed(t *testing.T) {
	for name, tc := range map[string]struct {
		opt  jev.Options
		call jev.Call
	}{
		"small body":     {jev.Options{CompressRequests: true}, sampleCall()},
		"option not set": {jev.Options{}, longCall()},
	} {
		var got seen
		c := startClient(t, compressionServer(t, &got), tc.opt)
		if _, err := c.Ask(context.Background(), tc.call); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.encoding != "" || got.body["model"] != config.PinnedJevModel {
			t.Fatalf("%s: encoding %q body %v", name, got.encoding, got.body)
		}
	}
}
