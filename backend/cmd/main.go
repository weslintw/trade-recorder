package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"trade-journal/internal/ctrader"
	"trade-journal/internal/database"
	"trade-journal/internal/handlers"
	"trade-journal/internal/middleware"
	"trade-journal/internal/minio"
	"trade-journal/internal/ws"

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
	// 初始化 WebSocket Hub
	ws.Init()

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

	// 啟動 cTrader 背景監聽管理器
	ctrader.StartManager(db)

	// 舊的 Base64 圖片已完成系統性遷移，不再執行自動遷移任務

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
				trades.POST("/:id/sync", handlers.SyncSingleTrade(db))
				trades.DELETE("/:id", handlers.DeleteTrade(db))
			}

			// 統計資料
			stats := authorized.Group("/stats")
			{
				stats.GET("/summary", handlers.GetStatsSummary(db))
				stats.GET("/equity-curve", handlers.GetEquityCurve(db))
				stats.GET("/by-symbol", handlers.GetStatsBySymbol(db))
				stats.GET("/by-strategy", handlers.GetStatsByStrategy(db))
				stats.GET("/by-color", handlers.GetStatsByColorTag(db))
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

			// 管理員路由
			admin := authorized.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.GET("/usage", handlers.GetSystemUsageStat(db))
			}

			// cTrader OAuth 啟動 (需要認證)
			authorized.GET("/auth/ctrader/url", handlers.CTraderAuthURL(db))

		}

		// WebSocket (移動到 Middleware 之外，因為瀏覽器 WS 無法傳送 Authorization Header)
		api.GET("/ws", func(c *gin.Context) {
			conn, err := ws.GetUpgrader().Upgrade(c.Writer, c.Request, nil)
			if err != nil {
				return
			}
			client := &ws.Client{Hub: ws.GlobalHub, Conn: conn, Send: make(chan []byte, 256)}
			ws.GlobalHub.Register() <- client
			go client.WritePump()
			go client.ReadPump()
		})

		// 公開路由
		api.GET("/shares/public/:token", handlers.GetSharedResource(db))
		// cTrader OAuth 回調 (公開路由)
		api.GET("/auth/ctrader/callback", handlers.CTraderCallback(db))

		// 圖片上傳 (目前先保持公開或也可加入認證)
		images := api.Group("/images")
		{
			images.POST("/upload", handlers.UploadImage(minioClient))
			images.GET("/:filename", handlers.GetImage(minioClient))
		}
	}

	// 靜態檔案服務 (SPA 優化版本:支援延遲載入)
	cwd, _ := os.Getwd()
	log.Printf("[Init] 當前工作目錄: %s", cwd)

	// 列出工作目錄內容以供除錯
	if entries, err := os.ReadDir(cwd); err == nil {
		log.Printf("[Init] 工作目錄內容:")
		for _, entry := range entries {
			if entry.IsDir() {
				log.Printf("  [DIR]  %s", entry.Name())
			} else {
				log.Printf("  [FILE] %s", entry.Name())
			}
		}
	}

	staticDir := detectStaticDir()
	if staticDir != "" {
		absPath, _ := filepath.Abs(staticDir)
		log.Printf("[Init] ✓ 找到靜態目錄: %s (絕對路徑: %s)", staticDir, absPath)

		// 驗證關鍵檔案存在
		indexPath := filepath.Join(staticDir, "index.html")
		if info, err := os.Stat(indexPath); err == nil {
			log.Printf("[Init] ✓ index.html 存在 (大小: %d bytes)", info.Size())
		}

		// 註冊 /assets 靜態路由
		assetsPath := filepath.Join(staticDir, "assets")
		if assetsInfo, err := os.Stat(assetsPath); err == nil && assetsInfo.IsDir() {
			r.Static("/assets", assetsPath)
			log.Printf("[Init] ✓ /assets 路由已註冊: %s", assetsPath)
		} else {
			log.Printf("[Init] ⚠ assets 目錄不存在: %s", assetsPath)
		}
	} else {
		log.Printf("[Init] ⚠ 啟動時未找到靜態目錄,將在請求時動態尋找")
		// 即使啟動時找不到,也註冊一個動態 /assets 處理器
		r.GET("/assets/*filepath", func(c *gin.Context) {
			currentStaticDir := detectStaticDir()
			if currentStaticDir == "" {
				c.AbortWithStatus(404)
				return
			}
			assetsPath := filepath.Join(currentStaticDir, "assets", c.Param("filepath"))
			if info, err := os.Stat(assetsPath); err == nil && !info.IsDir() {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
				c.File(assetsPath)
				return
			}
			c.AbortWithStatus(404)
		})
		log.Printf("[Init] ✓ 動態 /assets 處理器已註冊")
	}

	// SPA Fallback: 處理所有非 API 的請求
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// 1. API 請求不走 SPA fallback
		if strings.HasPrefix(path, "/api/") {
			return
		}

		// 2. 如果啟動時沒找到靜態目錄,現在再找一次 (處理啟動競爭)
		if staticDir == "" {
			staticDir = detectStaticDir()
			if staticDir != "" {
				log.Printf("[Dynamic] ✓ 請求時成功補獲靜態目錄: %s", staticDir)
			}
		}

		if staticDir == "" {
			log.Printf("[Error] 靜態目錄不存在,無法提供 SPA 服務: %s", path)
			c.String(503, "Service Unavailable: Static files not found")
			return
		}

		// 3. 檢查路徑是否對應到靜態檔案 (包含 /assets/)
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath == "" {
			cleanPath = "index.html"
		}
		filePath := filepath.Join(staticDir, cleanPath)

		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			// 快取控制
			if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") || strings.Contains(path, "/assets/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Header("Cache-Control", "public, max-age=3600")
			}

			// 安全地提供檔案
			if _, err := os.Open(filePath); err != nil {
				log.Printf("[Error] 無法開啟檔案 %s: %v", filePath, err)
				c.AbortWithStatus(500)
				return
			}
			c.File(filePath)
			return
		}

		// 4. SPA Fallback 保護：已知靜態副檔名沒找到就 404，不回傳 index.html
		ext := filepath.Ext(path)
		if ext != "" {
			staticExts := map[string]bool{
				".js": true, ".css": true, ".png": true, ".jpg": true,
				".jpeg": true, ".gif": true, ".svg": true, ".ico": true,
				".woff": true, ".woff2": true, ".ttf": true, ".json": true,
			}
			if staticExts[strings.ToLower(ext)] {
				c.AbortWithStatus(404)
				return
			}
		}

		// 5. 其餘路由導向 index.html (SPA 路由)
		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			log.Printf("[Error] index.html 不存在: %s", indexPath)
			c.String(503, "Service Unavailable: index.html not found")
			return
		}

		c.Header("Cache-Control", "no-cache")
		c.File(indexPath)
	})

	// 啟動伺服器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[Init] 伺服器啟動於 http://localhost:%s (包含 SPA 動態修復邏輯)", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("伺服器啟動失敗:", err)
	}
}

// detectStaticDir 尋找並回傳靜態檔案目錄路徑
func detectStaticDir() string {
	staticDirs := []string{"./frontend/dist", "../frontend/dist", "./dist", "../dist"}
	log.Printf("[detectStaticDir] 開始搜尋靜態目錄...")

	for _, dir := range staticDirs {
		absDir, _ := filepath.Abs(dir)

		// 檢查目錄是否存在
		info, err := os.Stat(dir)
		if err != nil {
			log.Printf("[detectStaticDir]   ✗ %s 不存在", dir)
			continue
		}

		if !info.IsDir() {
			log.Printf("[detectStaticDir]   ✗ %s 不是目錄", dir)
			continue
		}

		// 檢查 index.html 是否存在
		indexPath := filepath.Join(dir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			log.Printf("[detectStaticDir]   ✗ %s 存在但缺少 index.html", dir)
			continue
		}

		log.Printf("[detectStaticDir]   ✓ 找到有效目錄: %s (絕對路徑: %s)", dir, absDir)
		return dir
	}

	log.Printf("[detectStaticDir] ✗ 所有路徑都未找到靜態目錄")
	return ""
}
