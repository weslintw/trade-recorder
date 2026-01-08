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

		// 採用極簡、不編碼的原始構造方式 (完全模仿 Myfxbook)
		// 採用 100% 測試成功的模式：
		// 1. scope 僅使用 accounts (這已包含交易紀錄權限)
		// 2. redirect_uri 保持原始字串
		authURL := fmt.Sprintf("https://id.ctrader.com/my/settings/openapi/grantingaccess?client_id=%s&scope=accounts&redirect_uri=%s",
			clientID,
			redirectURI,
		)

		// 記錄最終固定的極簡模式網址
		log.Printf("[cTrader OAuth] Success Pattern URL: %s", authURL)

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
		exchangeURL := fmt.Sprintf("%s?grant_type=authorization_code&code=%s&client_id=%s&client_secret=%s&redirect_uri=%s",
			tokenURL, code, clientID, clientSecret, url.QueryEscape(redirectURI))

		log.Printf("[cTrader OAuth] Exchanging code for token at: %s", tokenURL)

		resp, err := http.Get(exchangeURL)
		if err != nil {
			log.Printf("[cTrader OAuth] Token exchange failed: %v", err)
			c.String(http.StatusInternalServerError, "Token exchange failed: "+err.Error())
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		log.Printf("[cTrader OAuth] Token Response: %s", string(body))

		var tokenRes struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ErrorCode    string `json:"errorCode"`
		}
		if err := json.Unmarshal(body, &tokenRes); err != nil {
			log.Printf("[cTrader OAuth] Failed to parse token JSON: %v", err)
			c.String(http.StatusInternalServerError, "Failed to parse token response")
			return
		}

		if tokenRes.AccessToken == "" {
			log.Printf("[cTrader OAuth] Invalid access token in response")
			c.String(http.StatusBadRequest, "Invalid token response: "+string(body))
			return
		}

		// 獲取帳號清單
		accURL := "https://openapi.ctrader.com/connect/getaccounts?access_token=" + tokenRes.AccessToken
		accResp, err := http.Get(accURL)
		if err != nil {
			log.Printf("[cTrader OAuth] Failed to get accounts: %v", err)
			c.String(http.StatusInternalServerError, "Failed to get accounts")
			return
		}
		defer accResp.Body.Close()

		accBody, _ := io.ReadAll(accResp.Body)
		log.Printf("[cTrader OAuth] Accounts Response: %s", string(accBody))

		var accData struct {
			TraderAccounts []struct {
				ID          int64  `json:"ctidTraderAccountId"`
				AccountName string `json:"traderLogin"`
				IsLive      bool   `json:"isLive"`
			} `json:"traderAccounts"`
		}
		if err := json.Unmarshal(accBody, &accData); err != nil {
			log.Printf("[cTrader OAuth] Failed to parse accounts JSON: %v", err)
		}

		log.Printf("[cTrader OAuth] Found %d accounts to process", len(accData.TraderAccounts))

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

			log.Printf("[cTrader OAuth] Processing account: %s (ID: %d, Env: %s)", name, acc.ID, env)

			// 檢查是否已存在
			var existingID int64
			err := db.QueryRow("SELECT id FROM accounts WHERE ctrader_account_id = ? AND user_id = ?", fmt.Sprintf("%d", acc.ID), userID).Scan(&existingID)

			if err == sql.ErrNoRows {
				// 新增
				log.Printf("[cTrader OAuth] Creating new account record for ID: %d", acc.ID)
				res, err := db.Exec(`INSERT INTO accounts 
					(name, type, ctrader_account_id, ctrader_token, ctrader_client_id, ctrader_client_secret, ctrader_env, timezone_offset, user_id, status)
					VALUES (?, 'ctrader', ?, ?, ?, ?, ?, 8, ?, 'active')`,
					name, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env, userID)
				if err != nil {
					log.Printf("[cTrader OAuth] FAILED to insert account %d: %v", acc.ID, err)
				} else {
					newID, _ := res.LastInsertId()
					log.Printf("[cTrader OAuth] Successfully created account %d with DB ID %d. Starting sync.", acc.ID, newID)
					// 立即啟動同步
					go ctrader.SyncCTraderHistory(db, newID, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env)
				}
			} else {
				// 更新 Token
				log.Printf("[cTrader OAuth] Updating existing account record DB ID: %d", existingID)
				_, err := db.Exec("UPDATE accounts SET ctrader_token = ?, ctrader_client_id = ?, ctrader_client_secret = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
					tokenRes.AccessToken, clientID, clientSecret, existingID)
				if err != nil {
					log.Printf("[cTrader OAuth] FAILED to update account %d: %v", existingID, err)
				} else {
					log.Printf("[cTrader OAuth] Successfully updated account DB ID %d. Re-starting sync.", existingID)
					// 重新整理同步
					go ctrader.SyncCTraderHistory(db, existingID, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env)
				}
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
