package jev_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"sitewise/internal/config"
	"sitewise/internal/jev"
)

const (
	canaryAPIKey   = "CANARY_JEV_API_KEY"
	canaryDocument = "CANARY_DOCUMENT_TEXT"
)

func TestConfiguredCapacity(t *testing.T) {
	if jev.TotalSlots != 24 || jev.InteractiveReserve != 8 {
		t.Fatalf("slots %d reserve %d", jev.TotalSlots, jev.InteractiveReserve)
	}
	if jev.InteractiveReserve < 1 || jev.InteractiveReserve >= jev.TotalSlots {
		t.Fatal("reserve must keep interactive room and leave background capacity")
	}
	if jev.InteractiveDeadline != 1200*time.Millisecond {
		t.Fatalf("deadline %s", jev.InteractiveDeadline)
	}
	if jev.Endpoint != "https://api.typesafe.ai/v1/systemone" {
		t.Fatal(jev.Endpoint)
	}
	if config.PinnedJevModel != "jev-1.13.0" {
		t.Fatal(config.PinnedJevModel)
	}
}

func TestNewRejectsMovingAliasAndBlankKey(t *testing.T) {
	_, err := jev.New(jev.Options{APIKey: canaryAPIKey, Model: "jev-latest"})
	if err == nil {
		t.Fatal("expected error")
	}
	if stringsContains(err.Error(), canaryAPIKey) {
		t.Fatalf("error contains api key: %s", err)
	}
	_, err = jev.New(jev.Options{APIKey: "  "})
	if err == nil || stringsContains(err.Error(), "CANARY") {
		t.Fatalf("blank key: %v", err)
	}
}

func TestAskSuccessRecordedResponse(t *testing.T) {
	var hits atomic.Int32
	var buf bytes.Buffer
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+canaryAPIKey {
			t.Errorf("authorization %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type %q", r.Header.Get("Content-Type"))
		}
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if got["model"] != config.PinnedJevModel {
			t.Errorf("model %v", got["model"])
		}
		state, _ := got["state"].(map[string]any)
		if state["title_block"] != canaryDocument {
			t.Errorf("state %v", got["state"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "success.json"))
	}), jev.Options{Logger: slog.New(slog.NewJSONHandler(&buf, nil))})

	res, err := c.Ask(context.Background(), sampleCall())
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("attempts %d", hits.Load())
	}
	if res.Model != config.PinnedJevModel || res.QuestionVersion != "intake-v1" {
		t.Fatalf("%+v", res)
	}
	kind := res.Answers["kind"]
	if kind.Type != jev.TypeChoice || kind.Choice != "drawing" || kind.Confidence == nil {
		t.Fatalf("kind %+v", kind)
	}
	near(t, *kind.Confidence, 0.84)
	near(t, kind.Probabilities["drawing"], 0.91)
	rev := res.Answers["is_revision"]
	if rev.Type != jev.TypeNoul || rev.Confidence != nil {
		t.Fatalf("noul %+v", rev)
	}
	near(t, rev.Noul, 0.12)
	if _, ok := res.Answers["notes"]; ok {
		t.Fatal("unknown question was applied")
	}
	if len(res.Unresolved) != 0 {
		t.Fatalf("unresolved %v", res.Unresolved)
	}
	if res.Usage.InputTokens != 120 || res.Usage.OutputTokens != 18 {
		t.Fatalf("usage %+v", res.Usage)
	}

	text := buf.String()
	assertNoCanary(t, text)
	lines := logLines(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("logs %d", len(lines))
	}
	line := lines[0]
	if line["msg"] != "jev" || line["model"] != config.PinnedJevModel || line["question_version"] != "intake-v1" {
		t.Fatalf("log %#v", line)
	}
	if _, ok := line["latency_us"]; !ok {
		t.Fatalf("log %#v", line)
	}
	if line["input_tokens"] != 120.0 || line["output_tokens"] != 18.0 {
		t.Fatalf("tokens %#v", line)
	}
	uncertainty, _ := line["uncertainty"].(map[string]any)
	near(t, uncertainty["kind"].(float64), 0.84)
	near(t, uncertainty["is_revision"].(float64), 0.12)
	if _, ok := uncertainty["notes"]; ok {
		t.Fatal("logged an unknown question")
	}
}

