package database

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

// InitDB 初始化SQLite資料庫
func InitDB() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./trade_journal.db"
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 測試連線
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// 執行Schema建立
	if err := createTables(db); err != nil {
		return nil, err
	}

	log.Println("資料庫初始化成功")
	return db, nil
}

func createTables(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(20) DEFAULT 'local', -- 'local' or 'metatrader'
		mt5_account_id VARCHAR(100),    -- MetaApi Account ID
		mt5_token TEXT,                 -- MetaApi Token
		status VARCHAR(20) DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trades (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL DEFAULT 1,
		symbol VARCHAR(20) NOT NULL,
		side VARCHAR(10) NOT NULL,
		entry_price REAL,
		exit_price REAL,
		lot_size REAL,
		pnl REAL,
		pnl_points REAL,
		notes TEXT,
		entry_reason TEXT,
		exit_reason TEXT,
		entry_time DATETIME NOT NULL,
		exit_time DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS trade_images (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trade_id INTEGER NOT NULL,
		image_type VARCHAR(20) NOT NULL,
		image_path VARCHAR(500) NOT NULL,
		image_order INTEGER DEFAULT 0,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (trade_id) REFERENCES trades(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(50) UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS trade_tags (
		trade_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (trade_id, tag_id),
		FOREIGN KEY (trade_id) REFERENCES trades(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_trades_symbol ON trades(symbol);
	CREATE INDEX IF NOT EXISTS idx_trades_entry_time ON trades(entry_time);
	CREATE INDEX IF NOT EXISTS idx_trade_images_trade_id ON trade_images(trade_id);
	CREATE INDEX IF NOT EXISTS idx_trade_tags_trade_id ON trade_tags(trade_id);

	CREATE TABLE IF NOT EXISTS daily_plans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL DEFAULT 1,
		plan_date DATETIME NOT NULL,
		symbol VARCHAR(20) DEFAULT 'XAUUSD',
		market_session VARCHAR(20),
		notes TEXT,
		trend_analysis TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// 遷移：為舊資料庫添加新欄位（如果不存在）
	migrationSQL := `
	-- 檢查並添加 entry_reason 欄位
	ALTER TABLE trades ADD COLUMN entry_reason TEXT;
	`
	// 忽略錯誤（如果欄位已存在會報錯，這是正常的）
	db.Exec(migrationSQL)

	migrationSQL2 := `
	-- 檢查並添加 exit_reason 欄位
	ALTER TABLE trades ADD COLUMN exit_reason TEXT;
	`
	db.Exec(migrationSQL2)

	migrationSQL3 := `
	-- 檢查並添加 trade_type 欄位 (actual=實際交易, observation=純觀察)
	ALTER TABLE trades ADD COLUMN trade_type VARCHAR(20) DEFAULT 'actual';
	`
	db.Exec(migrationSQL3)

	migrationSQL4 := `
	-- 檢查並添加 entry_strategy 欄位 (expert=達人, elite=菁英, legend=傳奇)
	ALTER TABLE trades ADD COLUMN entry_strategy VARCHAR(20);
	`
	db.Exec(migrationSQL4)

	migrationSQL5 := `
	-- 檢查並添加 entry_signals 欄位 (達人訊號，JSON格式)
	ALTER TABLE trades ADD COLUMN entry_signals TEXT;
	`
	db.Exec(migrationSQL5)

	migrationSQL6 := `
	-- 檢查並添加 entry_checklist 欄位 (菁英/傳奇檢查清單，JSON格式)
	ALTER TABLE trades ADD COLUMN entry_checklist TEXT;
	`
	db.Exec(migrationSQL6)

	migrationSQL7 := `
	-- 檢查並添加 market_session 欄位 (亞盤/歐盤/美盤)
	ALTER TABLE trades ADD COLUMN market_session VARCHAR(20);
	`
	db.Exec(migrationSQL7)

	migrationSQL8 := `
	-- 檢查並添加 timezone_offset 欄位 (時區偏移，如 +8)
	ALTER TABLE trades ADD COLUMN timezone_offset INTEGER DEFAULT 8;
	`
	db.Exec(migrationSQL8)

	migrationSQL9 := `
	-- 檢查並添加 trend_analysis 欄位 (各時間週期趨勢，JSON格式)
	ALTER TABLE trades ADD COLUMN trend_analysis TEXT;
	`
	db.Exec(migrationSQL9)

	migrationSQL10 := `
	-- 檢查並添加 entry_strategy_image 欄位 (進場種類圖片)
	ALTER TABLE trades ADD COLUMN entry_strategy_image TEXT;
	`
	db.Exec(migrationSQL10)

	migrationSQL11 := `
	-- 檢查並添加 entry_timeframe 欄位 (進場時區)
	ALTER TABLE trades ADD COLUMN entry_timeframe VARCHAR(10);
	`
	db.Exec(migrationSQL11)

	migrationSQL12 := `
	-- 檢查並添加 trend_type 欄位 (順勢/逆勢)
	ALTER TABLE trades ADD COLUMN trend_type VARCHAR(20);
	`
	db.Exec(migrationSQL12)

	migrationSQL13 := `
	-- 檢查並添加 entry_pattern 欄位 (進場樣態，僅菁英使用)
	ALTER TABLE trades ADD COLUMN entry_pattern VARCHAR(20);
	`
	db.Exec(migrationSQL13)

	migrationSQL14 := `
	-- 檢查並添加 entry_strategy_image_original 欄位
	ALTER TABLE trades ADD COLUMN entry_strategy_image_original TEXT;
	`
	db.Exec(migrationSQL14)

	migrationSQL15 := `
	-- 檢查並添加 symbol 欄位到 daily_plans
	ALTER TABLE daily_plans ADD COLUMN symbol VARCHAR(20) DEFAULT 'XAUUSD';
	`
	db.Exec(migrationSQL15)

	// 遷移：添加 account_id 欄位到舊表 (忽略錯誤，因為 SQLite 不支持 ALTER TABLE ADD COLUMN IF NOT EXISTS)
	db.Exec("ALTER TABLE trades ADD COLUMN account_id INTEGER NOT NULL DEFAULT 1;")
	db.Exec("ALTER TABLE daily_plans ADD COLUMN account_id INTEGER NOT NULL DEFAULT 1;")

	// 確保至少有一個預設帳號
	db.Exec(`INSERT OR IGNORE INTO accounts (id, name, type) VALUES (1, '預設帳號', 'local');`)

	// 刪除重複的規劃，只保留最新更新的一筆，以便建立唯一索引
	cleanupSQL := `
	DELETE FROM daily_plans 
	WHERE id NOT IN (
		SELECT MAX(id) 
		FROM daily_plans 
		GROUP BY date(plan_date), symbol, account_id
	);`
	db.Exec(cleanupSQL)

	// 最後才建立唯一索引，確保同品種同一天只能有一組規劃
	dropIdxSQL := `DROP INDEX IF EXISTS idx_daily_plans_date_symbol;`
	db.Exec(dropIdxSQL)
	idxSQL := `CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_plans_date_symbol ON daily_plans(plan_date, symbol, account_id);`
	db.Exec(idxSQL)

	migrationSQL16 := `
	-- 檢查並添加同步相關欄位到 accounts
	ALTER TABLE accounts ADD COLUMN sync_status VARCHAR(20) DEFAULT 'idle';
	ALTER TABLE accounts ADD COLUMN last_synced_at DATETIME;
	ALTER TABLE accounts ADD COLUMN last_sync_error TEXT;
	`
	db.Exec(migrationSQL16)

	migrationSQL17 := `
	-- 檢查並添加 initial_sl, bullet_size, rr_ratio 欄位
	ALTER TABLE trades ADD COLUMN initial_sl REAL;
	ALTER TABLE trades ADD COLUMN bullet_size REAL;
	ALTER TABLE trades ADD COLUMN rr_ratio REAL;
	`
	db.Exec(migrationSQL17)

	migrationSQL18 := `
	-- 檢查並添加 timezone_offset 欄位到 accounts
	ALTER TABLE accounts ADD COLUMN timezone_offset INTEGER DEFAULT 8;
	`
	db.Exec(migrationSQL18)

	migrationSQL19 := `
	-- 檢查並添加 ticket 欄位到 trades
	ALTER TABLE trades ADD COLUMN ticket VARCHAR(50);
	`
	db.Exec(migrationSQL19)

	migrationSQL20 := `
	-- 檢查並添加 exit_sl 欄位到 trades
	ALTER TABLE trades ADD COLUMN exit_sl REAL;
	`
	db.Exec(migrationSQL20)

	return nil
}
