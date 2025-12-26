# 打單紀錄器 (Trade Journal) 📊

CFG價差合約交易日誌系統，專為交易者設計，用於系統化記錄交易歷程、上傳盤面截圖，並透過數據分析進行有效復盤。

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)
![Svelte](https://img.shields.io/badge/Svelte-4.0+-FF3E00?logo=svelte)

## ✨ 功能特色

### 📝 完整的交易紀錄
- 記錄交易品種（XAUUSD, NAS100, EURUSD等）
- 記錄交易方向（做多/做空）、價格、手數、盈虧
- 自定義標籤系統（#突破、#回踩、#新聞單等）
- 撰寫交易心得與筆記，記錄心態與策略

### 🖼️ 智能圖片管理
- 支援多圖上傳（進場圖、平倉圖）
- 前端即時預覽功能
- 按月份自動分類整理
- 點擊放大檢視功能
- 儲存至MinIO雲端物件存儲

### 🔍 強大的復盤工具
- 歷史交易清單展示
- 多條件篩選（品種、方向、標籤、日期區間）
- 圖片快速瀏覽
- 編輯與刪除功能

### 📈 數據統計與視覺化
- **核心指標**：總交易數、勝率、總盈虧、平均盈虧
- **風險指標**：最大盈利、最大虧損、盈虧比（Profit Factor）
- **淨值曲線**：視覺化帳戶成長狀況
- **品種分析**：各交易品種績效統計

## 🚀 快速開始

### Windows 用戶（推薦）

直接執行一鍵啟動腳本：

```powershell
.\start.ps1
```

腳本會自動：
1. 下載並啟動MinIO
2. 啟動後端服務
3. 安裝前端依賴並啟動
4. 開啟瀏覽器

### 手動啟動

#### 前置需求
- [Golang](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [MinIO](https://min.io/download)

#### 步驟1: 啟動MinIO

**Windows:**
```powershell
# 下載MinIO
Invoke-WebRequest -Uri "https://dl.min.io/server/minio/release/windows-amd64/minio.exe" -OutFile "minio.exe"

# 啟動
.\minio.exe server .\minio-data --console-address ":9001"
```

**macOS/Linux:**
```bash
# 安裝 (macOS)
brew install minio/stable/minio

# 啟動
minio server ./minio-data --console-address ":9001"
```

MinIO Console: http://localhost:9001 (帳號/密碼: minioadmin)

#### 步驟2: 啟動後端

```bash
cd backend
cp .env.example .env  # 複製環境變數檔案
go mod download       # 下載依賴
go run cmd/main.go    # 啟動服務
```

後端運行於: http://localhost:8080

#### 步驟3: 啟動前端

```bash
cd frontend
npm install      # 安裝依賴
npm run dev      # 啟動開發服務器
```

前端運行於: http://localhost:5173

### Docker 部署

```bash
docker-compose up -d
```

所有服務將自動啟動：
- 前端: http://localhost:5173
- 後端: http://localhost:8080
- MinIO: http://localhost:9001

## 📁 專案結構

```
打單紀錄器/
├── backend/                    # Golang後端
│   ├── cmd/
│   │   └── main.go            # 主程式入口
│   ├── internal/
│   │   ├── database/          # 資料庫初始化
│   │   ├── handlers/          # API處理器
│   │   │   ├── trade.go       # 交易CRUD
│   │   │   ├── image.go       # 圖片上傳
│   │   │   └── stats.go       # 統計API
│   │   ├── models/            # 資料模型
│   │   └── minio/             # MinIO客戶端
│   ├── go.mod
│   ├── .env.example
│   └── Dockerfile
├── frontend/                   # Svelte前端
│   ├── src/
│   │   ├── components/
│   │   │   ├── TradeForm.svelte      # 交易表單
│   │   │   ├── TradeList.svelte      # 交易列表
│   │   │   ├── Dashboard.svelte      # 統計儀表板
│   │   │   └── EquityChart.svelte    # 淨值曲線圖
│   │   ├── lib/
│   │   │   └── api.js         # API封裝
│   │   ├── App.svelte         # 主應用
│   │   └── main.js
│   ├── package.json
│   ├── vite.config.js
│   ├── Dockerfile
│   └── nginx.conf
├── .gitignore
├── README.md
├── GUIDE.md                    # 詳細使用指南
├── QUICKSTART.md               # 快速開始指南
├── docker-compose.yml
├── start.ps1                   # Windows啟動腳本
└── start.sh                    # Linux/macOS啟動腳本
```

## 🔌 API 端點

### 交易紀錄
- `GET /api/v1/trades` - 取得交易列表（支援篩選與分頁）
- `GET /api/v1/trades/:id` - 取得單筆交易詳情
- `POST /api/v1/trades` - 建立新交易
- `PUT /api/v1/trades/:id` - 更新交易
- `DELETE /api/v1/trades/:id` - 刪除交易

### 圖片管理
- `POST /api/v1/images/upload` - 上傳圖片
- `GET /api/v1/images/:filename` - 取得圖片

### 統計資料
- `GET /api/v1/stats/summary` - 統計摘要
- `GET /api/v1/stats/equity-curve` - 淨值曲線數據
- `GET /api/v1/stats/by-symbol` - 各品種統計

### 標籤
- `GET /api/v1/tags` - 取得所有標籤

## 🗄️ 資料庫結構

### trades (交易紀錄)
```sql
CREATE TABLE trades (
    id INTEGER PRIMARY KEY,
    symbol VARCHAR(20),           -- 交易品種
    side VARCHAR(10),             -- long/short
    entry_price REAL,             -- 進場價格
    exit_price REAL,              -- 平倉價格
    lot_size REAL,                -- 手數
    pnl REAL,                     -- 盈虧金額
    pnl_points REAL,              -- 盈虧點數
    notes TEXT,                   -- 交易筆記
    entry_time DATETIME,          -- 進場時間
    exit_time DATETIME,           -- 平倉時間
    created_at DATETIME,
    updated_at DATETIME
);
```

### trade_images (交易圖片)
關聯交易ID與MinIO儲存路徑

### tags (標籤)
自定義標籤系統

### trade_tags (交易-標籤關聯)
多對多關聯表

## 🎨 畫面預覽

專案包含三個主要頁面：

1. **📋 交易紀錄列表**
   - 卡片式展示所有交易
   - 多條件篩選器
   - 圖片縮圖預覽
   - 快速編輯/刪除

2. **➕ 新增/編輯交易**
   - 完整的表單輸入
   - 圖片上傳與預覽
   - 標籤管理
   - 交易筆記編輯器

3. **📈 統計儀表板**
   - 9宮格統計卡片
   - 互動式淨值曲線圖（Chart.js）
   - 各品種績效表格

## 💡 使用建議

### 記錄交易的最佳實踐

1. **進場時**：立即記錄並截圖
   - 記錄進場理由、策略
   - 標記交易型態（#突破、#反轉等）
   - 上傳盤面截圖

2. **平倉後**：完成紀錄
   - 更新平倉價格與盈虧
   - 上傳結果截圖
   - 撰寫復盤心得

3. **定期復盤**：使用統計功能
   - 查看勝率與盈虧比
   - 分析各品種表現
   - 檢視淨值曲線趨勢

## 🔧 環境變數設定

建立 `backend/.env` 檔案：

```env
PORT=8080
DB_PATH=./trade_journal.db

# MinIO設定
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
```

## 🐛 常見問題

**Q: 後端無法啟動？**
- 檢查8080端口是否被佔用
- 確認Go版本 >= 1.21
- 執行 `go mod tidy`

**Q: 圖片上傳失敗？**
- 確認MinIO正在運行（http://localhost:9000）
- 檢查 `.env` 中的MinIO設定
- 查看MinIO Console確認bucket存在

**Q: 前端無法連接後端？**
- 確認後端運行於8080端口
- 檢查瀏覽器Console是否有CORS錯誤

詳細故障排除請參考 [GUIDE.md](GUIDE.md)

## 🔐 安全性提醒

⚠️ 本專案為個人使用設計，部署到生產環境時請注意：

- 更改MinIO預設帳號密碼
- 使用環境變數儲存敏感資訊
- 啟用HTTPS/TLS
- 實作使用者認證
- 設定適當的CORS政策

## 🛣️ 未來規劃

- [ ] 使用者認證與多帳號支援
- [ ] 交易策略分類系統
- [ ] 更多圖表類型（K線圖整合）
- [ ] Excel/PDF報表匯出
- [ ] 行動版APP
- [ ] 交易提醒與通知
- [ ] AI交易分析建議

## 🤝 貢獻

歡迎提交Issue和Pull Request！

## 📄 授權

本專案採用 MIT 授權條款

## 📞 支援

如有問題或建議，歡迎開Issue討論！

---

**Made with ❤️ for Traders**

