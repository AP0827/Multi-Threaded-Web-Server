# Non-Functional Requirements and Core Components

Project: Multithreaded Secure HTTP Server (Go)

## 1. Non-Functional Requirements

### 1.1 Performance
- Handle concurrent connections efficiently without major response-time degradation.
- Target throughput: sustain high RPS under moderate load (at least 100 concurrent clients).
- Latency targets for static responses under normal load:
	- Average latency under 100 ms.
	- Stable tail latency (P95 and P99).
- Use efficient I/O handling via the Go runtime scheduler.

### 1.2 Scalability
- Scale with higher concurrency using a worker pool.
- CPU and memory growth should be predictable as load increases.
- Keep request handling stateless where possible to support horizontal scaling.

### 1.3 Reliability
- Remain operational under sustained load without crashes.
- Gracefully handle:
	- Malformed requests.
	- Connection drops.
	- Partial reads and writes.
- Maintain long-running stability (no memory leaks, no goroutine leaks).

### 1.4 Availability
- Continue serving valid requests under partial attack conditions (for example, rate-limited clients).
- Ensure fault isolation so failure in one request does not affect others.

### 1.5 Security
- Protect against:
	- Denial of Service (DoS).
	- Injection attacks (basic SQLi and XSS patterns).
	- Malformed request exploitation.
- Enforce input validation at parse stage.
- Enforce per-client (IP-based) rate limiting.
- Log suspicious and malicious activity.

### 1.6 Maintainability
- Keep a modular architecture with clear separation of concerns:
	- Networking.
	- Parsing.
	- Security.
	- Routing.
- Define clean interfaces between components.
- Keep code readable and structured.

### 1.7 Observability
- Log:
	- Requests.
	- Errors.
	- Security violations.
- Optional metrics:
	- Request rate.
	- Error rate.
	- Blocked requests.

### 1.8 Resource Efficiency
- Use bounded concurrency through a worker pool.
- Control memory allocation in request handling.
- Avoid unbounded queues and goroutine explosion.

## 2. Core System Components

### 2.1 TCP Listener
Description: Accepts incoming client connections on a configured port.

Responsibilities:
- Bind to port.
- Accept new connections.
- Forward accepted connections to the job queue.

Constraints:
- Must not block indefinitely.
- Must handle high connection rates.

### 2.2 Worker Pool
Description: Fixed number of worker goroutines processing incoming connections.

Responsibilities:
- Consume jobs from queue.
- Execute request lifecycle.
- Prevent uncontrolled concurrency.

Design characteristics:
- Bounded worker count (N workers).
- Channel-based job queue.
- Backpressure when queue is full.

### 2.3 Job Queue
Description: Buffered channel storing incoming connection tasks.

Responsibilities:
- Decouple connection acceptance from processing.
- Provide load buffering.

Constraints:
- Fixed capacity.
- Overflow handling strategy required (drop or reject).

### 2.4 HTTP Request Parser
Description: Parses raw TCP data into structured HTTP request objects.

Responsibilities:
- Extract method (GET, POST), path, headers, and optional body.
- Validate HTTP format.

Constraints:
- Must handle partial reads.
- Must reject malformed requests.

### 2.5 Router
Description: Maps request paths to handler functions.

Responsibilities:
- Route by URL path.
- Invoke the corresponding handler.

Example routes:
- / -> index handler.
- /api -> API handler.

### 2.6 Response Builder
Description: Builds valid HTTP responses.

Responsibilities:
- Set status line.
- Add headers.
- Attach response body.

Example:

HTTP/1.1 200 OK
Content-Length: X

<body>

### 2.7 Rate Limiter (Token Bucket)
Description: Controls per-client request rate to reduce abuse.

Model: Token Bucket

Per client state:
- Token count.
- Refill rate.

Behavior:
- Tokens are added over time.
- Each request consumes one token.
- If no token exists, request is rejected.

Responsibilities:
- Track per-IP usage.
- Enforce request limits.

### 2.8 WAF (Web Application Firewall)
Description: Applies basic pattern-based filtering for malicious inputs.

Responsibilities:
- Detect SQL injection patterns.
- Detect XSS payloads.
- Block suspicious requests.

Example patterns:
- UNION SELECT
- <script>

### 2.9 Input Validator
Description: Validates incoming request correctness and safety.

Responsibilities:
- Enforce size limits.
- Validate required headers.
- Reject malformed structures.

### 2.10 Logging System
Description: Records operational and security events.

Responsibilities:
- Log IP address, request path, status code, and timestamps.
- Record blocked or suspicious requests.

### 2.11 Connection Handler
Description: Manages lifecycle of each connection.

Responsibilities:
- Read request.
- Pass request through pipeline:

Parse -> Validate -> Rate Limit -> WAF -> Route -> Respond

- Close connection safely.

### 2.12 File Server (Static Content)
Description: Serves static files from disk.

Responsibilities:
- Map URL to file path.
- Prevent directory traversal (../).
- Return file content.

### 2.13 Security Middleware Layer
Description: Security checks applied before routing.

Components:
- Input validator.
- Rate limiter.
- WAF.

Execution order:

Request -> Validation -> Rate Limit -> WAF -> Router

## 3. Optional Advanced Components

### 3.1 TLS Layer
- Enables HTTPS communication.
- Uses Go standard library (crypto/tls).
- Provides confidentiality and integrity.

### 3.2 LRU Cache
Description: Caches frequently accessed responses (for example, static files).

Behavior:
- Store recent responses.
- Evict least-recently-used item when full.

Benefits:
- Lower disk I/O.
- Better response latency.

### 3.3 Metrics Collector
Tracks:
- RPS.
- Latency.
- Error rates.

Use case:
- Benchmarking and performance tuning.

### 3.4 IP Blacklisting
- Temporarily blocks abusive clients.
- Complements rate limiting.

## 4. Request Processing Pipeline

Incoming Connection
-> Job Queue
-> Worker Thread
-> Read Request
-> HTTP Parsing
-> Input Validation
-> Rate Limiting
-> WAF Filtering
-> Routing
-> Response Generation
-> Send Response
-> Connection Close

## 5. Summary

This system is designed as a modular, concurrent, and secure HTTP server with:
- Controlled concurrency (worker pool).
- Structured request-processing pipeline.
- Integrated security mechanisms (rate limiting, WAF, validation).
- Extensible architecture for advanced features (TLS, caching, IDS).

This aligns with:
- Systems design principles.
- Cryptography and network security objectives.