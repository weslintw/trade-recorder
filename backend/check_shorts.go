package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./trade_journal.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 1. Check Count of Short vs Long
	var shortCount, longCount int
	db.QueryRow("SELECT COUNT(*) FROM trades WHERE side = 'short'").Scan(&shortCount)
	db.QueryRow("SELECT COUNT(*) FROM trades WHERE side = 'long'").Scan(&longCount)

	fmt.Printf("Trades Stats: Long: %d, Short: %d\n", longCount, shortCount)

	// 2. Fetch recent SHORT trades and verify ownership
	rows, err := db.Query(`
		SELECT t.id, t.ticket, t.symbol, t.entry_price, t.pnl, t.account_id, a.user_id, t.entry_signals
		FROM trades t 
		JOIN accounts a ON t.account_id = a.id 
		WHERE t.side = 'short' 
		ORDER BY t.id DESC 
		LIMIT 5
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\nRecent SHORT Trades:")
	for rows.Next() {
		var id int
		var ticket string
		var symbol string
		var ep, pnl float64
		var accID, userID int
		var signals sql.NullString

		rows.Scan(&id, &ticket, &symbol, &ep, &pnl, &accID, &userID, &signals)
		fmt.Printf("[ID: %d] Ticket: %s | Sym: %s | PnL: %.2f | Acc: %d (User: %d) | Signals: %s\n",
			id, ticket, symbol, pnl, accID, userID, signals.String)

		// Simulate Ownership Check
		if userID != 1 {
			fmt.Printf("  WARNING: User ID mismatch! Expected 1 (Admin)\n")
		}
	}
}
