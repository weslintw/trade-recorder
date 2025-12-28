package models

import "time"

// DailyPlan 每日規劃模型
type DailyPlan struct {
	ID            int64     `json:"id"`
	AccountID     int64     `json:"account_id"`
	PlanDate      time.Time `json:"plan_date"`
	Symbol        string    `json:"symbol"`
	MarketSession string    `json:"market_session"`
	Notes         string    `json:"notes"`
	TrendAnalysis string    `json:"trend_analysis"` // JSON string
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// DailyPlanCreate 建立每日規劃請求
type DailyPlanCreate struct {
	AccountID     int64     `json:"account_id" binding:"required"`
	PlanDate      time.Time `json:"plan_date" binding:"required"`
	Symbol        string    `json:"symbol"`
	MarketSession string    `json:"market_session"`
	Notes         string    `json:"notes"`
	TrendAnalysis string    `json:"trend_analysis"` // JSON string
}

// DailyPlanQuery 查詢參數
type DailyPlanQuery struct {
	AccountID     int64  `form:"account_id"`
	StartDate     string `form:"start_date"`
	EndDate       string `form:"end_date"`
	Symbol        string `form:"symbol"`
	MarketSession string `form:"market_session"`
	Page          int    `form:"page"`
	PageSize      int    `form:"page_size"`
}
