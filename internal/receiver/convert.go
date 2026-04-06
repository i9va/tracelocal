package receiver

import (
	"math"
	"time"

	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/henriqueholanda/tracelocal/internal/model"
)

// spanFromProto converts a proto Span (with its resource and scope context) into
// the internal model.Span. Invalid ID byte slices are silently zeroed.
func spanFromProto(
	pb *tracepb.Span,
	res *resourcepb.Resource,
	scope *commonpb.InstrumentationScope,
) model.Span {
	return model.Span{
		TraceID:    traceIDFromBytes(pb.GetTraceId()),
		SpanID:     spanIDFromBytes(pb.GetSpanId()),
		ParentID:   spanIDFromBytes(pb.GetParentSpanId()),
		TraceState: pb.GetTraceState(),
		Name:       pb.GetName(),
		Kind:       spanKindFromProto(pb.GetKind()),
		StartTime:  timeFromUnixNano(pb.GetStartTimeUnixNano()),
		EndTime:    timeFromUnixNano(pb.GetEndTimeUnixNano()),

		Attributes:            attrsFromProto(pb.GetAttributes()),
		DroppedAttributeCount: int(pb.GetDroppedAttributesCount()),

		Events:            eventsFromProto(pb.GetEvents()),
		DroppedEventCount: int(pb.GetDroppedEventsCount()),

		Links:            linksFromProto(pb.GetLinks()),
		DroppedLinkCount: int(pb.GetDroppedLinksCount()),

		Status: statusFromProto(pb.GetStatus()),

		Resource:             resourceFromProto(res),
		InstrumentationScope: scopeFromProto(scope),
	}
}

// --- ID helpers -------------------------------------------------------------

func traceIDFromBytes(b []byte) model.TraceID {
	var id model.TraceID
	if len(b) == 16 {
		copy(id[:], b)
	}
	return id
}

func spanIDFromBytes(b []byte) model.SpanID {
	var id model.SpanID
	if len(b) == 8 {
		copy(id[:], b)
	}
	return id
}

// --- Time -------------------------------------------------------------------

func timeFromUnixNano(ns uint64) time.Time {
	if ns == 0 {
		return time.Time{}
	}
	// Guard against overflow: uint64 values above MaxInt64 would wrap negative.
	if ns > math.MaxInt64 {
		ns = math.MaxInt64
	}
	return time.Unix(0, int64(ns))
}

// --- Attributes -------------------------------------------------------------

func attrsFromProto(kvs []*commonpb.KeyValue) model.Attributes {
	if len(kvs) == 0 {
		return nil
	}
	out := make(model.Attributes, 0, len(kvs))
	for _, kv := range kvs {
		out = append(out, model.Attribute{
			Key:   kv.GetKey(),
			Value: anyValueFromProto(kv.GetValue()),
		})
	}
	return out
}

func anyValueFromProto(v *commonpb.AnyValue) model.AttributeValue {
	if v == nil {
		return model.StringValue("")
	}
	switch val := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return model.StringValue(val.StringValue)
	case *commonpb.AnyValue_BoolValue:
		return model.BoolValue(val.BoolValue)
	case *commonpb.AnyValue_IntValue:
		return model.Int64Value(val.IntValue)
	case *commonpb.AnyValue_DoubleValue:
		return model.Float64Value(val.DoubleValue)
	case *commonpb.AnyValue_ArrayValue:
		return arrayValueFromProto(val.ArrayValue)
	default:
		// kvlist and bytes are not in the model; fall back to string representation
		return model.StringValue(v.String())
	}
}

