package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"time"
	"trade-journal/internal/ctrader"
	"trade-journal/internal/database"
	"trade-journal/internal/models"
	"trade-journal/internal/ws"

	"github.com/gin-gonic/gin"
)

// GetTrades 取得交易清單
func GetTrades(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		userID := c.GetInt64("user_id")
		var query models.TradeQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("[GetTrades DEBUG] User:%d, Acc:%d, Symbol:%s, Page:%d", userID, query.AccountID, query.Symbol, query.Page)

		// 預設分頁
		if query.Page <= 0 {
			query.Page = 1
		}
		if query.PageSize <= 0 {
			query.PageSize = 20
		}

		offset := (query.Page - 1) * query.PageSize

		// 建立查詢 (優化版：已補回關鍵篩選欄位)
		sqlQuery := `
		SELECT DISTINCT t.id, t.account_id, COALESCE(t.trade_type, 'actual'), t.symbol, t.side, t.entry_price, t.exit_price, 
			   t.lot_size, t.pnl, t.pnl_points, t.entry_strategy, t.entry_timeframe, t.market_session, t.initial_sl, t.bullet_size, t.rr_ratio, 
			   COALESCE(a.timezone_offset, t.timezone_offset, 8), t.ticket, t.exit_sl, t.entry_signals, t.entry_checklist, t.entry_pattern,
			   t.entry_time, t.color_tag, t.exit_time, t.created_at, t.updated_at, t.sl_history, t.pnl_series, t.journal
		FROM trades t
		LEFT JOIN accounts a ON t.account_id = a.id
		LEFT JOIN trade_tags tt ON t.id = tt.trade_id
		LEFT JOIN tags tg ON tt.tag_id = tg.id
		WHERE a.user_id = ?
	`
		args := []interface{}{userID}

		if query.AccountID > 0 {
			sqlQuery += " AND t.account_id = ?"
			args = append(args, query.AccountID)
		} else {
			// 如果沒有提供帳號 ID，目前邏輯不返回任何交易以避免混合不同帳號資料
			c.JSON(http.StatusOK, []models.Trade{})
			return
		}

		if query.Symbol != "" {
			sqlQuery += " AND t.symbol = ?"
			args = append(args, query.Symbol)
		}

		if query.Side != "" {
			sqlQuery += " AND t.side = ?"
			args = append(args, query.Side)
		}

		if query.Tag != "" {
			sqlQuery += " AND tg.name = ?"
			args = append(args, query.Tag)
		}

		if query.StartDate != "" {
			sqlQuery += " AND t.entry_time >= ?"
			args = append(args, query.StartDate)
		}

		if query.EndDate != "" {
			sqlQuery += " AND t.entry_time <= ?"
			args = append(args, query.EndDate)
		}

		if query.Strategy != "" && query.Strategy != "all" {
			// 前端可能傳 "expert" (達人), "elite" (菁英), "legend" (傳奇)
			// 資料庫存的是小寫英文，所以直接匹配即可
			sqlQuery += " AND t.entry_strategy = ?"
			args = append(args, query.Strategy)
		}

		if query.Keyword != "" {
			// 關鍵字搜尋：搜尋訊號、樣態、清單以及筆記
			// 使用 LIKE %keyword% 進行模糊搜尋
			searchPattern := "%" + query.Keyword + "%"
			sqlQuery += " AND (t.entry_signals LIKE ? OR t.entry_pattern LIKE ? OR t.entry_checklist LIKE ? OR t.notes LIKE ?)"
			args = append(args, searchPattern, searchPattern, searchPattern, searchPattern)
		}

		sqlQuery += " ORDER BY t.entry_time DESC LIMIT ? OFFSET ?"
		args = append(args, query.PageSize, offset)

		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		trades := []models.Trade{}
		for rows.Next() {
			var trade models.Trade
			err := rows.Scan(
				&trade.ID, &trade.AccountID, &trade.TradeType, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice,
				&trade.LotSize, &trade.PnL, &trade.PnLPoints, &trade.EntryStrategy, &trade.EntryTimeframe, &trade.MarketSession, &trade.InitialSL, &trade.BulletSize, &trade.RRRatio,
				&trade.TimezoneOffset, &trade.Ticket, &trade.ExitSL, &trade.EntrySignals, &trade.EntryChecklist, &trade.EntryPattern,
				&trade.EntryTime, &trade.ColorTag, &trade.ExitTime, &trade.CreatedAt, &trade.UpdatedAt, &trade.SLHistory, &trade.PnLSeries, &trade.Journal,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			trades = append(trades, trade)
		}

		// Collect IDs for bulk loading relations
		tradeIDs := make([]int64, 0, len(trades))
		tradeMap := make(map[int64]*models.Trade)
		for i := range trades {
			tradeIDs = append(tradeIDs, trades[i].ID)
			tradeMap[trades[i].ID] = &trades[i]
		}

		if len(tradeIDs) > 0 {
			// Bulk load images
			placeholders := make([]string, len(tradeIDs))
			args := make([]interface{}, len(tradeIDs))
			for i, id := range tradeIDs {
				placeholders[i] = "?"
				args[i] = id
			}

			imgQuery := `
				SELECT id, trade_id, image_type, image_path, file_size, created_at
				FROM trade_images 
				WHERE trade_id IN (%s)
			`
			imgQuery = fmt.Sprintf(imgQuery, strings.Join(placeholders, ","))

			imgRows, err := db.Query(imgQuery, args...)
			if err == nil {
				defer imgRows.Close()
				for imgRows.Next() {
					var img models.Image
					if err := imgRows.Scan(&img.ID, &img.TradeID, &img.ImageType, &img.ImagePath, &img.FileSize, &img.CreatedAt); err == nil {
						if t, ok := tradeMap[img.TradeID]; ok {
							t.Images = append(t.Images, img)
						}
					}
				}
			}

			// Bulk load tags
			tagQuery := fmt.Sprintf(`
				SELECT tt.trade_id, t.id, t.name, t.created_at
				FROM tags t
				JOIN trade_tags tt ON t.id = tt.tag_id
				WHERE tt.trade_id IN (%s)
			`, strings.Join(placeholders, ","))

			tagRows, err := db.Query(tagQuery, args...)
			if err == nil {
				defer tagRows.Close()
				for tagRows.Next() {
					var tradeID int64
					var tag models.Tag
					if err := tagRows.Scan(&tradeID, &tag.ID, &tag.Name, &tag.CreatedAt); err == nil {
						if t, ok := tradeMap[tradeID]; ok {
							t.Tags = append(t.Tags, tag)
						}
					}
				}
			}
		}

		// 計算總數
		countQuery := `SELECT COUNT(DISTINCT t.id) FROM trades t
			LEFT JOIN accounts a ON t.account_id = a.id
			LEFT JOIN trade_tags tt ON t.id = tt.trade_id
			LEFT JOIN tags tg ON tt.tag_id = tg.id
			WHERE a.user_id = ?`

		countArgs := []interface{}{userID}
		if query.AccountID > 0 {
			countQuery += " AND t.account_id = ?"
			countArgs = append(countArgs, query.AccountID)
		}
		if query.Symbol != "" {
			countQuery += " AND t.symbol = ?"
			countArgs = append(countArgs, query.Symbol)
		}
		if query.Side != "" {
			countQuery += " AND t.side = ?"
			countArgs = append(countArgs, query.Side)
		}
		if query.Tag != "" {
			countQuery += " AND tg.name = ?"
			countArgs = append(countArgs, query.Tag)
		}

		var total int
		db.QueryRow(countQuery, countArgs...).Scan(&total)

		// Estimate size
		jsonData, _ := json.Marshal(trades)
		sizeKB := float64(len(jsonData)) / 1024

		log.Printf("[GetTrades PERF] Total duration: %v, items: %d, total: %d, size: %.2f KB", time.Since(startTime), len(trades), total, sizeKB)

		c.JSON(http.StatusOK, gin.H{
			"data": trades,
			"pagination": gin.H{
				"page":      query.Page,
				"page_size": query.PageSize,
				"total":     total,
			},
		})
	}
}

