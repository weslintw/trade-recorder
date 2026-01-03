package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trade-journal/internal/ctrader"
	"trade-journal/internal/models"
	"trade-journal/internal/mt5"

	"github.com/gin-gonic/gin"
)

// GetAccounts 取得所有帳號
func GetAccounts(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		query := `
			SELECT 
				id, name, type, COALESCE(mt5_account_id, ''), COALESCE(mt5_token, ''), 
				COALESCE(ctrader_account_id, ''), COALESCE(ctrader_token, ''),
				COALESCE(ctrader_client_id, ''), COALESCE(ctrader_client_secret, ''),
				COALESCE(ctrader_env, 'live'),
				status, 
				COALESCE(timezone_offset, 8), COALESCE(sync_status, 'idle'), last_synced_at, 
				COALESCE(last_sync_error, ''), created_at, updated_at,
				(
					SELECT COALESCE(SUM(
						LENGTH(COALESCE(entry_strategy_image, '')) +
						LENGTH(COALESCE(entry_strategy_image_original, '')) +
						LENGTH(COALESCE(legend_king_image, '')) +
						LENGTH(COALESCE(legend_king_image_original, '')) +
						LENGTH(COALESCE(legend_htf_image, '')) +
						LENGTH(COALESCE(legend_htf_image_original, '')) +
						LENGTH(COALESCE(entry_signals, '')) +
						LENGTH(COALESCE(entry_checklist, '')) +
						LENGTH(COALESCE(trend_analysis, '')) +
						LENGTH(COALESCE(notes, '')) +
						LENGTH(COALESCE(entry_reason, '')) +
						LENGTH(COALESCE(exit_reason, ''))
					), 0) FROM trades WHERE account_id = a.id
				) + (
					SELECT COALESCE(SUM(LENGTH(COALESCE(image_path, ''))), 0) 
					FROM trade_images 
					WHERE trade_id IN (SELECT id FROM trades WHERE account_id = a.id)
				) + (
					SELECT COALESCE(SUM(
						LENGTH(COALESCE(notes, '')) +
						LENGTH(COALESCE(trend_analysis, ''))
					), 0) FROM daily_plans WHERE account_id = a.id
				) AS storage_usage
			FROM accounts a 
			WHERE user_id = ? 
			ORDER BY created_at ASC`

		rows, err := db.Query(query, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var accounts = []models.Account{}
		for rows.Next() {
			var acc models.Account
			err := rows.Scan(
				&acc.ID, &acc.Name, &acc.Type, &acc.MT5AccountID, &acc.MT5Token, 
				&acc.CTraderAccountID, &acc.CTraderToken,
				&acc.CTraderClientID, &acc.CTraderClientSecret,
				&acc.CTraderEnv,
				&acc.Status, 
				&acc.TimezoneOffset, &acc.SyncStatus, &acc.LastSyncedAt, &acc.LastSyncError, 
				&acc.CreatedAt, &acc.UpdatedAt, &acc.StorageUsage,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			accounts = append(accounts, acc)
		}

		c.JSON(http.StatusOK, accounts)
	}
}

// CreateAccount 建立帳號
func CreateAccount(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.AccountCreate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt64("user_id")
		res, err := db.Exec("INSERT INTO accounts (name, type, mt5_account_id, mt5_token, ctrader_account_id, ctrader_token, ctrader_client_id, ctrader_client_secret, ctrader_env, timezone_offset, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			req.Name, req.Type, req.MT5AccountID, req.MT5Token, req.CTraderAccountID, req.CTraderToken, req.CTraderClientID, req.CTraderClientSecret, req.CTraderEnv, req.TimezoneOffset, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		id, _ := res.LastInsertId()

		// 如果是 MetaTrader 帳號，觸發同步 (目前 placeholder)
		if req.Type == "metatrader" {
			go mt5.SyncMT5History(db, id, req.MT5AccountID, req.MT5Token)
		} else if req.Type == "ctrader" {
			go ctrader.SyncCTraderHistory(db, id, req.CTraderAccountID, req.CTraderToken, req.CTraderClientID, req.CTraderClientSecret, req.CTraderEnv)
		}

		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "帳號建立成功"})
	}
}

