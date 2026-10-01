#!/usr/bin/env bash
# Restore SiteWise onto an ISOLATED host for a rehearsal or a real recovery.
#
# The host must not be reachable by users: restored sessions and invites are
# live credentials. Run as root after installing the same packages and
# configuration as a production host (docs/operations.md, steps 1-6),
# without starting sitewise or Caddy.
#
#   restore.sh [--rehearsal] [--target 'YYYY-MM-DD HH:MM:SS+10'] [--expect facts.json]
#
# Without --target, recovery replays all archived WAL (latest state).
# --rehearsal turns WAL archiving off on this host, so a rehearsal never
# writes a new timeline into the production repository. Give a rehearsal
# host read-only repository credentials as well.
set -euo pipefail

target=""
expect=""
rehearsal=no
while [ $# -gt 0 ]; do
	case "$1" in
	--target) target="$2"; shift 2 ;;
	--expect) expect="$2"; shift 2 ;;
	--rehearsal) rehearsal=yes; shift ;;
	*) echo "unknown argument $1" >&2; exit 2 ;;
	esac
done

FILES=/var/lib/sitewise/files
PGDATA=/var/lib/postgresql/17/main
export RCLONE_CONFIG=/etc/sitewise/backup/rclone.conf
REMOTE=sitewise-crypt:files
DB_URL='postgres:///sitewise?host=/var/run/postgresql'

log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*"; }

if systemctl is-active --quiet sitewise; then
	echo "sitewise is running; a restore host must not serve users" >&2
	exit 1
fi

log "stop postgresql"
systemctl stop postgresql

log "pgbackrest restore"
args=(--stanza=sitewise --delta)
if [ -n "$target" ]; then
	args+=(--type=time "--target=$target" --target-action=promote)
fi
runuser -u postgres -- pgbackrest "${args[@]}" restore

if [ "$rehearsal" = yes ]; then
	echo "archive_mode = off" > /etc/postgresql/17/main/conf.d/zz-restore-rehearsal.conf
fi

log "start postgresql"
systemctl start postgresql
until runuser -u postgres -- pg_isready -q; do sleep 1; done

log "files restore"
install -d -o sitewise -g sitewise -m 0750 "$FILES"
rclone copy "$REMOTE" "$FILES" --transfers 8 --checkers 16
chown -R sitewise:sitewise "$FILES"

log "restore-check"
check=(/opt/sitewise/bin/sitewise restore-check -out /var/lib/sitewise/restore-facts.json)
if [ -n "$expect" ]; then
	check+=(-expect "$expect")
fi
runuser -u sitewise -- env SITEWISE_DATABASE_URL="$DB_URL" SITEWISE_FILE_DIR="$FILES" "${check[@]}"
log "restore verified"
