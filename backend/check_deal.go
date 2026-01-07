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

	var id int64
	var ticket string
	err = db.QueryRow("SELECT id, ticket FROM trades WHERE ticket = 'ctrader-deal-284997314'").Scan(&id, &ticket)
	if err == sql.ErrNoRows {
		fmt.Println("Deal NOT found in trades table.")
	} else if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Found Trade ID: %d, Ticket: %s\n", id, ticket)
	}
}
