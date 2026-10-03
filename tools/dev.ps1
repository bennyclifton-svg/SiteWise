# Run SiteWise locally and open it in the browser, signed in.
#
#   & "D:\AI Projects\sitewise\tools\dev.ps1"
#
# The TypeSafe key comes from .env at the repo root (git-ignored):
#   SITEWISE_JEV_API_KEY=<your TypeSafe key>     (TYPESAFE_API_KEY= also works)
# A key already set in the shell wins over .env.
#
# http://127.0.0.1:8080/dev/login signs you in as the local owner; bookmark it.
# It exists only on this machine: serve refuses -dev-login on any other address
# or in production. -Build rebuilds the web UI first.
#
# Data lives in the sitewise_dev database and .tools/dev-files, never in the
# sitewise_test database the tests reset.
param(
    [switch]$Build,
    [string]$Addr = '127.0.0.1:8080'
)
# Continue, not Stop: Windows PowerShell 5.1 turns a native command's stderr
# into a terminating error. Every native call below checks its exit code.
$ErrorActionPreference = 'Continue'
Set-StrictMode -Version Latest

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$Tools = Join-Path $Root '.tools'

$env:PATH = (Join-Path $Tools 'go\bin') + [IO.Path]::PathSeparator + $env:PATH
$env:GOCACHE = Join-Path $Tools 'go-cache'
$env:GOPATH = Join-Path $Tools 'go-path'
$env:GOTOOLCHAIN = 'local'

# .env holds KEY=VALUE lines. A variable already set in the shell is kept.
$EnvFile = Join-Path $Root '.env'
if (Test-Path $EnvFile) {
    foreach ($line in Get-Content $EnvFile) {
        $line = $line.Trim()
        if ($line -eq '' -or $line.StartsWith('#') -or -not $line.Contains('=')) { continue }
        $name, $value = $line.Split('=', 2)
        $name = $name.Trim()
        $value = $value.Trim().Trim('"').Trim("'")
        if (-not [Environment]::GetEnvironmentVariable($name)) {
            [Environment]::SetEnvironmentVariable($name, $value)
        }
    }
}
if (-not $env:SITEWISE_JEV_API_KEY -and $env:TYPESAFE_API_KEY) {
    $env:SITEWISE_JEV_API_KEY = $env:TYPESAFE_API_KEY
}
if (-not $env:SITEWISE_JEV_API_KEY) {
    Write-Host "No TypeSafe key. Add SITEWISE_JEV_API_KEY=<key> to $EnvFile, or set `$env:SITEWISE_JEV_API_KEY."
    exit 1
}

$Port = [int]($Addr.Split(':')[-1])
$Exe = Join-Path $Tools 'sitewise-dev.exe'

# A previous run of this script may still hold the port; stop only that.
$holder = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue | Select-Object -First 1
if ($holder) {
    $proc = Get-Process -Id $holder.OwningProcess -ErrorAction SilentlyContinue
    if ($proc -and $proc.Path -eq $Exe) {
        Write-Host 'Stopping the previous SiteWise run...'
        Stop-Process -Id $proc.Id -Force
        Start-Sleep -Milliseconds 500
    } else {
        Write-Host "Port $Port is in use by another program (PID $($holder.OwningProcess)). Close it or use -Addr 127.0.0.1:<other port>."
        exit 1
    }
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

# A built executable, not go run: Ctrl+C then stops the server itself rather
# than leaving a child process holding the port.
Write-Host 'Building SiteWise...'
& go build -o $Exe ./cmd/sitewise
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

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

$Login = "http://$Addr/dev/login"
# Development reads only documents uploaded from now (the dev database holds a
# large unread corpus backlog) and applies the provisional profile floors.
$server = Start-Process -FilePath $Exe -ArgumentList @('serve', '-addr', $Addr, '-dev-login', '-background-backlog=false', '-profile-provisional') -NoNewWindow -PassThru
try {
    $deadline = (Get-Date).AddSeconds(60)
    while (-not $server.HasExited -and (Get-Date) -lt $deadline) {
        if (Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue) { break }
        Start-Sleep -Milliseconds 250
    }
    if ($server.HasExited) {
        Write-Host 'SiteWise stopped during startup; see the messages above.'
        exit 1
    }
    Write-Host ''
    Write-Host "SiteWise is running. Opening $Login"
    Write-Host 'Bookmark that link: it signs you in. Press Ctrl+C here to stop.'
    if (-not $env:SITEWISE_NO_BROWSER) { Start-Process $Login }
    $server.WaitForExit()
} finally {
    if (-not $server.HasExited) { Stop-Process -Id $server.Id -Force }
}
