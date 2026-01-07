package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "trade_journal.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check all trades for position 257101309
	rows, err := db.Query(`
		SELECT id, ticket, symbol, side, entry_price, lot_size, entry_time, exit_time, notes 
		FROM trades 
		WHERE account_id = 7 AND (ticket LIKE '%257101309%')
		ORDER BY id DESC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("All trades for position 257101309:")
	fmt.Println("====================================")

	for rows.Next() {
		var id int64
		var ticket, symbol, side, notes string
		var entryPrice, lotSize float64
		var entryTime, exitTime sql.NullString

		rows.Scan(&id, &ticket, &symbol, &side, &entryPrice, &lotSize, &entryTime, &exitTime, &notes)
		fmt.Printf("ID: %d\n", id)
		fmt.Printf("Ticket: %s\n", ticket)
		fmt.Printf("Symbol: '%s'\n", symbol)
		fmt.Printf("Side: %s\n", side)
		fmt.Printf("Entry Price: %.5f\n", entryPrice)
		fmt.Printf("Lot Size: %.2f\n", lotSize)
		fmt.Printf("Entry Time: %v\n", entryTime)
		fmt.Printf("Exit Time: %v\n", exitTime)
		fmt.Printf("Notes: %s\n", notes)
		fmt.Println("------------------------------------")
	}
}
