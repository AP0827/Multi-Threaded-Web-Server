# Software Requirement Specification (SRS)

**Project:** MTWS – Multithreaded Secure HTTP Server  
**Type:** Educational Network Security Research Server  
**Version:** 1.0  
**Date:** May 2026

---

## 1. Overview

MTWS is a standalone HTTP/1.1 server implemented in Go, designed for:
- **HTTP/1.1 parsing research:** Strict compliance, unambiguous interpretation
- **WAF (Web Application Firewall) research:** Signature-based threat detection
- **Security comparison:** Side-by-side testing vs. nginx + ModSecurity
- **Performance benchmarking:** Throughput, latency, and scalability analysis

The software is **self-contained** and **does not require** a traditional web application framework.

---

## 2. Runtime Requirements

### 2.1 Go Runtime

| Requirement | Specification | Notes |
|-------------|---------------|-------|
| **Language** | Go | Version 1.26.1 or later |
| **Compilation** | Native binary | No JIT; statically linked by default |
| **Runtime scheduler** | Goroutine-based | Efficient M:N multiplexing |
| **Memory model** | Garbage collected | No manual memory management |
| **Concurrency model** | CSP (channels) | For job queue management |

**Why Go:**
- Built-in network I/O (`net` package) with goroutine support
- Efficient HTTP parsing libraries (`bufio`, `net/http` patterns)
- Single static binary for portability
- Suitable for teaching OS concepts (concurrency, scheduling)

### 2.2 Standard Library Dependencies

MTWS uses **only** Go standard library modules:

| Module | Version | Usage |
|--------|---------|-------|
| `net` | stdlib | TCP listener, connection handling |
| `bufio` | stdlib | Buffered I/O for request parsing |
| `strings` | stdlib | String manipulation, parsing |
| `time` | stdlib | Read deadlines, rate limiter timing |
| `sync` | stdlib | Mutex for rate limiter thread safety |
| `log` | stdlib | Simple logging |
| `os` | stdlib | Environment variable access |
| `strconv` | stdlib | Configuration parsing |

**No external dependencies:** Reduces supply chain risk, simplifies deployment.

### 2.3 Operating System Interfaces

| OS Interface | Usage |
|--------------|-------|
| **POSIX sockets** | TCP/IP listening and connection handling |
| **epoll (Linux) / kqueue (macOS) / IOCP (Windows)** | Async I/O (managed by Go runtime) |
| **File descriptors** | One per active connection |
| **Signal handling** | SIGTERM for graceful shutdown (future enhancement) |
| **Timers** | For read deadlines (anti-Slowloris) |

---

## 3. Build & Deployment

### 3.1 Build Requirements

```bash
# Compile to native binary
go build -o mtws ./cmd/server

# Compile for specific OS/arch
GOOS=linux GOARCH=amd64 go build -o mtws ./cmd/server
GOOS=darwin GOARCH=arm64 go build -o mtws ./cmd/server
```

### 3.2 Supported Build Platforms

| Platform | Architecture | Status |
|----------|--------------|--------|
| Linux | x86-64 (amd64) | Primary (tested) |
| Linux | ARM64 (aarch64) | Supported |
| macOS | x86-64 (Intel) | Supported |
| macOS | ARM64 (Apple Silicon) | Supported |
| Windows | x86-64 (amd64) | Supported (via native Go or WSL2) |
| Windows | ARM64 | Supported (Go 1.16+) |

### 3.3 Cross-Compilation

No special build tools required; use Go's built-in cross-compilation:

```bash
# Build for Linux on macOS
GOOS=linux GOARCH=amd64 go build -o mtws ./cmd/server

# Build for Windows on Linux
GOOS=windows GOARCH=amd64 go build -o mtws.exe ./cmd/server
```

---

## 4. Runtime Environment

### 4.1 Minimum Requirements

| Requirement | Specification |
|-------------|---------------|
| **OS** | Linux kernel 4.4+, macOS 10.15+, Windows 10+ |
| **CPU** | x86-64 or ARM64 architecture |
| **RAM** | 256 MB available |
| **Network** | TCP/IP stack (loopback or Ethernet) |
| **Permissions** | Bind to port ≥1024 (no root required) or root for port <1024 |

