#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/_common.sh"

URL="${1:-http://localhost:8080/}"
VALID_REQUESTS="${2:-90}"
MALFORMED_REQUESTS="${3:-15}"
BURST_REQUESTS="${4:-60}"
CONCURRENCY="${5:-30}"
BURST_BODY_BYTES="${6:-262144}"

ensure_curl

parse_target() {
  local target_url="$1"
  local without_scheme host_port host path

  without_scheme="${target_url#http://}"
  host_port="${without_scheme%%/*}"
  host="${host_port%%:*}"
  if [[ "$host_port" == *:* ]]; then
    port="${host_port##*:}"
  else
    port="80"
  fi

  path="/${without_scheme#*/}"
  if [[ "$without_scheme" != *"/"* ]]; then
    path="/"
  fi

  printf '%s %s %s\n' "$host" "$port" "$path"
}

send_valid_request() {
  curl -s -o /dev/null -w "%{http_code}\n" "$1"
}

send_raw_request() {
  local host port path response_line status_code
  read -r host port path <<<"$(parse_target "$1")"

  exec 3<>"/dev/tcp/$host/$port"
  printf 'GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n' "$path" "$host" >&3
  IFS= read -r response_line <&3 || true
  exec 3>&-
  exec 3<&-

  status_code="$(awk '{print $2}' <<<"$response_line")"
  if [ -z "$status_code" ]; then
    status_code="000"
  fi
  printf '%s\n' "$status_code"
}

send_malformed_request() {
  local host port response_line status_code
  read -r host port _ <<<"$(parse_target "$1")"

  exec 3<>"/dev/tcp/$host/$port"
  printf 'MALFORMED REQUEST\r\n\r\n' >&3
  IFS= read -r response_line <&3 || true
  exec 3>&-
  exec 3<&-

  status_code="$(awk '{print $2}' <<<"$response_line")"
  if [ -z "$status_code" ]; then
    status_code="000"
  fi
  printf '%s\n' "$status_code"
}

TMP_FILE="$(mktemp)"
REQUEST_PLAN="$(mktemp)"
trap 'rm -f "$TMP_FILE" "$REQUEST_PLAN"' EXIT

echo "Mixed status test"
echo "URL: $URL"
echo "Valid requests: $VALID_REQUESTS"
echo "Malformed requests: $MALFORMED_REQUESTS"
echo "Burst requests: $BURST_REQUESTS"
echo "Concurrency: $CONCURRENCY"
echo "Burst body bytes: $BURST_BODY_BYTES"
echo

BURST_BODY="$(head -c "$BURST_BODY_BYTES" </dev/zero | tr '\0' 'a')"

for _ in $(seq 1 "$VALID_REQUESTS"); do echo valid; done > "$REQUEST_PLAN"
for _ in $(seq 1 "$MALFORMED_REQUESTS"); do echo malformed; done >> "$REQUEST_PLAN"
for _ in $(seq 1 "$BURST_REQUESTS"); do echo burst; done >> "$REQUEST_PLAN"

export -f parse_target send_valid_request send_raw_request send_malformed_request

START_MS="$(date +%s%3N)"
shuf "$REQUEST_PLAN" | xargs -P "$CONCURRENCY" -I{} bash -c '
  kind="$1"
  url="$2"
  case "$kind" in
    valid)
      send_valid_request "$url"
      ;;
    malformed)
      send_malformed_request "$url"
      ;;
    burst)
      curl -s -o /dev/null -w "%{http_code}\n" -X POST --data-binary "$BURST_BODY" "$url"
      ;;
  esac
' _ {} "$URL" >> "$TMP_FILE"
END_MS="$(date +%s%3N)"
DURATION_MS=$((END_MS - START_MS))

print_status_summary "$TMP_FILE"
echo "- Duration: ${DURATION_MS}ms"

echo
echo "This mix can surface 200, 400, 429, and 503 depending on server pressure."