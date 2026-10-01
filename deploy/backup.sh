#!/usr/bin/env bash
# Nightly SiteWise backup, run as root by sitewise-backup.service.
#
# 1. PostgreSQL: WAL is already archived continuously by archive_command;
#    this takes the weekly full or nightly differential base backup.
# 2. Files: blobs are named by their SHA-256 and never change, so copying is
#    incremental and --immutable turns any changed blob into an error rather
#    than an overwrite.
# 3. Verify: every local blob is present off-site with the same hash, the
#    archive is working, and the database still matches its blobs.
# A failed step exits nonzero, which systemd records and OnFailure can page.
set -euo pipefail

FILES=/var/lib/sitewise/files
STATE=/var/lib/sitewise-backup
RCLONE_CONF=/etc/sitewise/backup/rclone.conf
# rclone crypt remote over the object store bucket; see operations.md.
# Blobs and the nightly restore facts live in separate paths under it.
REMOTE=sitewise-crypt:files
FACTS=sitewise-crypt:facts
DB_URL='postgres:///sitewise?host=/var/run/postgresql'

export RCLONE_CONFIG="$RCLONE_CONF"
export RCLONE_CACHE_DIR="$STATE/rclone-cache"

log() { printf '%s %s\n' "$(date -u +%FT%TZ)" "$*"; }

install -d -m 0750 "$STATE"

# 1. Base backup. Sunday is full, other days differential.
type=diff
if [ "$(date +%u)" = 7 ]; then type=full; fi
log "pgbackrest $type backup"
runuser -u postgres -- pgbackrest --stanza=sitewise --type="$type" backup

# 2. Content-hash files.
log "files copy"
rclone copy "$FILES" "$REMOTE" --immutable --exclude '/tmp/**' --transfers 8 --checkers 16

# 3a. Every local blob exists off-site with a matching hash. cryptcheck
#     compares through the encryption layer.
log "files verify"
rclone cryptcheck "$FILES" "$REMOTE" --one-way --exclude '/tmp/**'

# 3b. Archiving works: forces a WAL switch and waits for it to reach the repo.
log "pgbackrest check"
runuser -u postgres -- pgbackrest --stanza=sitewise check

# 3c. Database and blobs agree, tenant references hold. The report holds
#     counts and hashes only and is kept for the next restore rehearsal.
log "restore-check"
facts=$(mktemp)
chown sitewise "$facts"
runuser -u sitewise -- env SITEWISE_DATABASE_URL="$DB_URL" SITEWISE_FILE_DIR="$FILES" \
	/opt/sitewise/bin/sitewise restore-check -out "$facts"
install -m 0640 "$facts" "$STATE/restore-facts.json"
rm -f "$facts"
rclone copyto "$STATE/restore-facts.json" "$FACTS/restore-facts-$(date -u +%Y%m%dT%H%M%SZ).json" --immutable

date -u +%FT%TZ > "$STATE/last-success"
log "backup complete"
