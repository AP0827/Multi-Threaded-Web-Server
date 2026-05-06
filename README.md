# MTWS - Multi Threaded Web Server (Go)

MTWS is a research-oriented HTTP/1.1 server written in Go on top of raw TCP
sockets. Its core thesis is that WAF inspection should happen inside the HTTP
parser itself, so the security engine and the application server cannot parse
ambiguous requests differently.

## Features

- **Custom HTTP/1.1 Parser**: Request parsing with support for chunked transfer encoding and trailers
- **Worker Pool Architecture**: Bounded concurrency with a configurable pool size and job queue
- **Token Bucket Rate Limiting**: Per-IP request throttling with configurable capacity and refill rate
- **Request Validation**: URI normalization, header validation, and content inspection
- **WAF Integration**: Signature-based policy enforcement with pluggable rule files
- **ModSecurity Comparison**: Side-by-side comparison with nginx + ModSecurity CRS
- **Load Testing**: Benchmarking scripts for burst and sustained traffic patterns

## Project Structure

| Directory | Purpose |
|-----------|---------|
| `cmd/server/` | HTTP server entrypoint and connection dispatch |
| `core/` | Connection lifecycle and response handling |
| `http/` | HTTP/1.1 parser with malformed request detection |
| `pool/` | Worker pool for bounded concurrency |
| `security/` | Rate limiting and WAF policy enforcement |
| `scripts/` | Load testing and utility scripts |
| `docs/` | Design documentation and compliance specs |
| `docker/` | ModSecurity CRS customization for proxy path |

## Requirements

- **Go** 1.26 or later
- **Docker** and **Docker Compose** (for multi-stack deployment)

## Quick Start

### Build and Run

```bash
go run ./cmd/server
```

The server listens on `:8080` by default.

Useful local checks:

```powershell
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
curl http://127.0.0.1:8080/metrics
curl http://127.0.0.1:8080/static/
```

## Docker Comparison Stack

Sprint 3 introduces a two-path comparison environment:
- `mtws` on `http://localhost:8080`
- `nginx + ModSecurity CRS` proxying to a separate standard-library backend on `http://localhost:8081`

Start both services:
```bash
docker compose up --build
```

**Service Endpoints:**
- MTWS (`http://localhost:8080`) – Direct connection
- nginx + ModSecurity proxy (`http://localhost:8081`) – Via proxy forward to standard-library backend

This dual-path setup enables security research into HTTP/1.1 parsing discrepancies between implementations.

### Benchmark Mode

To avoid rate-limiting contamination in latency measurements, disable the token bucket:

```bash
docker compose -f docker-compose.yml -f docker-compose.benchmark.yml up --build
```

**Configuration:**
- `MTWS_RATE_LIMIT_DISABLED=true` – Disable token bucket globally
- `MTWS_BENCHMARK_MODE=true` – Enable benchmark-specific tuning
- `MTWS_RATE_LIMIT_RATE` – Token-bucket refresh rate (per-IP tokens/sec)
- `MTWS_RATE_LIMIT_CAPACITY` – Token-bucket capacity (max burst)
- `MTWS_WAF_POLICY_FILE` – Path to WAF signature policy file

## Testing and Research

### Lab Tool – Payload Replay and Benchmarking

The lab tool supports compliance testing and security research:

**Compare parsing behavior across both paths:**

```bash
go run ./cmd/lab compare
go run ./cmd/lab compare -json-out experiments/results/compare.json
```

**Benchmark direct MTWS:**

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 200 -concurrency 10
```

**Benchmark proxy path:**

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8081/health -requests 200 -concurrency 10
```