func TestAskDropsInvalidChoice(t *testing.T) {
	res, err := askFixture(t, "invalid_choice.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Answers["kind"]; ok {
		t.Fatalf("invalid choice applied: %+v", res.Answers["kind"])
	}
	if res.Answers["is_revision"].Type != jev.TypeNoul {
		t.Fatalf("%+v", res)
	}
	near(t, res.Answers["is_revision"].Noul, 0.5)
	if stringsJoin(res.Unresolved) != "kind" {
		t.Fatalf("unresolved %v", res.Unresolved)
	}
}

func TestAskRejectsWrongModel(t *testing.T) {
	res, err := askFixture(t, "wrong_model.json")
	if !errors.Is(err, jev.ErrBadResponse) {
		t.Fatalf("err %v", err)
	}
	if len(res.Answers) != 0 {
		t.Fatalf("answers applied from wrong model: %+v", res.Answers)
	}
}

func TestAskDropsWrongType(t *testing.T) {
	res, err := askFixture(t, "wrong_type.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Answers["kind"]; ok {
		t.Fatal("wrong type applied")
	}
	near(t, res.Answers["is_revision"].Noul, 0.2)
	if stringsJoin(res.Unresolved) != "kind" {
		t.Fatalf("unresolved %v", res.Unresolved)
	}
}

func TestAskKeepsPartialWhenAnswerMissing(t *testing.T) {
	res, err := askFixture(t, "missing_answer.json")
	if err != nil {
		t.Fatal(err)
	}
	if res.Answers["kind"].Choice != "report" {
		t.Fatalf("kind %+v", res.Answers["kind"])
	}
	if _, ok := res.Answers["is_revision"]; ok {
		t.Fatal("missing answer was invented")
	}
	if stringsJoin(res.Unresolved) != "is_revision" {
		t.Fatalf("unresolved %v", res.Unresolved)
	}
}

func TestAskDropsBadProbability(t *testing.T) {
	res, err := askFixture(t, "bad_probability.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Answers["kind"]; ok {
		t.Fatal("bad probability applied")
	}
	near(t, res.Answers["is_revision"].Noul, 0.33)
	if stringsJoin(res.Unresolved) != "kind" {
		t.Fatalf("unresolved %v", res.Unresolved)
	}
}

func TestAskDropsDistributionAndNoulRange(t *testing.T) {
	sum := []byte(`{"model":"jev-1.13.0","answers":{"kind":{"type":"choice","choice":"drawing","probabilities":{"drawing":0.5,"report":0.4},"confidence":0.2},"is_revision":{"type":"noul","noul":0.25}},"usage":{"input_tokens":1,"output_tokens":1}}`)
	res, err := askBody(t, sum, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Answers["kind"]; ok {
		t.Fatal("short distribution applied")
	}
	near(t, res.Answers["is_revision"].Noul, 0.25)

	high := []byte(`{"model":"jev-1.13.0","answers":{"kind":{"type":"choice","choice":"drawing","probabilities":{"drawing":1,"report":0},"confidence":1},"is_revision":{"type":"noul","noul":1.2}}}`)
	res, err = askBody(t, high, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
	if res.Answers["kind"].Choice != "drawing" {
		t.Fatalf("kind %+v", res.Answers["kind"])
	}
	if _, ok := res.Answers["is_revision"]; ok {
		t.Fatal("noul above 1 applied")
	}

	broken := []byte(`{"model":"jev-1.13.0","answers":{"kind":"nope","is_revision":{"type":"noul","noul":0.4}}}`)
	res, err = askBody(t, broken, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Answers["kind"]; ok {
		t.Fatal("malformed choice applied")
	}
	near(t, res.Answers["is_revision"].Noul, 0.4)

	nan := []byte(`{"model":"jev-1.13.0","answers":{"kind":{"type":"choice","choice":"drawing","probabilities":{"drawing":1,"report":0},"confidence":1},"is_revision":{"type":"noul","noul":NaN}}}`)
	res, err = askBody(t, nan, http.StatusOK)
	if !errors.Is(err, jev.ErrBadResponse) {
		t.Fatalf("err %v", err)
	}
	if len(res.Answers) != 0 {
		t.Fatalf("partial parse of non-finite json: %+v", res.Answers)
	}
}

func TestAskValidatesScore(t *testing.T) {
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "score.json"))
	}), jev.Options{})
	call := jev.Call{
		State: "title block",
		Questions: map[string]jev.Question{
			"quality": {
				Type:         jev.TypeScore,
				Instructions: "How complete is the title block?",
				Criteria:     []string{"poor", "fair", "good"},
			},
		},
		QuestionVersion: "score-v1",
	}
	res, err := c.Ask(context.Background(), call)
	if err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("attempts %d", hits.Load())
	}
	quality := res.Answers["quality"]
	if quality.Type != jev.TypeScore || quality.Confidence == nil {
		t.Fatalf("%+v", quality)
	}
	near(t, quality.Score, 1.25)
	near(t, *quality.Confidence, 0.4)

	mismatch := bytes.Replace(fixture(t, "score.json"), []byte(`"poor"`), []byte(`"bad"`), 1)
	res, err = askBody(t, mismatch, http.StatusOK)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Answers) != 0 || stringsJoin(res.Unresolved) != "quality" {
		t.Fatalf("%+v", res)
	}
}