// GetTrade 取得單筆交易
func GetTrade(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		var trade models.Trade
		err := db.QueryRow(`
			SELECT t.id, t.account_id, COALESCE(t.trade_type, 'actual'), t.symbol, t.side, t.entry_price, t.exit_price, t.lot_size, t.pnl, t.pnl_points,
				   COALESCE(t.notes, ''), t.entry_reason, t.exit_reason, t.entry_strategy, t.entry_strategy_image, t.entry_strategy_image_original, t.entry_signals, t.entry_checklist,
				   t.entry_pattern, t.trend_analysis, t.entry_timeframe, t.trend_type, t.market_session, t.initial_sl, t.bullet_size, t.rr_ratio, COALESCE(a.timezone_offset, t.timezone_offset, 8), t.ticket, t.exit_sl,
				   t.legend_king_htf, t.legend_king_image, t.legend_king_image_original, t.legend_htf, t.legend_htf_image, t.legend_htf_image_original, t.legend_de_htf, t.legend_images, t.expert_images, t.elite_images,
				   t.entry_time, t.color_tag, t.exit_time, t.created_at, t.updated_at, t.sl_history, t.pnl_series, t.journal
			FROM trades t
			LEFT JOIN accounts a ON t.account_id = a.id
			WHERE t.id = ? AND a.user_id = ?
		`, id, userID).Scan(
			&trade.ID, &trade.AccountID, &trade.TradeType, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice,
			&trade.LotSize, &trade.PnL, &trade.PnLPoints, &trade.Notes, &trade.EntryReason, &trade.ExitReason,
			&trade.EntryStrategy, &trade.EntryStrategyImage, &trade.EntryStrategyImageOriginal, &trade.EntrySignals, &trade.EntryChecklist, &trade.EntryPattern, &trade.TrendAnalysis,
			&trade.EntryTimeframe, &trade.TrendType, &trade.MarketSession, &trade.InitialSL, &trade.BulletSize, &trade.RRRatio, &trade.TimezoneOffset, &trade.Ticket, &trade.ExitSL,
			&trade.LegendKingHTF, &trade.LegendKingImage, &trade.LegendKingImageOriginal, &trade.LegendHTF, &trade.LegendHTFImage, &trade.LegendHTFImageOriginal, &trade.LegendDeHTF, &trade.LegendImages, &trade.ExpertImages, &trade.EliteImages,
			&trade.EntryTime, &trade.ColorTag, &trade.ExitTime, &trade.CreatedAt, &trade.UpdatedAt, &trade.SLHistory, &trade.PnLSeries, &trade.Journal,
		)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易紀錄不存在"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		loadTradeRelations(db, &trade)
		c.JSON(http.StatusOK, trade)
	}
}

