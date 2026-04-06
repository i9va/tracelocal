package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/henriqueholanda/tracelocal/internal/receiver"
	"github.com/henriqueholanda/tracelocal/internal/store"
	"github.com/henriqueholanda/tracelocal/internal/tui"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	port := flag.Int("port", 4317, "OTLP gRPC listen port")
	httpPort := flag.Int("http-port", 4318, "OTLP HTTP listen port (0 to disable)")
	cap := flag.Int("capacity", 1000, "max number of traces to keep in memory")
	logFile := flag.String("log-file", "", "write internal logs to `file` (default: discard)")
	printVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *printVersion {
		fmt.Println(version)
		return
	}

	grpcAddr := fmt.Sprintf(":%d", *port)
	httpAddr := ""
	if *httpPort != 0 {
		httpAddr = fmt.Sprintf(":%d", *httpPort)
	}

	log := newLogger(*logFile)

	s := store.New(*cap, log)
	recv := receiver.New(grpcAddr, s, log)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	p := tea.NewProgram(tui.New(s, grpcAddr, httpAddr), tuiOptions()...)

	// gRPC receiver — fatal: quit TUI if it fails to start.
	go func() {
		if err := recv.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "grpc receiver error: %v\n", err)
			cancel()
			p.Send(tui.ReceiverErrMsg{Err: err})
		}
	}()

	// HTTP receiver — non-fatal: log the error but keep running.
	if httpAddr != "" {
		httpRecv := receiver.NewHTTP(httpAddr, s, log)
		go func() {
			if err := httpRecv.Start(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "http receiver error: %v\n", err)
			}
		}()
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

// newLogger returns a logger that writes to the given file path, or discards
// output when path is empty (the default for interactive TUI use).
func newLogger(path string) *slog.Logger {
	var w io.Writer = io.Discard
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open log file: %v\n", err)
			os.Exit(1)
		}
		w = f
	}
	return slog.New(slog.NewTextHandler(w, nil))
}
