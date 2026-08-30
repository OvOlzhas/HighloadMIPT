#!/usr/bin/env sh

set -eu

if ! command -v wrk >/dev/null 2>&1; then
    echo "wrk is required. Install it and run this script again."
    exit 1
fi

duration="${DURATION:-15s}"
threads="${THREADS:-4}"
results_dir="results"
mkdir -p "$results_dir"

run_get() {
    name="$1"
    url="$2"
    connections="$3"
    echo "Running $name with c=$connections"
    wrk -t"$threads" -c"$connections" -d"$duration" --latency "$url" | tee "$results_dir/${name}-c${connections}.txt"
    echo ""
}

run_post() {
    name="$1"
    url="$2"
    connections="$3"
    echo "Running $name with c=$connections"
    wrk -t"$threads" -c"$connections" -d"$duration" --latency -s scripts/post.lua "$url" | tee "$results_dir/${name}-c${connections}.txt"
    echo ""
}

# Port 8081 is a direct request to backend-1
# Port 8090 is NGINX without rate limiting and is bound only to localhost
# Port 8080 remains the normal protected API Gateway
for connections in 10 100 1000; do
    run_get "direct-get" "http://localhost:8081/api/history" "$connections"
    run_post "direct-post" "http://localhost:8081/api/generate" "$connections"
    run_post "nginx-post" "http://localhost:8090/api/generate" "$connections"

    curl -sS -o /dev/null "http://localhost:8090/api/history?benchmark=1"
    run_get "nginx-cached-get" "http://localhost:8090/api/history?benchmark=1" "$connections"
done
