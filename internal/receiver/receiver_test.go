package receiver_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"

	"github.com/henriqueholanda/tracelocal/internal/receiver"
	"github.com/henriqueholanda/tracelocal/internal/store"
	"github.com/henriqueholanda/tracelocal/internal/model"
)

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func strAttr(key, val string) *commonpb.KeyValue {
	return &commonpb.KeyValue{
		Key:   key,
		Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: val}},
	}
}

// startReceiver binds on a free port and returns the address and a cancel func.
func startReceiver(t *testing.T, s *store.Store) (addr string, cancel context.CancelFunc) {
	t.Helper()

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = lis.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	recv := receiver.New(addr, s, discardLog())

	serveErr := make(chan error, 1)
	go func() { serveErr <- recv.Serve(ctx, lis) }()

	t.Cleanup(func() {
		cancel()
		if err := <-serveErr; err != nil {
			t.Errorf("Serve: %v", err)
		}
	})

	return addr, cancel
}

func TestEndToEnd_BasicExport(t *testing.T) {
	s := store.New(100, discardLog())
	addr, _ := startReceiver(t, s)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	now := time.Now()

	// Two-span trace: root + one child.
	var traceIDBytes = make([]byte, 16)
	var rootIDBytes = make([]byte, 8)
	var childIDBytes = make([]byte, 8)
	traceIDBytes[0] = 0xAB
	rootIDBytes[0] = 0x01
	childIDBytes[0] = 0x02

	req := &collectortracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{strAttr("service.name", "svc-a")},
				},
				ScopeSpans: []*tracepb.ScopeSpans{
					{
						Scope: &commonpb.InstrumentationScope{Name: "mylib", Version: "1.0.0"},
						Spans: []*tracepb.Span{
							{
								TraceId:           traceIDBytes,
								SpanId:            rootIDBytes,
								Name:              "root-op",
								Kind:              tracepb.Span_SPAN_KIND_SERVER,
								StartTimeUnixNano: uint64(now.UnixNano()),
								EndTimeUnixNano:   uint64(now.Add(10 * time.Millisecond).UnixNano()),
								Status:            &tracepb.Status{Code: tracepb.Status_STATUS_CODE_OK},
								Attributes: []*commonpb.KeyValue{
									strAttr("http.method", "GET"),
								},
							},
							{
								TraceId:           traceIDBytes,
								SpanId:            childIDBytes,
								ParentSpanId:      rootIDBytes,
								Name:              "child-op",
								Kind:              tracepb.Span_SPAN_KIND_CLIENT,
								StartTimeUnixNano: uint64(now.Add(2 * time.Millisecond).UnixNano()),
								EndTimeUnixNano:   uint64(now.Add(8 * time.Millisecond).UnixNano()),
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	if _, err := collectortracepb.NewTraceServiceClient(conn).Export(
		ctx, req, grpc.WaitForReady(true),
	); err != nil {
		t.Fatalf("Export: %v", err)
	}

	// Export is synchronous: spans are in the store before it returns.
	var expectedID model.TraceID
	copy(expectedID[:], traceIDBytes)

	tr, ok := s.Get(expectedID)
	if !ok {
		t.Fatal("trace not found in store after export")
	}
	if len(tr.Spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(tr.Spans))
	}

	root := tr.RootSpan()
	if root == nil {
		t.Fatal("no root span")
	}
	if root.Name != "root-op" {
		t.Errorf("root name: got %q, want %q", root.Name, "root-op")
	}
	if root.ServiceName() != "svc-a" {
		t.Errorf("service: got %q, want %q", root.ServiceName(), "svc-a")
	}
	if root.Kind != model.SpanKindServer {
		t.Errorf("kind: got %v, want server", root.Kind)
	}
	if root.Status.Code != model.StatusOK {
		t.Errorf("status: got %v, want OK", root.Status.Code)
	}
	if root.InstrumentationScope.Name != "mylib" {
		t.Errorf("scope: got %q, want mylib", root.InstrumentationScope.Name)
	}
	if v := root.Attributes.GetString("http.method"); v != "GET" {
		t.Errorf("http.method: got %q, want GET", v)
	}
}

func TestEndToEnd_MultipleTraces(t *testing.T) {
	s := store.New(100, discardLog())
	addr, _ := startReceiver(t, s)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	now := time.Now()
	client := collectortracepb.NewTraceServiceClient(conn)

	makeReq := func(tracePrefix, spanPrefix byte, name string) *collectortracepb.ExportTraceServiceRequest {
		tid := make([]byte, 16)
		sid := make([]byte, 8)
		tid[0] = tracePrefix
		sid[0] = spanPrefix
		return &collectortracepb.ExportTraceServiceRequest{
			ResourceSpans: []*tracepb.ResourceSpans{{
				ScopeSpans: []*tracepb.ScopeSpans{{
					Spans: []*tracepb.Span{{
						TraceId:           tid,
						SpanId:            sid,
						Name:              name,
						StartTimeUnixNano: uint64(now.UnixNano()),
						EndTimeUnixNano:   uint64(now.Add(time.Millisecond).UnixNano()),
					}},
				}},
			}},
		}
	}

	ctx := context.Background()
	for i, name := range []string{"op-1", "op-2", "op-3"} {
		if _, err := client.Export(ctx, makeReq(byte(i+1), byte(i+1), name), grpc.WaitForReady(true)); err != nil {
			t.Fatalf("Export %d: %v", i, err)
		}
	}

	all := s.GetAll()
	if len(all) != 3 {
		t.Fatalf("expected 3 traces, got %d", len(all))
	}
}

func TestEndToEnd_ErrorStatus(t *testing.T) {
	s := store.New(100, discardLog())
	addr, _ := startReceiver(t, s)

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	now := time.Now()
	tid := make([]byte, 16)
	sid := make([]byte, 8)
	tid[0] = 0xFF
	sid[0] = 0xFF

	req := &collectortracepb.ExportTraceServiceRequest{
		ResourceSpans: []*tracepb.ResourceSpans{{
			ScopeSpans: []*tracepb.ScopeSpans{{
				Spans: []*tracepb.Span{{
					TraceId:           tid,
					SpanId:            sid,
					Name:              "failing-op",
					StartTimeUnixNano: uint64(now.UnixNano()),
					EndTimeUnixNano:   uint64(now.Add(time.Millisecond).UnixNano()),
					Status: &tracepb.Status{
						Code:    tracepb.Status_STATUS_CODE_ERROR,
						Message: "connection refused",
					},
				}},
			}},
		}},
	}

	ctx := context.Background()
	if _, err := collectortracepb.NewTraceServiceClient(conn).Export(ctx, req, grpc.WaitForReady(true)); err != nil {
		t.Fatalf("Export: %v", err)
	}

	var expectedID model.TraceID
	copy(expectedID[:], tid)

	tr, ok := s.Get(expectedID)
	if !ok {
		t.Fatal("trace not found")
	}
	if tr.Spans[0].Status.Code != model.StatusError {
		t.Errorf("status: got %v, want error", tr.Spans[0].Status.Code)
	}
	if tr.Spans[0].Status.Message != "connection refused" {
		t.Errorf("status message: got %q", tr.Spans[0].Status.Message)
	}
}
