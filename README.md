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
utils/           Utility helpers (future)
```

## Requirements
- Go 1.26+

## Run
```bash
go run ./cmd/server
```

The server listens on port 8080 by default.

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
