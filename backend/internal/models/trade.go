package models

import "time"

// Trade 交易紀錄模型
type Trade struct {
	ID                         int64      `json:"id"`
	AccountID                  int64      `json:"account_id"`
	TradeType                  string     `json:"trade_type"` // "actual"=實際交易 或 "observation"=純觀察
	Symbol                     string     `json:"symbol"`
	Side                       string     `json:"side"` // "long" 或 "short"
	EntryPrice                 *float64   `json:"entry_price"`
	ExitPrice                  *float64   `json:"exit_price,omitempty"`
	LotSize                    *float64   `json:"lot_size"`
	PnL                        *float64   `json:"pnl,omitempty"`
	PnLPoints                  *float64   `json:"pnl_points,omitempty"`
	Notes                      string     `json:"notes"`
	EntryReason                *string    `json:"entry_reason,omitempty"`
	ExitReason                 *string    `json:"exit_reason,omitempty"`
	EntryStrategy              *string    `json:"entry_strategy,omitempty"`                // "expert"=達人, "elite"=菁英, "legend"=傳奇
	EntryStrategyImage         *string    `json:"entry_strategy_image,omitempty"`          // 進場種類圖片(Base64)
	EntryStrategyImageOriginal *string    `json:"entry_strategy_image_original,omitempty"` // 進場種類原始圖片(Base64)
	EntrySignals               *string    `json:"entry_signals,omitempty"`                 // JSON array of selected signals
	EntryChecklist             *string    `json:"entry_checklist,omitempty"`               // JSON object of checklist items
	EntryPattern               *string    `json:"entry_pattern,omitempty"`                 // 進場樣態（僅菁英使用）
	TrendAnalysis              *string    `json:"trend_analysis,omitempty"`                // JSON object of trend per timeframe
	EntryTimeframe             *string    `json:"entry_timeframe,omitempty"`               // "M1", "M5", "M15", "M30", "H1", "H4", "D1"
	TrendType                  *string    `json:"trend_type,omitempty"`                    // "with_trend"=順勢, "against_trend"=逆勢
	MarketSession              *string    `json:"market_session,omitempty"`                // "asian"=亞盤, "european"=歐盤, "us"=美盤
	InitialSL                  *float64   `json:"initial_sl,omitempty"`                    // 初始停損價
	BulletSize                 *float64   `json:"bullet_size,omitempty"`                   // 子彈大小 (風險金額)
	RRRatio                    *float64   `json:"rr_ratio,omitempty"`                      // 風報比
	TimezoneOffset             *int       `json:"timezone_offset,omitempty"`               // UTC 偏移，如 8 表示 UTC+8
	Ticket                     *string    `json:"ticket,omitempty"`                        // 平台成交編號
	ExitSL                     *float64   `json:"exit_sl,omitempty"`                       // 平倉時的停損價
	LegendKingHTF              *string    `json:"legend_king_htf,omitempty"`               // 傳奇：王者回調時區
	LegendKingImage            *string    `json:"legend_king_image,omitempty"`             // 傳奇：王者回調圖片
	LegendKingImageOriginal    *string    `json:"legend_king_image_original,omitempty"`
	LegendHTF                  *string    `json:"legend_htf,omitempty"`                    // 傳奇：大時區破測破時區
	LegendHTFImage             *string    `json:"legend_htf_image,omitempty"`              // 傳奇：大時區圖片
	LegendHTFImageOriginal     *string    `json:"legend_htf_image_original,omitempty"`
	LegendDeHTF                *string    `json:"legend_de_htf,omitempty"`                 // 傳奇：整理段時區
	EntryTime                  time.Time  `json:"entry_time"`
	ColorTag                   *string    `json:"color_tag,omitempty"`                     // "red", "yellow", "green"
	ExitTime                   *time.Time `json:"exit_time,omitempty"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	Images                     []Image    `json:"images,omitempty"`
	Tags                       []Tag      `json:"tags,omitempty"`
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
	AccountID                  int64         `json:"account_id" binding:"required"`
	TradeType                  string        `json:"trade_type" binding:"required,oneof=actual observation"`
	Symbol                     string        `json:"symbol" binding:"required"`
	Side                       string        `json:"side" binding:"required,oneof=long short"`
	EntryPrice                 *float64      `json:"entry_price"` // observation 時可為空
	ExitPrice                  *float64      `json:"exit_price"`
	LotSize                    *float64      `json:"lot_size"` // observation 時可為空
	PnL                        *float64      `json:"pnl"`
	PnLPoints                  *float64      `json:"pnl_points"`
	Notes                      string        `json:"notes"`
	EntryReason                string        `json:"entry_reason"`
	ExitReason                 string        `json:"exit_reason"`
	EntryStrategy              string        `json:"entry_strategy"`                // "expert", "elite", "legend"
	EntryStrategyImage         string        `json:"entry_strategy_image"`          // 進場種類圖片(Base64)
	EntryStrategyImageOriginal string        `json:"entry_strategy_image_original"` // 進場種類原始圖片(Base64)
	EntrySignals               string        `json:"entry_signals"`                 // JSON array
	EntryChecklist             string        `json:"entry_checklist"`               // JSON object
	EntryPattern               string        `json:"entry_pattern"`                 // 進場樣態
	TrendAnalysis              string        `json:"trend_analysis"`                // JSON object
	EntryTimeframe             string        `json:"entry_timeframe"`               // "M1", "M5", "M15", "M30", "H1", "H4", "D1"
	TrendType                  string        `json:"trend_type"`                    // "with_trend", "against_trend"
	MarketSession              string        `json:"market_session"`                // "asian", "european", "us"
	InitialSL                  *float64      `json:"initial_sl"`
	BulletSize                 *float64      `json:"bullet_size"`
	RRRatio                    *float64      `json:"rr_ratio"`
	TimezoneOffset             int           `json:"timezone_offset"` // UTC 偏移
	ExitSL                     *float64      `json:"exit_sl"`         // 平倉時的停損價
	LegendKingHTF              string        `json:"legend_king_htf"`
	LegendKingImage            string        `json:"legend_king_image"`
	LegendKingImageOriginal    string        `json:"legend_king_image_original"`
	LegendHTF                  string        `json:"legend_htf"`
	LegendHTFImage             string        `json:"legend_htf_image"`
	LegendHTFImageOriginal     string        `json:"legend_htf_image_original"`
	LegendDeHTF                string        `json:"legend_de_htf"`
	EntryTime                  time.Time     `json:"entry_time" binding:"required"`
	ColorTag                   string        `json:"color_tag"`
	ExitTime                   *time.Time    `json:"exit_time"`
	Tags                       []string      `json:"tags"`
	Images                     []ImageUpload `json:"images"`
}

// ImageUpload 圖片上傳資料
type ImageUpload struct {
	ImageType string `json:"image_type" binding:"required,oneof=entry exit"`
	ImagePath string `json:"image_path" binding:"required"`
}

// TradeQuery 查詢參數
type TradeQuery struct {
	AccountID int64  `form:"account_id"`
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

// StrategyStats 策略統計 (達人/菁英/傳奇)
type StrategyStats struct {
	Strategy      string            `json:"strategy"`
	TotalTrades   int               `json:"total_trades"`
	WinningTrades int               `json:"winning_trades"`
	WinRate       float64           `json:"win_rate"`
	TotalPnL      float64           `json:"total_pnl"`
	SubItemStats  []SubItemStats    `json:"sub_item_stats"`
}

// SubItemStats 策略子項目統計 (訊號/樣態/檢查項)
type SubItemStats struct {
	Name          string  `json:"name"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
}

// ColorStats 顏色標籤統計
type ColorStats struct {
	Color         string  `json:"color"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	WinRate       float64 `json:"win_rate"`
	TotalPnL      float64 `json:"total_pnl"`
}
