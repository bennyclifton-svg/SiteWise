# Required installed-runtime gate; never silently skips missing OCR dependencies.
param([string]$Pdf = '', [string]$Output = '')
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$env:GOCACHE = Join-Path $Root '.tools/go-cache'
$env:GOPATH = Join-Path $Root '.tools/go-path'
$env:GOTOOLCHAIN = 'local'
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
if (-not $env:SITEWISE_TESSERACT) { $env:SITEWISE_TESSERACT = 'C:\Program Files\Tesseract-OCR\tesseract.exe' }
if (-not $env:SITEWISE_TESSDATA) { $env:SITEWISE_TESSDATA = Join-Path $Root '.tools/tesseract/tessdata' }
if (-not $env:SITEWISE_OCR_PYTHON) { $env:SITEWISE_OCR_PYTHON = Join-Path $env:USERPROFILE '.cache/codex-runtimes/codex-primary-runtime/dependencies/python/python.exe' }
if (-not $Pdf) { $Pdf = Join-Path $Root 'testdata/identity/ocr-drawing.pdf' }
$env:SITEWISE_OCR_TEST_PDF = (Resolve-Path -LiteralPath $Pdf).Path
$env:SITEWISE_OCR_TIMING_OUTPUT = $Output
$env:SITEWISE_OCR_GATE = '1'
try {
    & (Join-Path $Root '.tools/go/bin/go.exe') test ./internal/intake -run '^TestOCRFilingSpeedGate$' -count=1 -v
    if ($LASTEXITCODE -ne 0) { throw 'OCR gate failed' }
} finally {
    Remove-Item Env:SITEWISE_OCR_GATE -ErrorAction SilentlyContinue
}