// UpdateAccount 更新帳號
func UpdateAccount(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req models.AccountUpdate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt64("user_id")
		// 這裡為了簡化先做全量更新，實際上應該檢查 nil
		res, err := db.Exec("UPDATE accounts SET name = COALESCE(?, name), mt5_account_id = COALESCE(?, mt5_account_id), mt5_token = COALESCE(?, mt5_token), ctrader_account_id = COALESCE(?, ctrader_account_id), ctrader_token = COALESCE(?, ctrader_token), ctrader_client_id = COALESCE(?, ctrader_client_id), ctrader_client_secret = COALESCE(?, ctrader_client_secret), ctrader_env = COALESCE(?, ctrader_env), timezone_offset = COALESCE(?, timezone_offset), updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?",
			req.Name, req.MT5AccountID, req.MT5Token, req.CTraderAccountID, req.CTraderToken, req.CTraderClientID, req.CTraderClientSecret, req.CTraderEnv, req.TimezoneOffset, id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到該帳號或無權限更新"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "帳號更新成功"})
	}
}

// DeleteAccount 刪除帳號
func DeleteAccount(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		res, err := db.Exec("DELETE FROM accounts WHERE id = ? AND user_id = ?", id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到該帳號或無權限刪除"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "帳號已刪除"})
	}
}

// SyncAccountHistory 手動觸發帳號同步
func SyncAccountHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		var acc models.Account
		err := db.QueryRow("SELECT id, type, COALESCE(mt5_account_id, ''), COALESCE(mt5_token, ''), COALESCE(ctrader_account_id, ''), COALESCE(ctrader_token, ''), COALESCE(ctrader_client_id, ''), COALESCE(ctrader_client_secret, ''), COALESCE(ctrader_env, 'live') FROM accounts WHERE id = ? AND user_id = ?", id, userID).
			Scan(&acc.ID, &acc.Type, &acc.MT5AccountID, &acc.MT5Token, &acc.CTraderAccountID, &acc.CTraderToken, &acc.CTraderClientID, &acc.CTraderClientSecret, &acc.CTraderEnv)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到該帳號"})
			return
		}

		if acc.Type != "metatrader" && acc.Type != "ctrader" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只有 MetaTrader 或 cTrader 帳號可以同步"})
			return
		}

		// 執行同步
		db.Exec("UPDATE accounts SET sync_status = 'syncing', updated_at = CURRENT_TIMESTAMP WHERE id = ?", acc.ID)
		if acc.Type == "metatrader" {
			go mt5.SyncMT5History(db, acc.ID, acc.MT5AccountID, acc.MT5Token)
		} else if acc.Type == "ctrader" {
			go ctrader.SyncCTraderHistory(db, acc.ID, acc.CTraderAccountID, acc.CTraderToken, acc.CTraderClientID, acc.CTraderClientSecret, acc.CTraderEnv)
		}

		c.JSON(http.StatusOK, gin.H{"message": "同步指令已發送，這可能需要一點時間。"})
	}
}

// parseTradeTime 嘗試多種可能的日期格式
func parseTradeTime(timeStr string) (time.Time, error) {
	layouts := []string{
		"2006/01/02 15:04",
		"2006/01/02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"02.01.2006 15:04",
		"02.01.2006 15:04:05",
		time.RFC3339,
	}

	timeStr = strings.TrimSpace(timeStr)
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, timeStr, time.Local)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("無法解析時間字串: %s", timeStr)
}

