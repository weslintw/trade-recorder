package handlers

import (
	"database/sql"
	"net/http"

	"trade-journal/internal/models"

	"github.com/gin-gonic/gin"
)

// GetTrades 取得交易清單
func GetTrades(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var query models.TradeQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 預設分頁
		if query.Page <= 0 {
			query.Page = 1
		}
		if query.PageSize <= 0 {
			query.PageSize = 20
		}

		offset := (query.Page - 1) * query.PageSize

		// 建立查詢
		sqlQuery := `
		SELECT DISTINCT t.id, t.account_id, COALESCE(t.trade_type, 'actual'), t.symbol, t.side, t.entry_price, t.exit_price, 
			   t.lot_size, t.pnl, t.pnl_points, COALESCE(t.notes, ''), t.entry_reason, t.exit_reason,
			   t.entry_strategy, t.entry_strategy_image, t.entry_strategy_image_original, t.entry_signals, t.entry_checklist, t.entry_pattern, t.trend_analysis, 
			   t.entry_timeframe, t.trend_type, t.market_session, t.initial_sl, t.bullet_size, t.rr_ratio, COALESCE(a.timezone_offset, t.timezone_offset, 8), t.ticket, t.exit_sl,
			   t.entry_time, t.exit_time, t.created_at, t.updated_at
		FROM trades t
		LEFT JOIN accounts a ON t.account_id = a.id
		LEFT JOIN trade_tags tt ON t.id = tt.trade_id
		LEFT JOIN tags tg ON tt.tag_id = tg.id
		WHERE 1=1
	`
		args := []interface{}{}

		if query.AccountID > 0 {
			sqlQuery += " AND t.account_id = ?"
			args = append(args, query.AccountID)
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
				&trade.LotSize, &trade.PnL, &trade.PnLPoints, &trade.Notes, &trade.EntryReason, &trade.ExitReason,
				&trade.EntryStrategy, &trade.EntryStrategyImage, &trade.EntryStrategyImageOriginal, &trade.EntrySignals, &trade.EntryChecklist, &trade.EntryPattern, &trade.TrendAnalysis,
				&trade.EntryTimeframe, &trade.TrendType, &trade.MarketSession, &trade.InitialSL, &trade.BulletSize, &trade.RRRatio, &trade.TimezoneOffset, &trade.Ticket, &trade.ExitSL,
				&trade.EntryTime, &trade.ExitTime, &trade.CreatedAt, &trade.UpdatedAt,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 載入關聯資料
			loadTradeRelations(db, &trade)
			trades = append(trades, trade)
		}

		// 計算總數
		countQuery := `SELECT COUNT(DISTINCT t.id) FROM trades t
			LEFT JOIN trade_tags tt ON t.id = tt.trade_id
			LEFT JOIN tags tg ON tt.tag_id = tg.id
			WHERE 1=1`

		countArgs := []interface{}{}
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

		var trade models.Trade
		err := db.QueryRow(`
			SELECT t.id, t.account_id, COALESCE(t.trade_type, 'actual'), t.symbol, t.side, t.entry_price, t.exit_price, t.lot_size, t.pnl, t.pnl_points,
				   COALESCE(t.notes, ''), t.entry_reason, t.exit_reason, t.entry_strategy, t.entry_strategy_image, t.entry_strategy_image_original, t.entry_signals, t.entry_checklist,
				   t.entry_pattern, t.trend_analysis, t.entry_timeframe, t.trend_type, t.market_session, t.initial_sl, t.bullet_size, t.rr_ratio, COALESCE(a.timezone_offset, t.timezone_offset, 8), t.ticket, t.exit_sl, t.entry_time, t.exit_time, t.created_at, t.updated_at
			FROM trades t
			LEFT JOIN accounts a ON t.account_id = a.id
			WHERE t.id = ?
		`, id).Scan(
			&trade.ID, &trade.AccountID, &trade.TradeType, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice,
			&trade.LotSize, &trade.PnL, &trade.PnLPoints, &trade.Notes, &trade.EntryReason, &trade.ExitReason,
			&trade.EntryStrategy, &trade.EntryStrategyImage, &trade.EntryStrategyImageOriginal, &trade.EntrySignals, &trade.EntryChecklist, &trade.EntryPattern, &trade.TrendAnalysis,
			&trade.EntryTimeframe, &trade.TrendType, &trade.MarketSession, &trade.InitialSL, &trade.BulletSize, &trade.RRRatio, &trade.TimezoneOffset, &trade.Ticket, &trade.ExitSL,
			&trade.EntryTime, &trade.ExitTime, &trade.CreatedAt, &trade.UpdatedAt,
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
			INSERT INTO trades (account_id, trade_type, symbol, side, entry_price, exit_price, lot_size, pnl, pnl_points, notes, entry_reason, exit_reason, entry_strategy, entry_strategy_image, entry_strategy_image_original, entry_signals, entry_checklist, entry_pattern, trend_analysis, entry_timeframe, trend_type, market_session, initial_sl, bullet_size, rr_ratio, timezone_offset, entry_time, exit_time)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, req.AccountID, req.TradeType, req.Symbol, req.Side, req.EntryPrice, req.ExitPrice, req.LotSize, req.PnL, req.PnLPoints, req.Notes, req.EntryReason, req.ExitReason, req.EntryStrategy, req.EntryStrategyImage, req.EntryStrategyImageOriginal, req.EntrySignals, req.EntryChecklist, req.EntryPattern, req.TrendAnalysis, req.EntryTimeframe, req.TrendType, req.MarketSession, req.InitialSL, req.BulletSize, req.RRRatio, req.TimezoneOffset, req.EntryTime, req.ExitTime)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		tradeID, _ := result.LastInsertId()

		// 插入標籤
		for _, tagName := range req.Tags {
			var tagID int64
			err = tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
			if err == sql.ErrNoRows {
				result, _ := tx.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
				tagID, _ = result.LastInsertId()
			}
			tx.Exec("INSERT INTO trade_tags (trade_id, tag_id) VALUES (?, ?)", tradeID, tagID)
		}

		// 插入圖片
		for _, img := range req.Images {
			tx.Exec("INSERT INTO trade_images (trade_id, image_type, image_path) VALUES (?, ?, ?)",
				tradeID, img.ImageType, img.ImagePath)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": tradeID, "message": "交易紀錄建立成功"})
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
				   entry_pattern=?, trend_analysis=?, entry_timeframe=?, trend_type=?, market_session=?, initial_sl=?, bullet_size=?, rr_ratio=?, timezone_offset=?, entry_time=?, exit_time=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=?
		`, req.AccountID, req.TradeType, req.Symbol, req.Side, req.EntryPrice, req.ExitPrice, req.LotSize, req.PnL,
			req.PnLPoints, req.Notes, req.EntryReason, req.ExitReason, req.EntryStrategy, req.EntryStrategyImage, req.EntryStrategyImageOriginal, req.EntrySignals, req.EntryChecklist,
			req.EntryPattern, req.TrendAnalysis, req.EntryTimeframe, req.TrendType, req.MarketSession, req.InitialSL, req.BulletSize, req.RRRatio, req.TimezoneOffset, req.EntryTime, req.ExitTime, id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 更新標籤（先刪除再插入）
		tx.Exec("DELETE FROM trade_tags WHERE trade_id = ?", id)
		for _, tagName := range req.Tags {
			var tagID int64
			err = tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
			if err == sql.ErrNoRows {
				result, _ := tx.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
				tagID, _ = result.LastInsertId()
			}
			tx.Exec("INSERT INTO trade_tags (trade_id, tag_id) VALUES (?, ?)", id, tagID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "交易紀錄更新成功"})
	}
}

// DeleteTrade 刪除交易
func DeleteTrade(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		result, err := db.Exec("DELETE FROM trades WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易紀錄不存在"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "交易紀錄刪除成功"})
	}
}

// loadTradeRelations 載入交易的關聯資料（圖片和標籤）
func loadTradeRelations(db *sql.DB, trade *models.Trade) {
	// 載入圖片
	imgRows, _ := db.Query(`
		SELECT id, trade_id, image_type, image_path, created_at
		FROM trade_images WHERE trade_id = ?
	`, trade.ID)
	defer imgRows.Close()

	for imgRows.Next() {
		var img models.Image
		imgRows.Scan(&img.ID, &img.TradeID, &img.ImageType, &img.ImagePath, &img.CreatedAt)
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
		rows, err := db.Query("SELECT id, name, created_at FROM tags ORDER BY name")
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
