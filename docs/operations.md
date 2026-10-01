# SiteWise operations

How to build the production host from clean, watch it, back it up, restore
it, and ship a change. The files referred to are in `deploy/` and ship in the
release tarball. Written 1 October 2026 (instant-intake plan, Task 12).

**Status.** The artifacts, the health and speed endpoints and `restore-check`
are built and tested. A local restore rehearsal passed (see
[evidence](evidence/2026-10-01-restore-rehearsal-local/README.md)). The
VPS rebuild, the live latency gate on the VPS and the off-site restore
rehearsal have **not** been run. They are the release gates in
[section 7](#7-release-gates-before-the-first-invitation). No invitation is
issued until all of them pass.

## Host layout

| What | Where | Owner, mode |
|---|---|---|
| Release (binary, intake vocabulary, deploy files, this doc) | `/opt/sitewise-releases/<commit>/` | root, 0755 |
| Current release | `/opt/sitewise` → symlink to one release | root |
| Secrets and settings | `/etc/sitewise/sitewise.env` | root:root 0600 |
| Backup credentials (rclone) | `/etc/sitewise/backup/rclone.conf` | root:root 0600 |
| pgBackRest config | `/etc/pgbackrest/pgbackrest.conf` | root:postgres 0640 |
| Blobs, named by SHA-256 | `/var/lib/sitewise/files/<2 hex>/<sha256>` | sitewise, 0750 dir |
| Backup state (`last-success`, nightly facts) | `/var/lib/sitewise-backup/` | root, 0750 |
| PostgreSQL 17 data | `/var/lib/postgresql/17/main` | postgres |
| App listener | `127.0.0.1:8080` (Caddy only) | |
| Database | unix socket `/var/run/postgresql`, peer auth | |

The service user `sitewise` cannot read its own secrets file, cannot write
outside `/var/lib/sitewise`, and has no capabilities (`deploy/sitewise.service`).
systemd reads the environment file as root before dropping privileges.

## 1. Clean VPS build

Target: Ubuntu 24.04 LTS, x86_64, at least 2 vCPU / 4 GB RAM / NVMe, in an
Australian region. The old VPS is replaced, not upgraded. Only these carry
over: the TypeSafe Jev API key, the mail relay credentials, DNS access and
the ACME contact address. Generate the session secret fresh. Nothing else
moves: no database, no Supabase backup, no old files.

1. **Base.** Create the VPS with SSH key login only. Then:
   ```sh
   apt update && apt full-upgrade -y
   apt install -y ufw unattended-upgrades curl gnupg ca-certificates
   timedatectl set-timezone Australia/Sydney
   ufw default deny incoming && ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw enable
   sed -i 's/^#\?PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && systemctl reload ssh
   ```
2. **Packages.** PostgreSQL 17 and pgBackRest from the PGDG repository,
   Caddy from its official repository, rclone from Ubuntu.
   ```sh
   install -d /usr/share/postgresql-common/pgdg
   curl -fsSo /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc https://www.postgresql.org/media/keys/ACCC4CF8.asc
   echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt noble-pgdg main" > /etc/apt/sources.list.d/pgdg.list
   curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/gpg.key | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
   curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt > /etc/apt/sources.list.d/caddy-stable.list
   apt update && apt install -y postgresql-17 pgbackrest caddy rclone
   ```
3. **Service user and directories.**
   ```sh
   useradd --system --home-dir /var/lib/sitewise --shell /usr/sbin/nologin sitewise
   install -d -m 0750 /etc/sitewise /etc/sitewise/backup /var/lib/sitewise-backup
   install -d -o postgres -g postgres -m 0750 /var/spool/pgbackrest /var/log/pgbackrest
   ```
4. **Release.** Build with `deploy/package.sh` from a clean commit on the
   build machine, copy the tarball and its `.sha256`, then:
   ```sh
   sha256sum -c sitewise-<commit>-linux-amd64.tar.gz.sha256
   install -d /opt/sitewise-releases
   tar -C /opt/sitewise-releases -xzf sitewise-<commit>-linux-amd64.tar.gz
   ln -sfn /opt/sitewise-releases/sitewise-<commit>-linux-amd64 /opt/sitewise
   ```
5. **PostgreSQL.** Socket only, archiving to pgBackRest.
   ```sh
   cp /opt/sitewise/deploy/postgresql-sitewise.conf /etc/postgresql/17/main/conf.d/sitewise.conf
   runuser -u postgres -- createuser --no-superuser --no-createrole --no-createdb sitewise
   runuser -u postgres -- createdb --owner sitewise sitewise
   ```
   Ubuntu's default `pg_hba.conf` has `local all all peer`, so OS user
   `sitewise` reaches role `sitewise` with no password. The app applies its
   own migrations at start under an advisory lock.
6. **Off-site storage.** Create one S3-compatible bucket in an Australian
   region, with versioning or object lock enabled. Create two keys: a
   read-write key for this host, and a read-only key for restore rehearsals.
   - Copy `deploy/pgbackrest.conf` to `/etc/pgbackrest/pgbackrest.conf` and
     fill the `REPLACE_*` values. The cipher passphrase is generated once
     (`openssl rand -base64 48`) and also stored off-host in the owner's
     password manager. Without it, no backup can be restored.
   - Configure rclone: `rclone config --config /etc/sitewise/backup/rclone.conf`.
     Create an `s3` remote named `sitewise-s3` on the same bucket, then a
     `crypt` remote named `sitewise-crypt` over `sitewise-s3:<bucket>/sitewise`
     with two generated passwords, also kept off-host.
   - Initialise and prove archiving:
     ```sh
     runuser -u postgres -- pgbackrest --stanza=sitewise stanza-create
     systemctl restart postgresql
     runuser -u postgres -- pgbackrest --stanza=sitewise check
     runuser -u postgres -- pgbackrest --stanza=sitewise --type=full backup
     ```
7. **Mail.** The app sends invites with SMTP and no AUTH to
   `SITEWISE_SMTP_ADDR`. Run a local relay that holds the provider
   credentials: a Postfix "null client" bound to `127.0.0.1:25`, relaying to
   the provider on port 587 with SASL and TLS. Send a test message through it
   before going further.
8. **Secrets.** `install -m 0600 /opt/sitewise/deploy/sitewise.env.example /etc/sitewise/sitewise.env`,
   then fill in the values. Session secret: `openssl rand -hex 32`. Never put
   a value in a shell history line: edit the file.
9. **Units.**
   ```sh
   cp /opt/sitewise/deploy/sitewise.service /opt/sitewise/deploy/sitewise-backup.service /opt/sitewise/deploy/sitewise-backup.timer /etc/systemd/system/
   systemd-analyze verify /etc/systemd/system/sitewise.service /etc/systemd/system/sitewise-backup.service
   systemctl daemon-reload
   systemctl enable --now sitewise sitewise-backup.timer
   journalctl -u sitewise -n 20     # expect "sitewise listening addr=127.0.0.1:8080 model=jev-1.13.0"
   ```
   A missing or blank secret stops the service at start with
   `missing configuration: <NAMES>`. Values are never printed.
   The sandbox has not yet run under real systemd (only `systemd-analyze
   verify` on Ubuntu 24.04's systemd 255). On first start, check
   `systemd-analyze security sitewise`, then drop one PDF. PDFium runs as
   WebAssembly compiled by wazero, so a sandbox fault shows up there first.
   If it fails, read the denied call from `journalctl -u sitewise` and
   relax that one setting only.
10. **Caddy.** Copy `deploy/Caddyfile` to `/etc/caddy/Caddyfile`, replace
    `REPLACE_ACME_EMAIL`, then `caddy validate --config /etc/caddy/Caddyfile`
    and `systemctl reload caddy`. Certificates are issued once DNS points
    here (step 12).
11. **First owner.** `runuser -u sitewise -- env $(grep -v '^#' /etc/sitewise/sitewise.env | xargs) /opt/sitewise/bin/sitewise bootstrap -org "<org>" -email <owner>`
    creates the org and mails the owner an invite. Do this only after
    section 7 passes.
12. **DNS cutover.** Point `sitewise.au` and `www.sitewise.au` A/AAAA records
    at the new host. Only do this after section 7 passes, then retire the old VPS.

## 2. Configuration

| Variable | Required | Notes |
|---|---|---|
| `SITEWISE_DATABASE_URL` | yes | `postgres:///sitewise?host=/var/run/postgresql` |
| `SITEWISE_FILE_DIR` | yes | `/var/lib/sitewise/files` |
| `SITEWISE_SESSION_SECRET` | yes | fresh random |
| `SITEWISE_JEV_API_KEY` | yes | authorized TypeSafe key |
| `SITEWISE_JEV_MODEL` | yes | must be `jev-1.13.0`; aliases are refused |
| `SITEWISE_ENV` | production | `production` sets secure cookies and requires the three below |
| `SITEWISE_PUBLIC_ORIGIN` | in production | `https://sitewise.au`; mutating requests must carry this Origin |
| `SITEWISE_SMTP_ADDR` | in production | local relay, `127.0.0.1:25` |
| `SITEWISE_MAIL_FROM` | in production | invite sender |

`serve` flags: `-addr` (loopback only), `-data` (intake vocabulary and
cut-offs, shipped in `share/intake`), `-max-upload` (default 200 MiB, which
must stay below Caddy's `request_body max_size`).

## 3. Health and speed

| Endpoint | Who | Answers |
|---|---|---|
| `GET /healthz` | anyone (Caddy, uptime monitor) | `{"status":"ok"}` or `"degraded"` with 200; `"down"` with 503 |
| `GET /api/health` | org owners | database, Jev circuit and last reach, backlog of the caller's own org, reasons |
| `GET /api/speed` | org owners | recent p50/p90 per budget path against `bench/budgets.json` |

The public answer is the status word only: no counts, ages, versions or
reasons. Each change of public status is logged with its reasons
(`journalctl -u sitewise | grep health`). That log is the operator's view of
the whole process. `/api/health` shows only the caller's org queue, so no
tenant learns another tenant's activity.

Status rules (`internal/httpapi/health.go`):

| Signal | Degraded when | Down when |
|---|---|---|
| Database ping (500 ms bound) | | it fails or hangs |
| Jev reachability | no response from the endpoint for over 2 min (the probe runs every 30 s), or none since start | |
| Jev circuit breaker | open or half-open | |
| Intake backlog (oldest pending filing) | older than 30 s | |
| Background backlog (oldest of each stage) | older than 15 min | |

Nothing on these pages calls Jev. Reachability comes from an
unauthenticated `HEAD` probe that costs no evaluation and no slot.
Percentiles come from an in-memory window of the last 1,024 observations
per path. `within_budget` stays null until a path has the gate's 20 samples.
Server-side `whole_intake` runs from the filing start to its event write. The
release gate's `whole_intake` (`cmd/intake-bench`) also includes the final
upload byte and SSE delivery.

**Known:** the background Jev worker loop is not yet started by `serve`.
`full_text` and `jev_retry` jobs therefore accumulate, and health turns
`degraded` 15 minutes after the first filing. That is a true signal, and it
is a release gate (section 7).

## 4. Backups

| What | How | When | Retention |
|---|---|---|---|
| PostgreSQL WAL | `archive_command` → pgBackRest → bucket, encrypted client-side | continuously; at least every 60 s (`archive_timeout`) | covers all retained base backups |
| PostgreSQL base backup | `backup.sh` → `pgbackrest backup` | nightly 02:30 Sydney; full on Sunday, differential otherwise | 4 fulls (~1 month PITR) |
| Blobs | `backup.sh` → `rclone copy --immutable` to the crypt remote | nightly | never deleted: content-addressed and immutable |
| Restore facts | `backup.sh` → `sitewise restore-check -out` → `facts/` | nightly | kept |

Every nightly run verifies itself. `rclone cryptcheck --one-way` proves
every local blob is off-site with a matching hash. `pgbackrest check` forces
a WAL switch and waits for it to arrive. `restore-check` proves each
database file row has a blob whose bytes hash to its name, and that no row
references another org. Any failure fails the unit:
`systemctl status sitewise-backup`, `journalctl -u sitewise-backup`.
`/var/lib/sitewise-backup/last-success` holds the time of the last full
success. Alert if it is older than 26 hours.

`restore-check` reads every blob each night. That is fine at today's size.
Revisit it if the file store passes roughly 100 GB.

## 5. Restore and rehearsal

`deploy/restore.sh` restores onto a host built with section 1 steps 1–9, with
`sitewise` and Caddy stopped. It restores PostgreSQL with pgBackRest (latest
state, or `--target` a point in time), copies blobs back, then runs
`sitewise restore-check`. With `--expect`, that check compares the result to
a facts report from the source.

`restore-check` needs only `SITEWISE_DATABASE_URL` and `SITEWISE_FILE_DIR`,
so a rehearsal host never holds the Jev key or session secret. It never
migrates: a missing migration must fail the check, not be repaired. It
fails on:
- a referenced blob that is missing, or whose bytes do not hash to its name
- an unvalidated foreign key
- any row whose parent is not in the row's own org. This includes
  `events.document_id`, which has no foreign key.
- with `--expect`: any difference in per-org counts (users, memberships,
  projects, files, documents, decisions, passages, events), foreign key
  count, applied migrations, or the digest of all referenced blob hashes.

**Rehearsal procedure (required before the first invitation, then quarterly):**

1. On production, while no one is using it (before invitations, or in a
   maintenance window): `runuser -u sitewise -- env SITEWISE_DATABASE_URL=... SITEWISE_FILE_DIR=/var/lib/sitewise/files /opt/sitewise/bin/sitewise restore-check -out /root/source-facts.json`,
   then `systemctl start sitewise-backup` and wait for it to succeed.
2. Build a separate VPS with section 1 steps 1–9 and **no** DNS and **no**
   public firewall ports other than SSH. Give it the read-only bucket key.
   Restored sessions and invites are live credentials; the local rehearsal
   proved a restored session cookie still signs in.
3. Copy `source-facts.json` across. Run
   `deploy/restore.sh --rehearsal --expect source-facts.json`.
   `--rehearsal` turns WAL archiving off there, so the rehearsal never writes
   a timeline into the production repository.
4. Start `sitewise` on the rehearsal host, bound to loopback. Through an SSH
   tunnel, sign in as two orgs' owners and confirm each sees its own
   documents and gets 404 for the other's project and document IDs.
5. Record the outputs, timings and the commit in
   `docs/evidence/<date>-restore-rehearsal-vps/`, without org names or
   document content. Then destroy the rehearsal host.

## 6. Deployment and change checklist

Every change to production follows this list in order. Copy it into the
change record and tick each line.

**Before**
- [ ] Commit is on `main`, and `deploy/package.sh` ran from a clean tree.
- [ ] `tools/check.ps1` passed on that commit: knowledge, web build, `go test ./...`, Playwright, replayed accuracy, replayed latency.
- [ ] If intake questions, cut-offs or the Jev client changed: `cmd/intake-eval -live` recorded and reviewed, and the baseline accepted by the owner.
- [ ] New migrations reviewed. They run at start inside one transaction, so a failing one stops the start and leaves the old schema intact.
- [ ] Last nightly backup succeeded (`last-success` under 26 h old).

**Deploy**
- [ ] Copy the tarball and `.sha256`, then `sha256sum -c`.
- [ ] Unpack to `/opt/sitewise-releases/`, note the previous target of `/opt/sitewise`.
- [ ] If units, Caddyfile or Postgres config changed: diff them against `/etc` and copy the changed ones, run `systemd-analyze verify` and `caddy validate`.
- [ ] `ln -sfn <new release> /opt/sitewise && systemctl daemon-reload && systemctl restart sitewise`. Shutdown ends event streams and waits up to 20 s for filings. Unfinished filings resume on start, and clients reconnect from their cursor.

**Verify (within 5 minutes)**
- [ ] `journalctl -u sitewise -n 50` shows `listening` and no migration error.
- [ ] `curl -s https://sitewise.au/healthz` returns `ok` (or `degraded` for a known reason only).
- [ ] As owner, `/api/health` shows the database OK, circuit `closed`, and Jev reached within 30 s.
- [ ] Drop one known text-layer PDF into a test project. It files, and `/api/speed` shows `whole_intake` within budget once 20 samples exist.

**Roll back** if any verify line fails: point `/opt/sitewise` back at the
previous release and restart. If the new release applied a migration that the
old binary cannot read, restore instead (section 5) to the time before the
deploy. A migration must therefore never drop or rename a column that the
previous release reads.

## 7. Release gates before the first invitation

| Gate | State on 2026-10-01 |
|---|---|
| Clean VPS built with section 1 | not started |
| `tools/check.ps1` passes on the release commit | fails. On the Task 12 tree: knowledge, `go vet` and `go test ./...` pass. Replayed accuracy fails only for want of an accepted baseline. The replayed bench fails extraction (p50 133 / p90 614 ms), rules p90 (4 ms) and Jev p90 (1,200 ms). It passes `whole_intake` (419 / 1,237 ms) and every API path, including `health_speed` (0.5 / 1.1 ms). Web build and Playwright were not rerun |
| Service sandbox verified under systemd on the VPS | not run (section 1 step 9) |
| Live latency gate on the VPS: `intake-bench -live -target <vps>` (copy the private corpus over temporarily and delete it after) | not run. The dev-host replay breaks the extraction, rules and list budgets (commit 5ac5e4d) |
| Intake cut-offs fitted and baseline accepted by owner | not done: all thresholds unknown, so every Jev answer stays blank (5ac5e4d) |
| Background Jev worker started by `serve` | not done, so health degrades after 15 min of filings |
| Mail relay delivers an invite end to end | not done |
| Off-site backup running, `last-success` fresh | not started |
| Restore rehearsal on an isolated VPS (section 5) | not run; local rehearsal passed |
