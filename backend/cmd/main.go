package main

import (
	"log"
	"os"

	"trade-journal/internal/database"
	"trade-journal/internal/handlers"
	"trade-journal/internal/minio"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化資料庫
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("無法初始化資料庫:", err)
	}
	defer db.Close()

	// 初始化MinIO
	minioClient, err := minio.InitMinIO()
	if err != nil {
		log.Fatal("無法初始化MinIO:", err)
	}

	// 設置Gin路由
	r := gin.Default()

	// CORS設定
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:5173"} // Svelte開發伺服器
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept"}
	r.Use(cors.New(config))

	// 健康檢查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API路由
	api := r.Group("/api/v1")
	{
		// 帳號管理
		accounts := api.Group("/accounts")
		{
			accounts.GET("", handlers.GetAccounts(db))
			accounts.POST("", handlers.CreateAccount(db))
			accounts.PUT("/:id", handlers.UpdateAccount(db))
			accounts.DELETE("/:id", handlers.DeleteAccount(db))
			accounts.POST("/:id/sync", handlers.SyncAccountHistory(db))
		}

		// 交易紀錄
		trades := api.Group("/trades")
		{
			trades.GET("", handlers.GetTrades(db))
			trades.GET("/:id", handlers.GetTrade(db))
			trades.POST("", handlers.CreateTrade(db))
			trades.PUT("/:id", handlers.UpdateTrade(db))
			trades.DELETE("/:id", handlers.DeleteTrade(db))
		}

		// 圖片上傳
		images := api.Group("/images")
		{
			images.POST("/upload", handlers.UploadImage(minioClient))
			images.GET("/:filename", handlers.GetImage(minioClient))
		}

		// 統計資料
		stats := api.Group("/stats")
		{
			stats.GET("/summary", handlers.GetStatsSummary(db))
			stats.GET("/equity-curve", handlers.GetEquityCurve(db))
			stats.GET("/by-symbol", handlers.GetStatsBySymbol(db))
		}

		// 標籤管理
		tags := api.Group("/tags")
		{
			tags.GET("", handlers.GetTags(db))
		}

		// 每日規劃
		dailyPlans := api.Group("/daily-plans")
		{
			dailyPlans.GET("", handlers.GetDailyPlans(db))
			dailyPlans.GET("/:id", handlers.GetDailyPlan(db))
			dailyPlans.POST("", handlers.CreateDailyPlan(db))
			dailyPlans.PUT("/:id", handlers.UpdateDailyPlan(db))
			dailyPlans.DELETE("/:id", handlers.DeleteDailyPlan(db))
		}
	}

	// 啟動伺服器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("伺服器啟動於 http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("伺服器啟動失敗:", err)
	}
}
