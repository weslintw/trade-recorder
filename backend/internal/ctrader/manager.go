package ctrader

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Manager struct {
	db          *sql.DB
	connections map[int64]*AccountConn
	mu          sync.RWMutex
}

type AccountConn struct {
	AccountID int64
	Conn      *websocket.Conn
	StopChan  chan struct{}
}

var GlobalManager *Manager

func StartManager(db *sql.DB) {
	GlobalManager = &Manager{db: db, connections: make(map[int64]*AccountConn)}
	go GlobalManager.run()
}

func (m *Manager) run() {
	for {
		m.reconcileConnections()
		time.Sleep(30 * time.Second)
	}
}

func (m *Manager) reconcileConnections() {
	rows, err := m.db.Query("SELECT id, ctrader_account_id, ctrader_token, ctrader_client_id, ctrader_client_secret, ctrader_env, sync_status FROM accounts WHERE type = 'ctrader' AND ctrader_token != ''")
	if err != nil { return }
	defer rows.Close()

	activeIDs := make(map[int64]bool)
	for rows.Next() {
		var id int64
		var ctid, token, cid, secret, env, status string
		if rows.Scan(&id, &ctid, &token, &cid, &secret, &env, &status) != nil { continue }
		if status == "syncing" { continue }
		activeIDs[id] = true
		m.mu.RLock(); _, exists := m.connections[id]; m.mu.RUnlock()
		if !exists { m.startListener(id, ctid, token, cid, secret, env) }
	}

	m.mu.Lock()
	for id, conn := range m.connections {
		if !activeIDs[id] {
			close(conn.StopChan); delete(m.connections, id)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) startListener(accountID int64, ctid, token, cid, secret, env string) {
	stopChan := make(chan struct{})
	m.mu.Lock(); m.connections[accountID] = &AccountConn{AccountID: accountID, StopChan: stopChan}; m.mu.Unlock()
	go m.listenerLoop(accountID, ctid, token, cid, secret, env, stopChan)
}

func (m *Manager) listenerLoop(accountID int64, ctid, token, cid, secret, env string, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan: return
		default:
			if err := m.connectAndListen(accountID, ctid, token, cid, secret, env, stopChan); err != nil {
				select { case <-stopChan: return; case <-time.After(10 * time.Second): }
			}
		}
	}
}

func (m *Manager) StopListener(accountID int64) {
	m.mu.Lock(); defer m.mu.Unlock()
	if conn, ok := m.connections[accountID]; ok {
		close(conn.StopChan); delete(m.connections, accountID)
	}
}

func (m *Manager) connectAndListen(accountID int64, ctidStr, token, cid, secret, env string, stopChan chan struct{}) error {
	url := CTraderLiveURL; if env == "demo" { url = CTraderDemoURL }
	conn, _, err := websocket.DefaultDialer.Dial(url, nil); if err != nil { return err }; defer conn.Close()

	if err := sendAndVerify(conn, PayloadAppAuthReq, map[string]string{"clientId": cid, "clientSecret": secret}, PayloadAppAuthRes); err != nil { return err }
	ctid, _ := strconv.ParseInt(ctidStr, 10, 64)
	if err := sendAndVerify(conn, PayloadAccountAuthReq, map[string]interface{}{"ctidTraderAccountId": ctid, "accessToken": token}, PayloadAccountAuthRes); err != nil { return err }

	symbolMap := make(map[int64]string); symbolLotSizeMap := make(map[int64]int64)
	fetchSymbol := func(sid int64) {
		if _, ok := symbolMap[sid]; ok { return }
		resp, err := sendRequest(conn, PayloadSymbolByIdReq, map[string]interface{}{"ctidTraderAccountId": ctid, "symbolId": []int64{sid}})
		if err == nil {
			var p struct { Symbols []struct { SymbolID int64 `json:"symbolId"`; SymbolName string `json:"symbolName"`; LotSize int64 `json:"lotSize"` } `json:"symbol"` }
			json.Unmarshal(resp.Payload, &p)
			for _, s := range p.Symbols { symbolMap[s.SymbolID] = s.SymbolName; symbolLotSizeMap[s.SymbolID] = s.LotSize }
		}
	}

	posResp, err := sendRequest(conn, PayloadReconcileReq, map[string]interface{}{"ctidTraderAccountId": ctid})
	if err == nil {
		var p struct { Positions []struct { PositionID int64 `json:"positionId"`; TradeData struct { SymbolID int64 `json:"symbolId"`; Volume int64 `json:"volume"`; TradeSide int `json:"tradeSide"`; EntryPrice float64 `json:"entryPrice"`; EntryTimestamp int64 `json:"entryTimestamp"` } `json:"tradeData"`; SymbolName string `json:"symbolName"`; StopLoss float64 `json:"stopLoss"` } `json:"position"` }
		json.Unmarshal(posResp.Payload, &p)
		for _, pos := range p.Positions {
			fetchSymbol(pos.TradeData.SymbolID)
			symbol := symbolMap[pos.TradeData.SymbolID]; if symbol == "" { symbol = pos.SymbolName }
			lotSize := symbolLotSizeMap[pos.TradeData.SymbolID]; if lotSize == 0 { lotSize = 100000 }
			ticket := fmt.Sprintf("ctrader-pos-%d", pos.PositionID) // NEW FORMAT
			vol := float64(pos.TradeData.Volume) / float64(lotSize)
			side := "long"; if pos.TradeData.TradeSide == 2 { side = "short" }
			var exists bool
			m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?))", accountID, ticket, fmt.Sprintf("ctrader-%d", pos.PositionID)).Scan(&exists)
			if !exists {
				m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, lot_size, entry_time, trade_type, notes, ticket, initial_sl)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					accountID, symbol, side, pos.TradeData.EntryPrice, vol, time.UnixMilli(pos.TradeData.EntryTimestamp), "actual", "cTrader Push: Initial Sync", ticket, 0)
			}
		}
	}

	conn.SetReadDeadline(time.Time{})
	heartbeat := time.NewTicker(25 * time.Second); defer heartbeat.Stop()
	errChan := make(chan error, 1)
	go func() {
		for {
			_, message, err := conn.ReadMessage(); if err != nil { errChan <- err; return }
			var msg CTraderMessage; if json.Unmarshal(message, &msg) != nil { continue }
			if msg.PayloadType == PayloadExecutionEvent {
				m.handleExecutionEvent(accountID, msg.Payload, symbolMap, symbolLotSizeMap, fetchSymbol, ctid)
			}
		}
	}()

	for {
		select {
		case <-stopChan: return nil
		case err := <-errChan: return err
		case <-heartbeat.C:
			if conn.WriteJSON(CTraderMessage{PayloadType: PayloadHeartbeatEvent, Payload: json.RawMessage("{}")}) != nil { return err }
		}
	}
}