// determineMarketSession 根據時間判斷市場時段 (與前端邏輯保持一致)
func determineMarketSession(t time.Time) string {
	// 轉為 GMT+8 (台灣/香港時間) 進行判斷
	loc := time.FixedZone("GMT", 8*3600)
	t8 := t.In(loc)

	hour := t8.Hour()
	minute := t8.Minute()
	timeInMinutes := hour*60 + minute

	// 判斷是否為夏令時間 (3月~11月) - 簡單判斷
	month := t8.Month()
	isDST := month >= 3 && month <= 11

	// 亞盤：08:00 - 15:00
	if timeInMinutes >= 8*60 && timeInMinutes < 15*60 {
		return "asian"
	}

	// 歐盤
	var euroStart, euroEnd int
	if isDST {
		euroStart = 15 * 60
		euroEnd = 23 * 60
	} else {
		euroStart = 16 * 60
		euroEnd = 24 * 60
	}
	if timeInMinutes >= euroStart && (timeInMinutes < euroEnd || euroEnd == 24*60) {
		return "european"
	}

	// 美盤 (處理跨日)
	var usStart, usEnd int
	if isDST {
		usStart = 20 * 60
		usEnd = 4 * 60
	} else {
		usStart = 21 * 60
		usEnd = 5 * 60
	}

	if usStart > usEnd { // 跨日
		if timeInMinutes >= usStart || timeInMinutes < usEnd {
			return "us"
		}
	} else {
		if timeInMinutes >= usStart && timeInMinutes < usEnd {
			return "us"
		}
	}

	return "asian" // 預設
}

