# Sprint 4 Experiment Guide

This sprint is about producing repeatable evidence, not just adding features.
The repository now includes a Go-native lab tool and a starter payload corpus so
you can compare MTWS directly against the split-proxy stack with the same exact
inputs.

## 1. Start the comparison stack

```bash
docker compose up --build
```

Targets:
- Direct MTWS: `127.0.0.1:8080`
- ModSecurity proxy in front of the standard backend: `127.0.0.1:8081`

## 2. Replay raw discrepancy and attack payloads

```bash
go run ./cmd/lab compare
```

Useful flags:
- `-mtws 127.0.0.1:8080`
- `-proxy 127.0.0.1:8081`
- `-payload-dir experiments/payloads`
- `-timeout 5s`
- `-json-out experiments/results/compare.json`

The `compare` subcommand opens a raw TCP connection, writes the payload bytes as
stored on disk, and records the first response line from each target. This is
important because discrepancy research depends on byte-exact replay rather than
high-level HTTP client normalization.

Interpretation:
- `200` on both sides means both stacks accepted the payload.
- `403` from MTWS indicates the in-parser WAF blocked a signature during parse.
- `4xx` divergence between MTWS and the proxy highlights an interpretation gap.
- A proxy-side `200` combined with an MTWS `403` is the specific thesis-aligned
  result to look for in signature-bearing requests.
- The proxy result now reflects ModSecurity plus a different backend parser,
  rather than MTWS behind a proxy, which makes divergence claims meaningful.

Starter payloads included:
- `001-benign-health.http`
- `010-uri-union-select-encoded.http`
- `020-header-script.http`
- `030-uri-traversal.http`
- `040-obs-fold-header.http`
- `050-duplicate-content-length.http`

The last two are especially useful for parser discrepancy exploration even when
they are not direct WAF-signature matches.

## 3. Benchmark direct versus proxy path

Direct MTWS:

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 200 -concurrency 10
```

With JSON export:

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8080/health -requests 200 -concurrency 10 -json-out experiments/results/benchmark-mtws.json
```

Proxy path:

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8081/health -requests 200 -concurrency 10
```

With JSON export:

```bash
go run ./cmd/lab benchmark -url http://127.0.0.1:8081/health -requests 200 -concurrency 10 -json-out experiments/results/benchmark-proxy.json
```

The benchmark reports:
- Throughput
- Average latency
- Min / P50 / P95 / P99 / Max latency
- Status-code counts
- Transport error counts

For a cleaner latency comparison:
- Use `/health`
- Keep concurrency modest at first
- Watch for `429` responses from the rate limiter
- If rate limiting dominates the benchmark, temporarily reduce request volume or
  increase capacity during a dedicated benchmark run

## 4. Suggested evidence workflow

1. Run `go run ./cmd/lab compare` and save the terminal output.
2. Add new payloads to `experiments/payloads/` whenever you find an ambiguity or
   published discrepancy case you want to reproduce.
3. Run direct and proxy benchmarks three or more times each.
4. Record median throughput and P95 latency across runs.
5. Use ModSecurity audit logs from the proxy container to explain why the proxy
   accepted or rejected each request.
6. Use `docs/final-report-template.md` to turn the captured JSON and logs into
   the final academic write-up.

## 5. Important research note

This harness gives you the structure needed to prove the thesis, but the actual
proof still depends on live execution against the Docker stack. Until those runs
are performed, the repository contains a validated experiment framework rather
than final empirical conclusions.