func (m *Manager) handleExecutionEvent(accountID int64, payload json.RawMessage, symbolMap map[int64]string, lotSizeMap map[int64]int64, fetchSymbol func(int64), ctid int64) {
	var event struct {
		ExecutionType int `json:"executionType"`
		Deal struct { DealID int64 `json:"dealId"`; Volume int64 `json:"volume"`; SymbolID int64 `json:"symbolId"`; ExecutionPrice float64 `json:"executionPrice"`; ExecutionTimestamp int64 `json:"executionTimestamp"`; TradeSide int `json:"tradeSide"`; PositionID int64 `json:"positionId"`; ClosePositionDetail struct { EntryPrice float64 `json:"entryPrice"`; GrossProfit int64 `json:"grossProfit"`; Commission int64 `json:"commission"`; Swap int64 `json:"swap"` } `json:"closePositionDetail"` } `json:"deal"`
		Position struct { PositionID int64 `json:"positionId"`; TradeData struct { SymbolID int64 `json:"symbolId"`; Volume int64 `json:"volume"`; EntryPrice float64 `json:"entryPrice"` } `json:"tradeData"`; StopLoss float64 `json:"stopLoss"` } `json:"position"`
	}
	if json.Unmarshal(payload, &event) != nil { return }
	if event.ExecutionType != 2 && event.ExecutionType != 8 { return }
	deal := event.Deal; if deal.DealID == 0 { return }

	fetchSymbol(deal.SymbolID)
	symbol := symbolMap[deal.SymbolID]; lotSize := lotSizeMap[deal.SymbolID]; if lotSize == 0 { lotSize = 100000 }
	ticket := fmt.Sprintf("ctrader-deal-%d", deal.DealID)
	posTicket := fmt.Sprintf("ctrader-pos-%d", deal.PositionID)
	legacyTicket := fmt.Sprintf("ctrader-%d", deal.PositionID)
	vol := float64(deal.Volume) / float64(lotSize); execTime := time.UnixMilli(deal.ExecutionTimestamp)

	if deal.ClosePositionDetail.EntryPrice > 0 {
		var initialSL float64
		m.db.QueryRow("SELECT initial_sl FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?)", accountID, posTicket, legacyTicket).Scan(&initialSL)
		m.db.Exec("DELETE FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?)", accountID, posTicket, legacyTicket)
		side := "long"; if deal.TradeSide == 1 { side = "short" }
		pnl := float64(deal.ClosePositionDetail.GrossProfit + deal.ClosePositionDetail.Commission + deal.ClosePositionDetail.Swap) / 100.0
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes, ticket, initial_sl, exit_sl)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				accountID, symbol, side, deal.ClosePositionDetail.EntryPrice, deal.ExecutionPrice, vol, pnl, execTime, execTime, "actual", "cTrader Push: Closed Position", ticket, initialSL, event.Position.StopLoss)
		}
	} else {
		ticket = posTicket; side := "long"; if deal.TradeSide == 2 { side = "short" }
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, lot_size, entry_time, trade_type, notes, ticket, initial_sl)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				accountID, symbol, side, deal.ExecutionPrice, vol, execTime, "actual", "cTrader Push: Open Position", ticket, event.Position.StopLoss)
		}
	}
}
