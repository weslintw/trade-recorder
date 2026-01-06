package ctrader

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Manager struct {
	db              *sql.DB
	connections     map[int64]*AccountConn
	mu              sync.RWMutex
	lastResample    sync.Map // map[string]time.Time (ticket -> last resample time)
	symbolDigitsMap sync.Map // map[int64]int (symbolID -> digits)
}

type AccountConn struct {
	AccountID int64
	Conn      *websocket.Conn
	StopChan  chan struct{}
	Waiters   map[string]chan *CTraderMessage
	WaitMu    sync.Mutex
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
	if err != nil {
		return
	}
	defer rows.Close()

	activeIDs := make(map[int64]bool)
	for rows.Next() {
		var id int64
		var ctid, token, cid, secret, env, status string
		if rows.Scan(&id, &ctid, &token, &cid, &secret, &env, &status) != nil {
			continue
		}
		if status == "syncing" {
			continue
		}
		activeIDs[id] = true
		m.mu.RLock()
		_, exists := m.connections[id]
		m.mu.RUnlock()
		if !exists {
			log.Printf("[cTrader Reconcile] Starting listener for Account %d", id)
			m.startListener(id, ctid, token, cid, secret, env)
		}
	}

	m.mu.Lock()
	for id, conn := range m.connections {
		if !activeIDs[id] {
			close(conn.StopChan)
			delete(m.connections, id)
		}
	}
	m.mu.Unlock()
}

func (m *Manager) startListener(accountID int64, ctid, token, cid, secret, env string) {
	stopChan := make(chan struct{})
	m.mu.Lock()
	m.connections[accountID] = &AccountConn{
		AccountID: accountID,
		StopChan:  stopChan,
		Waiters:   make(map[string]chan *CTraderMessage),
	}
	m.mu.Unlock()
	go m.listenerLoop(accountID, ctid, token, cid, secret, env, stopChan)
}

func (m *Manager) listenerLoop(accountID int64, ctid, token, cid, secret, env string, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		default:
			if err := m.connectAndListen(accountID, ctid, token, cid, secret, env, stopChan); err != nil {
				select {
				case <-stopChan:
					return
				case <-time.After(10 * time.Second):
				}
			}
		}
	}
}

func (m *Manager) StopListener(accountID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.connections[accountID]; ok {
		close(conn.StopChan)
		delete(m.connections, accountID)
	}
}

