<#
.SYNOPSIS
  Helps share the running demo through an ngrok HTTPS tunnel. Installs nothing.
.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts\share.ps1
  powershell -ExecutionPolicy Bypass -File scripts\share.ps1 -Url https://abc123.ngrok-free.app
#>
param([string]$Url)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $root '.env.local'
if (-not (Test-Path $envFile)) { Copy-Item (Join-Path $root '.env.local.example') $envFile }

if (-not (Get-Command ngrok -ErrorAction SilentlyContinue)) {
  Write-Host 'ngrok is not installed. Install it yourself from https://ngrok.com/download and run: ngrok config add-authtoken <token>' -ForegroundColor Yellow
}
Write-Host ''
Write-Host '1. In a separate window start the tunnel to the web app:'
Write-Host '     ngrok http 3000' -ForegroundColor Green
if (-not $Url) { $Url = Read-Host '2. Paste the https URL that ngrok prints (https://....ngrok-free.app)' }
$Url = $Url.Trim().TrimEnd('/')
if ($Url -notmatch '^https://') { throw 'The URL must start with https://' }

function Set-EnvValue($name, $value) {
  $lines = @(Get-Content $envFile)
  $found = $false
  $lines = $lines | ForEach-Object { if ($_ -match "^$name=") { $found = $true; "$name=$value" } else { $_ } }
  if (-not $found) { $lines += "$name=$value" }
  Set-Content -Path $envFile -Value $lines
}
Set-EnvValue 'SITE_URL' $Url
Set-EnvValue 'ALLOWED_ORIGINS' "$Url,http://localhost:3000,http://127.0.0.1:3000"
Write-Host ''
Write-Host "Saved SITE_URL and ALLOWED_ORIGINS in .env.local for $Url" -ForegroundColor Green
Write-Host '3. Restart the demo so the API and the web app pick the new address up (Ctrl+C, then):'
Write-Host '     powershell -ExecutionPolicy Bypass -File scripts\demo.ps1' -ForegroundColor Green
Write-Host "4. Send people this link: $Url/uz"
Write-Host '   To go back to localhost, set SITE_URL=http://localhost:3000 in .env.local.'
