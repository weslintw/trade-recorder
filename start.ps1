# 打單紀錄器 - 一鍵啟動腳本
$ErrorActionPreference = "Stop"

Write-Host "================================" -ForegroundColor Cyan
Write-Host "   打單紀錄器 - 啟動中..." -ForegroundColor Cyan
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""

# 取得專案根目錄
$ProjectRoot = $PSScriptRoot
Set-Location $ProjectRoot

# 檢查MinIO
Write-Host "[1/3] 檢查MinIO..." -ForegroundColor Yellow
$minioExe = Join-Path $ProjectRoot "minio.exe"
if (!(Test-Path $minioExe)) {
    Write-Host "MinIO未安裝，正在下載..." -ForegroundColor Yellow
    try {
        Invoke-WebRequest -Uri "https://dl.min.io/server/minio/release/windows-amd64/minio.exe" -OutFile $minioExe
        Write-Host "[OK] MinIO下載完成" -ForegroundColor Green
    } catch {
        Write-Host "[錯誤] MinIO下載失敗: $_" -ForegroundColor Red
        Read-Host "按 Enter 鍵退出"
        exit 1
    }
}

# 建立MinIO資料目錄
$minioData = Join-Path $ProjectRoot "minio-data"
if (!(Test-Path $minioData)) {
    New-Item -ItemType Directory -Path $minioData | Out-Null
}

# 檢查MinIO是否已在運行
$minioProcess = Get-Process -Name "minio" -ErrorAction SilentlyContinue
if ($minioProcess) {
    Write-Host "[OK] MinIO已經在運行" -ForegroundColor Green
} else {
    # 啟動MinIO
    Write-Host "啟動MinIO..." -ForegroundColor Yellow
    Start-Process -FilePath $minioExe -ArgumentList "server",$minioData,"--console-address",":9001" -WindowStyle Hidden
    Start-Sleep -Seconds 3
    Write-Host "[OK] MinIO已啟動 (http://localhost:9001)" -ForegroundColor Green
}
Write-Host ""

# 啟動後端
Write-Host "[2/3] 啟動後端..." -ForegroundColor Yellow
$backendPath = Join-Path $ProjectRoot "backend"
Set-Location $backendPath

# 檢查 8080 端口是否被佔用
$port8080 = Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue
if ($port8080) {
    Write-Host "檢測到 8080 端口已被佔用" -ForegroundColor Yellow
    Write-Host "[OK] 後端已經在運行" -ForegroundColor Green
} else {
    # 檢查是否需要下載依賴
    if (!(Test-Path "go.sum")) {
        Write-Host "下載Go模組..." -ForegroundColor Yellow
        go mod download
    }

    # 複製環境變數檔案
    $envFile = Join-Path $backendPath ".env"
    $envExample = Join-Path $backendPath ".env.example"
    if (!(Test-Path $envFile)) {
        if (Test-Path $envExample) {
            Copy-Item $envExample $envFile
            Write-Host "[OK] 環境變數檔案已建立" -ForegroundColor Green
        }
    }

    # 啟動後端
    Write-Host "啟動後端服務..." -ForegroundColor Yellow
    $startBackend = "Set-Location '$backendPath'; go run cmd/main.go"
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $startBackend
    Start-Sleep -Seconds 3
    Write-Host "[OK] 後端已啟動 (http://localhost:8080)" -ForegroundColor Green
}
Set-Location $ProjectRoot
Write-Host ""

# 啟動前端
Write-Host "[3/3] 啟動前端..." -ForegroundColor Yellow
$frontendPath = Join-Path $ProjectRoot "frontend"
Set-Location $frontendPath

# 檢查前端是否已在運行
$port5173 = Get-NetTCPConnection -LocalPort 5173 -ErrorAction SilentlyContinue
$port5174 = Get-NetTCPConnection -LocalPort 5174 -ErrorAction SilentlyContinue
if ($port5173 -or $port5174) {
    Write-Host "[OK] 前端已經在運行" -ForegroundColor Green
    if ($port5173) {
        $frontendUrl = "http://localhost:5173"
    } else {
        $frontendUrl = "http://localhost:5174"
    }
} else {
    # 檢查是否需要安裝依賴
    $nodeModules = Join-Path $frontendPath "node_modules"
    if (!(Test-Path $nodeModules)) {
        Write-Host "安裝前端套件（使用pnpm）..." -ForegroundColor Yellow
        try {
            pnpm install
        } catch {
            Write-Host "[警告] pnpm 安裝失敗，嘗試使用 npm..." -ForegroundColor Yellow
            npm install
        }
    }

    # 啟動前端
    Write-Host "啟動前端服務..." -ForegroundColor Yellow
    $startFrontend = "Set-Location '$frontendPath'; pnpm run dev"
    Start-Process powershell -ArgumentList "-NoExit", "-Command", $startFrontend
    Start-Sleep -Seconds 3
    Write-Host "[OK] 前端已啟動 (http://localhost:5173)" -ForegroundColor Green
    $frontendUrl = "http://localhost:5173"
}
Set-Location $ProjectRoot
Write-Host ""

Write-Host "================================" -ForegroundColor Cyan
Write-Host "   所有服務已啟動！" -ForegroundColor Green
Write-Host "================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "前端應用: $frontendUrl" -ForegroundColor White
Write-Host "後端API:  http://localhost:8080" -ForegroundColor White
Write-Host "MinIO:    http://localhost:9001 (minioadmin/minioadmin)" -ForegroundColor White
Write-Host ""
Write-Host "按 Enter 鍵開啟瀏覽器..." -ForegroundColor Yellow
Read-Host

Start-Process $frontendUrl
