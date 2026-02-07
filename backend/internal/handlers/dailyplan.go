package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"strconv"
	"trade-journal/internal/ctrader"
	"trade-journal/internal/database"
	"trade-journal/internal/models"
	"trade-journal/internal/ws"

	"github.com/gin-gonic/gin"
)

// GetDailyPlans 取得每日規劃清單
func GetDailyPlans(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		userID := c.GetInt64("user_id")
		var query models.DailyPlanQuery
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
		if query.PageSize > 10000 {
			query.PageSize = 10000
		}

		offset := (query.Page - 1) * query.PageSize

		// 建立查詢 (優化版：移除 JOIN，直接利用複合索引)
		sqlQuery := `
			SELECT p.id, p.account_id, p.plan_date, p.symbol, p.market_session, COALESCE(p.notes, ''), COALESCE(p.trend_analysis, '{}'), p.created_at, p.updated_at
			FROM daily_plans p
			JOIN accounts a ON p.account_id = a.id
			WHERE p.account_id = ? AND a.user_id = ?
		`

		args := []interface{}{query.AccountID, userID}
		if query.AccountID <= 0 {
			c.JSON(http.StatusOK, gin.H{
				"data":      []models.DailyPlan{},
				"total":     0,
				"page":      query.Page,
				"page_size": query.PageSize,
			})
			return
		}

		if query.StartDate != "" {
			sqlQuery += " AND plan_date >= ?"
			args = append(args, query.StartDate)
		}

		if query.EndDate != "" {
			sqlQuery += " AND plan_date <= ?"
			args = append(args, query.EndDate)
		}

		if query.MarketSession != "" {
			sqlQuery += " AND (market_session = ? OR market_session = 'all')"
			args = append(args, query.MarketSession)
		}

		if query.Symbol != "" {
			sqlQuery += " AND symbol = ?"
			args = append(args, query.Symbol)
		}

		sqlQuery += " ORDER BY plan_date DESC LIMIT ? OFFSET ?"
		args = append(args, query.PageSize, offset)

		rows, err := db.Query(sqlQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		plans := []models.DailyPlan{}
		for rows.Next() {
			var plan models.DailyPlan
			var rawNotes, rawTrend string
			err := rows.Scan(
				&plan.ID, &plan.AccountID, &plan.PlanDate, &plan.Symbol, &plan.MarketSession, &rawNotes,
				&rawTrend, &plan.CreatedAt, &plan.UpdatedAt,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// 限制傳輸體積：如果單筆內容超過 500KB，在列表視圖中進行截斷，防止前端崩潰
			// 使用 rune 切片確保 UTF-8 安全，不分段截斷多位元組字元
			safeTruncate := func(s string, limit int) string {
				if len(s) <= limit {
					return s
				}
				// 先按字節截斷到大概位置
				sub := s[:limit]
				// 找到最後一個完整的 UTF-8 字元邊界
				for i := len(sub); i > 0; i-- {
					if (sub[i-1] & 0xc0) != 0x80 { // 不是從屬位元組
						return sub[:i] + "... (Truncated)"
					}
				}
				return "... (Truncated)"
			}

			plan.Notes = safeTruncate(rawNotes, 500000)

			// TrendAnalysis 若過大直接回傳空物件，因為列表視圖通常不顯示它
			if len(rawTrend) > 500000 {
				plan.TrendAnalysis = "{}"
			} else {
				plan.TrendAnalysis = rawTrend
			}

			plans = append(plans, plan)
		}

		// 計算總數
		var total int
		countQuery := `SELECT COUNT(*) FROM daily_plans WHERE account_id = ?`
		countArgs := []interface{}{query.AccountID}

		if query.StartDate != "" {
			countQuery += " AND plan_date >= ?"
			countArgs = append(countArgs, query.StartDate)
		}
		if query.EndDate != "" {
			countQuery += " AND plan_date <= ?"
			countArgs = append(countArgs, query.EndDate)
		}
		if query.MarketSession != "" {
			countQuery += " AND (market_session = ? OR market_session = 'all')"
			countArgs = append(countArgs, query.MarketSession)
		}
		if query.Symbol != "" {
			countQuery += " AND symbol = ?"
			countArgs = append(countArgs, query.Symbol)
		}

		db.QueryRow(countQuery, countArgs...).Scan(&total)

		// Estimate size
		jsonData, _ := json.Marshal(plans)
		sizeKB := float64(len(jsonData)) / 1024

		duration := time.Since(startTime)
		log.Printf("[GetDailyPlans PERF] Total duration: %v, items: %d, total: %d, size: %.2f KB", duration, len(plans), total, sizeKB)

		// 如果真的很慢，記錄慢查詢參數
		if duration > 1*time.Second {
			log.Printf("[GetDailyPlans SLOW] AccountID: %d, Symbol: %s, duration: %v", query.AccountID, query.Symbol, duration)
		}

		c.JSON(http.StatusOK, gin.H{
			"data":      plans,
			"total":     total,
			"page":      query.Page,
			"page_size": query.PageSize,
		})
	}
}

// GetDailyPlan 取得單一每日規劃
func GetDailyPlan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		var plan models.DailyPlan
		var rawNotes, rawTrend string
		err := db.QueryRow(`
			SELECT p.id, p.account_id, p.plan_date, p.symbol, p.market_session, COALESCE(p.notes, ''), COALESCE(p.trend_analysis, '{}'), p.created_at, p.updated_at
			FROM daily_plans p
			JOIN accounts a ON p.account_id = a.id
			WHERE p.id = ? AND a.user_id = ?
		`, id, userID).Scan(
			&plan.ID, &plan.AccountID, &plan.PlanDate, &plan.Symbol, &plan.MarketSession, &rawNotes,
			&rawTrend, &plan.CreatedAt, &plan.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "規劃不存在"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 單筆詳情也進行截斷保護，如果真的超過 1MB 則強行報警並截斷
		if len(rawNotes) > 1000000 {
			plan.Notes = rawNotes[:1000000] + "... (Content exceeds 1MB, truncated to prevent crash)"
		} else {
			plan.Notes = rawNotes
		}

		if len(rawTrend) > 1000000 {
			plan.TrendAnalysis = "{}" // 過大則重設
		} else {
			plan.TrendAnalysis = rawTrend
		}

		c.JSON(http.StatusOK, plan)
	}
}

