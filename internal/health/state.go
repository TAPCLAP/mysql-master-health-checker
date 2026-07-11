package health

import (
	"sync"
	"time"
)

// Result is the cached outcome of a background MySQL check.
type Result struct {
	Available     bool
	ReadOnly      bool
	ReadOnlyKnown bool
	Healthy       bool
	Reason        string
	CheckedAt     time.Time
}

// Store keeps the latest health check result in memory.
type Store struct {
	mu     sync.RWMutex
	result Result
}

// NewStore creates a store with an unknown initial state.
func NewStore() *Store {
	return &Store{
		result: Result{
			Healthy: false,
			Reason:  "initial check pending",
		},
	}
}

// Update replaces the cached result.
func (s *Store) Update(result Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.result = result
}

// Snapshot returns a copy of the current result.
func (s *Store) Snapshot() Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.result
}
