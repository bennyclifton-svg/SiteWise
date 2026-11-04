#!/usr/bin/env bash
# Run a separate SiteWise for the synthetic harness, from bash on this host.
#
#   bash testing/synthetic/harness/serve.sh            # start (builds first)
#   bash testing/synthetic/harness/serve.sh stop
#
# Why separate: /dev/login signs in to the one "Local dev" organisation,
# which holds the owner's real projects. This instance uses its own
# database (sitewise_synth) and file store, on 127.0.0.1:8091, so test
# projects never mix with real ones. Postgres is the dev cluster on 5433
# (started here if it is down). Live Jev: the key comes from .env, as in
# tools/dev.ps1, and is never printed.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
TOOLS="$ROOT/.tools"
PGBIN="$TOOLS/pgsql-dist/pgsql/bin"
ADDR=127.0.0.1:8091
EXE="$TOOLS/sitewise-synth.exe"
PIDFILE="$TOOLS/synth-server.pid"
LOG="$TOOLS/synth-server.log"

if [[ "${1:-}" == stop ]]; then
  if [[ -f "$PIDFILE" ]]; then
    taskkill //PID "$(cat "$PIDFILE")" //F >/dev/null 2>&1 || true
    rm -f "$PIDFILE"
  fi
  echo stopped
  exit 0
fi

if ! "$PGBIN/pg_ctl.exe" -D "$TOOLS/pgdata" status >/dev/null 2>&1; then
  echo "starting PostgreSQL on 5433"
  "$PGBIN/pg_ctl.exe" -D "$TOOLS/pgdata" -l "$TOOLS/pg.log" -o "-p 5433" -w start >/dev/null
fi
if [[ "$("$PGBIN/psql.exe" -h 127.0.0.1 -p 5433 -U sitewise -d postgres -Atc "select 1 from pg_database where datname='sitewise_synth'")" != 1 ]]; then
  "$PGBIN/createdb.exe" -h 127.0.0.1 -p 5433 -U sitewise sitewise_synth
fi

# .env holds KEY=VALUE lines; a variable already set wins.
if [[ -f "$ROOT/.env" ]]; then
  while IFS='=' read -r name value; do
    name="$(echo "$name" | tr -d '[:space:]')"
    [[ -z "$name" || "$name" == \#* ]] && continue
    value="${value%$'\r'}"; value="${value#\"}"; value="${value%\"}"; value="${value#\'}"; value="${value%\'}"
    [[ -z "${!name:-}" ]] && export "$name=$value"
  done < "$ROOT/.env"
fi
export SITEWISE_JEV_API_KEY="${SITEWISE_JEV_API_KEY:-${TYPESAFE_API_KEY:-}}"
[[ -n "$SITEWISE_JEV_API_KEY" ]] || { echo "no Jev key in .env"; exit 1; }

cd "$ROOT"
export PATH="$TOOLS/go/bin:$PATH" GOCACHE="$TOOLS/go-cache" GOPATH="$TOOLS/go-path" GOTOOLCHAIN=local
go build -o "$EXE" ./cmd/sitewise

SECRET="$TOOLS/synth-session-secret"
[[ -f "$SECRET" ]] || head -c 32 /dev/urandom | base64 > "$SECRET"
export SITEWISE_DATABASE_URL="postgres://sitewise@127.0.0.1:5433/sitewise_synth?sslmode=disable"
export SITEWISE_FILE_DIR="$TOOLS/synth-files"
export SITEWISE_SESSION_SECRET="$(cat "$SECRET")"
export SITEWISE_JEV_MODEL=jev-1.13.0
export SITEWISE_ENV=development
mkdir -p "$SITEWISE_FILE_DIR"

# Same flags as tools/dev.ps1: provisional profile thresholds, dev login.
"$EXE" serve -addr "$ADDR" -dev-login -profile-provisional >"$LOG" 2>&1 &
echo $! > "$PIDFILE"
for _ in $(seq 1 240); do
  if curl -s -m 1 "http://$ADDR/health" >/dev/null 2>&1; then
    echo "SiteWise (synthetic) on http://$ADDR, log $LOG"
    exit 0
  fi
  sleep 0.25
done
echo "did not start; see $LOG"; tail -20 "$LOG"; exit 1
