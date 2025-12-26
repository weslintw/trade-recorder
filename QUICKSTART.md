# 快速啟動指南

## Windows 快速啟動

### 1. 安裝MinIO並啟動

在專案根目錄執行：

```powershell
# 下載MinIO (只需執行一次)
Invoke-WebRequest -Uri "https://dl.min.io/server/minio/release/windows-amd64/minio.exe" -OutFile "minio.exe"

# 啟動MinIO
.\minio.exe server .\minio-data --console-address ":9001"
```

保持此終端開啟，MinIO Console: http://localhost:9001

### 2. 啟動後端 (新開終端)

```powershell
cd backend
go mod download
go run cmd/main.go
```

後端運行於: http://localhost:8080

### 3. 啟動前端 (新開終端)

```powershell
cd frontend
npm install
npm run dev
```

前端運行於: http://localhost:5173

### 4. 開始使用

開啟瀏覽器訪問: http://localhost:5173

## 預設設定

- **MinIO帳號**: minioadmin
- **MinIO密碼**: minioadmin
- **後端端口**: 8080
- **前端端口**: 5173
- **資料庫**: trade_journal.db (自動建立)

## 測試API

```powershell
# 健康檢查
curl http://localhost:8080/health

# 取得交易列表
curl http://localhost:8080/api/v1/trades

# 取得統計資料
curl http://localhost:8080/api/v1/stats/summary
```

## 常見問題

**Q: 端口被佔用？**
A: 在 `backend/.env` 修改 `PORT=8080` 改為其他端口

**Q: MinIO無法啟動？**
A: 檢查9000和9001端口是否被佔用

**Q: 圖片無法上傳？**
A: 確認MinIO正在運行，檢查 `backend/.env` 設定

## 專案結構

```
打單紀錄器/
├── backend/              # Golang後端
│   ├── cmd/             # 主程式入口
│   ├── internal/        # 內部套件
│   │   ├── database/    # 資料庫
│   │   ├── handlers/    # API處理器
│   │   ├── models/      # 資料模型
│   │   └── minio/       # MinIO客戶端
│   ├── go.mod
│   └── .env.example
├── frontend/            # Svelte前端
│   ├── src/
│   │   ├── components/  # Svelte元件
│   │   ├── lib/         # 工具函式
│   │   ├── App.svelte   # 主應用
│   │   └── main.js
│   ├── package.json
│   └── vite.config.js
├── minio-data/          # MinIO資料目錄
├── README.md
└── GUIDE.md
```

