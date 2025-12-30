# 前端啟動腳本

$projectRoot = "C:\Users\Wes\Documents\wes-projects\打單紀錄器"
$frontendPath = Join-Path $projectRoot "frontend"

Write-Host "前端專案路徑: $frontendPath" -ForegroundColor Cyan

if (Test-Path $frontendPath) {
    Set-Location $frontendPath
    
    Write-Host "檢查 node_modules..." -ForegroundColor Yellow
    if (!(Test-Path "node_modules")) {
        Write-Host "安裝依賴中（使用 pnpm）..." -ForegroundColor Yellow
        pnpm install
    }
    
    Write-Host "啟動開發伺服器..." -ForegroundColor Green
    pnpm run dev --host 0.0.0.0
} else {
    Write-Host "錯誤：找不到 frontend 目錄" -ForegroundColor Red
    Write-Host "預期路徑：$frontendPath" -ForegroundColor Red
}