func (m *Manager) connectAndListen(accountID int64, ctidStr, token, cid, secret, env string, stopChan chan struct{}) error {
	url := CTraderLiveURL
	if env == "demo" {
		url = CTraderDemoURL
	}
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	m.mu.Lock()
	if ac, ok := m.connections[accountID]; ok {
		ac.Conn = conn
	}
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		if ac, ok := m.connections[accountID]; ok {
			ac.Conn = nil
		}
		m.mu.Unlock()
	}()

	if err := sendAndVerify(conn, PayloadAppAuthReq, map[string]string{"clientId": cid, "clientSecret": secret}, PayloadAppAuthRes); err != nil {
		return err
	}
	ctid, _ := strconv.ParseInt(ctidStr, 10, 64)
	if err := sendAndVerify(conn, PayloadAccountAuthReq, map[string]interface{}{"ctidTraderAccountId": ctid, "accessToken": token}, PayloadAccountAuthRes); err != nil {
		return err
	}

	symbolMap := make(map[int64]string)
	symbolLotSizeMap := make(map[int64]int64)
	// Pre-fetch ALL symbols (Light) to guarantee we have names
	symListResp, err := sendRequest(conn, PayloadSymbolsListReq, map[string]interface{}{"ctidTraderAccountId": ctid})
	if err == nil {
		var p struct {
			Symbols []struct {
				SymbolID   int64  `json:"symbolId"`
				SymbolName string `json:"symbolName"`
			} `json:"symbol"`
		}
		json.Unmarshal(symListResp.Payload, &p)
		for _, s := range p.Symbols {
			symbolMap[s.SymbolID] = s.SymbolName
		}
		log.Printf("[cTrader Manager] Pre-fetched %d symbols", len(p.Symbols))
	} else {
		log.Printf("[cTrader Manager] Failed to pre-fetch symbols: %v", err)
	}

	fetchSymbol := func(sid int64) string {
		// If we have lot size, we assume we have everything needed (including name from pre-fetch)
		if symbolLotSizeMap[sid] > 0 {
			return symbolMap[sid]
		}

		// If we miss details (LotSize), fetch detailed info
		log.Printf("[cTrader Manager] Fetching details for SymbolID: %d", sid)
		resp, err := sendRequest(conn, PayloadSymbolByIdReq, map[string]interface{}{"ctidTraderAccountId": ctid, "symbolId": []int64{sid}})
		if err != nil {
			log.Printf("[cTrader Manager] ERROR fetching symbol %d: %v", sid, err)
			return symbolMap[sid] // Return at least name if we have it
		}

		var p struct {
			Symbols []struct {
				SymbolID   int64  `json:"symbolId"`
				SymbolName string `json:"symbolName"`
				Digits     int    `json:"digits"`
				LotSize    int64  `json:"lotSize"`
			} `json:"symbol"`
		}
		if err := json.Unmarshal(resp.Payload, &p); err != nil {
			log.Printf("[cTrader Manager] ERROR unmarshaling symbol: %v", err)
			return symbolMap[sid]
		}
		for _, s := range p.Symbols {
			if s.SymbolName != "" {
				symbolMap[s.SymbolID] = s.SymbolName
			}
			symbolLotSizeMap[s.SymbolID] = s.LotSize
			m.symbolDigitsMap.Store(s.SymbolID, s.Digits)
			if s.SymbolID == sid {
				return symbolMap[sid]
			}
		}
		name := symbolMap[sid]
		log.Printf("[cTrader Manager] Symbol Resolution: ID %d -> Name %s", sid, name)
		return name
	}

	posResp, err := sendRequest(conn, PayloadReconcileReq, map[string]interface{}{"ctidTraderAccountId": ctid})
	if err == nil {
		var p struct {
			Position []struct {
				PositionID int64   `json:"positionId"`
				Price      float64 `json:"price"`
				TradeData  struct {
					SymbolID      int64   `json:"symbolId"`
					Volume        int64   `json:"volume"`
					TradeSide     int     `json:"tradeSide"`
					OpenTimestamp int64   `json:"openTimestamp"`
					EntryPrice    float64 `json:"entryPrice"`
				} `json:"tradeData"`
				SymbolName             string  `json:"symbolName"`
				StopLoss               float64 `json:"stopLoss"`
				UtcLastUpdateTimestamp int64   `json:"utcLastUpdateTimestamp"`
			} `json:"position"`
		}
		if err := json.Unmarshal(posResp.Payload, &p); err != nil {
			log.Printf("[cTrader Manager] JSON Unmarshal Error for Reconcile: %v", err)
		}
		if len(p.Position) > 0 {
			log.Printf("[cTrader Manager] Found %d open positions for Account %d", len(p.Position), accountID)
		}
		for _, pos := range p.Position {
			// Ensure we have FULL details (Digits, LotSize) for this active symbol
			symbolName := fetchSymbol(pos.TradeData.SymbolID)

			// Get digits for debugging
			digits := 2
			if d, ok := m.symbolDigitsMap.Load(pos.TradeData.SymbolID); ok {
				digits = d.(int)
			}
			log.Printf("[cTrader Manager] Position %d (Symbol: %s, ID: %d) using Digits: %d", pos.PositionID, symbolName, pos.TradeData.SymbolID, digits)

			// Sanity check for future timestamps
			entryTS := pos.TradeData.OpenTimestamp
			if entryTS == 0 || entryTS > time.Now().Add(1*time.Hour).UnixMilli() {
				if pos.UtcLastUpdateTimestamp > 0 && pos.UtcLastUpdateTimestamp < time.Now().Add(1*time.Hour).UnixMilli() {
					entryTS = pos.UtcLastUpdateTimestamp
				} else {
					entryTS = time.Now().UnixMilli()
				}
				log.Printf("[cTrader Manager] Corrected future/invalid timestamp for Position %d to %d", pos.PositionID, entryTS)
			}

			// Price discovery: try top-level 'price' then 'tradeData.entryPrice'
			entryPrice := pos.Price
			if entryPrice == 0 {
				entryPrice = pos.TradeData.EntryPrice
			}
			log.Printf("[cTrader Manager] Position %d - Picked Price: %f (from Price:%f, EntryPrice:%f)", pos.PositionID, entryPrice, pos.Price, pos.TradeData.EntryPrice)

			symbol := symbolName
			if symbol == "" {
				symbol = pos.SymbolName
				log.Printf("[cTrader Manager] Using fallback symbol: %s", symbol)
			}
			lotSize := symbolLotSizeMap[pos.TradeData.SymbolID]
			if lotSize == 0 {
				lotSize = 100000
			}
			ticket := fmt.Sprintf("ctrader-pos-%d", pos.PositionID) // NEW FORMAT
			vol := float64(pos.TradeData.Volume) / float64(lotSize)
			side := "long"
			if pos.TradeData.TradeSide == 2 {
				side = "short"
			}
			log.Printf("[cTrader Manager] Position %d resolved to Symbol: %s", pos.PositionID, symbol)

			var exists bool
			m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?))", accountID, ticket, fmt.Sprintf("ctrader-%d", pos.PositionID)).Scan(&exists)
			if !exists {
				// Only insert if we have a valid price, or use a default if it's still 0 but we want to see it
				_, err := m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, lot_size, entry_time, trade_type, notes, ticket, initial_sl)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					accountID, symbol, side, entryPrice, vol, time.UnixMilli(entryTS), "actual", "cTrader Push: Initial Sync", ticket, 0)
				if err != nil {
					log.Printf("[cTrader Manager] FAILED to insert position %d: %v", pos.PositionID, err)
				} else {
					log.Printf("[cTrader Manager] Successfully inserted position %d as %s (Price: %f)", pos.PositionID, symbol, entryPrice)
				}
			} else {
				// If it already exists, ensure symbol, price and status are correct
				// If entry_price was 0, also clear pnl_series to force a clean historical re-fetch
				m.db.Exec("UPDATE trades SET symbol = ?, entry_price = CASE WHEN entry_price = 0 THEN ? ELSE entry_price END, pnl_series = CASE WHEN entry_price = 0 THEN NULL ELSE pnl_series END, exit_price = NULL, entry_time = ? WHERE account_id = ? AND (ticket = ? OR ticket = ?)",
					symbol, entryPrice, time.UnixMilli(entryTS), accountID, ticket, fmt.Sprintf("ctrader-%d", pos.PositionID))
				log.Printf("[cTrader Manager] Updated existing position %d with Symbol: %s, Price: %f, Time: %v", pos.PositionID, symbol, entryPrice, time.UnixMilli(entryTS))
			}
		}

		// Subscribe to spots for active positions
		activeSymbolIDs := make(map[int64]bool)
		for _, pos := range p.Position {
			activeSymbolIDs[pos.TradeData.SymbolID] = true
		}
		if len(activeSymbolIDs) > 0 {
			ids := make([]int64, 0, len(activeSymbolIDs))
			for sid := range activeSymbolIDs {
				ids = append(ids, sid)
			}
			log.Printf("[cTrader Manager] Subscribing to spots for %d symbols", len(ids))
			subResp, subErr := sendRequest(conn, PayloadSubscribeSpotsReq, map[string]interface{}{
				"ctidTraderAccountId":      ctid,
				"symbolId":                 ids,
				"subscribeToSpotTimestamp": true,
			})
			// Try Depth as well
			sendRequest(conn, 2156, map[string]interface{}{
				"ctidTraderAccountId": ctid,
				"symbolId":            ids,
			})
			if subErr != nil {
				log.Printf("[cTrader Manager] Subscription FAILED: %v", subErr)
			} else {
				// 3. Subscription OK. Now trigger an immediate background PnL fetch for these positions
				for _, pos := range p.Position {
					t := fmt.Sprintf("ctrader-pos-%d", pos.PositionID)
					ticket := t
					sID := pos.TradeData.SymbolID
					entry := pos.Price
					if entry == 0 {
						entry = pos.TradeData.EntryPrice
					}
					eTS := pos.TradeData.OpenTimestamp
					side := "long"
					if pos.TradeData.TradeSide == 2 {
						side = "short"
					}

					go func(tStr string, accID int64, ent float64, etMilli int64, sid int64, sStr string, vol int64) {
						time.Sleep(5 * time.Second) // Wait for everything to settle
						m.mu.RLock()
						ac, ok := m.connections[accID]
						m.mu.RUnlock()
						if ok && ac != nil {
							sideInt := 1
							if sStr == "short" {
								sideInt = 2
							}
							digits := 2
							if d, ok := m.symbolDigitsMap.Load(sid); ok {
								digits = d.(int)
							}
							log.Printf("[cTrader Manager] Initial Sparkline Fetch for %s (Digits: %d, Vol: %d)", tStr, digits, vol)
							newSeriesStr := fetchPnLSeries(ac, ac.Conn, accID, sid, etMilli, time.Now().UnixMilli(), ent, vol, sideInt, digits)
							if newSeriesStr != "" {
								m.db.Exec("UPDATE trades SET pnl_series = ? WHERE ticket = ?", newSeriesStr, tStr)
							}
						}
					}(ticket, accountID, entry, eTS, sID, side, pos.TradeData.Volume)
				}
				log.Printf("[cTrader Manager] Subscription OK. Type: %d, Payload: %s", subResp.PayloadType, string(subResp.Payload))
			}
		}
	}

	conn.SetReadDeadline(time.Time{})
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	errChan := make(chan error, 1)
	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("[cTrader Manager] Read error for Account %d: %v", accountID, err)
				errChan <- err
				return
			}
			var msg CTraderMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				log.Printf("[cTrader Manager] JSON Unmarshal Error (Message len %d): %v", len(message), err)
				continue
			}

			if msg.PayloadType != 51 {
				// log.Printf("[cTrader Manager] Msg Received Type: %d, Payload: %s", msg.PayloadType, string(msg.Payload))
			}

			if msg.PayloadType == PayloadExecutionEvent {
				log.Printf("[cTrader Manager] ExecutionEvent received for Account %d", accountID)
				m.handleExecutionEvent(accountID, msg.Payload, symbolMap, symbolLotSizeMap, fetchSymbol, ctid)
			} else if msg.PayloadType == PayloadSpotEvent {
				m.handleSpotEvent(accountID, msg.Payload, symbolMap, symbolLotSizeMap, fetchSymbol)
			} else if msg.PayloadType == 2155 {
				m.handleDepthEvent(accountID, msg.Payload, symbolMap, symbolLotSizeMap, fetchSymbol)
			}

			// Deliver to waiters if any
			if msg.ClientMsgID != "" {
				m.mu.RLock()
				ac, ok := m.connections[accountID]
				m.mu.RUnlock()
				if ok && ac != nil {
					ac.WaitMu.Lock()
					if ch, exists := ac.Waiters[msg.ClientMsgID]; exists {
						select {
						case ch <- &msg:
						default:
						}
					}
					ac.WaitMu.Unlock()
				}
			}
		}
	}()

	for {
		select {
		case <-stopChan:
			return nil
		case err := <-errChan:
			return err
		case <-heartbeat.C:
			if conn.WriteJSON(CTraderMessage{PayloadType: PayloadHeartbeatEvent, Payload: json.RawMessage("{}")}) != nil {
				return err
			}
		}
	}
}

