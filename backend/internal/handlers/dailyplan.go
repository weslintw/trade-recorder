package handlers

import (
	"database/sql"
	"net/http"

	"trade-journal/internal/models"

	"github.com/gin-gonic/gin"
)

// GetDailyPlans 取得每日規劃清單
func GetDailyPlans(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		offset := (query.Page - 1) * query.PageSize

		// 建立查詢
		sqlQuery := `
			SELECT id, account_id, plan_date, symbol, market_session, COALESCE(notes, ''), COALESCE(trend_analysis, '{}'), created_at, updated_at
			FROM daily_plans
			WHERE 1=1
		`

		args := []interface{}{}

		if query.AccountID > 0 {
			sqlQuery += " AND account_id = ?"
			args = append(args, query.AccountID)
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
			err := rows.Scan(
				&plan.ID, &plan.AccountID, &plan.PlanDate, &plan.Symbol, &plan.MarketSession, &plan.Notes,
				&plan.TrendAnalysis, &plan.CreatedAt, &plan.UpdatedAt,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			plans = append(plans, plan)
		}

		// 計算總數
		var total int
		countQuery := `SELECT COUNT(*) FROM daily_plans WHERE 1=1`
		countArgs := []interface{}{}

		if query.AccountID > 0 {
			countQuery += " AND account_id = ?"
			countArgs = append(countArgs, query.AccountID)
		}

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

		var plan models.DailyPlan
		err := db.QueryRow(`
			SELECT id, account_id, plan_date, symbol, market_session, COALESCE(notes, ''), COALESCE(trend_analysis, '{}'), created_at, updated_at
			FROM daily_plans WHERE id = ?
		`, id).Scan(
			&plan.ID, &plan.AccountID, &plan.PlanDate, &plan.Symbol, &plan.MarketSession, &plan.Notes,
			&plan.TrendAnalysis, &plan.CreatedAt, &plan.UpdatedAt,
		)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "規劃不存在"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
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
	}
}

// DeleteDailyPlan 刪除每日規劃
func DeleteDailyPlan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		_, err := db.Exec("DELETE FROM daily_plans WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "規劃刪除成功"})
	}
}
