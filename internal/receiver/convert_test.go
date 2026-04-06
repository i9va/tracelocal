package receiver

import (
	"testing"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/henriqueholanda/tracelocal/internal/model"
)

var (
	testTraceIDBytes = []byte{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	testSpanIDBytes  = []byte{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
)

func mustTraceID(b []byte) model.TraceID { return traceIDFromBytes(b) }
func mustSpanID(b []byte) model.SpanID   { return spanIDFromBytes(b) }

func pbSpan() *tracepb.Span {
	return &tracepb.Span{
		TraceId:           testTraceIDBytes,
		SpanId:            testSpanIDBytes,
		Name:              "GET /api/users",
		Kind:              tracepb.Span_SPAN_KIND_SERVER,
		StartTimeUnixNano: uint64(time.Unix(1_000, 0).UnixNano()),
		EndTimeUnixNano:   uint64(time.Unix(1_000, 0).Add(5 * time.Millisecond).UnixNano()),
		Attributes: []*commonpb.KeyValue{
			{Key: "http.method", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "GET"}}},
			{Key: "http.status_code", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: 200}}},
			{Key: "error", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_BoolValue{BoolValue: false}}},
		},
		Status: &tracepb.Status{Code: tracepb.Status_STATUS_CODE_OK},
		Events: []*tracepb.Span_Event{
			{
				TimeUnixNano: uint64(time.Unix(1_000, 0).Add(2 * time.Millisecond).UnixNano()),
				Name:         "cache.miss",
			},
		},
	}
}

func pbResource() *resourcepb.Resource {
	return &resourcepb.Resource{
		Attributes: []*commonpb.KeyValue{
			{Key: "service.name", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "user-service"}}},
			{Key: "service.version", Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: "1.2.3"}}},
		},
	}
}

func pbScope() *commonpb.InstrumentationScope {
	return &commonpb.InstrumentationScope{Name: "mylib", Version: "0.1.0"}
}

func TestSpanFromProto_identity(t *testing.T) {
	s := spanFromProto(pbSpan(), pbResource(), pbScope())

	if s.TraceID != mustTraceID(testTraceIDBytes) {
		t.Errorf("TraceID mismatch: got %s", s.TraceID)
	}
	if s.SpanID != mustSpanID(testSpanIDBytes) {
		t.Errorf("SpanID mismatch: got %s", s.SpanID)
	}
	if s.ParentID != (model.SpanID{}) {
		t.Errorf("expected zero ParentID, got %s", s.ParentID)
	}
}

func TestSpanFromProto_fields(t *testing.T) {
	s := spanFromProto(pbSpan(), pbResource(), pbScope())

	if s.Name != "GET /api/users" {
		t.Errorf("Name: got %q", s.Name)
	}
	if s.Kind != model.SpanKindServer {
		t.Errorf("Kind: got %v", s.Kind)
	}
	if s.Duration() != 5*time.Millisecond {
		t.Errorf("Duration: got %v", s.Duration())
	}
	if s.Status.Code != model.StatusOK {
		t.Errorf("StatusCode: got %v", s.Status.Code)
	}
}

func TestSpanFromProto_resource(t *testing.T) {
	s := spanFromProto(pbSpan(), pbResource(), pbScope())

	if s.ServiceName() != "user-service" {
		t.Errorf("ServiceName: got %q", s.ServiceName())
	}
	if v := s.Resource.Attributes.GetString("service.version"); v != "1.2.3" {
		t.Errorf("service.version: got %q", v)
	}
}

func TestSpanFromProto_scope(t *testing.T) {
	s := spanFromProto(pbSpan(), pbResource(), pbScope())

	if s.InstrumentationScope.Name != "mylib" {
		t.Errorf("scope name: got %q", s.InstrumentationScope.Name)
	}
}

func TestSpanFromProto_attributes(t *testing.T) {
	s := spanFromProto(pbSpan(), pbResource(), pbScope())

	if v := s.Attributes.GetString("http.method"); v != "GET" {
		t.Errorf("http.method: got %q", v)
	}

	v, ok := s.Attributes.Get("http.status_code")
	if !ok || v.AsInt64() != 200 {
		t.Errorf("http.status_code: got %v %v", ok, v)
	}

	v, ok = s.Attributes.Get("error")
	if !ok || v.AsBool() != false {
		t.Errorf("error attr: got %v %v", ok, v)
	}
}

func TestSpanFromProto_events(t *testing.T) {
	s := spanFromProto(pbSpan(), pbResource(), pbScope())

	if len(s.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(s.Events))
	}
	if s.Events[0].Name != "cache.miss" {
		t.Errorf("event name: got %q", s.Events[0].Name)
	}
}

func TestSpanFromProto_nilResource(t *testing.T) {
	s := spanFromProto(pbSpan(), nil, nil)
	if s.ServiceName() != "" {
		t.Errorf("expected empty service name for nil resource, got %q", s.ServiceName())
	}
}

func TestSpanFromProto_badIDs(t *testing.T) {
	pb := pbSpan()
	pb.TraceId = []byte{0x01} // wrong length
	pb.SpanId = []byte{}      // empty

	s := spanFromProto(pb, nil, nil)
	if !s.TraceID.IsZero() {
		t.Error("expected zero TraceID for bad bytes")
	}
	if !s.SpanID.IsZero() {
		t.Error("expected zero SpanID for bad bytes")
	}
}

func TestSpanFromProto_errorStatus(t *testing.T) {
	pb := pbSpan()
	pb.Status = &tracepb.Status{Code: tracepb.Status_STATUS_CODE_ERROR, Message: "timeout"}

	s := spanFromProto(pb, nil, nil)
	if s.Status.Code != model.StatusError {
		t.Errorf("expected StatusError, got %v", s.Status.Code)
	}
	if s.Status.Message != "timeout" {
		t.Errorf("status message: got %q", s.Status.Message)
	}
}

func TestSpanFromProto_arrayAttribute(t *testing.T) {
	pb := pbSpan()
	pb.Attributes = []*commonpb.KeyValue{
		{
			Key: "tags",
			Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_ArrayValue{
				ArrayValue: &commonpb.ArrayValue{Values: []*commonpb.AnyValue{
					{Value: &commonpb.AnyValue_StringValue{StringValue: "a"}},
					{Value: &commonpb.AnyValue_StringValue{StringValue: "b"}},
				}},
			}},
		},
	}
	s := spanFromProto(pb, nil, nil)
	v, ok := s.Attributes.Get("tags")
	if !ok {
		t.Fatal("tags attribute missing")
	}
	if ss := v.AsStringSlice(); len(ss) != 2 || ss[0] != "a" || ss[1] != "b" {
		t.Errorf("tags: got %v", ss)
	}
}