### 4.2 Environment Variables

All configuration via environment variables (12-factor app):

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `MTWS_LISTEN_ADDR` | string | `:8080` | Listen address (format: `:port` or `host:port`) |
| `MTWS_WORKER_POOL_SIZE` | int | `10` | Number of concurrent request handlers |
| `MTWS_JOB_QUEUE_SIZE` | int | `200` | Buffered job queue capacity |
| `MTWS_RATE_LIMIT_RATE` | float | `5.0` | Tokens per second per IP |
| `MTWS_RATE_LIMIT_CAPACITY` | float | `10.0` | Max burst tokens per IP |
| `MTWS_RATE_LIMIT_DISABLED` | bool | `false` | Disable rate limiter entirely |
| `MTWS_BENCHMARK_MODE` | bool | `false` | Enable benchmark tuning (disables rate limit) |
| `MTWS_WAF_POLICY_FILE` | string | `security/waf/default-policy.txt` | Path to WAF signature file |
| `MTWS_LOG_LEVEL` | string | `info` | Logging level (future: `debug`, `info`, `warn`, `error`) |

### 4.3 Configuration at Runtime

```bash
# Example: Run with custom pool size and rate limit disabled
MTWS_WORKER_POOL_SIZE=20 MTWS_RATE_LIMIT_DISABLED=true go run ./cmd/server

# Example: Benchmark mode (no rate limiting)
MTWS_BENCHMARK_MODE=true go run ./cmd/server

# Example: Listen on specific interface and port
MTWS_LISTEN_ADDR=0.0.0.0:9000 go run ./cmd/server
```

---

## 5. External Services & Dependencies

### 5.1 None Required (Minimal Footprint)

MTWS is **self-contained**:
- ✗ No database server
- ✗ No message broker (Redis, RabbitMQ, etc.)
- ✗ No caching layer (Memcached, Redis)
- ✗ No authentication service (OAuth, LDAP)
- ✗ No monitoring agent (Prometheus exporter, etc.)

**Design rationale:** Educational project; focus on core security and networking concepts.

### 5.2 Optional: Docker Compose Stack

For ModSecurity comparison, optional Docker stack provided:

```yaml
services:
  mtws:
    build: .
    ports: ["8080:8080"]
    environment:
      MTWS_WORKER_POOL_SIZE: 10
      
  nginx-modsec:
    image: nginx:latest  # with ModSecurity module
    ports: ["8081:80"]
    volumes:
      - ./docker/modsecurity:/etc/nginx/modsec
```

**Requirements for Docker deployment:**
- Docker Engine ≥20.10
- Docker Compose ≥1.29
- ~500 MB disk (Alpine base + ModSecurity CRS)

---

## 6. Testing Framework

### 6.1 Unit Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

**Test files:**
- `http/parser_test.go` – HTTP parser edge cases
- `http/parser_fuzz_test.go` – Fuzz testing for malformed requests
- `security/waf/automaton_test.go` – WAF pattern matching
- `security/ratelimiter/limiter_test.go` (if present) – Rate limiter correctness
- `config/config_test.go` – Configuration parsing

### 6.2 Load Testing

Built-in lab tool for benchmarking:

```bash
# Benchmark MTWS directly
go run ./cmd/lab benchmark \
  -url http://127.0.0.1:8080/health \
  -requests 1000 \
  -concurrency 50

# Compare behavior across two servers
go run ./cmd/lab compare -json-out results.json
```

### 6.3 Fuzz Testing

```bash
# Run fuzz tests (Go 1.18+)
go test -fuzz FuzzParseRequest ./http
```

---

## 7. Code Quality & Linting

### 7.1 Go Code Standards

| Tool | Usage | Status |
|------|-------|--------|
| `go fmt` | Code formatting | Built-in; must pass before commit |
| `go vet` | Static analysis | Built-in; checks common bugs |
| `golangci-lint` | Advanced linting | Optional; recommended for PRs |
| `go test` | Unit testing | Built-in; must pass all tests |

### 7.2 Recommended Linting

