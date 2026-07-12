#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/_common.sh"

URL="${1:-http://localhost:8080/}"
DURATION_ARG="${2:-60}"
RPS="${3:-30}"
CONCURRENCY="${4:-20}"

RUN_FOREVER=0
SECONDS_TO_RUN=0
if [[ "$DURATION_ARG" == "inf" || "$DURATION_ARG" == "infinite" || "$DURATION_ARG" == "0" ]]; then
  RUN_FOREVER=1
else
  SECONDS_TO_RUN="$DURATION_ARG"
fi

if ! [[ "$RPS" =~ ^[0-9]+$ ]] || [[ "$RPS" -le 0 ]]; then
  echo "RPS must be a positive integer"
  exit 1
fi

if ! [[ "$CONCURRENCY" =~ ^[0-9]+$ ]] || [[ "$CONCURRENCY" -le 0 ]]; then
  echo "Concurrency must be a positive integer"
  exit 1
fi

if [[ "$RUN_FOREVER" -eq 0 ]]; then
  if ! [[ "$SECONDS_TO_RUN" =~ ^[0-9]+$ ]] || [[ "$SECONDS_TO_RUN" -le 0 ]]; then
    echo "Duration must be a positive integer seconds value, or use inf"
    exit 1
  fi
fi

ensure_curl

TMP_FILE="$(mktemp)"
TICK_FILE="$(mktemp)"

cleanup() {
  rm -f "$TMP_FILE" "$TICK_FILE"
}
trap cleanup EXIT

echo "Sustained load test"
echo "URL: $URL"
if [[ "$RUN_FOREVER" -eq 1 ]]; then
  echo "Duration: infinite (Ctrl+C to stop)"
else
  echo "Duration: ${SECONDS_TO_RUN}s"
fi
echo "Target requests/sec: $RPS"
echo "Concurrency cap: $CONCURRENCY"
echo

START_MS="$(date +%s%3N)"
SECONDS_ELAPSED=0

while true; do
  TICK_START_MS="$(date +%s%3N)"
  : > "$TICK_FILE"

  seq 1 "$RPS" | xargs -P "$CONCURRENCY" -n1 sh -c 'curl -s -o /dev/null -w "%{http_code}\n" "$1"' _ "$URL" >> "$TICK_FILE"
  cat "$TICK_FILE" >> "$TMP_FILE"

  TICK_DONE="$(wc -l < "$TICK_FILE" | tr -d ' ')"
  TICK_200="$(awk '$1==200{c++} END{print c+0}' "$TICK_FILE")"
  TICK_429="$(awk '$1==429{c++} END{print c+0}' "$TICK_FILE")"
  TICK_503="$(awk '$1==503{c++} END{print c+0}' "$TICK_FILE")"
  TICK_OTHER="$(awk '$1!=200 && $1!=429 && $1!=503{c++} END{print c+0}' "$TICK_FILE")"

  TOTAL_DONE="$(wc -l < "$TMP_FILE" | tr -d ' ')"
  printf '[%4ss] tick=%3s | 200=%3s 429=%3s 503=%3s other=%3s | total=%s\n' \
    "$SECONDS_ELAPSED" "$TICK_DONE" "$TICK_200" "$TICK_429" "$TICK_503" "$TICK_OTHER" "$TOTAL_DONE"

  SECONDS_ELAPSED=$((SECONDS_ELAPSED + 1))
  if [[ "$RUN_FOREVER" -eq 0 && "$SECONDS_ELAPSED" -ge "$SECONDS_TO_RUN" ]]; then
    break
  fi

  TICK_END_MS="$(date +%s%3N)"
  TICK_DURATION_MS=$((TICK_END_MS - TICK_START_MS))
  if [[ "$TICK_DURATION_MS" -lt 1000 ]]; then
    sleep "$(awk -v ms="$TICK_DURATION_MS" 'BEGIN { printf "%.3f", (1000-ms)/1000 }')"
  fi
done
END_MS="$(date +%s%3N)"
DURATION_MS=$((END_MS - START_MS))

echo
print_status_summary "$TMP_FILE"
echo "- Duration: ${DURATION_MS}ms"

echo
TOO_MANY_429="$(awk '$1==429{c++} END{print c+0}' "$TMP_FILE")"
if [ "$TOO_MANY_429" -gt 0 ]; then
  echo "Rate limiter was active during sustained traffic (429 responses observed)."
else
  echo "No 429 observed. Increase RPS or duration, or lower limiter thresholds."
fi

echo
echo "Tip: for queue-pressure demos, run with limiter disabled:"
echo "  MTWS_RATE_LIMIT_DISABLED=true go run ./cmd/server"
