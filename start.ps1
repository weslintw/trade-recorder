# 打單紀錄器 - 一鍵啟動腳本
$ErrorActionPreference = "Stop"

$StartedProcesses = @()

# 定義需要清理的端口
$ServicePorts = @(8080, 5173, 5174, 9000, 9001)

function Stop-AllServices {
    Write-Host ""
    Write-Host "================================" -ForegroundColor Cyan
    Write-Host "   正在清理/關閉服務實例..." -ForegroundColor Yellow
    
    # 1. 根據記錄的 PID 關閉
    foreach ($procId in $StartedProcesses) {
        if ($procId -and (Get-Process -Id $procId -ErrorAction SilentlyContinue)) {
            Write-Host "停止記錄的進程 $procId..." -ForegroundColor Gray
            taskkill /F /T /PID $procId 2>$null | Out-Null
        }
    }
    
    # 2. 根據端口強制清理 (解決第二次啟動無法連線的核心問題)
    foreach ($port in $ServicePorts) {
        $conns = Get-NetTCPConnection -LocalPort $port -ErrorAction SilentlyContinue
        foreach ($conn in $conns) {
            if ($conn.OwningProcess -gt 0) {
                $pidToKill = $conn.OwningProcess
                Write-Host "端口 $port 被進程 $pidToKill 佔用，正在強制解除..." -ForegroundColor Gray
                taskkill /F /T /PID $pidToKill 2>$null | Out-Null
            }
        }
    }
    
    # 3. 額外確保特定名稱的進程也關閉
    Get-Process -Name "minio" -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    
    Write-Host "[OK] 清理完成" -ForegroundColor Green
    Write-Host "================================" -ForegroundColor Cyan
}

try {
    # 啟動前先做一次清理，確保環境乾淨
    Stop-AllServices

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

    # 啟動MinIO
    Write-Host "啟動MinIO..." -ForegroundColor Yellow
    $p = Start-Process -FilePath $minioExe -ArgumentList "server",$minioData,"--console-address",":9001" -WindowStyle Hidden -PassThru
    $StartedProcesses += $p.Id
    Start-Sleep -Seconds 2
    Write-Host "[OK] MinIO已啟動 (http://localhost:9001)" -ForegroundColor Green
    Write-Host ""

    # 啟動後端
    Write-Host "[2/3] 啟動後端..." -ForegroundColor Yellow
    $backendPath = Join-Path $ProjectRoot "backend"
    Set-Location $backendPath

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
    $p = Start-Process powershell -ArgumentList "-NoExit", "-Command", $startBackend -PassThru
    $StartedProcesses += $p.Id
    
    # 等待後端就緒 (最多等待 10 秒)
    $retries = 0
    while (!(Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue) -and $retries -lt 20) {
        Start-Sleep -Milliseconds 500
        $retries++
    }
    Write-Host "[OK] 後端已啟動 (http://localhost:8080)" -ForegroundColor Green
    Set-Location $ProjectRoot
    Write-Host ""

    # 啟動前端
    Write-Host "[3/3] 啟動前端..." -ForegroundColor Yellow
    $frontendPath = Join-Path $ProjectRoot "frontend"
    Set-Location $frontendPath

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
    $p = Start-Process powershell -ArgumentList "-NoExit", "-Command", $startFrontend -PassThru
    $StartedProcesses += $p.Id
    
    # 等待前端就緒
    $retries = 0
    $frontendUrl = "http://localhost:5173"
    while (!(Get-NetTCPConnection -LocalPort 5173 -ErrorAction SilentlyContinue) -and $retries -lt 20) {
        # 同時檢查 5174
        if (Get-NetTCPConnection -LocalPort 5174 -ErrorAction SilentlyContinue) {
            $frontendUrl = "http://localhost:5174"
            break
        }
        Start-Sleep -Milliseconds 500
        $retries++
    }
    
    Write-Host "[OK] 前端已啟動 ($frontendUrl)" -ForegroundColor Green
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
    
    Write-Host "已開啟瀏覽器..." -ForegroundColor Gray
    Start-Process $frontendUrl
    
    Write-Host "--------------------------------" -ForegroundColor Gray
    Write-Host " 提示：按下任意鍵將 停止所有服務 並 退出視窗 " -ForegroundColor DarkCyan
    Write-Host "--------------------------------" -ForegroundColor Gray
    
    # 保持腳本運行直到使用者按鍵
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

} finally {
    Stop-AllServices
}


