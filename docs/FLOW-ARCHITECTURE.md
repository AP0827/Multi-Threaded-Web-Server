# MTWS Data Flow & Architecture

## High-Level Flow

### Request Lifecycle

```
[Client Connection] 
    ↓
[TCP Listener Accept] (cmd/server/main.go)
    ↓
[Extract Client IP] 
    ↓
[Rate Limiter Check] (security/ratelimiter/limiter.go)
    ├─ Token Bucket per-IP
    ├─ If denied → Close connection
    └─ If allowed → Continue
    ↓
[Job Queue] (buffered Go channel, config: 200 capacity)
    ├─ Job contains: net.Conn + Router
    └─ Sent by listener goroutine
    ↓
[Worker Pool] (pool/pool.go, 10 workers by default)
    ├─ Each worker receives Job from channel
    ├─ Spawned at startup via StartWorkerPool()
    └─ Blocks reading from jobs channel
    ↓
[Connection Handler] (core/connection.go - HandleConnection)
    ├─ Set read deadline (5 seconds, anti-Slowloris)
    ├─ Parse HTTP/1.1 Request
    └─ If parse fails → 400/403, close
    ↓
[HTTP Parser] (http/parser.go)
    ├─ Strict HTTP/1.1 compliance
    ├─ Parse request line, headers, body/trailers
    ├─ WAF scans all fields (URI, headers, body, trailers)
    └─ If malicious pattern detected → 403 Forbidden + security log
    ↓
[Router Dispatch] (core/router.go)
    ├─ Extract path from request
    ├─ Lookup handler in routes map
    ├─ If not found → 404 Not Found
    └─ If found → execute handler
    ↓
[Handler Execution] (user-defined, configured at startup)
    ├─ Examples: /health, /echo, etc.
    ├─ Writes response via ResponseWriter
    └─ Handler must complete before connection closes
    ↓
[Response Writer] (core/response.go)
    ├─ Buffers response status + headers
    ├─ Writes body (simple or chunked)
    ├─ Flushes to socket
    ↓
[Connection Close] (defer in HandleConnection)
    └─ Resources cleaned up, worker ready for next job
```

---

## Queue & Buffering

### Job Queue (Buffered Channel)

- **Type:** Go channel of `pool.Job` struct
- **Capacity:** 200 (configurable: `JobQueueSize`)
- **Created at startup:** `jobs := make(chan pool.Job, config.JobQueueSize)`
- **Sender:** Main listener goroutine (in Accept loop)
- **Receivers:** Worker pool goroutines (10 workers, configurable)
- **Behavior:**
  - If queue is full → **listener blocks** until a worker drains a job
  - If queue is empty → **worker blocks** until a job arrives
  - Decouples accept rate from request processing rate

### Practical Queueing Example

```
Scenario: Listener accepting at 1000 req/sec, workers processing at 500 req/sec

Time 0.0s:    0 jobs queued
Time 0.1s:    50 jobs queued (1000 - 500 × 0.1)
Time 0.2s:    100 jobs queued
Time 0.35s:   175 jobs queued
Time 0.36s:   200 jobs queued → LISTENER BLOCKS (queue full)
Time 0.5s:    150 jobs queued (workers draining: -50 from full)
```

If listener stays blocked, new TCP SYN packets queue in the OS kernel's listen backlog.

---

## Component Interactions

### 1. **Listener → Rate Limiter → Job Queue**
```
For each accepted conn:
  1. Listener extracts clientIP
  2. Calls limiter.Allow(clientIP)
     - Token bucket: if tokens ≥ 1, deduct 1 and return true
     - Otherwise return false
  3. If true → post Job to queue
  4. If false → close conn immediately
```

### 2. **Worker Pool ↔ Job Queue**
```
StartWorkerPool(numWorkers, jobs chan):
  For i = 0 to numWorkers:
    spawn worker(i, jobs)
      Worker i loops:
        job := ← jobs  (blocks if empty)
        core.HandleConnection(job.Conn, job.Router)
```

### 3. **Connection Handler → Parser → WAF**
```
HandleConnection(conn, router):
  reader := bufio.NewReader(conn)
  req, err := mtwshttp.ParseRequest(reader)
    
    Inside ParseRequest:
      1. Read request line
      2. For each header:
         - WAF scans header name & value
      3. Parse body (Content-Length or chunked)
      4. If chunked, parse trailers
         - WAF scans trailer values
      5. Return req or SecurityError
```

### 4. **Router → Handlers**
```
Router.ServeHTTP(writer, req):
  path := extract_path(req)
  handler := router.routes[path]
  if handler == nil:
    writeErrorResponse(conn, "404 Not Found")
  else:
    handler(writer, req)  // user code runs here
```

---

## Concurrency Model

- **Listener goroutine:** 1 (main thread)
- **Worker goroutines:** Configurable (default: 10)
- **Total active goroutines at peak:** ~12 + any spawned by handlers
- **Synchronization:** 
  - Job queue (buffered channel) for listener ↔ workers
  - Rate limiter uses `sync.Mutex` to protect token buckets map
  - No shared mutable state between worker threads (stateless request handling)

---

## Security Scanning Pipeline

All request fields are scanned by the **Aho-Corasick automaton (WAF)** before routing:

| Field | Scanned? | When |
|-------|----------|------|
| Request URI | Yes | During request line parse |
| Header names | Yes | During header parse |
| Header values | Yes | During header parse |
| Body bytes (decoded) | Yes | During body/chunk parse |
| Trailer names | Yes | During trailer parse |
| Trailer values | Yes | During trailer parse |

If any match found → `SecurityError` returned → 403 Forbidden + security log.

---

## Configuration Points

| Setting | Default | Impact |
|---------|---------|--------|
| `ServerAddress` | `:8080` | Listen address |
| `WorkerPoolSize` | 10 | Number of concurrent request handlers |
| `JobQueueSize` | 200 | Buffered job queue capacity |
| `RateLimitRate` | 5.0 tokens/sec | Per-IP token refill rate |
| `RateLimitCapacity` | 10.0 tokens | Burst capacity per IP |
| `MTWS_RATE_LIMIT_DISABLED` | false | Disable rate limiter globally |
| `MTWS_BENCHMARK_MODE` | false | Disable rate limiter + other tuning |

---

## Error Handling

| Error Type | HTTP Response | Logged |
|------------|---------------|--------|
| Connection accept error | N/A (logged only) | Yes |
| Rate limit exceeded | Connection closed silently | No (unless logged by limiter) |
| Malformed request | 400 Bad Request | Yes (parse rejection) |
| WAF rule matched | 403 Forbidden | Yes (security event) |
| Invalid request target | 404 Not Found | Yes (path not in routes) |
| Handler panic | 500 Internal Server Error | Yes |
| Connection timeout | Connection closes | Yes (read deadline exceeded) |

---

## Performance Characteristics

- **Throughput:** Limited by worker pool size × handler latency + queue contention
- **Latency:** Request path duration (accept + queue wait + parse + route + handler + write)
- **Scalability:** Horizontal scaling requires distributing clients to multiple server instances; single process limited by OS file descriptors
- **Memory:** Per-connection buffer (bufio.Reader/Writer ~4KB + request/response structures), plus per-IP rate limiter state

