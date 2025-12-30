#!/bin/bash
set -e

echo "--- Starting Trade Recorder Environment ---"
echo "Current Directory: $(pwd)"
ls -F

# 1. 啟動 MinIO 並將日誌導向文件，避免干擾主日誌
echo "--- [1/2] Starting MinIO in background ---"
mkdir -p /app/minio-data
/usr/local/bin/minio server /app/minio-data --console-address ":9001" > /app/minio.log 2>&1 &

# 2. 等待 MinIO 就緒
echo "Waiting for MinIO (10s)..."
sleep 10

# 3. 啟動 Go 後端 (前景執行，確保容器不退出並輸出日誌)
echo "--- [2/2] Starting Go Backend binary ---"
if [ -f "./main" ]; then
    chmod +x ./main
    echo "Binary found, executing..."
    # 執行後端，並確保所有輸出都導向 stdout/stderr
    ./main 2>&1
else
    echo "ERROR: Backend binary 'main' not found in $(pwd)"
    ls -la
    exit 1
fi
