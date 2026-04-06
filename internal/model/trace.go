package model

import "time"

// Trace groups all spans that share the same TraceID.
type Trace struct {
	ID       TraceID
	Spans    []Span
	HasError bool // true if any span in the trace has StatusError
}

// RootSpan returns the first span whose ParentID is zero (root span),
// falling back to the first span in the slice if none qualifies.
func (t Trace) RootSpan() *Span {
	for i := range t.Spans {
		if t.Spans[i].IsRoot() {
			return &t.Spans[i]
		}
	}
	if len(t.Spans) > 0 {
		return &t.Spans[0]
	}
	return nil
}

// Duration returns the wall-clock duration of the trace: from the earliest
// span start to the latest span end across all spans.
func (t Trace) Duration() time.Duration {
	if len(t.Spans) == 0 {
		return 0
	}
	earliest := t.Spans[0].StartTime
	latest := t.Spans[0].EndTime
	for _, s := range t.Spans[1:] {
		if s.StartTime.Before(earliest) {
			earliest = s.StartTime
		}
		if s.EndTime.After(latest) {
			latest = s.EndTime
		}
	}
	return latest.Sub(earliest)
}
