package models

import "time"

// Trade 交易紀錄模型
type Trade struct {
	ID             int64      `json:"id"`
	TradeType      string     `json:"trade_type"` // "actual"=實際交易 或 "observation"=純觀察
	Symbol         string     `json:"symbol"`
	Side           string     `json:"side"` // "long" 或 "short"
	EntryPrice     float64    `json:"entry_price"`
	ExitPrice      *float64   `json:"exit_price,omitempty"`
	LotSize        float64    `json:"lot_size"`
	PnL            *float64   `json:"pnl,omitempty"`
	PnLPoints      *float64   `json:"pnl_points,omitempty"`
	Notes          string     `json:"notes"`
	EntryReason    *string    `json:"entry_reason,omitempty"`
	ExitReason     *string    `json:"exit_reason,omitempty"`
	EntryStrategy  *string    `json:"entry_strategy,omitempty"`  // "expert"=達人, "elite"=菁英, "legend"=傳奇
	EntryStrategyImage *string `json:"entry_strategy_image,omitempty"` // 進場種類圖片(Base64)
	EntrySignals   *string    `json:"entry_signals,omitempty"`   // JSON array of selected signals
	EntryChecklist *string    `json:"entry_checklist,omitempty"` // JSON object of checklist items
	EntryPattern   *string    `json:"entry_pattern,omitempty"`   // 進場樣態（僅菁英使用）
	TrendAnalysis  *string    `json:"trend_analysis,omitempty"`  // JSON object of trend per timeframe
	EntryTimeframe *string    `json:"entry_timeframe,omitempty"` // "M1", "M5", "M15", "M30", "H1", "H4", "D1"
	TrendType      *string    `json:"trend_type,omitempty"`      // "with_trend"=順勢, "against_trend"=逆勢
	MarketSession  *string    `json:"market_session,omitempty"`  // "asian"=亞盤, "european"=歐盤, "us"=美盤
	TimezoneOffset *int       `json:"timezone_offset,omitempty"` // UTC 偏移，如 8 表示 UTC+8
	EntryTime      time.Time  `json:"entry_time"`
	ExitTime       *time.Time `json:"exit_time,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	Images         []Image    `json:"images,omitempty"`
	Tags           []Tag      `json:"tags,omitempty"`
}

// Image 圖片模型
type Image struct {
	ID        int64     `json:"id"`
	TradeID   int64     `json:"trade_id"`
	ImageType string    `json:"image_type"` // "entry" 或 "exit"
	ImagePath string    `json:"image_path"`
	CreatedAt time.Time `json:"created_at"`
}

// Tag 標籤模型
type Tag struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// TradeCreate 建立交易請求
type TradeCreate struct {
	TradeType      string    `json:"trade_type" binding:"required,oneof=actual observation"`
	Symbol         string    `json:"symbol" binding:"required"`
	Side           string    `json:"side" binding:"required,oneof=long short"`
	EntryPrice     *float64  `json:"entry_price"` // observation 時可為空
	ExitPrice      *float64  `json:"exit_price"`
	LotSize        *float64  `json:"lot_size"` // observation 時可為空
	PnL            *float64  `json:"pnl"`
	PnLPoints      *float64  `json:"pnl_points"`
	Notes          string    `json:"notes"`
	EntryReason    string    `json:"entry_reason"`
	ExitReason     string    `json:"exit_reason"`
	EntryStrategy  string    `json:"entry_strategy"` // "expert", "elite", "legend"
	EntryStrategyImage string `json:"entry_strategy_image"` // 進場種類圖片(Base64)
	EntrySignals   string    `json:"entry_signals"`  // JSON array
	EntryChecklist string    `json:"entry_checklist"` // JSON object
	EntryPattern   string    `json:"entry_pattern"`   // 進場樣態
	TrendAnalysis  string    `json:"trend_analysis"`  // JSON object
	EntryTimeframe string    `json:"entry_timeframe"` // "M1", "M5", "M15", "M30", "H1", "H4", "D1"
	TrendType      string    `json:"trend_type"`      // "with_trend", "against_trend"
	MarketSession  string    `json:"market_session"`  // "asian", "european", "us"
	TimezoneOffset int       `json:"timezone_offset"` // UTC 偏移
	EntryTime      time.Time `json:"entry_time" binding:"required"`
	ExitTime       *time.Time `json:"exit_time"`
	Tags           []string  `json:"tags"`
	Images         []ImageUpload `json:"images"`
}

// ImageUpload 圖片上傳資料
type ImageUpload struct {
	ImageType string `json:"image_type" binding:"required,oneof=entry exit"`
	ImagePath string `json:"image_path" binding:"required"`
}

// TradeQuery 查詢參數
type TradeQuery struct {
	Symbol    string `form:"symbol"`
	Side      string `form:"side"`
	Tag       string `form:"tag"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

// StatsSummary 統計摘要
type StatsSummary struct {
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
	AveragePnL    float64 `json:"average_pnl"`
	LargestWin    float64 `json:"largest_win"`
	LargestLoss   float64 `json:"largest_loss"`
	ProfitFactor  float64 `json:"profit_factor"`
}

// EquityPoint 淨值曲線點
type EquityPoint struct {
	Date   string  `json:"date"`
	Equity float64 `json:"equity"`
}

// SymbolStats 品種統計
type SymbolStats struct {
	Symbol        string  `json:"symbol"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
}

