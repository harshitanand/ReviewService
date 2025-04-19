#!/bin/sh
# wait-for-it.sh — waits for a TCP host:port to be available

set -e

host="$1"
shift
port="$1"
shift

timeout=60
interval=2
start_time=$(date +%s)

echo "⏳ Waiting for $host:$port to become available..."

until nc -z "$host" "$port"; do
  if [ $(( $(date +%s) - start_time )) -ge $timeout ]; then
    echo "❌ Timeout waiting for $host:$port after ${timeout}s"
    exit 1
  fi
  sleep $interval
done

echo "✅ $host:$port is now available — proceeding..."
exec "$@"
