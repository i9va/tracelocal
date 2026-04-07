package store

import (
	"container/list"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/i9va/tracelocal/internal/model"
)

const defaultCapacity = 1000

// Store holds traces in an in-memory ring buffer.
type Store struct {
	mu       sync.RWMutex
	traces   map[model.TraceID]*model.Trace
	order    *list.List // elements are model.TraceID, front = oldest
	capacity int
	log      *slog.Logger
	dropped  atomic.Int64 // spans rejected due to zero TraceID
}

// New creates a Store with the given capacity (falls back to default if <= 0).
func New(capacity int, log *slog.Logger) *Store {
	if capacity <= 0 {
		capacity = defaultCapacity
	}
	return &Store{
		traces:   make(map[model.TraceID]*model.Trace, capacity),
		order:    list.New(),
		capacity: capacity,
		log:      log,
	}
}

// Add inserts or appends spans to the matching trace.
// When the ring buffer is full the oldest trace is evicted.
func (s *Store) Add(span model.Span) error {
	if span.TraceID.IsZero() {
		s.dropped.Add(1)
		return fmt.Errorf("store: span has zero TraceID")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.traces[span.TraceID]; ok {
		t.Spans = append(t.Spans, span)
		if span.Status.Code == model.StatusError {
			t.HasError = true
		}
		return nil
	}

	// Evict the oldest trace when at capacity. list.Remove is O(1).
	if s.order.Len() >= s.capacity {
		front := s.order.Front()
		if front != nil {
			oldest := front.Value.(model.TraceID)
			s.order.Remove(front)
			delete(s.traces, oldest)
			s.log.Info("evicted oldest trace", "trace", oldest, "capacity", s.capacity)
		}
	}

	t := &model.Trace{
		ID:       span.TraceID,
		Spans:    []model.Span{span},
		HasError: span.Status.Code == model.StatusError,
	}
	s.traces[span.TraceID] = t
	s.order.PushBack(span.TraceID)
	return nil
}

// GetAll returns a snapshot of all traces, newest first.
// Each returned Trace is a copy; callers may read it without holding any lock.
func (s *Store) GetAll() []model.Trace {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.Trace, 0, s.order.Len())
	for e := s.order.Back(); e != nil; e = e.Prev() {
		t := s.traces[e.Value.(model.TraceID)]
		spans := make([]model.Span, len(t.Spans))
		copy(spans, t.Spans)
		out = append(out, model.Trace{ID: t.ID, Spans: spans, HasError: t.HasError})
	}
	return out
}

// Get returns a snapshot of a single trace by ID.
// The returned Trace is a copy; callers may read it without holding any lock.
func (s *Store) Get(traceID model.TraceID) (model.Trace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.traces[traceID]
	if !ok {
		return model.Trace{}, false
	}
	spans := make([]model.Span, len(t.Spans))
	copy(spans, t.Spans)
	return model.Trace{ID: t.ID, Spans: spans, HasError: t.HasError}, true
}

// Dropped returns the number of spans rejected due to a zero TraceID.
func (s *Store) Dropped() int64 {
	return s.dropped.Load()
}
