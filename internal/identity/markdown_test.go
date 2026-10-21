package identity_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sitewise/internal/identity"
)

func TestMarkdownExtraction(t *testing.T) {
	body := "\ufeff# Fire strategy\r\n\r\nDocument number: FS-101\r\n| System | Requirement |\r\n![detail](https://example.invalid/detail.png)\r\n"
	for _, format := range []string{"md", "markdown"} {
		got, err := identity.Extract(context.Background(), format, strings.NewReader(body), int64(len(body)), identity.Limits{})
		if err != nil {
			t.Fatal(err)
		}
		if !got.TextLayer || len(got.Runs) != 4 || !got.Runs[0].Source.Heading || got.Runs[0].Text != "# Fire strategy" || got.Runs[3].Text != "![detail](https://example.invalid/detail.png)" {
			t.Fatalf("%+v", got)
		}
	}
}

func TestMarkdownLimitsAndInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		limits     identity.Limits
		want       error
	}{
		{"invalid UTF8", "\xff", identity.Limits{}, identity.ErrMalformed},
		{"binary", "# Heading\x00", identity.Limits{}, identity.ErrMalformed},
		{"byte limit", "too big", identity.Limits{MaxBytes: 2}, identity.ErrTooLarge},
		{"full source run limit", "# Heading\nTail", identity.Limits{MaxRuns: 1, RequireComplete: true}, identity.ErrTooLarge},
		{"empty", "", identity.Limits{}, nil},
		{"whitespace", "\ufeff \r\n\t", identity.Limits{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := identity.Extract(context.Background(), "md", strings.NewReader(tc.body), int64(len(tc.body)), tc.limits)
			if !errors.Is(err, tc.want) || (tc.want == nil && got.TextLayer) {
				t.Fatalf("text=%+v err=%v", got, err)
			}
		})
	}
	got, err := identity.Extract(context.Background(), "md", strings.NewReader("# Heading\nTail"), 14, identity.Limits{MaxRuns: 1})
	if err != nil || !got.Incomplete || len(got.Runs) != 1 {
		t.Fatalf("bounded text=%+v err=%v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = identity.Extract(ctx, "md", strings.NewReader("text"), 4, identity.Limits{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
