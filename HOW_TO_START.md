# 🎉 啟動指南

## ✅ 當前狀態

### 後端
✅ **已成功啟動並運行中！**
- URL: http://localhost:8080
- 所有 API 端點正常
- 資料庫已初始化

### 前端
❌ 需要手動啟動（vite 執行檔問題）

### MinIO
⚠️ 需要啟動

---

## 🚀 啟動方式

我已建立三個獨立的啟動腳本，方便您分別管理各服務。

### 方法 1：使用獨立啟動腳本（推薦）

開啟三個 PowerShell 視窗，分別執行：

#### 終端 1 - MinIO
```powershell
.\start-minio.ps1
```

#### 終端 2 - 後端（如果還沒啟動）
```powershell
.\start-backend.ps1
```

#### 終端 3 - 前端
```powershell
.\start-frontend.ps1
```

### 方法 2：手動啟動

#### 啟動 MinIO
```powershell
.\minio.exe server .\minio-data --console-address ":9001"
```

#### 啟動後端
```powershell
cd backend
go run cmd/main.go
```

#### 啟動前端
```powershell
cd frontend
pnpm install  # 首次需要
pnpm run dev
```

---

## 📊 服務資訊

| 服務 | 狀態 | URL | 說明 |
|------|------|-----|------|
| 後端 API | ✅ 運行中 | http://localhost:8080 | Golang REST API |
| 前端應用 | ⏳ 待啟動 | http://localhost:5173 | Svelte UI |
| MinIO | ⏳ 待啟動 | http://localhost:9001 | 圖片儲存 (minioadmin/minioadmin) |

---

## 🔧 前端啟動問題排除

如果前端啟動失敗（錯誤 3221225477），請嘗試：

### 方案 A：清理並重裝
```powershell
cd frontend
Remove-Item -Recurse -Force node_modules
Remove-Item -Force pnpm-lock.yaml
pnpm install
pnpm run dev
```

### 方案 B：檢查防毒軟體
- Windows Defender 可能攔截 vite 執行檔
- 暫時停用或將專案資料夾加入白名單

### 方案 C：使用 npm 替代
```powershell
cd frontend
npm install
npm run dev
```

---

## 📝 驗證後端運作

後端已成功啟動！您可以測試：

```powershell
# 健康檢查
curl http://localhost:8080/health

# 取得交易列表
curl http://localhost:8080/api/v1/trades

# 取得統計資料
curl http://localhost:8080/api/v1/stats/summary
```

---

## 🎯 完整啟動流程

1. **開啟終端 1**：執行 `.\start-minio.ps1`
   - 等待看到 "MinIO Object Storage Server" 訊息

2. **開啟終端 2**：執行 `.\start-backend.ps1`（如果還沒啟動）
   - 等待看到 "伺服器啟動於 http://localhost:8080"

3. **開啟終端 3**：執行 `.\start-frontend.ps1`
   - 等待看到 "Local: http://localhost:5173"

4. **開啟瀏覽器**：訪問 http://localhost:5173

---

## 💡 使用提示

### 新增交易
1. 點擊「➕ 新增交易」
2. 填寫交易資訊
3. 上傳進場圖和平倉圖
4. 新增標籤
5. 儲存

### 查看統計
1. 點擊「📈 統計面板」
2. 查看勝率、盈虧比
3. 檢視淨值曲線

---

## 🔥 重點提醒

- ✅ 後端已正常運行
- ✅ 資料庫已初始化
- ✅ 無需 GCC 或任何 C 編譯器
- ⚠️ 前端需要手動啟動
- ⚠️ MinIO 需要啟動

---

## 📚 相關檔案

- `start-frontend.ps1` - 前端啟動腳本
- `start-backend.ps1` - 後端啟動腳本
- `start-minio.ps1` - MinIO 啟動腳本
- `SUCCESS.md` - 完整測試結果
- `FIXED.md` - 修復說明

---

**開始記錄您的交易！** 📈💪

