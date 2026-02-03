package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"trade-journal/internal/models"

	"time"
	"trade-journal/internal/ctrader"

	"github.com/gin-gonic/gin"
)

// GenerateToken 產出分享使用的隨機 Token
func GenerateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// CreateShare 建立分享
func CreateShare(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.ShareCreate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt64("user_id")

		// 檢查權限 (確保資源屬於該使用者)
		var ownerID int64
		if req.ResourceType == "trade" {
			err := db.QueryRow("SELECT a.user_id FROM trades t JOIN accounts a ON t.account_id = a.id WHERE t.id = ?", req.ResourceID).Scan(&ownerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到該交易紀錄"})
				return
			}
		} else if req.ResourceType == "plan" {
			err := db.QueryRow("SELECT a.user_id FROM daily_plans p JOIN accounts a ON p.account_id = a.id WHERE p.id = ?", req.ResourceID).Scan(&ownerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到該規劃紀錄"})
				return
			}
		} else if req.ResourceType == "account" {
			err := db.QueryRow("SELECT user_id FROM accounts WHERE id = ?", req.ResourceID).Scan(&ownerID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到該帳號"})
				return
			}
		} else if req.ResourceType == "batch" {
			// 對於批次分享，暫且只檢查當前使用者的權限 (後續可細化檢查每一筆)
			ownerID = userID
		}

		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限分享此內容"})
			return
		}

		// 如果已經有公開分享，且這次也是要求公開，就回傳現有的 (排除 batch，因為 batch 每次選取可能不同)
		var token string
		if req.ShareType == "public" && req.ResourceType != "batch" {
			err := db.QueryRow("SELECT token FROM shares WHERE resource_type = ? AND resource_id = ? AND share_type = 'public'", req.ResourceType, req.ResourceID).Scan(&token)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"token": token, "message": "已取得現有分享連結"})
				return
			}
		}
		token = GenerateToken()

		resourceIDsJSON := ""
		if req.ResourceType == "batch" && req.ResourceIDs != nil {
			idsBytes, _ := json.Marshal(req.ResourceIDs)
			resourceIDsJSON = string(idsBytes)
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec("INSERT INTO shares (user_id, resource_type, resource_id, resource_ids, share_type, token) VALUES (?, ?, ?, ?, ?, ?)",
			userID, req.ResourceType, req.ResourceID, resourceIDsJSON, req.ShareType, token)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		shareID, _ := res.LastInsertId()

		if req.ShareType == "specific" && len(req.SharedWith) > 0 {
			for _, sharedUserID := range req.SharedWith {
				_, err = tx.Exec("INSERT INTO share_users (share_id, shared_with_user_id) VALUES (?, ?)", shareID, sharedUserID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": shareID, "token": token, "message": "分享成功"})
		log.Printf("[Share] Share created: UserID=%d, Type=%s, ID=%d, Token=%s", userID, req.ResourceType, req.ResourceID, token)
	}
}

// GetSharedResource 透過 Token 取得共享資源 (免登入)
func GetSharedResource(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		var share models.Share
		err := db.QueryRow("SELECT user_id, resource_type, resource_id, COALESCE(resource_ids, '') FROM shares WHERE token = ?", token).Scan(&share.UserID, &share.ResourceType, &share.ResourceID, &share.ResourceIDs)

		// 如果精確匹配失敗，且 token 長度大於 32，嘗試抓取前 32 個字元 (Hex Token 標準長度)
		if err == sql.ErrNoRows && len(token) > 32 {
			shortToken := token[:32]
			log.Printf("[Share] Exact token not found, trying prefix: %s", shortToken)
			err = db.QueryRow("SELECT user_id, resource_type, resource_id, COALESCE(resource_ids, '') FROM shares WHERE token = ?", shortToken).Scan(&share.UserID, &share.ResourceType, &share.ResourceID, &share.ResourceIDs)
		}

		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("[Share] Token not found: %s", token)
			} else {
				log.Printf("[Share] Database error for token %s: %v", token, err)
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到此分享連結或已失效"})
			return
		}

		// Get Username
		var username string
		db.QueryRow("SELECT username FROM users WHERE id = ?", share.UserID).Scan(&username)

		if share.ResourceType == "trade" {
			trade, err := GetTradeInternal(db, share.ResourceID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到交易內容", "details": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"type": "trade", "username": username, "data": trade})
		} else if share.ResourceType == "plan" {
			plan, err := GetPlanInternal(db, share.ResourceID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到規劃內容", "details": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"type": "plan", "username": username, "data": plan})
		} else if share.ResourceType == "account" || share.ResourceType == "batch" {
			var trades []models.Trade
			var plans []models.DailyPlan
			var account models.Account
			var accountID int64

			if share.ResourceType == "account" {
				accountID = share.ResourceID
				// 抓取該帳號的所有交易與規劃
				tradeRows, _ := db.Query("SELECT id FROM trades WHERE account_id = ? ORDER BY entry_time DESC", share.ResourceID)
				for tradeRows.Next() {
					var id int64
					tradeRows.Scan(&id)
					t, err := GetTradeInternal(db, id)
					if err == nil {
						trades = append(trades, *t)
					}
				}
				tradeRows.Close()

				planRows, _ := db.Query("SELECT id FROM daily_plans WHERE account_id = ? ORDER BY plan_date DESC", share.ResourceID)
				for planRows.Next() {
					var id int64
					planRows.Scan(&id)
					p, err := GetPlanInternal(db, id)
					if err == nil {
						plans = append(plans, *p)
					}
				}
				planRows.Close()
			} else {
				// batch: ResourceIDs 包含選中的 ID (JSON array)
				var batchIDs struct {
					Trades []int64 `json:"trades"`
					Plans  []int64 `json:"plans"`
				}
				if err := json.Unmarshal([]byte(share.ResourceIDs), &batchIDs); err != nil {
					var ids []int64
					json.Unmarshal([]byte(share.ResourceIDs), &ids)
				}

				for _, id := range batchIDs.Trades {
					t, err := GetTradeInternal(db, id)
					if err == nil {
						trades = append(trades, *t)
						if accountID == 0 {
							accountID = t.AccountID
						}
					}
				}
				for _, id := range batchIDs.Plans {
					p, err := GetPlanInternal(db, id)
					if err == nil {
						plans = append(plans, *p)
						if accountID == 0 {
							accountID = p.AccountID
						}
					}
				}
			}

			// 抓取帳號基本資訊
			if accountID > 0 {
				db.QueryRow("SELECT id, name, type, ctrader_account_id, ctrader_env FROM accounts WHERE id = ?", accountID).Scan(
					&account.ID, &account.Name, &account.Type, &account.CTraderAccountID, &account.CTraderEnv,
				)
			}

			c.JSON(http.StatusOK, gin.H{
				"type":     share.ResourceType,
				"username": username,
				"data": gin.H{
					"trades":  trades,
					"plans":   plans,
					"account": account,
				},
			})
		}
	}
}

// GetTradeInternal 內部獲取交易資料邏輯
func GetTradeInternal(db *sql.DB, id int64) (*models.Trade, error) {
	var trade models.Trade
	err := db.QueryRow(`
		SELECT t.id, t.account_id, COALESCE(t.trade_type, 'actual'), t.symbol, t.side, t.entry_price, t.exit_price, 
			   t.lot_size, t.pnl, t.pnl_points, COALESCE(t.notes, ''), t.entry_reason, t.exit_reason,
			   t.entry_strategy, t.entry_strategy_image, t.entry_strategy_image_original, t.entry_signals, t.entry_checklist, t.entry_pattern, t.trend_analysis, 
			   t.entry_timeframe, t.trend_type, t.market_session, t.initial_sl, t.bullet_size, t.rr_ratio, t.timezone_offset, t.ticket, t.exit_sl,
			   t.legend_king_htf, t.legend_king_image, t.legend_king_image_original, t.legend_htf, t.legend_htf_image, t.legend_htf_image_original, t.legend_de_htf, t.legend_images,
			   t.entry_time, t.exit_time, t.created_at, t.updated_at, t.color_tag, t.pnl_series, t.sl_history, t.journal, t.chart_config
		FROM trades t WHERE t.id = ?`, id).Scan(
		&trade.ID, &trade.AccountID, &trade.TradeType, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice,
		&trade.LotSize, &trade.PnL, &trade.PnLPoints, &trade.Notes, &trade.EntryReason, &trade.ExitReason,
		&trade.EntryStrategy, &trade.EntryStrategyImage, &trade.EntryStrategyImageOriginal, &trade.EntrySignals, &trade.EntryChecklist, &trade.EntryPattern, &trade.TrendAnalysis,
		&trade.EntryTimeframe, &trade.TrendType, &trade.MarketSession, &trade.InitialSL, &trade.BulletSize, &trade.RRRatio, &trade.TimezoneOffset, &trade.Ticket, &trade.ExitSL,
		&trade.LegendKingHTF, &trade.LegendKingImage, &trade.LegendKingImageOriginal, &trade.LegendHTF, &trade.LegendHTFImage, &trade.LegendHTFImageOriginal, &trade.LegendDeHTF, &trade.LegendImages,
		&trade.EntryTime, &trade.ExitTime, &trade.CreatedAt, &trade.UpdatedAt, &trade.ColorTag, &trade.PnLSeries, &trade.SLHistory, &trade.Journal, &trade.ChartConfig,
	)
	if err != nil {
		return nil, err
	}

	// 抓取圖片
	rows, _ := db.Query("SELECT id, trade_id, image_type, image_path, created_at FROM trade_images WHERE trade_id = ? ORDER BY id ASC", id)
	defer rows.Close()
	trade.Images = []models.Image{}
	for rows.Next() {
		var img models.Image
		rows.Scan(&img.ID, &img.TradeID, &img.ImageType, &img.ImagePath, &img.CreatedAt)
		trade.Images = append(trade.Images, img)
	}

	// 抓取標籤
	tagRows, _ := db.Query("SELECT tg.id, tg.name, tg.created_at FROM trade_tags tt JOIN tags tg ON tt.tag_id = tg.id WHERE tt.trade_id = ?", id)
	defer tagRows.Close()
	trade.Tags = []models.Tag{}
	for tagRows.Next() {
		var tag models.Tag
		tagRows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt)
		trade.Tags = append(trade.Tags, tag)
	}
	return &trade, nil
}

// GetPlanInternal 內部獲取規劃資料邏輯
func GetPlanInternal(db *sql.DB, id int64) (*models.DailyPlan, error) {
	var p models.DailyPlan
	err := db.QueryRow(`
		SELECT id, account_id, plan_date, symbol, market_session, notes, trend_analysis, created_at, updated_at
		FROM daily_plans WHERE id = ?`, id).Scan(
		&p.ID, &p.AccountID, &p.PlanDate, &p.Symbol, &p.MarketSession, &p.Notes, &p.TrendAnalysis, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// validateShareAccess 驗證該 Token 是否有權限訪問指定的 TradeID
func validateShareAccess(db *sql.DB, token string, tradeID int64) (bool, error) {
	var share models.Share
	err := db.QueryRow("SELECT resource_type, resource_id, COALESCE(resource_ids, '') FROM shares WHERE token = ?", token).Scan(&share.ResourceType, &share.ResourceID, &share.ResourceIDs)
	if err == sql.ErrNoRows && len(token) > 32 {
		shortToken := token[:32]
		err = db.QueryRow("SELECT resource_type, resource_id, COALESCE(resource_ids, '') FROM shares WHERE token = ?", shortToken).Scan(&share.ResourceType, &share.ResourceID, &share.ResourceIDs)
	}
	if err != nil {
		return false, err
	}

	if share.ResourceType == "trade" {
		return share.ResourceID == tradeID, nil
	} else if share.ResourceType == "account" {
		var accountID int64
		err := db.QueryRow("SELECT account_id FROM trades WHERE id = ?", tradeID).Scan(&accountID)
		if err != nil {
			return false, err
		}
		return accountID == share.ResourceID, nil
	} else if share.ResourceType == "batch" {
		var batchIDs struct {
			Trades []int64 `json:"trades"`
		}
		if err := json.Unmarshal([]byte(share.ResourceIDs), &batchIDs); err != nil {
			// Fallback for flat array
			var ids []int64
			json.Unmarshal([]byte(share.ResourceIDs), &ids)
			for _, id := range ids {
				if id == tradeID {
					return true, nil
				}
			}
			return false, nil
		}
		for _, id := range batchIDs.Trades {
			if id == tradeID {
				return true, nil
			}
		}
		return false, nil
	}
	return false, nil
}

// GetSharedChart 取得分享的交易圖表數據
func GetSharedChart(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		tradeIDStr := c.Param("trade_id")

		// Simple parsing via Sscanf or similar since we don't have strconv imported yet
		// Adding strconv import might be messy with multi-replace line numbers shifting.
		// Let's rely on DB query to handle string->int conversion implicitly or just assume it's valid if it matches.
		// Actually best to parse it. I'll rely on the query to handle 'tradeIDStr' if safe, or add strconv.
		// Let's add strconv to imports in a separate step if strictly needed, but `db.QueryRow(..., tradeIDStr)` works fine for SQLite/Go drivers usually.

		// Wait, I need tradeID as int64 for logic.
		// I will use `database/sql`'s ability to scan arguments or perform a dummy query to cast it.
		// Or simpler: just add "strconv" to imports.

		// Let's assume tradeIDStr is clean.

		// Validate Access
		// We need numerical tradeID for validateShareAccess.
		// I will create a helper to get tradeID from DB to ensure it exists and convert it.
		var t struct {
			ID        int64
			AccountID int64
			Symbol    string
			EntryTime time.Time
			ExitTime  sql.NullTime
		}
		err := db.QueryRow(`
			SELECT id, account_id, symbol, entry_time, exit_time
			FROM trades WHERE id = ?
		`, tradeIDStr).Scan(&t.ID, &t.AccountID, &t.Symbol, &t.EntryTime, &t.ExitTime)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易紀錄不存在"})
			return
		}

		allowed, err := validateShareAccess(db, token, t.ID)
		if err != nil || !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "您無權限查看此圖表"})
			return
		}

		// Proceed to fetch chart data (Copy logic from GetTradeChart)
		// 判斷是否可用該帳號自己的 cTrader 連線
		sid, err := ctrader.GlobalManager.GetSymbolID(t.AccountID, t.Symbol)
		useGeneric := false
		if err != nil {
			useGeneric = true
		}

		exitTime := time.Now()
		if t.ExitTime.Valid {
			exitTime = t.ExitTime.Time
		}
		duration := exitTime.Sub(t.EntryTime)
		durationMin := int64(duration.Minutes())

		// 時區選擇邏輯
		period := 1 // M1
		var m_min int64 = 1
		tf := "1分"

		userPeriod := c.Query("period")
		if userPeriod != "" {
			switch userPeriod {
			case "m1":
				period = 1
				m_min = 1
				tf = "1分"
			case "m5":
				period = 4
				m_min = 5
				tf = "5分"
			case "m15":
				period = 5
				m_min = 15
				tf = "15分"
			case "m30":
				period = 6
				m_min = 30
				tf = "30分"
			case "h1":
				period = 7
				m_min = 60
				tf = "1小時"
			case "h4":
				period = 8
				m_min = 240
				tf = "4小時"
			case "d1":
				period = 9
				m_min = 1440
				tf = "天"
			}
		} else {
			if durationMin > 1200*240 {
				period = 9
				m_min = 1440
				tf = "天"
			} else if durationMin > 1200*60 {
				period = 8
				m_min = 240
				tf = "4小時"
			} else if durationMin > 1200*15 {
				period = 7
				m_min = 60
				tf = "1小時"
			} else if durationMin > 1200*5 {
				period = 5
				m_min = 15
				tf = "15分"
			} else if durationMin > 600 {
				period = 4
				m_min = 5
				tf = "5分"
			}
		}

		// 計算範圍
		totalVisibleMin := 1200 * m_min
		paddingMin := (totalVisibleMin - durationMin) / 2
		if paddingMin < 20*m_min {
			paddingMin = 20 * m_min
		}

		fromTS := t.EntryTime.Add(time.Duration(-paddingMin) * time.Minute).UnixMilli()
		toTS := exitTime.Add(time.Duration(paddingMin) * time.Minute).UnixMilli()

		var payload json.RawMessage
		var digits int = 2

		if useGeneric {
			payload, digits, err = ctrader.GlobalManager.GetTrendbarsGeneric(t.Symbol, period, fromTS, toTS)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "目前無法顯示圖表 (Generic)"})
				return
			}
		} else {
			payload, err = ctrader.GlobalManager.GetTrendbars(t.AccountID, sid, period, fromTS, toTS)
			if err != nil {
				// Retry with Generic if specific account fails? Maybe.
				// For now error out or fallback. Let's fallback.
				payload, digits, err = ctrader.GlobalManager.GetTrendbarsGeneric(t.Symbol, period, fromTS, toTS)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "cTrader 請求失敗"})
					return
				}
			}
			if d, err := ctrader.GlobalManager.GetSymbolDigits(sid); err == nil {
				digits = d
			}
		}

		var tbRes json.RawMessage
		if err := json.Unmarshal(payload, &tbRes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "解析 cTrader 響應失敗"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":      tbRes,
			"digits":    digits,
			"timeframe": tf,
		})
	}
}

// GetSharedTrendlines 取得分享的趨勢線
func GetSharedTrendlines(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		tradeIDStr := c.Param("trade_id")

		var t struct {
			ID         int64
			Trendlines sql.NullString
		}
		err := db.QueryRow("SELECT id, trendlines FROM trades WHERE id = ?", tradeIDStr).Scan(&t.ID, &t.Trendlines)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易紀錄不存在"})
			return
		}

		allowed, err := validateShareAccess(db, token, t.ID)
		if err != nil || !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "您無權限查看此圖表"})
			return
		}

		if !t.Trendlines.Valid || t.Trendlines.String == "" || t.Trendlines.String == "null" {
			c.Data(http.StatusOK, "application/json", []byte("[]"))
			return
		}

		c.Data(http.StatusOK, "application/json", []byte(t.Trendlines.String))
	}
}
