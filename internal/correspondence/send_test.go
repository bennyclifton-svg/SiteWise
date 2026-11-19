package correspondence

import (
	"context"
	"net"
	"testing"
	"time"
)

// Mail review L3: a relay that accepts the connection and never answers
// fails the attempt within the timeout, so the outbox can retry it.
func TestSMTPHungRelayTimesOut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close() // hold it open, say nothing
		}
	}()
	start := time.Now()
	err = SMTP{Addr: ln.Addr().String(), Timeout: 300 * time.Millisecond}.Send(context.Background(), "s@example.test", []string{"a@example.net"}, []byte("x"))
	if err == nil || time.Since(start) > 3*time.Second {
		t.Fatalf("hung relay: %v after %v", err, time.Since(start))
	}
}
