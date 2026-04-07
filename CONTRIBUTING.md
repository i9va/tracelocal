# Contributing to tracelocal

Thanks for your interest in contributing. This document covers how to get set up, what to work on, and how to get your changes merged.

## Before you start

For anything beyond a small bug fix or typo, **open an issue first**. Describing what you want to change and why saves both of us time — it avoids duplicate work and ensures the direction fits the project before you invest effort writing code.

## Setup

```sh
git clone https://github.com/i9va/tracelocal.git
cd tracelocal
go mod download
make build   # verify everything compiles
make test    # verify tests pass
```

Requirements: Go 1.22 or later.

## Making changes

**Branch from `main`:**
```sh
git checkout -b your-branch-name
```

**Run the test suite before pushing:**
```sh
go test -race ./...
go vet ./...
golangci-lint run ./...
```

**Keep commits focused.** One logical change per commit makes review easier and keeps history readable. Write commit messages in the imperative: `fix: handle zero TraceID in store` not `fixed a bug`.

## Pull requests

- Reference the related issue in the PR description (e.g. `Closes #42`)
- Keep PRs small — a focused 200-line PR gets reviewed faster than a sprawling 1000-line one
- All CI checks must pass before merge
- At least one approval is required

## Architecture

The project follows a strict no-global-state policy. Dependencies are injected via constructors:

```
cmd/tracelocal/   entry point — wires store, receiver, TUI
internal/receiver/  OTLP gRPC + HTTP servers
internal/store/     thread-safe in-memory ring buffer
internal/tui/       bubbletea terminal UI
pkg/model/          OTel data types
```

If you're adding a new package, follow the same pattern: constructor takes its dependencies, no `init()` side effects, no package-level vars that mutate at runtime.

## What to work on

Check the [open issues](https://github.com/i9va/tracelocal/issues) for ideas. Issues labelled `good first issue` are a good place to start if you're new to the codebase.

## Code style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Wrap errors with context: `fmt.Errorf("store: add: %w", err)`
- Never write to stdout directly — the TUI owns it; use the injected `*slog.Logger`
- Prefer table-driven tests

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
