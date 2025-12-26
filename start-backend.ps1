# 後端啟動腳本

$projectRoot = "C:\Users\Wes\Documents\wes-projects\打單紀錄器"
$backendPath = Join-Path $projectRoot "backend"

Write-Host "後端專案路徑: $backendPath" -ForegroundColor Cyan

if (Test-Path $backendPath) {
    Set-Location $backendPath
    
    Write-Host "檢查環境變數檔案..." -ForegroundColor Yellow
    if (!(Test-Path ".env")) {
        if (Test-Path ".env.example") {
            Copy-Item ".env.example" ".env"
            Write-Host "[OK] 環境變數檔案已建立" -ForegroundColor Green
        }
    }
    
    Write-Host "啟動後端服務..." -ForegroundColor Green
    go run cmd/main.go
} else {
    Write-Host "錯誤：找不到 backend 目錄" -ForegroundColor Red
    Write-Host "預期路徑：$backendPath" -ForegroundColor Red
}

