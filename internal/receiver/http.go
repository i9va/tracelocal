package receiver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	collectortracepb "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/henriqueholanda/tracelocal/internal/store"
)

// HTTPReceiver listens for OTLP/HTTP traces on a configurable address.
// It handles POST /v1/traces with either application/x-protobuf or
// application/json bodies, matching the OTLP/HTTP spec.
type HTTPReceiver struct {
	addr  string
	store *store.Store
	log   *slog.Logger
}

// NewHTTP creates an HTTPReceiver bound to addr.
func NewHTTP(addr string, s *store.Store, log *slog.Logger) *HTTPReceiver {
	return &HTTPReceiver{addr: addr, store: s, log: log}
}

// Start binds to r.addr and calls Serve. It blocks until ctx is canceled.
func (r *HTTPReceiver) Start(ctx context.Context) error {
	lis, err := net.Listen("tcp", r.addr)
	if err != nil {
		return fmt.Errorf("http receiver: listen %s: %w", r.addr, err)
	}
	return r.Serve(ctx, lis)
}

// Serve registers the OTLP/HTTP handler on lis and blocks until ctx is
// canceled. Callers can pass a pre-created listener (useful in tests).
func (r *HTTPReceiver) Serve(ctx context.Context, lis net.Listener) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/traces", r.handleTraces)

	srv := &http.Server{Handler: mux}
	r.log.Info("OTLP HTTP receiver listening", "addr", lis.Addr())

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(lis) }()

	select {
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), gracefulStopTimeout)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			srv.Close() //nolint:errcheck
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http receiver: serve: %w", err)
	}
}

func (r *HTTPReceiver) handleTraces(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var pbReq collectortracepb.ExportTraceServiceRequest
	ct := req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") {
		if err := protojson.Unmarshal(body, &pbReq); err != nil {
			http.Error(w, "unmarshal json: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		if err := proto.Unmarshal(body, &pbReq); err != nil {
			http.Error(w, "unmarshal proto: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	svc := &traceService{store: r.store, log: r.log}
	resp, err := svc.Export(req.Context(), &pbReq)
	if err != nil {
		http.Error(w, "export: "+err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := proto.Marshal(resp)
	if err != nil {
		http.Error(w, "marshal response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusOK)
	w.Write(out) //nolint:errcheck
}
