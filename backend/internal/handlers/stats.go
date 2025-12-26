package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"trade-journal/internal/models"
)

// GetStatsSummary 取得統計摘要
func GetStatsSummary(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var stats models.StatsSummary

		// 總交易數
		db.QueryRow("SELECT COUNT(*) FROM trades WHERE exit_price IS NOT NULL").Scan(&stats.TotalTrades)

		// 勝場數與敗場數
		db.QueryRow("SELECT COUNT(*) FROM trades WHERE pnl > 0").Scan(&stats.WinningTrades)
		db.QueryRow("SELECT COUNT(*) FROM trades WHERE pnl < 0").Scan(&stats.LosingTrades)

		// 勝率
		if stats.TotalTrades > 0 {
			stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100
		}

		// 總盈虧
		db.QueryRow("SELECT COALESCE(SUM(pnl), 0) FROM trades WHERE pnl IS NOT NULL").Scan(&stats.TotalPnL)

		// 平均盈虧
		if stats.TotalTrades > 0 {
			stats.AveragePnL = stats.TotalPnL / float64(stats.TotalTrades)
		}

		// 最大盈利
		db.QueryRow("SELECT COALESCE(MAX(pnl), 0) FROM trades WHERE pnl > 0").Scan(&stats.LargestWin)

		// 最大虧損
		db.QueryRow("SELECT COALESCE(MIN(pnl), 0) FROM trades WHERE pnl < 0").Scan(&stats.LargestLoss)

		// 盈虧比（Profit Factor）
		var totalProfit, totalLoss float64
		db.QueryRow("SELECT COALESCE(SUM(pnl), 0) FROM trades WHERE pnl > 0").Scan(&totalProfit)
		db.QueryRow("SELECT COALESCE(ABS(SUM(pnl)), 0) FROM trades WHERE pnl < 0").Scan(&totalLoss)

		if totalLoss > 0 {
			stats.ProfitFactor = totalProfit / totalLoss
		}

		c.JSON(http.StatusOK, stats)
	}
}

// GetEquityCurve 取得淨值曲線
func GetEquityCurve(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rows, err := db.Query(`
			SELECT DATE(exit_time) as date, SUM(pnl) as daily_pnl
			FROM trades
			WHERE exit_time IS NOT NULL AND pnl IS NOT NULL
			GROUP BY DATE(exit_time)
			ORDER BY date ASC
		`)
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
		rows, err := db.Query(`
			SELECT 
				symbol,
				COUNT(*) as total_trades,
				SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as winning_trades,
				COALESCE(SUM(pnl), 0) as total_pnl
			FROM trades
			WHERE exit_price IS NOT NULL
			GROUP BY symbol
			ORDER BY total_trades DESC
		`)
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