func (m *Manager) handleExecutionEvent(accountID int64, payload json.RawMessage, symbolMap map[int64]string, lotSizeMap map[int64]int64, fetchSymbol func(int64) string, ctid int64) {
	var event struct {
		ExecutionType int `json:"executionType"`
		Deal          struct {
			DealID              int64   `json:"dealId"`
			Volume              int64   `json:"volume"`
			SymbolID            int64   `json:"symbolId"`
			ExecutionPrice      float64 `json:"executionPrice"`
			ExecutionTimestamp  int64   `json:"executionTimestamp"`
			TradeSide           int     `json:"tradeSide"`
			PositionID          int64   `json:"positionId"`
			ClosePositionDetail struct {
				EntryPrice  float64 `json:"entryPrice"`
				GrossProfit int64   `json:"grossProfit"`
				Commission  int64   `json:"commission"`
				Swap        int64   `json:"swap"`
			} `json:"closePositionDetail"`
		} `json:"deal"`
		Position struct {
			PositionID int64 `json:"positionId"`
			TradeData  struct {
				SymbolID      int64   `json:"symbolId"`
				Volume        int64   `json:"volume"`
				EntryPrice    float64 `json:"entryPrice"`
				OpenTimestamp int64   `json:"openTimestamp"`
				TradeSide     int     `json:"tradeSide"`
			} `json:"tradeData"`
			StopLoss float64 `json:"stopLoss"`
		} `json:"position"`
	}
	if json.Unmarshal(payload, &event) != nil {
		return
	}
	log.Printf("[cTrader Manager] Event Received - Acc: %d, Type: %d, DealID: %d, PosID: %d", accountID, event.ExecutionType, event.Deal.DealID, event.Position.PositionID)

	// Handle: 2 (FILLED), 3 (PARTIALLY_FILLED), 8 (TRADE)
	if event.ExecutionType != 2 && event.ExecutionType != 3 && event.ExecutionType != 8 {
		log.Printf("[cTrader Manager] Ignoring event type %d for Account %d", event.ExecutionType, accountID)
		return
	}
	deal := event.Deal
	if deal.DealID == 0 {
		log.Printf("[cTrader Manager] Event has no DealID, skipping... (Type: %d)", event.ExecutionType)
		return
	}

	symbol := fetchSymbol(deal.SymbolID)
	if symbol == "" {
		log.Printf("[cTrader Manager] ERROR: Failed to fetch symbol for SymbolID %d", deal.SymbolID)
		return
	}
	lotSize := lotSizeMap[deal.SymbolID]
	if lotSize == 0 {
		lotSize = 100000
	}
	ticket := fmt.Sprintf("ctrader-deal-%d", deal.DealID)
	posTicket := fmt.Sprintf("ctrader-pos-%d", deal.PositionID)
	legacyTicket := fmt.Sprintf("ctrader-%d", deal.PositionID)
	vol := float64(deal.Volume) / float64(lotSize)
	execTime := time.UnixMilli(deal.ExecutionTimestamp)

	if deal.ClosePositionDetail.EntryPrice > 0 {
		var initialSL sql.NullFloat64
		var entryTime sql.NullTime
		var existingSeries sql.NullString

		// Try to preserve original entry time and SL from the open position record
		// MUST use Null types to avoid scan errors if fields are NULL
		err := m.db.QueryRow("SELECT initial_sl, entry_time, pnl_series FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?)",
			accountID, posTicket, legacyTicket).Scan(&initialSL, &entryTime, &existingSeries)

		// Priority: 1. API OpenTimestamp, 2. DB preserved time, 3. ExecTime (last resort)
		finalEntryTime := execTime
		if event.Position.TradeData.OpenTimestamp > 0 {
			finalEntryTime = time.UnixMilli(event.Position.TradeData.OpenTimestamp)
		} else if entryTime.Valid && !entryTime.Time.IsZero() {
			finalEntryTime = entryTime.Time
		} else {
			log.Printf("[cTrader Manager] Warning: Could not find original entry time for closing position %d (err: %v), using exit time", deal.PositionID, err)
		}

		m.db.Exec("DELETE FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?)", accountID, posTicket, legacyTicket)

		side := "long"
		if deal.TradeSide == 1 { // Closing Buy(1) means original was Short
			side = "short"
		}

		pnl := float64(deal.ClosePositionDetail.GrossProfit+deal.ClosePositionDetail.Commission+deal.ClosePositionDetail.Swap) / 100.0
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			seriesVal := existingSeries.String
			_, err := m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes, ticket, initial_sl, exit_sl, pnl_series)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				accountID, symbol, side, deal.ClosePositionDetail.EntryPrice, deal.ExecutionPrice, vol, pnl, finalEntryTime, execTime, "actual", "cTrader Push: Closed Position", ticket, initialSL.Float64, deal.ExecutionPrice, seriesVal)
			if err != nil {
				log.Printf("[cTrader Manager] Failed to insert Push Closed trade: %v", err)
			} else {
				log.Printf("[cTrader Manager] Successfully inserted Push Closed trade: %s", ticket)

				// TRIGGER IMMEDIATE SPARKLINE SYNC FOR CLOSED TRADE
				go func(tStr string, accID int64, ent float64, startMilli, endMilli int64, sid int64, sStr string, v int64) {
					m.mu.RLock()
					ac, ok := m.connections[accID]
					m.mu.RUnlock()
					if ok && ac != nil {
						sInt := 1
						if sStr == "short" {
							sInt = 2
						}
						digits := 2
						if d, ok := m.symbolDigitsMap.Load(sid); ok {
							digits = d.(int)
						}

						newSeriesStr := fetchPnLSeries(ac, ac.Conn, accID, sid, startMilli, endMilli, ent, v, sInt, digits)
						if newSeriesStr != "" {
							m.db.Exec("UPDATE trades SET pnl_series = ? WHERE ticket = ?", newSeriesStr, tStr)
						}
					}
				}(ticket, accountID, deal.ClosePositionDetail.EntryPrice, finalEntryTime.UnixMilli(), deal.ExecutionTimestamp, deal.SymbolID, side, deal.Volume)
			}
		}
	} else {
		ticket = posTicket
		side := "long"
		if deal.TradeSide == 2 {
			side = "short"
		}
		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			// Use API's OpenTimestamp if available, otherwise use ExecutionTimestamp
			entryMilli := deal.ExecutionTimestamp
			if event.Position.TradeData.OpenTimestamp > 0 {
				entryMilli = event.Position.TradeData.OpenTimestamp
			}

			_, err := m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, lot_size, entry_time, trade_type, notes, ticket, initial_sl)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				accountID, symbol, side, deal.ExecutionPrice, vol, time.UnixMilli(entryMilli), "actual", "cTrader Push: Open Position", ticket, event.Position.StopLoss)
			if err != nil {
				log.Printf("[cTrader Manager] Failed to insert Push Open trade: %v", err)
			} else {
				log.Printf("[cTrader Manager] Successfully inserted Push Open trade: %s", ticket)
				// TRIGGER IMMEDIATE SPARKLINE SYNC FOR NEW OPEN POSITION
				go m.triggerSyncForTrade(accountID, ticket, deal.ExecutionPrice, entryMilli, time.Now().UnixMilli(), deal.SymbolID, side, deal.Volume)
			}
		}
	}
}

