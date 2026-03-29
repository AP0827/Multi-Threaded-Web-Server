Non-Functional Requirements & Core System Components

Project: Multithreaded Secure HTTP Server (Go)

1. Non-Functional Requirements (NFRs)
1.1 Performance
The system shall handle concurrent connections efficiently without significant degradation in response time.
Target throughput: sustain high Requests Per Second (RPS) under moderate load (≥100 concurrent clients).
Latency requirements:
Average latency within acceptable bounds (<100ms for static responses under normal load).
Tail latency (P95/P99) must remain stable under load.
Efficient I/O handling using non-blocking mechanisms (Go runtime scheduler).
1.2 Scalability
The system shall scale with increased concurrency using a worker pool model.
Resource usage (CPU, memory) must grow predictably with load.
Horizontal scalability should be conceptually supported (stateless request handling).
1.3 Reliability
The server must remain operational under sustained load without crashes.
Graceful handling of:
malformed requests
connection drops
partial reads/writes
Long-running stability (no memory leaks or goroutine leaks).
1.4 Availability
The server must continue serving valid requests even under partial attack conditions (e.g., rate-limited clients).
Fault isolation: failure in one request must not affect others.
1.5 Security
Protection against:
Denial of Service (DoS)
Injection attacks (basic SQLi/XSS patterns)
Malformed request exploitation
Input validation enforced at parsing stage.
Rate limiting enforced per client (IP-based).
Logging of suspicious and malicious activity.
1.6 Maintainability
Modular architecture with clear separation of concerns:
networking
parsing
security
routing
Clean interfaces between components.
Readable and well-structured codebase.
1.7 Observability
Logging of:
requests
errors
security violations
Metrics collection (optional):
request rate
error rate
blocked requests
1.8 Resource Efficiency
Bounded concurrency using worker pool.
Controlled memory allocation for request handling.
Avoidance of unbounded queues and goroutine explosion.
2. Core System Components
2.1 TCP Listener
Description

Accepts incoming client connections over a specified port.

Responsibilities
Bind to port
Accept new connections
Forward connections to job queue
Constraints
Must not block indefinitely
Must handle high connection rates
2.2 Thread Pool (Worker Pool)
Description

Fixed number of worker goroutines processing incoming connections.

Responsibilities
Consume jobs from queue
Execute request lifecycle
Prevent uncontrolled concurrency
Design Characteristics
Bounded worker count (N workers)
Channel-based job queue
Backpressure when queue is full
2.3 Job Queue
Description

Buffered channel storing incoming connection tasks.

Responsibilities
Decouple connection acceptance from processing
Provide load buffering
Constraints
Fixed capacity
Overflow handling strategy required (drop/reject)
2.4 HTTP Request Parser
Description

Parses raw TCP data into structured HTTP request objects.

Responsibilities
Extract:
method (GET, POST)
path
headers
body (if applicable)
Validate HTTP format
Constraints
Must handle partial reads
Must reject malformed requests
2.5 Router
Description

Maps request paths to appropriate handlers.

Responsibilities
Route based on URL path
Invoke corresponding handler function
Example
/        → index handler
/api     → API handler
2.6 Response Builder
Description

Constructs valid HTTP responses.

Responsibilities
Set status line
Add headers
Attach response body
Example
HTTP/1.1 200 OK
Content-Length: X

<body>
2.7 Rate Limiter (Token Bucket)
Description

Controls request rate per client to mitigate abuse.

Model: Token Bucket
Each client has:
token count
refill rate
Behavior
Tokens added over time
Request consumes 1 token
If no tokens → reject request
Responsibilities
Track per-IP usage
Enforce request limits
2.8 WAF (Web Application Firewall)
Description

Performs basic pattern-based filtering of malicious inputs.

Responsibilities
Detect:
SQL injection patterns
XSS payloads
Block suspicious requests
Example Patterns
"UNION SELECT"
"<script>"
2.9 Input Validator
Description

Validates incoming requests for correctness and safety.

Responsibilities
Enforce size limits
Validate headers
Reject malformed structures
2.10 Logging System
Description

Records system activity and security events.

Responsibilities
Log:
IP address
request path
status code
timestamps
Record blocked or suspicious requests
2.11 Connection Handler
Description

Manages lifecycle of each connection.

Responsibilities
Read request

Pass through pipeline:

parse → validate → rate limit → WAF → route → respond
Close connection safely
2.12 File Server (Static Content)
Description

Serves static files from filesystem.

Responsibilities
Map URL to file path
Prevent directory traversal (../)
Return file contents
2.13 Security Middleware Layer
Description

Pipeline of security checks applied before routing.

Components
Rate limiter
WAF
Input validator
Execution Order
Request → Validation → Rate Limit → WAF → Router
3. Optional Advanced Components
3.1 TLS Layer
Enables HTTPS communication
Uses standard library (crypto/tls)
Ensures confidentiality and integrity
3.2 LRU Cache
Description

Caches frequently accessed responses (e.g., static files).

Behavior
Stores recent responses
Evicts least recently used entry when full
Benefits
Reduces disk I/O
Improves response latency
3.3 Metrics Collector
Tracks:
RPS
latency
error rates
Useful for benchmarking
3.4 IP Blacklisting
Temporarily blocks abusive clients
Works alongside rate limiter
4. Request Processing Pipeline
Incoming Connection
    ↓
Job Queue
    ↓
Worker Thread
    ↓
Read Request
    ↓
HTTP Parsing
    ↓
Input Validation
    ↓
Rate Limiting
    ↓
WAF Filtering
    ↓
Routing
    ↓
Response Generation
    ↓
Send Response
    ↓
Connection Close
5. Summary

The system is designed as a modular, concurrent, and secure HTTP server with:

Controlled concurrency (worker pool)
Structured request processing pipeline
Integrated security mechanisms (rate limiting, WAF, validation)
Extensible architecture for advanced features (TLS, caching, IDS)

This ensures alignment with both:

systems design principles
cryptography and network security objectives