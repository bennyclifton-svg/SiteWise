param([string]$Pdf = '')
$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root
$env:GOCACHE = Join-Path $Root '.tools/go-cache'
$env:GOPATH = Join-Path $Root '.tools/go-path'
$env:SITEWISE_TEST_DATABASE_URL = 'postgres://sitewise@127.0.0.1:5433/sitewise_test?sslmode=disable'
$env:TEMP = Join-Path $Root 'tmp'
$env:TMP = $env:TEMP
if (-not $env:SITEWISE_TESSERACT) { $env:SITEWISE_TESSERACT = 'C:\Program Files\Tesseract-OCR\tesseract.exe' }
if (-not $env:SITEWISE_TESSDATA) { $env:SITEWISE_TESSDATA = Join-Path $Root '.tools/tesseract/tessdata' }
if (-not $env:SITEWISE_OCR_PYTHON) { $env:SITEWISE_OCR_PYTHON = Join-Path $env:USERPROFILE '.cache/codex-runtimes/codex-primary-runtime/dependencies/python/python.exe' }
if (-not $Pdf) { $Pdf = Join-Path $Root 'testdata/identity/identity-page.pdf' }
$env:SITEWISE_OCR_TEST_PDF = (Resolve-Path -LiteralPath $Pdf).Path
$env:SITEWISE_VISUAL_GATE = '1'
try {
    & (Join-Path $Root '.tools/go/bin/go.exe') test ./internal/intake -run '^TestVisualIdentitySpeedGate$' -count=1 -timeout 10m -v
    if ($LASTEXITCODE -ne 0) { throw 'Visual identity gate failed' }
} finally { Remove-Item Env:SITEWISE_VISUAL_GATE -ErrorAction SilentlyContinue }
