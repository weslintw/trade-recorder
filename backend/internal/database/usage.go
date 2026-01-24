package database

import (
	"database/sql"
	"log"
	"sync"
)

var (
	usageUpdateMu sync.Map // map[int64]*sync.Mutex
)

// UpdateAccountStorageUsage 更新指定帳號的儲存空間佔用量 (加上併發保護避免鎖定)
func UpdateAccountStorageUsage(db *sql.DB, accountID int64) error {
	// 使用 sync.Map 確保同一個帳號同一時間只有一個計算在跑
	actualMu, _ := usageUpdateMu.LoadOrStore(accountID, &sync.Mutex{})
	mu := actualMu.(*sync.Mutex)
	if !mu.TryLock() {
		// 如果該帳號已經在計算中了，直接跳過本次請求，減少 DB 壓力
		return nil
	}
	defer mu.Unlock()

	// 計算邏輯保持不變，但加上 Limit 或優化
	// 這裡維持原邏輯但因為有了 TryLock，不會產生堆積效應
	query := `
		UPDATE accounts SET storage_usage = (
			SELECT COALESCE(SUM(
				LENGTH(COALESCE(notes, '')) +
				LENGTH(COALESCE(entry_reason, '')) +
				LENGTH(COALESCE(exit_reason, '')) +
				LENGTH(COALESCE(entry_signals, '')) +
				LENGTH(COALESCE(entry_checklist, '')) +
				LENGTH(COALESCE(trend_analysis, '')) +
				LENGTH(COALESCE(journal, '')) +
				LENGTH(COALESCE(entry_pattern, '')) +
				LENGTH(COALESCE(legend_images, '')) +
				LENGTH(COALESCE(expert_images, '')) +
				LENGTH(COALESCE(elite_images, '')) +
				LENGTH(COALESCE(legend_king_image, '')) +
				LENGTH(COALESCE(legend_htf_image, '')) +
				LENGTH(COALESCE(sl_history, '')) +
				LENGTH(COALESCE(pnl_series, ''))
			), 0) FROM trades WHERE account_id = ?
		) + (
			SELECT COALESCE(SUM(
				LENGTH(COALESCE(notes, '')) +
				LENGTH(COALESCE(trend_analysis, ''))
			), 0) FROM daily_plans WHERE account_id = ?
		) + (
			SELECT COALESCE(SUM(file_size), 0) FROM trade_images 
			WHERE trade_id IN (SELECT id FROM trades WHERE account_id = ?)
		)
		WHERE id = ?
	`
	_, err := db.Exec(query, accountID, accountID, accountID, accountID)
	if err != nil {
		log.Printf("[Usage] 更新帳號 %d 空間佔用失敗: %v", accountID, err)
		return err
	}
	return nil
}

// UpdateAllAccountsStorageUsage 更新所有帳號的儲存空間佔用量
func UpdateAllAccountsStorageUsage(db *sql.DB) {
	rows, err := db.Query("SELECT id FROM accounts")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			UpdateAccountStorageUsage(db, id)
		}
	}
	log.Println("[Usage] 已完成所有帳號的空間佔用量校準")
}
