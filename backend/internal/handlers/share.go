package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"log"
	"net/http"
	"trade-journal/internal/models"

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
		}

		if ownerID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限分享此內容"})
			return
		}

		// 如果已經有公開分享，且這次也是要求公開，就回傳現有的
		var token string
		if req.ShareType == "public" {
			err := db.QueryRow("SELECT token FROM shares WHERE resource_type = ? AND resource_id = ? AND share_type = 'public'", req.ResourceType, req.ResourceID).Scan(&token)
			if err == nil {
				c.JSON(http.StatusOK, gin.H{"token": token, "message": "已取得現有分享連結"})
				return
			}
			token = GenerateToken()
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec("INSERT INTO shares (user_id, resource_type, resource_id, share_type, token) VALUES (?, ?, ?, ?, ?)",
			userID, req.ResourceType, req.ResourceID, req.ShareType, token)
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
		err := db.QueryRow("SELECT resource_type, resource_id FROM shares WHERE token = ?", token).Scan(&share.ResourceType, &share.ResourceID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("[Share] Token not found: %s", token)
			} else {
				log.Printf("[Share] Database error for token %s: %v", token, err)
			}
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到此分享連結或已失效"})
			return
		}

		if share.ResourceType == "trade" {
			trade, err := GetTradeInternal(db, share.ResourceID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到交易內容", "details": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"type": "trade", "data": trade})
		} else if share.ResourceType == "plan" {
			plan, err := GetPlanInternal(db, share.ResourceID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "找不到規劃內容", "details": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"type": "plan", "data": plan})
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
			   t.legend_king_htf, t.legend_king_image, t.legend_king_image_original, t.legend_htf, t.legend_htf_image, t.legend_htf_image_original, t.legend_de_htf,
			   t.entry_time, t.exit_time, t.created_at, t.updated_at
		FROM trades t WHERE t.id = ?`, id).Scan(
		&trade.ID, &trade.AccountID, &trade.TradeType, &trade.Symbol, &trade.Side, &trade.EntryPrice, &trade.ExitPrice,
		&trade.LotSize, &trade.PnL, &trade.PnLPoints, &trade.Notes, &trade.EntryReason, &trade.ExitReason,
		&trade.EntryStrategy, &trade.EntryStrategyImage, &trade.EntryStrategyImageOriginal, &trade.EntrySignals, &trade.EntryChecklist, &trade.EntryPattern, &trade.TrendAnalysis,
		&trade.EntryTimeframe, &trade.TrendType, &trade.MarketSession, &trade.InitialSL, &trade.BulletSize, &trade.RRRatio, &trade.TimezoneOffset, &trade.Ticket, &trade.ExitSL,
		&trade.LegendKingHTF, &trade.LegendKingImage, &trade.LegendKingImageOriginal, &trade.LegendHTF, &trade.LegendHTFImage, &trade.LegendHTFImageOriginal, &trade.LegendDeHTF,
		&trade.EntryTime, &trade.ExitTime, &trade.CreatedAt, &trade.UpdatedAt,
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
