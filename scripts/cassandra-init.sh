#!/usr/bin/env bash
set -euo pipefail

HOST="${CASSANDRA_HOSTS%%,*}"
PORT="${CASSANDRA_PORT:-9042}"
KEYSPACE="${CASSANDRA_KEYSPACE}"

run_cql() {
  local statement="$1"
  local attempts=0
  local max_attempts=30
  while (( attempts < max_attempts )); do
    if cqlsh "$HOST" "$PORT" -e "$statement"; then
      return 0
    fi
    attempts=$(( attempts + 1 ))
    echo "cqlsh failed (attempt ${attempts}/${max_attempts}); retrying in 2s..." >&2
    sleep 2
  done
  echo "cqlsh exhausted retries for statement: $statement" >&2
  return 1
}

run_cql "CREATE KEYSPACE IF NOT EXISTS ${KEYSPACE} WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};"

run_cql "CREATE TABLE IF NOT EXISTS ${KEYSPACE}.event_reactions (
  event_id text,
  created_by text,
  like_value tinyint,
  created_at timestamp,
  PRIMARY KEY ((event_id), created_by)
);"

run_cql "CREATE INDEX IF NOT EXISTS event_reactions_like_value_idx ON ${KEYSPACE}.event_reactions (like_value);"
run_cql "CREATE INDEX IF NOT EXISTS event_reactions_created_by_idx ON ${KEYSPACE}.event_reactions (created_by);"
