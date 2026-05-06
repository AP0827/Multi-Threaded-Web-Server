#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/_common.sh"

URL="${1:-http://localhost:8080/}"
TOTAL_REQUESTS="${2:-120}"
CONCURRENCY="${3:-30}"

ensure_curl

TMP_FILE="$(mktemp)"
trap 'rm -f "$TMP_FILE"' EXIT

echo "Burst test"
echo "URL: $URL"
echo "Total requests: $TOTAL_REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo

START_MS="$(date +%s%3N)"
seq 1 "$TOTAL_REQUESTS" | xargs -P "$CONCURRENCY" -I{} bash -c 'curl -s -o /dev/null -w "%{http_code}\n" "$1"' _ "$URL" >> "$TMP_FILE"
END_MS="$(date +%s%3N)"
DURATION_MS=$((END_MS - START_MS))

print_status_summary "$TMP_FILE"
echo "- Duration: ${DURATION_MS}ms"

TOO_MANY_429="$(awk '$1==429{c++} END{print c+0}' "$TMP_FILE")"
if [ "$TOO_MANY_429" -gt 0 ]; then
  echo
  echo "Rate limiter is demonstrably active (429 responses observed)."
else
  echo
  echo "No 429 observed. Increase total requests/concurrency to trigger limiter."
fi
