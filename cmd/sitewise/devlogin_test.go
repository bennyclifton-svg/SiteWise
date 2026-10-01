package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDevLoginOnlyOnLoopbackOutsideProduction(t *testing.T) {
	cases := []struct {
		env, addr string
		ok        bool
	}{
		{"development", "127.0.0.1:8080", true},
		{"development", "localhost:8080", true},
		{"development", "[::1]:8080", true},
		{"development", "0.0.0.0:8080", false},
		{"development", ":8080", false},
		{"development", "192.168.1.5:8080", false},
		{"production", "127.0.0.1:8080", false},
	}
	for _, c := range cases {
		if err := devLoginAllowed(c.env, c.addr); (err == nil) != c.ok {
			t.Errorf("%s %s: %v", c.env, c.addr, err)
		}
	}
}

// serve refuses -dev-login on a public address before opening the database.
func TestServeRefusesDevLoginOffLoopback(t *testing.T) {
	env := map[string]string{
		"SITEWISE_DATABASE_URL":   "postgres://sitewise@127.0.0.1:1/none",
		"SITEWISE_FILE_DIR":       "unused",
		"SITEWISE_SESSION_SECRET": "s",
		"SITEWISE_JEV_API_KEY":    "k",
		"SITEWISE_JEV_MODEL":      "jev-1.13.0",
	}
	var stderr, stdout bytes.Buffer
	code := run([]string{"serve", "-dev-login", "-addr", "0.0.0.0:0"}, func(k string) string { return env[k] }, &stderr, &stdout)
	if code == 0 || !strings.Contains(stderr.String(), "dev-login") {
		t.Fatalf("code %d stderr %q", code, stderr.String())
	}
}
