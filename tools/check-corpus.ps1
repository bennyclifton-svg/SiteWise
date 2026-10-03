# Offline release check for a frozen corpus sample. Never makes live calls.
param(
    [Parameter(Mandatory=$true)][string]$Gold,
    [Parameter(Mandatory=$true)][string]$Recordings,
    [Parameter(Mandatory=$true)][string]$AppResults,
    [Parameter(Mandatory=$true)][string]$Out,
    [int]$PerFolder = 2,
    [string]$Data = 'data/intake'
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$env:PATH = (Join-Path $Root '.tools\go\bin') + [IO.Path]::PathSeparator + $env:PATH
$env:GOCACHE = Join-Path $Root '.tools\go-cache'
$env:GOPATH = Join-Path $Root '.tools\go-path'
$env:GOTOOLCHAIN = 'local'
foreach ($inputFile in @($Gold, $Recordings, $AppResults)) {
    if (-not (Test-Path -LiteralPath $inputFile -PathType Leaf)) { throw "Missing evidence: $inputFile" }
}
# Run both checks so a failed draft gate does not hide persisted-app results.
& go run ./cmd/corpus-eval -mode replay -per-folder $PerFolder -data $Data -gold $Gold -recordings $Recordings -out (Join-Path $Out 'replay') -gate
$draftExit = $LASTEXITCODE
& go run ./cmd/corpus-eval -mode app-score -per-folder $PerFolder -gold $Gold -app-results $AppResults -out (Join-Path $Out 'app') -gate
$appExit = $LASTEXITCODE
if ($draftExit -ne 0 -or $appExit -ne 0) { exit 1 }
