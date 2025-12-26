# 前端快速重啟腳本

Write-Host "停止舊的前端進程..." -ForegroundColor Yellow
Get-Process | Where-Object {$_.ProcessName -like "*node*"} | Stop-Process -Force -ErrorAction SilentlyContinue

Write-Host "切換到前端目錄..." -ForegroundColor Yellow
Push-Location
Set-Location -LiteralPath "frontend"

Write-Host "啟動前端開發伺服器..." -ForegroundColor Green
pnpm run dev

Pop-Location

