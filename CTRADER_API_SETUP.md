# cTrader Open API 申請指南

## 📝 Step 1: 註冊 cTrader Open API 帳號

### 1.1 訪問註冊頁面
https://connect.ctrader.com/

### 1.2 使用您的 cTrader ID 登入
- 使用您現有的 cTrader 帳號登入
- 或使用 Email 註冊新的開發者帳號

---

## 🔧 Step 2: 建立應用程式

### 2.1 進入開發者控制台
登入後訪問：https://connect.ctrader.com/apps

### 2.2 點擊「Create New App」

### 2.3 填寫應用程式資訊

**基本資訊：**
- **App Name**: `Trade Journal` 或 `打單紀錄器`
- **Description**: `交易日誌系統 - 用於記錄和分析交易歷程`
- **Website**: `http://localhost:8080` （開發環境）
- **App Type**: 選擇 `Web Application`

**Redirect URIs（重要！）：**
```
http://localhost:8080/api/v1/ctrader/callback
http://localhost:5174/auth/ctrader/callback
```

**權限（Scopes）：**
勾選以下權限：
- ✅ `trading` - 存取交易資料
- ✅ `accounts:read` - 讀取帳戶資訊
- ✅ `history:read` - 讀取歷史交易

---

## 🔑 Step 3: 取得憑證

建立完成後，您會得到：

1. **Client ID**（公開）
   ```
   例如：7_5az7pj935owsss8kw84oko0cg0ks...
   ```

2. **Client Secret**（保密！）
   ```
   例如：49u5vdfa6e8oo4ogk8ksws0c0gckk...
   ```

⚠️ **請安全保存這些資訊！**

---

## 📋 Step 4: 測試 API 存取

### 4.1 使用 cTrader Playground
訪問：https://spotware.github.io/openapi-playground/

### 4.2 測試連線
1. 輸入您的 Client ID 和 Secret
2. 點擊「Authorize」
3. 確認可以看到您的帳戶資訊

---

## 🔍 Step 5: 查看 API 文件

### 重要文件連結

1. **官方文件**
   https://help.ctrader.com/open-api/

2. **REST API 參考**
   https://openapi.ctrader.com/rest/

3. **認證流程**
   https://help.ctrader.com/open-api/authentication/

4. **交易 API**
   https://help.ctrader.com/open-api/trading-api/

---

## 🎯 我們需要的 API 端點

### 1. 取得帳戶資訊
```
GET /v3/accounts
```

### 2. 取得交易歷史
```
GET /v3/accounts/{accountId}/history
```

參數：
- `from` - 開始時間（Unix timestamp）
- `to` - 結束時間（Unix timestamp）

### 3. 取得特定訂單
```
GET /v3/accounts/{accountId}/orders/{orderId}
```

---

## ⚙️ Step 6: 設定環境變數

取得憑證後，在 `backend/.env` 新增：

```env
# cTrader Open API
CTRADER_CLIENT_ID=你的_client_id_這裡
CTRADER_CLIENT_SECRET=你的_client_secret_這裡
CTRADER_REDIRECT_URL=http://localhost:8080/api/v1/ctrader/callback
CTRADER_API_URL=https://openapi.ctrader.com
```

---

## ✅ 完成檢查清單

申請完成後，請確認：

- [ ] 已取得 Client ID
- [ ] 已取得 Client Secret
- [ ] 已設定正確的 Redirect URIs
- [ ] 已勾選正確的權限（trading, accounts:read, history:read）
- [ ] 已測試 API 連線成功
- [ ] 已將憑證加入 `.env` 檔案

---

## 🚀 申請完成後

完成以上步驟後，請回覆：
```
✅ cTrader API 已申請完成
Client ID: 7_5az... （前幾碼）
```

我將立即開始實作整合功能！

---

## 📞 遇到問題？

### 常見問題

**Q: 找不到「Create New App」按鈕？**
A: 確認您已用 cTrader ID 登入，並且訪問的是 https://connect.ctrader.com/apps

**Q: 權限選項不顯示？**
A: 某些 broker 可能需要先聯絡客服開通 API 存取權限

**Q: 測試連線失敗？**
A: 檢查 Redirect URI 是否完全一致（包含 http:// 和結尾的 /）

**Q: 需要付費嗎？**
A: cTrader Open API 基本使用是免費的

---

## 📧 需要協助？

如果遇到任何問題，可以：
1. 查看 cTrader 官方文件
2. 聯絡您的 broker 客服
3. 訪問 cTrader 社群論壇

---

**祝申請順利！** 🎉

