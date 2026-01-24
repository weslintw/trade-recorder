package ctrader

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"trade-journal/internal/ws"

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
	AccountID        int64
	Conn             *websocket.Conn
	StopChan         chan struct{}
	Waiters          map[string]chan *CTraderMessage
	WaitMu           sync.Mutex
	Mu               sync.RWMutex
	SymbolMap        map[int64]string
	SymbolLotSizeMap map[int64]int64
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
		AccountID:        accountID,
		StopChan:         stopChan,
		Waiters:          make(map[string]chan *CTraderMessage),
		SymbolMap:        make(map[int64]string),
		SymbolLotSizeMap: make(map[int64]int64),
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

	m.mu.RLock()
	ac, _ := m.connections[accountID]
	m.mu.RUnlock()
	if ac == nil {
		return fmt.Errorf("account connection lost")
	}

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
		ac.Mu.Lock()
		for _, s := range p.Symbols {
			ac.SymbolMap[s.SymbolID] = s.SymbolName
		}
		ac.Mu.Unlock()
		log.Printf("[cTrader Manager] Pre-fetched %d symbols", len(p.Symbols))
	} else {
		log.Printf("[cTrader Manager] Failed to pre-fetch symbols: %v", err)
	}

	fetchSymbol := func(sid int64) string {
		ac.Mu.RLock()
		hasLotSize := ac.SymbolLotSizeMap[sid] > 0
		nameIfFound := ac.SymbolMap[sid]
		ac.Mu.RUnlock()

		if hasLotSize {
			return nameIfFound
		}

		// If we miss details (LotSize), fetch detailed info
		log.Printf("[cTrader Manager] Fetching details for SymbolID: %d", sid)
		resp, err := sendRequest(conn, PayloadSymbolByIdReq, map[string]interface{}{"ctidTraderAccountId": ctid, "symbolId": []int64{sid}})
		if err != nil {
			log.Printf("[cTrader Manager] ERROR fetching symbol %d: %v", sid, err)
			return nameIfFound
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
			return nameIfFound
		}
		for _, s := range p.Symbols {
			ac.Mu.Lock()
			if s.SymbolName != "" {
				ac.SymbolMap[s.SymbolID] = s.SymbolName
			}
			ac.SymbolLotSizeMap[s.SymbolID] = s.LotSize
			ac.Mu.Unlock()
			m.symbolDigitsMap.Store(s.SymbolID, s.Digits)
			if s.SymbolID == sid {
				ac.Mu.RLock()
				name := ac.SymbolMap[sid]
				ac.Mu.RUnlock()
				return name
			}
		}
		ac.Mu.RLock()
		name := ac.SymbolMap[sid]
		ac.Mu.RUnlock()
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
			ac.Mu.RLock()
			lotSize := ac.SymbolLotSizeMap[pos.TradeData.SymbolID]
			ac.Mu.RUnlock()
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

			if msg.PayloadType != 51 && msg.PayloadType != 2155 && msg.PayloadType != 2131 {
				log.Printf("[cTrader Manager] Msg Received Type: %d, Account: %d, Payload Size: %d", msg.PayloadType, accountID, len(msg.Payload))
			}

			if msg.PayloadType == PayloadExecutionEvent {
				log.Printf("[cTrader Manager] ExecutionEvent received for Account %d", accountID)
				m.handleExecutionEvent(accountID, msg.Payload, ac.SymbolMap, ac.SymbolLotSizeMap, fetchSymbol, ctid)
			} else if msg.PayloadType == PayloadSpotEvent {
				m.handleSpotEvent(accountID, msg.Payload, ac.SymbolMap, ac.SymbolLotSizeMap, fetchSymbol)
			} else if msg.PayloadType == 2155 {
				m.handleDepthEvent(accountID, msg.Payload, ac.SymbolMap, ac.SymbolLotSizeMap, fetchSymbol)
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
		var tradeID int64
		var journal, entryReason, entryStrategy, entryStrategyImg, entryStrategyImgOrig, entrySignals, entryChecklist, entryPattern, trendAnalysis, entryTimeframe, trendType, marketSession, colorTag, notes sql.NullString
		var legKingHTF, legKingImg, legKingImgOrig, legHTF, legHTFImg, legHTFImgOrig, legDeHTF, legImages sql.NullString

		// Try to preserve original entry time, SL and ALL strategy/analysis fields
		err := m.db.QueryRow(`SELECT id, initial_sl, entry_time, pnl_series, journal, entry_reason, entry_strategy, 
			entry_strategy_image, entry_strategy_image_original, entry_signals, entry_checklist, entry_pattern, 
			trend_analysis, entry_timeframe, trend_type, market_session, color_tag, notes,
			legend_king_htf, legend_king_image, legend_king_image_original, legend_htf, legend_htf_image, legend_htf_image_original, legend_de_htf, legend_images
			FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?)`,
			accountID, posTicket, legacyTicket).Scan(
			&tradeID, &initialSL, &entryTime, &existingSeries, &journal, &entryReason, &entryStrategy,
			&entryStrategyImg, &entryStrategyImgOrig, &entrySignals, &entryChecklist, &entryPattern,
			&trendAnalysis, &entryTimeframe, &trendType, &marketSession, &colorTag, &notes,
			&legKingHTF, &legKingImg, &legKingImgOrig, &legHTF, &legHTFImg, &legHTFImgOrig, &legDeHTF, &legImages)

		// Priority: 1. API OpenTimestamp, 2. DB preserved time, 3. ExecTime (last resort)
		finalEntryTime := execTime
		if event.Position.TradeData.OpenTimestamp > 0 {
			finalEntryTime = time.UnixMilli(event.Position.TradeData.OpenTimestamp)
		} else if entryTime.Valid && !entryTime.Time.IsZero() {
			finalEntryTime = entryTime.Time
		} else {
			log.Printf("[cTrader Manager] Warning: Could not find original entry time for closing position %d (err: %v), using exit time", deal.PositionID, err)
		}

		side := "long"
		if deal.TradeSide == 1 { // Closing Buy(1) means original was Short
			side = "short"
		}

		pnl := float64(deal.ClosePositionDetail.GrossProfit+deal.ClosePositionDetail.Commission+deal.ClosePositionDetail.Swap) / 100.0
		bullet, rr := 0.0, 0.0
		mult := getMultiplier(symbol)
		if initialSL.Valid && initialSL.Float64 > 0 && deal.ClosePositionDetail.EntryPrice > 0 {
			riskPoints := math.Abs(deal.ClosePositionDetail.EntryPrice - initialSL.Float64)
			bullet = math.Round(riskPoints*mult*getPointValue(symbol)*vol*100) / 100

			// Signed PnL Points relative to side
			pnlPoints := (deal.ExecutionPrice - deal.ClosePositionDetail.EntryPrice) * mult
			if side == "short" {
				pnlPoints = -pnlPoints
			}
			pnlPoints = math.Round(pnlPoints*100) / 100

			if riskPoints > 0 {
				rr = math.Round((pnlPoints/(riskPoints*mult))*100) / 100
			}
		}

		var exists bool
		m.db.QueryRow("SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND ticket = ?)", accountID, ticket).Scan(&exists)
		if !exists {
			seriesVal := existingSeries.String
			// Use preserved notes if available, otherwise use default
			finalNotes := notes.String
			if finalNotes == "" || finalNotes == "cTrader Push: Initial Sync" {
				finalNotes = "cTrader Push: Closed Position"
			}

			res, err := m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes, ticket, initial_sl, exit_sl, bullet_size, rr_ratio, pnl_series, journal, entry_reason, entry_strategy, entry_strategy_image, entry_strategy_image_original, entry_signals, entry_checklist, entry_pattern, trend_analysis, entry_timeframe, trend_type, market_session, color_tag, legend_king_htf, legend_king_image, legend_king_image_original, legend_htf, legend_htf_image, legend_htf_image_original, legend_de_htf, legend_images)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				accountID, symbol, side, deal.ClosePositionDetail.EntryPrice, deal.ExecutionPrice, vol, pnl, finalEntryTime, execTime, "actual", finalNotes, ticket, initialSL, deal.ExecutionPrice, bullet, rr, seriesVal, journal, entryReason, entryStrategy, entryStrategyImg, entryStrategyImgOrig, entrySignals, entryChecklist, entryPattern, trendAnalysis, entryTimeframe, trendType, marketSession, colorTag, legKingHTF, legKingImg, legKingImgOrig, legHTF, legHTFImg, legHTFImgOrig, legDeHTF, legImages)

			if err != nil {
				log.Printf("[cTrader Manager] Failed to insert Push Closed trade: %v", err)
			} else {
				newID, _ := res.LastInsertId()
				if tradeID > 0 && newID > 0 {
					log.Printf("[cTrader Manager] Migrating associations from %d to %d", tradeID, newID)
					// Copy images
					m.db.Exec("INSERT INTO trade_images (trade_id, image_type, image_path, file_size, description) SELECT ?, image_type, image_path, file_size, description FROM trade_images WHERE trade_id = ?", newID, tradeID)
					// Copy tags
					m.db.Exec("INSERT INTO trade_tags (trade_id, tag_id) SELECT ?, tag_id FROM trade_tags WHERE trade_id = ?", newID, tradeID)
				}

				// Only delete the POS record if the volume is now 0 (full close)
				if event.Position.TradeData.Volume == 0 {
					m.db.Exec("DELETE FROM trades WHERE account_id = ? AND (ticket = ? OR ticket = ?)", accountID, posTicket, legacyTicket)
					log.Printf("[cTrader Manager] Full close of position %d", deal.PositionID)

					// --- 退訂邏輯：檢查是否還有同品種的開倉位 ---
					go func(accID int64, sym string, sid int64) {
						var otherExists int
						m.db.QueryRow("SELECT 1 FROM trades WHERE account_id = ? AND symbol = ? AND exit_price IS NULL LIMIT 1", accID, sym).Scan(&otherExists)
						if otherExists == 0 {
							log.Printf("[cTrader Manager] No more open positions for %s. Unsubscribing from spots...", sym)
							m.mu.RLock()
							targetAc, ok := m.connections[accID]
							m.mu.RUnlock()
							if ok && targetAc != nil && targetAc.Conn != nil {
								targetAc.Conn.WriteJSON(CTraderMessage{
									ClientMsgID: fmt.Sprintf("unsub-%d", time.Now().UnixNano()),
									PayloadType: PayloadUnsubscribeSpotsReq,
									Payload:     json.RawMessage(fmt.Sprintf(`{"ctidTraderAccountId": %d, "symbolId": [%d]}`, ctid, sid)),
								})
							}
						}
					}(accountID, symbol, deal.SymbolID)
				} else {
					log.Printf("[cTrader Manager] Partial close of position %d, remaining volume: %d", deal.PositionID, event.Position.TradeData.Volume)
					remVol := float64(event.Position.TradeData.Volume) / float64(lotSize)
					m.db.Exec("UPDATE trades SET lot_size = ? WHERE account_id = ? AND (ticket = ? OR ticket = ?)", remVol, accountID, posTicket, legacyTicket)
				}

				log.Printf("[cTrader Manager] Successfully inserted Push Closed trade: %s", ticket)
				ws.GlobalHub.BroadcastUpdate(accountID, "TRADE_UPDATE")

				// TRIGGER FULL MANUAL SYNC AFTER SEVERAL SECONDS (as requested by user)
				go func(tStr string, accID int64) {
					time.Sleep(5 * time.Second)
					m.ManualSyncTrade(accID, tStr)
				}(ticket, accountID)
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
				ws.GlobalHub.BroadcastUpdate(accountID, "TRADE_UPDATE")
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
		if time.Since(last.(time.Time)) < 5*time.Second {
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
		res, err := m.db.Exec(`UPDATE trades SET pnl = (? - entry_price) * lot_size * ?, updated_at = CURRENT_TIMESTAMP 
			WHERE account_id = ? AND symbol = ? AND side = 'long' AND exit_price IS NULL`,
			bid, multiplier, accountID, symbol)
		if err == nil && res != nil {
			if n, _ := res.RowsAffected(); n > 0 {
				// Updated
			}
		}
	}

	if ask > 0 {
		res, err := m.db.Exec(`UPDATE trades SET pnl = (entry_price - ?) * lot_size * ?, updated_at = CURRENT_TIMESTAMP 
			WHERE account_id = ? AND symbol = ? AND side = 'short' AND exit_price IS NULL`,
			ask, multiplier, accountID, symbol)
		if err == nil && res != nil {
			if n, _ := res.RowsAffected(); n > 0 {
				prices := make(map[string]interface{})
				prices[symbol] = map[string]interface{}{
					"price": ask,
					"time":  time.Now().UnixMilli(),
				}
				
				msg := ws.WSMessage{
					Type:      "PRICE_UPDATE",
					AccountID: accountID,
					Data:      prices,
				}
				data, _ := json.Marshal(msg)
				ws.GlobalHub.Broadcast(data)
			}
		}
	}

	// === PERFORMANCE OPTIMIZATION ===
	// We no longer update pnl_series in DB every 5 seconds to reduce write pressure.
	// The front-end can calculate the current tip of the sparkline from the price updates.
	// Only full manual sync or trade closing will refresh the historical series.
	return
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

	startTime := time.Now()
	resChan := make(chan *CTraderMessage, 1)
	ac.WaitMu.Lock()
	ac.Waiters[msg.ClientMsgID] = resChan
	ac.WaitMu.Unlock()

	defer func() {
		ac.WaitMu.Lock()
		delete(ac.Waiters, msg.ClientMsgID)
		ac.WaitMu.Unlock()
	}()

	log.Printf("[cTrader Manager Communication] SENDING Type: %d, ID: %s", msg.PayloadType, msg.ClientMsgID)

	if err := ac.Conn.WriteJSON(msg); err != nil {
		log.Printf("[cTrader Manager Communication] SEND ERROR Type: %d, ID: %s, Error: %v", msg.PayloadType, msg.ClientMsgID, err)
		return nil, err
	}

	select {
	case res := <-resChan:
		log.Printf("[cTrader Manager Communication] RECEIVED Type: %d (for Request %s), Took: %v", res.PayloadType, msg.ClientMsgID, time.Since(startTime))
		return res, nil
	case <-time.After(15 * time.Second):
		log.Printf("[cTrader Manager Communication] TIMEOUT for Request %s (Type %d) after 15s", msg.ClientMsgID, msg.PayloadType)
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
func (m *Manager) ManualSyncTrade(accID int64, ticket string) {
	log.Printf("[cTrader Manager] Manual sync requested for %s (Acc: %d)", ticket, accID)

	m.mu.RLock()
	ac, ok := m.connections[accID]
	m.mu.RUnlock()
	if !ok || ac == nil {
		log.Printf("[cTrader Manager] Manual sync failed: connection not found for acc %d", accID)
		return
	}

	// 1. Get Account Info (ctid)
	var ctidStr string
	err := m.db.QueryRow("SELECT ctrader_account_id FROM accounts WHERE id = ?", accID).Scan(&ctidStr)
	if err != nil {
		log.Printf("[cTrader Manager] Manual sync failed to query account %d: %v", accID, err)
		return
	}
	ctid, _ := strconv.ParseInt(ctidStr, 10, 64)

	// 2. Get Local Trade Details
	var entryPrice float64
	var entryTime time.Time
	var exitTime sql.NullTime
	var side string
	var lotSize float64
	var symbol string
	var existingPnL float64
	err = m.db.QueryRow(`
		SELECT entry_price, entry_time, exit_time, side, lot_size, symbol, pnl 
		FROM trades WHERE ticket = ? AND account_id = ?`,
		ticket, accID).Scan(&entryPrice, &entryTime, &exitTime, &side, &lotSize, &symbol, &existingPnL)

	if err != nil {
		log.Printf("[cTrader Manager] Manual sync failed to query trade %s: %v", ticket, err)
		return
	}

	// 3. Identification & Remote Fetch
	isDeal := strings.Contains(ticket, "deal")
	isPos := strings.Contains(ticket, "pos")
	var idFromTicket int64
	parts := strings.Split(ticket, "-")
	if len(parts) >= 3 {
		idFromTicket, _ = strconv.ParseInt(parts[len(parts)-1], 10, 64)
	}

	// Helper to find SymbolID
	var symbolID int64 = -1
	ac.Mu.RLock()
	for sid, sname := range ac.SymbolMap {
		if sname == symbol {
			symbolID = sid
			break
		}
	}
	ac.Mu.RUnlock()

	// --- A. SYNC DEAL (Closed or Historical) ---
	if isDeal && idFromTicket > 0 {
		// Search window: EntryTime - 7 days to Now (or ExitTime + 1 day)
		to := time.Now().UnixMilli()
		if exitTime.Valid {
			to = exitTime.Time.Add(24 * time.Hour).UnixMilli()
		}
		from := entryTime.Add(-7 * 24 * time.Hour).UnixMilli()

		log.Printf("[cTrader Manager] Syncing Deal %d (From: %d, To: %d)", idFromTicket, from, to)

		dResp, err := ac.SendRequest(CTraderMessage{
			ClientMsgID: fmt.Sprintf("sync-deal-%d", idFromTicket),
			PayloadType: PayloadDealListReq,
			Payload: func() json.RawMessage {
				b, _ := json.Marshal(map[string]interface{}{
					"ctidTraderAccountId": ctid,
					"fromTimestamp":       from,
					"toTimestamp":         to,
				})
				return b
			}(),
		})

		if err == nil {
			var dl struct {
				Deal []struct {
					DealID              int64   `json:"dealId"`
					SymbolID            int64   `json:"symbolId"`
					Volume              int64   `json:"volume"`
					ExecutionPrice      float64 `json:"executionPrice"`
					ExecutionTimestamp  int64   `json:"executionTimestamp"`
					TradeSide           int     `json:"tradeSide"`
					ClosePositionDetail struct {
						EntryPrice  float64 `json:"entryPrice"`
						GrossProfit int64   `json:"grossProfit"`
						Commission  int64   `json:"commission"`
						Swap        int64   `json:"swap"`
					} `json:"closePositionDetail"`
				} `json:"deal"`
			}
			json.Unmarshal(dResp.Payload, &dl)

			for _, d := range dl.Deal {
				if d.DealID == idFromTicket {
					// Found it! Update DB
					log.Printf("[cTrader Manager] Deal %d Found! Updating...", d.DealID)

					newEntryPrice := d.ClosePositionDetail.EntryPrice
					newExitPrice := d.ExecutionPrice
					newPnl := float64(d.ClosePositionDetail.GrossProfit+d.ClosePositionDetail.Commission+d.ClosePositionDetail.Swap) / 100.0
					newExitTime := time.UnixMilli(d.ExecutionTimestamp)

					// Update local vars for sparkline
					entryPrice = newEntryPrice
					// entryTime = ... (Usually entry time is on the OPENING deal, this is closing deal.
					// Ideally we query the opening deal too, but let's stick to closing details for PnL/Exit)
					exitTime = sql.NullTime{Time: newExitTime, Valid: true}
					if d.TradeSide == 1 {
						side = "short"
					} else {
						side = "long"
					} // Closing Buy(1) = Short
					log.Printf("[cTrader Manager] Sync Deal %d: Side Resolved to %s (TradeSide: %d)", d.DealID, side, d.TradeSide)

					// Note: LotSize needs symbol info to calculate from Volume
					if symbolID == -1 {
						symbolID = d.SymbolID
					}

					// Update DB
					_, dbErr := m.db.Exec(`UPDATE trades SET 
						entry_price = ?, exit_price = ?, pnl = ?, exit_time = ?, updated_at = CURRENT_TIMESTAMP
						WHERE ticket = ? AND account_id = ?`,
						newEntryPrice, newExitPrice, newPnl, newExitTime, ticket, accID)

					if dbErr != nil {
						log.Printf("[cTrader Manager] DB Update Failed: %v", dbErr)
					}
					break
				}
			}
		} else {
			log.Printf("[cTrader Manager] DealList Fetch Failed: %v", err)
		}
	} else if isPos && idFromTicket > 0 {
		// --- B. SYNC POSITION (Open) ---
		log.Printf("[cTrader Manager] Syncing Position %d", idFromTicket)
		pResp, err := ac.SendRequest(CTraderMessage{
			ClientMsgID: fmt.Sprintf("sync-pos-%d", idFromTicket),
			PayloadType: PayloadReconcileReq,
			Payload: func() json.RawMessage {
				b, _ := json.Marshal(map[string]interface{}{"ctidTraderAccountId": ctid})
				return b
			}(),
		})

		if err == nil {
			var p struct {
				Position []struct {
					PositionID int64   `json:"positionId"`
					Price      float64 `json:"price"`
					TradeData  struct {
						SymbolID      int64   `json:"symbolId"`
						Volume        int64   `json:"volume"`
						EntryPrice    float64 `json:"entryPrice"`
						OpenTimestamp int64   `json:"openTimestamp"`
						TradeSide     int     `json:"tradeSide"`
					} `json:"tradeData"`
				} `json:"position"`
			}
			json.Unmarshal(pResp.Payload, &p)

			found := false
			for _, pos := range p.Position {
				if pos.PositionID == idFromTicket {
					found = true
					log.Printf("[cTrader Manager] Position %d Found as Open! Updating...", pos.PositionID)

					newEntryPrice := pos.Price
					if newEntryPrice == 0 {
						newEntryPrice = pos.TradeData.EntryPrice
					}
					newEntryTime := time.UnixMilli(pos.TradeData.OpenTimestamp)

					// Update local vars
					entryPrice = newEntryPrice
					entryTime = newEntryTime
					if pos.TradeData.TradeSide == 2 {
						side = "short"
					} else {
						side = "long"
					}
					if symbolID == -1 {
						symbolID = pos.TradeData.SymbolID
					}

					// Update DB
					m.db.Exec(`UPDATE trades SET 
						entry_price = ?, entry_time = ?, updated_at = CURRENT_TIMESTAMP
						WHERE ticket = ? AND account_id = ?`,
						newEntryPrice, newEntryTime, ticket, accID)
					break
				}
			}

			if !found {
				log.Printf("[cTrader Manager] Position %d NOT found in Open Positions. Checking Deal history...", idFromTicket)

				// Search DealList window
				from := entryTime.Add(-24 * time.Hour).UnixMilli()
				to := time.Now().UnixMilli()

				dResp, derr := ac.SendRequest(CTraderMessage{
					ClientMsgID: fmt.Sprintf("sync-pos-closed-%d", idFromTicket),
					PayloadType: PayloadDealListReq,
					Payload: func() json.RawMessage {
						b, _ := json.Marshal(map[string]interface{}{"ctidTraderAccountId": ctid, "fromTimestamp": from, "toTimestamp": to})
						return b
					}(),
				})

				if derr == nil {
					var dl struct {
						Deal []struct {
							DealID              int64   `json:"dealId"`
							PositionID          int64   `json:"positionId"`
							SymbolID            int64   `json:"symbolId"`
							Volume              int64   `json:"volume"`
							ExecutionPrice      float64 `json:"executionPrice"`
							ExecutionTimestamp  int64   `json:"executionTimestamp"`
							TradeSide           int     `json:"tradeSide"`
							ClosePositionDetail struct {
								EntryPrice  float64 `json:"entryPrice"`
								GrossProfit int64   `json:"grossProfit"`
								Commission  int64   `json:"commission"`
								Swap        int64   `json:"swap"`
							} `json:"closePositionDetail"`
						} `json:"deal"`
					}
					json.Unmarshal(dResp.Payload, &dl)

					migratedCount := 0
					for _, d := range dl.Deal {
						if d.PositionID == idFromTicket && d.ClosePositionDetail.EntryPrice > 0 {
							log.Printf("[cTrader Manager] Found closing Deal %d for Position %d. Performing recovery migration.", d.DealID, d.PositionID)

							dealTicket := fmt.Sprintf("ctrader-deal-%d", d.DealID)

							// Get multiplier
							if symbolID == -1 {
								symbolID = d.SymbolID
							}
							ac.Mu.RLock()
							mFull := ac.SymbolLotSizeMap[symbolID]
							ac.Mu.RUnlock()
							if mFull == 0 {
								mFull = 100000
							}
							dealVol := float64(d.Volume) / float64(mFull)
							dealPnl := float64(d.ClosePositionDetail.GrossProfit+d.ClosePositionDetail.Commission+d.ClosePositionDetail.Swap) / 100.0

							// Migrate fields (journal, strategies, images, tags)
							var tradeID int64
							var journal, entryReason, entryStrategy, entryStrategyImg, entryStrategyImgOrig, entrySignals, entryChecklist, entryPattern, trendAnalysis, entryTimeframe, trendType, marketSession, colorTag, notes sql.NullString
							var legKingHTF, legKingImg, legKingImgOrig, legHTF, legHTFImg, legHTFImgOrig, legDeHTF, legImages sql.NullString
							var preservedSL sql.NullFloat64
							var preservedSeries sql.NullString

							m.db.QueryRow(`SELECT id, journal, entry_reason, entry_strategy, 
								entry_strategy_image, entry_strategy_image_original, entry_signals, entry_checklist, entry_pattern, 
								trend_analysis, entry_timeframe, trend_type, market_session, color_tag, notes,
								legend_king_htf, legend_king_image, legend_king_image_original, legend_htf, legend_htf_image, legend_htf_image_original, legend_de_htf, legend_images,
								initial_sl, pnl_series
								FROM trades WHERE ticket = ? AND account_id = ?`, ticket, accID).Scan(
								&tradeID, &journal, &entryReason, &entryStrategy,
								&entryStrategyImg, &entryStrategyImgOrig, &entrySignals, &entryChecklist, &entryPattern,
								&trendAnalysis, &entryTimeframe, &trendType, &marketSession, &colorTag, &notes,
								&legKingHTF, &legKingImg, &legKingImgOrig, &legHTF, &legHTFImg, &legHTFImgOrig, &legDeHTF, &legImages,
								&preservedSL, &preservedSeries)

							finalNotes := notes.String
							if finalNotes == "" || finalNotes == "cTrader Push: Initial Sync" {
								finalNotes = "cTrader Sync: Recovered Closed Position"
							}

							bullet, rr := 0.0, 0.0
							mult := getMultiplier(symbol)
							if preservedSL.Valid && preservedSL.Float64 > 0 && d.ClosePositionDetail.EntryPrice > 0 {
								riskPoints := math.Abs(d.ClosePositionDetail.EntryPrice - preservedSL.Float64)
								bullet = math.Round(riskPoints*mult*getPointValue(symbol)*dealVol*100) / 100

								// Signed PnL Points relative to side
								pnlPoints := (d.ExecutionPrice - d.ClosePositionDetail.EntryPrice) * mult
								if side == "short" {
									pnlPoints = -pnlPoints
								}
								pnlPoints = math.Round(pnlPoints*100) / 100

								if riskPoints > 0 {
									rr = math.Round((pnlPoints/(riskPoints*mult))*100) / 100
								}
							}

							// Check if deal already exists
							var exists int
							m.db.QueryRow("SELECT 1 FROM trades WHERE ticket = ? AND account_id = ?", dealTicket, accID).Scan(&exists)
							if exists == 0 {
								res, ierr := m.db.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes, ticket, initial_sl, exit_sl, bullet_size, rr_ratio, pnl_series, journal, entry_reason, entry_strategy, entry_strategy_image, entry_strategy_image_original, entry_signals, entry_checklist, entry_pattern, trend_analysis, entry_timeframe, trend_type, market_session, color_tag, legend_king_htf, legend_king_image, legend_king_image_original, legend_htf, legend_htf_image, legend_htf_image_original, legend_de_htf, legend_images)
									VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
									accID, symbol, side, d.ClosePositionDetail.EntryPrice, d.ExecutionPrice, dealVol, dealPnl, entryTime, time.UnixMilli(d.ExecutionTimestamp), "actual", finalNotes, dealTicket, preservedSL, d.ExecutionPrice, bullet, rr, preservedSeries.String, journal, entryReason, entryStrategy, entryStrategyImg, entryStrategyImgOrig, entrySignals, entryChecklist, entryPattern, trendAnalysis, entryTimeframe, trendType, marketSession, colorTag, legKingHTF, legKingImg, legKingImgOrig, legHTF, legHTFImg, legHTFImgOrig, legDeHTF, legImages)

								if ierr == nil {
									newID, _ := res.LastInsertId()
									if tradeID > 0 && newID > 0 {
										m.db.Exec("INSERT INTO trade_images (trade_id, image_type, image_path, file_size, description) SELECT ?, image_type, image_path, file_size, description FROM trade_images WHERE trade_id = ?", newID, tradeID)
										m.db.Exec("INSERT INTO trade_tags (trade_id, tag_id) SELECT ?, tag_id FROM trade_tags WHERE trade_id = ?", newID, tradeID)
									}
								}
							}
							migratedCount++
						}
					}

					if migratedCount > 0 {
						m.db.Exec("DELETE FROM trades WHERE ticket = ? AND account_id = ?", ticket, accID)
						log.Printf("[cTrader Manager] Stuck position record %s removed, successfully migrated %d deals", ticket, migratedCount)
						ws.GlobalHub.BroadcastUpdate(accID, "TRADE_UPDATE")
						return
					}
				}
			}
		}
	}

	if symbolID == -1 {
		log.Printf("[cTrader Manager] Manual sync failed: symbol ID not found locally for %s", symbol)
		return
	}

	// 4. Regenerate Sparkline (using potentially updated values)
	startMilli := entryTime.UnixMilli()
	endMilli := time.Now().UnixMilli()
	if exitTime.Valid {
		endMilli = exitTime.Time.UnixMilli()
	}

	// Adjust vol (lotSize) - Gold multiplier consideration
	vol := int64(lotSize * 100000)
	if strings.Contains(strings.ToUpper(symbol), "XAU") || strings.Contains(strings.ToUpper(symbol), "GOLD") {
		vol = int64(lotSize * 100)
	}

	go func() {
		m.mu.RLock()
		ac, ok := m.connections[accID]
		m.mu.RUnlock()
		if ok && ac != nil {
			sInt := 1
			if side == "short" {
				sInt = 2
			}
			digits := 2
			if d, ok := m.symbolDigitsMap.Load(symbolID); ok {
				digits = d.(int)
			}
			newSeriesStr := fetchPnLSeries(ac, ac.Conn, ctid, symbolID, startMilli, endMilli, entryPrice, vol, sInt, digits)
			if newSeriesStr != "" {
				m.db.Exec("UPDATE trades SET pnl_series = ? WHERE ticket = ?", newSeriesStr, ticket)
			}
		}
	}()
}
