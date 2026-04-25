#!/usr/bin/env bash
set -euo pipefail

URL="${1:-http://localhost:8080/}"
TOTAL_REQUESTS="${2:-120}"
CONCURRENCY="${3:-30}"

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required but not installed"
  exit 1
fi

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

echo "Burst test"
echo "URL: $URL"
echo "Total requests: $TOTAL_REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo

START_MS="$(date +%s%3N)"
seq 1 "$TOTAL_REQUESTS" | xargs -n1 -P "$CONCURRENCY" -I{} sh -c 'curl -s -o /dev/null -w "%{http_code}\n" "$0"' "$URL" >> "$TMP_FILE"
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

if [ "$TOO_MANY_429" -gt 0 ]; then
  echo
  echo "Rate limiter is demonstrably active (429 responses observed)."
else
  echo
  echo "No 429 observed. Increase total requests/concurrency to trigger limiter."
fi
