package ctrader

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

const (
	CTraderLiveURL = "wss://live.ctraderapi.com:5035"
	// Payload Types
	PayloadAppAuthReq     = 2100
	PayloadAppAuthRes     = 2101
	PayloadAccountAuthReq = 2102
	PayloadAccountAuthRes = 2103
	PayloadReconcileReq   = 2124
	PayloadReconcileRes   = 2125
	PayloadDealListReq    = 2137
	PayloadDealListRes    = 2138
	PayloadErrorRes       = 2142
)

type CTraderMessage struct {
	ClientMsgID string          `json:"clientMsgId,omitempty"`
	PayloadType uint32          `json:"payloadType"`
	Payload     json.RawMessage `json:"payload"`
}

// SyncCTraderHistory 從 cTrader Open API 同步歷史交易與持倉
func SyncCTraderHistory(db *sql.DB, accountID int64, cTraderAccountID string, token string, clientID string, clientSecret string) error {
	log.Printf("Starting cTrader sync for account %d (Account ID: %s)", accountID, cTraderAccountID)

	// 更新狀態為同步中
	db.Exec("UPDATE accounts SET sync_status = 'syncing', last_sync_error = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?", accountID)

	err := internalSync(db, accountID, cTraderAccountID, token, clientID, clientSecret)
	if err != nil {
		log.Printf("cTrader Sync Failed: %v", err)
		db.Exec("UPDATE accounts SET sync_status = 'failed', last_sync_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", err.Error(), accountID)
		return err
	}

	db.Exec("UPDATE accounts SET sync_status = 'success', last_sync_error = '', last_synced_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", accountID)
	return nil
}

func internalSync(db *sql.DB, accountID int64, cTraderAccountIDStr string, token string, clientID string, clientSecret string) error {
	cTraderAccountID, err := strconv.ParseInt(cTraderAccountIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid cTrader Account ID: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(CTraderLiveURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to cTrader: %v", err)
	}
	defer conn.Close()

	// 1. App Auth
	err = sendAndVerify(conn, PayloadAppAuthReq, map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
	}, PayloadAppAuthRes)
	if err != nil {
		return fmt.Errorf("app auth failed: %v", err)
	}

	// 2. Account Auth
	err = sendAndVerify(conn, PayloadAccountAuthReq, map[string]interface{}{
		"ctidTraderAccountId": cTraderAccountID,
		"accessToken":         token,
	}, PayloadAccountAuthRes)
	if err != nil {
		return fmt.Errorf("account auth failed: %v", err)
	}

	// 3. Get Closed Deals (Last 30 days)
	toTime := time.Now().UnixMilli()
	fromTime := time.Now().AddDate(0, 0, -30).UnixMilli()
	dealsResp, err := sendRequest(conn, PayloadDealListReq, map[string]interface{}{
		"ctidTraderAccountId": cTraderAccountID,
		"fromTimestamp":       fromTime,
		"toTimestamp":         toTime,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch deals: %v", err)
	}

	var dealsPayload struct {
		Deals []struct {
			DealID     int64   `json:"dealId"`
			SymbolName string  `json:"symbolName"`
			Volume     int64   `json:"volume"`
			ExecutionPrice float64 `json:"executionPrice"`
			ExecutionTimestamp int64 `json:"executionTimestamp"`
			TradeSide  string  `json:"tradeSide"`  // BUY, SELL
			PositionID int64   `json:"positionId"`
			ClosePositionDetail struct {
				EntryPrice float64 `json:"entryPrice"`
				GrossProfit int64 `json:"grossProfit"`
				Commission int64 `json:"commission"`
				Swap int64 `json:"swap"`
			} `json:"closePositionDetail"`
		} `json:"deal"`
	}
	json.Unmarshal(dealsResp.Payload, &dealsPayload)

	// 4. Get Open Positions
	positionsResp, err := sendRequest(conn, PayloadReconcileReq, map[string]interface{}{
		"ctidTraderAccountId": cTraderAccountID,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch positions: %v", err)
	}

	var positionsPayload struct {
		Positions []struct {
			PositionID int64 `json:"positionId"`
			TradeSide  string `json:"tradeSide"`
			SymbolName string `json:"symbolName"`
			Volume     int64  `json:"volume"`
			EntryPrice float64 `json:"entryPrice"`
			EntryTimestamp int64 `json:"entryTimestamp"`
		} `json:"position"`
	}
	json.Unmarshal(positionsResp.Payload, &positionsPayload)

	// 5. Save to DB
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Process Closed Deals
	for _, deal := range dealsPayload.Deals {
		if deal.ClosePositionDetail.EntryPrice == 0 {
			continue // Not a closing deal
		}

		ticket := fmt.Sprintf("ctrader-%d", deal.PositionID)
		entryTime := time.UnixMilli(deal.ExecutionTimestamp) // Approximation
		exitTime := time.UnixMilli(deal.ExecutionTimestamp)
		side := "long"
		if deal.TradeSide == "BUY" { // Closing a SHORT
			side = "short"
		}
		
		vol := float64(deal.Volume) / 100000.0
		pnl := float64(deal.ClosePositionDetail.GrossProfit + deal.ClosePositionDetail.Commission + deal.ClosePositionDetail.Swap) / 100.0

		var exists bool
		tx.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			tx.Exec(`
				INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes, ticket)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, accountID, deal.SymbolName, side, deal.ClosePositionDetail.EntryPrice, deal.ExecutionPrice, vol, pnl, entryTime, exitTime, "actual", "cTrader Sync: Closed Position", ticket)
		}
	}

	// Process Open Positions
	for _, pos := range positionsPayload.Positions {
		ticket := fmt.Sprintf("ctrader-%d", pos.PositionID)
		entryTime := time.UnixMilli(pos.EntryTimestamp)
		side := "long"
		if pos.TradeSide == "SHORT" {
			side = "short"
		}
		vol := float64(pos.Volume) / 100000.0

		var exists bool
		tx.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			tx.Exec(`
				INSERT INTO trades (account_id, symbol, side, entry_price, lot_size, entry_time, trade_type, notes, ticket)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, accountID, pos.SymbolName, side, pos.EntryPrice, vol, entryTime, "actual", "cTrader Sync: Open Position", ticket)
		}
	}

	return tx.Commit()
}

func sendRequest(conn *websocket.Conn, payloadType uint32, payload interface{}) (*CTraderMessage, error) {
	clientMsgID := fmt.Sprintf("%d", time.Now().UnixNano())
	payloadJSON, _ := json.Marshal(payload)
	msg := CTraderMessage{
		ClientMsgID: clientMsgID,
		PayloadType: payloadType,
		Payload:     payloadJSON,
	}

	err := conn.WriteJSON(msg)
	if err != nil {
		return nil, err
	}

	// Timeout for response
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		var resp CTraderMessage
		err := conn.ReadJSON(&resp)
		if err != nil {
			return nil, err
		}

		if resp.PayloadType == PayloadErrorRes {
			return nil, fmt.Errorf("cTrader error: %s", string(resp.Payload))
		}

		if resp.ClientMsgID == clientMsgID {
			return &resp, nil
		}
	}
}

func sendAndVerify(conn *websocket.Conn, payloadType uint32, payload interface{}, expectedType uint32) error {
	resp, err := sendRequest(conn, payloadType, payload)
	if err != nil {
		return err
	}
	if resp.PayloadType != expectedType {
		return fmt.Errorf("unexpected response type: %d, expected %d", resp.PayloadType, expectedType)
	}
	return nil
}
