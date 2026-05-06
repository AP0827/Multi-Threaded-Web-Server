#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/_common.sh"

URL="${1:-http://localhost:8080/}"
SECONDS_TO_RUN="${2:-10}"
RPS="${3:-20}"

ensure_curl

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

echo "Sustained load test"
echo "URL: $URL"
echo "Duration: ${SECONDS_TO_RUN}s"
echo "Target requests/sec: $RPS"
echo

START_MS="$(date +%s%3N)"
for _ in $(seq 1 "$SECONDS_TO_RUN"); do
  for _ in $(seq 1 "$RPS"); do
    curl -s -o /dev/null -w "%{http_code}\n" "$URL" >> "$TMP_FILE" &
  done
  wait
  sleep 1
done
END_MS="$(date +%s%3N)"
DURATION_MS=$((END_MS - START_MS))

print_status_summary "$TMP_FILE"
echo "- Duration: ${DURATION_MS}ms"

echo
TOO_MANY_429="$(awk '$1==429{c++} END{print c+0}' "$TMP_FILE")"
if [ "$TOO_MANY_429" -gt 0 ]; then
  echo "Rate limiter was active during sustained traffic (429 responses observed)."
else
  echo "No 429 observed. Increase RPS or duration to stress the limiter."
fi
