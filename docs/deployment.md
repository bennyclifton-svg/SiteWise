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

Live since 8 Nov 2026 (first release `79a8d5f`, fresh empty database, no
local data copied). Current release: `76bb126`, deployed 16 Nov 2026; it ran
migrations 050 to 063. A dump taken just before that deploy is on the server
at `/root/sitewise-before-76bb126.dump`. Done: operations.md section 1 steps
1–5 and 8–10, OCR.

- The model key was added to `/etc/sitewise/sitewise.env` on 16 Nov 2026, so
  the tender pass and chat start on (see "Tender pass" below; chat uses the
  same key). Each org still switches the model on for itself in Account.
  The settings file as it was before is beside it, `sitewise.env.bak-20261116`.
- For about ten seconds after a restart `/healthz` answers 503, then ok.

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

## Tender pass (optional)

The tender pass lists the lines of a tender with one model call in the
background. It needs two things on the server, and is off without them.

- Settings in `/etc/sitewise/sitewise.env`: `SITEWISE_ANTHROPIC_HAIKU_KEY`
  (the provider's key) and, only to change the model, `SITEWISE_TENDER_MODEL`
  (default `claude-haiku-5-5`). Without the key, tenders are read the old way.
- Page images: the pass sends pictures of the pages it chose. They are made
  by the same Python the OCR helper uses (`/opt/sitewise-ocr`, pypdfium2,
  Pillow). Without that Python the pass cannot run and tenders are read the
  old way.

Having the key is not enough: each org switches the model on for itself in
Account. Nobody but an org's owner switches it on for that org.

To confirm the model is reachable: switch it on for the owner's org, drop a
tender on a package, and look for the line `llm call` in
`journalctl -u sitewise`. It gives the model, the time taken and the token
counts, never the key or the document.

`SITEWISE_TENDER_SKIP_JEV_PRICES=1` stops the old reading asking Jev for
the prices the pass reads. Leave it unset until the going-live checklist in
`docs/evidence/2026-11-14-tender-comparison-slice8.md` has been run.

## Project email (optional; off until set)

Each project has a private address (`p-` and 32 random characters at the
receiving domain); mail to it is filed, and SiteWise sends transmittals
out (`docs/plans/2026-11-14-email-intake.md`,
`docs/plans/2026-11-17-correspondence-transmittals.md`). It is off until
both settings below are set: the webhook answers 404 and no address shows.
**Run email intake's Task 0 first** (its "Task 0 findings" section lists
what to record); nothing has been tried against real Mailgun.

1. **DNS** at Crazy Domains: the MX records Mailgun's panel shows for the
   receiving domain, recommended `in.sitewise.au` (the root domain stays free
   for ordinary mailboxes).
2. **Mailgun:** spam filtering on for that domain; one route,
   `match_recipient(".*@in\.sitewise\.au")` with
   `forward("https://sitewise.au/api/mail/inbound/mime")` and `stop()`. The
   URL must end in `mime`: Mailgun then posts the raw message (`body-mime`),
   which is what SiteWise verifies the sender's DKIM signature over; without
   it every message is held as "Could not confirm the sender". Mail goes out
   from `SITEWISE_MAIL_FROM` as "SiteWise", with Reply-To the PM and the
   project address.
   - The webhook signing key signs only a timestamp and a token, not the
     message. **Use this Mailgun account's signing key for this webhook
     only**: no other service may receive posts signed with it.
   - **Caddy:** limit `/api/mail/inbound*` to Mailgun's published webhook
     source addresses. Read the current ranges from Mailgun's own
     documentation page on the day (they are not copied here, and must not
     be guessed), and add to `/etc/caddy/Caddyfile` inside the site block:
     ```
     @mailgun_not path /api/mail/inbound*
     @mailgun_not not remote_ip <ranges from Mailgun's page>
     respond @mailgun_not 404
     ```
     then `systemctl reload caddy`. The app still checks every signature.
3. **Settings** in `/etc/sitewise/sitewise.env`, both or neither (one without
   the other stops the server starting):
   - `SITEWISE_MAIL_IN_DOMAIN=in.sitewise.au`
   - `SITEWISE_MAILGUN_SIGNING_KEY=` the account's **HTTP webhook signing
     key** (Mailgun: Sending → Webhooks), not the API key. Nothing reads the
     API key. Never write the value anywhere else.
   Restart. Sending uses the existing `SITEWISE_SMTP_*` and
   `SITEWISE_MAIL_FROM`; without them Send says sending is not set up.
   `SITEWISE_MAIL_SEND_DAILY_CAP` (optional, default 2,000) caps every org's
   sends together in a day.
4. **Check:** from the owner's mailbox send a message with a PDF to a
   project's address (Copy beside "Email documents to"): it files. From an
   address not on the team, send another: it is held; Accept it. Look for
   `mail webhook org=… : filed` in `journalctl -u sitewise` (never the
   subject, the addresses or the key). Caddy's 210 MB body limit is above the
   webhook's 40 MB cap; nothing changes there.

## Known gaps

- A tarball built on Windows loses the executable bit, hence the `chmod`
  in step 4. Without it the service fails with "Permission denied".
- Speed measured from Singapore includes about 90 ms more network time per
  request from Australia than a Sydney server would.
