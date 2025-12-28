package models

import "time"

// Account 交易帳號模型
type Account struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`           // 帳號別名，如 "個人實盤", "FTMO 挑戰"
	Type          string     `json:"type"`           // "local" 或 "metatrader"
	MT5AccountID  string     `json:"mt5_account_id"` // MetaApi Account ID
	MT5Token      string     `json:"mt5_token"`      // MetaApi Token
	Status        string     `json:"status"`         // "active", "disconnected"
	SyncStatus    string     `json:"sync_status"`    // "idle", "syncing", "success", "failed"
	LastSyncedAt  *time.Time `json:"last_synced_at"`
	LastSyncError string     `json:"last_sync_error"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// AccountCreate 建立帳號請求
type AccountCreate struct {
	Name         string `json:"name" binding:"required"`
	Type         string `json:"type" binding:"required,oneof=local metatrader"`
	MT5AccountID string `json:"mt5_account_id"`
	MT5Token     string `json:"mt5_token"`
}

// AccountUpdate 更新帳號請求
type AccountUpdate struct {
	Name         *string `json:"name"`
	MT5AccountID *string `json:"mt5_account_id"`
	MT5Token     *string `json:"mt5_token"`
}
