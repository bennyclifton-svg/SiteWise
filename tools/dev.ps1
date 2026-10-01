# Run SiteWise locally against repo-local PostgreSQL 17 on port 5433.
#
#   $env:SITEWISE_JEV_API_KEY = '<your TypeSafe key>'
#   ./tools/dev.ps1            # first run prints a sign-in link
#   ./tools/dev.ps1 -Invite    # print a fresh sign-in link (new local org)
#   ./tools/dev.ps1 -Build     # rebuild the web UI first
#
# Data lives in the sitewise_dev database and .tools/dev-files, never in the
# sitewise_test database the tests reset. The key is read from the shell and
# never written to disk.
param(
    [switch]$Invite,
    [switch]$Build,
    [string]$Addr = '127.0.0.1:8080',
    [string]$Email = 'owner@sitewise.local'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$Tools = Join-Path $Root '.tools'

$env:PATH = (Join-Path $Tools 'go\bin') + [IO.Path]::PathSeparator + $env:PATH
$env:GOCACHE = Join-Path $Tools 'go-cache'
$env:GOPATH = Join-Path $Tools 'go-path'
$env:GOTOOLCHAIN = 'local'

if (-not $env:SITEWISE_JEV_API_KEY) {
    Write-Error "Set your TypeSafe key first:  `$env:SITEWISE_JEV_API_KEY = '<key>'"
    exit 1
}

# PostgreSQL: start it if it is not already running.
$PgBin = Join-Path $Tools 'pgsql-dist\pgsql\bin'
$PgData = Join-Path $Tools 'pgdata'
& (Join-Path $PgBin 'pg_ctl.exe') -D $PgData status *> $null
if ($LASTEXITCODE -ne 0) {
    Write-Host 'Starting PostgreSQL on port 5433...'
    & (Join-Path $PgBin 'pg_ctl.exe') -D $PgData -l (Join-Path $Tools 'pg.log') -o '-p 5433' -w start
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
$exists = & (Join-Path $PgBin 'psql.exe') -h 127.0.0.1 -p 5433 -U sitewise -d postgres -Atc "select 1 from pg_database where datname = 'sitewise_dev'"
if ($exists -ne '1') {
    Write-Host 'Creating the sitewise_dev database...'
    & (Join-Path $PgBin 'createdb.exe') -h 127.0.0.1 -p 5433 -U sitewise sitewise_dev
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

# The Go binary embeds web/dist.
if ($Build -or -not (Test-Path (Join-Path $Root 'web\dist\index.html'))) {
    & npm --prefix web ci
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    & npm --prefix web run build
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

# A stable local session secret, kept in the ignored .tools directory.
$SecretFile = Join-Path $Tools 'dev-session-secret'
if (-not (Test-Path $SecretFile)) {
    $bytes = New-Object byte[] 32
    # Create().GetBytes works in Windows PowerShell 5.1 as well as 7.
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
    Set-Content -NoNewline -Path $SecretFile -Value ([Convert]::ToBase64String($bytes))
}

$env:SITEWISE_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_dev?sslmode=disable'
$env:SITEWISE_FILE_DIR = Join-Path $Tools 'dev-files'
$env:SITEWISE_SESSION_SECRET = Get-Content -Raw $SecretFile
$env:SITEWISE_JEV_MODEL = 'jev-1.13.0'
$env:SITEWISE_ENV = 'development'
New-Item -ItemType Directory -Force $env:SITEWISE_FILE_DIR | Out-Null

$InvitedMarker = Join-Path $Tools 'dev-invited'
if ($Invite -or -not (Test-Path $InvitedMarker)) {
    $token = & go run ./cmd/sitewise bootstrap -org 'Local dev' -email $Email 2>$null
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Set-Content -Path $InvitedMarker -Value (Get-Date -Format o)
    Write-Host ''
    Write-Host 'Sign in once with this link (single use, valid 24 hours):'
    Write-Host "  http://$Addr/#token=$($token.Trim())"
    Write-Host ''
}

Write-Host "SiteWise on http://$Addr  (Ctrl+C to stop)"
& go run ./cmd/sitewise serve -addr $Addr
exit $LASTEXITCODE