func TestInteractiveRateLimitAndOverloadAreOneAttempt(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		want   error
	}{
		{name: "429", status: http.StatusTooManyRequests, want: jev.ErrRateLimited},
		{name: "529", status: 529, want: jev.ErrOverloaded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hits atomic.Int32
			var sleeps atomic.Int32
			var buf bytes.Buffer
			body := canaryAPIKey + canaryDocument
			c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits.Add(1)
				w.WriteHeader(tc.status)
				_, _ = ioWrite(w, body)
			}), jev.Options{
				Logger: slog.New(slog.NewJSONHandler(&buf, nil)),
				Sleep: func(context.Context, time.Duration) error {
					sleeps.Add(1)
					return nil
				},
			})
			res, err := c.Ask(context.Background(), sampleCall())
			if !errors.Is(err, tc.want) {
				t.Fatalf("err %v", err)
			}
			if len(res.Answers) != 0 || hits.Load() != 1 || sleeps.Load() != 0 {
				t.Fatalf("hits %d sleeps %d answers %d", hits.Load(), sleeps.Load(), len(res.Answers))
			}
			assertNoCanary(t, err.Error())
			assertNoCanary(t, buf.String())
		})
	}
}

func TestBackgroundRetriesWithBackoff(t *testing.T) {
	var hits atomic.Int32
	var mu sync.Mutex
	var delays []time.Duration
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(529)
		_, _ = ioWrite(w, canaryDocument)
	}), jev.Options{
		BreakerThreshold: 10,
		Sleep: func(_ context.Context, d time.Duration) error {
			mu.Lock()
			delays = append(delays, d)
			mu.Unlock()
			return nil
		},
	})
	call := sampleCall()
	call.Priority = jev.PriorityBackground
	_, err := c.Ask(context.Background(), call)
	if !errors.Is(err, jev.ErrOverloaded) {
		t.Fatalf("err %v", err)
	}
	if hits.Load() != 3 {
		t.Fatalf("attempts %d", hits.Load())
	}
	if len(delays) != 2 || delays[0] != 200*time.Millisecond || delays[1] != 400*time.Millisecond {
		t.Fatalf("backoff %v", delays)
	}
}

func TestBackgroundDoesNotRetryRejectedRequest(t *testing.T) {
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = ioWrite(w, canaryDocument)
	}), jev.Options{})
	call := sampleCall()
	call.Priority = jev.PriorityBackground
	_, err := c.Ask(context.Background(), call)
	if !errors.Is(err, jev.ErrRequest) || hits.Load() != 1 {
		t.Fatalf("hits %d err %v", hits.Load(), err)
	}
	assertNoCanary(t, err.Error())
}

func TestSlowHeadersUseInteractiveDeadline(t *testing.T) {
	var hits atomic.Int32
	// HTTP/1.1 does not cancel the request context until a read sees the
	// client leave, so the handler waits on a test channel instead.
	release, stopRelease := newRelease()
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case <-release:
		case <-time.After(5 * time.Second):
		}
	}), jev.Options{})
	t.Cleanup(stopRelease)
	parent, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	start := time.Now()
	_, err := c.Ask(parent, sampleCall())
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err %v elapsed %s", err, elapsed)
	}
	if hits.Load() != 1 {
		t.Fatalf("attempts %d", hits.Load())
	}
	if elapsed < 1000*time.Millisecond || elapsed > 2500*time.Millisecond {
		t.Fatalf("elapsed %s", elapsed)
	}
}

