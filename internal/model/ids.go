package model

import (
	"encoding/hex"
	"fmt"
)

// TraceID is a 16-byte unique identifier for a trace (W3C trace-context format).
type TraceID [16]byte

// SpanID is an 8-byte unique identifier for a span (W3C trace-context format).
type SpanID [8]byte

// String returns the lowercase hex encoding of the TraceID.
func (t TraceID) String() string { return hex.EncodeToString(t[:]) }

// IsZero reports whether the TraceID is the zero value.
func (t TraceID) IsZero() bool { return t == (TraceID{}) }

// TraceIDFromHex parses a 32-character hex string into a TraceID.
func TraceIDFromHex(s string) (TraceID, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return TraceID{}, fmt.Errorf("model: invalid TraceID %q: %w", s, err)
	}
	if len(b) != 16 {
		return TraceID{}, fmt.Errorf("model: TraceID must be 16 bytes, got %d", len(b))
	}
	var id TraceID
	copy(id[:], b)
	return id, nil
}

// String returns the lowercase hex encoding of the SpanID.
func (s SpanID) String() string { return hex.EncodeToString(s[:]) }

// IsZero reports whether the SpanID is the zero value.
func (s SpanID) IsZero() bool { return s == (SpanID{}) }

// SpanIDFromHex parses a 16-character hex string into a SpanID.
func SpanIDFromHex(s string) (SpanID, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return SpanID{}, fmt.Errorf("model: invalid SpanID %q: %w", s, err)
	}
	if len(b) != 8 {
		return SpanID{}, fmt.Errorf("model: SpanID must be 8 bytes, got %d", len(b))
	}
	var id SpanID
	copy(id[:], b)
	return id, nil
}
