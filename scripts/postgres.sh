#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
action=${1:-start}
if [[ "$action" != start && "$action" != stop ]]; then
  echo "Usage: $0 [start|stop]" >&2
  exit 1
fi
# Own native cluster when PostgreSQL tools are installed; otherwise use Compose.
if ! command -v initdb >/dev/null || ! command -v pg_ctl >/dev/null; then
  if [[ "$action" == start ]]; then docker compose up -d postgres; else docker compose stop postgres; fi
  exit
fi
poker_db_dir="$PWD/backend/data/postgres"
if [[ "$action" == stop ]]; then
  if [[ -f "$poker_db_dir/PG_VERSION" ]] && pg_ctl -D "$poker_db_dir" status >/dev/null 2>&1; then
    pg_ctl -D "$poker_db_dir" -m fast stop
  fi
  exit
fi
if [[ ! -f "$poker_db_dir/PG_VERSION" ]]; then
  mkdir -p "$PWD/backend/data"
  poker_password_file=$(mktemp)
  trap 'rm -f "$poker_password_file"' EXIT
  printf '%s\n' 'local-poker-password' > "$poker_password_file"
  initdb -D "$poker_db_dir" -U poker --pwfile="$poker_password_file" --auth-host=scram-sha-256 --auth-local=trust --locale=C.UTF-8 --encoding=UTF8
fi
if pg_ctl -D "$poker_db_dir" status >/dev/null 2>&1; then
  echo "Planning Poker PostgreSQL is already running."
else
  pg_ctl -D "$poker_db_dir" -l "$PWD/backend/data/postgres.log" -o '-h 127.0.0.1 -p 55432 -k /tmp' start
fi