func TestSlowBodyStopsWithinCallDeadline(t *testing.T) {
	var hits atomic.Int32
	release, stopRelease := newRelease()
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-time.After(2 * time.Second):
		}
	}), jev.Options{})
	t.Cleanup(stopRelease)
	call := sampleCall()
	call.Deadline = 80 * time.Millisecond
	start := time.Now()
	res, err := c.Ask(context.Background(), call)
	if err == nil || len(res.Answers) != 0 {
		t.Fatalf("err %v answers %d", err, len(res.Answers))
	}
	if hits.Load() != 1 {
		t.Fatalf("attempts %d", hits.Load())
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("elapsed %s", time.Since(start))
	}
}

func TestQueueSaturation(t *testing.T) {
	entered := make(chan struct{}, 2)
	release, stopRelease := newRelease()
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		entered <- struct{}{}
		<-release
	}), jev.Options{Slots: 2, InteractiveReserve: 1})
	t.Cleanup(stopRelease)
	go func() { _, _ = c.Ask(context.Background(), heldCall()) }()
	go func() { _, _ = c.Ask(context.Background(), heldCall()) }()
	<-entered
	<-entered

	call := sampleCall()
	call.Deadline = 80 * time.Millisecond
	start := time.Now()
	_, err := c.Ask(context.Background(), call)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err %v", err)
	}
	if hits.Load() != 2 {
		t.Fatalf("attempts %d", hits.Load())
	}
	if time.Since(start) > 500*time.Millisecond {
		t.Fatalf("elapsed %s", time.Since(start))
	}
}

func TestDeadlineSpansAdmissionAndTransport(t *testing.T) {
	entered := make(chan struct{}, 2)
	rel1, stop1 := newRelease()
	rel2, stop2 := newRelease()
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		switch n {
		case 1:
			entered <- struct{}{}
			<-rel1
		case 2:
			entered <- struct{}{}
			<-rel2
		default:
			time.Sleep(300 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture(t, "success.json"))
		}
	}), jev.Options{Slots: 2, InteractiveReserve: 1})
	t.Cleanup(stop1)
	t.Cleanup(stop2)
	go func() { _, _ = c.Ask(context.Background(), heldCall()) }()
	go func() { _, _ = c.Ask(context.Background(), heldCall()) }()
	<-entered
	<-entered
	go func() {
		time.Sleep(300 * time.Millisecond)
		stop1()
	}()

	call := sampleCall()
	call.Deadline = 500 * time.Millisecond
	start := time.Now()
	_, err := c.Ask(context.Background(), call)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("succeeded; deadline was reset after admission")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err %v", err)
	}
	if hits.Load() != 3 {
		t.Fatalf("attempts %d", hits.Load())
	}
	if elapsed < 450*time.Millisecond || elapsed > 900*time.Millisecond {
		t.Fatalf("elapsed %s", elapsed)
	}
}

func TestBackgroundCannotTakeInteractiveReserve(t *testing.T) {
	entered := make(chan struct{})
	release, stopRelease := newRelease()
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			close(entered)
			<-release
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "success.json"))
	}), jev.Options{
		Slots:              2,
		InteractiveReserve: 1,
		Sleep:              func(context.Context, time.Duration) error { return nil },
	})
	t.Cleanup(stopRelease)
	holder := heldCall()
	holder.Priority = jev.PriorityBackground
	go func() { _, _ = c.Ask(context.Background(), holder) }()
	<-entered

	errCh := make(chan error, 1)
	go func() {
		call := sampleCall()
		call.Priority = jev.PriorityBackground
		call.Deadline = 40 * time.Millisecond
		_, err := c.Ask(context.Background(), call)
		errCh <- err
	}()
	res, err := c.Ask(context.Background(), sampleCall())
	if err != nil {
		t.Fatal(err)
	}
	if res.Answers["kind"].Choice != "drawing" {
		t.Fatalf("%+v", res.Answers)
	}
	if bgErr := <-errCh; bgErr == nil {
		t.Fatal("background took the reserved slot")
	}
	if hits.Load() != 2 {
		t.Fatalf("attempts %d", hits.Load())
	}
}

