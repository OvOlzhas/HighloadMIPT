#!/usr/bin/env sh

set -eu

gateway_url="${GATEWAY_URL:-http://localhost:8080}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "1. Cache: the first request must be MISS, the second HIT"
cache_key="$(date +%s)"
cache_url="${gateway_url}/api/history?cache_demo=${cache_key}"

curl -sS -D "$tmp_dir/first.headers" -o "$tmp_dir/first.json" "$cache_url"
curl -sS -D "$tmp_dir/second.headers" -o "$tmp_dir/second.json" "$cache_url"

grep -i '^X-Cache-Status:' "$tmp_dir/first.headers"
grep -i '^X-Cache-Status:' "$tmp_dir/second.headers"

if command -v jq >/dev/null 2>&1; then
    first_time="$(jq -r '.server_time' "$tmp_dir/first.json")"
    second_time="$(jq -r '.server_time' "$tmp_dir/second.json")"
    echo "first server_time:  $first_time"
    echo "second server_time: $second_time"
    test "$first_time" = "$second_time"
else
    echo "jq is not installed; compare server_time manually in the two saved responses"
fi

echo
echo "2. Round-robin: requests should alternate between backend-1 and backend-2"
i=1
while [ "$i" -le 6 ]; do
    curl -sS -D - -o /dev/null -X POST \
        -H 'Content-Type: application/x-www-form-urlencoded' \
        --data 'bot_prefix=promo' \
        "${gateway_url}/api/generate" \
        | grep -i '^X-Backend-Instance:'
    i=$((i + 1))
    sleep 0.25
done

echo
echo "3. Rate limit: among 20 fast requests there must be HTTP 429 responses"
i=1
while [ "$i" -le 20 ]; do
    curl -sS -o /dev/null -w '%{http_code}\n' "${gateway_url}/api/history?rate_test=1"
    i=$((i + 1))
done
