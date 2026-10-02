// Package store keeps live session hubs. The in-memory implementation suits a single instance.
package store

import (
	"errors"
	"sync"
	"time"

	"quizlive/internal/session"
)

var ErrFull = errors.New("too many concurrent sessions")

type Store interface {
	Add(h *session.Hub) error
	Get(id string) (*session.Hub, bool)
	Delete(id string)
}

type Memory struct {
	mu  sync.RWMutex
	max int
	m   map[string]*session.Hub
}

func NewMemory(max int) *Memory { return &Memory{max: max, m: map[string]*session.Hub{}} }

func (s *Memory) Add(h *session.Hub) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.max > 0 && len(s.m) >= s.max {
		return ErrFull
	}
	s.m[h.ID] = h
	return nil
}

func (s *Memory) Get(id string) (*session.Hub, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	h, ok := s.m[id]
	return h, ok
}

func (s *Memory) Delete(id string) {
	s.mu.Lock()
	h := s.m[id]
	delete(s.m, id)
	s.mu.Unlock()
	if h != nil {
		h.Stop()
	}
}

// Janitor evicts sessions idle for longer than ttl until stop is closed.
func (s *Memory) Janitor(ttl, every time.Duration, stop <-chan struct{}) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			s.mu.RLock()
			hubs := make([]*session.Hub, 0, len(s.m))
			for _, h := range s.m {
				hubs = append(hubs, h)
			}
			s.mu.RUnlock()
			for _, h := range hubs {
				if time.Since(h.IdleSince()) > ttl {
					s.Delete(h.ID)
				}
			}
		}
	}
}

// StopAll stops every hub (graceful shutdown).
func (s *Memory) StopAll() {
	s.mu.Lock()
	hubs := s.m
	s.m = map[string]*session.Hub{}
	s.mu.Unlock()
	for _, h := range hubs {
		h.Stop()
	}
}