func TestCancelWhileQueued(t *testing.T) {
	entered := make(chan struct{}, 2)
	release, stopRelease := newRelease()
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		entered <- struct{}{}
		<-release
	}), jev.Options{Slots: 2, InteractiveReserve: 1})
	t.Cleanup(stopRelease)
	go func() { _, _ = c.Ask(context.Background(), heldCall()) }()
	go func() { _, _ = c.Ask(context.Background(), heldCall()) }()
	<-entered
	<-entered

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		_, err := c.Ask(ctx, sampleCall())
		errCh <- err
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("queued call ignored cancellation")
	}
	if hits.Load() != 2 {
		t.Fatalf("attempts %d", hits.Load())
	}
}

func TestCircuitBreaker(t *testing.T) {
	var hits atomic.Int32
	var succeed atomic.Bool
	clock := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if !succeed.Load() {
			http.Error(w, canaryDocument, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "success.json"))
	}), jev.Options{
		BreakerThreshold: 2,
		BreakerCooldown:  time.Minute,
		Now: func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return clock
		},
	})
	if _, err := c.Ask(context.Background(), sampleCall()); !errors.Is(err, jev.ErrBadResponse) {
		t.Fatalf("first %v", err)
	}
	if _, err := c.Ask(context.Background(), sampleCall()); !errors.Is(err, jev.ErrBadResponse) {
		t.Fatalf("second %v", err)
	}
	if _, err := c.Ask(context.Background(), sampleCall()); !errors.Is(err, jev.ErrCircuitOpen) || hits.Load() != 2 {
		t.Fatalf("hits %d", hits.Load())
	}

	advance := func(d time.Duration) {
		mu.Lock()
		clock = clock.Add(d)
		mu.Unlock()
	}
	advance(30 * time.Second)
	if _, err := c.Ask(context.Background(), sampleCall()); !errors.Is(err, jev.ErrCircuitOpen) || hits.Load() != 2 {
		t.Fatalf("early probe hits %d err", hits.Load())
	}
	advance(30 * time.Second)
	if _, err := c.Ask(context.Background(), sampleCall()); err == nil || hits.Load() != 3 {
		t.Fatalf("probe hits %d err %v", hits.Load(), err)
	}
	if _, err := c.Ask(context.Background(), sampleCall()); !errors.Is(err, jev.ErrCircuitOpen) || hits.Load() != 3 {
		t.Fatalf("reopened hits %d", hits.Load())
	}
	advance(time.Minute)
	succeed.Store(true)
	res, err := c.Ask(context.Background(), sampleCall())
	if err != nil || res.Answers["kind"].Choice != "drawing" || hits.Load() != 4 {
		t.Fatalf("recover hits %d err %v", hits.Load(), err)
	}
	if _, err := c.Ask(context.Background(), sampleCall()); err != nil || hits.Load() != 5 {
		t.Fatalf("closed hits %d err %v", hits.Load(), err)
	}
}

func TestResponseBodyLimit(t *testing.T) {
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.Repeat([]byte("x"), 33))
	}), jev.Options{MaxBody: 32})
	res, err := c.Ask(context.Background(), sampleCall())
	if !errors.Is(err, jev.ErrBodyLimit) || len(res.Answers) != 0 || hits.Load() != 1 {
		t.Fatalf("hits %d err %v answers %d", hits.Load(), err, len(res.Answers))
	}
}

func TestChoiceOverOptionLimitDoesNotCall(t *testing.T) {
	var hits atomic.Int32
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
	}), jev.Options{})
	criteria := make(map[string]string, 256)
	for i := 0; i < 256; i++ {
		criteria[fmt.Sprintf("o%d", i)] = "option"
	}
	_, err := c.Ask(context.Background(), jev.Call{
		State: "x",
		Questions: map[string]jev.Question{
			"kind": {Type: jev.TypeChoice, Instructions: "Which?", Criteria: criteria},
		},
	})
	if !errors.Is(err, jev.ErrRequest) || hits.Load() != 0 {
		t.Fatalf("hits %d err %v", hits.Load(), err)
	}
}

func TestRedirectDoesNotLeakAuthorization(t *testing.T) {
	var hits atomic.Int32
	var leaked atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/secret", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			leaked.Store(true)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Redirect(w, r, "/secret", http.StatusFound)
	})
	c := startClient(t, mux, jev.Options{})
	_, err := c.Ask(context.Background(), sampleCall())
	if err == nil || leaked.Load() || hits.Load() != 1 {
		t.Fatalf("hits %d leaked %v err %v", hits.Load(), leaked.Load(), err)
	}
	assertNoCanary(t, err.Error())
}

