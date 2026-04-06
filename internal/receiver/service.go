package receiver

import (
	"context"
	"log/slog"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/henriqueholanda/tracelocal/internal/store"
	"github.com/henriqueholanda/tracelocal/internal/model"
)

// traceService implements the OTLP TraceServiceServer.
type traceService struct {
	collectortracepb.UnimplementedTraceServiceServer
	store *store.Store
	log   *slog.Logger
}

// Export receives a batch of resource spans, stores each span, and logs a
// structured summary. It always returns a successful response so that the SDK
// does not retry; per-span errors are logged and skipped.
func (ts *traceService) Export(
	ctx context.Context,
	req *collectortracepb.ExportTraceServiceRequest,
) (*collectortracepb.ExportTraceServiceResponse, error) {
	for _, rs := range req.GetResourceSpans() {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		res := rs.GetResource()
		for _, ss := range rs.GetScopeSpans() {
			scope := ss.GetScope()
			for _, pbSpan := range ss.GetSpans() {
				span := spanFromProto(pbSpan, res, scope)

				if err := ts.store.Add(span); err != nil {
					ts.log.ErrorContext(ctx, "failed to store span", "err", err, "trace", span.TraceID.String())
					continue
				}

				ts.logSpan(ctx, span)
			}
		}
	}
	return &collectortracepb.ExportTraceServiceResponse{}, nil
}

// logSpan emits a structured log line for every accepted span.
func (ts *traceService) logSpan(ctx context.Context, s model.Span) {
	args := []any{
		"trace", s.TraceID.String(),
		"span", s.SpanID.String(),
		"service", s.ServiceName(),
		"name", s.Name,
		"kind", s.Kind.String(),
		"dur", s.Duration(),
		"root", s.IsRoot(),
	}
	if s.Status.Code == model.StatusError {
		args = append(args, "error", s.Status.Message)
	}
	ts.log.InfoContext(ctx, "span received", args...)
}
