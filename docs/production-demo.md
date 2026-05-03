# MTWS Production-Demonstration Guide

MTWS is still a research server, not a drop-in replacement for Nginx, Caddy,
Envoy, or Apache. This guide defines the production-demonstration posture: the
server is built, configured, observed, and shut down like a real service while
preserving the research objective of parser-integrated WAF inspection.

## What Is Production-Shaped Now

- Runtime configuration is loaded from environment variables instead of being
  fixed in code.
- The server handles `SIGINT` and `SIGTERM` and waits for workers to drain.
- A bounded worker queue rejects overload with `503 Service Unavailable`
  instead of blocking the accept loop forever.
- Each connection gets read and write deadlines.
- HTTP/1.1 keep-alive is supported with idle timeout and max-request limits.
- The Docker service runs as a non-root user, with dropped capabilities,
  `no-new-privileges`, and a read-only filesystem.
- `/health` proves the HTTP server can respond.
- `/ready` is used by Docker healthchecks to prove the service is ready for
  traffic.
- `/metrics` exports Prometheus-style counters for traffic, WAF blocks, parse
  rejects, queue rejects, rate limiting, and response classes.
- Optional TLS can be enabled with certificate and key file paths.
- A local self-signed certificate can be generated with `cmd/certgen` for
  HTTPS demonstrations.
- Security events, parser rejects, and access logs are structured JSON.

## Runtime Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `MTWS_ADDR` | `:8080` | TCP listen address |
| `MTWS_WORKERS` | `10` | Fixed worker goroutine count |
| `MTWS_JOB_QUEUE_SIZE` | `200` | Buffered connection queue capacity |
| `MTWS_READ_TIMEOUT` | `5s` | Per-connection request read deadline |
| `MTWS_WRITE_TIMEOUT` | `5s` | Response write deadline |
| `MTWS_IDLE_TIMEOUT` | `30s` | Keep-alive idle deadline after each response |
| `MTWS_MAX_KEEPALIVE_REQUESTS` | `100` | Maximum requests served on one TCP connection |
| `MTWS_QUEUE_TIMEOUT` | `250ms` | Maximum time to wait for worker queue space |
| `MTWS_SHUTDOWN_TIMEOUT` | `10s` | Maximum worker-drain wait during shutdown |
| `MTWS_RATE_LIMIT_DISABLED` | unset | Disables token-bucket limiting when true |
| `MTWS_BENCHMARK_MODE` | unset | Disables rate limiting for clean benchmarks |
| `MTWS_RATE_LIMIT_RATE` | `5.0` | Token refill rate per client per second |
| `MTWS_RATE_LIMIT_CAPACITY` | `10.0` | Token bucket burst capacity per client |
| `MTWS_WAF_POLICY_FILE` | unset | Path to line-based WAF signature policy |
| `MTWS_TLS_CERT_FILE` | unset | TLS certificate path |
| `MTWS_TLS_KEY_FILE` | unset | TLS key path |

TLS is enabled only when both `MTWS_TLS_CERT_FILE` and `MTWS_TLS_KEY_FILE` are
set. Supplying only one is treated as a startup error.

## HTTPS Demo

Generate a local certificate:

```powershell
go run ./cmd/certgen -force
```

Start MTWS with TLS:

```powershell
$env:MTWS_TLS_CERT_FILE="certs/mtws-local.crt"
$env:MTWS_TLS_KEY_FILE="certs/mtws-local.key"
go run ./cmd/server
```

Test with curl's local/self-signed bypass:

```powershell
curl -k https://127.0.0.1:8080/ready
```

## Local Demo Flow

Start MTWS:

```powershell
go run ./cmd/server
```

Check readiness:

```powershell
curl http://127.0.0.1:8080/ready
```

Check metrics:

```powershell
curl http://127.0.0.1:8080/metrics
```

Trigger a WAF block:

```powershell
curl "http://127.0.0.1:8080/search?q=union%20select"
```

Expected result: `403 Forbidden`, plus a structured `waf_block` log entry.

## Docker Demo Flow

Start the hardened comparison stack:

```powershell
docker compose -f docker-compose.yml -f docker-compose.benchmark.yml up --build
```

In another terminal:

```powershell
curl http://127.0.0.1:8080/ready
curl http://127.0.0.1:8080/metrics
go run ./cmd/lab compare -json-out experiments/results/compare-production-demo.json
```

Run a sustained keep-alive soak test:

```powershell
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -duration 2m -concurrency 10 -keepalive -json-out experiments/results/soak-mtws.json
```

## Demonstration Talking Points

- MTWS accepts raw TCP connections and does not use Go's `net/http` server for
  the protected path.
- Parsing and WAF inspection happen in the same loop, eliminating the split
  parser boundary that causes WAF discrepancy bypasses.
- Strict parsing rejects ambiguous requests early.
- Bounded workers, deadlines, rate limiting, and queue rejection address common
  availability risks.
- Keep-alive demonstrates realistic HTTP/1.1 client behavior while enforcing
  idle and max-request limits to prevent connection hoarding.
- Metrics and structured logs provide operational evidence during the demo.

## Honest Remaining Limitations

- MTWS does not implement HTTP/2 or HTTP/3 because the research scope is a
  custom HTTP/1.1 parser.
- TLS is supported at the listener level, but certificate lifecycle management
  is left to deployment tooling.
- The WAF is signature-based and is not a complete commercial WAF replacement.
- The project still needs external security review and long-running soak results
  from the target deployment environment before real internet exposure.
