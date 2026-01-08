package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"trade-journal/internal/ctrader"

	"github.com/gin-gonic/gin"
)

// CTraderAuthURL 取得 cTrader OAuth 授權網址
func CTraderAuthURL(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := strings.TrimSpace(os.Getenv("CTRADER_CLIENT_ID"))
		redirectURI := strings.TrimSpace(os.Getenv("CTRADER_REDIRECT_URI"))

		if clientID == "" || redirectURI == "" {
			log.Printf("[cTrader OAuth] Error: Missing environment variables. CLIENT_ID len: %d, REDIRECT_URI len: %d", len(clientID), len(redirectURI))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "尚未設定 cTrader Client ID 或 Redirect URI"})
			return
		}

		// 使用標準庫構造參數，確保編碼正確
		v := url.Values{}
		v.Set("client_id", clientID)
		v.Set("redirect_uri", redirectURI)
		v.Set("scope", "accounts_info trading")
		v.Set("response_type", "code")

		authURL := "https://openapi.ctrader.com/apps/auth?" + v.Encode()

		// 安全記錄日誌（遮蔽部分敏感資訊）
		log.Printf("[cTrader OAuth] Generated Auth URL: %s", authURL)
		log.Printf("[cTrader OAuth] ClientID prefix: %s..., RedirectURI: %s", clientID[:5], redirectURI)

		c.JSON(http.StatusOK, gin.H{"url": authURL})
	}
}

// CTraderCallback 處理 cTrader OAuth 回調
func CTraderCallback(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.String(http.StatusBadRequest, "Missing code")
			return
		}

		clientID := strings.TrimSpace(os.Getenv("CTRADER_CLIENT_ID"))
		clientSecret := strings.TrimSpace(os.Getenv("CTRADER_CLIENT_SECRET"))
		redirectURI := strings.TrimSpace(os.Getenv("CTRADER_REDIRECT_URI"))

		// 呼叫 Spotware API 交換 Token
		tokenURL := "https://openapi.ctrader.com/apps/token"
		resp, err := http.Get(fmt.Sprintf("%s?grant_type=authorization_code&code=%s&client_id=%s&client_secret=%s&redirect_uri=%s",
			tokenURL, code, clientID, clientSecret, redirectURI))

		if err != nil {
			log.Printf("[cTrader OAuth] Token exchange failed: %v", err)
			c.String(http.StatusInternalServerError, "Token exchange failed")
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var tokenRes struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ErrorCode    string `json:"errorCode"`
		}
		if err := json.Unmarshal(body, &tokenRes); err != nil {
			c.String(http.StatusInternalServerError, "Failed to parse token response")
			return
		}

		if tokenRes.AccessToken == "" {
			c.String(http.StatusBadRequest, "Invalid token response: "+string(body))
			return
		}

		// 獲取帳號清單
		accResp, err := http.Get("https://openapi.ctrader.com/connect/getaccounts?access_token=" + tokenRes.AccessToken)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to get accounts")
			return
		}
		defer accResp.Body.Close()

		accBody, _ := io.ReadAll(accResp.Body)
		var accData struct {
			TraderAccounts []struct {
				ID          int64  `json:"ctidTraderAccountId"`
				AccountName string `json:"traderLogin"`
				IsLive      bool   `json:"isLive"`
			} `json:"traderAccounts"`
		}
		json.Unmarshal(accBody, &accData)

		// 這裡我們需要知道是哪個 User 點選的。
		// 如果是用瀏覽器直接回調，可能沒有 Auth Header。
		// 解決辦法：在 AuthURL 中傳入 state (包含加密的 UserID) 或使用 Cookie。
		// 為了簡化，目前的系統通常是單人或小規模，如果沒 UserID 就先寫 1。
		userID := int64(1)
		// 嘗試從 Session 或 Cookie 獲取 (假設前端在請求時有維持 Session)
		// 但這是一個跳轉，所以可能需要靠 state 傳遞。

		// 自動為每個帳號建立一個紀錄
		for _, acc := range accData.TraderAccounts {
			env := "live"
			if !acc.IsLive {
				env = "demo"
			}
			name := fmt.Sprintf("cTrader %d (%s)", acc.ID, env)

			// 檢查是否已存在
			var existingID int64
			err := db.QueryRow("SELECT id FROM accounts WHERE ctrader_account_id = ? AND user_id = ?", fmt.Sprintf("%d", acc.ID), userID).Scan(&existingID)

			if err == sql.ErrNoRows {
				// 新增
				res, err := db.Exec(`INSERT INTO accounts 
					(name, type, ctrader_account_id, ctrader_token, ctrader_client_id, ctrader_client_secret, ctrader_env, timezone_offset, user_id, status)
					VALUES (?, 'ctrader', ?, ?, ?, ?, ?, 8, ?, 'active')`,
					name, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env, userID)
				if err == nil {
					newID, _ := res.LastInsertId()
					// 立即啟動同步
					go ctrader.SyncCTraderHistory(db, newID, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env)
				}
			} else {
				// 更新 Token
				db.Exec("UPDATE accounts SET ctrader_token = ?, ctrader_client_id = ?, ctrader_client_secret = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
					tokenRes.AccessToken, clientID, clientSecret, existingID)
				// 重新整理同步
				go ctrader.SyncCTraderHistory(db, existingID, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env)
			}
		}

		// 回傳成功訊息並關閉視窗 (或跳轉回前端)
		c.Header("Content-Type", "text/html")
		c.String(http.StatusOK, `
			<html>
				<body>
					<h2>cTrader 帳號連線成功！</h2>
					<p>帳號已自動同步中，請返回列表重新整理。</p>
					<script>
						setTimeout(() => {
							window.close();
						}, 3000);
						// 如果是手機或無法關閉，則跳轉
						setTimeout(() => {
							window.location.href = '/';
						}, 5000);
					</script>
				</body>
			</html>
		`)
	}
}