var lastSpotUpdates sync.Map

func (m *Manager) handleDepthEvent(accountID int64, payload json.RawMessage, symbolMap map[int64]string, lotSizeMap map[int64]int64, fetchSymbol func(int64) string) {
	var event struct {
		SymbolID  int64 `json:"symbolId"`
		NewQuotes []struct {
			Bid uint64 `json:"bid"`
			Ask uint64 `json:"ask"`
		} `json:"newQuotes"`
	}

	if json.Unmarshal(payload, &event) != nil {
		return
	}

	var bid, ask float64
	for _, q := range event.NewQuotes {
		if q.Bid > 0 {
			bid = float64(q.Bid) / 100000.0
		}
		if q.Ask > 0 {
			ask = float64(q.Ask) / 100000.0
		}
	}

	if bid == 0 && ask == 0 {
		return
	}

	// Throttle: reuse the same map/logic
	if last, ok := lastSpotUpdates.Load(event.SymbolID); ok {
		if time.Since(last.(time.Time)) < 1*time.Second {
			return
		}
	}
	lastSpotUpdates.Store(event.SymbolID, time.Now())

	m.updatePnLFromPrices(accountID, event.SymbolID, bid, ask, symbolMap, lotSizeMap, fetchSymbol)
}

func (m *Manager) updatePnLFromPrices(accountID, symbolID int64, bid, ask float64, symbolMap map[int64]string, lotSizeMap map[int64]int64, fetchSymbol func(int64) string) {
	// Ensure we have symbol details
	if symbolMap[symbolID] == "" || lotSizeMap[symbolID] == 0 {
		fetchSymbol(symbolID)
	}

	// Hardcode fix for XAUUSD for specific user symbol ID 41
	if symbolID == 41 && symbolMap[41] == "" {
		symbolMap[41] = "XAUUSD"
	}

	symbol := symbolMap[symbolID]
	lotSize := lotSizeMap[symbolID]

	if symbol == "" || lotSize == 0 {
		return
	}

	multiplier := float64(lotSize)
	// cTrader quirk: Commodities (Gold/Silver) lotSize in ProtoOA is often contract size * 100.
	if strings.Contains(symbol, "XAU") || strings.Contains(symbol, "GOLD") || strings.Contains(symbol, "XAG") || strings.Contains(symbol, "SILVER") {
		multiplier = multiplier / 100.0
	}

	if bid > 0 {
		res, _ := m.db.Exec(`UPDATE trades SET pnl = (? - entry_price) * lot_size * ?, updated_at = CURRENT_TIMESTAMP 
			WHERE account_id = ? AND symbol = ? AND side = 'long' AND exit_price IS NULL`,
			bid, multiplier, accountID, symbol)
		if n, _ := res.RowsAffected(); n > 0 {
			// log.Printf("[cTrader Manager] Updated Long PnL for %s: %d rows (Bid=%f)", symbol, n, bid)
		}
	}

	if ask > 0 {
		res, _ := m.db.Exec(`UPDATE trades SET pnl = (entry_price - ?) * lot_size * ?, updated_at = CURRENT_TIMESTAMP 
			WHERE account_id = ? AND symbol = ? AND side = 'short' AND exit_price IS NULL`,
			ask, multiplier, accountID, symbol)
		if n, _ := res.RowsAffected(); n > 0 {
			// log.Printf("[cTrader Manager] Updated Short PnL for %s: %d rows (Ask=%f)", symbol, n, ask)
		}
	}

	// Real-time Sparkline (pnl_series) Update
	// Note: We use the symbol name string in the DB, so we still need to match it.
	rows, err := m.db.Query(`SELECT ticket, entry_price, pnl_series, entry_time, side FROM trades 
		WHERE account_id = ? AND symbol = ? AND exit_price IS NULL`, accountID, symbol)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var ticket, side string
			var seriesNull sql.NullString
			var entry float64
			var entryTime time.Time
			if err := rows.Scan(&ticket, &entry, &seriesNull, &entryTime, &side); err == nil {
				var series []float64
				seriesStr := seriesNull.String
				if seriesStr != "" {
					json.Unmarshal([]byte(seriesStr), &series)
				}

				// Ensure we have 32 points
				if len(series) != 32 {
					newSeries := make([]float64, 32)
					if len(series) > 0 {
						// Stretch existing data to 32 points
						for i := 0; i < 32; i++ {
							idx := int(float64(i) * float64(len(series)-1) / 31.0)
							newSeries[i] = series[idx]
						}
					} else {
						// Initialize with the current diff instead of 0s
						var cp float64
						if side == "long" {
							cp = bid
						} else {
							cp = ask
						}
						curDiff := cp - entry
						if side == "short" {
							curDiff = entry - cp
						}
						for i := 0; i < 32; i++ {
							newSeries[i] = curDiff
						}
					}
					series = newSeries
				}

				var cur float64
				if side == "long" {
					cur = bid
				} else {
					cur = ask
				}

				if cur == 0 {
					continue
				}

				// 1. Immediate Update: Refresh the LAST point of the 32 divisions
				diff := cur - entry
				if side == "short" {
					diff = entry - cur
				}
				series[31] = diff
				newJSON, _ := json.Marshal(series)
				m.db.Exec("UPDATE trades SET pnl_series = ? WHERE ticket = ?", string(newJSON), ticket)

				// 2. Periodic Resample: Every 1 minute, re-fetch historical distribution from API
				last, ok := m.lastResample.Load(ticket)
				if !ok || time.Since(last.(time.Time)) > 1*time.Minute {
					m.lastResample.Store(ticket, time.Now())

					// Run API fetch in background to not block price updates
					go func(t string, accID int64, e float64, et time.Time, sID int64, sStr string) {
						m.mu.RLock()
						conn, ok := m.connections[accID]
						m.mu.RUnlock()

						if ok && conn != nil && conn.Conn != nil {
							sideInt := 1
							if sStr == "short" {
								sideInt = 2
							}
							// Using fetchPnLSeries from sync.go (accessible within same package)
							digits := 5
							if d, ok := m.symbolDigitsMap.Load(sID); ok {
								digits = d.(int)
							}
							newSeriesStr := fetchPnLSeries(conn, conn.Conn, accID, sID, et.UnixMilli(), time.Now().UnixMilli(), e, 100, sideInt, digits)
							if newSeriesStr != "" {
								log.Printf("[cTrader Manager] Debug: Ticket %s PnL Series (32 points): %s", t, newSeriesStr)
								m.db.Exec("UPDATE trades SET pnl_series = ? WHERE ticket = ?", newSeriesStr, t)
							}
						}
					}(ticket, accountID, entry, entryTime, symbolID, side)
				}
			}
		}
	}
}