// ImportTradesCSV 從 CSV 匯入交易紀錄 (支援 FTMO 格式)
func ImportTradesCSV(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountIDStr := c.Param("id")
		accountID, _ := strconv.ParseInt(accountIDStr, 10, 64)

		source := c.PostForm("source")
		if source == "" {
			source = "ftmo" // 預設為 ftmo
		}

		if source != "ftmo" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目前僅支援 FTMO 格式匯入"})
			return
		}

		userID := c.GetInt64("user_id")
		// 檢查帳號所屬權
		var exists int
		db.QueryRow("SELECT 1 FROM accounts WHERE id = ? AND user_id = ?", accountID, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限操作此帳號"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "請上傳檔案"})
			return
		}

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法讀取檔案"})
			return
		}
		defer f.Close()

		reader := csv.NewReader(f)
		records, err := reader.ReadAll()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析 CSV 失敗: " + err.Error()})
			return
		}

		if len(records) < 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "CSV 檔案格式不正確或無資料"})
			return
		}

		// 檢查標題列 (FTMO 第一欄通常是 Ticket)
		if records[0][0] != "Ticket" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支援的 CSV 格式，目前僅支援 FTMO 格式"})
			return
		}

		var importedTickets []string
		var duplicateTickets []string
		var errorTickets []string

		for i, row := range records {
			if i == 0 {
				continue // 跳過標頭
			}

			// 列數檢查
			if len(row) < 14 {
				errorTickets = append(errorTickets, "Unknown (Row "+strconv.Itoa(i)+")")
				continue
			}

			// 解析欄位 (索引根據 FTMO 格式)
			ticket := row[0]
			openTimeStr := row[1]
			sideStr := row[2] // buy/sell
			volumeStr := row[3]
			symbol := row[4]
			entryPriceStr := row[5]
			slPriceStr := row[6]
			closeTimeStr := row[8]
			exitPriceStr := row[9]
			swapStr := row[10]
			commissionStr := row[11]
			profitStr := row[12]

			// 解析時間
			openTime, err := parseTradeTime(openTimeStr)
			if err != nil {
				log.Printf("Parse openTime error for ticket %s: %v", ticket, err)
				errorTickets = append(errorTickets, ticket)
				continue
			}

			closeTime, err := parseTradeTime(closeTimeStr)
			if err != nil {
				// 如果關倉時間解析失敗，可能尚未平倉？但在 FTMO 導出中通常都有
				log.Printf("Parse closeTime error for ticket %s: %v", ticket, err)
			}

			// 解析數值
			volume, _ := strconv.ParseFloat(volumeStr, 64)
			entryPrice, _ := strconv.ParseFloat(entryPriceStr, 64)
			exitPrice, _ := strconv.ParseFloat(exitPriceStr, 64)
			exitSl, _ := strconv.ParseFloat(slPriceStr, 64)
			swap, _ := strconv.ParseFloat(swapStr, 64)
			commission, _ := strconv.ParseFloat(commissionStr, 64)
			profit, _ := strconv.ParseFloat(profitStr, 64)

			side := "long"
			if sideStr == "sell" {
				side = "short"
			}

			totalPnL := profit + swap + commission

			// 計算合約乘數
			multiplier := 100.0 // 預設 (黃金 XAUUSD: $1 = 100點, 指數: 1.0 = 100點)
			symbolUpper := strings.ToUpper(symbol)
			if strings.Contains(symbolUpper, "JPY") {
				multiplier = 1000.0 // JPY 貨幣對 (0.001 = 1點)
			} else if strings.Contains(symbolUpper, "EUR") || strings.Contains(symbolUpper, "GBP") || strings.Contains(symbolUpper, "AUD") || (strings.Contains(symbolUpper, "USD") && !strings.Contains(symbolUpper, "XAU")) {
				multiplier = 100000.0 // 預設外匯 (0.00001 = 1點)
			}

			// 重新計算盈虧點數 (根據使用者定義：1點 = 最小價格單位)
			diff := exitPrice - entryPrice
			if side == "short" {
				diff = entryPrice - exitPrice
			}
			calculatedPips := math.Round(diff*multiplier*100) / 100

			// 計算子彈大小與風報比 (CSV 匯入暫時不提供初始 SL，因此不計算)
			var bulletSize interface{} = nil
			var rrRatio interface{} = nil
			var initialSl interface{} = nil

			// 自動判斷時段
			marketSession := determineMarketSession(openTime)

			// 去重檢查 (優先使用 Ticket)
			var exists bool
			if ticket != "" {
				err = db.QueryRow(`
					SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)
				`, accountID, ticket).Scan(&exists)
			} else {
				// 如果沒有 Ticket，才使用 entry_time + lot_size
				err = db.QueryRow(`
					SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND symbol = ? AND entry_time = ? AND lot_size = ?)
				`, accountID, symbol, openTime, volume).Scan(&exists)
			}

			if exists {
				duplicateTickets = append(duplicateTickets, ticket)
				continue
			}

			// 寫入資料庫
			_, err = db.Exec(`
				INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, pnl_points, entry_time, exit_time, trade_type, notes, timezone_offset, market_session, initial_sl, bullet_size, rr_ratio, ticket, exit_sl)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, accountID, symbol, side, entryPrice, exitPrice, volume, totalPnL, calculatedPips, openTime, closeTime, "actual", "FTMO CSV 匯入: Ticket "+ticket, 8, marketSession, initialSl, bulletSize, rrRatio, ticket, exitSl)

			if err != nil {
				log.Printf("Import failed for ticket %s: %v", ticket, err)
				errorTickets = append(errorTickets, ticket)
			} else {
				importedTickets = append(importedTickets, ticket)
			}
		}

		message := fmt.Sprintf("匯入完成：成功 %d 筆", len(importedTickets))
		if len(duplicateTickets) > 0 || len(errorTickets) > 0 {
			message += " (跳過："
			if len(duplicateTickets) > 0 {
				message += fmt.Sprintf("重複 %d 筆 ", len(duplicateTickets))
			}
			if len(errorTickets) > 0 {
				message += fmt.Sprintf("錯誤 %d 筆", len(errorTickets))
			}
			message += ")"
		}

		c.JSON(http.StatusOK, gin.H{
			"message":           message,
			"imported_count":    len(importedTickets),
			"duplicate_count":   len(duplicateTickets),
			"error_count":       len(errorTickets),
			"imported_tickets":  importedTickets,
			"duplicate_tickets": duplicateTickets,
			"error_tickets":     errorTickets,
		})
	}
}

// ClearAccountData 清除帳號的所有交易紀錄與規劃
func ClearAccountData(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		// 檢查帳號所屬權
		var exists int
		db.QueryRow("SELECT 1 FROM accounts WHERE id = ? AND user_id = ?", id, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限操作此帳號"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		// 刪除交易紀錄（這也會透過 CASCADE 刪除相關圖片與標籤）
		_, err = tx.Exec("DELETE FROM trades WHERE account_id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清除交易紀錄失敗: " + err.Error()})
			return
		}

		// 刪除每日規劃
		_, err = tx.Exec("DELETE FROM daily_plans WHERE account_id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清除每日規劃失敗: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "帳號資料已完成清除"})
	}
}
