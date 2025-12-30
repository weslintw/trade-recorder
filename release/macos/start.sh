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
