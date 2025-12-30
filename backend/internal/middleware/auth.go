package middleware

import (
	"net/http"
	"strings"
	"trade-journal/internal/utils"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 認證中間件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "需要授權標頭"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "授權標頭格式不正確"})
			c.Abort()
			return
		}

		claims, err := utils.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無效或已過期的權杖"})
			c.Abort()
			return
		}

		// 將使用者資訊存入 Context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("is_admin", claims.IsAdmin)

		c.Next()
	}
}

// AdminMiddleware 管理員中間件
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin := c.GetBool("is_admin")
		if !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "權限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}
