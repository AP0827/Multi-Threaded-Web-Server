# MTWS Final Report Template

## Title

Integrated In-Parser WAF Enforcement for HTTP/1.1:
An Experimental Study of Parsing-Discrepancy Resistance in MTWS

## Abstract

This report evaluates MTWS, a custom Go-based HTTP/1.1 server that integrates
WAF signature evaluation directly into the parsing loop, against a conventional
split-proxy deployment based on Nginx and ModSecurity CRS. The core hypothesis
is that when security inspection and parsing are the same operation, a large
class of parsing-discrepancy bypasses is structurally removed. The evaluation
compares byte-exact payload handling, blocking behavior, and latency overhead
between the two architectures.

Replace this abstract with final measured results once experiments are complete.

## 1. Research Question

Can an HTTP server that performs WAF signature matching inside its own parser
eliminate split-proxy parsing discrepancies without introducing meaningful
latency overhead?

## 2. System Design

### 2.1 MTWS Architecture

- Raw TCP accept loop
- Fixed worker pool
- Custom HTTP/1.1 parser
- In-parser signature matching
- Required `Host` enforcement and unsupported `Transfer-Encoding` rejection
- Strict request normalization
- Read-deadline-based Slowloris defense

### 2.2 Comparison Baseline

- `owasp/modsecurity-crs:4.25-nginx-lts`
- Reverse proxy architecture
- Separate backend using Go's standard `net/http` parser

## 3. Experimental Setup

### 3.1 Environment

- Host OS:
- CPU:
- Memory:
- Docker version:
- Go version:
- Test date:

### 3.2 Targets

- Direct MTWS: `http://127.0.0.1:8080`
- Split-proxy path: `http://127.0.0.1:8081`

### 3.3 Commands Used

Comparison run:

```bash
go run ./cmd/lab compare -json-out experiments/results/compare.json
```

Direct benchmark:

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 200 -concurrency 10 -json-out experiments/results/benchmark-mtws.json
```

Proxy benchmark:

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8081/health -requests 200 -concurrency 10 -json-out experiments/results/benchmark-proxy.json
```

## 4. Bypass and Discrepancy Results

### 4.1 Payload Outcome Table

| Payload | Intent | MTWS Result | Proxy Result | Divergence | Notes |
|---|---|---|---|---|---|
| 001-benign-health.http | Benign control | | | | |
| 010-uri-union-select-encoded.http | Encoded SQLi signature | | | | |
| 020-header-script.http | Header XSS signature | | | | |
| 030-uri-traversal.http | Traversal indicator | | | | |
| 040-obs-fold-header.http | Obsolete folding ambiguity | | | | |
| 050-duplicate-content-length.http | Duplicate length ambiguity | | | | |

### 4.2 Key Findings

- Payloads accepted by proxy but blocked by MTWS:
- Payloads rejected by both:
- Payloads producing parser-level divergence:
- Any unexpected false positives:

## 5. Latency and Throughput Results

### 5.1 Raw Results

| Path | Requests | Concurrency | Throughput (req/s) | Avg | P50 | P95 | P99 | Max | Errors |
|---|---|---|---|---|---|---|---|---|---|
| Direct MTWS | | | | | | | | | |
| Proxy Path | | | | | | | | | |

### 5.2 Interpretation

- Relative throughput change:
- Relative P95 latency change:
- Whether the overhead is statistically or operationally meaningful:

## 6. Analysis

### 6.1 Why MTWS Resists Split-Proxy Bypasses

Explain that MTWS does not hand inspected raw bytes to a different downstream
HTTP parser. The same parser that accepts or rejects the request also performs
signature matching, eliminating a second interpretation boundary.

### 6.2 Limitations

- Current signature set is intentionally small
- Only HTTP/1.1 request parsing is evaluated
- Results depend on the chosen discrepancy corpus
- More extensive benchmark repetition is still desirable

## 7. Conclusion

Summarize whether the experimental evidence supports the thesis:
- Structural bypass resistance:
- Observed discrepancies:
- Performance impact:

## Appendix A. Artifact Paths

- Comparison JSON:
- Benchmark JSON for MTWS:
- Benchmark JSON for proxy:
- ModSecurity audit logs:
- Docker Compose file:
- Payload corpus directory:
