package myfxbook

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"trade-journal/internal/database"
)

const BaseURL = "https://www.myfxbook.com/api"

type LoginResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Session string `json:"session"`
}

type HistoryResponse struct {
	Error   bool          `json:"error"`
	Message string        `json:"message"`
	History []Transaction `json:"history"`
}

type Transaction struct {
	OpenTime   string      `json:"openTime"`
	CloseTime  string      `json:"closeTime"`
	Symbol     string      `json:"symbol"`
	Action     string      `json:"action"` // "Buy", "Sell", "Buy Limit", etc.
	Sizing     Sizing      `json:"sizing"`
	OpenPrice  FlexFloat64 `json:"openPrice"`
	ClosePrice FlexFloat64 `json:"closePrice"`
	TP         FlexFloat64 `json:"tp"`
	SL         FlexFloat64 `json:"sl"`
	Pips       FlexFloat64 `json:"pips"`
	Profit     FlexFloat64 `json:"profit"`
	Interest   FlexFloat64 `json:"interest"` // Swap
	Commission FlexFloat64 `json:"commission"`
	Comment    string      `json:"comment"`
}

type Sizing struct {
	Type  string      `json:"type"`
	Value FlexFloat64 `json:"value"` // Lot size
}

// FlexFloat64 處理 Myfxbook API 有時回傳字串、有時回傳數字的問題
type FlexFloat64 float64

func (f *FlexFloat64) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	// 去掉前後引號 (如果是字串)
	s := string(b)
	if s[0] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	// 嘗試解析為 float
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	*f = FlexFloat64(val)
	return nil
}

// SyncMyfxbookHistory 同步 Myfxbook 最近交易
func SyncMyfxbookHistory(db *sql.DB, accountID int64, email, password, myfxbookAccountID string) {
	log.Printf("[Myfxbook] 開始同步帳號 %d (Myfxbook ID: %s)", accountID, myfxbookAccountID)

	// 1. Login
	session, err := login(email, password)
	if err != nil {
		updateSyncStatus(db, accountID, "failed", err.Error())
		return
	}
	defer logout(session)

	// 2. Get History
	transactions, err := getHistory(session, myfxbookAccountID)
	if err != nil {
		updateSyncStatus(db, accountID, "failed", err.Error())
		return
	}

	// 3. Save to DB
	count := 0
	for _, tx := range transactions {
		if tx.Action != "Buy" && tx.Action != "Sell" {
			continue // 忽略 Limit/Stop 等掛單
		}

		// 解析時間
		openTime, _ := parseTime(tx.OpenTime)
		closeTime, _ := parseTime(tx.CloseTime)

		side := "long"
		if strings.ToLower(tx.Action) == "sell" {
			side = "short"
		}

		totalPnL := float64(tx.Profit) + float64(tx.Interest) + float64(tx.Commission)

		// 產生一個唯一識別碼，如果沒有官方 Ticket ID，Myfxbook 單子通常會用 openTime + closeTime + symbol 作為特徵
		// 但如果 Myfxbook API 沒給 Ticket，我們只能用特徵比對或者看是否能從 Comment 挖
		ticket := ""
		if tx.Comment != "" {
			// 有些經紀商會把 Ticket 放在 Comment，但這不保全
		}
		// 為了確保不重複，我們這裡優先使用 ticket 欄位，如果沒有則使用 (account_id, symbol, entry_time, lot_size) 的組合

		// 檢查是否已存在
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM trades WHERE account_id = ? AND symbol = ? AND entry_time = ? AND lot_size = ?)
		`, accountID, tx.Symbol, openTime, float64(tx.Sizing.Value)).Scan(&exists)

		if exists {
			continue
		}

		// 自動判斷時段
		marketSession := determineMarketSession(openTime)

		_, err = db.Exec(`
			INSERT INTO trades (
				account_id, symbol, side, entry_price, exit_price, lot_size, 
				pnl, pnl_points, entry_time, exit_time, trade_type, 
				notes, timezone_offset, market_session, ticket, exit_sl
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, tx.Symbol, side, float64(tx.OpenPrice), float64(tx.ClosePrice), float64(tx.Sizing.Value),
			totalPnL, float64(tx.Pips), openTime, closeTime, "actual",
			tx.Comment, 8, marketSession, ticket, float64(tx.SL))

		if err == nil {
			count++
		} else {
			log.Printf("[Myfxbook] 儲存交易失敗: %v", err)
		}
	}

	updateSyncStatus(db, accountID, "success", fmt.Sprintf("成功同步 %d 筆交易", count))
	database.UpdateAccountStorageUsage(db, accountID)
	log.Printf("[Myfxbook] 帳號 %d 同步完成，新增 %d 筆", accountID, count)
}

func login(email, password string) (string, error) {
	url := fmt.Sprintf("%s/login.json?email=%s&password=%s", BaseURL, email, password)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	if res.Error {
		return "", fmt.Errorf(res.Message)
	}

	return res.Session, nil
}

func logout(session string) {
	url := fmt.Sprintf("%s/logout.json?session=%s", BaseURL, session)
	http.Get(url)
}

func getHistory(session, accountID string) ([]Transaction, error) {
	url := fmt.Sprintf("%s/get-history.json?session=%s&id=%s", BaseURL, session, accountID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Myfxbook API 有時會因為紀錄太多回傳慢，或者格式問題
	// 這裡讀取 body 以便除錯
	body, _ := io.ReadAll(resp.Body)

	var res HistoryResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("解析歷史紀錄失敗: %v", err)
	}

	if res.Error {
		return nil, fmt.Errorf(res.Message)
	}

	return res.History, nil
}

func updateSyncStatus(db *sql.DB, id int64, status, errStr string) {
	if status == "success" {
		db.Exec("UPDATE accounts SET sync_status = 'success', last_synced_at = CURRENT_TIMESTAMP, last_sync_error = ? WHERE id = ?", errStr, id)
	} else {
		db.Exec("UPDATE accounts SET sync_status = 'failed', last_sync_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", errStr, id)
	}
}

func parseTime(s string) (time.Time, error) {
	// Myfxbook format: "MM/DD/YYYY HH:MM"
	return time.Parse("01/02/2006 15:04", s)
}

func determineMarketSession(t time.Time) string {
	// 轉為 GMT+8 進行判斷
	loc := time.FixedZone("GMT", 8*3600)
	t8 := t.In(loc)

	hour := t8.Hour()
	minute := t8.Minute()
	timeInMinutes := hour*60 + minute

	// 簡化判斷
	if timeInMinutes >= 8*60 && timeInMinutes < 15*60 {
		return "asian"
	}
	if timeInMinutes >= 15*60 && timeInMinutes < 23*60 {
		return "european"
	}
	return "us"
}
