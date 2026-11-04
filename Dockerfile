# Stage 1: Build Frontend
FROM node:18-alpine AS frontend-builder
WORKDIR /app/frontend
RUN npm install -g pnpm
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ .
RUN pnpm run build

# Stage 2: Build Backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
COPY . .
WORKDIR /app/backend
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Stage 3: Final Production Image
FROM alpine:3.20
RUN apk update && apk --no-cache add ca-certificates sqlite-libs bash curl wget

# MinIO：dl.min.io 已停止提供二進位（410 Gone），改從 Chainguard 的映像複製（靜態連結，alpine 可直接跑）
COPY --from=cgr.dev/chainguard/minio:latest /usr/bin/minio /usr/local/bin/minio

WORKDIR /app

# 複製編譯完的後端
COPY --from=backend-builder /app/backend/main ./main
# 複製前端編譯好的靜態檔案
RUN mkdir -p /app/frontend/dist
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# 建立資料夾並確保權限 (全部放在 /app/data 底下方便掛載)
RUN mkdir -p /app/data/minio && chmod -R 777 /app/data

# 設定環境變數
ENV PORT=8080
ENV DB_PATH=/app/data/trade_journal.db
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080 9000 9001

# 啟動腳本：增加詳細的環境檢查
CMD ["/bin/bash", "-c", "\
echo '=== [INIT] Environment Check ===' && \
echo '[1/4] Current directory:' && pwd && \
echo '[2/4] /app directory structure:' && ls -laR /app | head -50 && \
echo '[3/4] Checking frontend/dist:' && \
if [ -d /app/frontend/dist ]; then \
  echo '  ✓ frontend/dist exists' && \
  ls -lh /app/frontend/dist | head -10 && \
  if [ -f /app/frontend/dist/index.html ]; then \
    echo '  ✓ index.html found ('$(stat -c%s /app/frontend/dist/index.html)' bytes)'; \
  else \
    echo '  ✗ index.html NOT FOUND!'; \
  fi; \
else \
  echo '  ✗ frontend/dist NOT FOUND!'; \
fi && \
echo '[4/4] Starting services...' && \
echo '--- Starting MinIO ---' && \
if [ -x /usr/local/bin/minio ]; then /usr/local/bin/minio server /app/data/minio --console-address ':9001' > /app/data/minio_startup.log 2>&1 & sleep 10; else echo 'minio binary missing, skip'; fi && \
echo '--- Starting Trade Recorder Backend ---' && \
./main 2>&1"]
