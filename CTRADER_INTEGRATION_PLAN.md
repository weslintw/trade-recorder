# cTrader 整合功能規劃

## 🎯 功能目標

1. **使用者認證**：使用 Gmail (Google OAuth) 登入
2. **cTrader 連結**：OAuth 連結用戶的 cTrader 帳號
3. **交易同步**：自動從 cTrader 撈取交易紀錄

---

## 📋 技術架構

### 後端技術棧
- **認證**：JWT Token + Google OAuth 2.0
- **cTrader API**：cTrader Open API (REST + WebSocket)
- **資料庫**：新增 users 和 ctrader_accounts 資料表

### 前端技術棧
- **登入**：Google OAuth 按鈕
- **帳號管理**：cTrader 連結設定頁面
- **同步**：一鍵同步按鈕

---

## 🗄️ 資料庫設計

### 1. users 表（使用者）
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    google_id VARCHAR(255) UNIQUE,
    name VARCHAR(100),
    avatar_url VARCHAR(500),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 2. ctrader_accounts 表（cTrader 帳號）
```sql
CREATE TABLE ctrader_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    account_id VARCHAR(100) NOT NULL,
    account_name VARCHAR(100),
    broker VARCHAR(100),
    last_sync_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

### 3. 修改 trades 表（加入 user_id）
```sql
ALTER TABLE trades ADD COLUMN user_id INTEGER;
ALTER TABLE trades ADD COLUMN ctrader_order_id VARCHAR(100);
ALTER TABLE trades ADD COLUMN source VARCHAR(20) DEFAULT 'manual';
```

---

## 🔐 Google OAuth 設定步驟

### 1. 建立 Google Cloud 專案
1. 訪問：https://console.cloud.google.com/
2. 建立新專案「Trade Journal」
3. 啟用 Google+ API

### 2. 建立 OAuth 2.0 憑證
1. 前往「API 和服務」→「憑證」
2. 建立「OAuth 2.0 用戶端 ID」
3. 應用程式類型：Web 應用程式
4. 授權重新導向 URI：
   - `http://localhost:8080/api/v1/auth/google/callback`
   - `http://localhost:5174/auth/callback`

### 3. 取得憑證
- **Client ID**：`YOUR_CLIENT_ID.apps.googleusercontent.com`
- **Client Secret**：`YOUR_CLIENT_SECRET`

---

## 🔌 cTrader Open API

### API 端點
- **Auth**：https://openapi.ctrader.com/
- **Trading API**：https://api.ctrader.com/

### 需要的權限
- `trading` - 存取交易資料
- `accounts` - 存取帳戶資訊

### 申請流程
1. 註冊：https://openapi.ctrader.com/
2. 建立應用程式
3. 取得 Client ID 和 Secret
4. 設定 Redirect URI：`http://localhost:8080/api/v1/ctrader/callback`

---

## 🚀 實作步驟

### Phase 1：使用者認證系統（預計 2-3 小時）
- [ ] 建立 users 資料表
- [ ] 實作 Google OAuth 登入流程
- [ ] 實作 JWT Token 生成與驗證
- [ ] 建立登入/登出 API
- [ ] 前端登入頁面

### Phase 2：cTrader 整合（預計 3-4 小時）
- [ ] 建立 ctrader_accounts 資料表
- [ ] 實作 cTrader OAuth 流程
- [ ] 實作 API 呼叫封裝
- [ ] 前端 cTrader 連結設定頁面

### Phase 3：交易同步（預計 2-3 小時）
- [ ] 實作從 cTrader 獲取交易紀錄
- [ ] 資料轉換與對應
- [ ] 自動/手動同步功能
- [ ] 前端同步 UI

---

## 📦 需要安裝的套件

### 後端
```bash
go get golang.org/x/oauth2
go get golang.org/x/oauth2/google
go get github.com/golang-jwt/jwt/v5
```

### 前端
```bash
pnpm add @auth/core
pnpm add jwt-decode
```

---

## 🔧 環境變數

在 `backend/.env` 新增：
```env
# Google OAuth
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback

# cTrader API
CTRADER_CLIENT_ID=your_ctrader_client_id
CTRADER_CLIENT_SECRET=your_ctrader_client_secret
CTRADER_REDIRECT_URL=http://localhost:8080/api/v1/ctrader/callback

# JWT
JWT_SECRET=your_random_secret_key_here
JWT_EXPIRY=24h
```

---

## 🎨 UI 設計

### 1. 登入頁面
- Google 登入按鈕
- 簡潔的歡迎訊息

### 2. 設定頁面（新增）
- cTrader 連結狀態
- 連結/解除連結按鈕
- 最後同步時間

### 3. 交易列表頁面（修改）
- 顯示資料來源標籤（手動/cTrader）
- 同步按鈕
- 同步進度顯示

---

## 📝 API 端點規劃

### 認證相關
- `GET /api/v1/auth/google` - 開始 Google 登入
- `GET /api/v1/auth/google/callback` - Google 回調
- `POST /api/v1/auth/logout` - 登出
- `GET /api/v1/auth/me` - 取得當前使用者

### cTrader 相關
- `GET /api/v1/ctrader/auth` - 開始 cTrader 連結
- `GET /api/v1/ctrader/callback` - cTrader 回調
- `GET /api/v1/ctrader/accounts` - 取得已連結帳號
- `POST /api/v1/ctrader/sync` - 手動同步交易
- `DELETE /api/v1/ctrader/disconnect` - 解除連結

---

## ⚠️ 注意事項

1. **安全性**
   - JWT Token 存儲在 HttpOnly Cookie
   - CSRF 保護
   - API Rate Limiting

2. **資料隱私**
   - cTrader Token 加密存儲
   - 使用者資料隔離

3. **同步策略**
   - 避免重複同步
   - 增量同步（只同步新資料）
   - 衝突處理（手動 vs 自動）

---

## 🎯 下一步行動

請確認：
1. 您是否已有 cTrader 帳號？
2. 是否需要我先實作 Google 登入，還是直接整合 cTrader？
3. 您希望同步所有歷史交易，還是只同步最近的？

確認後我將開始實作！

