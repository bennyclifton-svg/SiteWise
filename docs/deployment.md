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

## First deploy: what is left

Done on 7 Nov 2026: operations.md section 1 steps 1–3 (base, packages,
user and directories) and OCR.

Still to do, in order (operations.md section 1):

1. Step 4, first release (below).
2. Step 5, database: socket only, `createuser` and `createdb`.
3. Step 8, secrets: fill `/etc/sitewise/sitewise.env` on the server. The
   session secret is new (`openssl rand -hex 32`); the Jev key and the Stripe
   test keys are the owner's; the Mailgun SMTP login comes from Mailgun.
   Stripe on the server needs its own webhook endpoint
   `https://sitewise.au/api/stripe/webhook` made in the Stripe dashboard
   (operations.md section 8 step 7), not `stripe listen`.
4. Step 9, systemd units; step 10, Caddyfile with the owner's email for
   certificate notices.
5. Sign up as the owner and run the Stripe test (operations.md section 8
   step 9).

Can wait while only the owner tests, but not past the first client:
off-site backups (step 6, a bucket in an Australian region), the move to
Sydney, and the release gates in operations.md section 7.

## Redeploy (every update)

Run from Git Bash at the repo root on the owner's PC. Migrations run by
themselves when the app starts.

1. **Commit.** A release is built only from a clean, committed tree.
   Uncommitted work is not deployed.
2. **Check.** `tools/check.ps1` passes, or say which checks were skipped and
   why.
3. **Build.**
   ```sh
   deploy/package.sh        # prints dist/sitewise-<commit>-linux-amd64.tar.gz
   name=sitewise-<commit>-linux-amd64
   ```
4. **Copy and switch.**
   ```sh
   scp dist/$name.tar.gz dist/$name.tar.gz.sha256 root@103.100.36.246:/root/
   ssh root@103.100.36.246 "set -e; cd /root; sha256sum -c $name.tar.gz.sha256;
     install -d /opt/sitewise-releases;
     tar -C /opt/sitewise-releases -xzf $name.tar.gz;
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

- `deploy/package.sh` has not yet been run on the owner's Windows PC; the
  first build may need Go and Node on the Git Bash path (`.tools/go/bin`).
- Speed measured from Singapore includes about 90 ms more network time per
  request from Australia than a Sydney server would.
