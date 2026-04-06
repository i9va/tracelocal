# tracelocal

[![CI](https://github.com/i9va/tracelocal/actions/workflows/ci.yml/badge.svg)](https://github.com/i9va/tracelocal/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Zero-config local distributed tracing for backend developers.

Run `tracelocal`, point your OTLP SDK at `localhost:4317`, and watch your traces appear in the terminal — no Jaeger, no Tempo, no cloud account needed.

```
> svc-api › POST /checkout      45ms  (4 spans)
  svc-auth › ValidateToken      12ms  (1 span)
  svc-cart › GetCart             8ms  (2 spans)
```

```
svc-api › POST /checkout  45ms  (4 spans)

svc-api › POST /checkout    [████████████████████████]  45ms
  svc-auth › ValidateToken  [████████                ]  12ms
  svc-cart › GetCart        [        ██████          ]   8ms
    svc-cart › db.query     [        ████            ]   6ms

esc back  q quit
```

## Features

- **OTLP/gRPC receiver** on port 4317 — drop-in for any OTel SDK
- **OTLP/HTTP receiver** on port 4318 — supports `application/x-protobuf` and `application/json`
- **Live terminal UI** — span tree with proportional timing bars, auto-refreshes every 200 ms
- **In-memory ring buffer** — keeps the last N traces, evicts oldest automatically
- **Single static binary** — no runtime dependencies, no config files

## Install

### Homebrew (macOS/Linux)

```sh
# Coming soon
```

### go install

```sh
go install github.com/henriqueholanda/tracelocal/cmd/tracelocal@latest
```

### Binary releases

Download a pre-built binary from the [Releases](https://github.com/henriqueholanda/tracelocal/releases) page.

## Quick start

**1. Start tracelocal**

```sh
tracelocal
# receiver: listening on :4317
```

**2. Configure your SDK**

<details>
<summary>Go</summary>

```go
import (
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
)

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
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace.export import BatchSpanProcessor

exporter = OTLPSpanExporter(endpoint="http://localhost:4317", insecure=True)
provider.add_span_processor(BatchSpanProcessor(exporter))
```

</details>

<details>
<summary>Node.js</summary>

```js
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-grpc';

const exporter = new OTLPTraceExporter({ url: 'grpc://localhost:4317' });
```

</details>

<details>
<summary>Environment variable (any SDK)</summary>

```sh
# gRPC (port 4317)
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc

# HTTP (port 4318)
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
```

</details>

**3. Navigate the UI**

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `enter` | Open trace detail (span tree) |
| `esc` | Back to list |
| `q` / `ctrl+c` | Quit |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `4317` | OTLP gRPC listen port |
| `--http-port` | `4318` | OTLP HTTP listen port (set to `0` to disable) |
| `--capacity` | `1000` | Maximum number of traces kept in memory |

## Contributing

```sh
make test          # run all tests
go test -race ./...  # with race detector
make build         # build the binary
```

PRs and issues are welcome. Please open an issue before starting large changes.

## License

[MIT](LICENSE)
