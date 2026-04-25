#!/usr/bin/env bash
set -euo pipefail

URL="${1:-http://localhost:8080/}"
SECONDS_TO_RUN="${2:-10}"
RPS="${3:-20}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required but not installed"
  exit 1
fi

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

TOTAL_DONE="$(wc -l < "$TMP_FILE" | tr -d ' ')"
OK_200="$(awk '$1==200{c++} END{print c+0}' "$TMP_FILE")"
TOO_MANY_429="$(awk '$1==429{c++} END{print c+0}' "$TMP_FILE")"
OTHER="$(awk '$1!=200 && $1!=429{c++} END{print c+0}' "$TMP_FILE")"

echo "Results"
echo "- Completed: $TOTAL_DONE"
echo "- 200 OK: $OK_200"
echo "- 429 Too Many Requests: $TOO_MANY_429"
echo "- Other status codes: $OTHER"
echo "- Duration: ${DURATION_MS}ms"

echo
if [ "$TOO_MANY_429" -gt 0 ]; then
  echo "Rate limiter was active during sustained traffic (429 responses observed)."
else
  echo "No 429 observed. Increase RPS or duration to stress the limiter."
fi
