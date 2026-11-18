package httpapi

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// The guest cookie carries Secure when the server's cookies do: production,
// where cmd/sitewise sets SecureCookie from SITEWISE_ENV=production.
func TestGuestCookieSecureInProduction(t *testing.T) {
	for _, secure := range []bool{true, false} {
		c := guestCookie(Deps{SecureCookie: secure}, "t", time.Now())
		if c.Secure != secure || !c.HttpOnly || c.Path != "/api/guest" || !strings.Contains(c.String(), "SameSite=Strict") {
			t.Fatalf("secure=%v: %s", secure, c.String())
		}
	}
}

// L1: the limiter has a hard cap. A flood of new keys never grows it past
// limitKeysCap, each insert does bounded work, and a key already counted
// keeps its window.
func TestLimiterHasAHardCap(t *testing.T) {
	l := newAuthLimits()
	for i := 0; i < limitKeysCap+5_000; i++ {
		l.allow(fmt.Sprintf("flood:%d", i), 30, time.Minute)
		if len(l.counts) > limitKeysCap {
			t.Fatalf("limiter grew to %d", len(l.counts))
		}
	}
	start := time.Now()
	l.allow("signin:someone@example.test", 10, time.Minute)
	if time.Since(start) > 50*time.Millisecond {
		t.Fatalf("one allow took %v at the cap", time.Since(start))
	}
}

// L1: IPv6 clients are keyed by their /64; IPv4 as they are.
func TestLimitAddressKeysIPv6By64(t *testing.T) {
	a, b := limitAddress("2001:db8:1:2:aaaa::1"), limitAddress("2001:db8:1:2:bbbb::9")
	if a != b || a != "2001:db8:1:2::/64" {
		t.Fatalf("%q %q", a, b)
	}
	if limitAddress("2001:db8:1:3::1") == a {
		t.Fatal("another /64 shares the key")
	}
	if limitAddress("203.0.113.7") != "203.0.113.7" {
		t.Fatal("IPv4 changed")
	}
}
