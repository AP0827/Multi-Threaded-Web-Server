# MTWS - Multi Threaded Web Server (Go)

MTWS is a research-oriented HTTP/1.1 server written in Go on top of raw TCP
sockets. Its core thesis is that WAF inspection should happen inside the HTTP
parser itself, so the security engine and the application server cannot parse
ambiguous requests differently.

## Goals
- Demonstrate parser-integrated WAF inspection.
- Compare MTWS against a traditional Nginx + ModSecurity split-proxy stack.
- Enforce strict HTTP normalization and bounded concurrency.
- Provide production-demonstration controls: config, graceful shutdown,
  readiness, metrics, deadlines, TLS option, and hardened containers.

## Current Status
- Raw TCP listener, worker pool, custom parser, router, and response writer are implemented.
- In-parser WAF scanning covers URI, headers, body, chunked bodies, and trailers.
- Strict parser behavior rejects ambiguous HTTP syntax early.
- Docker comparison stack is available for MTWS vs Nginx + ModSecurity.
- Production-demonstration hardening is documented in `docs/production-demo.md`.

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
```powershell
go run ./cmd/server
```

The server listens on `:8080` by default.

Useful local checks:

```powershell
curl http://127.0.0.1:8080/health
curl http://127.0.0.1:8080/ready
curl http://127.0.0.1:8080/metrics
```

## Docker Comparison Stack

Sprint 3 introduces a two-path comparison environment:
- `mtws` on `http://localhost:8080`
- `nginx + ModSecurity CRS` proxying to a separate standard-library backend on `http://localhost:8081`

Start both services:
```bash
docker compose up --build
```

Start the same stack in benchmark mode, with MTWS rate limiting disabled so
latency measurements are not contaminated by `429` responses:
```bash
docker compose -f docker-compose.yml -f docker-compose.benchmark.yml up --build
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
