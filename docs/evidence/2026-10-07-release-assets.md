# Release asset correction — 7 October 2026

The post-publication acceptance audit found a concrete deployment defect:
`deploy/package.sh` shipped only intake data, while current startup also needs
the knowledge catalogue and three profile policies. The service used default
relative paths for those missing assets, so the packaged application could not
start correctly independently of the repository checkout.

Lane C. Outcome: the Linux package carries the exact runtime catalogue/policies
and its service resolves them from absolute release paths. No production host,
invitation, corpus transmission, dependency or application hot path changes.
The existing filing/edit budgets and final performance evidence remain applicable
to the unchanged application implementation; this is not VPS timing proof.

The script now copies `knowledge`, `data/profile` and the existing `data/intake`
to explicit `share` directories. It validates the actual staged assets through
all six runtime loaders before creating the archive. A full catalogue fingerprint
comparison detects omitted optional families that a successful load alone would
miss. Tests also reject missing policies, missing report templates, wrong policy
content and unexpected payload roots. Private evaluation corpora are excluded.
Staging replacement resolves and checks its target beneath the checkout's `dist`
directory and rejects symlink substitutions before any recursive removal.

Validation: all `cmd/sitewise` tests pass (1.981 s); the local package build passes
its staged-loader test. Inspection compares all 102 archived files byte-for-byte
against staging, verifies the archive checksum, and confirms ELF64 x86-64 with no
dynamic interpreter/dependency segment. The first trial was explicitly labelled dirty; the clean-commit result is below. This Windows-host
cross-build does not prove that systemd, PostgreSQL, Caddy or off-site backups run
on the intended Linux VPS. Those remain release gates requiring a target host.

## Clean-commit artifact

Commit: `b470fae3f8a83b2189267a2f176495ac053aca27`.

`deploy/package.sh` succeeds without ALLOW_DIRTY. Artifact: `dist/sitewise-b470fae3f8a8-linux-amd64.tar.gz`; SHA-256: `b4914a13ca51de534dbef8beb6614921eb14514501a90c4bcf8ab0ae059e631f`. All 102 archived files match staging; the binary is static ELF64 x86-64; the staged real-loader check passes. The artifact remains local and ignored; no host deployment occurred. Log: `tmp/m2-m3-package-clean.log`.
