package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"trade-journal/internal/database"
	"trade-journal/internal/handlers"
	"trade-journal/internal/middleware"
	"trade-journal/internal/minio"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func changeLogOutput() {
	// 確保日誌同時輸出到檔案與主機控制台 (stdout)
	f, err := os.OpenFile("backend_debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("[WARN] 無法建立日誌檔: %v", err)
		gin.DefaultWriter = os.Stdout
		log.SetOutput(os.Stdout)
		return
	}
	// 系統 log 與 Gin log 都設定為多重輸出
	writer := io.MultiWriter(f, os.Stdout)
	log.SetOutput(writer)
	gin.DefaultWriter = writer
}

func main() {
	changeLogOutput()
	
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
	
	// 從環境變數讀取允許的來源，預設包含本地開發環境
	allowedOrigins := []string{"http://localhost:5173", "http://localhost:5174"}
	if extraOrigins := os.Getenv("ALLOW_ORIGINS"); extraOrigins != "" {
		if extraOrigins == "*" {
			config.AllowAllOrigins = true
		} else {
			origins := strings.Split(extraOrigins, ",")
			allowedOrigins = append(allowedOrigins, origins...)
		}
	}
	
	if !config.AllowAllOrigins {
		config.AllowOrigins = allowedOrigins
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// 健康檢查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API路由
	api := r.Group("/api/v1")
	{
		// 認證路由
		auth := api.Group("/auth")
		{
			// 公開路徑
			auth.POST("/register", handlers.Register(db))
			auth.POST("/login", handlers.Login(db))

			// 需要認證的路徑
			protectedAuth := auth.Group("")
			protectedAuth.Use(middleware.AuthMiddleware())
			{
				protectedAuth.GET("/me", handlers.GetCurrentUser(db))
				protectedAuth.POST("/change-password", handlers.ChangePassword(db))
			}
		}

		// 其他需要認證的路由
		authorized := api.Group("")
		authorized.Use(middleware.AuthMiddleware())
		{
			// 帳號管理

			// 帳號管理
			accounts := authorized.Group("/accounts")
			{
				accounts.GET("", handlers.GetAccounts(db))
				accounts.POST("", handlers.CreateAccount(db))
				accounts.PUT("/:id", handlers.UpdateAccount(db))
				accounts.DELETE("/:id", handlers.DeleteAccount(db))
				accounts.DELETE("/:id/data", handlers.ClearAccountData(db))
				accounts.POST("/:id/sync", handlers.SyncAccountHistory(db))
				accounts.POST("/:id/import-csv", handlers.ImportTradesCSV(db))
			}

			// 交易紀錄
			trades := authorized.Group("/trades")
			{
				trades.GET("", handlers.GetTrades(db))
				trades.GET("/:id", handlers.GetTrade(db))
				trades.POST("", handlers.CreateTrade(db))
				trades.PUT("/:id", handlers.UpdateTrade(db))
				trades.DELETE("/:id", handlers.DeleteTrade(db))
			}

			// 統計資料
			stats := authorized.Group("/stats")
			{
				stats.GET("/summary", handlers.GetStatsSummary(db))
				stats.GET("/equity-curve", handlers.GetEquityCurve(db))
				stats.GET("/by-symbol", handlers.GetStatsBySymbol(db))
				stats.GET("/by-strategy", handlers.GetStatsByStrategy(db))
			}

			// 標籤管理
			tags := authorized.Group("/tags")
			{
				tags.GET("", handlers.GetTags(db))
			}

			// 每日規劃
			dailyPlans := authorized.Group("/daily-plans")
			{
				dailyPlans.GET("", handlers.GetDailyPlans(db))
				dailyPlans.GET("/:id", handlers.GetDailyPlan(db))
				dailyPlans.POST("", handlers.CreateDailyPlan(db))
				dailyPlans.PUT("/:id", handlers.UpdateDailyPlan(db))
				dailyPlans.DELETE("/:id", handlers.DeleteDailyPlan(db))
			}

			// 分享管理
			authorized.POST("/shares", handlers.CreateShare(db))
		}

		// 分享路由 (公開)
		api.GET("/shares/public/:token", handlers.GetSharedResource(db))

		// 圖片上傳 (目前先保持公開或也可加入認證)
		images := api.Group("/images")
		{
			images.POST("/upload", handlers.UploadImage(minioClient))
			images.GET("/:filename", handlers.GetImage(minioClient))
		}
	}

	// 靜態檔案服務 (用於打包後的版本)
	// 假設打包後的結構中，frontend/dist 就在執行檔同層或上層
	staticDirs := []string{"./frontend/dist", "../frontend/dist", "./dist"}
	var staticDir string
	for _, dir := range staticDirs {
		if _, err := os.Stat(dir); err == nil {
			staticDir = dir
			break
		}
	}

	if staticDir != "" {
		log.Printf("正在從 %s 服務靜態檔案", staticDir)
		// 服務靜態資源 (assets 由於有雜湊檔名，建議保留專屬路由)
		r.StaticFS("/assets", http.Dir(filepath.Join(staticDir, "assets")))
		
		// SPA Fallback: 任何不匹配 API 的路由都導向 index.html
		r.NoRoute(func(c *gin.Context) {
			// 首先檢查路徑是否對應到靜態目錄下的檔案（例如 /logo.png, /favicon.ico）
			path := c.Request.URL.Path
			filePath := filepath.Join(staticDir, path)
			if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
				c.File(filePath)
				return
			}

			// 如果不是檔案且不是 API 請求，則返回 index.html (SPA 路由)
			if !strings.HasPrefix(path, "/api/") {
				c.File(filepath.Join(staticDir, "index.html"))
			}
		})
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
