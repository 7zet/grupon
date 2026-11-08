<#
.SYNOPSIS
  Starts the whole Grupon demo on this PC: database check, migrations, demo data, API and web (production mode).
.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts\demo.ps1
  powershell -ExecutionPolicy Bypass -File scripts\demo.ps1 -Reset      # wipe and re-seed the demo data
.NOTES
  Settings come from .env.local (copy .env.local.example). Ctrl+C stops everything.
  Needs: PostgreSQL 16 (data in C:\dev\grupon-pgdata), Go, Node with Corepack. See README.md.
#>
param(
  [switch]$Reset,      # re-seed the demo data (TRUNCATE + rebuild)
  [switch]$SkipBuild   # reuse the previous web build
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Say($text) { Write-Host "==> $text" -ForegroundColor Cyan }

# 1. Settings -------------------------------------------------------------------------------------
$envFile = Join-Path $root '.env.local'
if (-not (Test-Path $envFile)) {
  Copy-Item (Join-Path $root '.env.local.example') $envFile
  Say 'Created .env.local from .env.local.example (edit it if you need other values).'
}
foreach ($line in Get-Content $envFile) {
  if ($line -match '^\s*([A-Z][A-Z0-9_]*)=(.*)$') {
    $value = ($Matches[2] -replace '\s+#.*$', '').Trim()
    Set-Item -Path "Env:$($Matches[1])" -Value $value
  }
}
$env:DEMO_MODE = 'true'
$env:RUN_WORKER_IN_API = 'true'
$env:APP_ENV = 'development'
$env:MIGRATIONS_DIR = Join-Path $root 'services/api/migrations'
$siteUrl = if ($env:SITE_URL) { $env:SITE_URL.TrimEnd('/') } else { 'http://localhost:3000' }
$webPort = if ($env:WEB_PORT) { $env:WEB_PORT } else { '3000' }
$pgBin = if ($env:PGBIN) { $env:PGBIN } else { 'C:\Program Files\PostgreSQL\16\bin' }
$pgData = if ($env:PGDATA) { $env:PGDATA } else { 'C:\dev\grupon-pgdata' }

# 2. PostgreSQL -----------------------------------------------------------------------------------
Say 'Checking PostgreSQL (port 5433)'
& "$pgBin\pg_isready.exe" -h localhost -p 5433 | Out-Null
if ($LASTEXITCODE -ne 0) {
  Say "PostgreSQL is not running. Starting it from $pgData"
  & "$pgBin\pg_ctl.exe" -D $pgData -l 'C:\dev\grupon-pgdata.log' -o '-p 5433' -w start | Out-Null
  & "$pgBin\pg_isready.exe" -h localhost -p 5433 | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'PostgreSQL did not start. See docs/ENVIRONMENT.md (local database) and C:\dev\grupon-pgdata.log.' }
}

# 3. Migrations + demo data ----------------------------------------------------------------------
$env:PGPASSWORD = 'app_dev_password'
$deals = & "$pgBin\psql.exe" -h localhost -p 5433 -U app -d app -At -c 'SELECT count(*) FROM deals' 2>$null
$seeded = ($LASTEXITCODE -eq 0) -and ([int]$deals -gt 0)
if ($Reset -or -not $seeded) {
  Say 'Running migrations and loading demo data (bootstrap)'
  go run ./services/api/cmd/bootstrap
  if ($LASTEXITCODE -ne 0) { throw 'bootstrap failed' }
  Remove-Item -Recurse -Force 'apps\web\.next\cache\fetch-cache' -ErrorAction SilentlyContinue
} else {
  Say "Demo data present ($deals deals). Applying any new migrations only."
  go run ./services/api/cmd/migrate
  if ($LASTEXITCODE -ne 0) { throw 'migrate failed' }
}

$children = @()
try {
  # 4. API ----------------------------------------------------------------------------------------
  Say 'Building and starting the API (with background worker)'
  New-Item -ItemType Directory -Force -Path bin | Out-Null
  go build -o bin\grupon-api.exe ./services/api/cmd/api
  if ($LASTEXITCODE -ne 0) { throw 'API build failed' }
  $children += Start-Process -FilePath (Join-Path $root 'bin\grupon-api.exe') -NoNewWindow -PassThru
  for ($i = 0; $i -lt 60; $i++) {
    try { if ((Invoke-WebRequest -UseBasicParsing http://localhost:8080/healthz -TimeoutSec 2).StatusCode -eq 200) { break } } catch { Start-Sleep -Seconds 1 }
  }

  # 5. Web (production build; the API must be up because public pages are pre-rendered) ----------
  Set-Location (Join-Path $root 'apps\web')
  if (-not $SkipBuild) {
    Say 'Building the web app (next build)'
    corepack pnpm exec next build
    if ($LASTEXITCODE -ne 0) { throw 'web build failed' }
  }
  Say 'Starting the web app (next start)'
  $children += Start-Process -FilePath 'corepack' -ArgumentList 'pnpm', 'exec', 'next', 'start', '--port', $webPort -NoNewWindow -PassThru
  Set-Location $root
  for ($i = 0; $i -lt 60; $i++) {
    try { if ((Invoke-WebRequest -UseBasicParsing "http://localhost:$webPort/uz" -TimeoutSec 5).StatusCode -eq 200) { break } } catch { Start-Sleep -Seconds 1 }
  }

  # 6. Summary --------------------------------------------------------------------------------------
  Write-Host ''
  Write-Host 'Grupon demo is running' -ForegroundColor Green
  Write-Host "  Site:   $siteUrl/uz   (also /ru, /en)"
  Write-Host '  API:    http://localhost:8080/api/v1'
  Write-Host ''
  Write-Host 'Demo accounts (the sign-in code is shown on the page):'
  Write-Host '  Buyer            +998900000001'
  Write-Host '  Lotus Spa owner  +998900000002   manager +998910000010   cashier +998900000003'
  Write-Host '  Any business MM  owner +998912MM100  manager +998912MM200  cashier +998912MM300   (MM = 01..30)'
  Write-Host '  Admin            +998900000004   (needs a 2FA code, see docs/DEMO.md)'
  Write-Host ''
  Write-Host 'Press Ctrl+C to stop everything.' -ForegroundColor Yellow
  while ($true) {
    Start-Sleep -Seconds 2
    foreach ($child in $children) { if ($child.HasExited) { throw "A process stopped unexpectedly (exit code $($child.ExitCode))." } }
  }
} finally {
  Set-Location $root
  foreach ($child in $children) {
    if ($child -and -not $child.HasExited) { & taskkill /PID $child.Id /T /F 2>&1 | Out-Null }
  }
  Write-Host 'Stopped.'
}
