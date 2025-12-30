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
# 複製整個專案以確保模組依賴正確
COPY . .
WORKDIR /app/backend
# 停用 CGO 使用純 Go SQLite 驅動，確保在 Alpine 完美執行
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Stage 3: Final Production Image
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs bash curl wget

# 如果需要內建 MinIO (移除此段則需外部 Minio)
RUN wget https://dl.min.io/server/minio/release/linux-amd64/minio -O /usr/local/bin/minio && \
    chmod +x /usr/local/bin/minio

WORKDIR /app

# 複製編譯完的後端
COPY --from=backend-builder /app/backend/main ./main
# 複製前端編譯好的靜態檔案 (由 Go 服務直接 Static Serve)
RUN mkdir -p /app/frontend/dist
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# 建立資料夾
RUN mkdir -p /app/data /app/minio-data && chmod 777 /app/data /app/minio-data

# 設定環境變數
ENV PORT=8080
ENV DB_PATH=/app/data/trade_journal.db
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080 9000 9001

# 啟動指令：同時啟動 MinIO (背景) 與 後端服務 (前景)
# 注意：這會讓後端服務在 8080 同時提供 API 與 前端網頁
CMD ["/bin/bash", "-c", "/usr/local/bin/minio server /app/minio-data --console-address ':9001' & sleep 3 && ./main"]
