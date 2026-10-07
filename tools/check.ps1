# Fail closed. A missing tool, a failed test, or a missing benchmark
# sample file exits nonzero. Nothing here treats a skip as success.
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

$env:PATH = (Join-Path $Root '.tools\go\bin') + [IO.Path]::PathSeparator + $env:PATH
$env:GOCACHE = Join-Path $Root '.tools\go-cache'
$env:GOPATH = Join-Path $Root '.tools\go-path'
$env:GOTOOLCHAIN = 'local'
# Keep test files beneath the checkout: restricted Windows runners may not
# permit helper processes to reopen files in the account's default temp root.
$TestTemp = Join-Path $Root 'tmp'
New-Item -ItemType Directory -Force $TestTemp | Out-Null
$env:TEMP = $TestTemp
$env:TMP = $TestTemp

function Require-Command([string]$Name) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Error "$Name is required and was not found"
        exit 1
    }
}

Require-Command python
Require-Command go
$Npm = if ($env:OS -eq 'Windows_NT') { 'npm.cmd' } else { 'npm' }
Require-Command $Npm

# Dedicated test database. Never point this at any other database.
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'

& python tools/check_knowledge.py --strict
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# The Go binary embeds web/dist, so the SPA is built before Go is tested.
& $Npm --prefix web ci
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& $Npm --prefix web run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Integration packages share the dedicated database and deterministic fixture
# IDs. Serialize packages so cleanup in one cannot remove another's fixtures.
& go test -p 1 ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& "$PSScriptRoot/check-ocr.ps1"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& $Npm --prefix web run test:e2e
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Accuracy: deterministic replay of the recorded live Jev run over the private
# corpus. A missing corpus, recording, sample floor, baseline or a regression
# fails here. Record with -live (SITEWISE_JEV_API_KEY) before the first run.
& go run ./cmd/intake-eval -manifest data/eval/intake/manifest.json -replay
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Profile regression gate uses private source excerpts and recorded Jev answers.
& go run ./cmd/profile-eval
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& go run ./cmd/profile-eval -cases data/eval/profile/private/hale-cases.json -recording data/eval/profile/private/hale-recording.json
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# Latency: every budgeted path through the real server, gated by
# bench/budgets.json. The release gate is the same command with -live on the
# intended VPS; replayed provider latency is a timing model, not evidence.
& go run ./cmd/intake-bench -manifest data/eval/intake/manifest.json -budgets bench/budgets.json
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
