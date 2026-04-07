<div align="center">
  <img src="docs/logo.svg" alt="tracelocal" height="52" />
  <br/><br/>
  <p><strong>Local distributed tracing — straight to your terminal.</strong></p>
  <p>No Jaeger. No Tempo. No cloud account. Just run one binary.</p>
  <br/>

  [![CI](https://github.com/i9va/tracelocal/actions/workflows/ci.yml/badge.svg)](https://github.com/i9va/tracelocal/actions/workflows/ci.yml)
  [![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
  [![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](go.mod)
</div>

---

```
svc-api  › POST /checkout    [████████████████████████]  45ms
svc-auth › ValidateToken     [████████                ]  12ms
svc-cart › GetCart           [        ██████          ]   8ms
svc-db   › db.query          [        ████            ]   6ms
```

Point your OTel SDK at `localhost:4317`, run `tracelocal`, and your spans appear live in a navigable terminal UI — with a span tree, timing bars, statistics, and error filtering.

## Why tracelocal?

Setting up Jaeger or Tempo locally takes 10–20 minutes. `tracelocal` takes 10 seconds:

```sh
go install github.com/i9va/tracelocal/cmd/tracelocal@latest
tracelocal
# OTLP/gRPC :4317   OTLP/HTTP :4318
```

That's it. Any OTel SDK speaks OTLP — nothing else to configure.

## Features

- **OTLP/gRPC** on `:4317` and **OTLP/HTTP** on `:4318` — drop-in for any OTel SDK
- **Live span tree** with proportional timing bars, auto-refreshes every 200 ms
- **Error filter** — press `e` to instantly surface only failing traces
- **Statistics view** — p50/p95/p99 latency and error rates per service, press `s`
- **Search** — filter by service or operation name with `/`
- **Ring buffer** — keeps the last N traces, evicts oldest automatically
- **Single static binary** — no runtime deps, no config files, no Docker

## Install

**go install** (requires Go 1.24+)
```sh
go install github.com/i9va/tracelocal/cmd/tracelocal@latest
```

**Binary release** — download for macOS, Linux, or Windows from the [Releases](https://github.com/i9va/tracelocal/releases) page:
```sh
curl -sSL https://github.com/i9va/tracelocal/releases/latest/download/tracelocal_$(uname -s)_$(uname -m).tar.gz \
  | tar -xz && mv tracelocal /usr/local/bin/
```

## Quick start

**1 — Start tracelocal**
```sh
tracelocal
```

**2 — Configure your SDK** (or just set env vars — works with any language)
```sh
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

<details>
<summary>Go</summary>

```go
exporter, err := otlptracegrpc.New(ctx,
    otlptracegrpc.WithEndpoint("localhost:4317"),
    otlptracegrpc.WithInsecure(),
)
tp := trace.NewTracerProvider(trace.WithBatcher(exporter))
```
</details>

<details>
<summary>Python</summary>

```python
exporter = OTLPSpanExporter(endpoint="http://localhost:4317", insecure=True)
provider.add_span_processor(BatchSpanProcessor(exporter))
```
</details>

<details>
<summary>Node.js</summary>

```js
const exporter = new OTLPTraceExporter({ url: 'grpc://localhost:4317' });
```
</details>

**3 — Navigate the UI**

| Key | Action |
|-----|--------|
| `↑` / `k` · `↓` / `j` | Move up / down |
| `enter` | Open trace — span tree + timing bars |
| `s` | Statistics view (p50 / p95 / p99 per service) |
| `e` | Toggle errors-only filter |
| `/` | Search by service or operation |
| `esc` | Back |
| `q` / `ctrl+c` | Quit |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `4317` | OTLP gRPC listen port |
| `--http-port` | `4318` | OTLP HTTP listen port (`0` to disable) |
| `--capacity` | `1000` | Max traces kept in memory |
| `--log` | _(stderr)_ | Write structured logs to a file |

## Contributing

```sh
make build           # build the binary
make test            # run all tests
go test -race ./...  # with race detector
golangci-lint run ./...  # lint
```

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request, and [SECURITY.md](SECURITY.md) before reporting a vulnerability. By participating in this project you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[Apache License 2.0](LICENSE)
