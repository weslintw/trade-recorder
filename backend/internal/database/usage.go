package database

import (
	"database/sql"
	"log"
)

// UpdateAccountStorageUsage 更新指定帳號的儲存空間佔用量
func UpdateAccountStorageUsage(db *sql.DB, accountID int64) error {
	// 計算邏輯：
	// 1. 交易紀錄中的文字長度 (notes, reasons, signals 等)
	// 2. 每日規劃中的文字長度 (notes, trend_analysis)
	// 3. 交易圖片關聯表中的檔案大小 (file_size)
	query := `
		UPDATE accounts SET storage_usage = (
			SELECT COALESCE(SUM(
				LENGTH(COALESCE(notes, '')) +
				LENGTH(COALESCE(entry_reason, '')) +
				LENGTH(COALESCE(exit_reason, '')) +
				LENGTH(COALESCE(entry_signals, '')) +
				LENGTH(COALESCE(entry_checklist, '')) +
				LENGTH(COALESCE(trend_analysis, '')) +
				LENGTH(COALESCE(entry_strategy_image, '')) +
				LENGTH(COALESCE(entry_strategy_image_original, '')) +
				LENGTH(COALESCE(legend_king_image, '')) +
				LENGTH(COALESCE(legend_king_image_original, '')) +
				LENGTH(COALESCE(legend_htf_image, '')) +
				LENGTH(COALESCE(legend_htf_image_original, ''))
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
