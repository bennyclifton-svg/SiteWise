# Deployment: the live server and how to redeploy

Short runbook for this server. The full manual, with the reasons behind each
step, is `docs/operations.md`; this note does not repeat it. Never write a
secret in this file or anywhere in the repo: the repository is public.

## The server

| | |
|---|---|
| Host | BinaryLane, 2 vCPU / 4 GB / 60 GB, A$19.60 a month |
| Region | Singapore (Sydney had no capacity on 7 Nov 2026). Move to Sydney before the first paying client: the Privacy Policy says Sydney |
| Address | `sitewise.au` → `103.100.36.246` (A records for `@` and `www` at Crazy Domains, TTL 300) |
| OS | Ubuntu 24.04 LTS, clock on Australia/Sydney. Do not run `do-release-upgrade` |
| Login | `ssh root@103.100.36.246`, key only (owner's `~/.ssh/id_ed25519`); passwords are off in `/etc/ssh/sshd_config.d/00-sitewise.conf` |
| Firewall | ufw: in 22, 80, 443 only |
| Installed | PostgreSQL 17, pgBackRest, Caddy, rclone, Tesseract 5 + English, OCR Python venv at `/opt/sitewise-ocr` (pypdfium2, Pillow) |
| App user | `sitewise` (system, no shell); directories from operations.md section 1 step 3 |
| Mail | Mailgun already verified for `sitewise.au` (MX, SPF, DKIM in DNS). Use `smtp.mailgun.org:587` |
| Secrets | `/etc/sitewise/sitewise.env`, root only, mode 0600. Template: `deploy/sitewise.env.example` |

Paths: releases unpack to `/opt/sitewise-releases/<name>`; `/opt/sitewise` is
a symlink to the live one; files live in `/var/lib/sitewise/files`; logs via
`journalctl -u sitewise` and `/var/log/caddy/sitewise.log`.

## State of the live server

Live since 8 Nov 2026 with commit `79a8d5f`, fresh empty database (no local
data was copied). Done: operations.md section 1 steps 1–5 and 8–10, OCR.

- Settings: `/etc/sitewise/sitewise.env` holds production values. Mail is
  Mailgun (`smtp.mailgun.org:587`, user `app@sitewise.au`, from
  `no-reply@sitewise.au`). Stripe is in **test mode** (sandbox keys, GST
  rate `txr_1UZSZEBMfqwV8iUmaIZnaEci`, its own dashboard webhook to
  `https://sitewise.au/api/stripe/webhook`). Switching to live mode means
  new live keys, prices, tax rate and webhook in that file.
- Profile readings show under provisional thresholds, as on the dev server
  (owner choice 8 Nov 2026): `/etc/systemd/system/sitewise.service.d/provisional.conf`
  adds `-profile-provisional`. It lives in `/etc`, so redeploys keep it.
  Delete it once the thresholds are owner-approved.
- Caddy: `deploy/Caddyfile` minus its global `email` block (Let's Encrypt
  no longer sends expiry mail). `/var/log/caddy` must be owned by `caddy`.
- `sitewise-backup.timer` is **not enabled**, and
  `/etc/postgresql/17/main/conf.d/zz-until-backups.conf` sets
  `archive_mode = off`, because off-site storage does not exist yet. Delete
  that file and enable the timer when step 6 is done.

Before the first client: off-site backups (step 6, a bucket in an
Australian region), the move to Sydney, Stripe live mode, and the release
gates in operations.md section 7.

## Redeploy (every update)

Run from Git Bash at the repo root on the owner's PC. Migrations run by
themselves when the app starts.

1. **Commit.** A release is built only from a clean, committed tree.
   Uncommitted work is not deployed.
2. **Check.** `tools/check.ps1` passes, or say which checks were skipped and
   why.
3. **Build** in a separate clean checkout. `package.sh` runs `npm ci`, which
   deletes `web/node_modules` first and fails half-way (EPERM) if a dev or
   Vite server is holding it, breaking the main checkout. A worktree avoids
   that.
   ```sh
   git worktree add --detach ../sitewise-release HEAD   # or: git -C ../sitewise-release checkout --detach main
   cd ../sitewise-release
   PATH="/d/AI Projects/sitewise/.tools/go/bin:$PATH" GOTOOLCHAIN=local deploy/package.sh
   cd dist; name=sitewise-<commit>-linux-amd64          # from the printed file name
   ```
4. **Copy and switch.**
   ```sh
   scp $name.tar.gz $name.tar.gz.sha256 root@103.100.36.246:/root/
   ssh root@103.100.36.246 "set -e; cd /root; sha256sum -c $name.tar.gz.sha256;
     install -d /opt/sitewise-releases;
     tar -C /opt/sitewise-releases -xzf $name.tar.gz;
     chmod 0755 /opt/sitewise-releases/$name/bin/sitewise /opt/sitewise-releases/$name/deploy/*.sh;
     readlink /opt/sitewise > /root/previous-release || true;
     ln -sfn /opt/sitewise-releases/$name /opt/sitewise;
     systemctl daemon-reload; systemctl restart sitewise"
   ```
   If `deploy/sitewise.service`, `deploy/Caddyfile` or the Postgres config
   changed in this release, copy the changed file to `/etc` first
   (operations.md section 6).
5. **Verify** within five minutes:
   ```sh
   ssh root@103.100.36.246 "journalctl -u sitewise -n 30 --no-pager"   # 'listening', no migration error
   curl -s https://sitewise.au/healthz                                 # ok
   ```
   Then open https://sitewise.au, sign in, and drop one PDF.
6. **Roll back** if a check fails:
   ```sh
   ssh root@103.100.36.246 'ln -sfn "$(cat /root/previous-release)" /opt/sitewise && systemctl restart sitewise'
   ```
   If the new release ran a migration the old one cannot read, restore
   instead (operations.md section 5).

Keep the last three releases in `/opt/sitewise-releases` and delete older
ones; the disk is 60 GB.

## Known gaps

- A tarball built on Windows loses the executable bit, hence the `chmod`
  in step 4. Without it the service fails with "Permission denied".
- Speed measured from Singapore includes about 90 ms more network time per
  request from Australia than a Sydney server would.
