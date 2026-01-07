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

	rows, err := db.Query(`SELECT id, ticket, symbol, side, entry_price, lot_size, entry_time 
		FROM trades 
		WHERE account_id = 7 AND ticket LIKE 'ctrader-pos-%' 
		ORDER BY id DESC 
		LIMIT 10`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("ID | Ticket | Symbol | Side | Entry Price | Lot Size | Entry Time")
	fmt.Println("-------------------------------------------------------------------")

	for rows.Next() {
		var id int64
		var ticket, symbol, side string
		var entryPrice, lotSize float64
		var entryTime string

		rows.Scan(&id, &ticket, &symbol, &side, &entryPrice, &lotSize, &entryTime)
		fmt.Printf("%d | %s | %s | %s | %.5f | %.2f | %s\n",
			id, ticket, symbol, side, entryPrice, lotSize, entryTime)
	}
}
