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

	id := 3787
	var accountID int64
	var symbol string
	var exitTime sql.NullTime

	// Check if trade exists regardless of user
	err = db.QueryRow("SELECT account_id, symbol, exit_time FROM trades WHERE id = ?", id).Scan(&accountID, &symbol, &exitTime)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("Trade ID %d NOT FOUND in database.\n", id)
		} else {
			fmt.Printf("Error querying trade: %v\n", err)
		}
	} else {
		fmt.Printf("Trade ID %d FOUND. AccountID: %d, Symbol: %s, ExitTime: %v\n", id, accountID, symbol, exitTime)

		// Check user ownership
		var userID int64
		err = db.QueryRow("SELECT user_id FROM accounts WHERE id = ?", accountID).Scan(&userID)
		if err != nil {
			fmt.Printf("Error getting user for account %d: %v\n", accountID, err)
		} else {
			fmt.Printf("Trade belongs to User ID: %d\n", userID)
		}
	}

	// List some recent trades to see what IS there
	rows, err := db.Query("SELECT id, symbol, entry_time FROM trades ORDER BY id DESC LIMIT 5")
	if err == nil {
		fmt.Println("Recent trades:")
		defer rows.Close()
		for rows.Next() {
			var rectId int
			var sym string
			var et string
			rows.Scan(&rectId, &sym, &et)
			fmt.Printf("ID: %d, Sym: %s, Time: %s\n", rectId, sym, et)
		}
	}
}
