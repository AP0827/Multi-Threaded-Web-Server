# Functional Requirements Specification (FRS)

**Project:** MTWS – Multithreaded Secure HTTP Server  
**Type:** Educational Network Security Research Server  
**Version:** 1.0  
**Date:** May 2026

---

## 1. Overview

This document specifies the **what** the system shall do. Each requirement describes a discrete function or capability that MTWS must provide.

---

## 2. Core HTTP Server Functionality

### FR-2.1 TCP Connection Acceptance
**Description:** The server shall accept incoming TCP connections on a configurable port.

- **Given:** Server is running and listening
- **When:** Client initiates TCP connection to `[listen_addr]:[port]`
- **Then:** Connection is accepted and handed off for HTTP request processing
- **Config:** `MTWS_LISTEN_ADDR` (default `:8080`)
- **Status:** ✓ Implemented

**Related code:** `cmd/server/main.go` (net.Listen, Accept loop)

---

### FR-2.2 HTTP/1.1 Request Parsing
**Description:** The server shall parse HTTP/1.1 requests according to RFC 7230–7231 with intentional restrictions for security research.

**Must parse:**
- **Request line:** `METHOD SP Request-Target SP HTTP-Version CRLF`
  - Valid methods: `GET`, `POST`, `PUT`, `DELETE`, `HEAD`, `OPTIONS`, `PATCH`, `TRACE`, `CONNECT` (per RFC)
  - Valid versions: Only `HTTP/1.1` (HTTP/1.0 rejected)
- **Headers:** Field-name `:` field-value (canonicalized, no duplicate headers)
- **Body framing:**
  - `Content-Length: N` → read exactly N bytes
  - `Transfer-Encoding: chunked` → parse chunk sizes, data, trailers
- **Trailers:** Optional, scanned like headers

**Shall reject (fail-closed):**
- Requests with invalid syntax (e.g., malformed request line)
- Requests with duplicate headers (ambiguity)
- Requests with both `Content-Length` and `Transfer-Encoding`
- Requests with HTTP version ≠ `HTTP/1.1`
- Invalid chunk encodings in chunked body
- Missing required `Host` header
- `Host` header with whitespace or control characters

**Response on parse failure:** `400 Bad Request`

**Related code:** `http/parser.go`, `http/request.go`

**Tests:** `http/parser_test.go`, `http/parser_fuzz_test.go`

**Status:** ✓ Implemented

---

### FR-2.3 HTTP Response Generation
**Description:** The server shall generate HTTP/1.1 responses with status code, headers, and optional body.

**Must support:**
- **Status codes:**
  - `200 OK` – Successful request
  - `400 Bad Request` – Malformed request
  - `403 Forbidden` – Security rule matched
  - `404 Not Found` – Route not found
  - `500 Internal Server Error` – Unhandled error
- **Headers:**
  - `Content-Type` (optional, set by handler)
  - `Content-Length` (automatic, computed from body)
  - `Connection: keep-alive` (future; currently close after each request)
  - `Date` (optional, set by handler)
- **Response format:**
  ```
  HTTP/1.1 200 OK\r\n
  Content-Type: text/plain\r\n
  Content-Length: 5\r\n
  \r\n
  hello
  ```

**Related code:** `core/response.go`

**Status:** ✓ Implemented

---

### FR-2.4 Request Routing
**Description:** The server shall route parsed requests to appropriate handlers based on request path.

**Must support:**
- **Route registration:** Handlers registered at startup (code-defined)
  ```go
  router.Handle("/health", healthHandler)
  router.Handle("/echo", echoHandler)
  ```
- **Path matching:** Exact match only (no wildcards, regex in v1.0)
- **Handler invocation:** Call registered handler function with request
- **404 handling:** Return 404 if path not found in routes

**Example routes (may vary by deployment):**
| Path | Handler | Response |
|------|---------|----------|
| `/health` | Health check | `200 OK` + `"healthy"` |
| (others) | Custom handlers | App-defined |

**Related code:** `core/router.go`

**Status:** ✓ Implemented

---

### FR-2.5 Connection Closure
**Description:** The server shall close connections cleanly after response is sent.

**Must:**
- Close socket after writing response
- Clean up buffers and resources
- Log connection closure (if configured)
- Handle client-initiated close gracefully

**Related code:** `core/connection.go` (defer conn.Close())

**Status:** ✓ Implemented

---

## 3. Concurrency & Job Queue

### FR-3.1 Worker Pool
**Description:** The server shall use a fixed-size worker pool for concurrent request processing.

