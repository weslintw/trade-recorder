#!/bin/bash

echo "================================"
echo "   打單紀錄器 - 啟動中..."
echo "================================"
echo ""

# 檢查MinIO
echo "[1/3] 啟動MinIO..."
if ! command -v minio &> /dev/null; then
    echo "請先安裝MinIO"
    echo "macOS: brew install minio/stable/minio"
    echo "Linux: 請參考 https://min.io/download"
    exit 1
fi

mkdir -p minio-data
minio server minio-data --console-address ":9001" &
sleep 3
echo "✓ MinIO已啟動 (http://localhost:9001)"
echo ""

# 啟動後端
echo "[2/3] 啟動後端..."
cd backend

if [ ! -f "go.sum" ]; then
    echo "下載Go模組..."
    go mod download
fi

if [ ! -f ".env" ]; then
    cp .env.example .env
    echo "✓ 環境變數檔案已建立"
fi

go run cmd/main.go &
cd ..
sleep 2
echo "✓ 後端已啟動 (http://localhost:8080)"
echo ""

# 啟動前端
echo "[3/3] 啟動前端..."
cd frontend

if [ ! -d "node_modules" ]; then
    echo "安裝npm套件..."
    npm install
fi

npm run dev -- --host 0.0.0.0 &
cd ..
echo "✓ 前端已啟動 (http://localhost:5173)"
echo ""

echo "================================"
echo "   所有服務已啟動！"
echo "================================"
echo ""
echo "📝 前端應用: http://localhost:5173"
echo "🔧 後端API:  http://localhost:8080"
echo "💾 MinIO:    http://localhost:9001 (minioadmin/minioadmin)"
echo ""
echo "按Ctrl+C停止所有服務"

wait

