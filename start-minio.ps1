# MinIO 啟動腳本

$projectRoot = "C:\Users\Wes\Documents\wes-projects\打單紀錄器"

Set-Location $projectRoot

Write-Host "檢查 MinIO..." -ForegroundColor Yellow

if (!(Test-Path "minio.exe")) {
    Write-Host "MinIO 未安裝，正在下載..." -ForegroundColor Yellow
    try {
        Invoke-WebRequest -Uri "https://dl.min.io/server/minio/release/windows-amd64/minio.exe" -OutFile "minio.exe"
        Write-Host "[OK] MinIO 下載完成" -ForegroundColor Green
    } catch {
        Write-Host "[錯誤] MinIO 下載失敗: $_" -ForegroundColor Red
        exit 1
    }
}

if (!(Test-Path "minio-data")) {
    New-Item -ItemType Directory -Path "minio-data" | Out-Null
}

Write-Host "啟動 MinIO..." -ForegroundColor Green
Write-Host "MinIO Console: http://localhost:9001" -ForegroundColor Cyan
Write-Host "帳號: minioadmin" -ForegroundColor White
Write-Host "密碼: minioadmin" -ForegroundColor White
Write-Host ""

.\minio.exe server .\minio-data --console-address ":9001"

