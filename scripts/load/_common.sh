#!/usr/bin/env bash

ensure_curl() {
  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required but not installed"
    exit 1
  fi
}

status_name() {
  case "$1" in
    200) echo "OK" ;;
    400) echo "Bad Request" ;;
    429) echo "Too Many Requests" ;;
    503) echo "Service Unavailable" ;;
    000) echo "No Response" ;;
    *) echo "Other" ;;
  esac
}

print_status_summary() {
  local file_path="$1"
  local total_done
  total_done="$(wc -l < "$file_path" | tr -d ' ')"

  echo "Results"
  echo "- Completed: $total_done"

  local status code count name
  while read -r code count; do
    [ -n "$code" ] || continue
    name="$(status_name "$code")"
    echo "- ${code} ${name}: ${count}"
  done < <(
    awk '{ counts[$1]++ } END { for (code in counts) printf "%s %d\n", code, counts[code] }' "$file_path" | sort -n
  )
}
