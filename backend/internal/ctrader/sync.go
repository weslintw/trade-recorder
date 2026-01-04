package ctrader

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	CTraderLiveURL = "wss://live.ctraderapi.com:5036"
	CTraderDemoURL = "wss://demo.ctraderapi.com:5036"
	
	PayloadAppAuthReq     = 2100
	PayloadAppAuthRes     = 2101
	PayloadAccountAuthReq = 2102
	PayloadAccountAuthRes = 2103
	PayloadSymbolsListReq = 2114
	PayloadSymbolsListRes = 2115
	PayloadSymbolByIdReq  = 2116
	PayloadSymbolByIdRes  = 2117
	PayloadReconcileReq   = 2124
	PayloadReconcileRes   = 2125
	PayloadDealListReq    = 2133
	PayloadDealListRes    = 2134
	PayloadOrderListReq   = 2175
	PayloadOrderListRes   = 2176
	PayloadOrderDetailsReq = 2181
	PayloadOrderDetailsRes = 2182
	PayloadOrderListByPositionIdReq = 2183
	PayloadOrderListByPositionIdRes = 2184
	PayloadHeartbeatEvent = 51
	PayloadExecutionEvent = 2126
	PayloadErrorRes       = 2142
)

type CTraderMessage struct {
	ClientMsgID string          `json:"clientMsgId,omitempty"`
	PayloadType uint32          `json:"payloadType"`
	Payload     json.RawMessage `json:"payload"`
}

func SyncCTraderHistory(db *sql.DB, accountID int64, cTraderAccountID string, token string, clientID string, clientSecret string, env string) error {
	log.Printf("[cTrader Sync] --- Manual Sync START for Account %d (v2.27) ---", accountID)
	// Clear existing trades for this account to start fresh and avoid conflicts
	db.Exec("DELETE FROM trades WHERE account_id = ?", accountID)
	db.Exec("UPDATE accounts SET sync_status = 'syncing (Preparing)...', last_sync_error = '', updated_at = CURRENT_TIMESTAMP WHERE id = ?", accountID)
	
	if GlobalManager != nil {
		GlobalManager.StopListener(accountID)
		time.Sleep(1 * time.Second)
	}

	err := internalSync(db, accountID, cTraderAccountID, token, clientID, clientSecret, env)
	if err != nil {
		log.Printf("[cTrader Sync] FAILED: %v", err)
		db.Exec("UPDATE accounts SET sync_status = 'failed', last_sync_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", err.Error(), accountID)
		return err
	}

	log.Printf("[cTrader Sync] --- Manual Sync SUCCESS for Account %d ---", accountID)
	db.Exec("UPDATE accounts SET sync_status = 'success', last_sync_error = '', last_synced_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", accountID)
	return nil
}

func getMultiplier(symbol string) float64 {
	if strings.Contains(symbol, "JPY") { return 100.0 } // 2 decimals JPY = 100 pips per $1
	if strings.Contains(symbol, "XAU") || strings.Contains(symbol, "GOLD") || strings.Contains(symbol, "XPT") { return 1.0 } // Gold: 1.0 = 1 point
	if strings.Contains(symbol, "NAS") || strings.Contains(symbol, "US30") || strings.Contains(symbol, "SPD") || strings.Contains(symbol, "HSI") { return 1.0 } // Indices: 1.0 = 1 point
	// Forex pairs: 4 decimals = 10000 points (Pips)
	if len(symbol) >= 6 && (strings.Contains(symbol, "USD") || strings.Contains(symbol, "EUR") || strings.Contains(symbol, "GBP")) { return 10000.0 }
	return 1.0 
}

type dealInfo struct {
	DealID int64; OrderID int64; SymbolID int64; Volume int64; ExecutionPrice float64; ExecutionTimestamp int64; TradeSide int; PositionID int64; ClosePositionDetail struct { EntryPrice float64; GrossProfit int64; Commission int64; Swap int64; StopLoss float64 `json:"stopLoss"` }
}