func arrayValueFromProto(arr *commonpb.ArrayValue) model.AttributeValue {
	if arr == nil || len(arr.Values) == 0 {
		return model.StringSliceValue(nil)
	}
	// Inspect each element to handle heterogeneous arrays correctly. OTel
	// recommends homogeneous arrays but the spec does not forbid mixed types.
	// Any type mismatch falls back to a string-slice representation.
	switch arr.Values[0].Value.(type) {
	case *commonpb.AnyValue_BoolValue:
		out := make([]bool, 0, len(arr.Values))
		for _, v := range arr.Values {
			bv, ok := v.Value.(*commonpb.AnyValue_BoolValue)
			if !ok {
				return arrayToStringSlice(arr.Values)
			}
			out = append(out, bv.BoolValue)
		}
		return model.BoolSliceValue(out)
	case *commonpb.AnyValue_IntValue:
		out := make([]int64, 0, len(arr.Values))
		for _, v := range arr.Values {
			iv, ok := v.Value.(*commonpb.AnyValue_IntValue)
			if !ok {
				return arrayToStringSlice(arr.Values)
			}
			out = append(out, iv.IntValue)
		}
		return model.Int64SliceValue(out)
	case *commonpb.AnyValue_DoubleValue:
		out := make([]float64, 0, len(arr.Values))
		for _, v := range arr.Values {
			dv, ok := v.Value.(*commonpb.AnyValue_DoubleValue)
			if !ok {
				return arrayToStringSlice(arr.Values)
			}
			out = append(out, dv.DoubleValue)
		}
		return model.Float64SliceValue(out)
	default:
		return arrayToStringSlice(arr.Values)
	}
}

// arrayToStringSlice converts every element to its string representation.
// Used as a fallback for heterogeneous or string-typed arrays.
func arrayToStringSlice(vals []*commonpb.AnyValue) model.AttributeValue {
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = anyValueFromProto(v).String()
	}
	return model.StringSliceValue(out)
}

// --- SpanKind ---------------------------------------------------------------

func spanKindFromProto(k tracepb.Span_SpanKind) model.SpanKind {
	switch k {
	case tracepb.Span_SPAN_KIND_INTERNAL:
		return model.SpanKindInternal
	case tracepb.Span_SPAN_KIND_SERVER:
		return model.SpanKindServer
	case tracepb.Span_SPAN_KIND_CLIENT:
		return model.SpanKindClient
	case tracepb.Span_SPAN_KIND_PRODUCER:
		return model.SpanKindProducer
	case tracepb.Span_SPAN_KIND_CONSUMER:
		return model.SpanKindConsumer
	default:
		return model.SpanKindUnspecified
	}
}

// --- Status -----------------------------------------------------------------

func statusFromProto(s *tracepb.Status) model.Status {
	if s == nil {
		return model.Status{}
	}
	var code model.StatusCode
	switch s.GetCode() {
	case tracepb.Status_STATUS_CODE_OK:
		code = model.StatusOK
	case tracepb.Status_STATUS_CODE_ERROR:
		code = model.StatusError
	default:
		code = model.StatusUnset
	}
	return model.Status{Code: code, Message: s.GetMessage()}
}

// --- Events -----------------------------------------------------------------

func eventsFromProto(evs []*tracepb.Span_Event) []model.Event {
	if len(evs) == 0 {
		return nil
	}
	out := make([]model.Event, len(evs))
	for i, e := range evs {
		out[i] = model.Event{
			Timestamp:             timeFromUnixNano(e.GetTimeUnixNano()),
			Name:                  e.GetName(),
			Attributes:            attrsFromProto(e.GetAttributes()),
			DroppedAttributeCount: int(e.GetDroppedAttributesCount()),
		}
	}
	return out
}

// --- Links ------------------------------------------------------------------

func linksFromProto(links []*tracepb.Span_Link) []model.Link {
	if len(links) == 0 {
		return nil
	}
	out := make([]model.Link, len(links))
	for i, l := range links {
		out[i] = model.Link{
			TraceID:               traceIDFromBytes(l.GetTraceId()),
			SpanID:                spanIDFromBytes(l.GetSpanId()),
			TraceState:            l.GetTraceState(),
			Attributes:            attrsFromProto(l.GetAttributes()),
			DroppedAttributeCount: int(l.GetDroppedAttributesCount()),
		}
	}
	return out
}

// --- Resource & InstrumentationScope ----------------------------------------

func resourceFromProto(r *resourcepb.Resource) model.Resource {
	if r == nil {
		return model.Resource{}
	}
	return model.Resource{
		Attributes:            attrsFromProto(r.GetAttributes()),
		DroppedAttributeCount: int(r.GetDroppedAttributesCount()),
	}
}

func scopeFromProto(s *commonpb.InstrumentationScope) model.InstrumentationScope {
	if s == nil {
		return model.InstrumentationScope{}
	}
	return model.InstrumentationScope{
		Name:                  s.GetName(),
		Version:               s.GetVersion(),
		Attributes:            attrsFromProto(s.GetAttributes()),
		DroppedAttributeCount: int(s.GetDroppedAttributesCount()),
	}
}
