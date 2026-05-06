# MTWS – Multi-Threaded Web Server

A Go-based HTTP server project focused on HTTP/1.1 parsing, bounded concurrency, rate limiting, and security-oriented request handling.

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

The server listens on `localhost:8080` by default. Test with:

```bash
curl http://localhost:8080/health
```

### Run Tests

```bash
go test ./...
```

## Multi-Stack Deployment

Compare MTWS against nginx + ModSecurity CRS by launching both services:

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

**Payload Library:**
- Raw attack payloads: `experiments/payloads/`
- Structured results: `experiments/results/`
- Experiment workflow: `docs/sprint4-experiments.md`
- Final report template: `docs/final-report-template.md`

**Fixture Format:**
- `.http` files are normalized to CRLF before replay (safe for malformed payloads)
- `.raw` files are replayed byte-exact (use for strict malformat testing)

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