type orderInfo struct {
	OrderID int64 `json:"orderId"`; PositionID int64 `json:"positionId"`; StopLoss float64 `json:"stopLoss"`; StopPrice float64 `json:"stopPrice"`; TradeTimestamp int64 `json:"utcLastUpdateTimestamp"`; TradeData struct { OpenTimestamp int64 `json:"openTimestamp"` } `json:"tradeData"`
}

func internalSync(db *sql.DB, accountID int64, cTraderAccountIDStr string, token string, clientID string, clientSecret string, env string) error {
	cTID, _ := strconv.ParseInt(cTraderAccountIDStr, 10, 64)
	url := CTraderLiveURL; if env == "demo" { url = CTraderDemoURL }
	
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil { return fmt.Errorf("dial failed: %v", err) }
	defer conn.Close()

	// 1. Auth sequence
	time.Sleep(500 * time.Millisecond)
	if err = sendAndVerify(conn, PayloadAppAuthReq, map[string]string{"clientId": clientID, "clientSecret": clientSecret}, PayloadAppAuthRes); err != nil { return err }
	time.Sleep(500 * time.Millisecond)
	if err = sendAndVerify(conn, PayloadAccountAuthReq, map[string]interface{}{"ctidTraderAccountId": cTID, "accessToken": token}, PayloadAccountAuthRes); err != nil { return err }

	symbolMap := make(map[int64]string)
	symbolLotSizeMap := make(map[int64]int64)

	// Populate symbol names
	time.Sleep(500 * time.Millisecond)
	sListResp, err := sendRequest(conn, PayloadSymbolsListReq, map[string]interface{}{"ctidTraderAccountId": cTID})
	if err == nil {
		var p struct { Symbol []struct { SymbolID int64 `json:"symbolId"`; SymbolName string `json:"symbolName"` } `json:"symbol"` }
		json.Unmarshal(sListResp.Payload, &p)
		for _, s := range p.Symbol { symbolMap[s.SymbolID] = s.SymbolName }
	}

	updateMetadata := func(sids []int64) {
		if len(sids) == 0 { return }
		var needed []int64
		for _, id := range sids { if _, ok := symbolLotSizeMap[id]; !ok { needed = append(needed, id) } }
		if len(needed) == 0 { return }
		time.Sleep(200 * time.Millisecond)
		resp, err := sendRequest(conn, PayloadSymbolByIdReq, map[string]interface{}{"ctidTraderAccountId": cTID, "symbolId": needed})
		if err == nil {
			var p struct { Symbol []struct { SymbolID int64 `json:"symbolId"`; SymbolName string `json:"symbolName"`; LotSize int64 `json:"lotSize"` } `json:"symbol"` }
			json.Unmarshal(resp.Payload, &p)
			for _, s := range p.Symbol {
				if s.SymbolName != "" { symbolMap[s.SymbolID] = s.SymbolName }
				symbolLotSizeMap[s.SymbolID] = s.LotSize
			}
		}
	}

	// 2. Data Collection (Batch Fetching Deals & Orders)
	now := time.Now()
	allDeals := []dealInfo{}
	allSids := []int64{}
	orderHistoryMap := make(map[int64][]orderInfo) // PositionID -> Orders

	log.Printf("[cTrader Sync] Step 2: Collecting history (Bulk v2.34)...")
	// Fetch 120 days of orders to cover cases where entries are older than 90-day deals
	for i := 0; i < 8; i++ { 
		db.Exec("UPDATE accounts SET sync_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", fmt.Sprintf("fetching history (%d/8)...", i+1), accountID)
		to := now.AddDate(0, 0, -15*(i)).UnixMilli()
		from := now.AddDate(0, 0, -15*(i+1)).UnixMilli()
		time.Sleep(300 * time.Millisecond)
		
		// Fetch Deals
		dResp, dErr := sendRequest(conn, PayloadDealListReq, map[string]interface{}{"ctidTraderAccountId": cTID, "fromTimestamp": from, "toTimestamp": to})
		if dErr == nil {
			var p struct { Deal []dealInfo `json:"deal"` }
			if err := json.Unmarshal(dResp.Payload, &p); err == nil { 
				allDeals = append(allDeals, p.Deal...)
				for _, d := range p.Deal { allSids = append(allSids, d.SymbolID) }
			}
		}

		// Fetch Orders (Bulk fetch to avoid hundreds of individual calls)
		time.Sleep(200 * time.Millisecond)
		oResp, oErr := sendRequest(conn, PayloadOrderListReq, map[string]interface{}{"ctidTraderAccountId": cTID, "fromTimestamp": from, "toTimestamp": to})
		if oErr == nil {
			var p struct { Order []orderInfo `json:"order"` }
			if err := json.Unmarshal(oResp.Payload, &p); err == nil {
				for _, o := range p.Order {
					orderHistoryMap[o.PositionID] = append(orderHistoryMap[o.PositionID], o)
				}
			}
		}
	}

	updateMetadata(allSids)

	// Group deals by PositionID
	posGroups := make(map[int64][]dealInfo)
	for _, d := range allDeals { posGroups[d.PositionID] = append(posGroups[d.PositionID], d) }

	// 3. Process Positions with Hybrid SL Search
	log.Printf("[cTrader Sync] Step 3: Processing %d positions (v2.27)...", len(posGroups))
	count := 0
	var tx *sql.Tx
	for pid, deals := range posGroups {
		count++
		// Commit/Start fresh transaction every 50 items to prevent locking & show progress
		if count % 50 == 1 {
			if count > 1 && tx != nil { tx.Commit() }
			tx, _ = db.Begin()
		}

		db.Exec("UPDATE accounts SET sync_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", fmt.Sprintf("scanning SL (%d/%d)...", count, len(posGroups)), accountID)
		
		sort.Slice(deals, func(i, j int) bool { return deals[i].ExecutionTimestamp < deals[j].ExecutionTimestamp })
		entryTime := deals[0].ExecutionTimestamp
		openingOrderID := deals[0].OrderID // The order that created this position
		
		// HYBRID SL SEARCH: Collect all unique SLs (with timestamps) + find authoritative Initial
		initialSL := 0.0
		type slEntry struct {
			Price float64 `json:"price"`
			Time  int64   `json:"time"`
		}
		allSLEntries := []slEntry{}
		orderSLMap := make(map[int64]float64)
		
		addSL := func(sl float64, t int64, orderID int64) {
			if sl <= 0 { return }
			found := false
			for i, existing := range allSLEntries {
				if math.Abs(existing.Price-sl) < 0.00001 {
					if t < existing.Time { allSLEntries[i].Time = t }
					found = true
					break
				}
			}
			if !found { allSLEntries = append(allSLEntries, slEntry{Price: sl, Time: t}) }
			
			// AUTHORITATIVE INITIAL SL: If this SL is from the order that opened the position, it's the winner
			if orderID == openingOrderID {
				initialSL = sl
				log.Printf("[SL DEBUG] -> Identified Initial SL (Authoritative Opening Order): %.5f", sl)
			}
		}

		
		log.Printf("[SL DEBUG] Position %d | EntryTime: %d | OpeningOrderID: %d", pid, entryTime, openingOrderID)

		// 1. ALWAYS fetch the opening order details directly (HIGHEST PRIORITY)
		time.Sleep(5 * time.Millisecond)
		odResp, odErr := sendRequest(conn, PayloadOrderDetailsReq, map[string]interface{}{
			"ctidTraderAccountId": cTID,
			"orderId": openingOrderID,
		})
		if odErr == nil {
			var od struct { Order struct { OrderID int64 `json:"orderId"`; StopLoss float64 `json:"stopLoss"`; StopPrice float64 `json:"stopPrice"`; TradeData struct { OpenTimestamp int64 `json:"openTimestamp"` } `json:"tradeData"` } `json:"order"` }
			json.Unmarshal(odResp.Payload, &od)
			sl := od.Order.StopLoss
			if sl == 0 { sl = od.Order.StopPrice }
			if sl > 0 {
				initialSL = sl
				addSL(sl, entryTime, openingOrderID)
				log.Printf("[SL DEBUG] ✓ Opening Order Direct Fetch: SL=%.5f (Order %d)", sl, openingOrderID)
			} else {
				log.Printf("[SL DEBUG] ✗ Opening Order has NO SL (Order %d)", openingOrderID)
			}
		} else {
			log.Printf("[SL DEBUG] ✗ Failed to fetch Opening Order details (Order %d): %v", openingOrderID, odErr)
		}

		// STEP 2: Check Bulk History (for SL modification timeline)
		history := orderHistoryMap[pid]
		sort.Slice(history, func(i, j int) bool { return history[i].TradeTimestamp < history[j].TradeTimestamp })
		for _, o := range history {
			sl := o.StopLoss; if sl == 0 { sl = o.StopPrice }
			if sl > 0 {
				log.Printf("[SL DEBUG] Bulk Order: ID=%d, SL=%.5f, Time=%d, Diff=%d", o.OrderID, sl, o.TradeTimestamp, o.TradeTimestamp-entryTime)
				if o.TradeData.OpenTimestamp > 0 && math.Abs(float64(o.TradeTimestamp-o.TradeData.OpenTimestamp)) > 60000 {
					isModified = true
				}

				// If Modified, use TradeTimestamp (Update Time) for history to reflect reality.
				// If NOT Modified, usage OpenTimestamp (Creation Time) to place it at start.
				addTime := o.TradeTimestamp
				if o.TradeData.OpenTimestamp > 0 && !isModified { addTime = o.TradeData.OpenTimestamp }
				
				// Debug logging for verification
				if isModified {
					log.Printf("[SL DEBUG] Modified Order %d: OpenTS=%d, UpdateTS=%d. Using UpdateTS for history.", o.OrderID, o.TradeData.OpenTimestamp, o.TradeTimestamp)
				}
				
				addSL(sl, addTime, o.OrderID)
				
				// Standard Check: Time Window (Creation or Update)
				// If modified, we check Current Time (Update), likely failing the 60s window -> Correct.
				// If not modified, we check Creation Time, passing the window -> Correct.
				checkTime := addTime

				if initialSL == 0 && math.Abs(float64(checkTime - entryTime)) <= 60000 { 
					if isModified {
						log.Printf("[SL DEBUG] -> Initial SL Order FOUND but MODIFIED (Diff > 60s). Skipping auto-populate. SL: %.5f", sl)
					} else {
						initialSL = sl 
						log.Printf("[SL DEBUG] -> Initial SL from Bulk (within 60s, CreationTS=%v): %.5f", o.TradeData.OpenTimestamp > 0, sl)
					}
				}
				orderSLMap[o.OrderID] = sl
			}
		}

		// STEP 3: Targeted Backtrace for additional history
		if true {
			time.Sleep(10 * time.Millisecond)
			exitTime := deals[len(deals)-1].ExecutionTimestamp
			olResp, olErr := sendRequest(conn, PayloadOrderListByPositionIdReq, map[string]interface{}{
				"ctidTraderAccountId": cTID, 
				"positionId": pid,
				"fromTimestamp": entryTime - 25*3600000,
				"toTimestamp": exitTime + 7200000,
			})
			if olErr == nil {
				var op struct { Order []struct { OrderID int64 `json:"orderId"`; StopLoss float64 `json:"stopLoss"`; StopPrice float64 `json:"stopPrice"`; TradeTimestamp int64 `json:"utcLastUpdateTimestamp"`; TradeData struct { OpenTimestamp int64 `json:"openTimestamp"` } `json:"tradeData"` } `json:"order"` }
				json.Unmarshal(olResp.Payload, &op)
				if len(op.Order) > 0 {
					sort.Slice(op.Order, func(i, j int) bool { return op.Order[i].TradeTimestamp < op.Order[j].TradeTimestamp })
					for _, o := range op.Order {
						sl := o.StopLoss; if sl == 0 { sl = o.StopPrice }
						if sl > 0 {
							if o.TradeData.OpenTimestamp > 0 && math.Abs(float64(o.TradeTimestamp-o.TradeData.OpenTimestamp)) > 60000 {
								isModified = true
							}

							// If Modified, use TradeTimestamp (Update Time) for history.
							addTime := o.TradeTimestamp
							if o.TradeData.OpenTimestamp > 0 && !isModified { addTime = o.TradeData.OpenTimestamp }
							
							addSL(sl, addTime, o.OrderID)
							
							// Initial SL Check target
							checkTime := addTime

							if initialSL == 0 && math.Abs(float64(checkTime - entryTime)) <= 60000 { 
								if isModified {
									log.Printf("[SL DEBUG] -> Initial SL Order FOUND but MODIFIED (Diff > 60s). Skipping auto-populate. SL: %.5f", sl)
								} else {
									initialSL = sl 
									log.Printf("[SL DEBUG] -> Initial SL from Targeted (within 60s, CreationTS=%v): %.5f", o.TradeData.OpenTimestamp > 0, sl)
								}
							}
							orderSLMap[o.OrderID] = sl
						}
					}
				}
			} else {
				log.Printf("[SL DEBUG] Targeted request failed for PID %d: %v", pid, olErr)
			}
		}

		// If initialSL is still 0, try to find the earliest SL within reasoning distance
	if initialSL == 0 && len(allSLEntries) > 0 {
		sort.Slice(allSLEntries, func(i, j int) bool {
			return allSLEntries[i].Time < allSLEntries[j].Time
		})
		
		// Final Fallback must also respect Time Window (e.g. 60s or slightly looser 5min?)
		// Let's stick to 60s Strict (v2.47 user request)
		earliest := allSLEntries[0]
		if math.Abs(float64(earliest.Time - entryTime)) <= 60000 {
			initialSL = earliest.Price
			log.Printf("[SL DEBUG] -> Initial SL fallback (Earliest valid seen): %.5f", initialSL)
		} else {
			log.Printf("[SL DEBUG] -> Earliest SL found (%.5f) is too far from entry (%d ms). Leaving Initial SL empty.", earliest.Price, earliest.Time - entryTime)
		}
	}
		slHistoryJSON, _ := json.Marshal(allSLEntries)

		// Process each closed deal in this position
		for _, d := range deals {
			if d.ClosePositionDetail.EntryPrice == 0 { continue }
			symbol := symbolMap[d.SymbolID]; if symbol == "" { symbol = "Unknown" }
			lotSize := symbolLotSizeMap[d.SymbolID]; if lotSize == 0 { lotSize = 100000 }
			side := "long"; if d.TradeSide == 1 { side = "short" }
			vol := float64(d.Volume) / float64(lotSize)
			pnl := float64(d.ClosePositionDetail.GrossProfit + d.ClosePositionDetail.Commission + d.ClosePositionDetail.Swap) / 100.0
			ticket := fmt.Sprintf("ctrader-deal-%d", d.DealID)
			
			exitSL := d.ClosePositionDetail.StopLoss
			if exitSL == 0 { exitSL = orderSLMap[d.OrderID] }
			
			bullet, rr := 0.0, 0.0
			mult := getMultiplier(symbol)
			if initialSL > 0 && d.ClosePositionDetail.EntryPrice > 0 {
				bullet = math.Round(math.Abs(d.ClosePositionDetail.EntryPrice - initialSL) * mult * 100) / 100
				
				// Signed PnL Points relative to side
				pnlPoints := (d.ExecutionPrice - d.ClosePositionDetail.EntryPrice) * mult
				if side == "short" { pnlPoints = -pnlPoints }
				pnlPoints = math.Round(pnlPoints * 100) / 100
				
				if bullet > 0 { rr = math.Round((pnlPoints / bullet) * 100) / 100 }
			}

			tx.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, exit_price, lot_size, pnl, entry_time, exit_time, trade_type, notes, ticket, initial_sl, exit_sl, bullet_size, rr_ratio, sl_history)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				accountID, symbol, side, d.ClosePositionDetail.EntryPrice, d.ExecutionPrice, vol, pnl, time.UnixMilli(entryTime), time.UnixMilli(d.ExecutionTimestamp), "actual", "cTrader Sync", ticket, initialSL, exitSL, bullet, rr, string(slHistoryJSON))
		}
	}
	if count > 0 && tx != nil { tx.Commit() }

	// 4. Open Positions Sync
	log.Printf("[cTrader Sync] Step 4: Open Positions (v2.28)...")
	pResp, err := sendRequest(conn, PayloadReconcileReq, map[string]interface{}{"ctidTraderAccountId": cTID})
	if err == nil {
		var p struct { Position []struct { 
			PositionID int64 `json:"positionId"`; Price float64 `json:"price"`; StopLoss float64 `json:"stopLoss"`;
			TradeData struct { SymbolID int64 `json:"symbolId"`; Volume int64 `json:"volume"`; TradeSide int `json:"tradeSide"`; EntryTimestamp int64 `json:"entryTimestamp"` } `json:"tradeData"`
		} `json:"position"` }
		if err := json.Unmarshal(pResp.Payload, &p); err == nil {
			countOpen := 0
			var tx *sql.Tx
			for _, pos := range p.Position {
				countOpen++
				if countOpen % 50 == 1 {
					if countOpen > 1 && tx != nil { tx.Commit() }
					tx, _ = db.Begin()
				}
				db.Exec("UPDATE accounts SET sync_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", fmt.Sprintf("syncing open positions (%d/%d)...", countOpen, len(p.Position)), accountID)

				symbol := symbolMap[pos.TradeData.SymbolID]; if symbol == "" { symbol = "Unknown" }
				initialSL := 0.0
				type slEntry struct {
					Price float64 `json:"price"`
					Time  int64   `json:"time"`
				}
				allSLEntries := []slEntry{}
				addSL := func(sl float64, t int64) {
					if sl <= 0 { return }
					for _, existing := range allSLEntries { if existing.Price == sl { return } }
					allSLEntries = append(allSLEntries, slEntry{Price: sl, Time: t})
				}
				
				// Try bulk first
				history := orderHistoryMap[pos.PositionID]
				sort.Slice(history, func(i, j int) bool { return history[i].TradeTimestamp < history[j].TradeTimestamp })
				for _, o := range history {
					sl := o.StopLoss; if sl == 0 { sl = o.StopPrice }
					if sl > 0 {
						addSL(sl, o.TradeTimestamp)
						if initialSL == 0 && math.Abs(float64(o.TradeTimestamp - pos.TradeData.EntryTimestamp)) <= 2000 { initialSL = sl }
					}
				}

				// Targeted fallback with explicit window
				time.Sleep(20 * time.Millisecond)
				olResp, olErr := sendRequest(conn, PayloadOrderListByPositionIdReq, map[string]interface{}{
					"ctidTraderAccountId": cTID, 
					"positionId": pos.PositionID,
					"fromTimestamp": pos.TradeData.EntryTimestamp - 24*3600000,
					"toTimestamp": time.Now().UnixMilli() + 3600000,
				})
				if olErr == nil {
					var op struct { Order []struct { StopLoss float64 `json:"stopLoss"`; StopPrice float64 `json:"stopPrice"`; TradeTimestamp int64 `json:"utcLastUpdateTimestamp"` } `json:"order"` }
					json.Unmarshal(olResp.Payload, &op)
					if len(op.Order) > 0 {
						sort.Slice(op.Order, func(i, j int) bool { return op.Order[i].TradeTimestamp < op.Order[j].TradeTimestamp })
						for _, o := range op.Order {
							sl := o.StopLoss; if sl == 0 { sl = o.StopPrice }
							if sl > 0 {
								addSL(sl, o.TradeTimestamp)
								if initialSL == 0 && math.Abs(float64(o.TradeTimestamp - pos.TradeData.EntryTimestamp)) <= 2000 { initialSL = sl }
							}
						}
					}
				}
				
				if initialSL == 0 && len(allSLEntries) > 0 { initialSL = allSLEntries[0].Price }
				slHistoryJSON, _ := json.Marshal(allSLEntries)

				bullet := 0.0; mult := getMultiplier(symbol)
				if initialSL > 0 && pos.Price > 0 { bullet = math.Round(math.Abs(pos.Price - initialSL) * mult * 100) / 100 }

				ticket := fmt.Sprintf("ctrader-pos-%d", pos.PositionID)
				lotSize := symbolLotSizeMap[pos.TradeData.SymbolID]; if lotSize == 0 { lotSize = 100000 }
				side := "long"; if pos.TradeData.TradeSide == 2 { side = "short" }
				vol := float64(pos.TradeData.Volume) / float64(lotSize)
				
				tx.Exec(`INSERT INTO trades (account_id, symbol, side, entry_price, lot_size, entry_time, trade_type, notes, ticket, initial_sl, exit_sl, bullet_size, sl_history)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					accountID, symbol, side, pos.Price, vol, time.UnixMilli(pos.TradeData.EntryTimestamp), "actual", "cTrader Open", ticket, initialSL, pos.StopLoss, bullet, string(slHistoryJSON))
			}
			if countOpen > 0 && tx != nil { tx.Commit() }
		}
	}
	log.Printf("[cTrader Sync] --- Manual Sync SUCCESS for Account %d (v2.28) ---", accountID)
	return nil
}

func sendRequest(conn *websocket.Conn, payloadType uint32, payload interface{}) (*CTraderMessage, error) {
	clientMsgID := fmt.Sprintf("m-%d", time.Now().UnixNano())
	payloadJSON, _ := json.Marshal(payload)
	msg := CTraderMessage{ClientMsgID: clientMsgID, PayloadType: payloadType, Payload: payloadJSON}
	if err := conn.WriteJSON(msg); err != nil { return nil, err }
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil { return nil, err }
		var resp CTraderMessage
		if err := json.Unmarshal(raw, &resp); err != nil { continue }
		if resp.PayloadType == PayloadHeartbeatEvent { continue }
		if resp.PayloadType == PayloadErrorRes { return nil, fmt.Errorf("cTrader Error: %s", string(resp.Payload)) }
		if resp.ClientMsgID == clientMsgID || 
		   (resp.PayloadType == PayloadAppAuthRes && payloadType == PayloadAppAuthReq) || 
		   (resp.PayloadType == PayloadAccountAuthRes && payloadType == PayloadAccountAuthReq) ||
		   (resp.PayloadType == PayloadOrderDetailsRes && payloadType == PayloadOrderDetailsReq) ||
		   (resp.PayloadType == PayloadOrderListByPositionIdRes && payloadType == PayloadOrderListByPositionIdReq) { 
			return &resp, nil 
		}
	}
}

func sendAndVerify(conn *websocket.Conn, payloadType uint32, payload interface{}, expected uint32) error {
	resp, err := sendRequest(conn, payloadType, payload); if err != nil { return err }
	if resp.PayloadType != expected { return fmt.Errorf("expected %d got %d", expected, resp.PayloadType) }
	return nil
}
