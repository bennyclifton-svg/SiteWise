package llm

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func okSend(res Response) func(context.Context, Request) (Response, error) {
	return func(context.Context, Request) (Response, error) { return res, nil }
}

func TestAskReturnsWhatSendReturns(t *testing.T) {
	var got Request
	c := newWithSend(Options{}, func(_ context.Context, req Request) (Response, error) {
		got = req
		return Response{Text: "answer", StopReason: "end_turn", InTokens: 3, OutTokens: 4}, nil
	})
	res, err := c.Ask(context.Background(), Request{Name: "n", Model: "m", System: "s"})
	if err != nil || res.Text != "answer" || res.StopReason != "end_turn" || got.Model != "m" || got.System != "s" {
		t.Fatalf("res %+v err %v got %+v", res, err, got)
	}
}

func TestAskWithNoKeyIsOff(t *testing.T) {
	c := New(Options{})
	if c.On() {
		t.Fatal("a client with no key is off")
	}
	if _, err := c.Ask(context.Background(), Request{}); !errors.Is(err, ErrOff) {
		t.Fatalf("err %v", err)
	}
}

func TestAskLimitsConcurrency(t *testing.T) {
	release := make(chan struct{})
	var started atomic.Int32
	c := newWithSend(Options{MaxConcurrent: 2}, func(context.Context, Request) (Response, error) {
		started.Add(1)
		<-release
		return Response{}, nil
	})
	done := make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		go func() { c.Ask(context.Background(), Request{}); done <- struct{}{} }()
	}
	deadline := time.Now().Add(2 * time.Second)
	for started.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(30 * time.Millisecond)
	if n := started.Load(); n != 2 {
		t.Fatalf("%d calls started with two slots", n)
	}
	release <- struct{}{}
	<-done
	deadline = time.Now().Add(2 * time.Second)
	for started.Load() < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if n := started.Load(); n != 3 {
		t.Fatalf("the third call did not start after a slot freed: %d", n)
	}
	close(release)
	<-done
	<-done
}

func TestAskStopsWaitingWhenContextEnds(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	c := newWithSend(Options{MaxConcurrent: 1}, func(context.Context, Request) (Response, error) {
		close(started)
		<-release
		return Response{}, nil
	})
	go c.Ask(context.Background(), Request{})
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Ask(ctx, Request{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err %v", err)
	}
	close(release)
}

func TestBreakerOpensAfterFiveFailures(t *testing.T) {
	calls := 0
	fail := true
	c := newWithSend(Options{}, func(context.Context, Request) (Response, error) {
		calls++
		if fail {
			return Response{}, errors.New("boom")
		}
		return Response{}, nil
	})
	now := time.Unix(1000, 0)
	c.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		c.Ask(context.Background(), Request{})
	}
	if _, err := c.Ask(context.Background(), Request{}); !errors.Is(err, ErrBusy) || calls != 5 {
		t.Fatalf("sixth call: err %v, calls %d", err, calls)
	}
	now = now.Add(61 * time.Second)
	fail = false
	if _, err := c.Ask(context.Background(), Request{}); err != nil || calls != 6 {
		t.Fatalf("after the wait: err %v, calls %d", err, calls)
	}
}

func TestRefusalDoesNotTripBreaker(t *testing.T) {
	calls := 0
	c := newWithSend(Options{}, func(context.Context, Request) (Response, error) {
		calls++
		return Response{}, ErrRefused
	})
	for i := 0; i < 6; i++ {
		if _, err := c.Ask(context.Background(), Request{}); !errors.Is(err, ErrRefused) {
			t.Fatalf("call %d: %v", i, err)
		}
	}
	if calls != 6 {
		t.Fatalf("calls %d", calls)
	}
}

func TestLogNeverCarriesContent(t *testing.T) {
	var buf bytes.Buffer
	c := newWithSend(Options{Logger: slog.New(slog.NewTextHandler(&buf, nil))}, okSend(Response{Text: "SECRET answer", StopReason: "end_turn"}))
	c.Ask(context.Background(), Request{Name: "tender_pass", Model: "m", System: "SECRET system", Parts: []Part{{Text: "SECRET page"}}})
	if out := buf.String(); strings.Contains(out, "SECRET") || !strings.Contains(out, "tender_pass") {
		t.Fatalf("log: %s", out)
	}
}

func TestRetryable(t *testing.T) {
	if Retryable(nil) || Retryable(ErrOff) || Retryable(ErrRefused) || !Retryable(errors.New("reset")) {
		t.Fatal("retry classes are wrong")
	}
}
