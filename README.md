# MTWS - Multi Threaded Web Server (Go)

MTWS is a learning-focused Go project for building a multithreaded HTTP server step by step.

## Goals
- Learn TCP server fundamentals in Go.
- Implement bounded concurrency with a worker pool.
- Build a custom HTTP request parser.
- Add security and reliability features incrementally.

## Current Status
- TCP listener and connection accept loop are in place.
- Worker pool skeleton is implemented.
- Basic connection handling exists.
- HTTP parser package and parser tests are being developed.

## Project Structure
```text
cmd/server/      Application entrypoint
core/            Connection lifecycle logic
docs/            Design notes and non-functional requirements
http/            HTTP request parsing and tests
pool/            Worker pool implementation
scripts/         Run helpers and load test scripts
security/        Security modules (rate limiter)
utils/           Utility helpers (future)
```

## Requirements
- Go 1.26+

## Run
```bash
go run ./cmd/server
```

The server listens on port 8080 by default.

## Docker Comparison Stack

Sprint 3 introduces a two-path comparison environment:
- `mtws` on `http://localhost:8080`
- `nginx + ModSecurity CRS` proxying to a separate standard-library backend on `http://localhost:8081`

Start both services:
```bash
docker compose up --build
```

Test the direct MTWS path:
```bash
curl http://localhost:8080/health
```

Test the split-proxy path:
```bash
curl http://localhost:8081/health
```

The ModSecurity container uses the official `owasp/modsecurity-crs` nginx image
and forwards traffic to an internal comparison backend built on Go's standard
`net/http` parser. This separation matters for the research thesis: the proxy
path must terminate at a different HTTP parser to expose split-proxy parsing
discrepancies honestly. Custom CRS tuning files live in `docker/modsecurity/`
so bypass and false-positive experiments have a stable place to be recorded.

## Sprint 4 Lab Tool

Replay raw payloads against both stacks:
```bash
go run ./cmd/lab compare
```

Replay raw payloads and save structured evidence:
```bash
go run ./cmd/lab compare -json-out experiments/results/compare.json
```

Benchmark MTWS directly:
```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 200 -concurrency 10
```

Benchmark the split-proxy path:
```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8081/health -requests 200 -concurrency 10
```

Starter discrepancy and attack payloads live in `experiments/payloads/`.
Structured results can be written into `experiments/results/`.
The detailed experiment workflow is documented in `docs/sprint4-experiments.md`,
and the final write-up template is in `docs/final-report-template.md`.
MTWS now enforces required `Host` semantics, rejects unsupported
`Transfer-Encoding`, and scans URI, headers, and body content inside the parser.

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

Burst test (high concurrent spike):
```bash
./scripts/load/burst_test.sh
```

Custom burst test:
```bash
./scripts/load/burst_test.sh http://localhost:8080/ 200 50
```

Arguments:
- URL (default: http://localhost:8080/)
- total requests (default: 120)
- concurrency (default: 30)

Sustained test (steady traffic over time):
```bash
./scripts/load/sustained_test.sh
```

Custom sustained test:
```bash
./scripts/load/sustained_test.sh http://localhost:8080/ 15 25
```

Arguments:
- URL (default: http://localhost:8080/)
- duration in seconds (default: 10)
- requests per second (default: 20)

Expected outcome with rate limiter enabled:
- Some requests return 200 OK.
- Under heavy load, some requests return 429 Too Many Requests.

## Test
```bash
go test ./...
```

## Learning Roadmap (High Level)
1. Bounded queue and worker behavior under load.
2. HTTP parsing (request line, headers, body).
3. Validation and error responses.
4. Routing and response builder.
5. Rate limiting and basic WAF checks.
6. Logging and observability improvements.

## Notes
- This is a conceptual learning project, so implementation is intentionally iterative.
- Design constraints are documented in docs/NonFunctionalRequirements.md.
