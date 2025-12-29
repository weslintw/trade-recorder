package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"trade-journal/internal/models"

	"github.com/gin-gonic/gin"
)

// GetStatsSummary 取得統計摘要
func GetStatsSummary(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Query("account_id")
		if accountID == "" {
			accountID = "1"
		}

		var stats models.StatsSummary

		// 總交易數
		db.QueryRow("SELECT COUNT(*) FROM trades WHERE account_id = ? AND exit_price IS NOT NULL", accountID).Scan(&stats.TotalTrades)

		// 勝場數與敗場數
		db.QueryRow("SELECT COUNT(*) FROM trades WHERE account_id = ? AND pnl > 0", accountID).Scan(&stats.WinningTrades)
		db.QueryRow("SELECT COUNT(*) FROM trades WHERE account_id = ? AND pnl < 0", accountID).Scan(&stats.LosingTrades)

		// 勝率
		if stats.TotalTrades > 0 {
			stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
		}

		// 總盈虧
		db.QueryRow("SELECT COALESCE(SUM(pnl), 0) FROM trades WHERE account_id = ? AND pnl IS NOT NULL", accountID).Scan(&stats.TotalPnL)

		// 平均盈虧
		if stats.TotalTrades > 0 {
			stats.AveragePnL = stats.TotalPnL / float64(stats.TotalTrades)
		}

		// 最大盈利
		db.QueryRow("SELECT COALESCE(MAX(pnl), 0) FROM trades WHERE account_id = ? AND pnl > 0", accountID).Scan(&stats.LargestWin)

		// 最大虧損
		db.QueryRow("SELECT COALESCE(MIN(pnl), 0) FROM trades WHERE account_id = ? AND pnl < 0", accountID).Scan(&stats.LargestLoss)

		// 盈虧比（Profit Factor）
		var totalProfit, totalLoss float64
		db.QueryRow("SELECT COALESCE(SUM(pnl), 0) FROM trades WHERE account_id = ? AND pnl > 0", accountID).Scan(&totalProfit)
		db.QueryRow("SELECT COALESCE(ABS(SUM(pnl)), 0) FROM trades WHERE account_id = ? AND pnl < 0", accountID).Scan(&totalLoss)

		if totalLoss > 0 {
			stats.ProfitFactor = totalProfit / totalLoss
		}

		c.JSON(http.StatusOK, stats)
	}
}

// GetEquityCurve 取得淨值曲線
func GetEquityCurve(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Query("account_id")
		if accountID == "" {
			accountID = "1"
		}

		rows, err := db.Query(`
			SELECT DATE(exit_time) as date, SUM(pnl) as daily_pnl
			FROM trades
			WHERE account_id = ? AND exit_time IS NOT NULL AND pnl IS NOT NULL
			GROUP BY DATE(exit_time)
			ORDER BY date ASC
		`, accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		equityCurve := []models.EquityPoint{}
		cumulativeEquity := 0.0

		for rows.Next() {
			var date string
			var dailyPnL float64
			rows.Scan(&date, &dailyPnL)

			cumulativeEquity += dailyPnL
			equityCurve = append(equityCurve, models.EquityPoint{
				Date:   date,
				Equity: cumulativeEquity,
			})
		}

		c.JSON(http.StatusOK, equityCurve)
	}
}

// GetStatsBySymbol 取得各品種統計
func GetStatsBySymbol(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Query("account_id")
		if accountID == "" {
			accountID = "1"
		}

		rows, err := db.Query(`
			SELECT 
				symbol,
				COUNT(*) as total_trades,
				SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
				COALESCE(SUM(pnl), 0) as total_pnl
			FROM trades
			WHERE account_id = ? AND exit_price IS NOT NULL
			GROUP BY symbol
			ORDER BY total_trades DESC
		`, accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		symbolStats := []models.SymbolStats{}
		for rows.Next() {
			var stat models.SymbolStats
			rows.Scan(&stat.Symbol, &stat.TotalTrades, &stat.WinningTrades, &stat.TotalPnL)

			if stat.TotalTrades > 0 {
				stat.WinRate = float64(stat.WinningTrades) / float64(stat.TotalTrades) * 100
			}

			symbolStats = append(symbolStats, stat)
		}

		c.JSON(http.StatusOK, symbolStats)
	}
}