Starter discrepancy and attack payloads live in `experiments/payloads/`.
Structured results can be written into `experiments/results/`.
The detailed experiment workflow is documented in `docs/sprint4-experiments.md`,
and the final write-up template is in `docs/final-report-template.md`.
MTWS now enforces required `Host` semantics, rejects unsupported
transfer codings, supports strict `Transfer-Encoding: chunked`, and scans URI,
headers, body content, and trailers inside the parser.
The lab tool normalizes `.http` fixtures to canonical CRLF line endings before
replay; use `.raw` files when you want byte-exact malformed payload delivery.
Runtime controls:
- `MTWS_ADDR` changes the listen address
- `MTWS_WORKERS` changes the worker goroutine count
- `MTWS_JOB_QUEUE_SIZE` changes the bounded queue size
- `MTWS_READ_TIMEOUT` sets the request read deadline
- `MTWS_WRITE_TIMEOUT` sets the response write deadline
- `MTWS_IDLE_TIMEOUT` sets the HTTP/1.1 keep-alive idle deadline
- `MTWS_MAX_KEEPALIVE_REQUESTS` caps requests served on one TCP connection
- `MTWS_QUEUE_TIMEOUT` sets how long accept waits for worker queue capacity
- `MTWS_STATIC_DIR` sets the fixed directory served under `/static/`
- `MTWS_SHUTDOWN_TIMEOUT` sets graceful worker-drain timeout
- `MTWS_RATE_LIMIT_DISABLED=true` disables the token bucket
- `MTWS_BENCHMARK_MODE=true` also disables the token bucket for benchmark runs
- `MTWS_RATE_LIMIT_RATE` and `MTWS_RATE_LIMIT_CAPACITY` override token-bucket settings
- `MTWS_WAF_POLICY_FILE` points MTWS at a line-based signature policy file
- `MTWS_TLS_CERT_FILE` and `MTWS_TLS_KEY_FILE` enable TLS when both are set
The exact HTTP subset is documented in `docs/http-compliance.md`.
The operational hardening profile is documented in `docs/production-demo.md`.
Copy `.env.example` when you want a visible deployment configuration template.

Generate a local self-signed TLS certificate for demos:
```powershell
go run ./cmd/certgen -force
```

Run a sustained keep-alive soak benchmark:
```powershell
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -duration 2m -concurrency 10 -keepalive -json-out experiments/results/soak-mtws.json
```

## Scripts

Make scripts executable once:
```bash
chmod +x scripts/run_server.sh scripts/load/*.sh
```

Run server via script:
```bash
./scripts/run_server.sh
```

### Load Testing Scripts

Make scripts executable:

```bash
chmod +x scripts/*.sh scripts/load/*.sh
```

**Burst Test** – High concurrent spike (observe rate-limit response):

```bash
./scripts/load/burst_test.sh                   # Defaults: 120 requests, 30 concurrency
./scripts/load/burst_test.sh URL TOTAL CONC   # Custom: URL, total requests, concurrency
```

**Sustained Test** – Steady traffic over time:

Mixed status test (valid + malformed + burst traffic together):
```bash
./scripts/load/mixed_test.sh
```

Custom mixed test:
```bash
./scripts/load/mixed_test.sh http://localhost:8080/ 90 15 60 30 262144
```

Arguments:
- URL (default: http://localhost:8080/)
- valid requests (default: 90)
- malformed requests (default: 15)
- burst requests (default: 60)
- concurrency (default: 30)
- burst body size in bytes (default: 262144)

Sustained test (steady traffic over time):
```bash
./scripts/load/sustained_test.sh                    # Defaults: 10 sec, 20 req/sec
./scripts/load/sustained_test.sh URL DURATION RPS  # Custom: URL, duration (sec), requests/sec
```

**Expected Results with Rate Limiting Enabled:**
- `200 OK` – Within rate limit
- `429 Too Many Requests` – Rate limit exceeded
- `503 Service Unavailable` – Worker queue full
- `400 Bad Request` – Malformed request

## HTTP/1.1 Compliance

MTWS enforces:
- Required `Host` header semantics
- Strict `Transfer-Encoding: chunked` handling
- URI normalization and traversal detection
- Header validation and trailer support
- Body content scanning and signature matching

For full compliance spec, see `docs/http-compliance.md`.

## Design Documentation

- **Non-Functional Requirements:** `docs/NonFunctionalRequirements.md`
- **HTTP Compliance:** `docs/http-compliance.md`
- **Experiment Workflow:** `docs/sprint4-experiments.md`
- **Final Report Template:** `docs/final-report-template.md`

## Development

```bash
go test ./...           # Run all tests
go build ./...          # Build all packages
go fmt ./...            # Format code
go vet ./...            # Run linter
```

## Utility Scripts

### Memory Usage

Measure peak memory usage while running the server or any other command:

```bash
./scripts/memory_usage.sh
./scripts/memory_usage.sh -- go run ./cmd/server
./scripts/memory_usage.sh -- bash scripts/load/burst_test.sh
```

By default, the script runs `go test ./...` and reports the peak resident set size (RSS) in KiB and MiB.
