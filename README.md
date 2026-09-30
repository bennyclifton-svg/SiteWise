# SiteWise

A fast construction-project app built on the physical building (systems, the
rules that govern them, and the interfaces between them), with Jev as its only
AI.

- Design: [docs/design/2026-09-29-foundation-design.md](docs/design/2026-09-29-foundation-design.md)
- Knowledge model: [knowledge/SCHEMA.md](knowledge/SCHEMA.md)
- Agent rules: [AGENTS.md](AGENTS.md)

Check the knowledge files: `python tools/check_knowledge.py`

## Run

The Go binary embeds the web build, so build the SPA first.

```powershell
npm --prefix web ci
npm --prefix web run build
go run ./cmd/sitewise bootstrap -org "My Org" -email me@example.com   # prints an invite token
go run ./cmd/sitewise serve -addr 127.0.0.1:8080
```

`serve` needs `SITEWISE_DATABASE_URL`, `SITEWISE_FILE_DIR`, `SITEWISE_SESSION_SECRET`,
`SITEWISE_JEV_API_KEY` and `SITEWISE_JEV_MODEL=jev-1.13.0`. Open
`http://127.0.0.1:8080/#token=<token>` to sign in. For UI work, run
`npm --prefix web run dev` with the Go server started with
`SITEWISE_PUBLIC_ORIGIN=http://localhost:5173`.

Full gate (knowledge, web build, Go tests, Playwright, latency): `tools/check.ps1`.
