package handlers

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trade-journal/internal/models"
	"trade-journal/internal/mt5"

	"github.com/gin-gonic/gin"
)

// GetAccounts 取得所有帳號
func GetAccounts(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, type, COALESCE(mt5_account_id, ''), COALESCE(mt5_token, ''), status, COALESCE(sync_status, 'idle'), last_synced_at, COALESCE(last_sync_error, ''), created_at, updated_at FROM accounts ORDER BY created_at ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var accounts []models.Account
		for rows.Next() {
			var acc models.Account
			err := rows.Scan(&acc.ID, &acc.Name, &acc.Type, &acc.MT5AccountID, &acc.MT5Token, &acc.Status, &acc.SyncStatus, &acc.LastSyncedAt, &acc.LastSyncError, &acc.CreatedAt, &acc.UpdatedAt)
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

		res, err := db.Exec("INSERT INTO accounts (name, type, mt5_account_id, mt5_token) VALUES (?, ?, ?, ?)",
			req.Name, req.Type, req.MT5AccountID, req.MT5Token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		id, _ := res.LastInsertId()

		// 如果是 MetaTrader 帳號，觸發同步 (目前 placeholder)
		if req.Type == "metatrader" {
			go mt5.SyncMT5History(db, id, req.MT5AccountID, req.MT5Token)
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

		// 這裡為了簡化先做全量更新，實際上應該檢查 nil
		_, err := db.Exec("UPDATE accounts SET name = COALESCE(?, name), mt5_account_id = COALESCE(?, mt5_account_id), mt5_token = COALESCE(?, mt5_token), updated_at = CURRENT_TIMESTAMP WHERE id = ?",
			req.Name, req.MT5AccountID, req.MT5Token, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "帳號更新成功"})
	}
}

// DeleteAccount 刪除帳號
func DeleteAccount(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// 不允許刪除 ID 為 1 的預設帳號
		if id == "1" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無法刪除預設帳號"})
			return
		}

		_, err := db.Exec("DELETE FROM accounts WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "帳號已刪除"})
	}
}

// SyncAccountHistory 手動觸發帳號同步
func SyncAccountHistory(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var acc models.Account
		err := db.QueryRow("SELECT id, type, COALESCE(mt5_account_id, ''), COALESCE(mt5_token, '') FROM accounts WHERE id = ?", id).
			Scan(&acc.ID, &acc.Type, &acc.MT5AccountID, &acc.MT5Token)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到該帳號"})
			return
		}

		if acc.Type != "metatrader" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "只有 MetaTrader 帳號可以同步"})
			return
		}

		// 執行同步
		db.Exec("UPDATE accounts SET sync_status = 'syncing', updated_at = CURRENT_TIMESTAMP WHERE id = ?", acc.ID)
		go mt5.SyncMT5History(db, acc.ID, acc.MT5AccountID, acc.MT5Token)

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

// ImportTradesCSV 從 CSV 匯入交易紀錄 (支援 FTMO 格式)
func ImportTradesCSV(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountIDStr := c.Param("id")
		accountID, _ := strconv.ParseInt(accountIDStr, 10, 64)

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

		var importedCount int
		var skippedCount int

		for i, row := range records {
			if i == 0 {
				continue // 跳過標頭
			}

			// 列數檢查
			if len(row) < 14 {
				continue
			}

			// 解析欄位 (索引根據 FTMO 格式)
			ticket := row[0]
			openTimeStr := row[1]
			sideStr := row[2] // buy/sell
			volumeStr := row[3]
			symbol := row[4]
			entryPriceStr := row[5]
			closeTimeStr := row[8]
			exitPriceStr := row[9]
			swapStr := row[10]
			commissionStr := row[11]
			profitStr := row[12]
			pipsStr := row[13]

			// 解析時間
			openTime, err := parseTradeTime(openTimeStr)
			if err != nil {
				log.Printf("Parse openTime error for ticket %s: %v", ticket, err)
				skippedCount++
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
			swap, _ := strconv.ParseFloat(swapStr, 64)
			commission, _ := strconv.ParseFloat(commissionStr, 64)
			profit, _ := strconv.ParseFloat(profitStr, 64)
			pips, _ := strconv.ParseFloat(pipsStr, 64)

			side := "long"
			if sideStr == "sell" {
				side = "short"
			}

			totalPnL := profit + swap + commission

			// 去重檢查
			var exists bool
			err = db.QueryRow(`
				SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND symbol = ? AND entry_time = ? AND lot_size = ?)
			`, accountID, symbol, openTime, volume).Scan(&exists)

			if exists {
				skippedCount++
				continue
			}

			// 寫入資料庫，加入 timezone_offset 預設為 8 (台灣)
			// 同時讓 market_session 為空，交給前端 Svelte 在第一次載入編輯時自動判斷
			_, err = db.Exec(`
				INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, pnl_points, entry_time, exit_time, trade_type, notes, timezone_offset)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, accountID, symbol, side, entryPrice, exitPrice, volume, totalPnL, pips, openTime, closeTime, "actual", "FTMO CSV 匯入: Ticket "+ticket, 8)

			if err != nil {
				log.Printf("Import failed for ticket %s: %v", ticket, err)
				skippedCount++
			} else {
				importedCount++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  fmt.Sprintf("匯入完成：成功 %d 筆，跳過 %d 筆（重複或失敗）", importedCount, skippedCount),
			"imported": importedCount,
			"skipped":  skippedCount,
		})
	}
}