// CreateTrade 建立交易
func CreateTrade(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.TradeCreate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt64("user_id")

		// 檢查帳號所屬權
		var exists int
		db.QueryRow("SELECT 1 FROM accounts WHERE id = ? AND user_id = ?", req.AccountID, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限操作此帳號"})
			return
		}

		// 如果是觀察記錄，且價格為空，給予預設值 0 避免資料庫 NOT NULL 限制
		if req.TradeType == "observation" {
			if req.EntryPrice == nil {
				zero := 0.0
				req.EntryPrice = &zero
			}
			if req.LotSize == nil {
				zero := 0.0
				req.LotSize = &zero
			}
		}

		// 開始交易
		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		// 插入交易紀錄
		result, err := tx.Exec(`
			INSERT INTO trades (account_id, trade_type, symbol, side, entry_price, exit_price, lot_size, pnl, pnl_points, notes, entry_reason, exit_reason, entry_strategy, entry_strategy_image, entry_strategy_image_original, entry_signals, entry_checklist, entry_pattern, trend_analysis, entry_timeframe, trend_type, market_session, initial_sl, bullet_size, rr_ratio, timezone_offset, exit_sl, legend_king_htf, legend_king_image, legend_king_image_original, legend_htf, legend_htf_image, legend_htf_image_original, legend_de_htf, legend_images, expert_images, elite_images, entry_time, color_tag, exit_time, journal)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, req.AccountID, req.TradeType, req.Symbol, req.Side, req.EntryPrice, req.ExitPrice, req.LotSize, req.PnL, req.PnLPoints, req.Notes, req.EntryReason, req.ExitReason, req.EntryStrategy, req.EntryStrategyImage, req.EntryStrategyImageOriginal, req.EntrySignals, req.EntryChecklist, req.EntryPattern, req.TrendAnalysis, req.EntryTimeframe, req.TrendType, req.MarketSession, req.InitialSL, req.BulletSize, req.RRRatio, req.TimezoneOffset, req.ExitSL, req.LegendKingHTF, req.LegendKingImage, req.LegendKingImageOriginal, req.LegendHTF, req.LegendHTFImage, req.LegendHTFImageOriginal, req.LegendDeHTF, req.LegendImages, req.ExpertImages, req.EliteImages, req.EntryTime, req.ColorTag, req.ExitTime, req.Journal)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tradeID, _ := result.LastInsertId()

		// 插入標籤 (現在標籤是使用者專屬的)
		for _, tagName := range req.Tags {
			var tagID int64
			err = tx.QueryRow("SELECT id FROM tags WHERE name = ? AND user_id = ?", tagName, userID).Scan(&tagID)
			if err == sql.ErrNoRows {
				result, _ := tx.Exec("INSERT INTO tags (name, user_id) VALUES (?, ?)", tagName, userID)
				tagID, _ = result.LastInsertId()
			}
			tx.Exec("INSERT INTO trade_tags (trade_id, tag_id) VALUES (?, ?)", tradeID, tagID)
		}

		// 插入圖片
		for _, img := range req.Images {
			tx.Exec("INSERT INTO trade_images (trade_id, image_type, image_path, file_size) VALUES (?, ?, ?, ?)",
				tradeID, img.ImageType, img.ImagePath, img.FileSize)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": tradeID, "message": "交易紀錄建立成功"})
		ws.GlobalHub.BroadcastUpdate(req.AccountID, "TRADE_UPDATE")
		go database.UpdateAccountStorageUsage(db, req.AccountID)
	}
}

// UpdateTrade 更新交易
func UpdateTrade(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.TradeCreate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// fmt.Printf("[UpdateTrade DEBUG] ID: %s, EntrySignals Payload: %s\n", id, req.EntrySignals)

		userID := c.GetInt64("user_id")

		// 檢查交易所屬權 (透過 join accounts)
		var exists int
		db.QueryRow("SELECT 1 FROM trades t JOIN accounts a ON t.account_id = a.id WHERE t.id = ? AND a.user_id = ?", id, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限更新此交易"})
			return
		}

		// 檢查目標帳號所屬權
		db.QueryRow("SELECT 1 FROM accounts WHERE id = ? AND user_id = ?", req.AccountID, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限將交易移動到此帳號"})
			return
		}

		// 如果是觀察記錄，且價格為空，給予預設值 0 避免資料庫 NOT NULL 限制
		if req.TradeType == "observation" {
			if req.EntryPrice == nil {
				zero := 0.0
				req.EntryPrice = &zero
			}
			if req.LotSize == nil {
				zero := 0.0
				req.LotSize = &zero
			}
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			UPDATE trades SET account_id=?, trade_type=?, symbol=?, side=?, entry_price=?, exit_price=?, lot_size=?, 
				   pnl=?, pnl_points=?, notes=?, entry_reason=?, exit_reason=?, entry_strategy=?, entry_strategy_image=?, entry_strategy_image_original=?, entry_signals=?, entry_checklist=?,
				   entry_pattern=?, trend_analysis=?, entry_timeframe=?, trend_type=?, market_session=?, initial_sl=?, bullet_size=?, rr_ratio=?, timezone_offset=?, exit_sl=?,
				   legend_king_htf=?, legend_king_image=?, legend_king_image_original=?, legend_htf=?, legend_htf_image=?, legend_htf_image_original=?, legend_de_htf=?, legend_images=?, expert_images=?, elite_images=?,
				   entry_time=?, color_tag=?, exit_time=?, journal=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=?
		`, req.AccountID, req.TradeType, req.Symbol, req.Side, req.EntryPrice, req.ExitPrice, req.LotSize, req.PnL,
			req.PnLPoints, req.Notes, req.EntryReason, req.ExitReason, req.EntryStrategy, req.EntryStrategyImage, req.EntryStrategyImageOriginal, req.EntrySignals, req.EntryChecklist,
			req.EntryPattern, req.TrendAnalysis, req.EntryTimeframe, req.TrendType, req.MarketSession, req.InitialSL, req.BulletSize, req.RRRatio, req.TimezoneOffset, req.ExitSL,
			req.LegendKingHTF, req.LegendKingImage, req.LegendKingImageOriginal, req.LegendHTF, req.LegendHTFImage, req.LegendHTFImageOriginal, req.LegendDeHTF, req.LegendImages, req.ExpertImages, req.EliteImages,
			req.EntryTime, req.ColorTag, req.ExitTime, req.Journal, id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 更新標籤（先刪除再插入）
		tx.Exec("DELETE FROM trade_tags WHERE trade_id = ?", id)
		for _, tagName := range req.Tags {
			var tagID int64
			err = tx.QueryRow("SELECT id FROM tags WHERE name = ? AND user_id = ?", tagName, userID).Scan(&tagID)
			if err == sql.ErrNoRows {
				result, _ := tx.Exec("INSERT INTO tags (name, user_id) VALUES (?, ?)", tagName, userID)
				tagID, _ = result.LastInsertId()
			}
			tx.Exec("INSERT INTO trade_tags (trade_id, tag_id) VALUES (?, ?)", id, tagID)
		}

		// 更新圖片（先刪除再插入）
		tx.Exec("DELETE FROM trade_images WHERE trade_id = ?", id)
		for _, img := range req.Images {
			tx.Exec("INSERT INTO trade_images (trade_id, image_type, image_path, file_size) VALUES (?, ?, ?, ?)",
				id, img.ImageType, img.ImagePath, img.FileSize)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "交易紀錄更新成功"})
		ws.GlobalHub.BroadcastUpdate(req.AccountID, "TRADE_UPDATE")
		go database.UpdateAccountStorageUsage(db, req.AccountID)
	}
}

