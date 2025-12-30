# Trade Recorder - Packaging Script (Multi-Platform)
$ErrorActionPreference = "Stop"

$ProjectRoot = $PSScriptRoot
$ReleaseRoot = Join-Path $ProjectRoot "release"
$WinRelease = Join-Path $ReleaseRoot "windows"
$MacRelease = Join-Path $ReleaseRoot "macos"

Write-Host "================================" -ForegroundColor Cyan
Write-Host "   Trade Recorder - Multi-Platform Build" -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan

# 1. Setup Release Folders
Write-Host "Stopping existing processes if any..." -ForegroundColor Gray
Stop-Process -Name "backend" -ErrorAction SilentlyContinue
Stop-Process -Name "minio" -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

if (Test-Path $ReleaseRoot) {
    Write-Host "Cleaning existing release folder..." -ForegroundColor Gray
    Remove-Item -Path $ReleaseRoot -Recurse -Force
}
New-Item -ItemType Directory -Path $WinRelease | Out-Null
New-Item -ItemType Directory -Path $MacRelease | Out-Null
New-Item -ItemType Directory -Path (Join-Path $WinRelease "minio-data") | Out-Null
New-Item -ItemType Directory -Path (Join-Path $MacRelease "minio-data") | Out-Null

# 2. Build Frontend
Write-Host "`n[1/4] Building Shared Frontend..." -ForegroundColor Yellow
Set-Location (Join-Path $ProjectRoot "frontend")
$env:NODE_OPTIONS = "--max-old-space-size=4096"

$TempDist = Join-Path $ReleaseRoot "temp_dist_frontend"
if (Get-Command pnpm -ErrorAction SilentlyContinue) {
    & pnpm install
    & npx vite build --outDir $TempDist --emptyOutDir
} else {
    & npm install
    & npx vite build --outDir $TempDist --emptyOutDir
}

if ($LASTEXITCODE -ne 0) {
    Write-Host "[Error] Frontend build failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}

# Copy to both release folders
Copy-Item -Path $TempDist -Destination (Join-Path $WinRelease "dist") -Recurse
Copy-Item -Path $TempDist -Destination (Join-Path $MacRelease "dist") -Recurse
Write-Host "[OK] Frontend built and copied to both platforms." -ForegroundColor Green

# 3. Build Backend
Write-Host "`n[2/4] Building Backend for Windows & Mac..." -ForegroundColor Yellow
Set-Location (Join-Path $ProjectRoot "backend")

# Disable CGO for easiest cross-compilation (since we use modernc.org/sqlite)
$env:CGO_ENABLED="0"

# Windows Build
Write-Host "Compiling for Windows (amd64)..." -ForegroundColor Gray
$env:GOOS="windows"
$env:GOARCH="amd64"
& go build -o (Join-Path $WinRelease "backend.exe") cmd/main.go

# MacOS Build (Apple Silicon - Most common now)
Write-Host "Compiling for MacOS (arm64 - Apple Silicon)..." -ForegroundColor Gray
$env:GOOS="darwin"
$env:GOARCH="arm64"
& go build -o (Join-Path $MacRelease "backend_mac") cmd/main.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "[Error] Backend build failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[OK] Backends built." -ForegroundColor Green

# 4. Copy Dependencies & Setup .env
Write-Host "`n[3/4] Copying dependencies..." -ForegroundColor Yellow
Set-Location $ProjectRoot

# Windows specific MinIO
if (Test-Path "minio.exe") {
    Copy-Item -Path "minio.exe" -Destination $WinRelease
}

# Setup .env for both
$envFile = if (Test-Path "backend\.env") { "backend\.env" } else { "backend\.env.example" }
Copy-Item -Path $envFile -Destination "$WinRelease\.env"
Copy-Item -Path $envFile -Destination "$MacRelease\.env"

# 5. Create Startup Scripts
Write-Host "`n[4/4] Creating startup scripts..." -ForegroundColor Yellow

# Windows Batch
$WinBat = @'
@echo off
setlocal
cd /d "%~dp0"
echo ========================================
echo   Trade Recorder - Starting (Windows)
echo ========================================
if not exist "minio.exe" (
    echo [!] minio.exe not found, downloading...
    powershell -Command "Invoke-WebRequest -Uri 'https://dl.min.io/server/minio/release/windows-amd64/minio.exe' -OutFile 'minio.exe'"
)
if not exist "minio-data" mkdir minio-data
echo [+] Starting Image Server...
start /b minio.exe server minio-data --console-address :9001
timeout /t 3 /nobreak > nul
echo [+] Starting Main Application...
start http://localhost:8080
backend.exe
pause
'@
$WinBat | Out-File -FilePath "$WinRelease\START_APP.bat" -Encoding utf8

# MacOS Shell Script
$MacSh = @'
#!/bin/bash
cd "$(dirname "$0")"
echo "========================================"
echo "   Trade Recorder - Starting (MacOS)"
echo "========================================"

# Check for MinIO
if [ ! -f "./minio" ]; then
    echo "[!] minio not found, downloading..."
    curl -O https://dl.min.io/server/minio/release/darwin-arm64/minio
    chmod +x minio
fi

mkdir -p minio-data

echo "[+] Starting Image Server..."
./minio server minio-data --console-address :9001 &

echo "[+] Waiting for services..."
sleep 3

echo "[+] Starting Main Application..."
open http://localhost:8080
chmod +x backend_mac
./backend_mac
'@
$MacSh | Out-File -FilePath "$MacRelease/start.sh" -Encoding utf8 # Note: PowerShell might add BOM, but bash usually handles it. Better to save as ASCII/UTF8 without BOM if possible, but Mac is flexible.

Write-Host "`n================================" -ForegroundColor Green
Write-Host "   Multi-platform Build Complete!" -ForegroundColor Green
Write-Host "   Windows: release\windows" -ForegroundColor White
Write-Host "   MacOS:   release\macos" -ForegroundColor White
Write-Host "================================" -ForegroundColor Green
Set-Location $ProjectRoot