**Must:**
- **Pool creation:** Initialize N worker goroutines at startup
- **Job queue:** Accept jobs (connections + router) into buffered channel
- **Worker blocking:** Workers block on empty queue
- **Listener blocking:** Listener blocks when queue is full
- **Config:**
  - `MTWS_WORKER_POOL_SIZE` (default 10)
  - `MTWS_JOB_QUEUE_SIZE` (default 200)

**Guarantees:**
- No more than N requests processed in parallel
- Bounded queue prevents unbounded goroutine creation
- Backpressure on listener when workers overloaded

**Related code:** `pool/pool.go`, `cmd/server/main.go`

**Status:** ✓ Implemented

---

### FR-3.2 Connection-to-Job Dispatch
**Description:** The server shall dispatch accepted connections as jobs to the worker pool.

**Must:**
- Listener accepts connection
- Extract client IP
- Rate limiter decision (FR-4.1)
- If allowed: Post Job to queue
- If denied: Close connection immediately

**Related code:** `cmd/server/main.go` (Accept loop + rate limiter check)

**Status:** ✓ Implemented

---

## 4. Security & Rate Limiting

### FR-4.1 Per-IP Rate Limiting
**Description:** The server shall enforce per-IP rate limiting using a token bucket algorithm.

**Must:**
- **Token bucket per IP:** One bucket per unique client IP
- **Refill rate:** Configurable tokens/second (default 5.0)
- **Capacity:** Configurable max burst (default 10.0)
- **Decision logic:**
  - If tokens ≥ 1: deduct 1 token, allow request
  - Else: deny request, close connection
  - On refill interval: add `rate * elapsed_time` tokens (capped at capacity)
- **Config:**
  - `MTWS_RATE_LIMIT_RATE` (default 5.0)
  - `MTWS_RATE_LIMIT_CAPACITY` (default 10.0)
  - `MTWS_RATE_LIMIT_DISABLED` (disable entirely)
- **Benchmark mode:** `MTWS_BENCHMARK_MODE=true` disables rate limiter

**Example (default config):**
```
IP 192.168.1.100:
  - Request 1 (t=0.0s):    10 tokens - 1 = 9 allowed
  - Request 2 (t=0.1s):    9 + (5.0*0.1) - 1 = 8.5 allowed
  - Request 10 (t=0.9s):   0.5 + (5.0*0.9) - 1 = 4.0 allowed
  - Request 11 (t=0.95s):  4.0 + (5.0*0.05) - 1 = 3.25 allowed
  - Request 50 (t=9.8s):   refill to 10, then -1 = 9 allowed
```

**Related code:** `security/ratelimiter/limiter.go`

**Tests:** (if present) Rate limiter unit tests

**Status:** ✓ Implemented

---

### FR-4.2 WAF (Web Application Firewall) Rule Matching
**Description:** The server shall scan request fields for malicious patterns using an Aho-Corasick automaton.

**Must:**
- **WAF engine:** Pattern-matching automaton
- **Policy file:** Load signatures from text file (default: `security/waf/default-policy.txt`)
- **Scanned fields:**
  - Request URI
  - Header names and values
  - Request body (decoded, if any)
  - Trailer names and values
- **Match behavior:**
  - If pattern matched → return `SecurityError` with field, pattern, rule ID
  - Parser rejects request immediately → `403 Forbidden`
- **Logging:** Log security events (field, pattern, client IP, rule ID, timestamp)
- **Policy format:** One pattern per line in policy file
  ```
  <script>
  union select
  ../../../
  ```

**Related code:** `security/waf/automaton.go`, `security/waf/policy.go`

**Tests:** `security/waf/automaton_test.go`

**Status:** ✓ Implemented

---

### FR-4.3 Malformed Request Rejection
**Description:** The server shall detect and reject malformed requests that could lead to security ambiguities.

**Must reject:**
- Invalid request line syntax
- Invalid header syntax
- Duplicate headers
- Both `Content-Length` and `Transfer-Encoding` in same request
- Invalid chunk sizes (chunked body)
- Invalid trailers
- Missing `Host` header
- Requests exceeding size limits (see limits in FR-2.2)

**Response:** `400 Bad Request`

**Related code:** `http/parser.go`

**Status:** ✓ Implemented

---

### FR-4.4 Connection Timeout
**Description:** The server shall enforce a read timeout to prevent Slowloris-style attacks.

**Must:**
- Set read deadline on accepted connection
- Timeout value: 5 seconds (hardcoded, configurable in future)
- On timeout: close connection, log event
- Effect: Clients must complete request within 5 seconds

**Related code:** `core/connection.go` (conn.SetReadDeadline)

**Status:** ✓ Implemented

---

## 5. Configuration & Startup

### FR-5.1 Environment Variable Configuration
**Description:** The server shall read all configuration from environment variables.

