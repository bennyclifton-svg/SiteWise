package httpapi

import (
	"context"
	"sync"
)

// The single server ends every open tab's event stream when its session ends.
type sessionStreams struct {
	mu     sync.Mutex
	active map[string]map[*context.CancelFunc]struct{}
}

func newSessionStreams() *sessionStreams {
	return &sessionStreams{active: make(map[string]map[*context.CancelFunc]struct{})}
}

func (s *sessionStreams) add(id string, cancel context.CancelFunc) func() {
	s.mu.Lock()
	if s.active[id] == nil {
		s.active[id] = make(map[*context.CancelFunc]struct{})
	}
	s.active[id][&cancel] = struct{}{}
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.active[id], &cancel)
		if len(s.active[id]) == 0 {
			delete(s.active, id)
		}
	}
}

func (s *sessionStreams) end(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for cancel := range s.active[id] {
		(*cancel)()
	}
	delete(s.active, id)
}
