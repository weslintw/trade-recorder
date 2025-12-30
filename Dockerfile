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
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs bash curl wget

# 下載 MinIO
RUN wget https://dl.min.io/server/minio/release/linux-amd64/minio -O /usr/local/bin/minio && \
    chmod +x /usr/local/bin/minio

WORKDIR /app

# 複製編譯完的後端
COPY --from=backend-builder /app/backend/main ./main
# 複製前端編譯好的靜態檔案
RUN mkdir -p /app/frontend/dist
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# 建立資料夾並確保權限
RUN mkdir -p /app/data /app/minio-data && chmod 777 /app/data /app/minio-data

# 設定環境變數
ENV PORT=8080
ENV DB_PATH=/app/data/trade_journal.db
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080 9000 9001

# 直接在 CMD 撰寫強健的啟動邏輯，避免外部腳本編碼問題
CMD ["/bin/bash", "-c", "echo '--- [INIT] Environment Check ---' && ls -l ./main && echo '--- [1/2] Starting MinIO ---' && /usr/local/bin/minio server /app/minio-data --console-address ':9001' > minio_startup.log 2>&1 & sleep 10 && echo '--- [2/2] Starting Trade Recorder Backend ---' && ./main 2>&1"]
