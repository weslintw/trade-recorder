# 打單紀錄器 - 使用說明

## 📦 安裝與設定

### 前置需求

1. **Golang** (1.21 或以上)
2. **Node.js** (18 或以上)
3. **MinIO** (用於圖片儲存)

### MinIO 安裝 (Windows)

```powershell
# 下載MinIO
wget https://dl.min.io/server/minio/release/windows-amd64/minio.exe -O minio.exe

# 啟動MinIO (預設帳密: minioadmin/minioadmin)
.\minio.exe server .\minio-data --console-address ":9001"
```

MinIO Console: http://localhost:9001
MinIO API: http://localhost:9000

### 後端設定

```powershell
cd backend

# 複製環境變數範例
Copy-Item .env.example .env

# 下載Go模組
go mod download

# 啟動後端伺服器
go run cmd/main.go
```

後端API將運行於: http://localhost:8080

### 前端設定

```powershell
cd frontend

# 安裝依賴
npm install

# 啟動開發伺服器
npm run dev
```

前端將運行於: http://localhost:5173

## 🚀 快速開始

1. 啟動MinIO (終端1)
2. 啟動後端 (終端2)
3. 啟動前端 (終端3)
4. 開啟瀏覽器訪問 http://localhost:5173

## 📝 功能使用

### 1. 新增交易紀錄

- 點擊「➕ 新增交易」
- 填寫交易資訊：
  - 選擇交易品種 (XAUUSD, NAS100等)
  - 選擇方向 (做多/做空)
  - 輸入進場價格、手數
  - 如已平倉，填寫平倉價格、盈虧
  - 上傳進場圖和平倉圖
  - 新增標籤 (如：#突破、#回踩)
  - 撰寫交易筆記
- 點擊「建立交易」

### 2. 瀏覽交易紀錄

- 在「📋 交易紀錄」頁面查看所有交易
- 使用篩選器依品種、方向、標籤、日期篩選
- 點擊圖片可放大檢視
- 可編輯或刪除交易

### 3. 統計分析

- 在「📈 統計面板」查看：
  - 總交易數、勝率、總盈虧
  - 平均盈虧、最大盈利/虧損
  - 盈虧比 (Profit Factor)
  - 淨值曲線圖
  - 各品種績效統計

## 🗄️ 資料庫結構

### trades (交易紀錄)
- id: 主鍵
- symbol: 交易品種
- side: 方向 (long/short)
- entry_price: 進場價格
- exit_price: 平倉價格
- lot_size: 手數
- pnl: 盈虧金額
- pnl_points: 盈虧點數
- notes: 交易筆記
- entry_time: 進場時間
- exit_time: 平倉時間

### trade_images (交易圖片)
- id: 主鍵
- trade_id: 交易ID (外鍵)
- image_type: 圖片類型 (entry/exit)
- image_path: MinIO路徑

### tags (標籤)
- id: 主鍵
- name: 標籤名稱

### trade_tags (交易-標籤關聯)
- trade_id: 交易ID
- tag_id: 標籤ID

## 🔧 API 端點

### 交易紀錄
- `GET /api/v1/trades` - 取得交易列表
- `GET /api/v1/trades/:id` - 取得單筆交易
- `POST /api/v1/trades` - 建立交易
- `PUT /api/v1/trades/:id` - 更新交易
- `DELETE /api/v1/trades/:id` - 刪除交易

### 圖片管理
- `POST /api/v1/images/upload` - 上傳圖片
- `GET /api/v1/images/:filename` - 取得圖片

### 統計資料
- `GET /api/v1/stats/summary` - 統計摘要
- `GET /api/v1/stats/equity-curve` - 淨值曲線
- `GET /api/v1/stats/by-symbol` - 品種統計

### 標籤
- `GET /api/v1/tags` - 取得所有標籤

## 📱 畫面截圖

專案包含三個主要頁面：
1. **交易紀錄列表** - 瀏覽、篩選、管理交易
2. **新增/編輯交易** - 完整的交易表單與圖片上傳
3. **統計儀表板** - 視覺化績效分析

## 🐛 故障排除

### 後端無法啟動
- 檢查8080端口是否被佔用
- 確認Go版本 >= 1.21
- 執行 `go mod tidy` 重新整理依賴

### 前端無法啟動
- 刪除 `node_modules` 並重新執行 `npm install`
- 檢查5173端口是否被佔用
- 確認Node.js版本 >= 18

### 圖片上傳失敗
- 確認MinIO正在運行
- 檢查 `.env` 中的MinIO設定
- 確認MinIO bucket已建立

### 資料庫錯誤
- 刪除 `trade_journal.db` 讓系統重新建立
- 檢查檔案權限

## 🔐 安全性建議

生產環境部署時：
1. 更改MinIO預設帳號密碼
2. 使用環境變數儲存敏感資訊
3. 啟用HTTPS
4. 設定適當的CORS政策
5. 實作使用者認證機制

## 📈 未來擴充

可考慮的功能：
- [ ] 使用者認證與多帳號支援
- [ ] 交易策略分類
- [ ] 更多圖表類型 (K線圖、熱力圖)
- [ ] 匯出Excel報表
- [ ] 行動版APP
- [ ] 交易提醒通知
- [ ] AI交易分析建議

## 💡 技術棧

- **後端**: Golang + Gin框架
- **前端**: Svelte + Vite
- **資料庫**: SQLite
- **圖片儲存**: MinIO (S3相容)
- **圖表**: Chart.js

## 📄 授權

MIT License

## 🙋 支援

如有問題或建議，歡迎開Issue討論！