```bash
# Install golangci-lint (optional)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linting
golangci-lint run ./...

# Format code
go fmt ./...
```

---

## 8. Logging & Observability

### 8.1 Logging

- **Library:** Go standard `log` package
- **Format:** Simple text (no structured JSON, for educational clarity)
- **Output:** Stdout/stderr

**Example logs:**

```
Server running on port:8080!
Rate limiter enabled: rate=5.00 capacity=10.00
New request from 127.0.0.1:54321
Request method=GET path=/health version=HTTP/1.1
```

### 8.2 Security Event Logging

Logged on WAF/parsing violations:

```
[SECURITY] Client 192.168.1.100 - Rule 001 matched in field=URI pattern='<script>' rule_id='001' status_code=403
```

### 8.3 Metrics & Benchmarks

Output to JSON files (not real-time):

```json
{
  "name": "benchmark-mtws-clean.json",
  "total_requests": 1000,
  "successful_requests": 998,
  "failed_requests": 2,
  "avg_latency_ms": 12.5,
  "p95_latency_ms": 45.2,
  "p99_latency_ms": 102.3,
  "throughput_rps": 8234
}
```

---

## 9. Backward & Forward Compatibility

### 9.1 API Stability

- **Config:** Environment variable interface is stable (semver-like)
- **Command-line:** No CLI tools (configuration only via env vars)
- **HTTP response:** Compliant with HTTP/1.1 spec (stable across versions)

### 9.2 Breaking Changes Policy

Unlikely for educational project, but if needed:
- Major version bump for breaking changes
- Deprecation warnings in logs for ≥2 releases before removal

---

## 10. Security Properties

### 10.1 Input Validation

- **HTTP parser:** Rejects malformed/ambiguous requests (fail-closed)
- **WAF:** Scans all user-controlled fields (URI, headers, body, trailers)
- **Rate limiter:** Per-IP rate enforcement to prevent DoS

### 10.2 Known Limitations

- **No encryption (TLS):** Not implemented; focus on application security
- **No authentication:** Out of scope for demo
- **No output encoding:** Handlers must be secure
- **No CSRF protection:** Not applicable to stateless server

---

## 11. Documentation

### 11.1 Required Documentation

| Document | Location | Purpose |
|----------|----------|---------|
| README | `README.md` | Quick start, feature overview |
| HTTP Compliance | `docs/http-compliance.md` | Spec details, design rationale |
| Non-Functional Requirements | `docs/NonFunctionalRequirements.md` | Performance, reliability targets |
| Code comments | Source files | Inline explanations |

### 11.2 Example Handlers

- Health check: `/health` → `200 OK`
- Echo request: `/echo` → echo back request details (future)
- Static assets: `/static/*` → serve files (future)

---

## 12. Deployment Checklist

Before running in production (lab):

- [ ] Go 1.26+ installed: `go version`
- [ ] Source code checked out: `git clone ...` or `cd MTWS`
- [ ] All tests pass: `go test ./...`
- [ ] Binary built: `go build -o mtws ./cmd/server`
- [ ] Config verified: Check environment variables
- [ ] WAF policy loaded: Confirm `default-policy.txt` exists
- [ ] Port available: Verify port 8080 (or configured) is free
- [ ] Logs writable: Confirm stdout/stderr are open
- [ ] Network reachable: `curl http://localhost:8080/health`

---

## 13. Summary

| Aspect | Specification |
|--------|---------------|
| **Language** | Go 1.26+ |
| **External dependencies** | None (stdlib only) |
| **Database** | None |
| **Message broker** | None (Go channels for queuing) |
| **Container support** | Docker & Docker Compose (optional) |
| **Supported OS** | Linux, macOS, Windows |
| **Architecture** | x86-64, ARM64 |
| **Logging** | Stdout/stderr (standard `log` package) |
| **Configuration** | Environment variables |
| **Testing** | Go test suite + lab tool |
| **Linting** | `go fmt`, `go vet` + optional `golangci-lint` |
| **Documentation** | Inline comments + Markdown docs |
| **Build system** | Standard Go toolchain (`go build`, `go test`) |

