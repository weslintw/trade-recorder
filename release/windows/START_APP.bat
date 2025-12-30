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
