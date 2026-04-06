package model_test

import (
	"testing"
	"time"

	"github.com/henriqueholanda/tracelocal/internal/model"
)

// --- ID tests ---------------------------------------------------------------

func TestTraceIDRoundtrip(t *testing.T) {
	const hex = "4bf92f3577b34da6a3ce929d0e0e4736"
	id, err := model.TraceIDFromHex(hex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.String() != hex {
		t.Errorf("got %q, want %q", id.String(), hex)
	}
}

func TestSpanIDRoundtrip(t *testing.T) {
	const hex = "00f067aa0ba902b7"
	id, err := model.SpanIDFromHex(hex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id.String() != hex {
		t.Errorf("got %q, want %q", id.String(), hex)
	}
}

func TestTraceIDFromHex_invalid(t *testing.T) {
	cases := []string{
		"not-hex",
		"4bf92f", // too short
		"4bf92f3577b34da6a3ce929d0e0e47364bf92f3577b34da6", // too long
	}
	for _, c := range cases {
		if _, err := model.TraceIDFromHex(c); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

func TestIDZero(t *testing.T) {
	var tid model.TraceID
	var sid model.SpanID
	if !tid.IsZero() {
		t.Error("zero TraceID should be zero")
	}
	if !sid.IsZero() {
		t.Error("zero SpanID should be zero")
	}
}

// --- Attribute tests --------------------------------------------------------

func TestAttributes_Get(t *testing.T) {
	attrs := model.Attributes{
		{Key: "http.method", Value: model.StringValue("GET")},
		{Key: "http.status_code", Value: model.Int64Value(200)},
		{Key: "error", Value: model.BoolValue(false)},
	}

	v, ok := attrs.Get("http.method")
	if !ok || v.AsString() != "GET" {
		t.Errorf("unexpected value: %v %v", ok, v)
	}

	v, ok = attrs.Get("http.status_code")
	if !ok || v.AsInt64() != 200 {
		t.Errorf("unexpected value: %v %v", ok, v)
	}

	if _, ok := attrs.Get("missing"); ok {
		t.Error("expected missing key to be absent")
	}
}

func TestAttributeValue_String(t *testing.T) {
	cases := []struct {
		v    model.AttributeValue
		want string
	}{
		{model.StringValue("hello"), "hello"},
		{model.BoolValue(true), "true"},
		{model.Int64Value(42), "42"},
		{model.Float64Value(3.14), "3.14"},
	}
	for _, c := range cases {
		if got := c.v.String(); got != c.want {
			t.Errorf("String() = %q, want %q", got, c.want)
		}
	}
}

// --- Span tests -------------------------------------------------------------

func makeSpan(traceHex, spanHex, parentHex string) model.Span {
	tid, _ := model.TraceIDFromHex(traceHex)
	sid, _ := model.SpanIDFromHex(spanHex)
	var pid model.SpanID
	if parentHex != "" {
		pid, _ = model.SpanIDFromHex(parentHex)
	}
	return model.Span{
		TraceID:   tid,
		SpanID:    sid,
		ParentID:  pid,
		Name:      "op",
		Kind:      model.SpanKindServer,
		StartTime: time.Unix(0, 0),
		EndTime:   time.Unix(0, int64(time.Millisecond)),
		Resource: model.Resource{
			Attributes: model.Attributes{
				{Key: "service.name", Value: model.StringValue("svc-a")},
			},
		},
	}
}

func TestSpan_IsRoot(t *testing.T) {
	root := makeSpan("4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7", "")
	child := makeSpan("4bf92f3577b34da6a3ce929d0e0e4736", "0102030405060708", "00f067aa0ba902b7")

	if !root.IsRoot() {
		t.Error("expected root span to be root")
	}
	if child.IsRoot() {
		t.Error("expected child span to not be root")
	}
}

func TestSpan_Duration(t *testing.T) {
	s := makeSpan("4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7", "")
	if s.Duration() != time.Millisecond {
		t.Errorf("got %v, want %v", s.Duration(), time.Millisecond)
	}
}

func TestSpan_ServiceName(t *testing.T) {
	s := makeSpan("4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7", "")
	if s.ServiceName() != "svc-a" {
		t.Errorf("got %q, want %q", s.ServiceName(), "svc-a")
	}
}

func TestSpanKind_String(t *testing.T) {
	cases := []struct {
		k    model.SpanKind
		want string
	}{
		{model.SpanKindInternal, "internal"},
		{model.SpanKindServer, "server"},
		{model.SpanKindClient, "client"},
		{model.SpanKindProducer, "producer"},
		{model.SpanKindConsumer, "consumer"},
		{model.SpanKindUnspecified, "unspecified"},
	}
	for _, c := range cases {
		if got := c.k.String(); got != c.want {
			t.Errorf("SpanKind(%d).String() = %q, want %q", c.k, got, c.want)
		}
	}
}

// --- Trace tests ------------------------------------------------------------

func TestTrace_RootSpan(t *testing.T) {
	root := makeSpan("4bf92f3577b34da6a3ce929d0e0e4736", "00f067aa0ba902b7", "")
	child := makeSpan("4bf92f3577b34da6a3ce929d0e0e4736", "0102030405060708", "00f067aa0ba902b7")

	tid, _ := model.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	tr := model.Trace{ID: tid, Spans: []model.Span{child, root}} // root is second

	if got := tr.RootSpan(); got.SpanID != root.SpanID {
		t.Errorf("RootSpan returned wrong span")
	}
}

func TestTrace_Duration(t *testing.T) {
	now := time.Now()
	tid, _ := model.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	s1id, _ := model.SpanIDFromHex("00f067aa0ba902b7")
	s2id, _ := model.SpanIDFromHex("0102030405060708")

	tr := model.Trace{
		ID: tid,
		Spans: []model.Span{
			{TraceID: tid, SpanID: s1id, StartTime: now, EndTime: now.Add(10 * time.Millisecond)},
			{TraceID: tid, SpanID: s2id, StartTime: now.Add(2 * time.Millisecond), EndTime: now.Add(15 * time.Millisecond)},
		},
	}
	if got := tr.Duration(); got != 15*time.Millisecond {
		t.Errorf("got %v, want %v", got, 15*time.Millisecond)
	}
}

func TestTrace_Duration_empty(t *testing.T) {
	tid, _ := model.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	tr := model.Trace{ID: tid}
	if tr.Duration() != 0 {
		t.Error("empty trace should have zero duration")
	}
}

func TestTrace_HasError(t *testing.T) {
	tid, _ := model.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	tr := model.Trace{ID: tid, HasError: true}
	if !tr.HasError {
		t.Error("expected HasError to be true")
	}
}