// GetStatsByStrategy 取得各策略統計 (包含子項目)
func GetStatsByStrategy(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID := c.Query("account_id")
		if accountID == "" {
			accountID = "1"
		}

		rows, err := db.Query(`
			SELECT 
				COALESCE(entry_strategy, 'unspecified') as strategy,
				entry_signals,
				entry_checklist,
				entry_pattern,
				pnl
			FROM trades
			WHERE account_id = ? AND exit_price IS NOT NULL
		`, accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		strategyMap := make(map[string]*models.StrategyStats)
		// 子項目統計的 Map: strategy -> subitemName -> stats
		subItemMap := make(map[string]map[string]*models.SubItemStats)

		for rows.Next() {
			var strategy string
			var signalsRaw, checklistRaw, patternRaw sql.NullString
			var pnl float64
			rows.Scan(&strategy, &signalsRaw, &checklistRaw, &patternRaw, &pnl)

			if _, ok := strategyMap[strategy]; !ok {
				strategyMap[strategy] = &models.StrategyStats{
					Strategy: strategy,
					SubItemStats: []models.SubItemStats{}, // 初始化為空陣列
				}
				subItemMap[strategy] = make(map[string]*models.SubItemStats)
			}

			s := strategyMap[strategy]
			s.TotalTrades++
			if pnl > 0 {
				s.WinningTrades++
			}
			s.TotalPnL += pnl

			// 處理子項目
			var items []string
			if strategy == "expert" && signalsRaw.Valid && signalsRaw.String != "[]" && signalsRaw.String != "" {
				var sigs []string
				json.Unmarshal([]byte(signalsRaw.String), &sigs)
				items = sigs
			} else if (strategy == "elite" || strategy == "legend") && checklistRaw.Valid && checklistRaw.String != "{}" && checklistRaw.String != "" {
				var checklist map[string]bool
				json.Unmarshal([]byte(checklistRaw.String), &checklist)
				for item, checked := range checklist {
					if checked {
						items = append(items, item)
					}
				}
			}
			
			// 菁英還有樣態
			if strategy == "elite" && patternRaw.Valid && patternRaw.String != "[]" && patternRaw.String != "" {
				var patterns []struct{ Name string }
				json.Unmarshal([]byte(patternRaw.String), &patterns)
				for _, p := range patterns {
					if p.Name != "" {
						items = append(items, "樣態: "+p.Name)
					}
				}
			}

			for _, itemName := range items {
				if itemName == "" { continue }
				if _, ok := subItemMap[strategy][itemName]; !ok {
					subItemMap[strategy][itemName] = &models.SubItemStats{Name: itemName}
				}
				sub := subItemMap[strategy][itemName]
				sub.TotalTrades++
				if pnl > 0 {
					sub.WinningTrades++
				}
				sub.TotalPnL += pnl
			}
		}

		result := []models.StrategyStats{}
		for stra, s := range strategyMap {
			if s.TotalTrades > 0 {
				s.WinRate = float64(s.WinningTrades) / float64(s.TotalTrades) * 100
			}

			// 轉換子項目 Map 為 Slice 並排序
			for _, sub := range subItemMap[stra] {
				if sub.TotalTrades > 0 {
					sub.WinRate = float64(sub.WinningTrades) / float64(sub.TotalTrades) * 100
				}
				s.SubItemStats = append(s.SubItemStats, *sub)
			}
			sort.Slice(s.SubItemStats, func(i, j int) bool {
				return s.SubItemStats[i].TotalTrades > s.SubItemStats[j].TotalTrades
			})

			result = append(result, *s)
		}

		// 如果完全沒資料，回傳空陣列而不是 null
		if len(result) == 0 {
			c.JSON(http.StatusOK, []models.StrategyStats{})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}
