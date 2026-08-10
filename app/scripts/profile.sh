#!/usr/bin/env bash
# Build the server, replay app/scripts/commands.txt against it while capturing
# a CPU profile, grab a heap snapshot at the end, always shut the server down.
set -euo pipefail

PORT=${PORT:-6379}
PPROF=localhost:6060
CMDS=${CMDS:-app/scripts/commands.txt}
CPU_SECONDS=${CPU_SECONDS:-20}
OUT=${OUT:-bench/$(date +%Y%m%d-%H%M%S)}
BIN=app/tmp/go-redis
LOADGEN=app/tmp/loadgen

cd "$(dirname "$0")/../.." # repo root
mkdir -p "$OUT" app/tmp

echo "==> building"
go build -o "$BIN" ./app
go build -o "$LOADGEN" ./app/scripts/loadgen

echo "==> starting server on :$PORT"
"$BIN" --port "$PORT" "$@" >"$OUT/server.log" 2>&1 &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null' EXIT

for _ in $(seq 1 50); do
  "$LOADGEN" -addr "localhost:$PORT" -file <(echo PING) >/dev/null 2>&1 && break
  kill -0 "$SERVER_PID" 2>/dev/null || { echo "server died:"; tail "$OUT/server.log"; exit 1; }
  sleep 0.1
done

echo "==> capturing ${CPU_SECONDS}s CPU profile while replaying $CMDS"
curl -fsS -o "$OUT/cpu.pprof" "http://$PPROF/debug/pprof/profile?seconds=$CPU_SECONDS" &
CURL_PID=$!

# Keep the server under load until the profile capture finishes.
while kill -0 "$CURL_PID" 2>/dev/null; do
  "$LOADGEN" -addr "localhost:$PORT" -file "$CMDS" -rounds 1000
done | tee "$OUT/results.txt"

wait "$CURL_PID"
curl -fsS -o "$OUT/heap.pprof" "http://$PPROF/debug/pprof/heap"

echo "==> done, artifacts in $OUT — explore with:"
echo "    go tool pprof -http=:8080 $BIN $OUT/cpu.pprof"
echo "    go tool pprof -http=:8081 $BIN $OUT/heap.pprof"
