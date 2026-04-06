package model

import "time"

// SpanKind describes the relationship between a span, its parents, and its children.
// Values mirror opentelemetry-proto SpanKind.
type SpanKind int

const (
	SpanKindUnspecified SpanKind = iota // SPAN_KIND_UNSPECIFIED
	SpanKindInternal                    // SPAN_KIND_INTERNAL – default for in-process work
	SpanKindServer                      // SPAN_KIND_SERVER – incoming synchronous RPC/HTTP
	SpanKindClient                      // SPAN_KIND_CLIENT – outgoing synchronous RPC/HTTP
	SpanKindProducer                    // SPAN_KIND_PRODUCER – outgoing async message
	SpanKindConsumer                    // SPAN_KIND_CONSUMER – incoming async message
)

func (k SpanKind) String() string {
	switch k {
	case SpanKindInternal:
		return "internal"
	case SpanKindServer:
		return "server"
	case SpanKindClient:
		return "client"
	case SpanKindProducer:
		return "producer"
	case SpanKindConsumer:
		return "consumer"
	default:
		return "unspecified"
	}
}

// StatusCode is the canonical status of a finished span.
type StatusCode int

const (
	StatusUnset StatusCode = iota // STATUS_CODE_UNSET – default; not an error
	StatusOK                      // STATUS_CODE_OK    – explicitly successful
	StatusError                   // STATUS_CODE_ERROR – a logical error occurred
)

// Status carries the final status of a span.
type Status struct {
	Code    StatusCode
	Message string // human-readable description, non-empty only on StatusError
}

// Event is a time-stamped structured log message attached to a span.
type Event struct {
	Timestamp             time.Time
	Name                  string
	Attributes            Attributes
	DroppedAttributeCount int
}

// Link is a reference to another span, possibly in a different trace.
// Used to express causal relationships across trace boundaries.
type Link struct {
	TraceID               TraceID
	SpanID                SpanID
	TraceState            string
	Attributes            Attributes
	DroppedAttributeCount int
}

// Resource describes the entity that produced the telemetry (e.g. a service instance).
// Attributes typically include "service.name", "service.version", "host.name", etc.
type Resource struct {
	Attributes            Attributes
	DroppedAttributeCount int
}

// ServiceName returns the "service.name" resource attribute, or "" if unset.
func (r Resource) ServiceName() string {
	return r.Attributes.GetString("service.name")
}

// InstrumentationScope identifies the library or component that emitted the span.
type InstrumentationScope struct {
	Name                  string
	Version               string
	Attributes            Attributes
	DroppedAttributeCount int
}

// Span is a single unit of work within a distributed trace.
// Field names and semantics follow the OpenTelemetry data model
// (opentelemetry-proto/opentelemetry/proto/trace/v1/trace.proto).
type Span struct {
	// Identity
	TraceID    TraceID
	SpanID     SpanID
	ParentID   SpanID // zero value indicates a root span
	TraceState string // W3C tracestate header value

	// Description
	Name string
	Kind SpanKind

	// Timing (nanosecond precision matches OTLP wire format)
	StartTime time.Time
	EndTime   time.Time

	// Span-level attributes
	Attributes            Attributes
	DroppedAttributeCount int

	// Time-stamped annotations
	Events            []Event
	DroppedEventCount int

	// References to causally-related spans in other traces
	Links            []Link
	DroppedLinkCount int

	// Outcome
	Status Status

	// Origin
	Resource             Resource
	InstrumentationScope InstrumentationScope
}

// Duration returns the elapsed time of the span, clamped to zero for spans
// with clock-skew or missing timestamps (EndTime < StartTime).
func (s Span) Duration() time.Duration {
	if d := s.EndTime.Sub(s.StartTime); d > 0 {
		return d
	}
	return 0
}

// IsRoot reports whether the span has no parent span.
func (s Span) IsRoot() bool {
	return s.ParentID.IsZero()
}

// ServiceName is a convenience accessor for the Resource "service.name" attribute.
func (s Span) ServiceName() string {
	return s.Resource.ServiceName()
}
