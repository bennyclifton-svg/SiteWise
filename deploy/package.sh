#!/usr/bin/env bash
# Build the release tarball: one static linux/amd64 binary with the web
# build, migrations, PDFium and latency budgets embedded, plus the intake
# vocabulary, building knowledge, profile policies and deployment files.
# Runs on Linux or Git Bash.
#
#   deploy/package.sh            -> dist/sitewise-<commit>-linux-amd64.tar.gz
#
# Refuses a dirty tree: a release must be reproducible from its commit.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd -P)
cd "$root"

if [ -n "$(git status --porcelain)" ] && [ "${ALLOW_DIRTY:-}" != 1 ]; then
	echo "working tree is dirty; commit first (ALLOW_DIRTY=1 for a local trial)" >&2
	exit 1
fi
commit=$(git rev-parse --short=12 HEAD)
if [ -n "$(git status --porcelain)" ]; then commit="$commit-dirty"; fi
name="sitewise-$commit-linux-amd64"
stage="$root/dist/$name"

# Never follow a substituted staging directory outside this checkout when
# replacing an earlier build of the same commit.
if [ -L "$root/dist" ] || [ -L "$stage" ]; then
	echo "release staging paths must not be symbolic links" >&2
	exit 1
fi
stage=$(realpath -m "$stage")
case "$stage" in
	"$root/dist/"*) ;;
	*) echo "release staging path is outside the checkout" >&2; exit 1 ;;
esac

rm -rf "$stage"
mkdir -p "$stage/bin" "$stage/share" "$stage/deploy" "$stage/docs"

npm --prefix web ci
npm --prefix web run build

# PDFium is WebAssembly run by wazero, so no cgo and no shared libraries.
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' \
	-o "$stage/bin/sitewise" ./cmd/sitewise

cp -r data/intake "$stage/share/intake"
cp -r data/profile "$stage/share/profile"
cp -r knowledge "$stage/share/knowledge"
cp deploy/sitewise.service deploy/sitewise-backup.service deploy/sitewise-backup.timer \
	deploy/Caddyfile deploy/sitewise.env.example deploy/pgbackrest.conf \
	deploy/postgresql-sitewise.conf deploy/backup.sh deploy/restore.sh "$stage/deploy/"
chmod 0755 "$stage/deploy/backup.sh" "$stage/deploy/restore.sh" "$stage/bin/sitewise"
cp docs/operations.md "$stage/docs/"
echo "$commit" > "$stage/COMMIT"

# Validate the files actually copied, including optional catalogue families
# whose absence would otherwise silently change the building model.
SITEWISE_RELEASE_ROOT="$stage" go test ./cmd/sitewise -run '^TestReleaseRuntimeAssets$' -count=1

tar -C dist -czf "dist/$name.tar.gz" "$name"
(cd dist && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
echo "dist/$name.tar.gz"
cat "dist/$name.tar.gz.sha256"
