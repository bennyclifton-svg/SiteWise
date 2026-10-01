#!/usr/bin/env bash
# Build the release tarball: one static linux/amd64 binary with the web
# build, migrations, PDFium and latency budgets embedded, plus the intake
# vocabulary and these deployment files. Runs on Linux or Git Bash.
#
#   deploy/package.sh            -> dist/sitewise-<commit>-linux-amd64.tar.gz
#
# Refuses a dirty tree: a release must be reproducible from its commit.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root"

if [ -n "$(git status --porcelain)" ] && [ "${ALLOW_DIRTY:-}" != 1 ]; then
	echo "working tree is dirty; commit first (ALLOW_DIRTY=1 for a local trial)" >&2
	exit 1
fi
commit=$(git rev-parse --short=12 HEAD)
if [ -n "$(git status --porcelain)" ]; then commit="$commit-dirty"; fi
name="sitewise-$commit-linux-amd64"
stage="dist/$name"

rm -rf "$stage"
mkdir -p "$stage/bin" "$stage/share" "$stage/deploy" "$stage/docs"

npm --prefix web ci
npm --prefix web run build

# PDFium is WebAssembly run by wazero, so no cgo and no shared libraries.
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' \
	-o "$stage/bin/sitewise" ./cmd/sitewise

cp -r data/intake "$stage/share/intake"
cp deploy/sitewise.service deploy/sitewise-backup.service deploy/sitewise-backup.timer \
	deploy/Caddyfile deploy/sitewise.env.example deploy/pgbackrest.conf \
	deploy/postgresql-sitewise.conf deploy/backup.sh deploy/restore.sh "$stage/deploy/"
chmod 0755 "$stage/deploy/backup.sh" "$stage/deploy/restore.sh" "$stage/bin/sitewise"
cp docs/operations.md "$stage/docs/"
echo "$commit" > "$stage/COMMIT"

tar -C dist -czf "dist/$name.tar.gz" "$name"
(cd dist && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
echo "dist/$name.tar.gz"
cat "dist/$name.tar.gz.sha256"
