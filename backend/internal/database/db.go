package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// InitDB 初始化SQLite資料庫
func InitDB() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./trade_journal.db"
	}

	// 檢查路徑，如果是目錄，則自動補上檔名
	info, err := os.Stat(dbPath)
	if err == nil && info.IsDir() {
		dbPath = filepath.Join(dbPath, "trade_journal.db")
	}
	
	absPath, _ := filepath.Abs(dbPath)
	log.Printf("[DB] 資料庫路徑: %s (Absolute: %s)", dbPath, absPath)

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
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username VARCHAR(50) UNIQUE NOT NULL,
		password TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL DEFAULT 1,
		name VARCHAR(100) NOT NULL,
		type VARCHAR(20) DEFAULT 'local', -- 'local' or 'metatrader'
		mt5_account_id VARCHAR(100),    -- MetaApi Account ID
		mt5_token TEXT,                 -- MetaApi Token
		ctrader_account_id VARCHAR(100), -- cTrader Account ID
		ctrader_token TEXT,             -- cTrader Token
		ctrader_client_id VARCHAR(100), -- cTrader Client ID
		ctrader_client_secret TEXT,     -- cTrader Client Secret
		status VARCHAR(20) DEFAULT 'active',
		storage_usage INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
		exit_sl REAL,
		legend_king_htf VARCHAR(20),
		legend_king_image TEXT,
		legend_king_image_original TEXT,
		legend_htf VARCHAR(20),
		legend_htf_image TEXT,
		legend_htf_image_original TEXT,
		legend_de_htf VARCHAR(20),
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
		user_id INTEGER NOT NULL DEFAULT 1,
		name VARCHAR(50) NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(user_id, name)
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
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);

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

	CREATE TABLE IF NOT EXISTS shares (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		resource_type VARCHAR(20) NOT NULL, -- 'trade', 'plan'
		resource_id INTEGER NOT NULL,
		share_type VARCHAR(20) NOT NULL,    -- 'public' (link), 'specific' (users)
		token VARCHAR(100) UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS share_users (
		share_id INTEGER NOT NULL,
		shared_with_user_id INTEGER NOT NULL,
		PRIMARY KEY (share_id, shared_with_user_id),
		FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE,
		FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return err
	}

	// 遷移：為舊資料庫添加新欄位（如果不存在）
	// SQLite 不支援 IF NOT EXISTS 於 ALTER TABLE，所以這裡忽略錯誤
	db.Exec("ALTER TABLE accounts ADD COLUMN user_id INTEGER NOT NULL DEFAULT 1;")
	db.Exec("ALTER TABLE tags ADD COLUMN user_id INTEGER NOT NULL DEFAULT 1;")

	// 確保至少有一個預設管理員使用者 (username='admin')
	var adminID int64
	var currentHash string
	err = db.QueryRow("SELECT id, password FROM users WHERE username = 'admin'").Scan(&adminID, &currentHash)
	if err == sql.ErrNoRows {
		log.Println("[DB] 找不到 admin 使用者，正在建立預設管理員帳號...")
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), 10)
		_, err = db.Exec(`INSERT INTO users (username, password, is_admin) VALUES ('admin', ?, 1);`, string(hashedPassword))
		if err != nil {
			log.Println("[DB] 建立管理員帳號失敗:", err)
		} else {
			log.Println("[DB] 預設管理員帳號建立成功 (admin/admin123)")
		}
	} else {
		log.Printf("[DB] 找到 admin 使用者 (ID: %d)", adminID)
	}

	migrationSQL := `
	-- 檢查並添加 entry_reason 欄位
	ALTER TABLE trades ADD COLUMN entry_reason TEXT;
	`
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

	// 遷移：添加 account_id 欄位到舊表
	db.Exec("ALTER TABLE trades ADD COLUMN account_id INTEGER NOT NULL DEFAULT 1;")
	db.Exec("ALTER TABLE daily_plans ADD COLUMN account_id INTEGER NOT NULL DEFAULT 1;")

	// 確保至少有一個預設帳號，並關聯到使用者 1
	// db.Exec(`INSERT OR IGNORE INTO accounts (id, name, type, user_id) VALUES (1, '預設帳號', 'local', 1);`)

	// 建立唯一索引，確保同品種同一天只能有一組規劃
	db.Exec(`DROP INDEX IF EXISTS idx_daily_plans_date_symbol;`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_plans_date_symbol ON daily_plans(plan_date, symbol, account_id);`)

	db.Exec("ALTER TABLE accounts ADD COLUMN sync_status VARCHAR(20) DEFAULT 'idle';")
	db.Exec("ALTER TABLE accounts ADD COLUMN last_synced_at DATETIME;")
	db.Exec("ALTER TABLE accounts ADD COLUMN last_sync_error TEXT;")
	db.Exec("ALTER TABLE accounts ADD COLUMN timezone_offset INTEGER DEFAULT 8;")
	db.Exec("ALTER TABLE accounts ADD COLUMN storage_usage INTEGER DEFAULT 0;")
	db.Exec("ALTER TABLE accounts ADD COLUMN ctrader_account_id VARCHAR(100);")
	db.Exec("ALTER TABLE accounts ADD COLUMN ctrader_token TEXT;")
	db.Exec("ALTER TABLE accounts ADD COLUMN ctrader_client_id VARCHAR(100);")
	db.Exec("ALTER TABLE accounts ADD COLUMN ctrader_client_secret TEXT;")

	db.Exec("ALTER TABLE trades ADD COLUMN initial_sl REAL;")
	db.Exec("ALTER TABLE trades ADD COLUMN bullet_size REAL;")
	db.Exec("ALTER TABLE trades ADD COLUMN rr_ratio REAL;")
	db.Exec("ALTER TABLE trades ADD COLUMN ticket VARCHAR(50);")
	db.Exec("ALTER TABLE trades ADD COLUMN exit_sl REAL;")

	db.Exec("ALTER TABLE trades ADD COLUMN legend_king_htf VARCHAR(20);")
	db.Exec("ALTER TABLE trades ADD COLUMN legend_king_image TEXT;")
	db.Exec("ALTER TABLE trades ADD COLUMN legend_king_image_original TEXT;")
	db.Exec("ALTER TABLE trades ADD COLUMN legend_htf VARCHAR(20);")
	db.Exec("ALTER TABLE trades ADD COLUMN legend_htf_image TEXT;")
	db.Exec("ALTER TABLE trades ADD COLUMN legend_htf_image_original TEXT;")
	db.Exec("ALTER TABLE trades ADD COLUMN legend_de_htf VARCHAR(20);")

	migrationSQL16 := `
	-- 檢查並添加 color_tag 欄位 (red, yellow, green)
	ALTER TABLE trades ADD COLUMN color_tag VARCHAR(20);
	`
	db.Exec(migrationSQL16)

	return nil
}