func TestWarmReusesHTTP2Connection(t *testing.T) {
	var conns atomic.Int32
	var postProto atomic.Int32
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		postProto.Store(int32(r.ProtoMajor))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, "success.json"))
	}))
	srv.EnableHTTP2 = true
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	srv.StartTLS()
	t.Cleanup(srv.Close)

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	tr := jev.NewTransport()
	tr.TLSClientConfig = &tls.Config{
		RootCAs:    pool,
		NextProtos: []string{"h2", "http/1.1"},
		ServerName: "127.0.0.1",
	}
	c, err := jev.New(jev.Options{
		BaseURL:   srv.URL,
		APIKey:    canaryAPIKey,
		Transport: tr,
		Logger:    slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Warm(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Ask(context.Background(), sampleCall()); err != nil {
		t.Fatal(err)
	}
	if postProto.Load() != 2 {
		t.Fatalf("protocol %d", postProto.Load())
	}
	if conns.Load() != 1 {
		t.Fatalf("connections %d", conns.Load())
	}
}

func newRelease() (<-chan struct{}, func()) {
	ch := make(chan struct{})
	var once sync.Once
	return ch, func() { once.Do(func() { close(ch) }) }
}

func startClient(t *testing.T, h http.Handler, opt jev.Options) *jev.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	if opt.BaseURL == "" {
		opt.BaseURL = srv.URL
	}
	if opt.APIKey == "" {
		opt.APIKey = canaryAPIKey
	}
	if opt.Transport == nil {
		opt.Transport = jev.NewTransport()
	}
	if opt.Logger == nil {
		opt.Logger = slog.New(slog.DiscardHandler)
	}
	c, err := jev.New(opt)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return c
}

func askFixture(t *testing.T, name string) (jev.Result, error) {
	t.Helper()
	return askBody(t, fixture(t, name), http.StatusOK)
}

func askBody(t *testing.T, body []byte, status int) (jev.Result, error) {
	t.Helper()
	c := startClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}), jev.Options{})
	questions := sampleQuestions()
	if bytes.Contains(body, []byte(`"quality"`)) {
		questions = map[string]jev.Question{
			"quality": {
				Type:         jev.TypeScore,
				Instructions: "How complete is the title block?",
				Criteria:     []string{"poor", "fair", "good"},
			},
		}
	}
	return c.Ask(context.Background(), jev.Call{
		State:           map[string]string{"title_block": canaryDocument},
		Questions:       questions,
		QuestionVersion: "intake-v1",
	})
}

func sampleQuestions() map[string]jev.Question {
	return map[string]jev.Question{
		"kind": {
			Type:         jev.TypeChoice,
			Instructions: "Which kind is this document?",
			Criteria: map[string]string{
				"drawing": "A drawing",
				"report":  "A report",
			},
		},
		"is_revision": {
			Type:         jev.TypeNoul,
			Instructions: "Does the evidence say this is a revision?",
			Criteria: map[string]string{
				"true":  "It is a revision",
				"false": "It is not",
			},
		},
	}
}

func heldCall() jev.Call {
	call := sampleCall()
	call.Deadline = 30 * time.Second
	return call
}

func sampleCall() jev.Call {
	return jev.Call{
		State:           map[string]string{"title_block": canaryDocument},
		Questions:       sampleQuestions(),
		QuestionVersion: "intake-v1",
		Priority:        jev.PriorityInteractive,
	}
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	b, err := os.ReadFile(filepath.Join(filepath.Dir(file), "testdata", "jev", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v want %v", got, want)
	}
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	sc := bufio.NewScanner(buf)
	for sc.Scan() {
		var line map[string]any
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			t.Fatalf("log %s: %v", sc.Text(), err)
		}
		out = append(out, line)
	}
	return out
}

func assertNoCanary(t *testing.T, text string) {
	t.Helper()
	if stringsContains(text, canaryAPIKey) || stringsContains(text, canaryDocument) {
		t.Fatalf("secret or document text leaked: %s", text)
	}
}

func stringsContains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && bytes.Contains([]byte(s), []byte(sub))
}

func stringsJoin(ids []string) string {
	return string(bytes.Join(split(ids), []byte(",")))
}

func split(ids []string) [][]byte {
	out := make([][]byte, len(ids))
	for i, id := range ids {
		out[i] = []byte(id)
	}
	return out
}

func ioWrite(w http.ResponseWriter, body string) (int, error) {
	return w.Write([]byte(body))
}