**Must:**
- **Startup phase:** Before listening, read env vars
- **Variables (see SRS-4.2):**
  - `MTWS_LISTEN_ADDR`
  - `MTWS_WORKER_POOL_SIZE`
  - `MTWS_JOB_QUEUE_SIZE`
  - `MTWS_RATE_LIMIT_RATE`
  - `MTWS_RATE_LIMIT_CAPACITY`
  - `MTWS_RATE_LIMIT_DISABLED`
  - `MTWS_BENCHMARK_MODE`
  - `MTWS_WAF_POLICY_FILE`
- **Defaults:** Use sensible defaults if not set
- **Validation:** Log warnings if invalid values (e.g., negative rate)

**Related code:** `config/config.go`

**Status:** ✓ Implemented

---

### FR-5.2 WAF Policy Loading
**Description:** The server shall load WAF signature patterns from a policy file at startup.

**Must:**
- **Policy file:** Plain text, one pattern per line
- **Location:** Configurable via `MTWS_WAF_POLICY_FILE`
- **Default:** `security/waf/default-policy.txt`
- **Parsing:** Skip empty lines, comments (if defined), treat non-empty lines as patterns
- **Initialization:** Build Aho-Corasick automaton from patterns on startup
- **Failure:** Log error and continue (or abort, depending on deployment)

**Related code:** `security/waf/policy.go`

**Status:** ✓ Implemented

---

### FR-5.3 Logging at Startup
**Description:** The server shall log startup configuration and status.

**Must log:**
- Server address and port
- Worker pool size
- Job queue size
- Rate limiter config (if enabled)
- WAF policy file path and pattern count
- Benchmark mode (if enabled)

**Example:**
```
Server running on port:8080!
Rate limiter enabled: rate=5.00 capacity=10.00
WAF loaded 42 patterns from security/waf/default-policy.txt
Worker pool size: 10
Job queue size: 200
```

**Related code:** `cmd/server/main.go`

**Status:** ✓ Implemented

---

## 6. Logging & Debugging

### FR-6.1 Request Logging
**Description:** The server shall log each request (method, path, version, client IP).

**Must log:**
- Client IP and port
- Request method and path
- HTTP version
- Timestamp

**Example:**
```
New request from 127.0.0.1:54321
Request method=GET path=/health version=HTTP/1.1
```

**Related code:** `core/connection.go` (log.Printf calls)

**Status:** ✓ Implemented

---

### FR-6.2 Security Event Logging
**Description:** The server shall log security events (rate limit denied, WAF rule matched, malformed requests).

**Must log for WAF matches:**
- Client IP
- Matched pattern and rule ID
- Scanned field (URI, header, body, trailer)
- HTTP status code (403)
- Timestamp

**Example:**
```
[SECURITY] Client 192.168.1.100 - Rule 001 matched in field=URI pattern='<script>' rule_id='001' status_code=403
```

**Related code:** `core/connection.go` (logSecurityEvent)

**Status:** ✓ Implemented

---

### FR-6.3 Parse Rejection Logging
**Description:** The server shall log requests rejected due to parsing errors.

**Must log:**
- Client IP
- Error reason (e.g., "duplicate headers", "invalid chunk size")
- HTTP status code (400)
- Timestamp

**Example:**
```
[PARSE_REJECT] Client 127.0.0.1 - Error: duplicate headers, status_code=400
```

**Related code:** `core/connection.go` (logParseReject)

**Status:** ✓ Implemented

---

## 7. Testing & Benchmarking

### FR-7.1 Unit Testing
**Description:** The system shall include unit tests for core components.

**Must test:**
- HTTP parser (valid and invalid requests)
- WAF pattern matching
- Rate limiter logic
- Configuration parsing
- Router dispatch

**Invocation:** `go test ./...`

**Related code:** `*_test.go` files throughout

**Status:** ✓ Implemented

---

### FR-7.2 Fuzz Testing
**Description:** The system shall include fuzz tests for fuzzing-friendly components.

**Must test:**
- HTTP parser with random/malformed input

**Invocation:** `go test -fuzz FuzzParseRequest ./http`

**Related code:** `http/parser_fuzz_test.go`

**Status:** ✓ Implemented

---

### FR-7.3 Benchmarking Tool (Lab)
**Description:** The server shall include a lab tool for performance benchmarking and comparative analysis.

**Must support:**
- **Benchmark mode:** Load test against URL, measure throughput/latency
  ```bash
  go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 1000 -concurrency 50
  ```
- **Compare mode:** Compare parsing behavior between MTWS and nginx+ModSecurity
  ```bash
  go run ./cmd/lab compare -json-out results.json
  ```
- **Payload replay:** Send test payloads to both servers, compare responses
- **Output:** JSON results with throughput, latency percentiles, success/failure counts