func (m *Manager) handleSpotEvent(accountID int64, payload json.RawMessage, symbolMap map[int64]string, lotSizeMap map[int64]int64, fetchSymbol func(int64) string) {
	var event struct {
		SymbolID int64   `json:"symbolId"`
		Bid      float64 `json:"bid"`
		Ask      float64 `json:"ask"`
	}

	if json.Unmarshal(payload, &event) != nil {
		return
	}

	// Throttle: 1 update per second per symbol
	if last, ok := lastSpotUpdates.Load(event.SymbolID); ok {
		if time.Since(last.(time.Time)) < 1*time.Second {
			return
		}
	}
	lastSpotUpdates.Store(event.SymbolID, time.Now())

	m.updatePnLFromPrices(accountID, event.SymbolID, event.Bid, event.Ask, symbolMap, lotSizeMap, fetchSymbol)
}

func (ac *AccountConn) SendRequest(msg CTraderMessage) (*CTraderMessage, error) {
	if ac == nil || ac.Conn == nil {
		return nil, fmt.Errorf("managed connection is nil")
	}

	resChan := make(chan *CTraderMessage, 1)
	ac.WaitMu.Lock()
	ac.Waiters[msg.ClientMsgID] = resChan
	ac.WaitMu.Unlock()

	defer func() {
		ac.WaitMu.Lock()
		delete(ac.Waiters, msg.ClientMsgID)
		ac.WaitMu.Unlock()
	}()

	if err := ac.Conn.WriteJSON(msg); err != nil {
		return nil, err
	}

	select {
	case res := <-resChan:
		return res, nil
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("request timeout (managed)")
	}
}

// triggerSyncForTrade fetches historical sparkline data in the background and updates the DB.
func (m *Manager) triggerSyncForTrade(accID int64, tStr string, ent float64, startMilli, endMilli int64, sid int64, sStr string, v int64) {
	time.Sleep(3 * time.Second) // Small delay to ensure DB transaction is finalized
	m.mu.RLock()
	ac, ok := m.connections[accID]
	m.mu.RUnlock()
	if ok && ac != nil {
		sInt := 1
		if sStr == "short" {
			sInt = 2
		}
		digits := 2
		if d, ok := m.symbolDigitsMap.Load(sid); ok {
			digits = d.(int)
		}

		newSeriesStr := fetchPnLSeries(ac, ac.Conn, accID, sid, startMilli, endMilli, ent, v, sInt, digits)
		if newSeriesStr != "" {
			m.db.Exec("UPDATE trades SET pnl_series = ? WHERE ticket = ?", newSeriesStr, tStr)
		}
	}
}
