# 多階段構建
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 複製依賴檔案
COPY go.mod go.sum ./
RUN go mod download

# 複製原始碼
COPY . .

# 編譯
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./cmd/main.go

# 最終映像
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

# 從builder複製編譯好的二進制檔案
COPY --from=builder /app/main .

# 建立資料目錄
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./main"]