**Related code:** `cmd/lab/main.go`, `cmd/lab/main_test.go`

**Status:** ✓ Implemented

---

### FR-7.4 Benchmark Payloads
**Description:** The system shall include a suite of test payloads for security research.

**Must provide:**
- Benign payloads (health check, simple GET)
- Attack payloads:
  - SQL injection patterns
  - XSS patterns
  - Path traversal
  - Header injection
  - Chunked encoding edge cases
- Experimental payloads (HTTP ambiguity cases)

**Location:** `experiments/payloads/` (e.g., `001-benign-health.http`)

**Format:** Simple HTTP request files (`.http` extension)

**Status:** ✓ Implemented

---

## 8. Docker & Deployment

### FR-8.1 Docker Support
**Description:** The system shall be buildable and runnable as a Docker container.

**Must provide:**
- `Dockerfile` for building MTWS image
- Multi-stage build (optional, for optimization)
- Alpine Linux base (or minimal Linux distro)
- Expose port 8080 (or configured)
- Entrypoint: run `./mtws`

**Related code:** `Dockerfile`

**Status:** ✓ Implemented

---

### FR-8.2 Docker Compose Stack
**Description:** The system shall provide Docker Compose configurations for multi-service deployments.

**Must provide:**
- `docker-compose.yml` – Main stack (MTWS + nginx+ModSecurity)
- `docker-compose.benchmark.yml` – Override for benchmark mode (disable rate limiter)
- Services:
  - `mtws` – Listen on port 8080
  - `nginx-modsec` – Listen on port 8081 (proxy to standard backend)
- Volume mounts: WAF policies, benchmark results

**Related code:** `docker-compose.yml`, `docker-compose.benchmark.yml`

**Status:** ✓ Implemented

---

## 9. Example Handlers

### FR-9.1 Health Check Handler
**Description:** The server shall provide a `/health` endpoint for readiness checks.

**Must:**
- **Path:** `/health`
- **Method:** GET (or HEAD)
- **Response:**
  - Status: `200 OK`
  - Body: Plain text `"healthy"` or JSON `{"status":"ok"}`
- **Use case:** Container orchestration, load balancer health probes

**Related code:** `cmd/server/main.go` (buildRouter, healthHandler)

**Status:** ✓ Implemented

---

## 10. Error Handling

### FR-10.1 Graceful Error Responses
**Description:** The server shall respond to errors with appropriate HTTP status codes and error messages.

| Error Scenario | Status Code | Response Body |
|---|---|---|
| Malformed request | 400 Bad Request | (optional brief message) |
| WAF rule matched | 403 Forbidden | (optional brief message) |
| Route not found | 404 Not Found | (optional brief message) |
| Internal error | 500 Internal Server Error | (optional brief message) |

**Related code:** `core/response.go` (writeErrorResponse)

**Status:** ✓ Implemented

---

## 11. Future Enhancements (Out of Scope for v1.0)

- [ ] FR-11.1: TLS/HTTPS support
- [ ] FR-11.2: Keep-alive connections
- [ ] FR-11.3: Wildcard route matching
- [ ] FR-11.4: Custom error pages
- [ ] FR-11.5: Structured logging (JSON)
- [ ] FR-11.6: Prometheus metrics export
- [ ] FR-11.7: Graceful shutdown (SIGTERM)
- [ ] FR-11.8: Hot-reload of WAF policies
- [ ] FR-11.9: HTTP/2 support
- [ ] FR-11.10: Request body size streaming (for large uploads)

---

## 12. Acceptance Criteria Summary

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Accept TCP connections | ✓ | `cmd/server/main.go` |
| Parse HTTP/1.1 requests | ✓ | `http/parser.go` + tests |
| Generate HTTP responses | ✓ | `core/response.go` |
| Route requests to handlers | ✓ | `core/router.go` |
| Worker pool concurrency | ✓ | `pool/pool.go` |
| Rate limiting | ✓ | `security/ratelimiter/limiter.go` |
| WAF pattern matching | ✓ | `security/waf/automaton.go` |
| Environment config | ✓ | `config/config.go` |
| Logging | ✓ | `core/connection.go`, `logging.go` |
| Unit tests | ✓ | `*_test.go` files |
| Docker support | ✓ | `Dockerfile` |
| Lab tool (benchmarking) | ✓ | `cmd/lab/main.go` |
| Security payloads | ✓ | `experiments/payloads/` |

---

## 13. Summary

MTWS implements a **strict, security-conscious HTTP/1.1 server** suitable for:
- Learning HTTP parsing and security concepts
- Researching WAF effectiveness
- Benchmarking against reference implementations (nginx + ModSecurity)
- Teaching concurrency patterns (goroutines, channels, worker pools)

All functional requirements are **implemented and tested**.