// CreateDailyPlan 建立每日規劃
func CreateDailyPlan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.DailyPlanCreate
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

		// 檢查是否已存在同日期、同品種的規劃
		var existsID int64
		// 使用 date() 函數確保只比較日期部分
		err := db.QueryRow(`
			SELECT id FROM daily_plans 
			WHERE date(plan_date) = date(?) AND symbol = ? AND account_id = ?
		`, req.PlanDate, req.Symbol, req.AccountID).Scan(&existsID)

		if err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "該日期與品種的規劃已存在，請直接編輯原有的規劃"})
			return
		}

		if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "資料庫查詢錯誤: " + err.Error()})
			return
		}

		result, err := db.Exec(`
			INSERT INTO daily_plans (account_id, plan_date, symbol, market_session, notes, trend_analysis)
			VALUES (?, ?, ?, ?, ?, ?)
		`, req.AccountID, req.PlanDate, req.Symbol, req.MarketSession, req.Notes, req.TrendAnalysis)

		if err != nil {
			// 檢查是否為唯一索引衝突 (SQLite 錯誤碼 2067 或檢查錯誤字串)
			errStr := err.Error()
			if errStr != "" && (errStr[0:10] == "UNIQUE con" || errStr[0:15] == "constraint fail") {
				c.JSON(http.StatusConflict, gin.H{"error": "該日期與品種的規劃已存在，請直接編輯原有的規劃"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "儲存失敗: " + err.Error()})
			return
		}

		id, _ := result.LastInsertId()
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "規劃建立成功"})
		ws.GlobalHub.BroadcastUpdate(req.AccountID, "TRADE_UPDATE")
	}
}

