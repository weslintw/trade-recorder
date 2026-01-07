package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "trade_journal.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Update the empty symbol to XAUUSD (based on price 4464.70, it's clearly gold)
	result, err := db.Exec(`UPDATE trades SET symbol = 'XAUUSD' WHERE id = 3112`)
	if err != nil {
		log.Fatal(err)
	}

	rows, _ := result.RowsAffected()
	log.Printf("Updated %d rows", rows)
}
