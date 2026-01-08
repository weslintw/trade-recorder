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
	"strconv"
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

		userID := c.GetInt64("user_id")
		if userID == 0 {
			userID = 1 // 備用機制
		}

		// 採用 100% 測試成功的模式：
		// 1. scope 僅使用 accounts (這已包含交易紀錄權限)
		// 2. redirect_uri 保持原始字串
		// 3. 增加 state 參數來傳遞當前 UserID，確保回調時能對應回正確使用者
		authURL := fmt.Sprintf("https://id.ctrader.com/my/settings/openapi/grantingaccess?client_id=%s&scope=accounts&redirect_uri=%s&state=%d",
			clientID,
			redirectURI,
			userID,
		)

		// 記錄最終固定的極簡模式網址
		log.Printf("[cTrader OAuth] Success Pattern URL for User %d: %s", userID, authURL)

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

		// 獲取目前使用者的 ID (從 state 參數獲取)
		state := c.Query("state")
		userID, _ := strconv.ParseInt(state, 10, 64)
		if userID == 0 {
			userID = 1 // 備用機制
			log.Printf("[cTrader OAuth] Warning: No userID in state, falling back to 1")
		} else {
			log.Printf("[cTrader OAuth] Link accounts to UserID: %d", userID)
		}

		// 第一個處理成功的帳號 ID (用於回傳前端自動選取)
		var firstAccountID int64
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
					if firstAccountID == 0 {
						firstAccountID = newID
					}
					processedCount++
					log.Printf("[cTrader OAuth] Successfully created account %d with DB ID %d. Starting sync.", acc.ID, newID)
					go ctrader.SyncCTraderHistory(db, newID, fmt.Sprintf("%d", acc.ID), tokenRes.AccessToken, clientID, clientSecret, env)
				}
			} else {
				// 更新 Token
				if firstAccountID == 0 {
					firstAccountID = existingID
				}
				log.Printf("[cTrader OAuth] Updating existing account record DB ID: %d", existingID)
				_, err := db.Exec("UPDATE accounts SET name = ?, ctrader_token = ?, ctrader_client_id = ?, ctrader_client_secret = ?, ctrader_env = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?",
					name, tokenRes.AccessToken, clientID, clientSecret, env, existingID)
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
		c.String(http.StatusOK, fmt.Sprintf(`
			<html>
				<body>
					<div style="font-family: sans-serif; text-align: center; padding: 50px;">
						<h2 style="color: #10b981;">cTrader 帳號連線成功！</h2>
						<p>正在同步您的交易資料，本視窗即將關閉...</p>
					</div>
					<script>
						// 將新建立的帳號 ID 傳回給母視窗
						if (window.opener) {
							window.opener.postMessage({ 
								type: 'CTRADER_AUTH_SUCCESS', 
								accountId: %d || null
							}, '*');
						}
						
						setTimeout(() => {
							window.close();
						}, 2000);
						
						// 如果是手機或無法關閉，則跳轉
						setTimeout(() => {
							window.location.href = '/';
						}, 4000);
					</script>
				</body>
			</html>
		`, firstAccountID))
	}
}
