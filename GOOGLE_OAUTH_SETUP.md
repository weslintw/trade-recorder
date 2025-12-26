# Google OAuth 設定指南

## 📝 Step 1: 建立 Google Cloud 專案

### 1.1 訪問 Google Cloud Console
https://console.cloud.google.com/

### 1.2 建立新專案
1. 點擊頂部的專案選擇器
2. 點擊「新增專案」
3. 輸入專案名稱：`Trade Journal` 或 `打單紀錄器`
4. 點擊「建立」

---

## 🔧 Step 2: 啟用 Google+ API

### 2.1 前往 API 庫
1. 在左側選單選擇「API 和服務」→「程式庫」
2. 搜尋「Google+ API」
3. 點擊進入並點擊「啟用」

---

## 🔑 Step 3: 建立 OAuth 2.0 憑證

### 3.1 設定 OAuth 同意畫面
1. 前往「API 和服務」→「OAuth 同意畫面」
2. 選擇「外部」（External）
3. 填寫應用程式資訊：
   - **應用程式名稱**: `打單紀錄器`
   - **使用者支援電子郵件**: 您的 Gmail
   - **開發人員聯絡資訊**: 您的 Gmail

4. 點擊「儲存並繼續」

### 3.2 設定範圍（Scopes）
1. 點擊「新增或移除範圍」
2. 選擇以下範圍：
   - `userinfo.email`
   - `userinfo.profile`
   - `openid`

3. 點擊「儲存並繼續」

### 3.3 建立憑證
1. 前往「API 和服務」→「憑證」
2. 點擊「建立憑證」→「OAuth 2.0 用戶端 ID」
3. 選擇應用程式類型：**Web 應用程式**
4. 填寫資訊：
   - **名稱**: `Trade Journal Web Client`
   
   - **已授權的 JavaScript 來源**:
     ```
     http://localhost:5174
     http://localhost:8080
     ```
   
   - **已授權的重新導向 URI**:
     ```
     http://localhost:8080/api/v1/auth/google/callback
     http://localhost:5174/auth/callback
     ```

5. 點擊「建立」

---

## 📋 Step 4: 取得憑證

建立完成後會顯示：

**用戶端 ID (Client ID)**:
```
例如：123456789-abc123def456.apps.googleusercontent.com
```

**用戶端密鑰 (Client Secret)**:
```
例如：GOCSPX-abc123def456ghi789
```

⚠️ **立即複製並安全保存！**

---

## ⚙️ Step 5: 設定環境變數

在 `backend/.env` 新增：

```env
# Google OAuth
GOOGLE_CLIENT_ID=你的_client_id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=GOCSPX-你的_client_secret
GOOGLE_REDIRECT_URL=http://localhost:8080/api/v1/auth/google/callback

# 前端 URL
FRONTEND_URL=http://localhost:5174
```

---

## ✅ 完成檢查清單

- [ ] 已建立 Google Cloud 專案
- [ ] 已啟用 Google+ API
- [ ] 已設定 OAuth 同意畫面
- [ ] 已建立 OAuth 2.0 憑證
- [ ] 已設定正確的重新導向 URI
- [ ] 已取得 Client ID 和 Secret
- [ ] 已將憑證加入 `.env` 檔案

---

## 🚀 設定完成後

完成以上步驟後，請回覆：
```
✅ Google OAuth 已設定完成
Client ID: 123456789-abc...
```

---

## 📞 遇到問題？

### 常見問題

**Q: 找不到 Google+ API？**
A: Google+ API 已被棄用，但仍可用於 OAuth。也可以直接使用 Google Identity API。

**Q: 重新導向 URI 錯誤？**
A: 確保 URI 完全一致，包含 `http://` 和結尾的路徑。

**Q: 只能自己登入？**
A: 開發階段在「OAuth 同意畫面」→「測試使用者」新增允許的 Gmail 帳號。

**Q: 需要發布應用程式嗎？**
A: 開發測試不需要，但如果要給其他人使用，需要提交審核。

---

## 🔒 安全提醒

1. **絕對不要**將 Client Secret 提交到 Git
2. `.env` 檔案已加入 `.gitignore`
3. 定期輪換憑證
4. 只授予必要的權限範圍

---

**祝設定順利！** 🎉

