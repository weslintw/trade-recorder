# 構建階段
FROM node:18-alpine AS builder

WORKDIR /app

# 複製依賴檔案
COPY package*.json ./
RUN npm install

# 複製原始碼
COPY . .

# 構建
RUN npm run build

# 生產階段
FROM nginx:alpine

# 複製構建產物到nginx
COPY --from=builder /app/dist /usr/share/nginx/html

# 複製nginx設定
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]

