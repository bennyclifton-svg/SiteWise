// Package events is the durable cursor clients resume from.
// A commit writes the row first. Live delivery is best-effort and never
// waits on a slow client; a reconnect reads the same rows again.
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"sitewise/internal/store"
)

const pageSize = 100

// Log is the durable cursor. Implementations scope every read and write by org.
type Log interface {
	AppendEvent(ctx context.Context, orgID, kind, documentID, payload string) (store.StoredEvent, error)
	EventsAfter(ctx context.Context, orgID string, after int64, limit int) ([]store.StoredEvent, error)
}

// Broker fans committed events out to live subscribers without holding the
// commit open. When a subscriber's buffer is full the event is not queued;
// the subscriber is rung instead and reads it from the cursor.
type Broker struct {
	log  Log
	mu   sync.Mutex
	subs map[string]map[*subscription]struct{}
}

type subscription struct {
	ch   chan store.StoredEvent
	ring chan struct{}
}

// NewBroker serves log. log must already have committed the event before Notify.
func NewBroker(log Log) *Broker {
	return &Broker{log: log, subs: map[string]map[*subscription]struct{}{}}
}

// Commit appends the event, then wakes subscribers. The append is the durable
// part: Notify cannot block it, and a crash between the two is recovered by
// CatchUp.
func (b *Broker) Commit(ctx context.Context, orgID, kind, documentID, payload string) (store.StoredEvent, error) {
	ev, err := b.log.AppendEvent(ctx, orgID, kind, documentID, payload)
	if err != nil {
		return store.StoredEvent{}, err
	}
	b.Notify(orgID, ev)
	return ev, nil
}

// Notify wakes live subscribers for orgID. It does not write the log and it
// does not wait for a client to read.
func (b *Broker) Notify(orgID string, ev store.StoredEvent) {
	for _, sub := range b.current(orgID) {
		select {
		case sub.ch <- ev:
		default:
			sub.rung()
		}
	}
}

// Wake tells orgID's live streams to read the log from their cursor. Use it
// after a store transaction appended an event without going through Commit.
func (b *Broker) Wake(orgID string) {
	for _, sub := range b.current(orgID) {
		sub.rung()
	}
}

// Subscribers is the number of live receivers for orgID.
func (b *Broker) Subscribers(orgID string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs[orgID])
}

func (b *Broker) current(orgID string) []*subscription {
	b.mu.Lock()
	defer b.mu.Unlock()
	list := make([]*subscription, 0, len(b.subs[orgID]))
	for sub := range b.subs[orgID] {
		list = append(list, sub)
	}
	return list
}

func (s *subscription) rung() {
	select {
	case s.ring <- struct{}{}:
	default:
	}
}

// Subscribe registers a live receiver. The returned channel has a buffer of
// one so a slow client does not stall Commit. Cancel removes it.
func (b *Broker) Subscribe(orgID string) (<-chan store.StoredEvent, func()) {
	sub, cancel := b.subscribe(orgID)
	return sub.ch, cancel
}

func (b *Broker) subscribe(orgID string) (*subscription, func()) {
	sub := &subscription{ch: make(chan store.StoredEvent, 1), ring: make(chan struct{}, 1)}
	b.mu.Lock()
	if b.subs[orgID] == nil {
		b.subs[orgID] = map[*subscription]struct{}{}
	}
	b.subs[orgID][sub] = struct{}{}
	b.mu.Unlock()
	var once sync.Once
	return sub, func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs[orgID], sub)
			b.mu.Unlock()
		})
	}
}

// CatchUp reads the log after the client's cursor. It does not require that
// this process was the one that committed the events.
func (b *Broker) CatchUp(ctx context.Context, orgID string, after int64) ([]store.StoredEvent, error) {
	var out []store.StoredEvent
	for {
		page, err := b.log.EventsAfter(ctx, orgID, after, pageSize)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			return out, nil
		}
		for _, ev := range page {
			next, ok := Advance(after, ev.ID)
			if !ok {
				continue
			}
			out = append(out, ev)
			after = next
		}
		if len(page) < pageSize {
			return out, nil
		}
	}
}

// Serve writes missed events, then live ones, until ctx ends. Event ids at or
// below the cursor are skipped so a client can de-duplicate a replay. A ring
// reads the log from the cursor, which covers dropped and out-of-band events.
func (b *Broker) Serve(ctx context.Context, w io.Writer, orgID string, after int64) error {
	sub, cancel := b.subscribe(orgID)
	defer cancel()
	after, err := b.writeMissed(ctx, w, orgID, after)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sub.ring:
			if after, err = b.writeMissed(ctx, w, orgID, after); err != nil {
				return err
			}
		case ev := <-sub.ch:
			next, ok := Advance(after, ev.ID)
			if !ok {
				continue
			}
			if err := WriteSSE(w, ev); err != nil {
				return err
			}
			after = next
			flush(w)
		}
	}
}

func (b *Broker) writeMissed(ctx context.Context, w io.Writer, orgID string, after int64) (int64, error) {
	missed, err := b.CatchUp(ctx, orgID, after)
	if err != nil {
		return after, err
	}
	for _, ev := range missed {
		next, ok := Advance(after, ev.ID)
		if !ok {
			continue
		}
		if err := WriteSSE(w, ev); err != nil {
			return after, err
		}
		after = next
		flush(w)
	}
	return after, nil
}

// Advance moves the cursor when id is new. A replay of an id the client
// already applied returns false.
func Advance(last, id int64) (int64, bool) {
	if id <= last {
		return last, false
	}
	return id, true
}

// LastEventID parses the SSE Last-Event-ID header. An empty header is zero.
func LastEventID(header string) (int64, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(header, 10, 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("last event id")
	}
	return n, nil
}

// WriteSSE writes one event. The id line is the durable cursor.
func WriteSSE(w io.Writer, ev store.StoredEvent) error {
	payload := json.RawMessage(ev.Payload)
	if len(payload) == 0 {
		payload = json.RawMessage("{}")
	}
	if !json.Valid(payload) {
		return fmt.Errorf("event payload")
	}
	body, err := json.Marshal(struct {
		ID         int64           `json:"id"`
		Kind       string          `json:"kind"`
		DocumentID string          `json:"document_id,omitempty"`
		Payload    json.RawMessage `json:"payload"`
	}{ev.ID, ev.Kind, ev.DocumentID, payload})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Kind, body)
	return err
}

func flush(w io.Writer) {
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
}
