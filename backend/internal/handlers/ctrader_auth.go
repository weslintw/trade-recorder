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

		// 呼叫 Spotware API 交換 Token (改用符合規範的 POST)
		tokenURL := "https://openapi.ctrader.com/apps/token"
		formData := url.Values{}
		formData.Set("grant_type", "authorization_code")
		formData.Set("code", code)
		formData.Set("client_id", clientID)
		formData.Set("client_secret", clientSecret)
		formData.Set("redirect_uri", redirectURI)

		log.Printf("[cTrader OAuth] POSTing to exchange code for token...")
		resp, err := http.PostForm(tokenURL, formData)
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
			log.Printf("[cTrader OAuth] JSON Unmarshal error: %v", err)
			c.String(http.StatusInternalServerError, "Failed to parse token response")
			return
		}

		if tokenRes.AccessToken == "" {
			log.Printf("[cTrader OAuth] Invalid Token Response (AccessToken empty)")
			c.String(http.StatusBadRequest, "Invalid token response: "+string(body))
			return
		}

		// 獲取帳號清單 - 根據官方文件，REST 介面已不再保險，
		// 我們改用 WebSocket 手機握手來抓取帳號清單 (Discovery)
		accList, err := ctrader.GetAccountListByToken(clientID, clientSecret, tokenRes.AccessToken)
		if err != nil {
			log.Printf("[cTrader OAuth] WebSocket Discovery failed: %v", err)
			c.String(http.StatusInternalServerError, "抓取帳號列表失敗: "+err.Error())
			return
		}

		// 獲取目前使用者的 ID (暫時預設為 1)
		userID := int64(1)

		// 統計處理數量
		processedCount := 0
		for _, acc := range accList {
			env := "live"
			if !acc.IsLive {
				env = "demo"
			}
			name := fmt.Sprintf("cTrader %d (%s)", acc.ID, env)

			log.Printf("[cTrader OAuth] Saving/Updating account: %s", name)

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
					processedCount++
					log.Printf("[cTrader OAuth] Successfully created account %d with DB ID %d. Starting sync.", acc.ID, newID)
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
					processedCount++
					log.Printf("[cTrader OAuth] Successfully updated account DB ID %d. Re-starting sync.", existingID)
					go ctrader.SyncCTraderHistory(db, existingID, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env)
				}
			}
		}

		log.Printf("[cTrader OAuth] Successfully processed %d accounts", processedCount)

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
