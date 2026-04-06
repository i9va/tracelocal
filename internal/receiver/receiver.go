package receiver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/grpc"

	"github.com/henriqueholanda/tracelocal/internal/store"
)

const gracefulStopTimeout = 5 * time.Second

// Receiver listens for OTLP/gRPC traces and writes them to the store.
type Receiver struct {
	addr   string
	store  *store.Store
	server *grpc.Server
	log    *slog.Logger
}

// New creates a Receiver bound to addr.
func New(addr string, s *store.Store, log *slog.Logger) *Receiver {
	return &Receiver{addr: addr, store: s, log: log}
}

// Start binds to r.addr and calls Serve. It blocks until ctx is cancelled.
func (r *Receiver) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("receiver: listen %s: %w", r.addr, err)
	}
	return r.Serve(ctx, lis)
}

// Serve registers the OTLP handler on lis and blocks until ctx is cancelled.
// The gRPC server takes ownership of lis and closes it on shutdown.
// Callers can pass a pre-created listener (e.g. from net.Listen(":0")) to
// obtain the actual bound address before starting — useful in tests.
func (r *Receiver) Serve(ctx context.Context, lis net.Listener) error {
	r.server = grpc.NewServer()
	collectortracepb.RegisterTraceServiceServer(r.server, &traceService{store: r.store, log: r.log})

	r.log.Info("OTLP gRPC receiver listening", "addr", lis.Addr())

	errCh := make(chan error, 1)
	go func() { errCh <- r.server.Serve(lis) }()

	select {
	case <-ctx.Done():
		done := make(chan struct{})
		go func() {
			r.server.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(gracefulStopTimeout):
			r.server.Stop() // force-close after timeout
		}
		return nil
	case err := <-errCh:
		return fmt.Errorf("receiver: serve: %w", err)
	}
}