// UpdateDailyPlan 更新每日規劃
func UpdateDailyPlan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.DailyPlanCreate
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		userID := c.GetInt64("user_id")

		// 檢查規劃所屬權
		var exists int
		db.QueryRow("SELECT 1 FROM daily_plans p JOIN accounts a ON p.account_id = a.id WHERE p.id = ? AND a.user_id = ?", id, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限更新此規劃"})
			return
		}

		// 檢查目標帳號所屬權
		db.QueryRow("SELECT 1 FROM accounts WHERE id = ? AND user_id = ?", req.AccountID, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限將規劃移動到此帳號"})
			return
		}

		_, err := db.Exec(`
			UPDATE daily_plans 
			SET account_id=?, plan_date=?, symbol=?, market_session=?, notes=?, trend_analysis=?, updated_at=CURRENT_TIMESTAMP
			WHERE id=?
		`, req.AccountID, req.PlanDate, req.Symbol, req.MarketSession, req.Notes, req.TrendAnalysis, id)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "規劃更新成功"})
		ws.GlobalHub.BroadcastUpdate(req.AccountID, "TRADE_UPDATE")
		go database.UpdateAccountStorageUsage(db, req.AccountID)
	}
}

// DeleteDailyPlan 刪除每日規劃
func DeleteDailyPlan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetInt64("user_id")

		var accountID int64
		_ = db.QueryRow("SELECT account_id FROM daily_plans WHERE id = ?", id).Scan(&accountID)

		res, err := db.Exec("DELETE FROM daily_plans WHERE id = ? AND id IN (SELECT p.id FROM daily_plans p JOIN accounts a ON p.account_id = a.id WHERE a.user_id = ?)", id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "規劃不存在或無權限刪除"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "規劃刪除成功"})
		if accountID > 0 {
			ws.GlobalHub.BroadcastUpdate(accountID, "TRADE_UPDATE")
		} else {
			ws.GlobalHub.BroadcastUpdate(0, "TRADE_UPDATE")
		}
	}
}

// GetPlanningChartData 取得規劃圖表數據
func GetPlanningChartData(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountIDStr := c.Query("account_id")
		symbol := c.Query("symbol")
		periodStr := c.Query("period")
		userID := c.GetInt64("user_id")

		accountID, _ := strconv.ParseInt(accountIDStr, 10, 64)
		if accountID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "需要 account_id"})
			return
		}

		// 權限檢查
		var exists int
		db.QueryRow("SELECT 1 FROM accounts WHERE id = ? AND user_id = ?", accountID, userID).Scan(&exists)
		if exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限存取此帳號"})
			return
		}

		// 解析 period
		period := 4 // 預設 M5
		m_min := 5
		tf := "5分"
		if periodStr != "" {
			switch periodStr {
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
		}

		// 判斷是否可用該帳號自己的 cTrader 連線
		sid, err := ctrader.GlobalManager.GetSymbolID(accountID, symbol)
		useGeneric := false
		if err != nil {
			useGeneric = true
		}

		// 計算範圍：顯示最近的 1200 根 K 棒
		toTime := time.Now()
		fromTime := toTime.Add(time.Duration(-1200*m_min) * time.Minute)

		fromTS := fromTime.UnixMilli()
		toTS := toTime.UnixMilli()

		var payload json.RawMessage
		var digits int = 2

		if useGeneric {
			payload, digits, err = ctrader.GlobalManager.GetTrendbarsGeneric(symbol, period, fromTS, toTS)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "目前無法顯示圖表：品種 '" + symbol + "' 不在您的任何活躍 cTrader 帳號中"})
				return
			}
		} else {
			payload, err = ctrader.GlobalManager.GetTrendbars(accountID, sid, period, fromTS, toTS)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "cTrader 請求失敗: " + err.Error()})
				return
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
