package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"
	"trade-journal/internal/models"
	"trade-journal/internal/utils"

	"github.com/gin-gonic/gin"
)

// Register 註冊使用者
func Register(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserRegister
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法處理密碼"})
			return
		}

		// 檢查是否為第一個使用者，若是則設為管理員
		var count int
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		isAdmin := count == 0

		res, err := db.Exec(
			"INSERT INTO users (username, password, is_admin) VALUES (?, ?, ?)",
			req.Username, hashedPassword, isAdmin,
		)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "使用者名稱已存在"})
			return
		}

		userID, _ := res.LastInsertId()

		// 為新使用者建立預設帳號
		db.Exec("INSERT INTO accounts (name, type, user_id) VALUES (?, ?, ?)", "預設帳號", "local", userID)

		token, err := utils.GenerateToken(userID, req.Username, isAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法生成認證權杖"})
			return
		}

		c.JSON(http.StatusCreated, models.AuthResponse{
			Token: token,
			User: models.User{
				ID:       userID,
				Username: req.Username,
				IsAdmin:  isAdmin,
			},
		})
	}
}

// Login 登入使用者
func Login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.UserLogin
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 去除可能的空格
		req.Username = strings.TrimSpace(req.Username)
		req.Password = strings.TrimSpace(req.Password)

		var user models.User
		err := db.QueryRow(
			"SELECT id, username, password, is_admin FROM users WHERE username = ?",
			req.Username,
		).Scan(&user.ID, &user.Username, &user.Password, &user.IsAdmin)

		if err != nil {
			log.Printf("[LOGIN FAIL] 查詢使用者失敗: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "使用者名稱或密碼錯誤"})
			return
		}

		match := utils.CheckPasswordHash(req.Password, user.Password)
		log.Printf("[LOGIN] User: %s (ID: %d), Match: %v", user.Username, user.ID, match)

		if !match {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "使用者名稱或密碼錯誤"})
			return
		}

		token, err := utils.GenerateToken(user.ID, user.Username, user.IsAdmin)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "無法生成認證權杖"})
			return
		}

		c.JSON(http.StatusOK, models.AuthResponse{
			Token: token,
			User:  user,
		})
	}
}

// GetCurrentUser 獲取當前使用者資訊
func GetCurrentUser(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		
		var user models.User
		err := db.QueryRow(
			"SELECT id, username, is_admin, created_at FROM users WHERE id = ?",
			userID,
		).Scan(&user.ID, &user.Username, &user.IsAdmin, &user.CreatedAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "找不到使用者"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// ChangePassword 修改使用者密碼
func ChangePassword(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		var req models.UserChangePassword
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[DEBUG] 修改密碼請求綁定失敗: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "請提供正確的舊密碼與新密碼"})
			return
		}

		// 去除前後空格，避免複製貼上帶入換行符
		req.OldPassword = strings.TrimSpace(req.OldPassword)
		req.NewPassword = strings.TrimSpace(req.NewPassword)

		log.Printf("[DEBUG] 嘗試修改密碼: UserID=%d, OldPwdLen=%d, NewPwdLen=%d", userID, len(req.OldPassword), len(req.NewPassword))

		// 獲取目前的雜湊密碼
		var currentHash string
		err := db.QueryRow("SELECT password FROM users WHERE id = ?", userID).Scan(&currentHash)
		if err != nil {
			log.Printf("[DEBUG] 查詢 UserID=%d 密碼失敗: %v", userID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查詢使用者資訊失敗"})
			return
		}

		log.Printf("[DEBUG] 資料庫中的雜湊長度: %d, 前 10 位: %s", len(currentHash), currentHash[:min(10, len(currentHash))])

		// 驗證舊密碼
		if !utils.CheckPasswordHash(req.OldPassword, currentHash) {
			log.Printf("[DEBUG] 舊密碼驗證不符: UserID=%d, HashMatch=false", userID)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "舊密碼不正確",
				"debug": gin.H{
					"user_id": userID,
					"hash_len": len(currentHash),
				},
			})
			return
		}

		// 加密新密碼
		hashedPassword, err := utils.HashPassword(req.NewPassword)
		if err != nil {
			log.Printf("[DEBUG] 新密碼雜湊失敗: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密碼處理失敗"})
			return
		}

		// 更新密碼
		result, err := db.Exec("UPDATE users SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", hashedPassword, userID)
		if err != nil {
			log.Printf("[DEBUG] 更新資料庫密碼失敗: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新密碼失敗"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			log.Printf("[DEBUG] 更新失敗：找不到 UserID=%d，沒有任何行被更動", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失敗，找不到該使用者的資料"})
			return
		}

		log.Printf("[DEBUG] UserID=%d 密碼修改成功，RowsAffected=%d", userID, rows)
		c.JSON(http.StatusOK, gin.H{"message": "密碼修改成功"})
	}
}
