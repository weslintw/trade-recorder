package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminAccountUsage struct {
	AccountID     *int64     `json:"account_id"`
	AccountName   *string    `json:"account_name"`
	TradeCount    int        `json:"trade_count"`
	LastTradeTime *time.Time `json:"last_trade_time"`
	PlanCount     int        `json:"plan_count"`
	LastPlanDate  *time.Time `json:"last_plan_date"`
	StorageUsage  int64      `json:"storage_usage"`
}

type AdminUserUsage struct {
	UserID    int64               `json:"user_id"`
	Username  string              `json:"username"`
	IsAdmin   bool                `json:"is_admin"`
	Accounts  []AdminAccountUsage `json:"accounts"`
}

// GetSystemUsageStat 取得全系統使用狀況
func GetSystemUsageStat(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `
			SELECT 
				u.id as user_id, 
				u.username, 
				u.is_admin,
				a.id as account_id, 
				a.name as account_name,
				(
					SELECT COALESCE(SUM(
						LENGTH(COALESCE(entry_strategy_image, '')) +
						LENGTH(COALESCE(entry_strategy_image_original, '')) +
						LENGTH(COALESCE(legend_king_image, '')) +
						LENGTH(COALESCE(legend_king_image_original, '')) +
						LENGTH(COALESCE(legend_htf_image, '')) +
						LENGTH(COALESCE(legend_htf_image_original, '')) +
						LENGTH(COALESCE(entry_signals, '')) +
						LENGTH(COALESCE(entry_checklist, '')) +
						LENGTH(COALESCE(trend_analysis, '')) +
						LENGTH(COALESCE(notes, '')) +
						LENGTH(COALESCE(entry_reason, '')) +
						LENGTH(COALESCE(exit_reason, ''))
					), 0) FROM trades WHERE account_id = a.id
				) + (
					SELECT COALESCE(SUM(LENGTH(COALESCE(image_path, ''))), 0) 
					FROM trade_images 
					WHERE trade_id IN (SELECT id FROM trades WHERE account_id = a.id)
				) + (
					SELECT COALESCE(SUM(
						LENGTH(COALESCE(notes, '')) +
						LENGTH(COALESCE(trend_analysis, ''))
					), 0) FROM daily_plans WHERE account_id = a.id
				) AS storage_usage,
				(SELECT COUNT(*) FROM trades t WHERE t.account_id = a.id) as trade_count,
				(SELECT MAX(entry_time) FROM trades t WHERE t.account_id = a.id) as last_trade_time,
				(SELECT COUNT(*) FROM daily_plans dp WHERE dp.account_id = a.id) as plan_count,
				(SELECT MAX(plan_date) FROM daily_plans dp WHERE dp.account_id = a.id) as last_plan_date
			FROM users u
			LEFT JOIN accounts a ON u.id = a.user_id
			ORDER BY u.id, a.id;
		`

		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢失敗: " + err.Error()})
			return
		}
		defer rows.Close()

		userMap := make(map[int64]*AdminUserUsage)
		var userOrder []int64 // To keep order

		for rows.Next() {
			var userID int64
			var username string
			var isAdmin bool
			var accountID sql.NullInt64
			var accountName sql.NullString
			var storageUsage sql.NullInt64
			var tradeCount int
			var lastTradeTimeStr sql.NullString
			var planCount int
			var lastPlanDateStr sql.NullString

			err := rows.Scan(
				&userID,
				&username,
				&isAdmin,
				&accountID,
				&accountName,
				&storageUsage,
				&tradeCount,
				&lastTradeTimeStr,
				&planCount,
				&lastPlanDateStr,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "資料讀取失敗: " + err.Error()})
				return
			}

			if _, exists := userMap[userID]; !exists {
				userMap[userID] = &AdminUserUsage{
					UserID:   userID,
					Username: username,
					IsAdmin:  isAdmin,
					Accounts: []AdminAccountUsage{},
				}
				userOrder = append(userOrder, userID)
			}

			if accountID.Valid {
				acc := AdminAccountUsage{
					AccountID:    &accountID.Int64,
					AccountName:  &accountName.String,
					TradeCount:   tradeCount,
					PlanCount:    planCount,
					StorageUsage: storageUsage.Int64,
				}
				if lastTradeTimeStr.Valid {
					if t, err := time.Parse(time.RFC3339, lastTradeTimeStr.String); err == nil {
						acc.LastTradeTime = &t
					} else {
						// Try another format just in case, or ignore
						if t, err := time.Parse("2006-01-02 15:04:05", lastTradeTimeStr.String); err == nil {
							acc.LastTradeTime = &t
						}
					}
				}
				if lastPlanDateStr.Valid {
					if t, err := time.Parse(time.RFC3339, lastPlanDateStr.String); err == nil {
						acc.LastPlanDate = &t
					} else {
						if t, err := time.Parse("2006-01-02 15:04:05", lastPlanDateStr.String); err == nil {
							acc.LastPlanDate = &t
						}
					}
				}
				userMap[userID].Accounts = append(userMap[userID].Accounts, acc)
			}
		}

		result := make([]AdminUserUsage, 0, len(userOrder))
		for _, uid := range userOrder {
			result = append(result, *userMap[uid])
		}

		c.JSON(http.StatusOK, result)
	}
}
