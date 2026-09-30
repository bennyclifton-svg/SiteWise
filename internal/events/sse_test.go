package events_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"sitewise/internal/events"
	"sitewise/internal/store"
)

func TestSlowClientDoesNotBlockCommit(t *testing.T) {
	log := &memLog{}
	broker := events.NewBroker(log)
	_, cancel := broker.Subscribe("org")
	defer cancel()
	start := time.Now()
	for i := 0; i < 40; i++ {
		if _, err := broker.Commit(context.Background(), "org", "filing", "", `{"n":1}`); err != nil {
			t.Fatal(err)
		}
	}
	if time.Since(start) > time.Second {
		t.Fatalf("commits took %s", time.Since(start))
	}
	got, err := broker.CatchUp(context.Background(), "org", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 40 {
		t.Fatalf("stored %d", len(got))
	}
}

func TestReconnectAfterCommitBeforeSend(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	org, project := newID(t), newID(t)
	seedOrg(t, st, org, project)
	// Commit the row and do not notify a broker. A restarted process catches up.
	if _, err := st.AppendEvent(ctx, org, "filing", "", `{"document_id":"doc","status":"filed"}`); err != nil {
		t.Fatal(err)
	}
	broker := events.NewBroker(st)
	got, err := broker.CatchUp(ctx, org, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != "filing" || got[0].ID != 1 {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got[0].Payload, `"status":"filed"`) {
		t.Fatalf("payload %s", got[0].Payload)
	}

	pr, pw := io.Pipe()
	serveCtx, cancel := context.WithCancel(ctx)
	go func() {
		_ = broker.Serve(serveCtx, pw, org, 0)
		_ = pw.Close()
	}()
	buf := make([]byte, 4096)
	n, err := pr.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	cancel()
	text := string(buf[:n])
	if !strings.Contains(text, "id: 1\n") || !strings.Contains(text, "event: filing\n") {
		t.Fatalf("sse %s", text)
	}
	again, err := broker.CatchUp(ctx, org, got[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Fatalf("replay %+v", again)
	}
	if _, ok := events.Advance(got[0].ID, got[0].ID); ok {
		t.Fatal("duplicate id accepted")
	}
}

func TestOrgIsolatedEventIDs(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	orgA, orgB := newID(t), newID(t)
	projectA, projectB := newID(t), newID(t)
	seedOrg(t, st, orgA, projectA)
	seedOrg(t, st, orgB, projectB)
	if _, err := st.AppendEvent(ctx, orgA, "filing", "", `{"org":"a"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendEvent(ctx, orgB, "filing", "", `{"org":"b"}`); err != nil {
		t.Fatal(err)
	}
	broker := events.NewBroker(st)
	a, err := broker.CatchUp(ctx, orgA, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := broker.CatchUp(ctx, orgB, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 1 || len(b) != 1 || a[0].ID != 1 || b[0].ID != 1 {
		t.Fatalf("a %+v b %+v", a, b)
	}
	if strings.Contains(a[0].Payload, `"b"`) || strings.Contains(b[0].Payload, `"a"`) {
		t.Fatalf("a %s b %s", a[0].Payload, b[0].Payload)
	}
	miss, err := broker.CatchUp(ctx, orgA, b[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(miss) != 0 {
		t.Fatalf("org A walked org B's cursor %+v", miss)
	}

	live, cancel := broker.Subscribe(orgA)
	defer cancel()
	ev, err := broker.Commit(ctx, orgB, "job", "", `{"org":"b2"}`)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-live:
		t.Fatalf("org A received %+v", got)
	case <-time.After(50 * time.Millisecond):
	}
	if ev.ID != 2 {
		t.Fatalf("org B id %d", ev.ID)
	}
}

// A store transaction writes the event without the broker. Wake makes a live
// stream read it from the cursor.
func TestWakeDeliversEventCommittedOutsideBroker(t *testing.T) {
	log := &memLog{}
	broker := events.NewBroker(log)
	out := serveInto(t, broker, "org")
	waitFor(t, func() bool { return broker.Subscribers("org") == 1 })
	if _, err := log.AppendEvent(context.Background(), "org", "filing", "", `{"n":1}`); err != nil {
		t.Fatal(err)
	}
	broker.Wake("org")
	waitFor(t, func() bool { return strings.Contains(out.String(), "id: 1\n") })
}

// A live event dropped for a slow client is still delivered from the cursor
// without a reconnect.
func TestSlowStreamCatchesUpDroppedEvents(t *testing.T) {
	log := &memLog{}
	broker := events.NewBroker(log)
	pr, pw := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		_ = broker.Serve(ctx, pw, "org", 0)
		_ = pw.Close()
	}()
	waitFor(t, func() bool { return broker.Subscribers("org") == 1 })
	for i := 0; i < 5; i++ {
		if _, err := broker.Commit(ctx, "org", "filing", "", `{"n":1}`); err != nil {
			t.Fatal(err)
		}
	}
	out := &lockedBuffer{}
	go func() { _, _ = io.Copy(out, pr) }()
	waitFor(t, func() bool { return strings.Contains(out.String(), "id: 5\n") })
	if n := strings.Count(out.String(), "event: filing\n"); n != 5 {
		t.Fatalf("delivered %d of 5:\n%s", n, out.String())
	}
}

func TestLastEventID(t *testing.T) {
	n, err := events.LastEventID("")
	if err != nil || n != 0 {
		t.Fatalf("%d %v", n, err)
	}
	n, err = events.LastEventID("12")
	if err != nil || n != 12 {
		t.Fatalf("%d %v", n, err)
	}
	if _, err := events.LastEventID("-1"); err == nil {
		t.Fatal("accepted negative")
	}
}

func serveInto(t *testing.T, broker *events.Broker, org string) *lockedBuffer {
	t.Helper()
	out := &lockedBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = broker.Serve(ctx, out, org, 0) }()
	return out
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 2s")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type memLog struct {
	mu     sync.Mutex
	next   map[string]int64
	events map[string][]store.StoredEvent
}

func (m *memLog) AppendEvent(_ context.Context, orgID, kind, documentID, payload string) (store.StoredEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.next == nil {
		m.next = map[string]int64{}
		m.events = map[string][]store.StoredEvent{}
	}
	m.next[orgID]++
	ev := store.StoredEvent{ID: m.next[orgID], Kind: kind, DocumentID: documentID, Payload: payload}
	m.events[orgID] = append(m.events[orgID], ev)
	return ev, nil
}

func (m *memLog) EventsAfter(_ context.Context, orgID string, after int64, limit int) ([]store.StoredEvent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.StoredEvent
	for _, ev := range m.events[orgID] {
		if ev.ID <= after {
			continue
		}
		out = append(out, ev)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("SITEWISE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("SITEWISE_TEST_DATABASE_URL is required")
	}
	name, err := databaseName(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if name != "sitewise_test" {
		t.Fatalf("refusing to use database %q", name)
	}
	ctx := context.Background()
	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	return st
}

func seedOrg(t *testing.T, st *store.Store, org, project string) {
	t.Helper()
	ctx := context.Background()
	t.Cleanup(func() { _ = st.DeleteOrg(context.Background(), org) })
	if err := st.CreateOrg(ctx, org, "org"); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateProject(ctx, org, project, "project"); err != nil {
		t.Fatal(err)
	}
}

func newID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func databaseName(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", errors.New("database name missing")
	}
	return name, nil
}
