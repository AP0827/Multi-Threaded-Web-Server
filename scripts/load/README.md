# Load Test Scripts

This folder contains lightweight shell scripts to stress MTWS during demos and experiments.

## Files

- `burst_test.sh`: short, high-concurrency spike.
- `sustained_test.sh`: steady live load with per-second output (supports finite or infinite mode).
- `mixed_test.sh`: mixed traffic patterns (if present in your workflow).
- `_common.sh`: shared helpers used by scripts.

## Prerequisites

- MTWS server running on the target URL.
- `curl` installed.
- Run scripts with `bash`, not `go run`.

Correct:

```bash
bash scripts/load/burst_test.sh
bash scripts/load/sustained_test.sh
```

Incorrect:

```bash
go run ./scripts/load/burst_test.sh
```

## Sustained Test (Live Demo)

```bash
bash scripts/load/sustained_test.sh [URL] [DURATION] [RPS] [CONCURRENCY]
```

Defaults:

- URL: `http://localhost:8080/`
- DURATION: `60` seconds
- RPS: `30`
- CONCURRENCY: `20`

Examples:

```bash
# 2-minute demo, 40 req/s, concurrency 20
bash scripts/load/sustained_test.sh http://localhost:8080/ 120 40 20

# Run continuously until Ctrl+C
bash scripts/load/sustained_test.sh http://localhost:8080/ inf 40 20
```

The script prints per-second tick stats like:

```text
[  12s] tick= 40 | 200= 12 429= 28 503=  0 other=  0 | total=520
```

## Burst Test

```bash
bash scripts/load/burst_test.sh [URL] [TOTAL_REQUESTS] [CONCURRENCY]
```

Defaults:

- URL: `http://localhost:8080/`
- TOTAL_REQUESTS: `120`
- CONCURRENCY: `30`

## Why Queue Metrics Can Stay Flat

If dashboard queue depth remains near zero during tests, this is often expected:

1. Rate limiter rejects many requests early (429), before queueing.
2. Queue depth is sampled periodically, so very short spikes can be missed.
3. Workers may drain accepted jobs quickly if handlers are fast.

## How To Demonstrate Queue Pressure Clearly

Start server with limiter disabled:

```bash
MTWS_RATE_LIMIT_DISABLED=true go run ./cmd/server
```

Optional: reduce workers and queue capacity to force visible buildup:

```bash
MTWS_RATE_LIMIT_DISABLED=true MTWS_WORKERS=2 MTWS_JOB_QUEUE_SIZE=20 go run ./cmd/server
```

Then run sustained load:

```bash
bash scripts/load/sustained_test.sh http://localhost:8080/ inf 120 40
```

You should see queue depth and worker pressure move more clearly in the monitor.
