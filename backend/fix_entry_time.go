package main

import (
	"database/sql"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "trade_journal.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Update entry_time from 1970 to proper time (2026-01-06 14:34:04)
	properTime := time.Date(2026, 1, 6, 14, 34, 4, 0, time.FixedZone("CST", 8*3600))

	result, err := db.Exec(`UPDATE trades SET entry_time = ? WHERE id = 3113`, properTime)
	if err != nil {
		log.Fatal(err)
	}

	rows, _ := result.RowsAffected()
	log.Printf("Updated %d rows with entry_time: %s", rows, properTime)

	// Verify
	var entryTime string
	db.QueryRow("SELECT entry_time FROM trades WHERE id = 3113").Scan(&entryTime)
	log.Printf("Verified entry_time: %s", entryTime)
}
