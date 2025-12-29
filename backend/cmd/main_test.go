package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"trade-journal/internal/database"
	"trade-journal/internal/handlers"
	"trade-journal/internal/models"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() (*gin.Engine, error) {
	gin.SetMode(gin.TestMode)
	
	db, err := database.InitDB()
	if err != nil {
		return nil, err
	}

	r := gin.Default()
	
	api := r.Group("/api/v1")
	{
		trades := api.Group("/trades")
		{
			trades.GET("", handlers.GetTrades(db))
			trades.GET("/:id", handlers.GetTrade(db))
			trades.POST("", handlers.CreateTrade(db))
			trades.PUT("/:id", handlers.UpdateTrade(db))
			trades.DELETE("/:id", handlers.DeleteTrade(db))
		}
		
		stats := api.Group("/stats")
		{
			stats.GET("/summary", handlers.GetStatsSummary(db))
		}
	}
	
	return r, nil
}

func TestHealthCheck(t *testing.T) {
	router, err := setupTestRouter()
	if err != nil {
		t.Fatalf("設定路由失敗: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望狀態碼 200，得到 %d", w.Code)
	}
}

func TestCreateTrade(t *testing.T) {
	router, err := setupTestRouter()
	if err != nil {
		t.Fatalf("設定路由失敗: %v", err)
	}

	floatPtr := func(f float64) *float64 { return &f }

	trade := models.TradeCreate{
		Symbol:     "XAUUSD",
		Side:       "long",
		EntryPrice: floatPtr(2000.50),
		LotSize:    floatPtr(0.1),
		EntryTime:  time.Now(),
		Tags:       []string{"測試"},
	}

	jsonData, _ := json.Marshal(trade)
	
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/trades", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("期望狀態碼 201，得到 %d", w.Code)
	}
}

func TestGetTrades(t *testing.T) {
	router, err := setupTestRouter()
	if err != nil {
		t.Fatalf("設定路由失敗: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/trades", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望狀態碼 200，得到 %d", w.Code)
	}
}

func TestGetStatsSummary(t *testing.T) {
	router, err := setupTestRouter()
	if err != nil {
		t.Fatalf("設定路由失敗: %v", err)
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/stats/summary", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望狀態碼 200，得到 %d", w.Code)
	}

	var summary models.StatsSummary
	err = json.Unmarshal(w.Body.Bytes(), &summary)
	if err != nil {
		t.Errorf("解析回應失敗: %v", err)
	}
}

