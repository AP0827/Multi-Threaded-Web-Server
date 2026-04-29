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
- Mixed malformed traffic can also surface 400 Bad Request.
- Queue pressure can surface 503 Service Unavailable.

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
