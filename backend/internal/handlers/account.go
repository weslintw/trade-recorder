package handlers

import (
	"database/sql"
	"net/http"
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
