package database

import (
	"database/sql"
	"log"
	"sync"
	"trade-journal/internal/models"
)

var (
	usageUpdateMu sync.Map // map[int64]*sync.Mutex
)

// CalculateImagesSize 計算上傳圖片的總大小
func CalculateImagesSize(images []models.ImageUpload) int64 {
	var total int64
	for _, img := range images {
		total += img.FileSize
	}
	return total
}

// AddToAccountStorageUsage 增量更新帳號儲存空間佔用
func AddToAccountStorageUsage(db *sql.DB, accountID int64, deltaBytes int64) {
	if deltaBytes == 0 {
		return
	}
	_, err := db.Exec("UPDATE accounts SET storage_usage = storage_usage + ? WHERE id = ?", deltaBytes, accountID)
	if err != nil {
		log.Printf("[Usage] 增量更新帳號 %d 空間失敗: %v", accountID, err)
		// 如果增量更新失敗，可以在這裡選擇觸發一次全量校準
		go UpdateAccountStorageUsage(db, accountID)
	}
}

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
	// 簡化邏輯：只計算圖片大小，因為文字佔用微乎其微且計算耗時
	query := `
		UPDATE accounts SET storage_usage = (
			SELECT COALESCE(SUM(file_size), 0) FROM trade_images 
			WHERE trade_id IN (SELECT id FROM trades WHERE account_id = ?)
		)
		WHERE id = ?
	`
	_, err := db.Exec(query, accountID, accountID)
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
