# Sprint 4 Experiment Guide

This sprint is about producing repeatable evidence, not just adding features.
The repository now includes a Go-native lab tool and a starter payload corpus so
you can compare MTWS directly against the split-proxy stack with the same exact
inputs.

## 1. Start the comparison stack

```bash
docker compose up --build
```

For benchmark-only runs, start with the benchmark override so MTWS does not
return `429` during latency measurement:

```bash
docker compose -f docker-compose.yml -f docker-compose.benchmark.yml up --build
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

Fixture note:
- `.http` payloads are normalized to canonical `\r\n` HTTP line endings before replay
- `.raw` payloads are sent byte-for-byte exactly as stored on disk

Interpretation:
- `200` on both sides means both stacks accepted the payload.
- `403` from MTWS indicates the in-parser WAF blocked a signature during parse.
- `4xx` divergence between MTWS and the proxy highlights an interpretation gap.
- A proxy-side `200` combined with an MTWS `403` is the specific thesis-aligned
  result to look for in signature-bearing requests.
- The proxy result now reflects ModSecurity plus a different backend parser,
  rather than MTWS behind a proxy, which makes divergence claims meaningful.

Current normalized comparison snapshot from `experiments/results/compare-normalized.json`:
- `030-uri-traversal.http`: MTWS `403`, proxy `400`
- `130-chunked-benign-body.http`: MTWS `200`, proxy `403`
- `150-header-foldlike-sqli-value.http`: MTWS `403`, proxy `200`

How to interpret the current divergences:
- `030` is a classification difference, not an acceptance gap. Both stacks block
  the traversal-style request, but they classify it differently.
- `130` currently behaves like a proxy-side false positive on a benign chunked
  `/submit` request. MTWS now routes that request correctly and accepts it.
- `150` is currently the strongest thesis-supporting result in the corpus: MTWS
  blocks a tab-obfuscated header SQLi pattern that the split-proxy path accepts.

Starter payloads included:
- `001-benign-health.http`
- `010-uri-union-select-encoded.http`
- `020-header-script.http`
- `030-uri-traversal.http`
- `040-obs-fold-header.http`
- `050-duplicate-content-length.http`
- `060-uri-double-encoded-traversal.http`
- `070-uri-backslash-traversal.http`
- `080-uri-encoded-script.http`
- `090-uri-sqli-comment-obfuscation.http`
- `100-uri-javascript-encoded-colon.http`
- `110-header-onerror-xss.http`
- `120-chunked-script-body.http`
- `130-chunked-benign-body.http`
- `140-uri-tab-sqli.http`
- `150-header-foldlike-sqli-value.http`
- `160-body-comment-sqli.http`
- `170-trailer-script.http`

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
- Whether HTTP keep-alive was enabled

For a cleaner latency comparison:
- Use `/health`
- Keep concurrency modest at first
- Watch for `429` responses from the rate limiter
- If rate limiting dominates the benchmark, temporarily reduce request volume or
  increase capacity during a dedicated benchmark run
- Add `-keepalive` when you want a connection-reuse benchmark instead of a
  cold-connection benchmark

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
7. Summarize repeated benchmark JSON files with:

```bash
go run ./cmd/lab summarize -glob "experiments/results/benchmark-*.json"
```

## 5. Current findings and next steps

What is already established:
- The normalization hardening fixed the earlier miss on
  `090-uri-sqli-comment-obfuscation.http`.
- Route-parity fixes removed the accidental MTWS `404` on
  `130-chunked-benign-body.http`; the request now returns `200 OK`.
- MTWS now shows a meaningful detection advantage on
  `150-header-foldlike-sqli-value.http`.

Recommended next steps:
1. Freeze the current evidence set.
   Save `experiments/results/compare-normalized.json`, benchmark JSON files, and
   any relevant proxy audit-log excerpts you plan to cite.
2. Investigate the proxy-side `130` false positive.
   Pull the matching ModSecurity audit-log entry and identify which CRS rule or
   anomaly-score path caused the `403`.
3. Investigate the proxy miss on `150`.
   Confirm from proxy logs whether the tab-obfuscated header passed without any
   anomaly score, or whether it scored below blocking threshold.
4. Decide your thesis framing for `030`.
   Treat it as "both blocked, different classification" unless you want to do a
   deeper parser-semantic comparison.
5. Run final benchmark repetitions.
   Capture at least three clean MTWS runs and three clean proxy runs on `/health`
   using the benchmark compose override, then summarize medians.
6. Fill out `docs/final-report-template.md`.
   Use `150` as the primary example of MTWS catching a signature-bearing request
   that the split-proxy path misses, and use `130` as a proxy false-positive
   example.

## 6. Important research note

This harness gives you the structure needed to prove the thesis, but the actual
proof still depends on live execution against the Docker stack. Until those runs
are performed, the repository contains a validated experiment framework rather
than final empirical conclusions.
