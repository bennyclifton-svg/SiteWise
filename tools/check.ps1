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

function Require-Command([string]$Name) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        Write-Error "$Name is required and was not found"
        exit 1
    }
}

Require-Command python
Require-Command go

# Dedicated test database. Never point this at any other database.
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'

& python tools/check_knowledge.py --strict
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

& go run ./cmd/sitewise gate -budgets bench/budgets.json -samples bench/samples.json
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
