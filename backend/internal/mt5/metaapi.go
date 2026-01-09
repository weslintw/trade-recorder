package mt5

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
	"trade-journal/internal/models"
	"trade-journal/internal/ws"
)

const MetaApiBaseURL = "https://mt-client-api-v1.new-york.agiliumtrade.ai"
const MetaApiProvisioningURL = "https://mt-provisioning-api-v1.agiliumtrade.agiliumtrade.ai"

// SyncMT5History 從 MetaApi 同步歷史交易
func SyncMT5History(db *sql.DB, accountID int64, mt5AccountID string, token string) error {
	err := internalSync(db, accountID, mt5AccountID, token)
	if err != nil {
		log.Printf("MT5 Sync Failed: %v", err)
		db.Exec("UPDATE accounts SET sync_status = 'failed', last_sync_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", err.Error(), accountID)
		return err
	}

	db.Exec("UPDATE accounts SET sync_status = 'success', last_sync_error = '', last_synced_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", accountID)
	ws.GlobalHub.BroadcastUpdate(accountID, "TRADE_UPDATE")
	return nil
}

func internalSync(db *sql.DB, accountID int64, mt5AccountNumber string, token string) error {
	log.Printf("Starting MT5 sync for account %d (Search ID/Login: %s)", accountID, mt5AccountNumber)

	client := &http.Client{Timeout: 30 * time.Second}

	// 1. 自動偵測 Region 與正確的 MetaApi ID
	// 由於 MetaApi 網址可能因文件更新或地區而異，嘗試多個常見的 Provisioning 端點
	provisioningEndpoints := []string{
		"https://mt-provisioning-api-v1.agiliumtrade.ai",
		"https://mt-provisioning-api-v1.metaapi.cloud",
		"https://mt-provisioning-api-v1.agiliumtrade.agiliumtrade.ai",
	}

	var accountFound bool
	var actualAccountID string
	var region string

	var lastErr error
	for _, baseURL := range provisioningEndpoints {
		url := fmt.Sprintf("%s/users/current/accounts", baseURL)
		log.Printf("Trying to list accounts from: %s", url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("auth-token", token)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Failed to connect to %s: %v", baseURL, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			lastErr = fmt.Errorf("Provisioning API error (%d) at %s: %s", resp.StatusCode, baseURL, string(body))
			continue
		}

		var accounts []struct {
			ID               string      `json:"_id"`
			Login            interface{} `json:"login"`
			Region           string      `json:"region"`
			ConnectionStatus string      `json:"connectionStatus"` // CONNECTED, DISCONNECTED
			DeploymentStatus string      `json:"deploymentStatus"` // DEPLOYED, UNDEPLOYED
		}
		if err := json.NewDecoder(resp.Body).Decode(&accounts); err != nil {
			lastErr = err
			continue
		}

		// 搜尋匹配的帳號 (匹配 ID 或 Login)
		for _, acc := range accounts {
			accLoginStr := fmt.Sprintf("%v", acc.Login)
			if acc.ID == mt5AccountNumber || accLoginStr == mt5AccountNumber {
				actualAccountID = acc.ID
				region = acc.Region
				accountFound = true
				log.Printf("Matched account! MetaApi ID: %s, Login: %s, Region: %s, Status: %s/%s",
					acc.ID, accLoginStr, acc.Region, acc.DeploymentStatus, acc.ConnectionStatus)

				if acc.DeploymentStatus != "DEPLOYED" {
					return fmt.Errorf("帳號尚未部署 (Status: %s)。請至 MetaApi 後台點擊 'Deploy'。", acc.DeploymentStatus)
				}
				// 注意：有時剛啟動會是 DISCONNECTED，我們可以在下一步加 wait 參數
				break
			}
		}

		if accountFound {
			break
		}
	}

	if !accountFound {
		if lastErr != nil {
			return fmt.Errorf("無法解析帳號: %v (請檢查 Token 與帳號編號)", lastErr)
		}
		return fmt.Errorf("在您的 MetaApi 帳號清單中找不到 Login 為 '%s' 的帳號", mt5AccountNumber)
	}

	if region == "" {
		region = "new-york"
	}
	regionalBaseURL := fmt.Sprintf("https://mt-client-api-v1.%s.agiliumtrade.ai", region)

	// 2. 獲取成交 (Deals) - 加入 wait-for-synchronization 確保資料同步
	now := time.Now().UTC()
	startTime := now.AddDate(0, 0, -30).Format("2006-01-02T15:04:05.000Z")
	endTime := now.Format("2006-01-02T15:04:05.000Z")

	url := fmt.Sprintf("%s/users/current/accounts/%s/history-deals/time/%s/%s?wait-for-synchronization=true", regionalBaseURL, actualAccountID, startTime, endTime)
	log.Printf("Fetching history (with wait): %s", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("auth-token", token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 504 {
			return fmt.Errorf("同步超時 (504)。這通常表示您的 MT5 帳號尚未連線至券商，請檢查 MetaApi 後台的帳號密碼與伺服器設定是否正確。")
		}
		return fmt.Errorf("MetaApi 資料錯誤 (status %d): %s", resp.StatusCode, string(body))
	}

	var deals []struct {
		ID         string    `json:"id"`
		Symbol     string    `json:"symbol"`
		Type       string    `json:"type"`      // DEAL_TYPE_BUY, DEAL_TYPE_SELL
		EntryType  string    `json:"entryType"` // DEAL_ENTRY_IN, DEAL_ENTRY_OUT
		Volume     float64   `json:"volume"`
		Price      float64   `json:"price"`
		Profit     float64   `json:"profit"`
		Commission float64   `json:"commission"`
		Swap       float64   `json:"swap"`
		Time       time.Time `json:"time"`
		PositionID string    `json:"positionId"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&deals); err != nil {
		return err
	}

	// 2. 將成交組合為 Position (交易紀錄)
	positions := make(map[string]*models.Trade)

	for _, deal := range deals {
		trade, ok := positions[deal.PositionID]
		if !ok {
			vol := deal.Volume
			pnl := 0.0
			trade = &models.Trade{
				AccountID: accountID,
				Symbol:    deal.Symbol,
				LotSize:   &vol,
				PnL:       &pnl,
				TradeType: "actual",
			}
			positions[deal.PositionID] = trade
		}

		if deal.EntryType == "DEAL_ENTRY_IN" {
			trade.Side = "long"
			if deal.Type == "DEAL_TYPE_SELL" {
				trade.Side = "short"
			}
			price := deal.Price
			trade.EntryPrice = &price
			trade.EntryTime = deal.Time
		} else if deal.EntryType == "DEAL_ENTRY_OUT" {
			price := deal.Price
			trade.ExitPrice = &price
			trade.ExitTime = &deal.Time

			currentPnL := 0.0
			if trade.PnL != nil {
				currentPnL = *trade.PnL
			}
			newPnL := currentPnL + deal.Profit + deal.Commission + deal.Swap
			trade.PnL = &newPnL
		}
	}

	// 3. 批量查詢現有交易 (效能優化：一次性查詢而不是逐筆檢查)
	log.Printf("Checking for existing trades in batch...")

	// 建立查詢的時間範圍 (從最早到最晚的交易)
	var minTime, maxTime time.Time
	first := true
	for _, trade := range positions {
		if trade.ExitPrice == nil {
			continue
		}
		if first {
			minTime = trade.EntryTime
			maxTime = trade.EntryTime
			first = false
		} else {
			if trade.EntryTime.Before(minTime) {
				minTime = trade.EntryTime
			}
			if trade.EntryTime.After(maxTime) {
				maxTime = trade.EntryTime
			}
		}
	}

	// 如果沒有可插入的交易，直接返回
	if first {
		log.Printf("No completed trades to sync")
		return nil
	}

	// 批量查詢該時間範圍內的所有現有交易
	existingTrades := make(map[string]bool)
	rows, err := db.Query(`
		SELECT symbol, entry_time, lot_size 
		FROM trades 
		WHERE account_id = ? AND entry_time BETWEEN ? AND ?
	`, accountID, minTime, maxTime)

	if err != nil {
		log.Printf("Batch query error: %v", err)
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var symbol string
		var entryTime time.Time
		var lotSize float64

		if err := rows.Scan(&symbol, &entryTime, &lotSize); err != nil {
			continue
		}

		// 建立唯一鍵 (用於快速比對)
		key := fmt.Sprintf("%s|%s|%.2f", symbol, entryTime.Format(time.RFC3339), lotSize)
		existingTrades[key] = true
	}

	log.Printf("Found %d existing trades in database", len(existingTrades))

	// 4. 插入新交易 (使用內存中的 map 進行快速比對)
	insertedCount := 0
	skippedCount := 0

	for posID, trade := range positions {
		if trade.ExitPrice == nil {
			continue
		}

		// 建立此交易的唯一鍵
		key := fmt.Sprintf("%s|%s|%.2f", trade.Symbol, trade.EntryTime.Format(time.RFC3339), *trade.LotSize)

		if existingTrades[key] {
			skippedCount++
			continue
		}

		_, err = db.Exec(`
			INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, trade.Symbol, trade.Side, trade.EntryPrice, trade.ExitPrice, trade.LotSize, trade.PnL, trade.EntryTime, trade.ExitTime, "actual", "MT5 Sync: Position "+posID)

		if err != nil {
			log.Printf("Insert synced trade error: %v", err)
		} else {
			insertedCount++
		}
	}

	log.Printf("Sync complete: %d new trades inserted, %d duplicates skipped", insertedCount, skippedCount)

	return nil
}
