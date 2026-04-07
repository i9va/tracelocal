package store_test

import (
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/i9va/tracelocal/internal/model"
	"github.com/i9va/tracelocal/internal/store"
)

func discardLog() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// traceID builds a TraceID with the given byte in position 0.
func traceID(b byte) model.TraceID {
	var id model.TraceID
	id[0] = b
	return id
}

// traceID2 builds a TraceID using two bytes, giving 256×256 unique values.
func traceID2(a, b byte) model.TraceID {
	var id model.TraceID
	id[0] = a
	id[1] = b
	return id
}

// spanID builds a SpanID with the given byte in position 0.
func spanID(b byte) model.SpanID {
	var id model.SpanID
	id[0] = b
	return id
}

func makeSpan(tid model.TraceID, sid model.SpanID) model.Span {
	return model.Span{
		TraceID:   tid,
		SpanID:    sid,
		Name:      "test-span",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Millisecond),
	}
}

func TestAddAndGet(t *testing.T) {
	s := store.New(10, discardLog())
	tid := traceID(1)

	if err := s.Add(makeSpan(tid, spanID(1))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := s.Add(makeSpan(tid, spanID(2))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tr, ok := s.Get(tid)
	if !ok {
		t.Fatal("expected trace to exist")
	}
	if len(tr.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(tr.Spans))
	}
}

func TestRingBufferEviction(t *testing.T) {
	s := store.New(2, discardLog())

	t1, t2, t3 := traceID(1), traceID(2), traceID(3)
	s.Add(makeSpan(t1, spanID(1))) //nolint:errcheck
	s.Add(makeSpan(t2, spanID(2))) //nolint:errcheck
	s.Add(makeSpan(t3, spanID(3))) //nolint:errcheck // should evict t1

	if _, ok := s.Get(t1); ok {
		t.Error("trace 1 should have been evicted")
	}
	if _, ok := s.Get(t3); !ok {
		t.Error("trace 3 should exist")
	}
}

func TestAddZeroTraceID(t *testing.T) {
	s := store.New(10, discardLog())
	if err := s.Add(makeSpan(model.TraceID{}, spanID(1))); err == nil {
		t.Error("expected error for zero TraceID")
	}
	if got := s.Dropped(); got != 1 {
		t.Errorf("expected 1 dropped span, got %d", got)
	}
}

func TestHasError(t *testing.T) {
	s := store.New(10, discardLog())
	tid := traceID(1)

	span := makeSpan(tid, spanID(1))
	span.Status = model.Status{Code: model.StatusError, Message: "oops"}
	s.Add(span) //nolint:errcheck

	tr, ok := s.Get(tid)
	if !ok {
		t.Fatal("expected trace to exist")
	}
	if !tr.HasError {
		t.Error("expected HasError to be true for trace with error span")
	}
}

// TestConcurrentAccess verifies that concurrent Add and GetAll calls do not
// race. Run with: go test -race ./internal/store/...
func TestConcurrentAccess(t *testing.T) {
	const goroutines = 10
	const iters = 25 // goroutines * iters unique IDs fit in two bytes

	s := store.New(100, discardLog()) // capacity < total writes to trigger evictions

	var wg sync.WaitGroup

	// Concurrent writers: each goroutine writes to unique trace IDs.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				tid := traceID2(byte(i), byte(j))
				_ = s.Add(makeSpan(tid, spanID(byte(j))))
			}
		}(i)
	}

	// Concurrent readers: GetAll runs alongside the writers.
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iters; j++ {
				_ = s.GetAll()
			}
		}()
	}

	wg.Wait()

	// Sanity: store must not exceed capacity.
	if got := len(s.GetAll()); got > 100 {
		t.Errorf("store exceeded capacity: len=%d", got)
	}
}
