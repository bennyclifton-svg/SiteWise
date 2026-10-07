package jev_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sitewise/internal/jev"
	"strings"
	"testing"
)

func TestOversizedRequestsNeverReachTransport(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client, err := jev.New(jev.Options{APIKey: "size-test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	for _, largeState := range []bool{true, false} {
		call := jev.Call{State: "short", Questions: map[string]jev.Question{}, Priority: jev.PriorityBackground}
		if largeState {
			call.State = strings.Repeat("private source ", 10000)
		}
		count := 1
		if !largeState {
			count = 1000
		}
		for i := 0; i < count; i++ {
			call.Questions[strings.Repeat("q", i+1)] = jev.Question{Type: jev.TypeNoul, Instructions: "Does the text name an existing system?"}
		}
		_, err := client.Ask(context.Background(), call)
		if !errors.Is(err, jev.ErrRequestLimit) || called {
			t.Fatalf("size gate failed: called=%v err=%v", called, err)
		}
		if strings.Contains(err.Error(), "private source") {
			t.Fatal("error leaks passage")
		}
	}
}

func TestSmallRequestSizeRetainsEveryQuestion(t *testing.T) {
	call := jev.Call{State: "text", Questions: map[string]jev.Question{"one": {Type: jev.TypeNoul, Instructions: "Is a pump stated?"}, "two": {Type: jev.TypeNoul, Instructions: "Is a valve stated?"}}}
	s, err := jev.CheckRequestSize(call)
	if err != nil || s.Questions != 2 || len(call.Questions) != 2 || s.EstimatedTokens == 0 {
		t.Fatalf("preflight %+v %v", s, err)
	}
}
