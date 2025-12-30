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
WORKDIR /app/backend
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
# Compile the backend with CGO disabled for maximum portability
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

# Stage 3: Final Production Image
FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs
WORKDIR /app

# Copy the compiled Go binary
COPY --from=backend-builder /app/backend/main .

# Copy the frontend build artifacts (Go will serve these)
RUN mkdir -p /app/frontend/dist
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

# Default config
ENV PORT=8080
ENV DB_PATH=/app/data/trade_journal.db
RUN mkdir -p /app/data

# Exposure
EXPOSE 8080

# Run the app
CMD ["./main"]
