package handlers

import (
	"context"
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
	"trade-journal/internal/database"
	"trade-journal/internal/models"
	"trade-journal/internal/mt5"
	"trade-journal/internal/myfxbook"

	"github.com/gin-gonic/gin"
)

// GetAccounts 取得所有帳號
func GetAccounts(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		userID := c.GetInt64("user_id")

		// 建立具備 5 秒超時的 Context
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		query := `
			SELECT 
				id, name, type, COALESCE(mt5_account_id, ''), COALESCE(mt5_token, ''), 
				COALESCE(ctrader_account_id, ''), COALESCE(ctrader_token, ''),
				COALESCE(ctrader_client_id, ''), COALESCE(ctrader_client_secret, ''),
				COALESCE(ctrader_env, 'live'),
				COALESCE(myfxbook_email, ''), COALESCE(myfxbook_password, ''), COALESCE(myfxbook_account_id, ''),
				status, 
				COALESCE(timezone_offset, 8), COALESCE(sync_status, 'idle'), last_synced_at, 
				COALESCE(last_sync_error, ''), created_at, updated_at,
				COALESCE(a.storage_usage, 0) AS storage_usage
			FROM accounts a 
			WHERE user_id = ? 
			ORDER BY created_at ASC`

		rows, err := db.QueryContext(ctx, query, userID)
		if err != nil {
			if err == context.DeadlineExceeded {
				log.Printf("[GetAccounts] TIMEOUT after 5 seconds")
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "資料庫忙碌中，請稍後再試"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var accounts = []models.Account{}
		for rows.Next() {
			// 一樣加入中斷檢查
			select {
			case <-ctx.Done():
				return
			default:
			}

			var acc models.Account
			err := rows.Scan(
				&acc.ID, &acc.Name, &acc.Type, &acc.MT5AccountID, &acc.MT5Token,
				&acc.CTraderAccountID, &acc.CTraderToken,
				&acc.CTraderClientID, &acc.CTraderClientSecret,
				&acc.CTraderEnv,
				&acc.MyfxbookEmail, &acc.MyfxbookPassword, &acc.MyfxbookAccountID,
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

		duration := time.Since(startTime)
		if duration > 1*time.Second {
			log.Printf("[GetAccounts PERF] SLOW query: %v, count: %d", duration, len(accounts))
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
		res, err := db.Exec("INSERT INTO accounts (name, type, mt5_account_id, mt5_token, ctrader_account_id, ctrader_token, ctrader_client_id, ctrader_client_secret, ctrader_env, myfxbook_email, myfxbook_password, myfxbook_account_id, timezone_offset, user_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			req.Name, req.Type, req.MT5AccountID, req.MT5Token, req.CTraderAccountID, req.CTraderToken, req.CTraderClientID, req.CTraderClientSecret, req.CTraderEnv, req.MyfxbookEmail, req.MyfxbookPassword, req.MyfxbookAccountID, req.TimezoneOffset, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		id, _ := res.LastInsertId()

		// 如果是 MetaTrader 帳號，觸發同步 (目前 placeholder)
		if req.Type == "metatrader" {
			go mt5.SyncMT5History(db, id, req.MT5AccountID, req.MT5Token)
		} else if req.Type == "ctrader" {
			var fromTimestamp int64
			if req.SyncAll {
				fromTimestamp = time.Now().AddDate(-20, 0, 0).UnixMilli()
			} else if req.FromDate != "" {
				t, err := time.Parse("2006-01-02", req.FromDate)
				if err == nil {
					fromTimestamp = t.UnixMilli()
				}
			}

			if fromTimestamp == 0 {
				// 預設同步 120 天
				fromTimestamp = time.Now().AddDate(0, 0, -120).UnixMilli()
			}

			go ctrader.SyncCTraderHistory(db, id, req.CTraderAccountID, req.CTraderToken, req.CTraderClientID, req.CTraderClientSecret, req.CTraderEnv, fromTimestamp)
		} else if req.Type == "myfxbook" {
			go myfxbook.SyncMyfxbookHistory(db, id, req.MyfxbookEmail, req.MyfxbookPassword, req.MyfxbookAccountID)
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
		res, err := db.Exec("UPDATE accounts SET name = COALESCE(?, name), mt5_account_id = COALESCE(?, mt5_account_id), mt5_token = COALESCE(?, mt5_token), ctrader_account_id = COALESCE(?, ctrader_account_id), ctrader_token = COALESCE(?, ctrader_token), ctrader_client_id = COALESCE(?, ctrader_client_id), ctrader_client_secret = COALESCE(?, ctrader_client_secret), ctrader_env = COALESCE(?, ctrader_env), myfxbook_email = COALESCE(?, myfxbook_email), myfxbook_password = COALESCE(?, myfxbook_password), myfxbook_account_id = COALESCE(?, myfxbook_account_id), timezone_offset = COALESCE(?, timezone_offset), updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?",
			req.Name, req.MT5AccountID, req.MT5Token, req.CTraderAccountID, req.CTraderToken, req.CTraderClientID, req.CTraderClientSecret, req.CTraderEnv, req.MyfxbookEmail, req.MyfxbookPassword, req.MyfxbookAccountID, req.TimezoneOffset, id, userID)
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
		err := db.QueryRow("SELECT id, type, COALESCE(mt5_account_id, ''), COALESCE(mt5_token, ''), COALESCE(ctrader_account_id, ''), COALESCE(ctrader_token, ''), COALESCE(ctrader_client_id, ''), COALESCE(ctrader_client_secret, ''), COALESCE(ctrader_env, 'live'), COALESCE(myfxbook_email, ''), COALESCE(myfxbook_password, ''), COALESCE(myfxbook_account_id, '') FROM accounts WHERE id = ? AND user_id = ?", id, userID).
			Scan(&acc.ID, &acc.Type, &acc.MT5AccountID, &acc.MT5Token, &acc.CTraderAccountID, &acc.CTraderToken, &acc.CTraderClientID, &acc.CTraderClientSecret, &acc.CTraderEnv, &acc.MyfxbookEmail, &acc.MyfxbookPassword, &acc.MyfxbookAccountID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到該帳號"})
			return
		}

		if acc.Type != "metatrader" && acc.Type != "ctrader" && acc.Type != "myfxbook" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只有 MetaTrader, cTrader 或 Myfxbook 帳號可以同步"})
			return
		}

		// 解析選配的同步參數
		var syncReq struct {
			FromDate string `json:"from_date"`
			SyncAll  bool   `json:"sync_all"`
		}
		c.ShouldBindJSON(&syncReq) // 忽略錯誤，可能是舊版前端或無參數

		var fromTimestamp int64
		if syncReq.SyncAll {
			// 同步全部：設定為 20 年前
			fromTimestamp = time.Now().AddDate(-20, 0, 0).UnixMilli()
		} else if syncReq.FromDate != "" {
			t, err := time.Parse("2006-01-02", syncReq.FromDate)
			if err == nil {
				fromTimestamp = t.UnixMilli()
			}
		}

		if fromTimestamp == 0 {
			// 預設同步 120 天
			fromTimestamp = time.Now().AddDate(0, 0, -120).UnixMilli()
		}

		// 執行同步
		db.Exec("UPDATE accounts SET sync_status = 'syncing', updated_at = CURRENT_TIMESTAMP WHERE id = ?", acc.ID)
		if acc.Type == "metatrader" {
			go mt5.SyncMT5History(db, acc.ID, acc.MT5AccountID, acc.MT5Token)
		} else if acc.Type == "ctrader" {
			go ctrader.SyncCTraderHistory(db, acc.ID, acc.CTraderAccountID, acc.CTraderToken, acc.CTraderClientID, acc.CTraderClientSecret, acc.CTraderEnv, fromTimestamp)
		} else if acc.Type == "myfxbook" {
			go myfxbook.SyncMyfxbookHistory(db, acc.ID, acc.MyfxbookEmail, acc.MyfxbookPassword, acc.MyfxbookAccountID)
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
		"01/02/2006 15:04",
		"01/02/2006 15:04:05",
		"01/02/2006",
		"2006/01/02",
		time.RFC3339,
	}

	timeStr = strings.TrimSpace(timeStr)
	for _, layout := range layouts {
		t, err := time.ParseInLocation(layout, timeStr, time.UTC)
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

		if source != "ftmo" && source != "myfxbook" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目前僅支援 FTMO 與 Myfxbook 格式匯入"})
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

		// 檢查標題列
		if source == "ftmo" && records[0][0] != "Ticket" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支援的 CSV 格式，該檔案不符合 FTMO 格式"})
			return
		} else if source == "myfxbook" && records[0][0] != "Tags" && records[0][0] != "Symbol" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "不支援的 CSV 格式，該檔案不符合 Myfxbook 格式"})
			return
		}

		var importedTickets []string
		var duplicateTickets []string
		var errorTickets []string

		for i, row := range records {
			if i == 0 {
				continue // 跳過標頭
			}

			var ticket, openTimeStr, closeTimeStr, symbol, sideStr, volumeStr, entryPriceStr, exitPriceStr, slPriceStr, swapStr, commissionStr, profitStr, pipsStr, comment string

			if source == "ftmo" {
				if len(row) < 14 {
					errorTickets = append(errorTickets, "Unknown (Row "+strconv.Itoa(i)+")")
					continue
				}
				ticket = row[0]
				openTimeStr = row[1]
				sideStr = row[2]
				volumeStr = row[3]
				symbol = row[4]
				entryPriceStr = row[5]
				slPriceStr = row[6]
				closeTimeStr = row[8]
				exitPriceStr = row[9]
				swapStr = row[10]
				commissionStr = row[11]
				profitStr = row[12]
				comment = "FTMO CSV 匯入: Ticket " + ticket
			} else if source == "myfxbook" {
				if len(row) < 15 {
					errorTickets = append(errorTickets, "Unknown (Row "+strconv.Itoa(i)+")")
					continue
				}
				// Myfxbook format based on screenshot:
				// 0: Tags, 1: Ticket, 2: Open Date, 3: Close Date, 4: Symbol, 5: Action, 6: Units/Lots, 7: SL, 8: TP, 9: Open Price, 10: Close Price, 11: Commission, 12: Swap, 13: Pips, 14: Profit, 15: Gain, 16: Comment
				ticket = row[1]
				openTimeStr = row[2]
				closeTimeStr = row[3]
				symbol = row[4]
				sideStr = row[5]
				volumeStr = row[6]
				slPriceStr = row[7]
				// TP is row[8]
				entryPriceStr = row[9]
				exitPriceStr = row[10]
				commissionStr = row[11]
				swapStr = row[12]
				pipsStr = row[13]
				profitStr = row[14]
				if len(row) > 16 {
					comment = row[16]
				}
				if comment == "" {
					comment = "Myfxbook CSV 匯入"
				}
				if ticket != "" {
					comment += " (Ticket: " + ticket + ")"
				}
			}

			// 解析時間
			openTime, err := parseTradeTime(openTimeStr)
			if err != nil {
				log.Printf("Parse openTime error: %v", err)
				errorTickets = append(errorTickets, "Row "+strconv.Itoa(i))
				continue
			}

			closeTime, err := parseTradeTime(closeTimeStr)

			// 解析數值
			volume, _ := strconv.ParseFloat(volumeStr, 64)
			entryPrice, _ := strconv.ParseFloat(entryPriceStr, 64)
			exitPrice, _ := strconv.ParseFloat(exitPriceStr, 64)
			exitSl, _ := strconv.ParseFloat(slPriceStr, 64)
			swap, _ := strconv.ParseFloat(swapStr, 64)
			commission, _ := strconv.ParseFloat(commissionStr, 64)
			profit, _ := strconv.ParseFloat(profitStr, 64)
			pips, _ := strconv.ParseFloat(pipsStr, 64)

			side := "long"
			sideLower := strings.ToLower(sideStr)
			if sideLower == "sell" || sideLower == "short" {
				side = "short"
			}

			totalPnL := profit + swap + commission

			var calculatedPips float64
			if source == "myfxbook" {
				calculatedPips = pips
			} else {
				// FTMO 需要計算
				multiplier := 100.0
				symbolUpper := strings.ToUpper(symbol)
				if strings.Contains(symbolUpper, "JPY") {
					multiplier = 1000.0
				} else if strings.Contains(symbolUpper, "EUR") || strings.Contains(symbolUpper, "GBP") || strings.Contains(symbolUpper, "AUD") || (strings.Contains(symbolUpper, "USD") && !strings.Contains(symbolUpper, "XAU")) {
					multiplier = 100000.0
				}
				diff := exitPrice - entryPrice
				if side == "short" {
					diff = entryPrice - exitPrice
				}
				calculatedPips = math.Round(diff*multiplier*100) / 100
			}

			// 去重檢查
			var existingID int64
			// 同時檢查 Ticket 與特徵 (Symbol, EntryTime, Volume)
			// 如果 Ticket 存在，優先檢查 Ticket；但如果 API 先同步了，該紀錄可能沒有 Ticket，所以也要檢查特徵
			checkQuery := `
				SELECT id FROM trades 
				WHERE account_id = ? AND (
					(ticket != '' AND ticket = ?) OR 
					(symbol = ? AND entry_time = ? AND lot_size = ?)
				) LIMIT 1
			`
			err = db.QueryRow(checkQuery, accountID, ticket, symbol, openTime, volume).Scan(&existingID)

			if err == nil {
				// 已存在相似紀錄，如果是 API 同步產生的（無 Ticket），則補上 Ticket
				if ticket != "" {
					db.Exec("UPDATE trades SET ticket = ? WHERE id = ? AND (ticket IS NULL OR ticket = '')", ticket, existingID)
				}
				duplicateTickets = append(duplicateTickets, "Row "+strconv.Itoa(i))
				continue
			}

			// 自動判斷時段
			marketSession := determineMarketSession(openTime)

			// 寫入資料庫
			_, err = db.Exec(`
				INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, pnl_points, entry_time, exit_time, trade_type, notes, timezone_offset, market_session, ticket, exit_sl)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, accountID, symbol, side, entryPrice, exitPrice, volume, totalPnL, calculatedPips, openTime, closeTime, "actual", comment, 8, marketSession, ticket, exitSl)

			if err != nil {
				log.Printf("Import failed: %v", err)
				errorTickets = append(errorTickets, "Row "+strconv.Itoa(i))
			} else {
				importedTickets = append(importedTickets, "Row "+strconv.Itoa(i))
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

		// 更新儲存空間佔用
		database.UpdateAccountStorageUsage(db, accountID)

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

		// 清除已刪除票據紀錄 (重設同步黑名單)
		_, err = tx.Exec("DELETE FROM deleted_tickets WHERE account_id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清除票據紀錄失敗: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 更新儲存空間佔用 (重設為 0)
		accountID, _ := strconv.ParseInt(id, 10, 64)
		database.UpdateAccountStorageUsage(db, accountID)

		c.JSON(http.StatusOK, gin.H{"message": "帳號資料已完成清除"})
	}
}