// DeleteTrade 刪除交易
func DeleteTrade(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		// 獲取 Ticket 與 AccountID
		var ticket sql.NullString
		var accountID int64
		err := db.QueryRow(`
			SELECT t.ticket, t.account_id 
			FROM trades t 
			JOIN accounts a ON t.account_id = a.id 
			WHERE t.id = ? AND a.user_id = ?
		`, id, userID).Scan(&ticket, &accountID)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易紀錄不存在"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		// 如果是已同步交易（有 Ticket），紀錄為已刪除
		if ticket.Valid && ticket.String != "" {
			_, err = tx.Exec("INSERT OR IGNORE INTO deleted_tickets (account_id, ticket) VALUES (?, ?)", accountID, ticket.String)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "紀錄已刪除票據失敗"})
				return
			}
		}

		_, err = tx.Exec("DELETE FROM trades WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "交易紀錄刪除成功"})
		ws.GlobalHub.BroadcastUpdate(accountID, "TRADE_UPDATE")
		go database.UpdateAccountStorageUsage(db, accountID)
	}
}

// SyncSingleTrade 重新整理單筆交易資料
func SyncSingleTrade(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		var trade struct {
			ID          int64
			AccountID   int64
			Ticket      sql.NullString
			Symbol      string
			Side        string
			EntryPrice  float64
			LotSize     float64
			EntryTime   time.Time
			ExitTime    sql.NullTime
			AccountType string
		}

		err := db.QueryRow(`
			SELECT t.id, t.account_id, t.ticket, t.symbol, t.side, t.entry_price, t.lot_size, t.entry_time, t.exit_time, a.type
			FROM trades t
			JOIN accounts a ON t.account_id = a.id
			WHERE t.id = ? AND a.user_id = ?
		`, id, userID).Scan(&trade.ID, &trade.AccountID, &trade.Ticket, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.LotSize, &trade.EntryTime, &trade.ExitTime, &trade.AccountType)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易紀錄不存在"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if trade.AccountType == "ctrader" && trade.Ticket.Valid && trade.Ticket.String != "" {
			if ctrader.GlobalManager != nil {
				go ctrader.GlobalManager.ManualSyncTrade(trade.AccountID, trade.Ticket.String)
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "同步請求已送出，資料將在幾秒內更新"})
	}
}

