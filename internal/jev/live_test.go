//go:build live

package jev_test

import (
	"context"
	"os"
	"testing"
	"time"

	"sitewise/internal/jev"
)

// TestLiveCredential is opt-in. Run it with:
//
//	go test -tags live ./internal/jev
func TestLiveCredential(t *testing.T) {
	key := os.Getenv("SITEWISE_JEV_API_KEY")
	if key == "" {
		t.Fatal("SITEWISE_JEV_API_KEY is required when -tags live is set")
	}
	c, err := jev.New(jev.Options{APIKey: key})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, err := c.Ask(ctx, jev.Call{
		State: "ping",
		Questions: map[string]jev.Question{
			"is_short": {
				Type:         jev.TypeNoul,
				Instructions: "Is the state a single word?",
			},
		},
		QuestionVersion: "live-probe",
		Deadline:        10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ans, ok := res.Answers["is_short"]
	if !ok || ans.Type != jev.TypeNoul {
		t.Fatalf("%+v", res)
	}
}
