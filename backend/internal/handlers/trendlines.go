package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SaveTrendlines 儲存趨勢線
func SaveTrendlines(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tradeID := c.Param("id")
		userID := c.GetInt64("user_id")

		// 驗證權限
		var exists int
		err := db.QueryRow("SELECT 1 FROM trades t JOIN accounts a ON t.account_id = a.id WHERE t.id = ? AND a.user_id = ?", tradeID, userID).Scan(&exists)
		if err == sql.ErrNoRows || exists == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "無權限"})
			return
		}

		var linesData interface{}
		if err := c.ShouldBindJSON(&linesData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 將JSON轉為字串儲存
		var jsonStr string
		if linesData != nil {
			bytes, err := json.Marshal(linesData)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "無效的JSON格式"})
				return
			}
			jsonStr = string(bytes)
		}

		_, err = db.Exec("UPDATE trades SET trendlines = ? WHERE id = ?", jsonStr, tradeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "儲存成功"})
	}
}

// GetTrendlines 取得趨勢線
func GetTrendlines(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tradeID := c.Param("id")
		userID := c.GetInt64("user_id")

		var lines sql.NullString
		err := db.QueryRow(`
			SELECT t.trendlines
			FROM trades t
			JOIN accounts a ON t.account_id = a.id
			WHERE t.id = ? AND a.user_id = ?
		`, tradeID, userID).Scan(&lines)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "交易不存在"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !lines.Valid || lines.String == "" || lines.String == "null" {
			c.Data(http.StatusOK, "application/json", []byte("[]"))
			return
		}

		// 驗證JSON格式
		var test interface{}
		if err := json.Unmarshal([]byte(lines.String), &test); err != nil {
			// JSON無效，返回空陣列
			c.Data(http.StatusOK, "application/json", []byte("[]"))
			return
		}

		c.Data(http.StatusOK, "application/json", []byte(lines.String))
	}
}
