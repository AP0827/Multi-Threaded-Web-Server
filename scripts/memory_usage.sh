#!/usr/bin/env bash
set -euo pipefail

DEFAULT_COMMAND=(go test ./...)

if [[ "${1:-}" == "--" ]]; then
  shift
fi

if [[ $# -gt 0 ]]; then
  COMMAND=("$@")
else
  COMMAND=("${DEFAULT_COMMAND[@]}")
fi

if ! command -v ps >/dev/null 2>&1; then
  echo "ps is required but not installed" >&2
  exit 1
fi

echo "Memory usage monitor"
echo "Command: ${COMMAND[*]}"
echo

"${COMMAND[@]}" &
PID=$!
PEAK_RSS_KB=0

cleanup() {
  if kill -0 "$PID" >/dev/null 2>&1; then
    kill "$PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

while kill -0 "$PID" >/dev/null 2>&1; do
  RSS_KB="$(ps -o rss= -p "$PID" 2>/dev/null | tr -d ' ' || true)"
  if [[ -n "$RSS_KB" && "$RSS_KB" -gt "$PEAK_RSS_KB" ]]; then
    PEAK_RSS_KB="$RSS_KB"
  fi
  sleep 0.1
done

wait "$PID"
EXIT_CODE=$?
trap - EXIT

PEAK_RSS_MB="$(awk -v rss="$PEAK_RSS_KB" 'BEGIN { printf "%.2f", rss / 1024 }')"

echo "Peak RSS: ${PEAK_RSS_KB} KiB (${PEAK_RSS_MB} MiB)"
exit "$EXIT_CODE"