// loadTradeRelations 載入交易的關聯資料（圖片和標籤）
func loadTradeRelations(db *sql.DB, trade *models.Trade) {
	// 載入圖片
	imgRows, _ := db.Query(`
		SELECT id, trade_id, image_type, image_path, file_size, created_at
		FROM trade_images WHERE trade_id = ?
	`, trade.ID)
	defer imgRows.Close()

	for imgRows.Next() {
		var img models.Image
		imgRows.Scan(&img.ID, &img.TradeID, &img.ImageType, &img.ImagePath, &img.FileSize, &img.CreatedAt)
		trade.Images = append(trade.Images, img)
	}

	// 載入標籤
	tagRows, _ := db.Query(`
		SELECT t.id, t.name, t.created_at
		FROM tags t
		INNER JOIN trade_tags tt ON t.id = tt.tag_id
		WHERE tt.trade_id = ?
	`, trade.ID)
	defer tagRows.Close()

	for tagRows.Next() {
		var tag models.Tag
		tagRows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt)
		trade.Tags = append(trade.Tags, tag)
	}
}

// GetTags 取得所有標籤
func GetTags(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		rows, err := db.Query("SELECT id, name, created_at FROM tags WHERE user_id = ? ORDER BY name", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		tags := []models.Tag{}
		for rows.Next() {
			var tag models.Tag
			rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt)
			tags = append(tags, tag)
		}

		c.JSON(http.StatusOK, tags)
	}
}
